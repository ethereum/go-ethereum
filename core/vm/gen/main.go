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

// Command gen generates core/vm/interp_gen.go, the EVM interpreter's untraced
// fast-path dispatch. The output is a switch over the opcode byte that:
//
//   - inlines the hot, fork-stable opcodes (arithmetic / comparison / bitwise /
//     PUSH / DUP / SWAP / POP / JUMP / JUMPI / PC / MSIZE / JUMPDEST) by
//     splicing the existing opXxx handler bodies from instructions.go and
//     eips.go, with their static gas and stack bounds baked in as constants
//     derived from the per-fork instruction tables via vm.GenForks.
//
//   - calls the fork-invariant ops (KECCAK256 / MLOAD / MSTORE / MSTORE8,
//     see directCallOps) directly by name, skipping the table's function
//     pointers, which Go cannot inline through.
//
//   - dispatches everything fork-varying (CALL / CREATE / SSTORE / SLOAD / LOG /
//     the COPY family and so on) through the active per-fork JumpTable in the
//     default case, exactly as the legacy loop did, so volatile gas and opcode
//     logic stays shared rather than restated.
//
// The generated file is committed and a CI test asserts it matches `go generate`
// output. Do not hand-edit interp_gen.go.
//
// Usage: go generate ./core/vm/...
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
)

const stackLimit = 1024 // params.StackLimit

// inlineOps maps an opcode byte to the handler whose body is spliced inline
// for that opcode. These are the hot, fork-stable opcodes with no dynamic gas.
// The value is usually an opXxx handler, but PUSH3-PUSH32 and DUP1-DUP16 are
// factory-built (one shared makePush / makeDup each), so their value is the
// factory name and emitOpBody splices the factory body with the per-opcode size.
// Opcodes not listed here (or in directCallOps) fall through to the default case,
// which dispatches via the per-fork table.
var inlineOps = func() map[byte]string {
	m := map[byte]string{
		0x01: "opAdd", 0x02: "opMul", 0x03: "opSub", 0x04: "opDiv", 0x05: "opSdiv",
		0x06: "opMod", 0x07: "opSmod", 0x08: "opAddmod", 0x09: "opMulmod", 0x0b: "opSignExtend",
		0x10: "opLt", 0x11: "opGt", 0x12: "opSlt", 0x13: "opSgt", 0x14: "opEq", 0x15: "opIszero",
		0x16: "opAnd", 0x17: "opOr", 0x18: "opXor", 0x19: "opNot", 0x1a: "opByte",
		0x1b: "opSHL", 0x1c: "opSHR", 0x1d: "opSAR", 0x1e: "opCLZ",
		0x50: "opPop", 0x56: "opJump", 0x57: "opJumpi", 0x58: "opPc", 0x59: "opMsize", 0x5b: "opJumpdest",
		0x5f: "opPush0", 0x60: "opPush1", 0x61: "opPush2",
	}
	for code := 0x62; code <= 0x7f; code++ { // PUSH3-PUSH32
		m[byte(code)] = "makePush"
	}
	for code := 0x80; code <= 0x8f; code++ { // DUP1-DUP16
		m[byte(code)] = "makeDup"
	}
	for code := 0x90; code <= 0x9f; code++ { // SWAP1-SWAP16
		m[byte(code)] = fmt.Sprintf("opSwap%d", code-0x8f)
	}
	return m
}()

// directCallOps lists the opcodes (dynamic gas, not inlined) whose handler,
// dynamic-gas, and memory-size functions are the same across every fork
// (verified: untouched by any enableXxx). They are emitted as direct calls to
// those functions by name instead of the indirect operation.* pointer calls
// in the default case.
var directCallOps = map[byte][3]string{
	0x20: {"opKeccak256", "gasKeccak256", "memoryKeccak256"}, // KECCAK256
	0x51: {"opMload", "gasMLoad", "memoryMLoad"},             // MLOAD
	0x52: {"opMstore", "gasMStore", "memoryMStore"},          // MSTORE
	0x53: {"opMstore8", "gasMStore8", "memoryMStore8"},       // MSTORE8
}

// opSpec holds the per-opcode constants the generator bakes (gas, stack bounds, intro fork), derived from the per-fork tables.
type opSpec struct {
	defined  bool
	name     string // opcode mnemonic, e.g. "ADD"
	fork     string
	constGas uint64
	minStack int
	maxStack int
}

