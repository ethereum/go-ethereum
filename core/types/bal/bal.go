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

package bal

import (
	"bytes"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"maps"
)

// idxAccessListBuilder is responsible for producing the state accesses and
// reads recorded within the scope of a single index in the access list.
type idxAccessListBuilder struct {
	// stores the pre-index values of any account data that was modified in the
	// current index.
	prestates map[common.Address]*accountIdxPrestate

	// A stack which maintains a state access/modification list for each EVM
	// execution frame.
	//
	// Entering a frame appends an empty access list
	// and terminating a frame merges it into the intermediate access list
	// of the calling frame.  If it reverted, any account/storage mutations
	// are converted to accesses and account mutations are discarded before
	// merging.
	accessesStack []map[common.Address]*constructionAccountAccess
}

func newIdxAccessListBuilder() *idxAccessListBuilder {
	return &idxAccessListBuilder{
		make(map[common.Address]*accountIdxPrestate),
		[]map[common.Address]*constructionAccountAccess{
			make(map[common.Address]*constructionAccountAccess),
		},
	}
}

func (c *idxAccessListBuilder) storageRead(address common.Address, key common.Hash) {
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.StorageRead(key)
}

func (c *idxAccessListBuilder) accountRead(address common.Address) {
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
}

func (c *idxAccessListBuilder) storageWrite(address common.Address, key, prevVal, newVal common.Hash) {
	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &accountIdxPrestate{}
	}
	if c.prestates[address].storage == nil {
		c.prestates[address].storage = make(map[common.Hash]common.Hash)
	}
	if _, ok := c.prestates[address].storage[key]; !ok {
		c.prestates[address].storage[key] = prevVal
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.StorageWrite(key, newVal)
}

func (c *idxAccessListBuilder) balanceChange(address common.Address, prev, cur *uint256.Int) {
	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &accountIdxPrestate{}
	}
	if c.prestates[address].balance == nil {
		c.prestates[address].balance = prev
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.BalanceChange(cur)
}

func (c *idxAccessListBuilder) codeChange(address common.Address, prev, cur []byte) {
	// 7702 delegation clear and selfdestruct pass new code as 'nil'.
	// However, internally the constructionAccountAccess uses
	// nil as a sign that an account field was not modified.
	if cur == nil {
		cur = []byte{}
	}

	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &accountIdxPrestate{}
	}
	if c.prestates[address].code == nil {
		c.prestates[address].code = prev
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.CodeChange(cur)
}

// selfDestruct is invoked when an account which has been created and invoked
// SENDALL in the same transaction is removed as part of transaction finalization.
//
// Any storage accesses/modifications performed at the contract during execution
// are retained in the block access list as state reads.
func (c *idxAccessListBuilder) selfDestruct(address common.Address) {
	// convert all the account storage writes to reads, preserve the existing reads
	access := c.accessesStack[len(c.accessesStack)-1][address]
	for key, _ := range access.storageMutations {
		if access.storageReads == nil {
			access.storageReads = make(map[common.Hash]struct{})
		}
		access.storageReads[key] = struct{}{}
	}

	access.storageMutations = nil
}

func (c *idxAccessListBuilder) nonceChange(address common.Address, prev, cur uint64) {
	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &accountIdxPrestate{}
	}
	if c.prestates[address].nonce == nil {
		c.prestates[address].nonce = &prev
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.NonceChange(cur)
}

// enterScope is called after a new EVM frame has been entered.
func (c *idxAccessListBuilder) enterScope() {
	c.accessesStack = append(c.accessesStack, make(map[common.Address]*constructionAccountAccess))
}

// exitScope is called after an EVM call scope terminates.
func (c *idxAccessListBuilder) exitScope(evmErr bool) {
	childAccessList := c.accessesStack[len(c.accessesStack)-1]
	parentAccessList := c.accessesStack[len(c.accessesStack)-2]

	for addr, childAccess := range childAccessList {
		if _, ok := parentAccessList[addr]; !ok {
			parentAccessList[addr] = &constructionAccountAccess{}
		}
		if evmErr {
			parentAccessList[addr].MergeRevertedAccess(childAccess)
		} else {
			parentAccessList[addr].Merge(childAccess)
		}
	}

	c.accessesStack = c.accessesStack[:len(c.accessesStack)-1]
}

