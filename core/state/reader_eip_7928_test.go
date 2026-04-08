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
	"fmt"
	"maps"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/holiman/uint256"
)

type countingStateReader struct {
	accounts map[common.Address]int
	storages map[common.Address]map[common.Hash]int
	lock     sync.Mutex
}

func newRefStateReader() *countingStateReader {
	return &countingStateReader{
		accounts: make(map[common.Address]int),
		storages: make(map[common.Address]map[common.Hash]int),
	}
}

func (r *countingStateReader) validate(total int) error {
	var sum int
	for addr, n := range r.accounts {
		if n != 1 {
			return fmt.Errorf("duplicated account access: %x-%d", addr, n)
		}
		sum += 1

		slots, exists := r.storages[addr]
		if !exists {
			continue
		}
		for key, n := range slots {
			if n != 1 {
				return fmt.Errorf("duplicated storage access: %x-%x-%d", addr, key, n)
			}
			sum += 1
		}
	}
	for addr := range r.storages {
		_, exists := r.accounts[addr]
		if !exists {
			return fmt.Errorf("dangling storage access: %x", addr)
		}
	}
	if sum != total {
		return fmt.Errorf("unexpected number of access, want: %d, got: %d", total, sum)
	}
	return nil
}

func (r *countingStateReader) Account(addr common.Address) (*types.StateAccount, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.accounts[addr] += 1
	return nil, nil
}
func (r *countingStateReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	slots, exists := r.storages[addr]
	if !exists {
		slots = make(map[common.Hash]int)
		r.storages[addr] = slots
	}
	slots[slot] += 1
	return common.Hash{}, nil
}

func makeFetchTasks(n int) ([]*fetchTask, int) {
	var (
		total int
		tasks []*fetchTask
	)
	for i := 0; i < n; i++ {
		var slots []common.Hash
		if rand.Intn(3) != 0 {
			for j := 0; j < rand.Intn(100); j++ {
				slots = append(slots, testrand.Hash())
			}
		}
		tasks = append(tasks, &fetchTask{
			addr:  testrand.Address(),
			slots: slots,
		})
		total += len(slots) + 1
	}
	return tasks, total
}

func TestPrefetchReader(t *testing.T) {
	type suite struct {
		tasks   []*fetchTask
		threads int
		total   int
	}
	var suites []suite
	for i := 0; i < 100; i++ {
		tasks, total := makeFetchTasks(100)
		suites = append(suites, suite{
			tasks:   tasks,
			threads: rand.Intn(30) + 1,
			total:   total,
		})
	}
	// num(tasks) < num(threads)
	tasks, total := makeFetchTasks(1)
	suites = append(suites, suite{
		tasks:   tasks,
		threads: 100,
		total:   total,
	})
	for _, s := range suites {
		r := newRefStateReader()
		pr := newPrefetchStateReaderInternal(r, s.tasks, s.threads)
		pr.Wait()
		if err := r.validate(s.total); err != nil {
			t.Fatal(err)
		}
	}
}

func makeFakeSlots(n int) map[common.Hash]struct{} {
	slots := make(map[common.Hash]struct{})
	for i := 0; i < n; i++ {
		slots[testrand.Hash()] = struct{}{}
	}
	return slots
}

type noopStateReader struct{}

func (r *noopStateReader) Account(addr common.Address) (*types.StateAccount, error) { return nil, nil }
func (r *noopStateReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	return common.Hash{}, nil
}

type noopCodeReader struct{}

func (r *noopCodeReader) Has(addr common.Address, codeHash common.Hash) bool { return false }

func (r *noopCodeReader) Code(addr common.Address, codeHash common.Hash) []byte {
	return nil
}

func (r *noopCodeReader) CodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	return 0, nil
}

func TestReaderWithTracker(t *testing.T) {
	var r Reader = newReaderTracker(newReader(&noopCodeReader{}, &noopStateReader{}))

	accesses := map[common.Address]map[common.Hash]struct{}{
		testrand.Address(): makeFakeSlots(10),
		testrand.Address(): makeFakeSlots(0),
	}
	for addr, slots := range accesses {
		r.Account(addr)
		for slot := range slots {
			r.Storage(addr, slot)
		}
	}
	got := r.(StateReaderTracker).GetStateAccessList()
	if len(got) != len(accesses) {
		t.Fatalf("Unexpected access list, want: %d, got: %d", len(accesses), len(got))
	}
	for addr, slots := range got {
		entry, ok := accesses[addr]
		if !ok {
			t.Fatal("Unexpected access list")
		}
		if !maps.Equal(slots, entry) {
			t.Fatal("Unexpected slots")
		}
	}
}

// TestTrackerSurvivesStateDBCache verifies that the BAL reader tracker records
// account and storage accesses even when the StateDB serves them from its
// in-memory cache (stateObjects / originStorage). This is a regression test
// for a bug where a reverted transaction left cached entries that subsequent
// transactions read without hitting the reader, causing the BAL to be incomplete.
func TestTrackerSurvivesStateDBCache(t *testing.T) {
	var (
		sdb            = NewDatabaseForTesting()
		statedb, _     = New(types.EmptyRootHash, sdb)
		addr           = common.HexToAddress("0xaaaa")
		slot           = common.HexToHash("0x01")
	)
	// Set up committed state with one account that has a storage slot.
	statedb.SetBalance(addr, uint256.NewInt(1e18), tracing.BalanceChangeUnspecified)
	statedb.SetNonce(addr, 5, tracing.NonceChangeUnspecified)
	statedb.SetState(addr, slot, common.HexToHash("0x42"))
	root, _ := statedb.Commit(0, false, false)
	sdb.TrieDB().Commit(root, false)

	// Create a fresh StateDB with a reader tracker (as the miner does).
	var (
		reader, _ = sdb.Reader(root)
		tracked   = NewReaderWithTracker(reader)
		live, _   = NewWithReader(root, sdb, tracked)
		tracker   = live.Reader().(StateReaderTracker)
	)

	// Simulate a failed transaction: read account and storage, then revert.
	snap := live.Snapshot()
	live.GetNonce(addr)
	live.GetState(addr, slot)

	reads := tracker.GetStateAccessList()
	if _, ok := reads[addr]; !ok {
		t.Fatal("addr should be tracked after first read")
	}
	if _, ok := reads[addr][slot]; !ok {
		t.Fatal("slot should be tracked after first read")
	}

	tracker.Clear()
	live.RevertToSnapshot(snap)

	reads = tracker.GetStateAccessList()
	if len(reads) != 0 {
		t.Fatal("tracker should be empty after Clear")
	}

	// Simulate the next transaction reading the same account and slot.
	// Both hit the stateObjects/originStorage caches.
	live.GetNonce(addr)
	live.GetState(addr, slot)

	reads = tracker.GetStateAccessList()
	if _, ok := reads[addr]; !ok {
		t.Fatal("addr must be tracked on cache hit (account)")
	}
	if _, ok := reads[addr][slot]; !ok {
		t.Fatal("slot must be tracked on cache hit (storage)")
	}
}