type generator struct {
	fset         *token.FileSet
	handlers     map[string]*ast.FuncDecl // opXxx handlers from instructions.go and eips.go
	stackHelpers map[string]*ast.FuncDecl // (s *Stack) helpers from stack.go, spliced inline
	gasHelpers   map[string]*ast.FuncDecl // (g *GasBudget) charge methods from gascosts.go, spliced by name
	specs        [256]opSpec
	buf          *bytes.Buffer
}

// p is the writer of the generated file. Every line of output is appended
// to g.buf through it.
func (g *generator) p(format string, args ...any) {
	format = strings.TrimRight(strings.TrimPrefix(format, "\n"), " \t")
	fmt.Fprintf(g.buf, format, args...)
}

// ---------------------------------------------------------------------------
// Handler parsing + body splicing
// ---------------------------------------------------------------------------

// parseHandlers parses instructions.go, eips.go, stack.go and gascosts.go. It
// returns the top-level opXxx handlers by name, the //gen:inline *Stack helper
// methods by name (spliced into handler bodies), and the *GasBudget charge
// methods by name (spliced directly at gas steps).
func parseHandlers(vmDir string) (fset *token.FileSet, handlers, stackHelpers, gasHelpers map[string]*ast.FuncDecl) {
	fset = token.NewFileSet()
	handlers = map[string]*ast.FuncDecl{}
	stackHelpers = map[string]*ast.FuncDecl{}
	gasHelpers = map[string]*ast.FuncDecl{}
	for _, name := range []string{"instructions.go", "eips.go", "stack.go", "gascosts.go"} {
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
			case fn.Recv == nil: // top-level opXxx handler
				handlers[fn.Name.Name] = fn
			case methodReceiver(fn) == "Stack" && hasInlineMarker(fn): // (s *Stack) helper tagged //gen:inline
				stackHelpers[fn.Name.Name] = fn
			case methodReceiver(fn) == "GasBudget": // (g *GasBudget) charge method, spliced by name at gas steps
				gasHelpers[fn.Name.Name] = fn
			}
		}
	}
	return fset, handlers, stackHelpers, gasHelpers
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

// inlineOpcodeBody returns a named handler's body, rewritten so it can be spliced
// into the dispatch loop (see rewriteOpcodeReturns). The caller emits it with p.
func (g *generator) inlineOpcodeBody(handler string) string {
	fn := g.handlers[handler]
	if fn == nil {
		fatalf("no handler %q to inline", handler)
	}
	return g.rewriteOpcodeReturns(g.inlineStackHelpers(fn.Body.List, nil))
}

// inlineOpcodeFactoryBody splices the body of the executionFunc closure that a make*
// factory returns, substituting the factory's parameters with the per-opcode
// constants in args (positional, matching the factory signature). This lets
// closure-built handlers (makePush, makeDup) be derived from their single
// definition rather than restated in the generator. The caller emits the
// result with p.
func (g *generator) inlineOpcodeFactoryBody(factory string, args ...int) string {
	fn := g.handlers[factory]
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

// inlineGasBody splices a (g *GasBudget) charge method's body at a gas step,
// mapping the receiver to contract.Gas and the method's single uint64 parameter
// to the per-opcode gas constant. It is the gas-step analog of inlineOpcodeBody: the
// method's `return <err>` becomes the loop's out-of-gas exit and its trailing
// `return nil` is dropped so the opcode falls through to its remaining steps (see
// rewriteGasReturns). The receiver and parameter are substituted textually on
// word boundaries, which cannot touch fields like RegularGas.
func (g *generator) inlineGasBody(name, amount string) string {
	fn := g.gasHelpers[name]
	if fn == nil {
		fatalf("no gas helper %q to inline", name)
	}
	names := paramNames(fn)
	if len(names) != 1 {
		fatalf("gas helper %q takes %d params, want 1", name, len(names))
	}
	src := g.renderAst(fn.Body.List)
	src = regexp.MustCompile(`\b`+recvName(fn)+`\b`).ReplaceAllString(src, "contract.Gas")
	src = regexp.MustCompile(`\b`+names[0]+`\b`).ReplaceAllString(src, amount)
	return g.rewriteGasReturns(src)
}

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

// ---------------------------------------------------------------------------
// Spec derivation (from the per-fork tables, via vm.GenForks)
// ---------------------------------------------------------------------------

func (g *generator) deriveSpecs(forks []vm.GenFork) {
	for code := 0; code < 256; code++ {
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
			}
			break // first fork that defines it wins (its intro fork)
		}
	}
	// Sanity: every inlined opcode must be defined and have fork-stable static
	// gas / stack bounds across all forks where it appears (that is what makes
	// it safe to bake as a constant). Bail loudly otherwise.
	for code, handler := range inlineOps {
		g.checkStable(code, handler, forks)
	}
	// directCallOps opcodes bake their static gas and stack bounds the same way, so
	// they must be fork-stable too. Dynamic gas is allowed (it is charged through
	// the named gas function, not baked).
	for code := range directCallOps {
		g.checkDirectCallStable(code, forks)
	}
}