// finalise returns the net state mutations at the access list index as well as
// state which was accessed.  The idxAccessListBuilder instance should be discarded
// after calling finalise.
func (a *idxAccessListBuilder) finalise() (*StateDiff, StateAccesses) {
	diff := &StateDiff{make(map[common.Address]*AccountMutations)}
	stateAccesses := make(StateAccesses)

	for addr, access := range a.accessesStack[0] {
		// remove any mutations from the access list with no net difference vs the tx prestate value
		if access.nonce != nil && *a.prestates[addr].nonce == *access.nonce {
			access.nonce = nil
		}
		if access.balance != nil && a.prestates[addr].balance.Eq(access.balance) {
			access.balance = nil
		}

		if access.code != nil && bytes.Equal(access.code, a.prestates[addr].code) {
			access.code = nil
		}
		if access.storageMutations != nil {
			for key, val := range access.storageMutations {
				if a.prestates[addr].storage[key] == val {
					delete(access.storageMutations, key)
					access.storageReads[key] = struct{}{}
				}
			}
			if len(access.storageMutations) == 0 {
				access.storageMutations = nil
			}
		}

		if access.storageReads != nil {
			stateAccesses[addr] = access.storageReads
		}

		// if the account has no net mutations against the tx prestate, it must
		// be tracked as an account read.
		if len(access.code) == 0 && access.nonce == nil && access.balance == nil && len(access.storageMutations) == 0 {
			if _, ok := stateAccesses[addr]; !ok {
				stateAccesses[addr] = make(map[common.Hash]struct{})
			}
			continue
		}

		diff.Mutations[addr] = &AccountMutations{
			Balance:       access.balance,
			Nonce:         access.nonce,
			Code:          access.code,
			StorageWrites: access.storageMutations,
		}
	}

	return diff, stateAccesses
}

// FinaliseIdxChanges records all pending state mutations/accesses in the
// access list at the given index.  The set of pending state mutations/accesses are
// then emptied.
func (c *AccessListBuilder) FinaliseIdxChanges(idx uint16) {
	pendingDiff, pendingAccesses := c.idxBuilder.finalise()
	c.idxBuilder = newIdxAccessListBuilder()

	// merge the set of state accesses/modifications into the access list:
	// * any account or storage slot which was already recorded as a read
	//   in the block access list and modified in the index-being-finalized
	//   is removed from the set of accessed state in the block access list.
	// * any account/storage slot which was already recorded as modified
	//   in the block access list and read in the index-being-finalized is
	//   not included in the block access list's set of state reads.
	for addr, pendingAcctDiff := range pendingDiff.Mutations {
		finalizedAcctChanges, ok := c.FinalizedAccesses[addr]
		if !ok {
			finalizedAcctChanges = &constructionAccountAccesses{}
			c.FinalizedAccesses[addr] = finalizedAcctChanges
		}

		if pendingAcctDiff.Nonce != nil {
			if finalizedAcctChanges.nonceChanges == nil {
				finalizedAcctChanges.nonceChanges = make(map[uint16]uint64)
			}
			finalizedAcctChanges.nonceChanges[idx] = *pendingAcctDiff.Nonce
		}
		if pendingAcctDiff.Balance != nil {
			if finalizedAcctChanges.balanceChanges == nil {
				finalizedAcctChanges.balanceChanges = make(map[uint16]*uint256.Int)
			}
			finalizedAcctChanges.balanceChanges[idx] = pendingAcctDiff.Balance
		}
		if pendingAcctDiff.Code != nil {
			if finalizedAcctChanges.codeChanges == nil {
				finalizedAcctChanges.codeChanges = make(map[uint16]CodeChange)
			}
			finalizedAcctChanges.codeChanges[idx] = CodeChange{idx, pendingAcctDiff.Code}
		}
		if pendingAcctDiff.StorageWrites != nil {
			if finalizedAcctChanges.storageWrites == nil {
				finalizedAcctChanges.storageWrites = make(map[common.Hash]map[uint16]common.Hash)
			}
			for key, val := range pendingAcctDiff.StorageWrites {
				if _, ok := finalizedAcctChanges.storageWrites[key]; !ok {
					finalizedAcctChanges.storageWrites[key] = make(map[uint16]common.Hash)
				}
				finalizedAcctChanges.storageWrites[key][idx] = val

				if _, ok := finalizedAcctChanges.storageReads[key]; ok {
					delete(finalizedAcctChanges.storageReads, key)
				}
			}
		}
	}
	// record pending accesses in the BAL access set unless they were
	// already written in a previous index
	for addr, pendingAccountAccesses := range pendingAccesses {
		finalizedAcctAccesses, ok := c.FinalizedAccesses[addr]
		if !ok {
			finalizedAcctAccesses = &constructionAccountAccesses{}
			c.FinalizedAccesses[addr] = finalizedAcctAccesses
		}

		for key := range pendingAccountAccesses {
			if _, ok := finalizedAcctAccesses.storageWrites[key]; ok {
				continue
			}
			if finalizedAcctAccesses.storageReads == nil {
				finalizedAcctAccesses.storageReads = make(map[common.Hash]struct{})
			}
			finalizedAcctAccesses.storageReads[key] = struct{}{}
		}
	}
	c.lastFinalizedMutations = pendingDiff
	c.lastFinalizedAccesses = pendingAccesses
}

