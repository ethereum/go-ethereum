// Copyright 2025 The go-ethereum Authors
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
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// TestHistoryIndexerShortenDeadlock tests that a call to shorten does not
// deadlock when the indexer is active. This specifically targets the case where
// signal.result must be sent to unblock the caller.
func TestHistoryIndexerShortenDeadlock(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	freezer, _ := rawdb.NewStateFreezer(t.TempDir(), false, false)
	histories := makeHistories(1000)

	// Assume we only have 100 histories indexed
	for i, h := range histories[:100] {
		accountData, storageData, accountIndex, storageIndex := h.encode()
		rawdb.WriteStateHistory(freezer.(ethdb.AncientWriter), uint64(i+1), h.meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	indexer := newHistoryIndexer(db, freezer, uint64(len(histories)))
	defer indexer.close()
	defer freezer.Close()

	done := make(chan error, 1)
	go func() {
		done <- indexer.shorten(uint64(len(histories)))
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shorten returned an unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shorten to complete, potential deadlock")
	}
}
