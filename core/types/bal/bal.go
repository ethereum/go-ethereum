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
	"github.com/ethereum/go-ethereum/params"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

/*
BAL Building rework

type BALBuilder
* hold state for the current execution context:
	* the state mutations that have already been finalized (previous completed txs)
    * state reads that have been finalized
	* the pending state reads/mutations of the current tx

pending state:
	* a stack (pushing/popping as new execution frames are entered/exited),
      each item is a map (address -> accountStateAndModifications{})
finalized state:
	* the ConstructionBlockAccessList type (sans the pending state stuff that I have added there

TLDR:
* break the pending state into its own struct, out of ConstructionBlockAccessList
* create a 'BALBuilder' type that encompasses the 'finalized' ConstructionBlockAccessList and pending state
* ensure that this new model fits nicely with the BAL validation code path
*/

// CodeChange contains the runtime bytecode deployed at an address and the
// transaction index where the deployment took place.
type CodeChange struct {
	TxIdx uint16
	Code  []byte `json:"code,omitempty"`
}

var IgnoredBALAddresses map[common.Address]struct{} = map[common.Address]struct{}{
	params.SystemAddress:                {},
	common.BytesToAddress([]byte{0x01}): {},
	common.BytesToAddress([]byte{0x02}): {},
	common.BytesToAddress([]byte{0x03}): {},
	common.BytesToAddress([]byte{0x04}): {},
	common.BytesToAddress([]byte{0x05}): {},
	common.BytesToAddress([]byte{0x06}): {},
	common.BytesToAddress([]byte{0x07}): {},
	common.BytesToAddress([]byte{0x08}): {},
	common.BytesToAddress([]byte{0x09}): {},
	common.BytesToAddress([]byte{0x0a}): {},
	common.BytesToAddress([]byte{0x0b}): {},
	common.BytesToAddress([]byte{0x0c}): {},
	common.BytesToAddress([]byte{0x0d}): {},
	common.BytesToAddress([]byte{0x0e}): {},
	common.BytesToAddress([]byte{0x0f}): {},
	common.BytesToAddress([]byte{0x10}): {},
	common.BytesToAddress([]byte{0x11}): {},
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
	// Storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{}

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint16]*uint256.Int

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint16]uint64

	CodeChanges map[uint16]CodeChange

	curIdx uint16
	// changes are first accumulated in here, and "flushed" to the maps above
	// via invoking Finalise.
	curIdxChanges constructionAccountAccess
}

func (c *ConstructionAccountAccesses) StorageRead(key common.Hash) {
	c.curIdxChanges.StorageRead(key)
}

func (c *ConstructionAccountAccesses) StorageWrite(key, prevVal, newVal common.Hash) {
	c.curIdxChanges.StorageWrite(key, prevVal, newVal)
}

func (c *ConstructionAccountAccesses) BalanceChange(prev, cur *uint256.Int) {
	c.curIdxChanges.BalanceChange(prev, cur)
}

func (c *ConstructionAccountAccesses) CodeChange(prev, cur []byte) {
	c.curIdxChanges.CodeChange(prev, cur)
}

func (c *ConstructionAccountAccesses) NonceChange(prev, cur uint64) {
	c.curIdxChanges.NonceChange(prev, cur)
}

// Called at tx boundaries
func (c *ConstructionAccountAccesses) Finalise() {
	c.curIdxChanges.Finalise()
	if c.curIdxChanges.code != nil {
		c.CodeChanges[c.curIdx] = CodeChange{c.curIdx, c.curIdxChanges.code}
	}
	if c.curIdxChanges.nonce != nil {
		c.NonceChanges[c.curIdx] = *c.curIdxChanges.nonce
	}
	if c.curIdxChanges.balance != nil {
		c.BalanceChanges[c.curIdx] = c.curIdxChanges.balance
	}
	if c.curIdxChanges.storageMutations != nil {
		for key, val := range c.curIdxChanges.storageMutations {
			if _, ok := c.StorageWrites[key]; !ok {
				c.StorageWrites[key] = make(map[uint16]common.Hash)
			}

			c.StorageWrites[key][c.curIdx] = val
		}
	}
	if c.curIdxChanges.storageReads != nil {
		for key := range c.StorageReads {
			c.StorageReads[key] = struct{}{}
		}
	}
	c.curIdxChanges = constructionAccountAccess{}
}

