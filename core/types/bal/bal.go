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
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"maps"
)

// idxAccessListBuilder is responsible for producing the state accesses and
// reads recorded within the scope of a single index in the access list.
type idxAccessListBuilder struct {
	// stores the previous values of any account data that was modified in the
	// current index.
	prestates map[common.Address]*partialAccountState

	// a stack which maintains a set of state mutations/reads for each EVM
	// execution frame.
	//
	// <boilerplate for description about how the execution stack mechanics work when call frames end with/without erring/reverting>
	// TODO: how does this construction handle EVM errors which terminate execution of a transaction entirely.
	accessesStack []map[common.Address]*constructionAccountAccess
}

func newAccessListBuilder() *idxAccessListBuilder {
	return &idxAccessListBuilder{
		make(map[common.Address]*partialAccountState),
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
		c.prestates[address] = &AccountMutations{}
	}
	if c.prestates[address].StorageWrites == nil {
		c.prestates[address].StorageWrites = make(map[common.Hash]common.Hash)
	}
	if _, ok := c.prestates[address].StorageWrites[key]; !ok {
		c.prestates[address].StorageWrites[key] = prevVal
	}

	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.StorageWrite(key, prevVal, newVal)
}

func (c *idxAccessListBuilder) balanceChange(address common.Address, prev, cur *uint256.Int) {
	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &AccountMutations{}
	}
	if c.prestates[address].Balance == nil {
		c.prestates[address].Balance = prev
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
		c.prestates[address] = &AccountMutations{}
	}
	if c.prestates[address].Code == nil {
		c.prestates[address].Code = prev
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]

	acctAccesses.CodeChange(cur)
}

func (c *idxAccessListBuilder) selfDestruct(address common.Address) {
	// convert all the account storage writes to reads, preserve the existing reads
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		// TODO: figure out exactly which situations cause this case
		// it has to do with an account becoming empty and deleted
		// but why was it created as a stateObject without also having
		// any access/modification events on it?
		return
	}
	access := c.accessesStack[len(c.accessesStack)-1][address]
	for key, _ := range access.storageMutations {
		if access.storageReads == nil {
			access.storageReads = make(map[common.Hash]struct{})
		}
		access.storageReads[key] = struct{}{}
	}

	access.storageMutations = nil
	/*
		access.nonce = nil
		// TODO: should this be set to zero?  the semantics are that nil means unmodified since the prestate of the block.
		access.balance = nil
		access.code = nil
	*/
}

func (c *idxAccessListBuilder) nonceChange(address common.Address, prev, cur uint64) {
	if _, ok := c.prestates[address]; !ok {
		c.prestates[address] = &AccountMutations{}
	}
	if c.prestates[address].Nonce == nil {
		c.prestates[address].Nonce = &prev
	}
	if _, ok := c.accessesStack[len(c.accessesStack)-1][address]; !ok {
		c.accessesStack[len(c.accessesStack)-1][address] = &constructionAccountAccess{}
	}
	acctAccesses := c.accessesStack[len(c.accessesStack)-1][address]
	acctAccesses.NonceChange(cur)
}

func (c *idxAccessListBuilder) enterScope() {
	c.accessesStack = append(c.accessesStack, make(map[common.Address]*constructionAccountAccess))
}