func (c *AccessListBuilder) StorageRead(address common.Address, key common.Hash) {
	c.idxBuilder.storageRead(address, key)
}
func (c *AccessListBuilder) AccountRead(address common.Address) {
	c.idxBuilder.accountRead(address)
}
func (c *AccessListBuilder) StorageWrite(address common.Address, key, prevVal, newVal common.Hash) {
	c.idxBuilder.storageWrite(address, key, prevVal, newVal)
}
func (c *AccessListBuilder) BalanceChange(address common.Address, prev, cur *uint256.Int) {
	c.idxBuilder.balanceChange(address, prev, cur)
}
func (c *AccessListBuilder) NonceChange(address common.Address, prev, cur uint64) {
	c.idxBuilder.nonceChange(address, prev, cur)
}
func (c *AccessListBuilder) CodeChange(address common.Address, prev, cur []byte) {
	c.idxBuilder.codeChange(address, prev, cur)
}
func (c *AccessListBuilder) SelfDestruct(address common.Address) {
	c.idxBuilder.selfDestruct(address)
}

func (c *AccessListBuilder) EnterScope() {
	c.idxBuilder.enterScope()
}
func (c *AccessListBuilder) ExitScope(executionErr bool) {
	c.idxBuilder.exitScope(executionErr)
}

// CodeChange contains the runtime bytecode deployed at an address and the
// transaction index where the deployment took place.
type CodeChange struct {
	TxIdx uint16
	Code  []byte `json:"code,omitempty"`
}

// constructionAccountAccesses contains all state mutations which were applied
// to a single account during the execution of a block.
// It contains the final values for the account fields and storage slots which
// were mutated, indexed by where they occurred:
// * pre-transaction execution system contracts
// * each block transaction
// * post-transaction-execution system contracts, withdrawals and block reward.
//
// It also contains a set of storage slot accesses for state which was accessed
// but not modified.  State accesses are not keyed by an index where they occurred.
type constructionAccountAccesses struct {
	// storageWrites contain mutated storage slots and their values.
	// It is indexed by storage slot -> access list index -> post-state value
	storageWrites map[common.Hash]map[uint16]common.Hash

	storageReads   map[common.Hash]struct{}
	balanceChanges map[uint16]*uint256.Int
	nonceChanges   map[uint16]uint64
	codeChanges    map[uint16]CodeChange
}

type ConstructionBlockAccessList map[common.Address]*constructionAccountAccesses

// constructionAccountAccess contains fields for an account which were modified
// during execution of the current access list index.
// It also accumulates a set of storage slots which were accessed but not
// modified during the execution of the current index.
type constructionAccountAccess struct {
	code    []byte
	nonce   *uint64
	balance *uint256.Int

	storageMutations map[common.Hash]common.Hash
	storageReads     map[common.Hash]struct{}
}

// Merge adds the accesses/mutations from other into the calling instance:
// c.stateMutations <- c.stateMutations \union other.stateMutations
func (c *constructionAccountAccess) Merge(other *constructionAccountAccess) {
	if other.code != nil {
		c.code = other.code
	}
	if other.nonce != nil {
		c.nonce = other.nonce
	}
	if other.balance != nil {
		c.balance = other.balance
	}
	if other.storageMutations != nil {
		if c.storageMutations == nil {
			c.storageMutations = make(map[common.Hash]common.Hash)
		}
		for key, val := range other.storageMutations {
			c.storageMutations[key] = val
			delete(c.storageReads, key)
		}
	}
	if other.storageReads != nil {
		if c.storageReads == nil {
			c.storageReads = make(map[common.Hash]struct{})
		}
		// TODO: if the state was mutated in the caller, don't add it to the caller's reads.
		// need to have a test case for this, verify it fails in the current state, and then fix this bug.
		for key, val := range other.storageReads {
			c.storageReads[key] = val
		}
	}
}

