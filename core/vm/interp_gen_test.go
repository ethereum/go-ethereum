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

package vm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGeneratedDispatchUpToDate asserts that the committed interp_gen.go matches
// what `go generate` (core/vm/gen) produces from the current opcode/gas/fork
// definitions. It is the CI guard against hand-edits to the generated file and
// against the generator drifting from the committed output.
func TestGeneratedDispatchUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generator round-trip in -short mode")
	}
	tmp := filepath.Join(t.TempDir(), "interp_gen.go")
	cmd := exec.Command("go", "run", "./gen")
	cmd.Env = append(os.Environ(), "INTERP_GEN_OUT="+tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running generator: %v\n%s", err, out)
	}
	got, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("reading regenerated output: %v", err)
	}
	want, err := os.ReadFile("interp_gen.go")
	if err != nil {
		t.Fatalf("reading committed interp_gen.go: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("interp_gen.go is out of date; run `go generate ./core/vm/...` and commit the result")
	}
}