type constructionAccountAccess struct {
	// stores the tx-prestate values of any account/storage values which were modified
	//
	// TODO: it's a bit unfortunate that the prestate.StorageWrites is reused here to
	// represent the prestate values of keys which were written and it would be nice to
	// somehow make it clear that the field can contain both the mutated values or the
	// prestate depending on the context (or find a cleaner solution entirely, instead
	// of reusing the field here)
	prestate *AccountState

	code    []byte
	nonce   *uint64
	balance *uint256.Int

	storageMutations map[common.Hash]common.Hash
	storageReads     map[common.Hash]struct{}
}

func (c *constructionAccountAccess) StorageRead(key common.Hash) {
	if c.storageReads == nil {
		c.storageReads = make(map[common.Hash]struct{})
	}
	c.storageReads[key] = struct{}{}
}

func (c *constructionAccountAccess) StorageWrite(key, prevVal, newVal common.Hash) {
	if c.prestate.StorageWrites == nil {
		c.prestate.StorageWrites = make(map[common.Hash]common.Hash)
	}
	if _, ok := c.prestate.StorageWrites[key]; !ok {
		c.prestate.StorageWrites[key] = prevVal
	}
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

func (c *constructionAccountAccess) BalanceChange(prev, cur *uint256.Int) {
	if c.prestate.Balance == nil {
		c.prestate.Balance = prev
	}
	c.balance = cur
}

func (c *constructionAccountAccess) CodeChange(prev, cur []byte) {
	if c.prestate.Code == nil {
		c.prestate.Code = prev
	}
	c.code = cur
}

func (c *constructionAccountAccess) NonceChange(prev, cur uint64) {
	if c.prestate.Nonce == nil {
		c.prestate.Nonce = &prev
	}
	c.nonce = &cur
}

// Finalise clears any fields which had no net mutations compared to their
// prestate values.
func (c *constructionAccountAccess) Finalise() {
	if c.nonce != nil && *c.prestate.Nonce == *c.nonce {
		c.nonce = nil
	}
	if c.balance != nil && c.prestate.Balance.Eq(c.balance) {
		c.balance = nil
	}
	if c.code != nil && bytes.Equal(c.code, c.prestate.Code) {
		c.code = nil
	}
	if c.storageMutations != nil {
		for key, val := range c.storageMutations {
			if c.prestate.StorageWrites[key] == val {
				delete(c.storageMutations, key)
				c.storageReads[key] = struct{}{}
			}
		}
	}
}

// NewConstructionAccountAccesses initializes the account access object.
func NewConstructionAccountAccesses(idx uint16) *ConstructionAccountAccesses {
	return &ConstructionAccountAccesses{
		StorageWrites:  make(map[common.Hash]map[uint16]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint16]*uint256.Int),
		NonceChanges:   make(map[uint16]uint64),
		CodeChanges:    make(map[uint16]CodeChange),
		curIdx:         idx,
	}
}

// ConstructionBlockAccessList contains post-block modified state and some state accessed
// in execution (account addresses and storage keys).
type ConstructionBlockAccessList struct {
	Accounts map[common.Address]*ConstructionAccountAccesses
	curIdx   uint16
}

// NewConstructionBlockAccessList instantiates an empty access list.
func NewConstructionBlockAccessList() *ConstructionBlockAccessList {
	return &ConstructionBlockAccessList{
		make(map[common.Address]*ConstructionAccountAccesses),
		0,
	}
}

// EnterCallScope should be invoked when the EVM execution enters a new call fram
func (c *ConstructionBlockAccessList) EnterCallScope() {
}

