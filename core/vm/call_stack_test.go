// Copyright 2024 The go-ethereum Authors
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

package vm

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func BenchmarkCallStackPrecompile1(b *testing.B) {
	benchCallStackPrecompileN(b, 1)
}

func BenchmarkCallStackPrecompile10(b *testing.B) {
	benchCallStackPrecompileN(b, 10)
}

func BenchmarkCallStackPrecompile100(b *testing.B) {
	benchCallStackPrecompileN(b, 100)
}

func benchCallStackPrecompileN(b *testing.B, n int) {
	var calls []*csCall
	for i := 0; i < n; i++ {
		calls = append(calls, &csCall{
			Op:       OpCode(i),
			Address:  common.HexToAddress("0xCdA8dcaEe60ce9d63165Ef025fD98CDA2B99B5B2"),
			Selector: []byte{0xde, 0xad, 0xbe, 0xef},
		})
	}
	callStack := newCallStack()
	callStack.calls = calls
	for i := 0; i < b.N; i++ {
		callStack.Run(nil)
	}
}

func TestCallStackPrecompile(t *testing.T) {
	r := require.New(t)
	callStack := newCallStack()
	callStack.calls = []*csCall{
		{
			Op:       OpCode(0x10),
			Address:  common.HexToAddress("0xCdA8dcaEe60ce9d63165Ef025fD98CDA2B99B5B2"),
			Selector: []byte{0xab, 0xcd, 0xef, 0x12},
		},
		{
			Op:       OpCode(0x20),
			Address:  common.HexToAddress("0xCdA8dcaEe60ce9d63165Ef025fD98CDA2B99B5B2"),
			Selector: []byte{0xde, 0xad, 0xbe, 0xef},
		},
	}
	b, err := callStack.Run(nil)
	r.NoError(err)
	expectedB, err := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000002" +
			"0000000000000000000000000000000000000000000000000000000000000010" +
			"000000000000000000000000cda8dcaee60ce9d63165ef025fd98cda2b99b5b2" +
			"00000000000000000000000000000000000000000000000000000000abcdef12" +
			"0000000000000000000000000000000000000000000000000000000000000020" +
			"000000000000000000000000cda8dcaee60ce9d63165ef025fd98cda2b99b5b2" +
			"00000000000000000000000000000000000000000000000000000000deadbeef",
	)
	r.NoError(err)
	r.Equal(expectedB, b)
}
