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

package pathdb

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestAccountHistoryIndex_NotIndexed(t *testing.T) {
	// Indexing disabled: every accessor should bail with ErrStateHistoryNotIndexed.
	env := newTester(t, &testerConfig{layers: 4, enableIndex: false})
	defer env.release()

	var addr common.Address
	for h := range env.accounts {
		addr = env.accountPreimage(h)
		break
	}

	if _, err := env.db.AccountHistoryIndex(addr); !errors.Is(err, ErrStateHistoryNotIndexed) {
		t.Fatalf("AccountHistoryIndex: want ErrStateHistoryNotIndexed, got %v", err)
	}
	if _, err := env.db.HistoricAccount(addr, 1); !errors.Is(err, ErrStateHistoryNotIndexed) {
		t.Fatalf("HistoricAccount: want ErrStateHistoryNotIndexed, got %v", err)
	}
}

func TestAccountHistoryIndex_Indexed(t *testing.T) {
	// Force diff-layer flushes so state history is actually written.
	maxDiffLayers = 4
	defer func() { maxDiffLayers = 128 }()

	env := newTester(t, &testerConfig{layers: 32, enableIndex: true})
	defer env.release()
	waitIndexing(env.db)

	// Pick any account with at least one indexed entry.
	var (
		addr common.Address
		idx  HistoryIndexReader
	)
	for h := range env.accounts {
		a := env.accountPreimage(h)
		r, err := env.db.AccountHistoryIndex(a)
		if err != nil {
			t.Fatalf("AccountHistoryIndex(%x): %v", a, err)
		}
		if r.Count() > 0 {
			addr = a
			idx = r
			break
		}
	}
	if idx == nil {
		t.Fatal("no indexed account found across all current-state accounts")
	}

	// Every id reported by the index must yield a HistoricAccount read.
	for i := 0; i < idx.Count(); i++ {
		hid, err := idx.At(i)
		if err != nil {
			t.Fatalf("idx.At(%d): %v", i, err)
		}
		if _, err := env.db.HistoricAccount(addr, hid); err != nil {
			t.Fatalf("HistoricAccount(addr, %d): %v", hid, err)
		}
		blockNum, err := env.db.BlockNumberAt(hid)
		if err != nil {
			t.Fatalf("BlockNumberAt(%d): %v", hid, err)
		}
		if blockNum >= uint64(len(env.roots)) {
			t.Fatalf("BlockNumberAt(%d) = %d, beyond generated range %d", hid, blockNum, len(env.roots))
		}
	}

	last, err := env.db.LastIndexedBlockNumber()
	if err != nil {
		t.Fatalf("LastIndexedBlockNumber: %v", err)
	}
	if last == 0 {
		t.Fatal("LastIndexedBlockNumber returned 0 after waitIndexing")
	}
}
