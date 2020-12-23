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

package bitutil

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common/bitutil"
)

// Fuzz implements a go-fuzz fuzzer method to test various encoding method
// invocations.
func Fuzz(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	if data[0]%2 == 0 {
		return fuzzEncode(data[1:])
	}
	return fuzzDecode(data[1:])
}

// fuzzEncode implements a go-fuzz fuzzer method to test the bitset encoding and
// decoding algorithm.
func fuzzEncode(data []byte) int {
	proc, _ := bitutil.DecompressBytes(bitutil.CompressBytes(data), len(data))
	if !bytes.Equal(data, proc) {
		panic("content mismatch")
	}
	return 1
}

// fuzzDecode implements a go-fuzz fuzzer method to test the bit decoding and
// reencoding algorithm.
func fuzzDecode(data []byte) int {
	blob, err := bitutil.DecompressBytes(data, 1024)
	if err != nil {
		return 0
	}
	if comp := bitutil.CompressBytes(blob); !bytes.Equal(comp, data) {
		panic("content mismatch")
	}
	return 1
}
