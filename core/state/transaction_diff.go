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
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// TransactionAccount is the account metadata visible to transaction assertions.
// Absent accounts have zero balance/nonce and the empty code hash.
type TransactionAccount struct {
	Balance  uint256.Int
	Nonce    uint64
	CodeHash common.Hash
}

// TransactionBalanceChange is a net balance change since transaction start.
type TransactionBalanceChange struct {
	Address       common.Address
	Before, After uint256.Int
}

// TransactionSlotChange is a net change to a slot written by this transaction.
type TransactionSlotChange struct {
	Address            common.Address
	Key, Before, After common.Hash
}

// TransactionDeployment is newly installed, non-delegation contract code.
type TransactionDeployment struct {
	Address  common.Address
	CodeHash common.Hash
}

// TransactionDiff is the EIP-7906 view before gas settlement and deferred
// deletion. Tables are address/key sorted; events retain emission order.
// The caller must not mutate the returned view or the logs it references.
type TransactionDiff struct {
	Balances     []TransactionBalanceChange
	Slots        []TransactionSlotChange
	Deployed     []TransactionDeployment
	Events       []*types.Log
	AccountFlags map[common.Address]uint64
	SlotIndices  map[common.Address][]uint64
	EventIndices map[common.Address][]uint64
	TopicIndices map[common.Hash][]uint64
}

func transactionAccount(obj *stateObject) TransactionAccount {
	account := TransactionAccount{CodeHash: types.EmptyCodeHash}
	if obj != nil {
		account.Balance.Set(obj.data.Balance)
		account.Nonce = obj.data.Nonce
		account.CodeHash = common.BytesToHash(obj.data.CodeHash)
	}
	return account
}

// GetTransactionAccount returns the account's balance, nonce and code hash at
// transaction start without recording an access. The caller must perform the
// corresponding live read first.
func (s *StateDB) GetTransactionAccount(addr common.Address) TransactionAccount {
	account := transactionAccount(s.stateObjects[addr])
	if original := s.journal.mutations[addr]; original != nil {
		if original.balanceSet {
			account.Balance.Set(original.balance)
		}
		if original.nonceSet {
			account.Nonce = original.nonce
		}
		if original.codeSet {
			account.CodeHash = crypto.Keccak256Hash(original.code)
		}
	}
	return account
}

// GetTransactionState returns a slot's value at transaction start without
// recording an access. The caller must perform the live storage read first.
func (s *StateDB) GetTransactionState(addr common.Address, key common.Hash) common.Hash {
	if obj := s.stateObjects[addr]; obj != nil {
		return obj.getCommittedState(key)
	}
	return common.Hash{}
}

// GetTransactionDiff builds a transaction-local diff without recording BAL or
// EIP-2929 accesses. Storage wiped without an explicit slot write is not listed.
func (s *StateDB) GetTransactionDiff() *TransactionDiff {
	diff := &TransactionDiff{
		Events:       s.GetLogs(s.thash, 0, common.Hash{}, 0),
		AccountFlags: make(map[common.Address]uint64),
		SlotIndices:  make(map[common.Address][]uint64),
		EventIndices: make(map[common.Address][]uint64),
		TopicIndices: make(map[common.Hash][]uint64),
	}
	// Enumerate the slots written by surviving journal entries; reverted
	// writes left the journal together with their call frames.
	written := make(map[common.Address]map[common.Hash]struct{})
	for _, entry := range s.journal.entries {
		if change, ok := entry.(storageChange); ok {
			if written[change.account] == nil {
				written[change.account] = make(map[common.Hash]struct{})
			}
			written[change.account][change.key] = struct{}{}
		}
	}
	for addr := range s.journal.mutations {
		obj := s.stateObjects[addr]
		before := s.GetTransactionAccount(addr)
		after := transactionAccount(obj)
		var flags uint64
		if before.Nonce != after.Nonce {
			flags |= 1
		}
		if before.Balance != after.Balance {
			flags |= 2
			diff.Balances = append(diff.Balances, TransactionBalanceChange{addr, before.Balance, after.Balance})
		}
		if before.CodeHash != after.CodeHash {
			flags |= 8
		}
		diff.AccountFlags[addr] = flags
		if before.CodeHash == types.EmptyCodeHash && after.CodeHash != types.EmptyCodeHash {
			if _, delegation := types.ParseDelegation(obj.Code()); !delegation {
				diff.Deployed = append(diff.Deployed, TransactionDeployment{addr, after.CodeHash})
			}
		}
	}
	for addr, slots := range written {
		obj := s.stateObjects[addr]
		if obj == nil {
			continue
		}
		for key := range slots {
			before := obj.getCommittedState(key)
			after, dirty := obj.dirtyStorage[key]
			if !dirty {
				after = before
			}
			if before != after {
				diff.Slots = append(diff.Slots, TransactionSlotChange{addr, key, before, after})
				diff.AccountFlags[addr] |= 4
			}
		}
	}
	slices.SortFunc(diff.Balances, func(a, b TransactionBalanceChange) int { return a.Address.Cmp(b.Address) })
	slices.SortFunc(diff.Deployed, func(a, b TransactionDeployment) int { return a.Address.Cmp(b.Address) })
	slices.SortFunc(diff.Slots, func(a, b TransactionSlotChange) int {
		if cmp := a.Address.Cmp(b.Address); cmp != 0 {
			return cmp
		}
		return a.Key.Cmp(b.Key)
	})
	for i, slot := range diff.Slots {
		diff.SlotIndices[slot.Address] = append(diff.SlotIndices[slot.Address], uint64(i))
	}
	for i, event := range diff.Events {
		diff.EventIndices[event.Address] = append(diff.EventIndices[event.Address], uint64(i))
		for j := 1; j < len(event.Topics); j++ {
			topic := event.Topics[j]
			// An event matching multiple indexed positions appears only once.
			if !slices.Contains(event.Topics[1:j], topic) {
				diff.TopicIndices[topic] = append(diff.TopicIndices[topic], uint64(i))
			}
		}
	}
	return diff
}

// accessListSlot also identifies independently warmed assertion storage keys.
type accessListSlot struct {
	address common.Address
	slot    common.Hash
}

// AddSlotToAccessListOnly warms a storage key without warming its account.
func (s *StateDB) AddSlotToAccessListOnly(addr common.Address, slot common.Hash) {
	if _, present := s.accessList.Contains(addr, slot); present {
		return
	}
	if s.accessList.slotOnly == nil {
		s.accessList.slotOnly = make(map[accessListSlot]struct{})
	}
	key := accessListSlot{addr, slot}
	s.accessList.slotOnly[key] = struct{}{}
	s.journal.append(accessListAddSlotOnlyChange{key})
}

type accessListAddSlotOnlyChange struct{ key accessListSlot }

func (ch accessListAddSlotOnlyChange) revert(s *StateDB) { delete(s.accessList.slotOnly, ch.key) }
func (ch accessListAddSlotOnlyChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
}
func (ch accessListAddSlotOnlyChange) copy() journalEntry { return ch }
