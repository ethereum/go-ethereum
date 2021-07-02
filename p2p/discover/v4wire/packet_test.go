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

package v4wire

import (
	"encoding/hex"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestDecode(t *testing.T) {
	for i, test := range []struct {
		input  string
		expEnr int
	}{
		{
			input:  "f83acb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a3558082999983999999",
			expEnr: 0,
		},
		{
			input:  "f83acb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a3550182999983999999",
			expEnr: 1,
		},
		{
			input:  "f2cb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a355",
			expEnr: 0,
		},
		{
			input:  "f3cb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a35501",
			expEnr: 1,
		},
		{
			// This is how we previously encoded ENR sequence number of 0.
			input:  "f3cb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a35580",
			expEnr: 0,
		},
		{
			// This input was previously accepted, it contains a non-canonical rlp rest `00`. This vector
			// fails after commit '3e6f46caec51d82aef363632517eb5842eef6db6'
			input:  "f3cb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a35500",
			expEnr: -1,
		},
		{ // This is how besu encodes a pong with enr sequence 0
			input:  "f83bcb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a355880000000000000000",
			expEnr: -1,
		},
		{ // This is how besu encodes a pong with enr sequence 1
			input:  "f83bcb847f000001820cfa8215a8a000000000000000000000000000000000000000000000000000000000000012348443b9a355880000000000000001",
			expEnr: -1,
		},
	} {
		input, err := hex.DecodeString(test.input)
		if err != nil {
			t.Fatalf("test %d: invalid hex: %s", i, test.input)
		}
		var pongPacket Pong
		err = rlp.DecodeBytes(input, &pongPacket)
		if err != nil && test.expEnr > -1 {
			t.Errorf("test %d: did not accept packet %s\n%v", i, test.input, err)
			continue
		}
		if test.expEnr == -1 {
			if err == nil {
				t.Errorf("test %d: expected failure", i)
			}
			continue
		}
		if have, want := pongPacket.ENRSeq, uint64(test.expEnr); have != want {
			t.Logf("test %d: got %s\n", i, spew.Sdump(pongPacket))
			t.Errorf("test %d, wrong enr seq, have %d, want %d", i, have, want)
		}
	}
}
