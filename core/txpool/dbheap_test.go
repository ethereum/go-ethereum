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

package txpool

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDBHeap(t *testing.T) {
	heap := dbHeap{m: make(map[common.Address]dbNonceList, 0)}
	pool, _ := setupTxPool()
	var keys []*ecdsa.PrivateKey
	for i := 0; i < 100; i++ {
		key, _ := crypto.GenerateKey()
		keys = append(keys, key)
	}
	index := 0
	entries := make(map[uint64]struct{})
	for z, k := range keys {
		for i := 0; i < 100; i++ {
			tx := pricedTransaction(uint64(i), uint64(i+z*1000), big.NewInt(int64(i+z*1000)), k)
			entry, _ := pool.txToTxEntry(tx)
			heap.Add(entry, uint64(index))
			entries[uint64(index)] = struct{}{}
		}
	}
	results := heap.Pop(len(heap.m) + 1)
	for _, res := range results {
		if _, ok := entries[res]; !ok {
			t.Fail()
		}
	}
}
