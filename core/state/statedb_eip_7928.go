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
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"
)

// ApplyBlockAccessList installs the post-state recorded in a block access list
// directly into the state, without executing any transactions.
func (s *StateDB) ApplyBlockAccessList(list *bal.BlockAccessList) error {
	if list == nil {
		return nil
	}
	return s.applyBlockAccessList(*list, runtime.GOMAXPROCS(0))
}

type balSlot struct {
	key   common.Hash
	value common.Hash
}

type balAccount struct {
	access *bal.AccountAccess
	slots  []balSlot
	obj    *stateObject // nil if the account ends up untouched
}

// hasMetadataChange reports whether the account has a metadata mutation.
func (a *balAccount) hasMetadataChange() bool {
	return len(a.access.BalanceChanges) != 0 || len(a.access.NonceChanges) != 0 || len(a.access.CodeChanges) != 0
}

// mutated extends the mutation check with storage.
func (a *balAccount) mutated() bool {
	return a.hasMetadataChange() || len(a.slots) != 0
}

// balApplyContext holds the shared context through the concurrent workers.
type balApplyContext struct {
	accountReads atomic.Int64
	storageReads atomic.Int64
	prefetchMu   sync.Mutex
}

func (s *StateDB) applyBlockAccessList(list bal.BlockAccessList, threads int) error {
	var (
		accounts  = make([]*balAccount, 0, len(list))
		addresses = make([]common.Address, 0, len(list))
	)
	for i := range list {
		access := &(list)[i]
		entry := &balAccount{access: access}

		for j := range access.StorageChanges {
			change := &access.StorageChanges[j]
			if n := len(change.SlotChanges); n > 0 {
				entry.slots = append(entry.slots, balSlot{
					key:   change.Slot.Bytes32(),
					value: change.SlotChanges[n-1].PostValue.Bytes32(),
				})
			}
		}
		// Skip the read-only account. It is a cheap validation by checking
		// purely with the access list. Whether the account is truly mutated
		// is only known once its pre-state value is read.
		if !entry.mutated() {
			continue
		}
		accounts = append(accounts, entry)
		addresses = append(addresses, access.Address)
	}
	// Schedule background warming of the account trie for every mutated account.
	if s.prefetcher != nil && len(addresses) > 0 {
		if err := s.prefetcher.prefetch(common.Hash{}, s.originalRoot, common.Address{}, addresses, nil, false); err != nil {
			log.Error("Failed to prefetch account trie", "err", err)
		}
	}
	// Process the accounts by applying the final value of mutated fields.
	var ba balApplyContext
	if err := parallelBALApply(len(accounts), threads, func(i int) error {
		return s.prepareBALAccount(accounts[i], &ba)
	}); err != nil {
		return err
	}
	var storageLoaded int
	for _, entry := range accounts {
		storageLoaded += len(entry.slots)

		obj := entry.obj
		if obj == nil {
			continue
		}
		if obj.empty() {
			s.markDelete(obj.address)
			s.stateObjectsDestruct[obj.address] = obj
		} else {
			s.markUpdate(obj.address)
			s.setStateObject(obj)
		}
	}
	s.AccountLoaded += len(addresses)
	s.AccountReads += time.Duration(ba.accountReads.Load())
	s.StorageLoaded += storageLoaded
	s.StorageReads += time.Duration(ba.storageReads.Load())
	return nil
}

// prepareBALAccount loads one account's pre-state, schedules warming of the
// trie nodes required to hash its mutations, and builds the resulting state
// object.
func (s *StateDB) prepareBALAccount(entry *balAccount, ba *balApplyContext) error {
	// Resolve the account object from the database.
	var (
		addr  = entry.access.Address
		start = time.Now()
	)
	account, err := s.reader.Account(addr)
	ba.accountReads.Add(int64(time.Since(start)))
	if err != nil {
		return fmt.Errorf("load account %x: %w", addr, err)
	}
	obj := newObject(s, addr, account)

	// Apply the final value of each mutated field.
	if n := len(entry.access.BalanceChanges); n > 0 {
		obj.setBalance(entry.access.BalanceChanges[n-1].PostBalance.Clone())
	}
	if n := len(entry.access.NonceChanges); n > 0 {
		obj.setNonce(entry.access.NonceChanges[n-1].PostNonce)
	}
	if n := len(entry.access.CodeChanges); n > 0 {
		code := entry.access.CodeChanges[n-1].NewCode
		obj.setCode(crypto.Keccak256Hash(code), slices.Clone(code))
	}
	if err := s.applyBALStorage(obj, entry.slots, ba); err != nil {
		return err
	}
	// Drop accounts whose writes all reverted to their original value.
	if !entry.hasMetadataChange() && len(obj.pendingStorage) == 0 {
		return nil
	}
	entry.obj = obj
	return nil
}

// applyBALStorage schedules warming of the account's storage trie and stages the
// writes that actually change a slot's value. The storage trie itself is left
// unopened on the object; IntermediateRoot pulls the warmed trie back from the
// prefetcher (or opens it lazily if prefetching is disabled).
func (s *StateDB) applyBALStorage(obj *stateObject, slots []balSlot, ba *balApplyContext) error {
	if len(slots) == 0 {
		return nil
	}
	addr := obj.address

	// Schedule background warming of the storage trie paths to the mutated slots.
	if obj.data.Root != types.EmptyRootHash && s.prefetcher != nil {
		keys := make([]common.Hash, len(slots))
		for i := range slots {
			keys[i] = slots[i].key
		}
		ba.prefetchMu.Lock()
		s.prefetcher.prefetch(obj.addrHash(), obj.data.Root, addr, nil, keys, false)
		ba.prefetchMu.Unlock()
	}
	// Stage the writes that differ from the slot's pre-state value.
	for i := range slots {
		start := time.Now()
		origin, err := s.reader.Storage(addr, slots[i].key)
		ba.storageReads.Add(int64(time.Since(start)))
		if err != nil {
			return fmt.Errorf("load storage %x/%x: %w", addr, slots[i].key, err)
		}
		if slots[i].value == origin {
			continue // slot ended the block at its original value
		}
		obj.originStorage[slots[i].key] = origin
		obj.pendingStorage[slots[i].key] = slots[i].value
		obj.uncommittedStorage[slots[i].key] = origin
	}
	return nil
}

// parallelBALApply invokes apply for every index in [0, tasks) across at most
// workers goroutines, returning the first error reported by any of them.
func parallelBALApply(tasks, workers int, apply func(int) error) error {
	if tasks == 0 {
		return nil
	}
	workers = min(max(workers, 1), tasks)

	var (
		next  atomic.Uint64
		group errgroup.Group
	)
	for range workers {
		group.Go(func() error {
			for {
				i := int(next.Add(1)) - 1
				if i >= tasks {
					return nil
				}
				if err := apply(i); err != nil {
					return err
				}
			}
		})
	}
	return group.Wait()
}