func (g *generator) checkStable(code byte, what string, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		fatalf("opcode %#x (%s) selected for inlining but never defined", code, what)
	}
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		if o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack || o.DynamicGasFn != "" {
			fatalf("opcode %#x (%s) is not fork-stable (fork %s): cannot inline", code, what, fork.Name)
		}
	}
}

// checkDirectCallStable verifies a directCallOps opcode is safe to direct-call. Its static
// gas and stack bounds must be the same across every fork it appears in (they are
// baked as constants), and its handler, gas and memory functions must be the same
// across those forks too (they are called by name, so a fork that swapped one
// would otherwise be missed). Unlike checkStable it allows dynamic gas, which
// directCallOps ops carry by definition. It does not check the directCallOps map's names
// against the table, which the differential test covers.
func (g *generator) checkDirectCallStable(code byte, forks []vm.GenFork) {
	spec := g.specs[code]
	if !spec.defined {
		fatalf("opcode %#x (directCallOps) is never defined", code)
	}
	var exec, dyn, mem string
	seen := false
	for _, fork := range forks {
		o := fork.Ops[code]
		if !o.Defined {
			continue
		}
		if o.ConstantGas != spec.constGas || o.MinStack != spec.minStack || o.MaxStack != spec.maxStack {
			fatalf("opcode %#x (%s) is in directCallOps but not fork-stable (fork %s): static gas or stack bounds vary, cannot bake", code, spec.name, fork.Name)
		}
		// Handler, gas and memory functions must match across forks too, or
		// direct-calling them by name would skip a fork that swapped one. Names
		// come from FuncForPC via vm.GenForks (aliases resolve to the underlying
		// func, still stable across forks).
		if !seen {
			exec, dyn, mem, seen = o.ExecuteFn, o.DynamicGasFn, o.MemorySizeFn, true
		} else if o.ExecuteFn != exec || o.DynamicGasFn != dyn || o.MemorySizeFn != mem {
			fatalf("opcode %#x (%s) is in directCallOps but its functions vary by fork (fork %s): got %s/%s/%s, want %s/%s/%s, cannot direct-call",
				code, spec.name, fork.Name, o.ExecuteFn, o.DynamicGasFn, o.MemorySizeFn, exec, dyn, mem)
		}
	}
}

// ---------------------------------------------------------------------------
// Case emission
// ---------------------------------------------------------------------------

// emitStackChecks emits the underflow/overflow guards, mirroring the legacy
// loop's order (stack validated before gas). minExpr/maxExpr are the stack-bound
// expressions (baked constants on the inlined/direct paths, operation.minStack/
// operation.maxStack in the table path) and under/over select which guards to
// emit; the baked paths omit a guard whose bound is trivial.
func (g *generator) emitStackChecks(minExpr, maxExpr string, under, over bool) {
	switch {
	case under && over:
		g.p(`
			if sLen := stack.len(); sLen < %s {
				return nil, &ErrStackUnderflow{stackLen: sLen, required: %s}
			} else if sLen > %s {
				return nil, &ErrStackOverflow{stackLen: sLen, limit: %s}
			}
		`, minExpr, minExpr, maxExpr, maxExpr)
	case under:
		g.p(`
			if sLen := stack.len(); sLen < %s {
				return nil, &ErrStackUnderflow{stackLen: sLen, required: %s}
			}
		`, minExpr, minExpr)
	case over:
		g.p(`
			if sLen := stack.len(); sLen > %s {
				return nil, &ErrStackOverflow{stackLen: sLen, limit: %s}
			}
		`, maxExpr, maxExpr)
	}
}

