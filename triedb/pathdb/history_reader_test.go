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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

func waitIndexing(db *Database) {
	for {
		metadata := loadIndexMetadata(db.diskdb, typeStateHistory)
		if metadata != nil && metadata.Last >= db.tree.bottom().stateID() {
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

func checkHistoricalState(env *tester, root common.Hash, id uint64, hr *historyReader) error {
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
		hr    = newHistoryReader(env.db.diskdb, env.db.stateFreezer)
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
