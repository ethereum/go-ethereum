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
	"github.com/google/pprof/profile"
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

// TestGeneratedProfileUpToDate asserts that the committed PGO profile matches what
// the generator builds for the committed dispatch. It is the guard the profile most
// needs: a profile records the line each handler call sits on, so a dispatch change
// that does not regenerate it leaves a profile matching nothing. The build succeeds,
// every test passes, and the handlers quietly stop being inlined.
func TestGeneratedProfileUpToDate(t *testing.T) {
	src, err := generate()
	if err != nil {
		t.Fatalf("running generator: %v", err)
	}
	prof, err := profileFor(src)
	if err != nil {
		t.Fatalf("building the profile: %v", err)
	}
	var got bytes.Buffer
	if err := prof.Write(&got); err != nil {
		t.Fatalf("serializing the profile: %v", err)
	}
	path := filepath.Join(vmDir(), "..", "..", pgoFile)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed %s: %v", pgoFile, err)
	}
	if bytes.Equal(got.Bytes(), want) {
		return
	}
	t.Fatalf("%s is out of date; run `go generate ./core/vm/...` and commit the result.\n"+
		"  committed: %s\n  generated: %s", pgoFile, describeProfile(want), describeProfile(got.Bytes()))
}

// describeProfile summarizes a serialized profile for a failure message, since the
// bytes themselves say nothing useful.
func describeProfile(b []byte) string {
	p, err := profile.Parse(bytes.NewReader(b))
	if err != nil {
		return fmt.Sprintf("%d bytes, not a readable profile (%v)", len(b), err)
	}
	var caller *profile.Function
	callees := map[string]bool{}
	for _, fn := range p.Function {
		if strings.HasSuffix(fn.Name, dispatchName) {
			caller = fn
			continue
		}
		callees[fn.Name] = true
	}
	if caller == nil {
		return fmt.Sprintf("%d samples, %d handlers, no %s frame", len(p.Sample), len(callees), dispatchFunc)
	}
	// PGO keys a call site by its offset from the caller's first line, so that is
	// what a stale profile gets wrong. Report the span, since it moves when the
	// dispatch does and the start line alone does not.
	lo, hi := int64(-1), int64(-1)
	for _, loc := range p.Location {
		for _, ln := range loc.Line {
			if ln.Function != caller {
				continue
			}
			off := ln.Line - caller.StartLine
			if lo < 0 || off < lo {
				lo = off
			}
			if off > hi && off < 1e6 {
				hi = off
			}
		}
	}
	return fmt.Sprintf("%d samples, %d handlers, call sites at offsets %d..%d from line %d",
		len(p.Sample), len(callees), lo, hi, caller.StartLine)
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

// TestTierFor covers the classifier that decides whether a hotOps entry can take
// its own case, and which tier it lands in. Getting it wrong either emits a
// fork-varying value as a constant or writes a call to a name that does not exist.
func TestTierFor(t *testing.T) {
	forks := genForks()
	g := &generator{}
	g.deriveSpecs(forks)

	for _, tc := range []struct {
		op   vm.OpCode
		want tier
		why  string
	}{
		{vm.ADD, tierStatic, "fork-stable, constant gas only"},
		{vm.DUP1, tierStatic, "same"},
		{vm.KECCAK256, tierDynamic, "fork-stable but carries dynamic gas"},
		{vm.MLOAD, tierDynamic, "same"},
		{vm.SSTORE, tierTable, "gas varies by fork"},
		{vm.EXP, tierTable, "same"},
		{vm.BALANCE, tierTable, "same"},
		{vm.LOG0, tierTable, "handler is built by makeLog, so it has no name to call"},
		{vm.OpCode(0x0c), tierTable, "not an opcode in any fork"},
	} {
		if got, _ := g.tierFor(byte(tc.op), forks); got != tc.want {
			t.Errorf("tierFor(%s) = %v, want %v (%s)", tc.op, got, tc.want, tc.why)
		}
	}
}

// TestHotOpsAllGetCases checks that every opcode hotOps names actually ends up with
// its own case. An entry that quietly failed to qualify would cost performance
// without failing anything.
func TestHotOpsAllGetCases(t *testing.T) {
	g := &generator{}
	g.deriveSpecs(genForks())
	for _, op := range hotOps {
		if g.tierOf(byte(op)) == tierTable {
			t.Errorf("%s is in hotOps but was left on the table tier", op)
		}
	}
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
			// SSTORE's gas varies by fork, so listing it as hot would ask for a case
			// that cannot be emitted.
			name: "hotOps entry whose gas varies by fork",
			want: "cannot take its own case",
			fn:   func() { g.assignTiers([]vm.OpCode{vm.SSTORE}, genForks()) },
		},
		{
			// 0x0c is not an opcode in any fork, so there is no spec to emit from.
			name: "hotOps entry no fork defines",
			want: "no fork defines it",
			fn:   func() { g.assignTiers([]vm.OpCode{vm.OpCode(0x0c)}, genForks()) },
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
