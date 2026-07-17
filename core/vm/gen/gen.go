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
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// inlineOps selects the opcodes whose handler bodies are spliced inline: the
// hot, fork-stable opcodes with no dynamic gas. Which handler that is comes
// from the per-fork tables via vm.GenForks (see deriveSpecs), not from a
// restated name. Most resolve to a top-level opXxx handler. PUSH3-PUSH32 and
// DUP1-DUP16 resolve to makePush / makeDup closures, so emitInlineOp splices
// the factory body with the per-opcode size instead. Opcodes not selected here
// (or in directCallOps) fall through to the default case, which dispatches via
// the per-fork table.
var inlineOps = func() map[byte]bool {
	m := map[byte]bool{
		0x01: true, 0x02: true, 0x03: true, 0x04: true, 0x05: true, // ADD MUL SUB DIV SDIV
		0x06: true, 0x07: true, 0x08: true, 0x09: true, 0x0b: true, // MOD SMOD ADDMOD MULMOD SIGNEXTEND
		0x10: true, 0x11: true, 0x12: true, 0x13: true, 0x14: true, 0x15: true, // LT GT SLT SGT EQ ISZERO
		0x16: true, 0x17: true, 0x18: true, 0x19: true, 0x1a: true, // AND OR XOR NOT BYTE
		0x1b: true, 0x1c: true, 0x1d: true, 0x1e: true, // SHL SHR SAR CLZ
		0x50: true, 0x56: true, 0x57: true, 0x58: true, 0x59: true, 0x5b: true, // POP JUMP JUMPI PC MSIZE JUMPDEST
		0x5f: true, 0x60: true, 0x61: true, // PUSH0 PUSH1 PUSH2
	}
	for code := 0x62; code <= 0x7f; code++ { // PUSH3-PUSH32
		m[byte(code)] = true
	}
	for code := 0x80; code <= 0x8f; code++ { // DUP1-DUP16
		m[byte(code)] = true
	}
	for code := 0x90; code <= 0x9f; code++ { // SWAP1-SWAP16
		m[byte(code)] = true
	}
	return m
}()

// directCallOps selects the opcodes (dynamic gas, not inlined) whose handler,
// dynamic-gas, and memory-size functions are the same across every fork
// (verified by checkDirectCallStable). They are emitted as direct calls to
// those functions by name, with the names derived from the per-fork tables,
// instead of the indirect operation.* pointer calls in the default case. An
// aliased gas var derives as its underlying function, so MLOAD's charge is
// emitted as pureMemoryGascost rather than through the gasMLoad func var.
var directCallOps = map[byte]bool{
	0x20: true, // KECCAK256
	0x51: true, // MLOAD
	0x52: true, // MSTORE
	0x53: true, // MSTORE8
}

// opSpec holds the per-opcode facts the generator emits from: the constants
// (gas, stack bounds, intro fork) and the FuncForPC names of the opcode's
// handler, dynamic-gas and memory-size functions, all derived from the
// per-fork tables.
type opSpec struct {
	defined  bool
	name     string
	fork     string
	constGas uint64
	minStack int
	maxStack int
	execFn   string
	dynFn    string
	memFn    string
}

type generator struct {
	fset           *token.FileSet
	opcodeHandlers map[string]*ast.FuncDecl
	stackHelpers   map[string]*ast.FuncDecl
	gasHelpers     map[string]*ast.FuncDecl
	specs          [256]opSpec
	buf            *bytes.Buffer
}

// p is the writer of the generated file. Every line of output is appended
// to g.buf through it.
func (g *generator) p(format string, args ...any) {
	format = strings.TrimRight(strings.TrimPrefix(format, "\n"), " \t")
	fmt.Fprintf(g.buf, format, args...)
}

