// Copyright 2017 The go-ethereum Authors
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

package asm

import (
	"testing"

	"encoding/hex"
)

// Tests disassembling instructions
func TestInstructionIterator(t *testing.T) {
	for i, tc := range []struct {
		want    int
		code    string
		wantErr string
	}{
		{2, "61000000", ""},                             // valid code
		{0, "6100", "incomplete push instruction at 0"}, // invalid code
		{2, "5900", ""},                                 // push0
		{0, "", ""},                                     // empty

	} {
		var (
			have    int
			code, _ = hex.DecodeString(tc.code)
			it      = NewInstructionIterator(code)
		)
		for it.Next() {
			have++
		}
		var haveErr = ""
		if it.Error() != nil {
			haveErr = it.Error().Error()
		}
		if haveErr != tc.wantErr {
			t.Errorf("test %d: encountered error: %q want %q", i, haveErr, tc.wantErr)
			continue
		}
		if have != tc.want {
			t.Errorf("wrong instruction count, have %d want %d", have, tc.want)
		}
	}
}