// MergeRevertedAccess merges an account's accesses from a reverted execution
// frame into the caller:
// * storage reads are merged into the caller
// * storage mutations are converted into reads and merged into the caller
// * account field mutations are discarded.  If an account
func (c *constructionAccountAccess) MergeRevertedAccess(other *constructionAccountAccess) {
	if other.storageMutations != nil {
		if c.storageReads == nil {
			c.storageReads = make(map[common.Hash]struct{})
		}
		for key, _ := range other.storageMutations {
			if _, ok := c.storageMutations[key]; ok {
				continue
			}
			c.storageReads[key] = struct{}{}
		}
	}
	if other.storageReads != nil {
		if c.storageReads == nil {
			c.storageReads = make(map[common.Hash]struct{})
		}
		for key := range other.storageReads {
			if _, ok := c.storageMutations[key]; ok {
				continue
			}
			c.storageReads[key] = struct{}{}
		}
	}
}

// StorageRead records a storage slot read
func (c *constructionAccountAccess) StorageRead(key common.Hash) {
	if c.storageReads == nil {
		c.storageReads = make(map[common.Hash]struct{})
	}
	if _, ok := c.storageMutations[key]; !ok {
		c.storageReads[key] = struct{}{}
	}
}

// StorageWrite records a storage slot write
func (c *constructionAccountAccess) StorageWrite(key, newVal common.Hash) {
	if c.storageMutations == nil {
		c.storageMutations = make(map[common.Hash]common.Hash)
	}
	c.storageMutations[key] = newVal
	// a key can be first read and later written, but it must only show up
	// in either read or write sets, not both.
	//
	// the caller should not
	// call StorageRead on a slot that was already written
	delete(c.storageReads, key)
}

func (c *constructionAccountAccess) BalanceChange(cur *uint256.Int) {
	c.balance = cur
}

func (c *constructionAccountAccess) CodeChange(cur []byte) {
	c.code = cur
}

func (c *constructionAccountAccess) NonceChange(cur uint64) {
	c.nonce = &cur
}

// AccessListBuilder is used to build an EIP-7928 block access list
type AccessListBuilder struct {
	FinalizedAccesses ConstructionBlockAccessList

	idxBuilder *idxAccessListBuilder

	lastFinalizedMutations *StateDiff
	lastFinalizedAccesses  StateAccesses
}

// NewAccessListBuilder instantiates an empty access list.
func NewAccessListBuilder() *AccessListBuilder {
	return &AccessListBuilder{
		make(map[common.Address]*constructionAccountAccesses),
		newIdxAccessListBuilder(),
		nil,
		nil,
	}
}

// Copy returns a deep copy of the access list.
func (c *AccessListBuilder) Copy() *AccessListBuilder {
	res := NewAccessListBuilder()
	for addr, aa := range c.FinalizedAccesses {
		var aaCopy constructionAccountAccesses

		slotWrites := make(map[common.Hash]map[uint16]common.Hash, len(aa.storageWrites))
		for key, m := range aa.storageWrites {
			slotWrites[key] = maps.Clone(m)
		}
		aaCopy.storageWrites = slotWrites
		aaCopy.storageReads = maps.Clone(aa.storageReads)

		balances := make(map[uint16]*uint256.Int, len(aa.balanceChanges))
		for index, balance := range aa.balanceChanges {
			balances[index] = balance.Clone()
		}
		aaCopy.balanceChanges = balances
		aaCopy.nonceChanges = maps.Clone(aa.nonceChanges)

		codeChangesCopy := make(map[uint16]CodeChange)
		for idx, codeChange := range aa.codeChanges {
			codeChangesCopy[idx] = CodeChange{
				TxIdx: idx,
				Code:  bytes.Clone(codeChange.Code),
			}
		}
		res.FinalizedAccesses[addr] = &aaCopy
	}
	return res
}

// FinalizedIdxChanges returns the state mutations and accesses recorded in the latest
// access list index that was finalized.
func (c *AccessListBuilder) FinalizedIdxChanges() (*StateDiff, StateAccesses) {
	return c.lastFinalizedMutations, c.lastFinalizedAccesses
}

