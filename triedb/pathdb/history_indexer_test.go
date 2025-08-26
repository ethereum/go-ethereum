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
)

// TestHistoryIndexerShortenDeadlock tests that a call to shorten does not
// deadlock when the indexer is active. This specifically targets the case where
// signal.result must be sent to unblock the caller.
func TestHistoryIndexerShortenDeadlock(t *testing.T) {
	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	db := rawdb.NewMemoryDatabase()
	freezer, _ := rawdb.NewStateFreezer(t.TempDir(), false, false)
	defer freezer.Close()

	histories := makeStateHistories(100)
	for i, h := range histories {
		accountData, storageData, accountIndex, storageIndex := h.encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), h.meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	// As a workaround, assign a future block to keep the initer running indefinitely
	indexer := newHistoryIndexer(db, freezer, 200)
	defer indexer.close()

	done := make(chan error, 1)
	go func() {
		done <- indexer.shorten(200)
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
