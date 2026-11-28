// Copyright 2026 The go-ethereum Authors
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

package filtermaps

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestGetBlockLvPointerPartialHead(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	defer db.Close()
	rawdb.WriteBlockLvPointer(db, 20, 100)
	rawdb.WriteBlockLvPointer(db, 21, 200)
	f := &FilterMaps{
		db:             db,
		indexedRange:   filterMapsRange{initialized: true, blocks: common.NewRange[uint64](20, 1)},
		lvPointerCache: lru.NewCache[uint64, uint64](2),
	}
	backend := &FilterMapsMatcherBackend{f: f}

	for _, test := range []struct {
		block uint64
		want  uint64
	}{
		{20, 100},
		{21, 200},
		{22, 200},
	} {
		got, err := backend.GetBlockLvPointer(t.Context(), test.block)
		if err != nil {
			t.Fatalf("block %d: %v", test.block, err)
		}
		if got != test.want {
			t.Errorf("block %d: got pointer %d, want %d", test.block, got, test.want)
		}
	}
}
