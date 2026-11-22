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

package jsre

import (
	"bytes"
	"testing"

	"github.com/dop251/goja"
)

// TestPrettyPrintUnusualConstructor checks that objects without a usable
// constructor property can be printed.
func TestPrettyPrintUnusualConstructor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code string
		want string
	}{
		{"({a: 1})", "{\n  a: 1\n}"},
		{"Object.create(null)", "{}"},
		{"var o = Object.create(null); o.a = 1; o", "{\n  a: 1\n}"},
		{"({a: 1, constructor: undefined})", "{\n  a: 1\n}"},
		{"({a: 1, constructor: null})", "{\n  a: 1\n}"},
		{"({a: 1, constructor: {prototype: null}})", "{\n  a: 1\n}"},
	}
	for _, test := range tests {
		vm := goja.New()
		// The console always loads bignumber.js, which defines BigNumber.
		if _, err := vm.RunString("function BigNumber() {}"); err != nil {
			t.Fatal(err)
		}
		val, err := vm.RunString(test.code)
		if err != nil {
			t.Fatalf("%s: %v", test.code, err)
		}
		var buf bytes.Buffer
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: prettyPrint panicked: %v", test.code, r)
				}
			}()
			prettyPrint(vm, val, &buf)
		}()
		if got := buf.String(); got != test.want {
			t.Errorf("%s: got %q, want %q", test.code, got, test.want)
		}
	}
}
