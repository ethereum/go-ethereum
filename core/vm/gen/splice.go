// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"
)

// This file holds the splicing layer: parsing core/vm, rendering handler and
// helper bodies back to source, and rewriting their returns into loop control
// flow.

// gasHelperNames are the gas and memory helpers whose bodies the generator
// splices instead of calling.
var gasHelperNames = map[string]bool{
	"ChargeExecutionOnly": true,
	"computeMemorySize":   true,
	"chargeDynamicGas":    true,
}

// source is the parsed core/vm code the generator splices from.
type source struct {
	fset         *token.FileSet
	opHandlers   map[string]*ast.FuncDecl // top-level opXxx handlers and make* factories
	gasHelpers   map[string]*ast.FuncDecl // the helpers named in gasHelperNames
	stackMethods map[string]*ast.FuncDecl // *Stack methods, for expandStackMethod
}

// parseSource parses the vm files the generator splices from and sorts their
// functions into the three groups it looks things up in.
func parseSource(vmDir string) *source {
	s := &source{
		fset:         token.NewFileSet(),
		opHandlers:   map[string]*ast.FuncDecl{},
		gasHelpers:   map[string]*ast.FuncDecl{},
		stackMethods: map[string]*ast.FuncDecl{},
	}
	for _, name := range []string{"instructions.go", "eips.go", "gascosts.go", "interpreter.go", "stack.go"} {
		path := filepath.Join(vmDir, name)
		f, err := parser.ParseFile(s.fset, path, nil, parser.ParseComments)
		if err != nil {
			abortf("parse %s: %v", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			switch {
			// The gas helpers go by name because three of the four are methods on
			// GasBudget or Contract, so no receiver rule would pick them up.
			case gasHelperNames[fn.Name.Name]:
				s.gasHelpers[fn.Name.Name] = fn

			case methodReceiver(fn) == "Stack":
				s.stackMethods[fn.Name.Name] = fn

			// Handlers are plain functions. stack.go is excluded because its
			// non-method functions are arena plumbing, not opcode handlers.
			case fn.Recv == nil && name != "stack.go":
				s.opHandlers[fn.Name.Name] = fn
			}
		}
	}
	return s
}

// opHandler returns an opcode handler or make* factory by name.
func (s *source) opHandler(name string) *ast.FuncDecl {
	fn := s.opHandlers[name]
	if fn == nil {
		abortf("no handler %q in the parsed vm sources", name)
	}
	return fn
}

// helperBinds says what a spliced helper's receiver and parameters become in the
// dispatch loop. Binding is by role and positional, so nothing here restates an
// identifier: the names being replaced come from the helper's own declaration.
//
// A parameter left unbound keeps the name the helper gave it, which is deliberate.
// computeMemorySize's second parameter is stack and the loop already holds a local
// called stack, so only the first parameter needs a bind.
type helperBinds struct {
	recv   string   // what the receiver becomes, "" to leave it as written
	params []string // what each parameter becomes, in order, "" or absent to leave it
}

// spliceHelper renders a shared gas or memory helper's body against the dispatch's
// own variables, rebinding its receiver and parameters per b. For ChargeExecutionOnly
// at a PUSH1, recv "contract.Gas" and params {"3"} turn
//
//	if g.ExecutionGas < r {
//
// into
//
//	if contract.Gas.ExecutionGas < 3 {
//
// The rewrite renames every mention of a bound name, which is what substituting it
// means. It assumes a helper does not redeclare its own receiver or parameters, and
// none of them do. One that did would emit an assignment to a function or a constant,
// so the generated file would fail to compile rather than mean something else.
//
// The renames are undone before returning, since the same body is spliced once per
// opcode that uses it.
func (g *generator) spliceHelper(helper string, b helperBinds) string {
	decl := g.gasHelper(helper)

	// Pair each name in the declaration with what the dispatch calls it.
	rebind := map[string]string{}
	if b.recv != "" {
		if decl.Recv == nil {
			abortf("cannot splice %s: it has no receiver to bind to %s", helper, b.recv)
		}
		rebind[recvName(decl)] = b.recv
	}
	names := paramNames(decl)
	if len(b.params) > len(names) {
		abortf("cannot splice %s: %d binds for %d parameters", helper, len(b.params), len(names))
	}
	for i, to := range b.params {
		if to != "" {
			rebind[names[i]] = to
		}
	}

	// Rename in place, remembering each node so the body is left as parsed.
	type renamed struct {
		ident *ast.Ident
		was   string
	}
	var undo []renamed
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		id, isIdent := n.(*ast.Ident)
		if !isIdent {
			return true
		}
		if to, ok := rebind[id.Name]; ok {
			undo = append(undo, renamed{id, id.Name})
			id.Name = to
		}
		return true
	})
	defer func() {
		for _, r := range undo {
			r.ident.Name = r.was
		}
	}()
	return g.renderAst(decl.Body.List)
}

