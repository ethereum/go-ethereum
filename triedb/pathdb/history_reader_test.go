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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

func waitIndexing(db *Database) {
	for {
		// The metadata is persisted before the indexer reports inited, so check both
		metadata := loadIndexMetadata(db.diskdb, typeStateHistory)
		if metadata != nil && metadata.Last >= db.tree.bottom().stateID() && db.stateIndexer.inited() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func stateAvail(id uint64, env *tester) bool {
	if env.db.config.StateHistory == 0 {
		return true
	}
	dl := env.db.tree.bottom()
	if dl.stateID() <= env.db.config.StateHistory {
		return true
	}
	firstID := dl.stateID() - env.db.config.StateHistory + 1

	return id+1 >= firstID
}

func checkHistoricalState(env *tester, root common.Hash, id uint64, hr *stateHistoryReader) error {
	if !stateAvail(id, env) {
		return nil
	}

	// Short circuit if the historical state is no longer available
	if rawdb.ReadStateID(env.db.diskdb, root) == nil {
		return fmt.Errorf("state not found %d %x", id, root)
	}

	var (
		dl       = env.db.tree.bottom()
		stateID  = rawdb.ReadStateID(env.db.diskdb, root)
		accounts = env.snapAccounts[root]
		storages = env.snapStorages[root]
	)
	for addrHash, accountData := range accounts {
		latest, _ := dl.account(addrHash, 0)
		blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), *stateID, dl.stateID(), latest)
		if err != nil {
			return err
		}
		if !bytes.Equal(accountData, blob) {
			return fmt.Errorf("wrong account data, expected %x, got %x", accountData, blob)
		}
	}
	for i := 0; i < len(env.roots); i++ {
		if env.roots[i] == root {
			break
		}
		// Find all accounts deleted in the past, ensure the associated data is null
		for addrHash := range env.snapAccounts[env.roots[i]] {
			if _, ok := accounts[addrHash]; !ok {
				latest, _ := dl.account(addrHash, 0)
				blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), *stateID, dl.stateID(), latest)
				if err != nil {
					return err
				}
				if len(blob) != 0 {
					return fmt.Errorf("wrong account data, expected null, got %x", blob)
				}
			}
		}
	}
	for addrHash, slots := range storages {
		for slotHash, slotData := range slots {
			latest, _ := dl.storage(addrHash, slotHash, 0)
			blob, err := hr.read(newStorageIdentQuery(env.accountPreimage(addrHash), addrHash, env.hashPreimage(slotHash), slotHash), *stateID, dl.stateID(), latest)
			if err != nil {
				return err
			}
			if !bytes.Equal(slotData, blob) {
				return fmt.Errorf("wrong storage data, expected %x, got %x", slotData, blob)
			}
		}
	}
	for i := 0; i < len(env.roots); i++ {
		if env.roots[i] == root {
			break
		}
		// Find all storage slots deleted in the past, ensure the associated data is null
		for addrHash, slots := range env.snapStorages[env.roots[i]] {
			for slotHash := range slots {
				_, ok := storages[addrHash]
				if ok {
					_, ok = storages[addrHash][slotHash]
				}
				if !ok {
					latest, _ := dl.storage(addrHash, slotHash, 0)
					blob, err := hr.read(newStorageIdentQuery(env.accountPreimage(addrHash), addrHash, env.hashPreimage(slotHash), slotHash), *stateID, dl.stateID(), latest)
					if err != nil {
						return err
					}
					if len(blob) != 0 {
						return fmt.Errorf("wrong storage data, expected null, got %x", blob)
					}
				}
			}
		}
	}
	return nil
}

func TestHistoryReader(t *testing.T) {
	testHistoryReader(t, 0)  // with all histories reserved
	testHistoryReader(t, 10) // with latest 10 histories reserved
}

