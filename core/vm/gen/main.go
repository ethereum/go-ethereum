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
// It also writes the PGO profile that gets the dispatch's handler calls inlined,
// which has to be rebuilt with the dispatch because it records where those calls
// landed. See pgo.go.
//
// The profile has to sit in a main package's directory, because -pgo=auto looks
// nowhere else, so the default reaches out of core/vm to cmd/geth. -pgo names
// another destination, relative to the repo root unless it is absolute, and an
// empty value skips it.
//
// Usage: go generate ./core/vm/...
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	pgoOut := flag.String("pgo", pgoFile, "where to write the PGO profile, relative to the repo root unless absolute. Empty skips it")
	flag.Parse()

	formatted, err := generate()
	if err != nil {
		fatalf("%v", err)
	}
	out := filepath.Join(vmDir(), generatedFile)
	if err := os.WriteFile(out, formatted, 0644); err != nil {
		fatalf("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", out, len(formatted))

	// The profile records where the handler calls landed in the file just
	// written, so it has to be rebuilt whenever the dispatch is. Writing it
	// here rather than by hand is what keeps the two from drifting apart.
	if *pgoOut == "" {
		return
	}
	prof, err := profileFor(formatted)
	if err != nil {
		fatalf("%v", err)
	}
	dest := *pgoOut
	if !filepath.IsAbs(dest) {
		// The repository root, two levels up from core/vm.
		dest = filepath.Join(vmDir(), "..", "..", dest)
	}
	dest = filepath.Clean(dest)
	f, err := os.Create(dest)
	if err != nil {
		fatalf("create %s: %v", dest, err)
	}
	if err := prof.Write(f); err != nil {
		f.Close()
		fatalf("write %s: %v", dest, err)
	}
	f.Close()
	fmt.Fprintf(os.Stderr, "gen: wrote %s\n", dest)
}
