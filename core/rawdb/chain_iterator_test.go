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

package rawdb

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestChainIterator(t *testing.T) {
	// Construct test chain db
	chainDb := NewMemoryDatabase()

	var block *types.Block
	for i := uint64(0); i <= 10; i++ {
		if i == 0 {
			block = types.NewBlock(&types.Header{Number: big.NewInt(int64(i))}, nil, nil, nil) // Empty genesis block
		} else {
			tx := types.NewTransaction(i, common.BytesToAddress([]byte{0x11}), big.NewInt(111), 1111, big.NewInt(11111), []byte{0x11, 0x11, 0x11})
			block = types.NewBlock(&types.Header{Number: big.NewInt(int64(i))}, []*types.Transaction{tx}, nil, nil)
		}
		WriteBlock(chainDb, block)
		WriteCanonicalHash(chainDb, block.Hash(), block.NumberU64())
	}

	var cases = []struct {
		from, to uint64
		reverse  bool
		expect   []uint64
	}{
		{0, 11, true, []uint64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}},
		{0, 0, true, nil},
		{10, 11, true, []uint64{10}},
		{0, 11, false, []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{0, 0, false, nil},
		{10, 11, false, []uint64{10}},
	}
	for i, c := range cases {
		var visit []uint64
		err := iterateCanonicalChain(chainDb, c.from, c.to, nil, func(db ethdb.Batch, b *types.Block) { visit = append(visit, b.NumberU64()) }, c.reverse, "", "")
		if err != nil {
			t.Fatalf("Case %d failed, err %v", i, err)
		}
		if !reflect.DeepEqual(visit, c.expect) {
			t.Fatalf("Case %d failed, visit element mismatch, want %v, got %v", i, c.expect, visit)
		}
	}
}
