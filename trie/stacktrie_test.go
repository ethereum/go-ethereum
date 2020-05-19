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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

func TestRawHPRLP(t *testing.T) {
	got := rawLeafHPRLP([]byte{0x00, 0x01}, []byte{0x02, 0x03}, true)
	exp := []byte{198, 2, 32, 1, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for leaf with even length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x01}, []byte{0x02, 0x03}, true)
	exp = []byte{196, 49, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for leaf with odd length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x00, 0x01}, []byte{0x02, 0x03}, false)
	exp = []byte{198, 2, 0, 1, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for ext with even length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}

	got = rawLeafHPRLP([]byte{0x01}, []byte{0x02, 0x03}, false)
	exp = []byte{196, 17, 130, 2, 3}

	if !bytes.Equal(exp, got) {
		t.Fatalf("invalid RLP generated for ext with odd length key: got %v, expected %v", common.ToHex(got), common.ToHex(exp))
	}
}

func TestHashWithSmallRLP(t *testing.T) {
	trie := NewReStackTrie()
	trie.insert([]byte{0, 1, 2}, []byte("b"))
	trie.insert([]byte{0, 1, 3}, []byte("c"))

	aotrie := NewAppendOnlyTrie()
	aotrie.root = aotrie.insert(aotrie.root, nil, []byte{0, 1, 2}, valueNode([]byte("b")))
	aotrie.root = aotrie.insert(aotrie.root, nil, []byte{0, 1, 3}, valueNode([]byte("c")))

	d := sha3.NewLegacyKeccak256()
	d.Write(trie.hash())
	got := d.Sum(nil)
	exp := aotrie.Hash()

	if !bytes.Equal(got, exp[:]) {
		t.Fatalf("error calculating hash of ext-node-leaves < 32: %v != %v", common.ToHex(exp[:]), common.ToHex(got))
	}

	trie = NewReStackTrie()
	trie.insert([]byte{0, 1, 2}, []byte("ba"))
	trie.insert([]byte{0, 2, 3}, []byte("cr"))

	aotrie = NewAppendOnlyTrie()
	aotrie.root = aotrie.insert(aotrie.root, nil, []byte{0, 1, 2}, valueNode([]byte("ba")))
	aotrie.root = aotrie.insert(aotrie.root, nil, []byte{0, 2, 3}, valueNode([]byte("cr")))

	d.Reset()
	d.Write(trie.hash())
	got = d.Sum(nil)
	exph := aotrie.Hash()

	if !bytes.Equal(got, exph[:]) {
		t.Fatalf("error calculating hash of node-leaves < 32: %v != %v", common.ToHex(exph[:]), common.ToHex(got))
	}
}
