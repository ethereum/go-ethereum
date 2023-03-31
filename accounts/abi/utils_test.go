// Copyright 2023 The go-ethereum Authors
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

package abi

import "testing"

func TestResolveNameConflict(t *testing.T) {
	db := make(map[string]struct{})
	used := func(s string) bool {
		_, ok := db[s]
		return ok
	}

	var tests = []struct {
		input  string
		output string
	}{
		{
			input:  "1method",
			output: "M1method",
		},
		{
			input:  "1method",
			output: "M1method0",
		},
		{
			input:  "1method",
			output: "M1method1",
		},
		{
			input:  "method",
			output: "method",
		},
		{
			input:  "method",
			output: "method0",
		},
		{
			input:  "",
			output: "",
		},
	}

	for _, test := range tests {
		result := ResolveNameConflict(test.input, used)
		if result != test.output {
			t.Errorf("resolving name conflict failed, got %v want %v input %v", result, test.output, test.input)
		}
		db[result] = struct{}{}
	}
}
