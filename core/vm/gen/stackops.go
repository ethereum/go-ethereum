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
	"go/ast"
	"go/token"
	"strconv"
)

// This file holds the *Stack method rewriter. The dispatch keeps the stack in two
// loop locals, so a stack call becomes the lines the method would have run rather
// than a real call.

// stackCall is a matched call to a *Stack method.
type stackCall struct {
	method string      // method name
	lhs    []ast.Expr  // assignment targets, nil for a void call like dup
	tok    token.Token // the assignment token, := or =
	args   []ast.Expr  // call arguments (only dup has one)
}

// matchStackCall matches a statement that is a single stack method call,
// in one of the two normalized forms: an assignment `lhs... := scope.Stack.M(args)`
// or a bare `scope.Stack.M(args)`.
func (g *generator) matchStackCall(stmt ast.Stmt) (stackCall, bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		if len(s.Rhs) == 1 {
			if m, args, ok := g.stackMethodCall(s.Rhs[0]); ok {
				return stackCall{method: m, lhs: s.Lhs, tok: s.Tok, args: args}, true
			}
		}
	case *ast.ExprStmt:
		if m, args, ok := g.stackMethodCall(s.X); ok {
			return stackCall{method: m, args: args}, true
		}
	}
	return stackCall{}, false
}

// stackMethodCall unwraps scope.Stack.M(args) where M is a method of *Stack.
func (g *generator) stackMethodCall(e ast.Expr) (method string, args []ast.Expr, ok bool) {
	call, isCall := e.(*ast.CallExpr)
	if !isCall {
		return "", nil, false
	}
	sel, isSel := call.Fun.(*ast.SelectorExpr) // <recv>.M
	if !isSel || !isStackExpr(sel.X) || g.stackMethods[sel.Sel.Name] == nil {
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

// The dispatch keeps the whole stack in two loop locals:
//
//	sp = stack.size                        the number of items
//	sd = stack.inner.data[stack.bottom:]   the arena window this stack owns
//
// so a *Stack method can be rewritten into that frame rather than called. Stack
// keeps inner.top == bottom+size, so both counters are sp, and an index off
// either one loses its bottom term:
//
//	s.inner.top -= 2                 ->  sp -= 2
//	&s.inner.data[s.inner.top+1]     ->  &sd[sp+1]
//	s.inner.data[s.bottom+s.size-2]  ->  sd[sp-2]
//
// stackFrame is the context one such rewrite runs in.
type stackFrame struct {
	method string              // method being rewritten, for error messages
	recv   string              // its receiver name, "s"
	params map[string]ast.Expr // its parameters, holding the call's arguments
}

// expandStackMethod rewrites the called *Stack method into the lines the dispatch runs
// in its place, so the expansion follows stack.go instead of restating it. For
// `pos, cond := scope.Stack.pop2()` in opJumpi, pop2's body
//
//	s.inner.top -= 2
//	s.size -= 2
//	return &s.inner.data[s.inner.top+1], &s.inner.data[s.inner.top]
//
// becomes
//
//	sp -= 2
//	pos := &sd[sp+1]
//	cond := &sd[sp]
//
// params are the enclosing factory's constants, so makeDup's dup(size) expands with
// size already resolved to this opcode's depth.
func (g *generator) expandStackMethod(call stackCall, params map[string]int) string {
	fn := g.stackMethod(call.method)
	f := &stackFrame{
		method: call.method,
		recv:   recvName(fn),
		params: bindStackParams(fn, call.args, params),
	}

	var (
		out   []ast.Stmt
		decls = map[string]*ast.AssignStmt{} // locals the method declares
		moves = map[string]bool{}            // counter moves already emitted
	)
	for i, stmt := range fn.Body.List {
		switch s := stmt.(type) {
		case *ast.IncDecStmt: // s.size++ -> sp++
			move := &ast.IncDecStmt{X: f.expr(s.X), Tok: s.Tok}
			if g.duplicateMove(move, moves) {
				continue
			}
			out = append(out, move)

		// One of three shapes: a counter move, a data move like swap's
		// exchange, or a local the method goes on to return.
		case *ast.AssignStmt:
			rewritten := &ast.AssignStmt{Tok: s.Tok}
			for _, lhs := range s.Lhs {
				rewritten.Lhs = append(rewritten.Lhs, f.expr(lhs))
			}
			for _, rhs := range s.Rhs {
				rewritten.Rhs = append(rewritten.Rhs, f.expr(rhs))
			}
			switch s.Tok {
			case token.ADD_ASSIGN, token.SUB_ASSIGN: // s.inner.top -= 2 -> sp -= 2
				if g.duplicateMove(rewritten, moves) {
					continue
				}
			case token.DEFINE: // elem := &s.inner.data[s.inner.top] -> elem := &sd[sp]
				if id, ok := rewritten.Lhs[0].(*ast.Ident); ok {
					decls[id.Name] = rewritten
				}
			}
			out = append(out, rewritten)

		case *ast.ReturnStmt:
			if i != len(fn.Body.List)-1 {
				fatalf("(*Stack).%s returns before its last statement", call.method)
			}
			if len(s.Results) != len(call.lhs) {
				fatalf("(*Stack).%s returns %d values, call assigns %d", call.method, len(s.Results), len(call.lhs))
			}
			for j, res := range s.Results {
				// A local the method declared takes the caller's name instead of
				// being copied into it. See renameLocal.
				if id, ok := res.(*ast.Ident); ok && decls[id.Name] != nil {
					decls[id.Name].Tok = call.tok
					renameLocal(out, id.Name, call.lhs[j])
					continue
				}
				out = append(out, &ast.AssignStmt{
					Lhs: []ast.Expr{call.lhs[j]},
					Tok: call.tok,
					Rhs: []ast.Expr{f.expr(res)},
				})
			}

		default:
			fatalf("(*Stack).%s has a %T statement, which the dispatch cannot rewrite", call.method, stmt)
		}
	}
	return g.renderAst(out)
}

// duplicateMove reports whether this counter move was already emitted for the current
// expansion. inner.top and size are the same quantity in the loop's frame, so pop2's
//
//	s.inner.top -= 2
//	s.size -= 2
//
// both rewrite to `sp -= 2` and only the first is kept.
func (g *generator) duplicateMove(move ast.Stmt, seen map[string]bool) bool {
	line := g.renderAst([]ast.Stmt{move})
	if seen[line] {
		return true
	}
	seen[line] = true
	return false
}

// expr rewrites one expression of a stack method into the loop's frame.
func (f *stackFrame) expr(e ast.Expr) ast.Expr {
	switch x := e.(type) {
	case *ast.Ident:
		if arg, ok := f.params[x.Name]; ok {
			return arg // a parameter, like dup's n
		}
		return ast.NewIdent(x.Name)
	case *ast.BasicLit:
		return &ast.BasicLit{Kind: x.Kind, Value: x.Value}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: x.Op, X: f.expr(x.X)}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: f.expr(x.X), Index: f.expr(x.Index)}
	case *ast.BinaryExpr:
		// sd already starts at bottom, so an index off bottom drops the term.
		if x.Op == token.ADD {
			if f.field(x.X) == "bottom" {
				return f.expr(x.Y)
			}
			if f.field(x.Y) == "bottom" {
				return f.expr(x.X)
			}
		}
		return &ast.BinaryExpr{X: f.expr(x.X), Op: x.Op, Y: f.expr(x.Y)}
	case *ast.SelectorExpr:
		switch field := f.field(x); field {
		case "size", "inner.top":
			return ast.NewIdent("sp")
		case "inner.data":
			return ast.NewIdent("sd")
		default:
			// A read the loop locals do not cover, such as release's bare
			// s.bottom. Rewriting it would silently mean something else.
			fatalf("(*Stack).%s reads %s.%s, which has no sp/sd form", f.method, f.recv, field)
		}
	}
	fatalf("(*Stack).%s uses a %T, which has no sp/sd form", f.method, e)
	return nil
}

