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
	"os"
	"path/filepath"
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
		t.Fatal("interpreter_gen.go is out of date; run `go generate ./core/vm/...` and commit the result")
	}
}