// ExitCallScope is invoked when the EVM execution exits a call frame.
// If the call frame reverted, all state changes that occurred within that
// scope (or any nested calls) are discarded.  Storage reads performed in the
// nested scope(s) are retained and any storage writes that occurred are denoted
// as storage reads in the BAL.
func (c *ConstructionBlockAccessList) ExitCallScope(reverted bool) {
	if reverted {

	} else {

	}
}

func (c *ConstructionBlockAccessList) DiffAt(i int) *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountState)}

	idx := uint16(i)

	for addr, account := range c.Accounts {
		accountState := &AccountState{}
		if balance, ok := account.BalanceChanges[idx]; ok {
			accountState.Balance = balance
		}
		if nonce, ok := account.NonceChanges[idx]; ok {
			accountState.Nonce = &nonce
		}
		if code, ok := account.CodeChanges[idx]; ok {
			accountState.Code = code.Code
		}

		storageWrites := make(map[common.Hash]common.Hash)
		for slot, writes := range account.StorageWrites {
			if val, ok := writes[idx]; ok {
				storageWrites[slot] = val
			}
		}
		if len(storageWrites) > 0 {
			accountState.StorageWrites = storageWrites
		}

		if !accountState.Empty() {
			res.Mutations[addr] = accountState
		}
	}
	return res
}

func (c *ConstructionBlockAccessList) StateAccesses() StateAccesses {
	res := make(StateAccesses)
	for addr, acct := range c.Accounts {
		if len(acct.StorageReads) > 0 {
			res[addr] = acct.StorageReads
			continue
		}
		if len(acct.NonceChanges) == 0 && len(acct.BalanceChanges) == 0 && len(acct.StorageWrites) == 0 && len(acct.CodeChanges) == 0 {
			res[addr] = make(map[common.Hash]struct{})
		}
	}
	return res
}

// AccountRead records the address of an account that has been read during execution.
func (c *ConstructionBlockAccessList) AccountRead(addr common.Address) {
	if _, ok := c.Accounts[addr]; !ok {
		c.Accounts[addr] = NewConstructionAccountAccesses(c.curIdx)
	}
}

// StorageRead records a storage key read during execution.
func (c *ConstructionBlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccesses(c.curIdx)
	}
	if _, ok := c.Accounts[address].StorageWrites[key]; ok {
		return
	}
	c.Accounts[address].StorageRead(key)
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (c *ConstructionBlockAccessList) StorageWrite(address common.Address, key, prev, cur common.Hash) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccesses(c.curIdx)
	}
	c.Accounts[address].StorageWrite(key, prev, cur)
}

// CodeChange records the code of a newly-created contract.
func (c *ConstructionBlockAccessList) CodeChange(address common.Address, txIndex uint16, code []byte) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccesses(c.curIdx)
	}
	c.Accounts[address].CodeChanges[txIndex] = CodeChange{
		TxIdx: txIndex,
		Code:  bytes.Clone(code),
	}
}

// NonceChange records tx post-state nonce of any contract-like accounts whose
// nonce was incremented.
func (c *ConstructionBlockAccessList) NonceChange(address common.Address, pre uint64, cur uint64) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccesses(c.curIdx)
	}
	c.Accounts[address].NonceChange(pre, cur)
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (c *ConstructionBlockAccessList) BalanceChange(address common.Address, old, new *uint256.Int) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccesses(c.curIdx)
	}
	c.Accounts[address].BalanceChange(old, new)
}

func (c *ConstructionBlockAccessList) Finalise() {
	for _, account := range c.Accounts {
		account.Finalise()
	}
	c.curIdx++
}

func mergeStorageWrites(cur, next map[common.Hash]map[uint16]common.Hash) map[common.Hash]map[uint16]common.Hash {
	for slot, _ := range next {
		if _, ok := cur[slot]; !ok {
			cur[slot] = next[slot]
			continue
		}

		for idx, val := range next[slot] {
			cur[slot][idx] = val
		}
	}

	return cur
}