// emitStaticGas charges static gas by splicing the chargeRegularOnly body inline
// (call-free) for amount: a baked constant on the inlined and direct-call paths,
// operation.constantGas in the table path. It is the static-gas counterpart to
// emitDynamicGas.
func (g *generator) emitStaticGas(amount string) {
	g.p("%s", g.inlineGasBody("chargeRegularOnly", amount))
}

// emitOpBody emits the stack/gas guards and the opcode body (the portion that runs
// when the opcode is active for the current fork).
func (g *generator) emitOpBody(code byte) {
	spec := g.specs[code]
	g.emitStackChecks(fmt.Sprint(spec.minStack), fmt.Sprint(spec.maxStack), spec.minStack > 0, spec.maxStack < stackLimit)
	if spec.constGas != 0 {
		g.emitStaticGas(fmt.Sprint(spec.constGas))
	}

	// PUSH1-PUSH32 swap their execute function under EIP-4762 (verkle) to charge
	// code-chunk gas on the immediate bytes. Defer to the table handler there.
	// The baked static gas and stack guard above already match.
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

	switch h := inlineOps[code]; h {
	case "makePush": // PUSH3-PUSH32: splice makePush(size, size)
		n := int(code) - 0x5f
		g.p("%s", g.inlineOpcodeFactoryBody("makePush", n, n))
	case "makeDup": // DUP1-DUP16: splice makeDup(n)
		g.p("%s", g.inlineOpcodeFactoryBody("makeDup", int(code)-0x7f))
	default: // the rest: splice the opXxx handler body
		g.p("%s", g.inlineOpcodeBody(h))
	}
}

func (g *generator) emitInlineOp(code byte) {
	spec := g.specs[code]
	g.p("case %s:\n", spec.name)
	if spec.fork == "" {
		g.emitOpBody(code)
		return
	}
	// Fork-gated: run the inlined body only when the opcode is active for the
	// current fork. Otherwise mirror the legacy loop's undefined-opcode handling.
	g.p("if rules.%s {\n", spec.fork)
	g.emitOpBody(code)
	g.p("}\n")
	g.p(`
		res, err = opUndefined(&pc, evm, scope)
		break mainLoop
	`)
}

// emitDynamicGas emits the dynamic-gas computation and charge shared by the
// direct-call and default cases. gasFn is the dynamic-gas function to invoke:
// a name like "gasKeccak256" for a direct call, or "operation.dynamicGas" for
// the table case. A computation error is wrapped as ErrOutOfGas, then the cost
// is charged through GasBudget.chargeDynamic. The doubled %%w/%%v are not
// generator verbs: Fprintf collapses each %% to one %, leaving a literal
// fmt.Errorf("%w: %v", ...) in the generated code.
func (g *generator) emitDynamicGas(gasFn string) {
	g.p(`
		var dynamicCost GasCosts
		dynamicCost, err = %s(evm, contract, stack, mem, memorySize)
		if err != nil {
			return nil, fmt.Errorf("%%w: %%v", ErrOutOfGas, err)
		}
		if err := contract.Gas.chargeDynamic(dynamicCost); err != nil {
			return nil, err
		}
	`, gasFn)
}

// emitDirectCallOp emits an opcode case identical to the default case, except
// the handler, dynamic-gas, and memory-size functions are called by name
// rather than through the indirect operation.* table pointers. Valid only for
// fork-invariant ops (see directCallOps).
func (g *generator) emitDirectCallOp(code byte) {
	spec := g.specs[code]
	fns := directCallOps[code]
	g.p("case %s:\n", spec.name)
	g.emitStackChecks(fmt.Sprint(spec.minStack), fmt.Sprint(spec.maxStack), spec.minStack > 0, spec.maxStack < stackLimit)
	if spec.constGas != 0 {
		g.emitStaticGas(fmt.Sprint(spec.constGas))
	}
	// fns[2], fns[1], fns[0] are the memory-size, dynamic-gas and handler names.
	g.p(`
		var memorySize uint64
		{
			memSize, overflow := %s(stack)
			if overflow {
				return nil, ErrGasUintOverflow
			}
			if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
				return nil, ErrGasUintOverflow
			}
		}
	`, fns[2])
	g.emitDynamicGas(fns[1])
	g.p(`
		if memorySize > 0 {
			mem.Resize(memorySize)
		}
		res, err = %s(&pc, evm, scope)
		if err != nil {
			break mainLoop
		}
		pc++
		continue mainLoop
	`, fns[0])
}

