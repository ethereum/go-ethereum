// Copyright 2016 The go-ethereum Authors
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

// +build go1.4,nacl,faketime_simulation

package discv5

import (
	"os"
	"runtime"
	"testing"
	"unsafe"
)

// Enable fake time mode in the runtime, like on the go playground.
// There is a slight chance that this won't work because some go code
// might have executed before the variable is set.

//go:linkname faketime runtime.faketime
var faketime = 1

func TestMain(m *testing.M) {
	// We need to use unsafe somehow in order to get access to go:linkname.
	_ = unsafe.Sizeof(0)

	// Run the actual test. runWithPlaygroundTime ensures that the only test
	// that runs is the one calling it.
	runtime.GOMAXPROCS(8)
	os.Exit(m.Run())
}
