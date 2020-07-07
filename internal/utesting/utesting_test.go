// Copyright 2020 The go-ethereum Authors
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

package utesting

import (
	"strings"
	"testing"
)

func TestTest(t *testing.T) {
	tests := []Test{
		{
			Name: "successful test",
			Fn:   func(t *T) {},
		},
		{
			Name: "failing test",
			Fn: func(t *T) {
				t.Log("output")
				t.Error("failed")
			},
		},
		{
			Name: "panicking test",
			Fn: func(t *T) {
				panic("oh no")
			},
		},
	}
	results := RunTests(tests, nil)

	if results[0].Failed || results[0].Output != "" {
		t.Fatalf("wrong result for successful test: %#v", results[0])
	}
	if !results[1].Failed || results[1].Output != "output\nfailed\n" {
		t.Fatalf("wrong result for failing test: %#v", results[1])
	}
	if !results[2].Failed || !strings.HasPrefix(results[2].Output, "panic: oh no\n") {
		t.Fatalf("wrong result for panicking test: %#v", results[2])
	}
}