func (g *generator) emitDefault() {
	g.p(`
		default:
			operation := table[op]
	`)
	g.emitStackChecks("operation.minStack", "operation.maxStack", true, true)
	g.emitStaticGas("operation.constantGas")
	g.p(`
			var memorySize uint64
			if operation.dynamicGas != nil {
				if operation.memorySize != nil {
					memSize, overflow := operation.memorySize(stack)
					if overflow {
						return nil, ErrGasUintOverflow
					}
					if memorySize, overflow = math.SafeMul(toWordSize(memSize), 32); overflow {
						return nil, ErrGasUintOverflow
					}
				}
	`)
	g.emitDynamicGas("operation.dynamicGas")
	g.p(`
			}
			if memorySize > 0 {
				mem.Resize(memorySize)
			}
			res, err = operation.execute(&pc, evm, scope)
			if err != nil {
				break mainLoop
			}
			pc++
			continue mainLoop
	`)
}

// ---------------------------------------------------------------------------
// File emission
// ---------------------------------------------------------------------------

func (g *generator) emitFile() {
	g.p(`
		// Code generated by core/vm/gen; DO NOT EDIT.

		package vm

		import (
			"fmt"

			"github.com/ethereum/go-ethereum/common/math"
			"github.com/ethereum/go-ethereum/core/tracing"
		)

	`)

	g.p(`
		// execUntraced is the generated, tracing-free interpreter fast path. Hot,
		// fork-stable opcodes are inlined with their static gas and stack bounds baked
		// in. Fork-invariant ops (KECCAK256/MLOAD/MSTORE/MSTORE8) call their
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
				if isEIP4762 && !contract.IsDeployment && !contract.IsSystemCall {
					contractAddr := contract.Address()
					consumed, wanted := evm.TxContext.AccessEvents.CodeChunksRangeGas(contractAddr, pc, 1, uint64(len(contract.Code)), false, contract.Gas.RegularGas)
					contract.chargeRegular(consumed, evm.Config.Tracer, tracing.GasChangeWitnessCodeChunk)
					if consumed < wanted {
						return nil, ErrOutOfGas
					}
				}
				op := contract.GetOp(pc)
				switch op {
	`)
	// Inlined cases, in opcode order for readability.
	for code := 0; code < 256; code++ {
		b := byte(code)
		if _, named := inlineOps[b]; named {
			g.emitInlineOp(b)
		} else if _, dc := directCallOps[b]; dc {
			g.emitDirectCallOp(b)
		}
	}
	g.emitDefault()
	g.p("}\n") // switch
	g.p("}\n") // for
	g.p("if err == errStopToken {\nerr = nil\n}\n")
	g.p("return res, err\n")
	g.p("}\n") // func
}

// ---------------------------------------------------------------------------

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		fatalf("cannot resolve generator source path")
	}
	vmDir := filepath.Dir(filepath.Dir(self)) // .../core/vm/gen -> .../core/vm

	fset, handlers, stackHelpers, gasHelpers := parseHandlers(vmDir)
	g := &generator{fset: fset, handlers: handlers, stackHelpers: stackHelpers, gasHelpers: gasHelpers, buf: new(bytes.Buffer)}
	g.deriveSpecs(vm.GenForks())
	g.emitFile()

	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		dbg := filepath.Join(vmDir, "interp_gen.go.broken")
		os.WriteFile(dbg, g.buf.Bytes(), 0644)
		fatalf("gofmt failed (%v); wrote unformatted output to %s", err, dbg)
	}
	// INTERP_GEN_OUT lets the CI-match test (interp_gen_test.go) regenerate to a
	// temporary file and diff it against the committed one, without clobbering it.
	out := filepath.Join(vmDir, "interp_gen.go")
	if env := os.Getenv("INTERP_GEN_OUT"); env != "" {
		out = env
	}
	if err := os.WriteFile(out, formatted, 0644); err != nil {
		fatalf("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", out, len(formatted))
}
