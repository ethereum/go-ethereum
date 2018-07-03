// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// This file contains the code to process sources, to be able to deduct the
// original types.

package stack

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"math"
	"strings"
)

// cache is a cache of sources on the file system.
type cache struct {
	files  map[string][]byte
	parsed map[string]*parsedFile
}

// Augment processes source files to improve calls to be more descriptive.
//
// It modifies goroutines in place.
func Augment(goroutines []Goroutine) {
	c := &cache{}
	for i := range goroutines {
		c.augmentGoroutine(&goroutines[i])
	}
}

// augmentGoroutine processes source files to improve call to be more
// descriptive.
//
// It modifies the routine.
func (c *cache) augmentGoroutine(goroutine *Goroutine) {
	if c.files == nil {
		c.files = map[string][]byte{}
	}
	if c.parsed == nil {
		c.parsed = map[string]*parsedFile{}
	}
	// For each call site, look at the next call and populate it. Then we can
	// walk back and reformat things.
	for i := range goroutine.Stack.Calls {
		c.load(goroutine.Stack.Calls[i].SourcePath)
	}

	// Once all loaded, we can look at the next call when available.
	for i := 1; i < len(goroutine.Stack.Calls); i++ {
		// Get the AST from the previous call and process the call line with it.
		if f := c.getFuncAST(&goroutine.Stack.Calls[i]); f != nil {
			processCall(&goroutine.Stack.Calls[i], f)
		}
	}
}

// Private stuff.

// load loads a source file and parses the AST tree. Failures are ignored.
func (c *cache) load(fileName string) {
	if _, ok := c.parsed[fileName]; ok {
		return
	}
	c.parsed[fileName] = nil
	if !strings.HasSuffix(fileName, ".go") {
		// Ignore C and assembly.
		c.files[fileName] = nil
		return
	}
	log.Printf("load(%s)", fileName)
	if _, ok := c.files[fileName]; !ok {
		var err error
		if c.files[fileName], err = ioutil.ReadFile(fileName); err != nil {
			log.Printf("Failed to read %s: %s", fileName, err)
			c.files[fileName] = nil
			return
		}
	}
	fset := token.NewFileSet()
	src := c.files[fileName]
	parsed, err := parser.ParseFile(fset, fileName, src, 0)
	if err != nil {
		log.Printf("Failed to parse %s: %s", fileName, err)
		return
	}
	// Convert the line number into raw file offset.
	offsets := []int{0, 0}
	start := 0
	for l := 1; start < len(src); l++ {
		start += bytes.IndexByte(src[start:], '\n') + 1
		offsets = append(offsets, start)
	}
	c.parsed[fileName] = &parsedFile{offsets, parsed}
}

func (c *cache) getFuncAST(call *Call) *ast.FuncDecl {
	if p := c.parsed[call.SourcePath]; p != nil {
		return p.getFuncAST(call.Func.Name(), call.Line)
	}
	return nil
}

type parsedFile struct {
	lineToByteOffset []int
	parsed           *ast.File
}

// getFuncAST gets the callee site function AST representation for the code
// inside the function f at line l.
func (p *parsedFile) getFuncAST(f string, l int) (d *ast.FuncDecl) {
	// Walk the AST to find the lineToByteOffset that fits the line number.
	var lastFunc *ast.FuncDecl
	var found ast.Node
	// Inspect() goes depth first. This means for example that a function like:
	// func a() {
	//   b := func() {}
	//   c()
	// }
	//
	// Were we are looking at the c() call can return confused values. It is
	// important to look at the actual ast.Node hierarchy.
	ast.Inspect(p.parsed, func(n ast.Node) bool {
		if d != nil {
			return false
		}
		if n == nil {
			return true
		}
		if found != nil {
			// We are walking up.
		}
		if int(n.Pos()) >= p.lineToByteOffset[l] {
			// We are expecting a ast.CallExpr node. It can be harder to figure out
			// when there are multiple calls on a single line, as the stack trace
			// doesn't have file byte offset information, only line based.
			// gofmt will always format to one function call per line but there can
			// be edge cases, like:
			//   a = A{Foo(), Bar()}
			d = lastFunc
			//p.processNode(call, n)
			return false
		} else if f, ok := n.(*ast.FuncDecl); ok {
			lastFunc = f
		}
		return true
	})
	return
}

