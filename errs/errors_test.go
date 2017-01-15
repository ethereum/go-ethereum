// Copyright 2015 The go-ethereum Authors
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

package errs

import (
	"fmt"
	"testing"
)

func testErrors() *Errors {
	return &Errors{
		Package: "TEST",
		Errors: map[int]string{
			0: "zero",
			1: "one",
		},
	}
}

func TestErrorMessage(t *testing.T) {
	err := testErrors().New(0, "zero detail %v", "available")
	message := fmt.Sprintf("%v", err)
	exp := "[TEST] ERROR: zero: zero detail available"
	if message != exp {
		t.Errorf("error message incorrect. expected %v, got %v", exp, message)
	}
}