// parseHandlers parses instructions.go, eips.go, stack.go, gascosts.go and
// interpreter.go. It returns the top-level opXxx handlers by name, the
// //gen:inline *Stack helper methods, and the gas/memory helper functions
// (ChargeRegularOnly, computeMemorySize, chargeDynamicGas) whose bodies are
// spliced into the generated dispatch (all by name).
func parseHandlers(vmDir string) (fset *token.FileSet, opcodeHandlers, stackHelpers, gasHelpers map[string]*ast.FuncDecl) {
	fset = token.NewFileSet()
	opcodeHandlers = map[string]*ast.FuncDecl{}
	stackHelpers = map[string]*ast.FuncDecl{}
	gasHelpers = map[string]*ast.FuncDecl{}
	for _, name := range []string{"instructions.go", "eips.go", "stack.go", "gascosts.go", "interpreter.go"} {
		path := filepath.Join(vmDir, name)
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			fatalf("parse %s: %v", path, err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			switch {
			case fn.Name.Name == "ChargeRegularOnly" || fn.Name.Name == "computeMemorySize" || fn.Name.Name == "chargeDynamicGas" || fn.Name.Name == "chargeVerkleCodeChunkGas": // spliced gas/memory helpers
				gasHelpers[fn.Name.Name] = fn
			case fn.Recv == nil: // top-level opXxx handler
				opcodeHandlers[fn.Name.Name] = fn
			case methodReceiver(fn) == "Stack" && hasInlineMarker(fn): // (s *Stack) helper tagged //gen:inline
				stackHelpers[fn.Name.Name] = fn
			}
		}
	}
	return fset, opcodeHandlers, stackHelpers, gasHelpers
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

// hasInlineMarker reports whether fn is tagged //gen:inline, which marks a stack
// helper for splicing into the generated dispatch.
func hasInlineMarker(fn *ast.FuncDecl) bool {
	if fn.Doc == nil {
		return false
	}
	for _, c := range fn.Doc.List {
		if c.Text == "//gen:inline" {
			return true
		}
	}
	return false
}

var opcodeReturnRe = regexp.MustCompile(`^(\s*)return\s+([^,]+),\s*(.+)$`)

// spliceOpcodeBody returns a named handler's body, rewritten so it can be spliced
// into the dispatch loop (see rewriteOpcodeReturns). The caller emits it with p.
func (g *generator) spliceOpcodeBody(handler string) string {
	fn := g.opcodeHandlers[handler]
	if fn == nil {
		fatalf("no handler %q to inline", handler)
	}
	return g.rewriteOpcodeReturns(g.inlineStackHelpers(fn.Body.List, nil))
}

// spliceOpcodeFactoryBody splices the body of the executionFunc closure that a make*
// factory returns, substituting the factory's parameters with the per-opcode
// constants in args (positional, matching the factory signature). This lets
// closure-built handlers (makePush, makeDup) be derived from their single
// definition rather than restated in the generator. The caller emits the
// result with p.
func (g *generator) spliceOpcodeFactoryBody(factory string, args ...int) string {
	fn := g.opcodeHandlers[factory]
	if fn == nil {
		fatalf("no factory %q to inline", factory)
	}
	lit := factoryClosure(factory, fn)
	// Bind the factory parameters to the per-opcode constants, then inline.
	names := paramNames(fn)
	if len(names) != len(args) {
		fatalf("factory %q takes %d params, got %d args", factory, len(names), len(args))
	}
	params := map[string]int{}
	for i, nm := range names {
		params[nm] = args[i]
	}
	return g.rewriteOpcodeReturns(g.inlineStackHelpers(lit.Body.List, params))
}

// factoryClosure returns the executionFunc literal that a make* factory's body
// is a single `return func(...) {...}` of.
func factoryClosure(name string, fn *ast.FuncDecl) *ast.FuncLit {
	if len(fn.Body.List) != 1 {
		fatalf("factory %q body is not a single return", name)
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		fatalf("factory %q does not return a single value", name)
	}
	lit, ok := ret.Results[0].(*ast.FuncLit)
	if !ok {
		fatalf("factory %q does not return a func literal", name)
	}
	return lit
}

// renderAst converts AST statements back to formatted Go source text, the
// inverse of parsing. It uses the generator's fileset and emits nothing itself
// (the caller passes the result to p).
func (g *generator) renderAst(stmts []ast.Stmt) string {
	var raw bytes.Buffer
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	for _, stmt := range stmts {
		if err := cfg.Fprint(&raw, g.fset, stmt); err != nil {
			fatalf("print stmt: %v", err)
		}
		raw.WriteByte('\n')
	}
	return raw.String()
}

// rewriteOpcodeReturns rewrites a printed handler body so it runs inside the
// dispatch loop: the `*pc` dereference becomes the loop's `pc` local, and each
// `return r0, r1` becomes loop control flow. Success (r1 == nil) advances pc
// and continues, an error sets err and breaks. (Stack helpers were already
// inlined by inlineStackHelpers before the body was printed.)
func (g *generator) rewriteOpcodeReturns(src string) string {
	src = strings.ReplaceAll(src, "*pc", "pc")

	var out bytes.Buffer
	for _, line := range strings.Split(src, "\n") {
		if m := opcodeReturnRe.FindStringSubmatch(line); m != nil {
			indent, r0, r1 := m[1], strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
			if r1 == "nil" {
				out.WriteString(indent + "pc++\n")
				out.WriteString(indent + "continue mainLoop\n")
			} else {
				out.WriteString(indent + "res, err = " + r0 + ", " + r1 + "\n")
				out.WriteString(indent + "break mainLoop\n")
			}
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}

var gasReturnRe = regexp.MustCompile(`^(\s*)return\s+(\S.*)$`)

// rewriteGasReturns rewrites a spliced charge body so it runs as a gas step in
// the dispatch loop: a `return <err>` becomes the out-of-gas break, and the
// trailing `return nil` (success) is dropped so the opcode continues.
func (g *generator) rewriteGasReturns(src string) string {
	var out bytes.Buffer
	for _, line := range strings.Split(src, "\n") {
		if m := gasReturnRe.FindStringSubmatch(line); m != nil {
			indent, val := m[1], strings.TrimSpace(m[2])
			if val == "nil" {
				continue // success: fall through to the rest of the op
			}
			out.WriteString(indent + "res, err = nil, " + val + "\n")
			out.WriteString(indent + "break mainLoop\n")
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}

// rewriteStepReturns rewrites a spliced gas-step body's (value, error) returns so
// it runs inline in the dispatch loop: a non-nil error becomes the out-of-gas
// break; on success the value is assigned to target (or dropped when target is
// empty) and the op falls through.
func (g *generator) rewriteStepReturns(src, target string) string {
	var out bytes.Buffer
	for _, line := range strings.Split(src, "\n") {
		if m := opcodeReturnRe.FindStringSubmatch(line); m != nil {
			indent, val, errVal := m[1], strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
			if errVal == "nil" {
				if target != "" {
					out.WriteString(indent + target + " = " + val + "\n")
				}
				continue
			}
			out.WriteString(indent + "res, err = nil, " + errVal + "\n")
			out.WriteString(indent + "break mainLoop\n")
			continue
		}
		out.WriteString(line + "\n")
	}
	return out.String()
}

// stackCall is a matched call to a tagged helper.
type stackCall struct {
	helper string      // helper method name
	lhs    []ast.Expr  // assignment targets, nil for a void call like dup
	tok    token.Token // the assignment token, := or =
	args   []ast.Expr  // call arguments (only dup has one)
}

// matchStackHelper matches a statement that is a single must-expand helper call,
// in one of the two normalized forms: an assignment `lhs... := scope.Stack.H(args)`
// or a bare `scope.Stack.H(args)`.
func (g *generator) matchStackHelper(stmt ast.Stmt) (stackCall, bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 {
			if h, args, ok := g.stackHelperCall(s.Rhs[0]); ok {
				return stackCall{helper: h, lhs: s.Lhs, tok: s.Tok, args: args}, true
			}
		}
	case *ast.ExprStmt:
		if h, args, ok := g.stackHelperCall(s.X); ok {
			return stackCall{helper: h, args: args}, true
		}
	}
	return stackCall{}, false
}

// stackHelperCall unwraps scope.Stack.H(args) where H is a must-expand helper.
func (g *generator) stackHelperCall(e ast.Expr) (helper string, args []ast.Expr, ok bool) {
	call, isCall := e.(*ast.CallExpr)
	if !isCall {
		return "", nil, false
	}
	sel, isSel := call.Fun.(*ast.SelectorExpr) // <recv>.H
	if !isSel || g.stackHelpers[sel.Sel.Name] == nil || !isStackExpr(sel.X) {
		return "", nil, false
	}
	return sel.Sel.Name, call.Args, true
}

// isStackExpr reports whether e is the stack receiver: the `stack` local or
// scope.Stack.
func isStackExpr(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name == "stack"
	case *ast.SelectorExpr:
		return x.Sel.Name == "Stack"
	}
	return false
}

// inlineStackHelpers renders a handler body to source, inlining every must-expand
// helper call and printing other statements unchanged. params maps the factory
// parameters (makePush/makeDup) to their per-opcode constants.
func (g *generator) inlineStackHelpers(stmts []ast.Stmt, params map[string]int) string {
	var out strings.Builder
	// Walk the handler body one statement at a time. A statement that is a
	// tagged stack-helper call gets the helper's body spliced in: the generated
	// dispatch is past Go's big-function inline budget, so the call would not be
	// inlined otherwise. Every other statement is printed as written.
	for _, stmt := range stmts {
		if call, ok := g.matchStackHelper(stmt); ok {
			// e.g. `x, y := scope.Stack.pop1Peek1()` becomes the body of pop1Peek1.
			out.WriteString(g.inlineStackHelper(call, params))
		} else {
			// A plain statement: print it verbatim, then fill in any makePush or
			// makeDup factory params with this opcode's constants.
			out.WriteString(substParams(g.renderAst([]ast.Stmt{stmt}), params))
		}
	}
	return out.String()
}

// substParams replaces each factory parameter with its constant. It runs only
// on printed non-helper statements and on helper arguments, never on a helper
// expansion, so it cannot touch a field like stack.size. The parameter names do
// not textually overlap, so map order does not affect the result.
func substParams(src string, params map[string]int) string {
	for name, val := range params {
		src = regexp.MustCompile(`\b`+name+`\b`).ReplaceAllString(src, fmt.Sprint(val))
	}
	return src
}

// inlineStackHelper expands one helper call to its stack.go body. The single
// rule: the helper is straight-line statements then an optional final return
// whose result count matches the call's targets. Anything else is not in
// inlinable form and is a hard error (the shape post-condition). The receiver
// maps to the loop's `stack` local and each parameter to its call argument.
func (g *generator) inlineStackHelper(call stackCall, params map[string]int) string {
	fn := g.stackHelpers[call.helper]
	if fn == nil {
		fatalf("no stack helper %q to inline", call.helper)
	}
	// Peel an optional trailing return off the body.
	body := fn.Body.List
	var ret *ast.ReturnStmt
	if n := len(body); n > 0 {
		if r, isRet := body[n-1].(*ast.ReturnStmt); isRet {
			ret, body = r, body[:n-1]
		}
	}
	results := 0
	if ret != nil {
		results = len(ret.Results)
	}
	if len(call.lhs) != results {
		fatalf("stack helper %q returns %d values, call assigns %d", call.helper, results, len(call.lhs))
	}
	// Map the receiver to the loop local and each parameter to its argument.
	names := paramNames(fn)
	if len(names) != len(call.args) {
		fatalf("stack helper %q takes %d params, call passes %d", call.helper, len(names), len(call.args))
	}
	subst := map[string]string{recvName(fn): "stack"}
	for i, name := range names {
		subst[name] = substParams(renderInlineExpr(call.args[i], nil), params)
	}
	// The leading bookkeeping statements, then bind each return expression to
	// its assignment target.
	var out strings.Builder
	for _, stmt := range body {
		out.WriteString(renderInlineStmt(stmt, subst) + "\n")
	}
	for i, lhs := range call.lhs {
		out.WriteString(renderInlineExpr(lhs, nil) + " " + call.tok.String() + " " + renderInlineExpr(ret.Results[i], subst) + "\n")
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

// renderInlineStmt prints one helper-body statement with subst applied. Only the
// statement shapes the helpers use are handled; any other is not inlinable.
func renderInlineStmt(stmt ast.Stmt, subst map[string]string) string {
	switch s := stmt.(type) {
	case *ast.IncDecStmt: // s.inner.top++
		return renderInlineExpr(s.X, subst) + s.Tok.String()
	case *ast.AssignStmt: // s.size -= 2, data[x] = data[y], or the swap tuple a, b = b, a
		if len(s.Lhs) == len(s.Rhs) && len(s.Lhs) >= 1 {
			lhs := make([]string, len(s.Lhs))
			rhs := make([]string, len(s.Rhs))
			for i := range s.Lhs {
				lhs[i] = renderInlineExpr(s.Lhs[i], subst)
				rhs[i] = renderInlineExpr(s.Rhs[i], subst)
			}
			return strings.Join(lhs, ", ") + " " + s.Tok.String() + " " + strings.Join(rhs, ", ")
		}
	}
	fatalf("inline: unsupported statement %T in stack helper", stmt)
	return ""
}

// renderInlineExpr prints one helper-body expression, substituting any
// identifier found in subst. Only the shapes the helpers use are handled.
func renderInlineExpr(expr ast.Expr, subst map[string]string) string {
	switch e := expr.(type) {
	case *ast.Ident:
		if r, ok := subst[e.Name]; ok {
			return r
		}
		return e.Name
	case *ast.BasicLit:
		return e.Value
	case *ast.SelectorExpr: // x.field
		return renderInlineExpr(e.X, subst) + "." + e.Sel.Name
	case *ast.IndexExpr: // x[i]
		return renderInlineExpr(e.X, subst) + "[" + renderInlineExpr(e.Index, subst) + "]"
	case *ast.BinaryExpr: // x op y
		return renderInlineExpr(e.X, subst) + " " + e.Op.String() + " " + renderInlineExpr(e.Y, subst)
	case *ast.UnaryExpr: // &x
		return e.Op.String() + renderInlineExpr(e.X, subst)
	}
	fatalf("inline: unsupported expression %T in stack helper", expr)
	return ""
}

// deriveSpecs records each opcode's constant values (name, intro fork, static
// gas, stack bounds) and its handler, dynamic-gas and memory-size function
// names from the first fork that defines it, then checks that the opcodes
// chosen for inlining and direct-calling are safe to emit from those specs by
// verifying they are fork-stable (see checkStable and checkDirectCallStable).
func (g *generator) deriveSpecs(forks []vm.GenFork) {
	for code := range 256 {
		for _, fork := range forks {
			o := fork.Ops[code]
			if !o.Defined {
				continue
			}
			g.specs[code] = opSpec{
				defined:  true,
				name:     o.Name,
				fork:     fork.RuleField,
				constGas: o.ConstantGas,
				minStack: o.MinStack,
				maxStack: o.MaxStack,
				execFn:   o.ExecuteFn,
				dynFn:    o.DynamicGasFn,
				memFn:    o.MemorySizeFn,
			}
			break // first fork that defines it wins (its intro fork)
		}
	}

	// Every inlined opcode must be defined and keep the same handler and static
	// gas / stack bounds across all forks where it appears. Bail loudly otherwise.
	for code := range inlineOps {
		g.checkStable(code, forks)
	}

	// directCallOps opcodes emit their static gas and stack bounds as constants the
	// same way, so they must be fork-stable too. Dynamic gas is allowed (it is
	// charged through the named gas function, not a constant).
	for code := range directCallOps {
		g.checkDirectCallStable(code, forks)
	}
}

// checkStable verifies an opcode selected for inlining is safe to inline: it must
// be defined, its handler and its static gas and stack bounds must be the same
// across every fork it appears in (the body and constants are emitted from the
// first defining fork's spec), and it must have no dynamic gas, since an inlined
// op charges only its constant static gas. It bails loudly otherwise.
func (g *generator) checkStable(code byte, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		fatalf("opcode %#x selected for inlining but never defined", code)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		if o.ExecuteFn != spec.execFn || o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack || o.DynamicGasFn != "" {
			fatalf("opcode %#x (%s) is not fork-stable (fork %s): cannot inline", code, spec.name, fork.Name)
		}
	}
}

// checkDirectCallStable verifies a directCallOps opcode is safe to direct-call. Its static
// gas and stack bounds must be the same across every fork it appears in (they are
// emitted as constants), and its handler, gas and memory functions must be the same
// across those forks too (they are called by the first defining fork's names, so a
// fork that swapped one would otherwise be missed). Unlike checkStable it allows
// dynamic gas, which directCallOps ops carry by definition.
func (g *generator) checkDirectCallStable(code byte, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		fatalf("opcode %#x (directCallOps) is never defined", code)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		if o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack {
			fatalf("opcode %#x (%s) is in directCallOps but not fork-stable (fork %s): static gas or stack bounds vary, cannot emit as constants", code, spec.name, fork.Name)
		}
		if o.ExecuteFn != spec.execFn || o.DynamicGasFn != spec.dynFn || o.MemorySizeFn != spec.memFn {
			fatalf("opcode %#x (%s) is in directCallOps but its functions vary by fork (fork %s): got %s/%s/%s, want %s/%s/%s, cannot direct-call",
				code, spec.name, fork.Name, o.ExecuteFn, o.DynamicGasFn, o.MemorySizeFn, spec.execFn, spec.dynFn, spec.memFn)
		}
	}
}

// generateStackChecks returns the underflow/overflow guards, mirroring the legacy
// loop's order (stack validated before gas). minExpr and maxExpr are the
// stack-bound expressions (constants on the inlined/direct paths,
// operation.minStack/maxStack in the table path). under and over select which
// guards to emit, so those paths can omit a guard whose bound is trivial.
func (g *generator) generateStackChecks(minExpr, maxExpr any, under, over bool) string {
	switch {
	case under && over:
		return fmt.Sprintf(`if sLen := stack.len(); sLen < %v {
	return nil, &ErrStackUnderflow{stackLen: sLen, required: %v}
} else if sLen > %v {
	return nil, &ErrStackOverflow{stackLen: sLen, limit: %v}
}
`, minExpr, minExpr, maxExpr, maxExpr)
	case under:
		return fmt.Sprintf(`if sLen := stack.len(); sLen < %v {
	return nil, &ErrStackUnderflow{stackLen: sLen, required: %v}
}
`, minExpr, minExpr)
	case over:
		return fmt.Sprintf(`if sLen := stack.len(); sLen > %v {
	return nil, &ErrStackOverflow{stackLen: sLen, limit: %v}
}
`, maxExpr, maxExpr)
	}
	return ""
}

// generateStaticGas returns the static-gas charge, spliced call-free from the
// ChargeRegularOnly body for amount: a constant on the inlined and
// direct-call paths, operation.constantGas in the table path. The receiver maps
// to contract.Gas and the method's single uint64 parameter to amount, substituted
// textually on word boundaries (which cannot touch fields like RegularGas). Its
// `return <err>` becomes the loop's out-of-gas exit and its trailing `return nil`
// is dropped so the opcode falls through to its remaining steps (see
// rewriteGasReturns).
func (g *generator) generateStaticGas(amount any) string {
	fn := g.gasHelpers["ChargeRegularOnly"]
	if fn == nil {
		fatalf("no ChargeRegularOnly gas helper to inline")
	}
	names := paramNames(fn)
	if len(names) != 1 {
		fatalf("ChargeRegularOnly takes %d params, want 1", len(names))
	}
	src := g.renderAst(fn.Body.List)
	src = regexp.MustCompile(`\b`+recvName(fn)+`\b`).ReplaceAllString(src, "contract.Gas")
	src = regexp.MustCompile(`\b`+names[0]+`\b`).ReplaceAllString(src, fmt.Sprint(amount))
	return g.rewriteGasReturns(src)
}

// closureSegRe matches the anonymous trailing segments of a closure's
// FuncForPC name, "func31" or a nested "2".
var closureSegRe = regexp.MustCompile(`^(func\d+|\d+)$`)

// factoryName returns the factory a closure-built handler was created by
// (e.g. "makeDup" for "newFrontierInstructionSet.makeDup.func37"), or "" for
// a plain top-level handler name.
func factoryName(fn string) string {
	segs := strings.Split(fn, ".")
	n := len(segs)
	for n > 0 && closureSegRe.MatchString(segs[n-1]) {
		n--
	}
	if n == len(segs) || n == 0 {
		return ""
	}
	return segs[n-1]
}

// emitInlineOp emits an inlined opcode case: the stack and gas guards followed by
// the spliced opcode body. A fork-introduced opcode wraps that body in a fork gate
// so it runs only when the opcode is active for the current fork, otherwise the
// case mirrors the legacy loop's undefined-opcode handling.
func (g *generator) emitInlineOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)
	if spec.fork != "" {
		g.p("if rules.%s {\n", spec.fork)
	}

	// stack bounds check
	g.p("%s", g.generateStackChecks(spec.minStack, spec.maxStack, spec.minStack > 0, spec.maxStack < int(params.StackLimit)))

	// static gas
	if spec.constGas != 0 {
		g.p("%s", g.generateStaticGas(spec.constGas))
	}

	// PUSH1-PUSH32 swap their execute function under EIP-4762 (verkle) to charge
	// code-chunk gas on the immediate bytes. Defer to the table handler there.
	// The constant static gas and stack guard above already match.
	if code >= 0x60 && code <= 0x7f {
		g.p(`
			if isEIP4762 {
				res, err = table[op].execute(&pc, evm, scope)
				if err != nil {
					break mainLoop
				}
				pc++
				continue mainLoop
			}
		`)
	}

	// opcode body
	switch factory := factoryName(spec.execFn); factory {
	case "makePush": // PUSH3-PUSH32: splice makePush(size, size)
		n := int(code) - 0x5f
		g.p("%s", g.spliceOpcodeFactoryBody("makePush", n, n))
	case "makeDup": // DUP1-DUP16: splice makeDup(n)
		g.p("%s", g.spliceOpcodeFactoryBody("makeDup", int(code)-0x7f))
	case "": // the rest: splice the opXxx handler body
		g.p("%s", g.spliceOpcodeBody(spec.execFn))
	default:
		fatalf("opcode %#x (%s) is built by factory %q, which the generator cannot inline", code, spec.name, factory)
	}

	// If opcode is inactive for this fork, then close the gate
	// and fall back to the legacy loop's undefined-opcode handling.
	if spec.fork != "" {
		g.p(`
	    }
		  res, err = opUndefined(&pc, evm, scope)
			break mainLoop
		`)
	}
}

// emitDirectCallOp emits an opcode case identical to the default case, except
// the handler, dynamic-gas, and memory-size functions are called by name
// rather than through the indirect operation.* table pointers. Valid only for
// fork-invariant ops (see directCallOps).
func (g *generator) emitDirectCallOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)

	// stack bounds check
	g.p("%s", g.generateStackChecks(spec.minStack, spec.maxStack, spec.minStack > 0, spec.maxStack < int(params.StackLimit)))

	// static gas
	if spec.constGas != 0 {
		g.p("%s", g.generateStaticGas(spec.constGas))
	}

	// dynamic gas
	g.p("\nvar memorySize uint64\n")

	// Splice computeMemorySize's body, rewriting its operation.memorySize lookup to
	// the opcode's memory-size function and its returns for the dispatch loop.
	memSizeFn := g.gasHelpers["computeMemorySize"]
	if memSizeFn == nil {
		fatalf("no computeMemorySize gas helper to inline")
	}
	memSizeSrc := g.renderAst(memSizeFn.Body.List)
	memSizeSrc = strings.ReplaceAll(memSizeSrc, "operation.memorySize", spec.memFn)
	g.p("%s", g.rewriteStepReturns(memSizeSrc, "memorySize"))

	// Splice chargeDynamicGas's body the same way, rewriting operation.dynamicGas to
	// the opcode's gas function.
	dynGasFn := g.gasHelpers["chargeDynamicGas"]
	if dynGasFn == nil {
		fatalf("no chargeDynamicGas gas helper to inline")
	}
	dynGasSrc := g.renderAst(dynGasFn.Body.List)
	dynGasSrc = strings.ReplaceAll(dynGasSrc, "operation.dynamicGas", spec.dynFn)
	g.p("%s", g.rewriteStepReturns(dynGasSrc, ""))

	// resize memory
	g.p(`
		if memorySize > 0 {
			mem.Resize(memorySize)
		}
	`)

	// call the opcode handler
	g.p(`
		res, err = %s(&pc, evm, scope)
		if err != nil {
			break mainLoop
		}
	`, spec.execFn)

	// advance to the next opcode
	g.p(`
		pc++
		continue mainLoop
	`)
}