// gasHelper returns a spliced gas or memory helper by name.
func (s *source) gasHelper(name string) *ast.FuncDecl {
	fn := s.gasHelpers[name]
	if fn == nil {
		abortf("no gas helper %q in the parsed vm sources", name)
	}
	return fn
}

// stackMethod returns a *Stack method by name.
func (s *source) stackMethod(name string) *ast.FuncDecl {
	fn := s.stackMethods[name]
	if fn == nil {
		abortf("no (*Stack).%s in the parsed vm sources", name)
	}
	return fn
}

// methodReceiver returns the receiver type name of a pointer-receiver method
// (e.g. "Stack" for (s *Stack)), or "" if fn is not such a method.
func methodReceiver(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) != 1 {
		return ""
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return ""
	}
	id, ok := star.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

// spliceOpcodeBody returns a named handler's body, rewritten to run in the loop. For
// opAdd,
//
//	x, y := scope.Stack.pop1Peek1()
//	y.Add(x, y)
//	return nil, nil
//
// becomes
//
//	sp--
//	x := &sd[sp]
//	y := &sd[sp-1]
//	y.Add(x, y)
//	pc++
//	continue mainLoop
//
// renderBody expands the stack call, rewriteOpcodeReturns the return. The caller emits
// the result with p.
func (g *generator) spliceOpcodeBody(handler string) string {
	fn := g.opHandler(handler)
	return useContractLocal(g.rewriteOpcodeReturns(g.renderBody(fn.Body.List)))
}

// useContractLocal points a spliced body at the loop's own contract variable. The
// handlers reach it through the scope they are handed, but the generated loop already
// holds it, so every one of those is a pointer load the switch does not need.
func useContractLocal(src string) string {
	return strings.ReplaceAll(src, "scope.Contract", "contract")
}

// renderAst converts AST statements back to source text, the inverse of parsing. It
// emits nothing: most callers pass the result to p, but renderBody uses it in an error
// message and duplicateMove uses it as a comparison key.
func (g *generator) renderAst(stmts []ast.Stmt) string {
	var raw bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	for _, stmt := range stmts {
		if err := cfg.Fprint(&raw, g.fset, stmt); err != nil {
			abortf("print stmt: %v", err)
		}
		raw.WriteByte('\n')
	}
	return raw.String()
}

// returnRewrite says what a spliced body's return statements become in the
// dispatch loop. The three spliced kinds differ only in these four ways.
type returnRewrite struct {
	results  int    // values the spliced function returns, 1 or 2
	keepData bool   // on failure, carry the returned value into res
	target   string // on success, assign the returned value here, "" to drop it
	advance  bool   // on success, step the pc and go round the loop
}

// rewriteReturns walks a printed body and turns every return into loop control flow,
// per r. Every other line passes through unchanged.
//
// An opcode handler, results 2 with keepData and advance:
//
//	return nil, nil             ->  pc++
//	                                continue mainLoop
//	return nil, ErrInvalidJump  ->  res, err = nil, ErrInvalidJump
//	                                break mainLoop
//
// a gas charge, results 1:
//
//	return nil                  ->  (nothing, fall into the rest of the opcode)
//	return ErrOutOfGas          ->  res, err = nil, ErrOutOfGas
//	                                break mainLoop
//
// a gas step, results 2 with target memorySize:
//
//	return size, nil            ->  memorySize = size
//	return 0, ErrGasUintOverflow -> res, err = nil, ErrGasUintOverflow
//	                                break mainLoop
func (g *generator) rewriteReturns(src string, r returnRewrite) string {
	var out bytes.Buffer
	for _, line := range strings.Split(src, "\n") {
		indent, results, ok := parseReturn(line)
		if !ok {
			out.WriteString(line + "\n")
			continue
		}
		// The rewrite is shaped by how many values the spliced body returns, so a
		// return that disagrees would be turned into the wrong kind of step.
		if len(results) != r.results {
			abortf("cannot rewrite %q: operand count %d, the splice expects %d",
				strings.TrimSpace(line), len(results), r.results)
		}
		// A one-result body returns only an error, so it has no value to carry.
		value, errVal := "nil", results[0]
		if r.results == 2 {
			value, errVal = results[0], results[1]
		}
		if errVal != "nil" { // failure: leave the loop with the error
			data := "nil"
			if r.keepData {
				data = value
			}
			out.WriteString(indent + "res, err = " + data + ", " + errVal + "\n")
			out.WriteString(indent + "break mainLoop\n")
			continue
		}
		// Success: hand the value on, then either step the pc or fall through to
		// the rest of the opcode.
		if r.target != "" {
			out.WriteString(indent + r.target + " = " + value + "\n")
		}
		if r.advance {
			out.WriteString(indent + "pc++\n")
			out.WriteString(indent + "continue mainLoop\n")
		}
	}
	return out.String()
}