// StateDiff contains state mutations occuring over one or more access list
// index.
type StateDiff struct {
	Mutations map[common.Address]*AccountMutations `json:"Mutations,omitempty"`
}

// StateAccesses contains a set of accounts/storage that were accessed during the
// execution of one or more access list indices.
type StateAccesses map[common.Address]map[common.Hash]struct{}

// Merge combines adds the accesses from other into s.
func (s *StateAccesses) Merge(other StateAccesses) {
	for addr, accesses := range other {
		if _, ok := (*s)[addr]; !ok {
			(*s)[addr] = make(map[common.Hash]struct{})
		}
		for slot := range accesses {
			(*s)[addr][slot] = struct{}{}
		}
	}
}

// accountIdxPrestate records the account prestate at a access list index
// for components which were modified at that index.
type accountIdxPrestate struct {
	balance *uint256.Int
	nonce   *uint64
	code    ContractCode
	storage map[common.Hash]common.Hash
}

// AccountMutations contains mutations that were made to an account across
// one or more access list indices.
type AccountMutations struct {
	Balance       *uint256.Int                `json:"Balance,omitempty"`
	Nonce         *uint64                     `json:"Nonce,omitempty"`
	Code          ContractCode                `json:"Code,omitempty"`
	StorageWrites map[common.Hash]common.Hash `json:"StorageWrites,omitempty"`
}

// String returns a human-readable JSON representation of the account mutations.
func (a *AccountMutations) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(a)
	return res.String()
}

// Eq returns whether the calling instance is equal to the provided one.
func (a *AccountMutations) Eq(other *AccountMutations) bool {
	if a.Balance != nil || other.Balance != nil {
		if a.Balance == nil || other.Balance == nil {
			return false
		}

		if !a.Balance.Eq(other.Balance) {
			return false
		}
	}

	if (len(a.Code) != 0 || len(other.Code) != 0) && !bytes.Equal(a.Code, other.Code) {
		return false
	}

	if a.Nonce != nil || other.Nonce != nil {
		if a.Nonce == nil || other.Nonce == nil {
			return false
		}

		if *a.Nonce != *other.Nonce {
			return false
		}
	}

	if a.StorageWrites != nil || other.StorageWrites != nil {
		if a.StorageWrites == nil || other.StorageWrites == nil {
			return false
		}

		if !maps.Equal(a.StorageWrites, other.StorageWrites) {
			return false
		}
	}
	return true
}

// Copy returns a deep-copy of the instance.
func (a *AccountMutations) Copy() *AccountMutations {
	res := &AccountMutations{
		nil,
		nil,
		nil,
		nil,
	}
	if a.Nonce != nil {
		res.Nonce = new(uint64)
		*res.Nonce = *a.Nonce
	}
	if a.Code != nil {
		res.Code = bytes.Clone(a.Code)
	}
	if a.Balance != nil {
		res.Balance = new(uint256.Int).Set(a.Balance)
	}
	if a.StorageWrites != nil {
		res.StorageWrites = maps.Clone(a.StorageWrites)
	}
	return res
}

// String returns the state diff as a formatted JSON string.
func (s *StateDiff) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(s)
	return res.String()
}

// Merge merges the state changes present in next into the caller.  After,
// the state of the caller is the aggregate diff through next.
func (s *StateDiff) Merge(next *StateDiff) {
	for account, diff := range next.Mutations {
		if mut, ok := s.Mutations[account]; ok {
			if diff.Balance != nil {
				mut.Balance = diff.Balance
			}
			if diff.Code != nil {
				mut.Code = diff.Code
			}
			if diff.Nonce != nil {
				mut.Nonce = diff.Nonce
			}
			if len(diff.StorageWrites) > 0 {
				if mut.StorageWrites == nil {
					mut.StorageWrites = maps.Clone(diff.StorageWrites)
				} else {
					for key, val := range diff.StorageWrites {
						mut.StorageWrites[key] = val
					}
				}
			}
		} else {
			s.Mutations[account] = diff.Copy()
		}
	}
}

// Copy returns a deep copy of the StateDiff
func (s *StateDiff) Copy() *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountMutations)}
	for addr, accountDiff := range s.Mutations {
		cpy := accountDiff.Copy()
		res.Mutations[addr] = cpy
	}
	return res
}

// Copy returns a deep copy of the access list
func (e BlockAccessList) Copy() (res BlockAccessList) {
	for _, accountAccess := range e {
		res = append(res, accountAccess.Copy())
	}
	return
}
