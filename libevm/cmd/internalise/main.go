// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

// The internalise command modifies Go files in place, making exported methods
// internal.
//
// Usage:
//
//	internalise -file <filepath> <type>.<method> [<type>.<method> [...]]
//
// For example, with file foo.go containing declarations:
//
//	func (f *Foo) Bar() { ... }
//
//	func (Foo) Baz() { ... }
//
// running
//
//	internalise -file foo.go Foo.Bar Foo.Baz
//
// results in
//
//	func (f *Foo) bar() { ... }
//
//	func (Foo) baz() { ... }
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func main() {
	file := flag.String("file", "", "File to modify")
	flag.Parse()

	if err := run(*file, flag.Args()...); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run(fileName string, args ...string) error {
	methods := make(map[string]map[string]struct{})
	for _, a := range args {
		a = strings.TrimPrefix(strings.TrimSpace(a), "*")
		parts := strings.Split(a, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid method identifier %q", a)
		}
		typ, fn := parts[0], parts[1]
		if _, ok := methods[typ]; !ok {
			methods[typ] = make(map[string]struct{})
		}
		methods[typ][fn] = struct{}{}
	}

	fset := token.NewFileSet()
	mode := parser.SkipObjectResolution | parser.ParseComments
	parsed, err := parser.ParseFile(fset, fileName, nil, mode)
	if err != nil {
		return fmt.Errorf("parser.ParseFile(%q): %v", fileName, err)
	}

	for _, d := range parsed.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv.NumFields() != 1 {
			continue
		}

		var typ string
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.Ident:
			typ = t.Name
		case *ast.StarExpr:
			typ = t.X.(*ast.Ident).Name //nolint:forcetypeassert // Invariant of valid Go method
		}

		name := &fn.Name.Name
		if _, ok := methods[typ][*name]; !ok {
			continue
		}
		if n := []rune(*name); n[0] >= 'A' && n[0] <= 'Z' {
			n[0] += 'a' - 'A'
			*name = string(n)
		}
	}

	// Since we're not creating, the zero perm/mode is ignored.
	f, err := os.OpenFile(fileName, os.O_TRUNC|os.O_WRONLY, 0) //nolint:gosec // Variable file is under our direct control in go:generate
	if err != nil {
		return fmt.Errorf("os.OpenFile(%q, [write-only, truncate]): %v", fileName, err)
	}
	if err := format.Node(f, fset, parsed); err != nil {
		return fmt.Errorf("format.Node(%T): %v", parsed, err)
	}
	return f.Close()
}