func testHistoryReader(t *testing.T, historyLimit uint64) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
	config := &testerConfig{
		stateHistory: historyLimit,
		layers:       64,
		enableIndex:  true,
	}
	env := newTester(t, config)
	defer env.release()
	waitIndexing(env.db)

	var (
		roots = env.roots
		dl    = env.db.tree.bottom()
		hr    = newStateHistoryReader(env.db.diskdb, env.db.stateFreezer, nil)
	)
	for i, root := range roots {
		if root == dl.rootHash() {
			break
		}
		if err := checkHistoricalState(env, root, uint64(i+1), hr); err != nil {
			t.Fatal(err)
		}
	}

	// Pile up more histories on top, ensuring the historic reader is not affected
	env.extend(4)
	waitIndexing(env.db)

	for i, root := range roots {
		if root == dl.rootHash() {
			break
		}
		if err := checkHistoricalState(env, root, uint64(i+1), hr); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHistoricalStateReader(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	//log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
	config := &testerConfig{
		stateHistory: 0,
		layers:       64,
		enableIndex:  true,
	}
	env := newTester(t, config)
	defer env.release()
	waitIndexing(env.db)

	// non-canonical state
	fakeRoot := testrand.Hash()
	rawdb.WriteStateID(env.db.diskdb, fakeRoot, 10)

	_, err := env.db.HistoricReader(fakeRoot)
	if err == nil {
		t.Fatal("expected error")
	}
	t.Log(err)

	// canonical state
	realRoot := env.roots[9]
	_, err = env.db.HistoricReader(realRoot)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// TestHistoricalStateReaderPendingWrites ensures the historical reads wait for
// the histories written in background, rather than failing while the disk layer
// is ahead of the indexed histories, and are served correctly while the disk
// layer is behind.
func TestHistoricalStateReaderPendingWrites(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	// Disable the buffer flushes, which wait for the stalled writer below
	writeBuffer := 64 * 1024 * 1024
	env := newTester(t, &testerConfig{layers: 32, enableIndex: true, writeBuffer: &writeBuffer})
	defer env.release()
	waitIndexing(env.db)

	root := env.roots[9]
	reader, err := env.db.HistoricReader(root)
	if err != nil {
		t.Fatalf("Failed to construct the reader: %v", err)
	}
	// Stall the background writer and commit a new layer, leaving the disk layer
	// ahead of the indexed histories.
	var (
		stall  = make(chan struct{})
		resume = sync.OnceFunc(func() { close(stall) })
	)
	defer resume() // runs before the release, which waits for the writer

	if err := env.db.writer.submit(func() error { <-stall; return nil }); err != nil {
		t.Fatalf("Failed to stall the writer: %v", err)
	}
	env.extend(1)
	if meta := loadIndexMetadata(env.db.diskdb, typeStateHistory); meta.Last >= env.db.tree.bottom().stateID() {
		t.Fatalf("Unexpected indexed histories, indexed: %d, disk: %d", meta.Last, env.db.tree.bottom().stateID())
	}
	time.AfterFunc(100*time.Millisecond, resume)

	for addrHash, want := range env.snapAccounts[root] {
		blob, err := reader.AccountRLP(env.accountPreimage(addrHash))
		if err != nil {
			t.Fatalf("Failed to read the historical account: %v", err)
		}
		if !bytes.Equal(blob, want) {
			t.Fatalf("Unexpected account data, want: %x, got: %x", want, blob)
		}
	}
	// Commit more layers after resolving the latest values of the disk layer,
	// leaving it behind the indexed histories.
	var (
		dl     = env.db.tree.bottom()
		latest = make(map[common.Hash][]byte)
	)
	for addrHash := range env.snapAccounts[root] {
		latest[addrHash], _ = dl.account(addrHash, 0)
	}
	env.extend(2)
	if err := env.db.waitWrites(); err != nil {
		t.Fatalf("Failed to write histories: %v", err)
	}
	hr := newStateHistoryReader(env.db.diskdb, env.db.stateFreezer, nil)
	for addrHash, want := range env.snapAccounts[root] {
		blob, err := hr.read(newAccountIdentQuery(env.accountPreimage(addrHash), addrHash), reader.id, dl.stateID(), latest[addrHash])
		if err != nil {
			t.Fatalf("Failed to read the historical account: %v", err)
		}
		if !bytes.Equal(blob, want) {
			t.Fatalf("Unexpected account data, want: %x, got: %x", want, blob)
		}
	}
}

// TestHistoricalStateReaderConcurrentCommit ensures the historical reads are not
// affected by the disk layer replaced by concurrent commits.
func TestHistoricalStateReaderConcurrentCommit(t *testing.T) {
	maxDiffLayers = 4
	defer func() {
		maxDiffLayers = 128
	}()

	env := newTester(t, &testerConfig{layers: 16, enableIndex: true})
	defer env.release()
	waitIndexing(env.db)

	type slot struct {
		addr common.Address
		key  common.Hash
	}
	var (
		root     = env.roots[9]
		accounts = make(map[common.Address][]byte)
		storages = make(map[slot][]byte)
	)
	for addrHash, blob := range env.snapAccounts[root] {
		accounts[env.accountPreimage(addrHash)] = blob
	}
	for addrHash, slots := range env.snapStorages[root] {
		for slotHash, blob := range slots {
			storages[slot{env.accountPreimage(addrHash), env.hashPreimage(slotHash)}] = blob
		}
	}
	var (
		done = make(chan struct{})
		wg   sync.WaitGroup
	)
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			reader, err := env.db.HistoricReader(root)
			if err != nil {
				t.Errorf("Failed to construct the reader: %v", err)
				return
			}
			for {
				select {
				case <-done:
					return
				default:
				}
				for addr, want := range accounts {
					blob, err := reader.AccountRLP(addr)
					if err != nil {
						t.Errorf("Failed to read the historical account: %v", err)
						return
					}
					if !bytes.Equal(blob, want) {
						t.Errorf("Unexpected account data, want: %x, got: %x", want, blob)
						return
					}
				}
				for s, want := range storages {
					blob, err := reader.Storage(s.addr, s.key)
					if err != nil {
						t.Errorf("Failed to read the historical storage: %v", err)
						return
					}
					if !bytes.Equal(blob, want) {
						t.Errorf("Unexpected storage data, want: %x, got: %x", want, blob)
						return
					}
				}
			}
		})
	}
	// Commit layers on top, each of which replaces the bottom disk layer
	env.extend(64)
	close(done)
	wg.Wait()
}