// parseReturn splits a printed return statement into its leading indent and its
// operands, reporting false for any other line. Wrapping the operands as call
// arguments hands the split to go/parser, so an operand carrying its own comma, as
// in return f(a, b), nil, stays whole.
func parseReturn(line string) (string, []string, bool) {
	body := strings.TrimLeft(line, " \t")
	rest, found := strings.CutPrefix(body, "return")
	// A name that only starts with the word, such as returnData, is not a return.
	if !found || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return "", nil, false
	}
	src := "_f(" + rest + ")"
	expr, err := parser.ParseExpr(src)
	if err != nil {
		abortf("cannot parse the operands of %q: %v", strings.TrimSpace(line), err)
	}
	call, isCall := expr.(*ast.CallExpr)
	if !isCall {
		abortf("the operands of %q do not read as a list", strings.TrimSpace(line))
	}
	// Slice the originals out by position rather than reprinting, so the operands
	// reach the generated file exactly as the handler wrote them.
	var results []string
	for _, arg := range call.Args {
		results = append(results, strings.TrimSpace(src[arg.Pos()-1:arg.End()-1]))
	}
	return line[:len(line)-len(body)], results, true
}

// rewriteOpcodeReturns rewrites a printed handler body to run inside the loop: `*pc`
// becomes the loop's pc local, a successful return steps the pc, and a failing one
// carries the handler's value alongside the error. Stack calls were already expanded.
func (g *generator) rewriteOpcodeReturns(src string) string {
	src = strings.ReplaceAll(src, "*pc", "pc")
	return g.rewriteReturns(src, returnRewrite{results: 2, keepData: true, advance: true})
}

// rewriteGasReturns rewrites a spliced charge body to run as a gas step: a
// `return <err>` becomes the out-of-gas break, and the trailing `return nil` is
// dropped so the opcode continues into its remaining steps.
func (g *generator) rewriteGasReturns(src string) string {
	return g.rewriteReturns(src, returnRewrite{results: 1})
}

// rewriteStepReturns rewrites a spliced (value, error) gas step: a non-nil error
// breaks, and on success the value lands in target, or is dropped when target is
// empty.
func (g *generator) rewriteStepReturns(src, target string) string {
	return g.rewriteReturns(src, returnRewrite{results: 2, target: target})
}

// renderBody renders a handler body to source, one statement at a time. A stack method
// call becomes its sp/sd lines, anything else is printed as written with any factory
// parameter filled in. Across opAdd's two statements:
//
//	x, y := scope.Stack.pop1Peek1()  ->  sp--
//	                                     x := &sd[sp]
//	                                     y := &sd[sp-1]
//	y.Add(x, y)                      ->  y.Add(x, y)
func (g *generator) renderBody(stmts []ast.Stmt) string {
	var out strings.Builder
	for _, stmt := range stmts {
		if call, ok := g.matchStackCall(stmt); ok {
			out.WriteString(g.expandStackMethod(call))
			continue
		}
		// Any other route to the stack, such as a call nested in a larger
		// expression, would print through and read the stale view, since the
		// loop tracks the stack in sp and sd.
		ast.Inspect(stmt, func(n ast.Node) bool {
			if e, ok := n.(ast.Expr); ok && isStackExpr(e) {
				abortf("handler statement reaches the stack outside a plain method call: %s", g.renderAst([]ast.Stmt{stmt}))
			}
			return true
		})
		out.WriteString(g.renderAst([]ast.Stmt{stmt}))
	}
	return out.String()
}

// recvName returns a method's receiver name (e.g. "s").
func recvName(fn *ast.FuncDecl) string {
	if names := fn.Recv.List[0].Names; len(names) > 0 {
		return names[0].Name
	}
	return ""
}

// paramNames returns a function's parameter names, in order.
func paramNames(fn *ast.FuncDecl) []string {
	var names []string
	for _, f := range fn.Type.Params.List {
		for _, nm := range f.Names {
			names = append(names, nm.Name)
		}
	}
	return names
}
