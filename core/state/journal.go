// Copyright 2024 the go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type journal interface {
	// snapshot starts a new journal scope which can be reverted or discarded.
	// The lifeycle of journalling is as follows:
	// - snapshot() starts a 'scope'.
	// - The method snapshot() may be called any number of times.
	// - For each call to snapshot, there should be a corresponding call to end
	//  the scope via either of:
	//   - revertToSnapshot, which undoes the changes in the scope, or
	//   - discardSnapshot, which discards the ability to revert the changes in the scope.
	snapshot()

	// revertSnapshot reverts all state changes made since the last call to snapshot().
	revertSnapshot(s *StateDB)

	// discardSnapshot removes the latest snapshot; after calling this
	// method, it is no longer possible to revert to that particular snapshot, the
	// changes are considered part of the parent scope.
	discardSnapshot()

	// reset clears the journal so it can be reused.
	reset()

	// dirtyAccounts returns a list of all accounts modified in this journal
	dirtyAccounts() []common.Address

	// accessListAddAccount journals the adding of addr to the access list
	accessListAddAccount(addr common.Address)

	// accessListAddSlot journals the adding of addr/slot to the access list
	accessListAddSlot(addr common.Address, slot common.Hash)

	// logChange journals the adding of a log related to the txHash
	logChange(txHash common.Hash)

	// createObject journals the event of a new account created in the trie.
	createObject(addr common.Address)

	// createContract journals the creation of a new contract at addr.
	// OBS: This method must not be applied twice, it assumes that the pre-state
	// (i.e the rollback-state) is non-created.
	createContract(addr common.Address, account *types.StateAccount)

	// destruct journals the destruction of an account in the trie.
	// pre-state (i.e the rollback-state) is non-destructed (and, for the purpose
	// of EIP-XXX (TODO lookup), created in this tx).
	destruct(addr common.Address, account *types.StateAccount)

	// storageChange journals a change in the storage data related to addr.
	// It records the key and previous value of the slot.
	storageChange(addr common.Address, key, prev, origin common.Hash)

	// transientStateChange journals a change in the t-storage data related to addr.
	// It records the key and previous value of the slot.
	transientStateChange(addr common.Address, key, prev common.Hash)

	// refundChange journals that the refund has been changed, recording the previous value.
	refundChange(previous uint64)

	// balanceChange journals that the balance of addr has been changed, recording the previous value
	balanceChange(addr common.Address, account *types.StateAccount, destructed, newContract bool)

	// setCode journals that the code of addr has been set.
	setCode(addr common.Address, account *types.StateAccount, prevCode []byte)

	// nonceChange journals that the nonce of addr was changed, recording the previous value.
	nonceChange(addr common.Address, account *types.StateAccount, destructed, newContract bool)

	// touchChange journals that the account at addr was touched during execution.
	touchChange(addr common.Address, account *types.StateAccount, destructed, newContract bool)

	// copy returns a deep-copied journal.
	copy() journal
}
