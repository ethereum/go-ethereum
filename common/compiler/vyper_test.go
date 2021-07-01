// Copyright 2019 The go-ethereum Authors
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

package compiler

import (
	"os/exec"
	"testing"
)

func skipWithoutVyper(t *testing.T) {
	if _, err := exec.LookPath("vyper"); err != nil {
		t.Skip(err)
	}
}

func TestVyperCompiler(t *testing.T) {
	skipWithoutVyper(t)

	testSource := []string{"test.v.py"}
	source, err := slurpFiles(testSource)
	if err != nil {
		t.Error("couldn't read test files")
	}
	contracts, err := CompileVyper("", testSource...)
	if err != nil {
		t.Fatalf("error compiling test.v.py. result %v: %v", contracts, err)
	}
	if len(contracts) != 1 {
		t.Errorf("one contract expected, got %d", len(contracts))
	}
	c, ok := contracts["test.v.py"]
	if !ok {
		c, ok = contracts["<stdin>:test"]
		if !ok {
			t.Fatal("info for contract 'test.v.py' not present in result")
		}
	}
	if c.Code == "" {
		t.Error("empty code")
	}
	if c.Info.Source != source {
		t.Error("wrong source")
	}
	if c.Info.CompilerVersion == "" {
		t.Error("empty version")
	}
}

func TestVyperCompileError(t *testing.T) {
	skipWithoutVyper(t)

	contracts, err := CompileVyper("", "test_bad.v.py")
	if err == nil {
		t.Errorf("error expected compiling test_bad.v.py. got none. result %v", contracts)
	}
	t.Logf("error: %v", err)
}
