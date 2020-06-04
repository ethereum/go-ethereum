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

package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestRawHPRLP(t *testing.T) {
	got := rawLeafHPRLP([]byte{0x00, 0x01}, []byte{0x02, 0x03}, true)
	exp := []byte{198, 130, 32, 1, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for leaf with even length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x01}, []byte{0x02, 0x03}, true)
	exp = []byte{196, 49, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for leaf with odd length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x00, 0x01}, []byte{0x02, 0x03}, false)
	exp = []byte{198, 130, 0, 1, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for ext with even length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x01}, []byte{0x02, 0x03}, false)
	exp = []byte{196, 17, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for ext with odd length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}
}

// smallRLPTrie encodes a list of key, value pairs that will not
type smallRLPTrie []struct {
	Key, Value string
}

var smallRLPTests = []smallRLPTrie{
	// One leaf will have a size > 32, the other not.
	smallRLPTrie{
		{
			"2ba639a09a19480b3290299aa982d38c688871e70b0734ac8aa69b9d59492fb3",
			"8181",
		},
		{
			"2ba639a09acf0edbf01831ef3366124dece00d7e4c498f46126d214a8bca7436",
			"a03330333335343331333033613332333333613330333732653330333033303561",
		},
	},
	// Both leaves have sizes smaller than 32.
	smallRLPTrie{
		{
			"2ba639a09a19480b3290299aa982d38c688871e70b0734ac8aa69b9d59492fb3",
			"8181",
		},
		{
			"2ba639a09acf0edbf01831ef3366124dece00d7e4c498f46126d214a8bca7436",
			"a033",
		},
	},
	// Only one leaf
	smallRLPTrie{
		{
			"2ba639a09a19480b3290299aa982d38c688871e70b0734ac8aa69b9d59492fb3",
			"8181",
		},
	},
	// Leaf with an odd-length value and a size < 32
	smallRLPTrie{
		{
			"2ba639a09a19480b3290299aa982d38c688871e70b0734ac8aa69b9d59492fb3",
			"81",
		},
	},
}

func TestHashWithSmallRLP(t *testing.T) {
	for _, test := range smallRLPTests {
		trie := NewReStackTrie()
		for _, kv := range test {
			trie.TryUpdate(common.FromHex(kv.Key), common.FromHex(kv.Value))
		}

		aotrie := NewAppendOnlyTrie()
		for _, kv := range test {
			aotrie.TryUpdate(common.FromHex(kv.Key), common.FromHex(kv.Value))
		}

		got := trie.Hash()
		exp := aotrie.Hash()

		if !bytes.Equal(got[:], exp[:]) {
			t.Fatalf("error calculating hash for embedded RLP: %v != %v", common.ToHex(exp[:]), common.ToHex(got[:]))
		}
	}
}
