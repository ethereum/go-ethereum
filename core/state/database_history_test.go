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

package state

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// iteratedSlot is a storage slot as an iterator reports it.
type iteratedSlot struct {
	hash  common.Hash
	key   common.Hash
	value common.Hash
}

// iteratedAccount is an account as an iterator reports it, with its storage.
type iteratedAccount struct {
	hash    common.Hash
	address common.Address
	account types.StateAccount
	storage []iteratedSlot
}

// iterateState walks every account and storage slot of the state from the
// given account hash on.
func iterateState(t *testing.T, iteratee Iteratee, start common.Hash) []iteratedAccount {
	t.Helper()

	acctIt, err := iteratee.NewAccountIterator(start)
	if err != nil {
		t.Fatalf("failed to create account iterator: %v", err)
	}
	defer acctIt.Release()

	var accounts []iteratedAccount
	for acctIt.Next() {
		addr, err := acctIt.Address()
		if err != nil {
			t.Fatalf("failed to resolve the address of %x: %v", acctIt.Hash(), err)
		}
		acct := iteratedAccount{hash: acctIt.Hash(), address: addr, account: *acctIt.Account()}

		storageIt, err := iteratee.NewStorageIterator(acct.hash, common.Hash{})
		if err != nil {
			t.Fatalf("failed to create storage iterator: %v", err)
		}
		for storageIt.Next() {
			key, err := storageIt.Key()
			if err != nil {
				t.Fatalf("failed to resolve the slot key of %x: %v", storageIt.Hash(), err)
			}
			acct.storage = append(acct.storage, iteratedSlot{hash: storageIt.Hash(), key: key, value: storageIt.Slot()})
		}
		if err := storageIt.Error(); err != nil {
			t.Fatalf("storage iteration failed: %v", err)
		}
		storageIt.Release()
		accounts = append(accounts, acct)
	}
	if err := acctIt.Error(); err != nil {
		t.Fatalf("account iteration failed: %v", err)
	}
	return accounts
}

// TestHistoricIteratee checks that a historical state, served from the trienode
// history, iterates exactly as it did while it was live. It runs in a synctest
// bubble, so waiting for the background history indexer takes no real time.
func TestHistoricIteratee(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		disk, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer disk.Close()
		pconfig := *pathdb.Defaults
		pconfig.NoAsyncFlush = true
		pconfig.EnableStateIndexing = true
		pconfig.NoHistoryIndexDelay = true
		pconfig.TrienodeHistory = 0 // full chain
		tdb := triedb.NewDatabase(disk, &triedb.Config{Preimages: true, PathDB: &pconfig})
		defer tdb.Close()
		sdb := NewDatabase(tdb, nil)

		// Build the state to iterate later: accounts, a quarter of them with storage.
		state, _ := New(types.EmptyRootHash, sdb)
		for i := byte(0); i < 64; i++ {
			addr := common.BytesToAddress([]byte{i})
			state.AddBalance(addr, uint256.NewInt(uint64(i)+1), tracing.BalanceChangeUnspecified)
			state.SetNonce(addr, uint64(i), tracing.NonceChangeUnspecified)
			if i%4 == 0 {
				for j := byte(0); j < 8; j++ {
					state.SetState(addr, common.Hash{j}, common.Hash{i, j})
				}
			}
		}
		root, err := state.Commit(params.Rules{}, 1)
		if err != nil {
			t.Fatalf("failed to commit state: %v", err)
		}
		live, err := sdb.Iteratee(root)
		if err != nil {
			t.Fatalf("failed to iterate the live state: %v", err)
		}
		want := iterateState(t, live, common.Hash{})
		from := want[len(want)/2].hash
		wantFrom := iterateState(t, live, from)

		// Change balances, storage and accounts over the next blocks, then flush
		// them all, so the first state is served from history only.
		head := root
		for block := uint64(2); block <= 4; block++ {
			state, _ := New(head, sdb)
			for i := byte(0); i < 64; i += 3 {
				addr := common.BytesToAddress([]byte{i})
				state.AddBalance(addr, uint256.NewInt(block), tracing.BalanceChangeUnspecified)
				state.SetState(addr, common.Hash{0}, common.Hash{byte(block)})
			}
			state.AddBalance(common.BytesToAddress([]byte{byte(100 + block)}), uint256.NewInt(1), tracing.BalanceChangeUnspecified)
			if head, err = state.Commit(params.Rules{}, block); err != nil {
				t.Fatalf("failed to commit block %d: %v", block, err)
			}
		}
		if err := tdb.Commit(head, false); err != nil {
			t.Fatalf("failed to flush the state: %v", err)
		}
		if it, err := sdb.Iteratee(root); err == nil {
			if acctIt, err := it.NewAccountIterator(common.Hash{}); err == nil && acctIt.Next() {
				t.Fatal("the first state is still live, so this test reads nothing historic")
			}
		}

		// The trienode history is indexed in the background, and the indexer may
		// only start on its second heartbeat, 15 seconds in: the bubble's clock
		// passes that wait at once.
		hdb := NewHistoricDatabase(tdb, nil)
		var historic Iteratee
		for deadline := time.Now().Add(30 * time.Second); ; {
			if historic, err = hdb.Iteratee(root); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("historical state is not available: %v", err)
			}
			time.Sleep(50 * time.Millisecond)
		}
		got := iterateState(t, historic, common.Hash{})
		assertIterated(t, got, want)
		assertIterated(t, iterateState(t, historic, from), wantFrom)

		// The storage of an account with none is empty, and that of an account the
		// state does not hold is too.
		for _, hash := range []common.Hash{want[1].hash, {0xff}} {
			it, err := historic.NewStorageIterator(hash, common.Hash{})
			if err != nil {
				t.Fatalf("failed to create storage iterator: %v", err)
			}
			if it.Next() {
				t.Fatalf("storage of %x is not empty", hash)
			}
			it.Release()
		}
	})
}

func assertIterated(t *testing.T, got, want []iteratedAccount) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("iterated %d accounts, want %d", len(got), len(want))
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.hash != w.hash || g.address != w.address || g.account.Nonce != w.account.Nonce ||
			g.account.Balance.Cmp(w.account.Balance) != 0 || g.account.Root != w.account.Root {
			t.Fatalf("account %d: got %+v, want %+v", i, g, w)
		}
		if len(g.storage) != len(w.storage) {
			t.Fatalf("account %x: %d slots, want %d", w.hash, len(g.storage), len(w.storage))
		}
		for j := range w.storage {
			if g.storage[j] != w.storage[j] {
				t.Fatalf("account %x slot %d: got %+v, want %+v", w.hash, j, g.storage[j], w.storage[j])
			}
		}
	}
}