// MergeReads combines the account/storage reads from a completed EVM execution scope
// into the parent calling scope's access list.
// It is intended to be called when the child execution scope terminates in a revert
// which means that only the state reads performed by that execution should be reported
// in the BAL.
func (c *ConstructionBlockAccessList) MergeReads(childScope *ConstructionBlockAccessList) {
	for addr, accountAccess := range childScope.Accounts {
		if _, ok := c.Accounts[addr]; !ok {
			c.Accounts[addr] = NewConstructionAccountAccesses(c.curIdx)
			c.Accounts[addr].StorageReads = accountAccess.StorageReads
			continue
		}

		for storageRead, _ := range childScope.Accounts[addr].StorageReads {
			c.Accounts[addr].StorageReads[storageRead] = struct{}{}
		}
	}
}

// Merge combines the state changes from a nested execution with the parent context
// It is meant to be invoked after an EVM call completes (without reverting).
func (c *ConstructionBlockAccessList) Merge(childScope *ConstructionBlockAccessList) {
	for addr, accountAccess := range childScope.Accounts {
		if _, ok := c.Accounts[addr]; !ok {
			c.Accounts[addr] = accountAccess
			continue
		}

		// copy the entries from 'next' into 'c' overwriting 'c' entries with
		// 'next entries when the bal index matches.
		c.Accounts[addr].StorageWrites = mergeStorageWrites(c.Accounts[addr].StorageWrites, childScope.Accounts[addr].StorageWrites)

		for storageRead, _ := range childScope.Accounts[addr].StorageReads {
			c.Accounts[addr].StorageReads[storageRead] = struct{}{}
		}
		for idx, nonce := range childScope.Accounts[addr].NonceChanges {
			c.Accounts[addr].NonceChanges[idx] = nonce
		}
		for idx, code := range childScope.Accounts[addr].CodeChanges {
			c.Accounts[addr].CodeChanges[idx] = code
		}
		for idx, balance := range childScope.Accounts[addr].BalanceChanges {
			c.Accounts[addr].BalanceChanges[idx] = balance
		}
	}
}

// Copy returns a deep copy of the access list.
func (c *ConstructionBlockAccessList) Copy() *ConstructionBlockAccessList {
	res := NewConstructionBlockAccessList()
	for addr, aa := range c.Accounts {
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
		res.Accounts[addr] = &aaCopy
	}
	return res
}

type StateDiff struct {
	Mutations map[common.Address]*AccountState `json:"Mutations,omitempty"`
}

type StateAccesses map[common.Address]map[common.Hash]struct{}

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

type AccountState struct {
	Balance       *uint256.Int                `json:"Balance,omitempty"`
	Nonce         *uint64                     `json:"Nonce,omitempty"`
	Code          ContractCode                `json:"Code,omitempty"`
	StorageWrites map[common.Hash]common.Hash `json:"StorageWrites,omitempty"`
}

func (a *AccountState) Empty() bool {
	return a.Balance == nil && a.Nonce == nil && a.Code == nil && len(a.StorageWrites) == 0
}

func (a *AccountState) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(a)
	return res.String()
}

// Merge the changes of a future AccountState into the caller, resulting in the
// combined state changes through next.
func (a *AccountState) Merge(next *AccountState) {
	if next.Balance != nil {
		a.Balance = next.Balance
	}
	if next.Nonce != nil {
		a.Nonce = next.Nonce
	}
	if next.Code != nil {
		a.Code = next.Code
	}
	if next.StorageWrites != nil {
		if a.StorageWrites == nil {
			a.StorageWrites = maps.Clone(next.StorageWrites)
		} else {
			for key, val := range next.StorageWrites {
				a.StorageWrites[key] = val
			}
		}
	}
}

func NewEmptyAccountState() *AccountState {
	return &AccountState{
		nil,
		nil,
		nil,
		nil,
	}
}

func (a *AccountState) Eq(other *AccountState) bool {
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

func (a *AccountState) Copy() *AccountState {
	res := NewEmptyAccountState()
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

func (s *StateDiff) Copy() *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountState)}
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
