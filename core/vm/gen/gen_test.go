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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
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
	g := &generator{}
	g.deriveSpecs(genForks())

	for _, tc := range []struct {
		name string
		want string
		fn   func()
	}{
		{
			// 0x0c is not an opcode in any fork, so there is no spec to emit from.
			name: "own case for an opcode no fork defines",
			want: "never defined",
			fn:   func() { g.checkStaticStable(0x0c, genForks()) },
		},
		{
			name: "dynamic-gas case for an opcode no fork defines",
			want: "never defined",
			fn:   func() { g.checkDynamicStable(0x0c, genForks()) },
		},
		{
			// LOG0 comes from makeLog, so FuncForPC gives it a dotted closure name
			// with nothing callable in it.
			name: "handler that is a closure with no name to call",
			want: "no name to call",
			fn:   func() { g.handlerCall(byte(vm.LOG0)) },
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
