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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/trie/triestate"
)

func makeTestNode(owners []common.Hash, paths [][][]byte) *trienode.MergedNodeSet {
	merged := trienode.NewMergedNodeSet()
	for i, owner := range owners {
		set := trienode.NewNodeSet(owner)
		for _, path := range paths[i] {
			blob := testrand.Bytes(32)
			set.AddNode(path, &trienode.Node{
				Blob: blob,
				Hash: crypto.Keccak256Hash(blob),
			})
		}
		merged.Merge(set)
	}
	return merged
}

func TestNodeLookup(t *testing.T) {
	tr := newTestLayerTree() // base = 0x1

	tr.add(common.Hash{0x2}, common.Hash{0x1}, 1, makeTestNode(
		[]common.Hash{
			{0xa}, {0xb},
		},
		[][][]byte{
			{
				{0x1}, {0x2},
			},
			{
				{0x3},
			},
		},
	), triestate.New(nil, nil))

	tr.add(common.Hash{0x3}, common.Hash{0x2}, 2, makeTestNode(
		[]common.Hash{
			{0xa}, {0xc},
		},
		[][][]byte{
			{
				{0x1}, {0x3},
			},
			{
				{0x4},
			},
		},
	), triestate.New(nil, nil))

	tr.add(common.Hash{0x4}, common.Hash{0x3}, 3, trienode.NewMergedNodeSet(), triestate.New(nil, nil))

	var cases = []struct {
		account common.Hash
		path    []byte
		state   common.Hash
		expect  common.Hash
	}{
		{
			// unknown owner
			common.Hash{0xd}, nil, common.Hash{0x4}, common.Hash{0x1},
		},
		{
			// unknown path
			common.Hash{0xa}, []byte{0x4}, common.Hash{0x4}, common.Hash{0x1},
		},
		/*
			lookup node from the tip
		*/
		{
			common.Hash{0xa}, []byte{0x1}, common.Hash{0x4}, common.Hash{0x3},
		},
		{
			common.Hash{0xa}, []byte{0x2}, common.Hash{0x4}, common.Hash{0x2},
		},
		{
			common.Hash{0xa}, []byte{0x3}, common.Hash{0x4}, common.Hash{0x3},
		},
		{
			common.Hash{0xb}, []byte{0x3}, common.Hash{0x4}, common.Hash{0x2},
		},
		{
			common.Hash{0xc}, []byte{0x4}, common.Hash{0x4}, common.Hash{0x3},
		},
		/*
			lookup node from the middle
		*/
		{
			common.Hash{0xa}, []byte{0x1}, common.Hash{0x3}, common.Hash{0x3},
		},
		{
			common.Hash{0xa}, []byte{0x2}, common.Hash{0x3}, common.Hash{0x2},
		},
		{
			common.Hash{0xa}, []byte{0x3}, common.Hash{0x3}, common.Hash{0x3},
		},
		{
			common.Hash{0xb}, []byte{0x3}, common.Hash{0x3}, common.Hash{0x2},
		},
		{
			common.Hash{0xc}, []byte{0x4}, common.Hash{0x3}, common.Hash{0x3},
		},
		/*
			lookup node from the bottom
		*/
		{
			common.Hash{0xa}, []byte{0x1}, common.Hash{0x2}, common.Hash{0x2},
		},
		{
			common.Hash{0xa}, []byte{0x2}, common.Hash{0x2}, common.Hash{0x2},
		},
		{
			common.Hash{0xa}, []byte{0x3}, common.Hash{0x2}, common.Hash{0x1},
		},
		{
			common.Hash{0xb}, []byte{0x3}, common.Hash{0x2}, common.Hash{0x2},
		},
		{
			common.Hash{0xc}, []byte{0x4}, common.Hash{0x2}, common.Hash{0x1},
		},
	}
	for i, c := range cases {
		l := tr.lookupNode(c.account, c.path, c.state)
		if l.rootHash() != c.expect {
			t.Errorf("Unexpected tiphash, %d, want: %x, got: %x", i, c.expect, l.rootHash())
		}
	}
}