func (c *idxAccessListBuilder) exitScope(reverted bool) {
	// all storage writes in the child scope are converted into reads
	// if there were no storage writes, the account is reported in the BAL as a read (if it wasn't already in the BAL and/or mutated previously)
	childAccessList := c.accessesStack[len(c.accessesStack)-1]
	parentAccessList := c.accessesStack[len(c.accessesStack)-2]

	for addr, childAccess := range childAccessList {
		if _, ok := parentAccessList[addr]; ok {
		} else {
			parentAccessList[addr] = &constructionAccountAccess{}
		}
		if reverted {
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
		// remove any mutations from the access list with no net difference vs the tx prestate value
		if access.nonce != nil && *a.prestates[addr].Nonce == *access.nonce {
			access.nonce = nil
		}
		if access.balance != nil && a.prestates[addr].Balance.Eq(access.balance) {
			access.balance = nil
		}

		if access.code != nil && bytes.Equal(access.code, a.prestates[addr].Code) {
			access.code = nil
		}
		if access.storageMutations != nil {
			for key, val := range access.storageMutations {
				if a.prestates[addr].StorageWrites[key] == val {
					delete(access.storageMutations, key)
					access.storageReads[key] = struct{}{}
				}
			}
			if len(access.storageMutations) == 0 {
				access.storageMutations = nil
			}
		}

		// if the account has no net mutations against the tx prestate, only include
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

func (c *BlockAccessListBuilder) FinalisePendingChanges(idx uint16) {
	diff, accesses := c.idxBuilder.finalise()
	c.idxBuilder = newAccessListBuilder()

	for addr, stateDiff := range diff.Mutations {
		acctChanges, ok := c.FinalizedAccesses[addr]
		if !ok {
			acctChanges = &ConstructionAccountAccesses{}
			c.FinalizedAccesses[addr] = acctChanges
		}

		if stateDiff.Nonce != nil {
			if acctChanges.NonceChanges == nil {
				acctChanges.NonceChanges = make(map[uint16]uint64)
			}
			acctChanges.NonceChanges[idx] = *stateDiff.Nonce
		}
		if stateDiff.Balance != nil {
			if acctChanges.BalanceChanges == nil {
				acctChanges.BalanceChanges = make(map[uint16]*uint256.Int)
			}
			acctChanges.BalanceChanges[idx] = stateDiff.Balance
		}
		if stateDiff.Code != nil {
			if acctChanges.CodeChanges == nil {
				acctChanges.CodeChanges = make(map[uint16]CodeChange)
			}
			acctChanges.CodeChanges[idx] = CodeChange{idx, stateDiff.Code}
		}
		if stateDiff.StorageWrites != nil {
			if acctChanges.StorageWrites == nil {
				acctChanges.StorageWrites = make(map[common.Hash]map[uint16]common.Hash)
			}
			for key, val := range stateDiff.StorageWrites {
				if _, ok := acctChanges.StorageWrites[key]; !ok {
					acctChanges.StorageWrites[key] = make(map[uint16]common.Hash)
				}
				acctChanges.StorageWrites[key][idx] = val

				// TODO: investigate why commenting out the check here, and the corresponding
				// check under accesses causes GeneralStateTests blockchain tests to fail.
				// They should only contain one tx per test.
				//
				// key could have been read in a previous tx, delete it from the read set here
				if _, ok := acctChanges.StorageReads[key]; ok {
					delete(acctChanges.StorageReads, key)
				}
			}
		}
	}
	for addr, stateAccesses := range accesses {
		acctAccess, ok := c.FinalizedAccesses[addr]
		if !ok {
			acctAccess = &ConstructionAccountAccesses{}
			c.FinalizedAccesses[addr] = acctAccess
		}

		for key := range stateAccesses {
			// if key was written in a previous tx, but only read in this one:
			// don't include it in the storage reads set
			if _, ok := acctAccess.StorageWrites[key]; ok {
				continue
			}
			if acctAccess.StorageReads == nil {
				acctAccess.StorageReads = make(map[common.Hash]struct{})
			}
			acctAccess.StorageReads[key] = struct{}{}
		}
	}
	c.lastFinalizedMutations = diff
	c.lastFinalizedAccesses = accesses
}

func (c *BlockAccessListBuilder) StorageRead(address common.Address, key common.Hash) {
	c.idxBuilder.storageRead(address, key)
}
func (c *BlockAccessListBuilder) AccountRead(address common.Address) {
	c.idxBuilder.accountRead(address)
}
func (c *BlockAccessListBuilder) StorageWrite(address common.Address, key, prevVal, newVal common.Hash) {
	c.idxBuilder.storageWrite(address, key, prevVal, newVal)
}
func (c *BlockAccessListBuilder) BalanceChange(address common.Address, prev, cur *uint256.Int) {
	c.idxBuilder.balanceChange(address, prev, cur)
}
func (c *BlockAccessListBuilder) NonceChange(address common.Address, prev, cur uint64) {
	c.idxBuilder.nonceChange(address, prev, cur)
}
func (c *BlockAccessListBuilder) CodeChange(address common.Address, prev, cur []byte) {
	c.idxBuilder.codeChange(address, prev, cur)
}
func (c *BlockAccessListBuilder) SelfDestruct(address common.Address) {
	c.idxBuilder.selfDestruct(address)
}

func (c *BlockAccessListBuilder) EnterScope() {
	c.idxBuilder.enterScope()
}
func (c *BlockAccessListBuilder) ExitScope(reverted bool) {
	c.idxBuilder.exitScope(reverted)
}

// TODO: the BalReader Validation method should accept the computed values as
// a index/StateDiff/StateAccesses trio.

// BAL tracer maintains a BlockAccessListBuilder.
// For each BAL index, it instantiates an idxAccessListBuilder and
// appends the result to the access list where appropriate

// ---- below is the actual code written before my idea sketch above ----

// CodeChange contains the runtime bytecode deployed at an address and the
// transaction index where the deployment took place.
type CodeChange struct {
	TxIdx uint16
	Code  []byte `json:"code,omitempty"`
}

// TODO: make use of this
var IgnoredBALAddresses = map[common.Address]struct{}{
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
}

type constructionAccountAccess struct {
	code    []byte
	nonce   *uint64
	balance *uint256.Int

	storageMutations map[common.Hash]common.Hash
	storageReads     map[common.Hash]struct{}
}

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
	if _, ok := c.storageMutations[key]; ok {
		panic("FUCK")
	}
	// TODO: if a key is written in tx A, and later on read in tx B, it shoulnd't be in the read set.
	// ^ same for account.
	c.storageReads[key] = struct{}{}
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

// BlockAccessListBuilder is used to build an EIP-7928 block access list
type BlockAccessListBuilder struct {
	FinalizedAccesses map[common.Address]*ConstructionAccountAccesses

	idxBuilder *idxAccessListBuilder

	lastFinalizedMutations *StateDiff
	lastFinalizedAccesses  StateAccesses
}

// NewConstructionBlockAccessList instantiates an empty access list.
func NewConstructionBlockAccessList() *BlockAccessListBuilder {
	return &BlockAccessListBuilder{
		make(map[common.Address]*ConstructionAccountAccesses),
		newAccessListBuilder(),
		nil,
		nil,
	}
}

// Copy returns a deep copy of the access list.
func (c *BlockAccessListBuilder) Copy() *BlockAccessListBuilder {
	res := NewConstructionBlockAccessList()
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
func (c *BlockAccessListBuilder) FinalizedIdxChanges() (*StateDiff, StateAccesses) {
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

type partialAccountState struct {
	balance *uint256.Int                `json:"Balance,omitempty"`
	nonce   *uint64                     `json:"Nonce,omitempty"`
	code    ContractCode                `json:"Code,omitempty"`
	storage map[common.Hash]common.Hash `json:"StorageWrites,omitempty"`
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
