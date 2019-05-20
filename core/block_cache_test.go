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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func newChain(size int) (chain []*types.Block) {
	var parentHash common.Hash
	for i := 0; i < size; i++ {
		head := &types.Header{ParentHash: parentHash, Number: big.NewInt(int64(i))}
		block := types.NewBlock(head, nil, nil, nil)
		chain = append(chain, block)
		parentHash = block.Hash()
	}
	return chain
}

func insertChainCache(cache *BlockCache, chain []*types.Block) {
	for _, block := range chain {
		cache.Push(block)
	}
}

func TestNewBlockCache(t *testing.T) {
	chain := newChain(3)
	cache := NewBlockCache(2)
	insertChainCache(cache, chain)

	if cache.hashes[0] != chain[1].Hash() {
		t.Error("oldest block incorrect")
	}
}

func TestInclusion(t *testing.T) {
	chain := newChain(3)
	cache := NewBlockCache(3)
	insertChainCache(cache, chain)

	for _, block := range chain {
		if b := cache.Get(block.Hash()); b == nil {
			t.Errorf("getting %x failed", block.Hash())
		}
	}
}

func TestDeletion(t *testing.T) {
	chain := newChain(3)
	cache := NewBlockCache(3)
	insertChainCache(cache, chain)

	cache.Delete(chain[1].Hash())

	if cache.Has(chain[1].Hash()) {
		t.Errorf("expected %x not to be included")
	}
}