// field returns the dotted field path an expression reads off the receiver
// ("inner.top" for s.inner.top), or "" if it does not read one.
func (f *stackFrame) field(e ast.Expr) string {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	switch x := sel.X.(type) {
	case *ast.Ident:
		if x.Name == f.recv {
			return sel.Sel.Name
		}
	case *ast.SelectorExpr:
		if inner := f.field(x); inner != "" {
			return inner + "." + sel.Sel.Name
		}
	}
	return ""
}

// bindStackParams matches a stack method's parameters to the call's arguments, so
// makeDup's dup(size) expands with n already resolved to 2 for DUP2.
func bindStackParams(fn *ast.FuncDecl, args []ast.Expr, params map[string]int) map[string]ast.Expr {
	names := paramNames(fn)
	if len(names) != len(args) {
		fatalf("(*Stack).%s takes %d params, call passes %d", fn.Name.Name, len(names), len(args))
	}
	bound := map[string]ast.Expr{}
	for i, name := range names {
		arg := args[i]
		switch a := arg.(type) {
		case *ast.Ident:
			// A name the enclosing factory binds becomes that constant, so
			// makeDup's size arrives here as 2. Any other name is left alone.
			if v, ok := params[a.Name]; ok {
				arg = &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(v)}
			}
		case *ast.BasicLit:
			// Already a constant, embed as written.
		default:
			// Anything else, say dup(size+1), would embed a name the generated
			// function does not have. Stop here rather than emit broken Go.
			fatalf("(*Stack).%s is passed a %T for %s; the dispatch can bind only a literal or a factory constant",
				fn.Name.Name, arg, name)
		}
		bound[name] = arg
	}
	return bound
}

// renameLocal points a local the method declared at the caller's target. get's body
//
//	elem := &sd[sp]
//	sp++
//	return elem
//
// called as `a := scope.Stack.get()` becomes
//
//	a := &sd[sp]
//	sp++
//
// It walks only the statements built for this expansion, never the parsed stack.go
// tree, whose nodes are shared by every opcode that calls the same method.
func renameLocal(stmts []ast.Stmt, from string, to ast.Expr) {
	id, ok := to.(*ast.Ident)
	if !ok {
		fatalf("stack call assigns to a %T, want a plain name", to)
	}
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(n ast.Node) bool {
			if x, ok := n.(*ast.Ident); ok && x.Name == from {
				x.Name = id.Name
			}
			return true
		})
	}
}
