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

// transactionAccount is the account metadata visible to transaction
// assertions. Absent accounts have zero balance/nonce and the empty code hash.
type transactionAccount struct {
	balance  uint256.Int
	nonce    uint64
	codeHash common.Hash
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

// TransactionCodeChange is a net change to an account's code hash, computed
// over the code as stored: a delegation indicator is hashed itself.
type TransactionCodeChange struct {
	Address       common.Address
	Before, After common.Hash
}

// TransactionNonceChange is a net change to an account's nonce.
type TransactionNonceChange struct {
	Address       common.Address
	Before, After uint64
}

// TransactionDiff is the EIP-7906 view of the executing transaction before
// gas settlement. Tables are address/key sorted; events retain emission
// order. The caller must not mutate the returned view or the logs it
// references.
type TransactionDiff struct {
	Balances []TransactionBalanceChange
	Slots    []TransactionSlotChange
	Codes    []TransactionCodeChange
	Nonces   []TransactionNonceChange
	Events   []*types.Log
}

func newTransactionAccount(obj *stateObject) transactionAccount {
	account := transactionAccount{codeHash: types.EmptyCodeHash}
	if obj != nil {
		account.balance.Set(obj.data.Balance)
		account.nonce = obj.data.Nonce
		account.codeHash = common.BytesToHash(obj.data.CodeHash)
	}
	return account
}

// transactionStartAccount returns the account's balance, nonce and code hash
// at transaction start without recording an access.
func (s *StateDB) transactionStartAccount(addr common.Address) transactionAccount {
	account := newTransactionAccount(s.stateObjects[addr])
	if original := s.journal.mutations[addr]; original != nil {
		if original.balanceSet {
			account.balance.Set(original.balance)
		}
		if original.nonceSet {
			account.nonce = original.nonce
		}
		if original.codeSet {
			account.codeHash = crypto.Keccak256Hash(original.code)
		}
	}
	return account
}

// GetTransactionDiff builds a transaction-local diff without recording BAL or
// EIP-2929 accesses. Storage wiped without an explicit slot write is not listed.
func (s *StateDB) GetTransactionDiff() *TransactionDiff {
	diff := &TransactionDiff{
		Events: s.GetLogs(s.thash, 0, common.Hash{}, 0),
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
		before := s.transactionStartAccount(addr)
		after := newTransactionAccount(s.stateObjects[addr])
		if before.balance != after.balance {
			diff.Balances = append(diff.Balances, TransactionBalanceChange{addr, before.balance, after.balance})
		}
		if before.codeHash != after.codeHash {
			diff.Codes = append(diff.Codes, TransactionCodeChange{addr, before.codeHash, after.codeHash})
		}
		if before.nonce != after.nonce {
			diff.Nonces = append(diff.Nonces, TransactionNonceChange{addr, before.nonce, after.nonce})
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
			}
		}
	}
	slices.SortFunc(diff.Balances, func(a, b TransactionBalanceChange) int { return a.Address.Cmp(b.Address) })
	slices.SortFunc(diff.Codes, func(a, b TransactionCodeChange) int { return a.Address.Cmp(b.Address) })
	slices.SortFunc(diff.Nonces, func(a, b TransactionNonceChange) int { return a.Address.Cmp(b.Address) })
	slices.SortFunc(diff.Slots, func(a, b TransactionSlotChange) int {
		if cmp := a.Address.Cmp(b.Address); cmp != 0 {
			return cmp
		}
		return a.Key.Cmp(b.Key)
	})
	return diff
}