func name(n ast.Node) string {
	if _, ok := n.(*ast.InterfaceType); ok {
		return "interface{}"
	}
	if i, ok := n.(*ast.Ident); ok {
		return i.Name
	}
	if _, ok := n.(*ast.FuncType); ok {
		return "func"
	}
	if s, ok := n.(*ast.SelectorExpr); ok {
		return s.Sel.Name
	}
	// TODO(maruel): Implement anything missing.
	return "<unknown>"
}

// fieldToType returns the type name and whether if it's an ellipsis.
func fieldToType(f *ast.Field) (string, bool) {
	switch arg := f.Type.(type) {
	case *ast.ArrayType:
		return "[]" + name(arg.Elt), false
	case *ast.Ellipsis:
		return name(arg.Elt), true
	case *ast.FuncType:
		// Do not print the function signature to not overload the trace.
		return "func", false
	case *ast.Ident:
		return arg.Name, false
	case *ast.InterfaceType:
		return "interface{}", false
	case *ast.SelectorExpr:
		return arg.Sel.Name, false
	case *ast.StarExpr:
		return "*" + name(arg.X), false
	default:
		// TODO(maruel): Implement anything missing.
		return "<unknown>", false
	}
}

// extractArgumentsType returns the name of the type of each input argument.
func extractArgumentsType(f *ast.FuncDecl) ([]string, bool) {
	var fields []*ast.Field
	if f.Recv != nil {
		if len(f.Recv.List) != 1 {
			panic("Expect only one receiver; please fix panicparse's code")
		}
		// If it is an object receiver (vs a pointer receiver), its address is not
		// printed in the stack trace so it needs to be ignored.
		if _, ok := f.Recv.List[0].Type.(*ast.StarExpr); ok {
			fields = append(fields, f.Recv.List[0])
		}
	}
	var types []string
	extra := false
	for _, arg := range append(fields, f.Type.Params.List...) {
		// Assert that extra is only set on the last item of fields?
		var t string
		t, extra = fieldToType(arg)
		mult := len(arg.Names)
		if mult == 0 {
			mult = 1
		}
		for i := 0; i < mult; i++ {
			types = append(types, t)
		}
	}
	return types, extra
}

// processCall walks the function and populate call accordingly.
func processCall(call *Call, f *ast.FuncDecl) {
	values := make([]uint64, len(call.Args.Values))
	for i := range call.Args.Values {
		values[i] = call.Args.Values[i].Value
	}
	index := 0
	pop := func() uint64 {
		if len(values) != 0 {
			x := values[0]
			values = values[1:]
			index++
			return x
		}
		return 0
	}
	popName := func() string {
		n := call.Args.Values[index].Name
		v := pop()
		if len(n) == 0 {
			return fmt.Sprintf("0x%x", v)
		}
		return n
	}

	types, extra := extractArgumentsType(f)
	for i := 0; len(values) != 0; i++ {
		var t string
		if i >= len(types) {
			if !extra {
				// These are unexpected value! Print them as hex.
				call.Args.Processed = append(call.Args.Processed, popName())
				continue
			}
			t = types[len(types)-1]
		} else {
			t = types[i]
		}
		switch t {
		case "float32":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%g", math.Float32frombits(uint32(pop()))))
		case "float64":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%g", math.Float64frombits(pop())))
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%d", pop()))
		case "string":
			call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s, len=%d)", t, popName(), pop()))
		default:
			if strings.HasPrefix(t, "*") {
				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s)", t, popName()))
			} else if strings.HasPrefix(t, "[]") {
				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s len=%d cap=%d)", t, popName(), pop(), pop()))
			} else {
				// Assumes it's an interface. For now, discard the object value, which
				// is probably not a good idea.
				call.Args.Processed = append(call.Args.Processed, fmt.Sprintf("%s(%s)", t, popName()))
				pop()
			}
		}
		if len(values) == 0 && call.Args.Elided {
			return
		}
	}
}
