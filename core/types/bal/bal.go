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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"log/slog"
	"maps"
	"slices"
)

// idxAccessListBuilder is responsible for producing the state accesses and
// reads recorded within the scope of a single index in the access list.
type idxAccessListBuilder struct {
	// stores the previous values of any account data that was modified in the
	// current index.
	prestates map[common.Address]*accountIdxPrestate

	// a stack which maintains a set of state mutations/reads for each EVM
	// execution frame.  Entering a frame appends an intermediate access list
	// and terminating a frame merges the accesses/modifications into the
	// intermediate access list of the calling frame.
	accessesStack []map[common.Address]*constructionAccountAccess

	// context logger for instrumenting debug logging
	logger *slog.Logger
}

func newAccessListBuilder(logger *slog.Logger) *idxAccessListBuilder {
	return &idxAccessListBuilder{
		make(map[common.Address]*accountIdxPrestate),
		[]map[common.Address]*constructionAccountAccess{
			make(map[common.Address]*constructionAccountAccess),
		},
		logger,
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
	acctAccesses.StorageWrite(key, prevVal, newVal)
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
	// auth unset and selfdestruct pass code change as 'nil'
	// however, internally in the access list accumulation of state changes,
	// a nil field on an account means that it was never modified in the block.
	if cur == nil {
		cur = []byte{}
	}

	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &accountIdxPrestate{}
	}
	if c.prestates[address].code == nil {
		if prev == nil {
			prev = []byte{}
		}
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
// of the current call are retained in the block access list as state reads.
func (c *idxAccessListBuilder) selfDestruct(address common.Address) {
	access := c.accessesStack[len(c.accessesStack)-1][address]
	if len(access.storageMutations) != 0 && access.storageReads == nil {
		access.storageReads = make(map[common.Hash]struct{})
	}
	for key, _ := range access.storageMutations {
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

// enterScope is called after a new EVM call frame has been entered.
func (c *idxAccessListBuilder) enterScope() {
	c.accessesStack = append(c.accessesStack, make(map[common.Address]*constructionAccountAccess))
}

// exitScope is called after an EVM call scope terminates.  If the call scope
// terminates with an error:
// * the scope's state accesses are added to the calling scope's access list
// * mutated accounts/storage are added into the calling scope's access list as state accesses
func (c *idxAccessListBuilder) exitScope(evmErr bool) {
	childAccessList := c.accessesStack[len(c.accessesStack)-1]
	parentAccessList := c.accessesStack[len(c.accessesStack)-2]

	for addr, childAccess := range childAccessList {
		if _, ok := parentAccessList[addr]; ok {
		} else {
			parentAccessList[addr] = &constructionAccountAccess{}
		}
		if evmErr {
			// all storage writes in the child scope are converted into reads
			// if there were no storage writes, the account is reported in the BAL as a read (if it wasn't already in the BAL and/or mutated previously)
			parentAccessList[addr].MergeReads(childAccess)
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
		// remove any reported mutations from the access list with no net difference vs the index prestate value
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

		// if the account has no net mutations against the index prestate, only include
		// it in the state read set
		if len(access.code) == 0 && access.nonce == nil && access.balance == nil && len(access.storageMutations) == 0 {
			stateAccesses[addr] = make(map[common.Hash]struct{})
			if access.storageReads != nil {
				stateAccesses[addr] = access.storageReads
			}
			continue
		}

		stateAccesses[addr] = access.storageReads
		diff.Mutations[addr] = &AccountMutations{
			Balance:       access.balance,
			Nonce:         access.nonce,
			Code:          access.code,
			StorageWrites: access.storageMutations,
		}
	}

	return diff, stateAccesses
}

func (c *AccessListBuilder) EnterTx(txHash common.Hash) {
	c.idxBuilder = newAccessListBuilder(slog.New(slog.DiscardHandler))
}

// FinaliseIdxChanges records all pending state mutations/accesses in the
// access list at the given index.  The set of pending state mutations/accesse are
// then emptied.
func (c *AccessListBuilder) FinaliseIdxChanges(idx uint16) {
	pendingDiff, pendingAccesses := c.idxBuilder.finalise()
	c.idxBuilder = newAccessListBuilder(slog.New(slog.DiscardHandler))

	for addr, pendingAcctDiff := range pendingDiff.Mutations {
		finalizedAcctChanges, ok := c.FinalizedAccesses[addr]
		if !ok {
			finalizedAcctChanges = &ConstructionAccountAccesses{}
			c.FinalizedAccesses[addr] = finalizedAcctChanges
		}

		if pendingAcctDiff.Nonce != nil {
			if finalizedAcctChanges.NonceChanges == nil {
				finalizedAcctChanges.NonceChanges = make(map[uint16]uint64)
			}
			finalizedAcctChanges.NonceChanges[idx] = *pendingAcctDiff.Nonce
		}
		if pendingAcctDiff.Balance != nil {
			if finalizedAcctChanges.BalanceChanges == nil {
				finalizedAcctChanges.BalanceChanges = make(map[uint16]*uint256.Int)
			}
			finalizedAcctChanges.BalanceChanges[idx] = pendingAcctDiff.Balance
		}
		if pendingAcctDiff.Code != nil {
			if finalizedAcctChanges.CodeChanges == nil {
				finalizedAcctChanges.CodeChanges = make(map[uint16]CodeChange)
			}
			finalizedAcctChanges.CodeChanges[idx] = CodeChange{idx, pendingAcctDiff.Code}
		}
		if pendingAcctDiff.StorageWrites != nil {
			if finalizedAcctChanges.StorageWrites == nil {
				finalizedAcctChanges.StorageWrites = make(map[common.Hash]map[uint16]common.Hash)
			}
			for key, val := range pendingAcctDiff.StorageWrites {
				if _, ok := finalizedAcctChanges.StorageWrites[key]; !ok {
					finalizedAcctChanges.StorageWrites[key] = make(map[uint16]common.Hash)
				}
				finalizedAcctChanges.StorageWrites[key][idx] = val

				// if any of the newly-written storage slots were previously
				// accessed, they must be removed from the accessed state set.

				// TODO: commenting this 'if' results in no test failures.
				// double-check that this edge-case was fixed by a future
				// release of the eest BAL tests.
				if _, ok := finalizedAcctChanges.StorageReads[key]; ok {
					delete(finalizedAcctChanges.StorageReads, key)
				}
			}
		}
	}
	// record pending accesses in the BAL access set unless they were
	// already written in a previous index
	for addr, pendingAccountAccesses := range pendingAccesses {
		finalizedAcctAccesses, ok := c.FinalizedAccesses[addr]
		if !ok {
			finalizedAcctAccesses = &ConstructionAccountAccesses{}
			c.FinalizedAccesses[addr] = finalizedAcctAccesses
		}

		for key := range pendingAccountAccesses {
			if _, ok := finalizedAcctAccesses.StorageWrites[key]; ok {
				continue
			}
			if finalizedAcctAccesses.StorageReads == nil {
				finalizedAcctAccesses.StorageReads = make(map[common.Hash]struct{})
			}
			finalizedAcctAccesses.StorageReads[key] = struct{}{}
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

// ConstructionAccountAccesses contains post-block account state for mutations as well as
// all storage keys that were read during execution.  It is used when building block
// access list during execution.
type ConstructionAccountAccesses struct {
	// StorageWrites is the post-state values of an account's storage slots
	// that were modified in a block, keyed by the slot key and the tx index
	// where the modification occurred.
	StorageWrites map[common.Hash]map[uint16]common.Hash

	// StorageReads is the set of slot keys that were accessed during block
	// execution.
	//
	// storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{}

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint16]*uint256.Int

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint16]uint64

	CodeChanges map[uint16]CodeChange
}

// constructionAccountAccess contains fields for an account which were modified
// during execution of the current access list index.
// It also accumulates a set of storage slots which were accessed but not
// modified.
type constructionAccountAccess struct {
	code    []byte
	nonce   *uint64
	balance *uint256.Int

	storageMutations map[common.Hash]common.Hash
	storageReads     map[common.Hash]struct{}
}

// Merge adds the accesses/mutations from other into the calling instance. If
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

// MergeReads merges accesses from a reverted execution from:
// * any reads/writes from the reverted frame which weren't mutated
// in the current frame, are merged into the current frame as reads.
func (c *constructionAccountAccess) MergeReads(other *constructionAccountAccess) {
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

func (c *constructionAccountAccess) StorageRead(key common.Hash) {
	if c.storageReads == nil {
		c.storageReads = make(map[common.Hash]struct{})
	}
	if _, ok := c.storageMutations[key]; !ok {
		c.storageReads[key] = struct{}{}
	}
}

func (c *constructionAccountAccess) StorageWrite(key, prevVal, newVal common.Hash) {
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

type ConstructionBlockAccessList map[common.Address]*ConstructionAccountAccesses

// AccessListBuilder is used to build an EIP-7928 block access list
type AccessListBuilder struct {
	FinalizedAccesses ConstructionBlockAccessList

	idxBuilder *idxAccessListBuilder

	lastFinalizedMutations *StateDiff
	lastFinalizedAccesses  StateAccesses
	logger                 *slog.Logger
}

// NewAccessListBuilder instantiates an empty access list.
func NewAccessListBuilder() *AccessListBuilder {
	logger := slog.New(slog.DiscardHandler)
	return &AccessListBuilder{
		make(map[common.Address]*ConstructionAccountAccesses),
		newAccessListBuilder(logger),
		nil,
		nil,
		logger,
	}
}

// Copy returns a deep copy of the access list.
func (c *AccessListBuilder) Copy() *AccessListBuilder {
	res := NewAccessListBuilder()
	for addr, aa := range c.FinalizedAccesses {
		var aaCopy ConstructionAccountAccesses

		slotWrites := make(map[common.Hash]map[uint16]common.Hash, len(aa.StorageWrites))
		for key, m := range aa.StorageWrites {
			slotWrites[key] = maps.Clone(m)
		}
		aaCopy.StorageWrites = slotWrites
		aaCopy.StorageReads = maps.Clone(aa.StorageReads)

		balances := make(map[uint16]*uint256.Int, len(aa.BalanceChanges))
		for index, balance := range aa.BalanceChanges {
			balances[index] = balance.Clone()
		}
		aaCopy.BalanceChanges = balances
		aaCopy.NonceChanges = maps.Clone(aa.NonceChanges)

		codeChangesCopy := make(map[uint16]CodeChange)
		for idx, codeChange := range aa.CodeChanges {
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

func (a *AccountMutations) LogDiff(addr common.Address, other *AccountMutations) {
	var diff []interface{}

	if a.Balance != nil || other.Balance != nil {
		if a.Balance == nil || other.Balance == nil || !a.Balance.Eq(other.Balance) {
			diff = append(diff, "local balance", a.Balance, "remote balance", other.Balance)
		}
	}
	if (len(a.Code) != 0 || len(other.Code) != 0) && !bytes.Equal(a.Code, other.Code) {
		diff = append(diff, "local code", a.Code, "remote code", other.Code)
	}
	if a.Nonce != nil || other.Nonce != nil {
		if a.Nonce == nil || other.Nonce == nil || *a.Nonce != *other.Nonce {
			diff = append(diff, "local nonce", a.Nonce, "remote nonce", other.Nonce)
		}
	}

	if a.StorageWrites != nil || other.StorageWrites != nil {
		if !maps.Equal(a.StorageWrites, other.StorageWrites) {
			union := make(map[common.Hash]struct{})
			for slot, _ := range a.StorageWrites {
				union[slot] = struct{}{}
			}
			for slot, _ := range other.StorageWrites {
				union[slot] = struct{}{}
			}

			orderedKeys := slices.SortedFunc(maps.Keys(union), func(hash common.Hash, hash2 common.Hash) int {
				return bytes.Compare(hash[:], hash2[:])
			})

			for _, key := range orderedKeys {
				aVal, inA := a.StorageWrites[key]
				otherVal, inOther := other.StorageWrites[key]

				if (inA && !inOther) || (!inA && inOther) || !bytes.Equal(aVal[:], otherVal[:]) {
					diff = append(diff, fmt.Sprintf("storage-local-%x", key), aVal)
					diff = append(diff, fmt.Sprintf("storage-remote-%x", key), otherVal)
				}
			}
		}
	}

	if len(diff) > 0 {
		log.Error(fmt.Sprintf("diff between remote/local BAL for address %x", addr), diff...)
	}
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
