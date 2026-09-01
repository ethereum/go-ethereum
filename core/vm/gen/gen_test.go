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
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestGeneratedDispatchUpToDate asserts that the committed interpreter_gen.go
// matches what the generator produces from the current opcode, gas and fork
// definitions. It is the CI guard against hand-edits to the generated file and
// against the generator drifting from the committed output.
func TestGeneratedDispatchUpToDate(t *testing.T) {
	got, err := generate()
	if err != nil {
		t.Fatalf("running generator: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(vmDir(), "interpreter_gen.go"))
	if err != nil {
		t.Fatalf("reading committed interpreter_gen.go: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("interpreter_gen.go is out of date; run `go generate ./core/vm/...` and commit the result.\n"+
			"First difference (- committed, + generated):\n%s", firstDiff(want, got))
	}
}

// firstDiff shows where two versions of the generated file start to differ, with a
// few lines of context, so a generator change reads as a hunk rather than as "out
// of date".
func firstDiff(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	for i := range max(len(wantLines), len(gotLines)) {
		w, g := lineAt(wantLines, i), lineAt(gotLines, i)
		if w == g {
			continue
		}
		var b strings.Builder
		for j := max(0, i-3); j < i; j++ {
			fmt.Fprintf(&b, "  %d\t%s\n", j+1, wantLines[j])
		}
		fmt.Fprintf(&b, "- %d\t%s\n", i+1, w)
		fmt.Fprintf(&b, "+ %d\t%s\n", i+1, g)
		return b.String()
	}
	return "(no differing line, so the files differ only in trailing content)"
}

// lineAt returns a line, or a marker once that side has run out of them.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "<end of file>"
}

// stackExpansions pins what each *Stack method the dispatch uses becomes in sp/sd
// form, with x, y and z standing in for the call's assignment targets. The
// generator derives these from stack.go rather than storing them (see
// expandStackMethod), so this table is the independent statement of intent: change
// a stack helper's body and the failure names the method, instead of surfacing
// later as an interpreter divergence.
func stackExpansions() map[string]string {
	ops := map[string]string{
		"get":       "x := &sd[sp]\nsp++\n",
		"drop":      "sp--\n",
		"peek":      "x := &sd[sp-1]\n",
		"pop1":      "sp--\nx := &sd[sp]\n",
		"pop2":      "sp -= 2\nx := &sd[sp+1]\ny := &sd[sp]\n",
		"pop1Peek1": "sp--\nx := &sd[sp]\ny := &sd[sp-1]\n",
		"pop2Peek1": "sp -= 2\nx := &sd[sp+1]\ny := &sd[sp]\nz := &sd[sp-1]\n",
		"dup":       "sd[sp] = sd[sp-n]\nsp++\n",
	}
	// stack.go spells the swaps out one method per depth, so build the expectation
	// the same way instead of listing sixteen near-identical lines.
	for n := 1; n <= 16; n++ {
		depth := strconv.Itoa(n + 1)
		ops["swap"+strconv.Itoa(n)] = "sd[sp-" + depth + "], sd[sp-1] = sd[sp-1], sd[sp-" + depth + "]\n"
	}
	return ops
}

func TestStackExpansions(t *testing.T) {
	g := &generator{source: parseSource(vmDir())}
	for method, want := range stackExpansions() {
		if got := expandForTest(g, method); got != want {
			t.Errorf("(*Stack).%s expands to:\n%s\nwant:\n%s", method, got, want)
		}
	}
}

// expandForTest expands one stack method with placeholder targets, standing in for
// the handler call site the generator would have matched.
func expandForTest(g *generator, method string) string {
	fn := g.stackMethod(method)
	call := stackCall{method: method, tok: token.DEFINE}
	for i := range fn.Type.Results.NumFields() {
		call.lhs = append(call.lhs, ast.NewIdent([]string{"x", "y", "z"}[i]))
	}
	// Pass each parameter through by name, so dup's depth stays symbolic.
	for _, name := range paramNames(fn) {
		call.args = append(call.args, ast.NewIdent(name))
	}
	return g.expandStackMethod(call)
}

// tripped runs fn and returns the message of the guard it trips, or "" if fn
// completed. It is the test-side counterpart of generate's recover: a genError means
// a guard fired, anything else is a real bug and keeps its stack.
func tripped(t *testing.T, fn func()) string {
	t.Helper()
	var msg string
	func() {
		defer func() {
			switch r := recover().(type) {
			case nil:
			case genError:
				msg = r.Error()
			default:
				panic(r)
			}
		}()
		fn()
	}()
	return msg
}

// TestGuards covers the checks that stop the generator when core/vm changes in a way
// it cannot express. They are the reason the generated dispatch is safe to trust, so
// each one needs to fire rather than fall through and emit wrong code.
func TestGuards(t *testing.T) {
	g := &generator{source: parseSource(vmDir())}
	g.deriveSpecs(genForks())

	for _, tc := range []struct {
		name string
		want string
		fn   func()
	}{
		{
			// A stack call whose argument is not a constant the dispatch can embed.
			name: "stack argument that is not a literal or a name",
			want: "can bind only a literal or a name",
			fn: func() {
				plusOne := &ast.BinaryExpr{X: ast.NewIdent("n"), Op: token.ADD, Y: &ast.BasicLit{Kind: token.INT, Value: "1"}}
				bindStackParams(g.stackMethod("dup"), []ast.Expr{plusOne})
			},
		},
		{
			// push calls another method, which has no sp/sd form.
			name: "stack method whose body calls out",
			want: "(*Stack).push uses a *ast.CallExpr",
			fn: func() {
				g.expandStackMethod(stackCall{method: "push", args: []ast.Expr{ast.NewIdent("d")}})
			},
		},
		{
			// release reads a bare s.bottom, which would silently become sp = 0.
			name: "stack method reading a field with no sp/sd form",
			want: "(*Stack).release reads s.bottom",
			fn: func() {
				g.expandStackMethod(stackCall{method: "release"})
			},
		},
		{
			// pairReturnRe splits the value at the first comma, so one nested in
			// brackets would be cut in half rather than emitted whole.
			name: "return whose value holds a comma",
			want: "its value holds a comma",
			fn: func() {
				g.rewriteReturns("return f(a, b), nil\n", returnRewrite{results: 2})
			},
		},
		{
			name: "handler that is not in the parsed sources",
			want: `no handler "opNotAThing"`,
			fn:   func() { g.opHandler("opNotAThing") },
		},
		{
			name: "gas helper that is not in the parsed sources",
			want: `no gas helper "chargeNothing"`,
			fn:   func() { g.gasHelper("chargeNothing") },
		},
		{
			// 0x0c is not an opcode in any fork, so there is no spec to emit from.
			name: "inlining an opcode no fork defines",
			want: "never defined",
			fn:   func() { g.checkInlineStable(0x0c, genForks()) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tripped(t, tc.fn)
			if got == "" {
				t.Fatalf("no guard fired, wanted one mentioning %q", tc.want)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("guard said %q, want it to mention %q", got, tc.want)
			}
		})
	}
}
