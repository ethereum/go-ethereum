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

package trie

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
)

func TestVerkleIterator(t *testing.T) {
	trie := NewVerkleTrie(verkle.New(), NewDatabase(rawdb.NewMemoryDatabase()), utils.NewPointCache(), true)
	account0 := &types.StateAccount{
		Nonce:    1,
		Balance:  big.NewInt(2),
		Root:     emptyRoot,
		CodeHash: nil,
	}
	// NOTE: the code size isn't written to the trie via TryUpdateAccount
	// so it will be missing from the test nodes.
	trie.TryUpdateAccount(zero[:], account0)
	account1 := &types.StateAccount{
		Nonce:    1337,
		Balance:  big.NewInt(2000),
		Root:     emptyRoot,
		CodeHash: nil,
	}
	// This address is meant to hash to a value that has the same first byte as 0xbf
	var clash = common.Hex2Bytes("69fd8034cdb20934dedffa7dccb4fb3b8062a8be")
	trie.TryUpdateAccount(clash, account1)

	// Manually go over every node to check that we get all
	// the correct nodes.
	it := trie.NodeIterator(nil)
	var leafcount int
	for it.Next(true) {
		t.Logf("Node: %x", it.Path())
		if it.Leaf() {
			leafcount++
			t.Logf("\tLeaf: %x", it.LeafKey())
		}
	}
	if leafcount != 6 {
		t.Fatalf("invalid leaf count: %d != 6", leafcount)
	}
}