// emitDefault emits the switch's default case: every opcode not inlined or
// direct-called (the fork-varying ops such as CALL, CREATE, SSTORE, SLOAD, LOG
// and the COPY family) is dispatched through the active per-fork table, exactly
// as the legacy loop did, so their volatile gas and opcode logic stays shared
// rather than restated here.
func (g *generator) emitDefault() {
	g.p(`
		default:
			operation := table[op]
	`)
	// stack bounds check
	g.p("%s", g.generateStackChecks("operation.minStack", "operation.maxStack", true, true))

	// static gas
	g.p("%s", g.generateStaticGas("operation.constantGas"))

	// dynamic gas
	g.p(`
			var memorySize uint64
			if memorySize, _, err = contract.meterDynamicGas(operation, evm, stack, mem); err != nil {
				return nil, err
			}
	`)

	// resize memory
	g.p(`
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
	`)

	// call the opcode handler
	g.p(`
			res, err = operation.execute(&pc, evm, scope)
			if err != nil {
				break mainLoop
			}
	`)

	// advance to the next opcode
	g.p(`
			pc++
			continue mainLoop
	`)
}

// createFile emits the whole generated file into g.buf: the header, package and
// imports, then the execUntraced function (its locals and dispatch loop, the
// verkle code-chunk gas, and a switch with one case per opcode built by the
// emit* helpers). main formats the buffer and writes it to interpreter_gen.go.
//
// The switch has three tiers:
//
//   - the hot, fork-stable opcodes (arithmetic / comparison / bitwise / PUSH /
//     DUP / SWAP / POP / JUMP / JUMPI / PC / MSIZE / JUMPDEST) are inlined by
//     splicing the existing opXxx handler bodies from instructions.go and
//     eips.go, with their static gas and stack bounds emitted as constants
//     derived from the per-fork instruction tables via vm.GenForks.
//
//   - the fork-invariant ops (KECCAK256 / MLOAD / MSTORE / MSTORE8, see
//     directCallOps) are called directly by name, skipping the table's function
//     pointers, which Go cannot inline through.
//
//   - everything fork-varying (CALL / CREATE / SSTORE / SLOAD / LOG / the COPY
//     family and so on) is dispatched through the active per-fork JumpTable in
//     the default case, exactly as the legacy loop did, so volatile gas and
//     opcode logic stays shared rather than restated.
func (g *generator) createFile() {
	// file header, package clause, and imports
	g.p(`
		// Code generated by core/vm/gen; DO NOT EDIT.

		package vm

		import (
			"fmt"

			"github.com/ethereum/go-ethereum/common/math"
			"github.com/ethereum/go-ethereum/core/tracing"
		)

	`)

	// execUntraced: doc comment, loop-local declarations, and the dispatch loop
	g.p(`
		// execUntraced is the generated, tracing-free interpreter fast path. Hot,
		// fork-stable opcodes are inlined with their static gas and stack bounds emitted
		// as constants. Fork-invariant ops (KECCAK256/MLOAD/MSTORE/MSTORE8) call their
		// handler and gas functions directly by name. Everything fork-varying is
		// dispatched through the active per-fork table in the default case. EVM.Run
		// selects this path when no tracer is configured.
		func (evm *EVM) execUntraced(scope *ScopeContext) (ret []byte, err error) {
			var (
				contract  = scope.Contract
				mem       = scope.Memory
				stack     = scope.Stack
				table     = evm.table
				rules     = evm.chainRules
				isEIP4762 = rules.IsEIP4762
				pc        = uint64(0)
				res       []byte
			)
			_ = mem
			_ = rules
			_ = isEIP4762
			_ = table
		mainLoop:
			for {
	`)

	// verkle code-chunk gas, spliced from chargeVerkleCodeChunkGas
	ccgFn := g.gasHelpers["chargeVerkleCodeChunkGas"]
	if ccgFn == nil {
		fatalf("no chargeVerkleCodeChunkGas gas helper to inline")
	}
	g.p("%s", g.rewriteGasReturns(g.renderAst(ccgFn.Body.List)))

	// fetch the opcode and open the dispatch switch
	g.p(`
				op := contract.GetOp(pc)
				switch op {
	`)

	// one case per inlined or direct-call opcode, in opcode order
	for code := range 256 {
		b := byte(code)
		if inlineOps[b] {
			g.emitInlineOp(b)
		} else if directCallOps[b] {
			g.emitDirectCallOp(b)
		}
	}

	// the default case: fork-varying ops via the per-fork table
	g.emitDefault()

	// close the switch and loop, clear the stop token, and return
	g.p(`
				}
			}
			if err == errStopToken {
				err = nil
			}
			return res, err
		}
	`)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}

// vmDir returns the core/vm directory, the parent of this generator package. It
// is resolved from this source file's own path so it does not depend on the
// directory the generator or the test happens to run from.
func vmDir() string {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fatalf("cannot resolve generator source path")
	}
	return filepath.Dir(filepath.Dir(self)) // .../core/vm/gen -> .../core/vm
}

// generate parses the opcode, gas and fork definitions under core/vm and returns
// the formatted contents of interpreter_gen.go. It is the shared core of the
// generator: main writes the result to disk, and the up-to-date test in
// gen_test.go compares it against the committed file.
func generate() ([]byte, error) {
	fset, opcodeHandlers, stackHelpers, gasHelpers := parseHandlers(vmDir())
	g := &generator{fset: fset, opcodeHandlers: opcodeHandlers, stackHelpers: stackHelpers, gasHelpers: gasHelpers, buf: new(bytes.Buffer)}
	g.deriveSpecs(vm.GenForks())
	g.createFile()

	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		dbg := filepath.Join(vmDir(), "interpreter_gen.go.broken")
		os.WriteFile(dbg, g.buf.Bytes(), 0644)
		return nil, fmt.Errorf("gofmt failed (%v); wrote unformatted output to %s", err, dbg)
	}
	return formatted, nil
}
