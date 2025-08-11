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
)

func waitIndexing(db *Database) {
	for {
		metadata := loadIndexMetadata(db.diskdb)
		if metadata != nil && metadata.Last >= db.tree.bottom().stateID() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func checkHistoricState(env *tester, root common.Hash, hr *historyReader) error {
	// Short circuit if the historical state is no longer available
	if rawdb.ReadStateID(env.db.diskdb, root) == nil {
		return nil
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

	env := newTester(t, historyLimit, false, 64, true, "")
	defer env.release()
	waitIndexing(env.db)

	var (
		roots = env.roots
		dRoot = env.db.tree.bottom().rootHash()
		hr    = newHistoryReader(env.db.diskdb, env.db.freezer)
	)
	for _, root := range roots {
		if root == dRoot {
			break
		}
		if err := checkHistoricState(env, root, hr); err != nil {
			t.Fatal(err)
		}
	}

	// Pile up more histories on top, ensuring the historic reader is not affected
	env.extend(4)
	waitIndexing(env.db)

	for _, root := range roots {
		if root == dRoot {
			break
		}
		if err := checkHistoricState(env, root, hr); err != nil {
			t.Fatal(err)
		}
	}
}
