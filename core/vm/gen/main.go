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

// Command gen generates core/vm/interpreter_gen.go, the EVM interpreter's untraced
// fast-path dispatch, a switch over the opcode byte. The generated file is
// committed and a CI test asserts it matches `go generate` output. Do not
// hand-edit interpreter_gen.go.
//
// Usage: go generate ./core/vm/...
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	formatted, err := generate()
	if err != nil {
		fatalf("%v", err)
	}
	out := filepath.Join(vmDir(), "interpreter_gen.go")
	if err := os.WriteFile(out, formatted, 0644); err != nil {
		fatalf("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", out, len(formatted))
}
