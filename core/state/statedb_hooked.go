// Copyright 2024 The go-ethereum Authors
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
	"bytes"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// hookedStateDB represents a statedb which emits calls to tracing-hooks
// on state operations.
type hookedStateDB struct {
	inner *StateDB
	hooks *tracing.Hooks
}

// NewHookedState wraps the given stateDb with the given hooks
func NewHookedState(stateDb *StateDB, hooks *tracing.Hooks) *hookedStateDB {
	s := &hookedStateDB{stateDb, hooks}
	if s.hooks == nil {
		s.hooks = new(tracing.Hooks)
	}
	return s
}

func (s *hookedStateDB) CreateAccount(addr common.Address) {
	s.inner.CreateAccount(addr)
}

func (s *hookedStateDB) CreateContract(addr common.Address) {
	s.inner.CreateContract(addr)
}

func (s *hookedStateDB) IsNewContract(addr common.Address) bool {
	return s.inner.IsNewContract(addr)
}

func (s *hookedStateDB) GetBalance(addr common.Address) *uint256.Int {
	return s.inner.GetBalance(addr)
}

func (s *hookedStateDB) GetNonce(addr common.Address) uint64 {
	return s.inner.GetNonce(addr)
}

func (s *hookedStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.inner.GetCodeHash(addr)
}

func (s *hookedStateDB) GetCode(addr common.Address) []byte {
	return s.inner.GetCode(addr)
}

func (s *hookedStateDB) GetCodeSize(addr common.Address) int {
	return s.inner.GetCodeSize(addr)
}

func (s *hookedStateDB) AddRefund(u uint64) {
	s.inner.AddRefund(u)
}

func (s *hookedStateDB) SubRefund(u uint64) {
	s.inner.SubRefund(u)
}

func (s *hookedStateDB) GetRefund() uint64 {
	return s.inner.GetRefund()
}

func (s *hookedStateDB) GetStateAndCommittedState(addr common.Address, hash common.Hash) (common.Hash, common.Hash) {
	return s.inner.GetStateAndCommittedState(addr, hash)
}

func (s *hookedStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	return s.inner.GetState(addr, hash)
}

func (s *hookedStateDB) GetStorageRoot(addr common.Address) common.Hash {
	return s.inner.GetStorageRoot(addr)
}

func (s *hookedStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.inner.GetTransientState(addr, key)
}

func (s *hookedStateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	s.inner.SetTransientState(addr, key, value)
}

func (s *hookedStateDB) HasSelfDestructed(addr common.Address) bool {
	return s.inner.HasSelfDestructed(addr)
}

func (s *hookedStateDB) Exist(addr common.Address) bool {
	return s.inner.Exist(addr)
}

func (s *hookedStateDB) Empty(addr common.Address) bool {
	return s.inner.Empty(addr)
}

func (s *hookedStateDB) AddressInAccessList(addr common.Address) bool {
	return s.inner.AddressInAccessList(addr)
}

func (s *hookedStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.inner.SlotInAccessList(addr, slot)
}

func (s *hookedStateDB) AddAddressToAccessList(addr common.Address) {
	s.inner.AddAddressToAccessList(addr)
}

func (s *hookedStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.inner.AddSlotToAccessList(addr, slot)
}

func (s *hookedStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.inner.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (s *hookedStateDB) RevertToSnapshot(i int) {
	s.inner.RevertToSnapshot(i)
}

func (s *hookedStateDB) Snapshot() int {
	return s.inner.Snapshot()
}

func (s *hookedStateDB) AddPreimage(hash common.Hash, bytes []byte) {
	s.inner.AddPreimage(hash, bytes)
}

func (s *hookedStateDB) Witness() *stateless.Witness {
	return s.inner.Witness()
}

func (s *hookedStateDB) AccessEvents() *AccessEvents {
	return s.inner.AccessEvents()
}

func (s *hookedStateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.inner.SubBalance(addr, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		newBalance := new(uint256.Int).Sub(&prev, amount)
		s.hooks.OnBalanceChange(addr, prev.ToBig(), newBalance.ToBig(), reason)
	}
	return prev
}

func (s *hookedStateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.inner.AddBalance(addr, amount, reason)
	if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
		newBalance := new(uint256.Int).Add(&prev, amount)
		s.hooks.OnBalanceChange(addr, prev.ToBig(), newBalance.ToBig(), reason)
	}
	return prev
}

func (s *hookedStateDB) SetNonce(address common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	prev := s.inner.GetNonce(address)
	s.inner.SetNonce(address, nonce, reason)
	if s.hooks.OnNonceChangeV2 != nil {
		s.hooks.OnNonceChangeV2(address, prev, nonce, reason)
	} else if s.hooks.OnNonceChange != nil {
		s.hooks.OnNonceChange(address, prev, nonce)
	}
}

func (s *hookedStateDB) SetCode(address common.Address, code []byte, reason tracing.CodeChangeReason) []byte {
	prev := s.inner.SetCode(address, code, reason)

	if s.hooks.OnCodeChangeV2 != nil || s.hooks.OnCodeChange != nil {
		prevHash := crypto.Keccak256Hash(prev)
		codeHash := crypto.Keccak256Hash(code)

		// Invoke the hooks only if the contract code is changed
		if prevHash != codeHash {
			if s.hooks.OnCodeChangeV2 != nil {
				s.hooks.OnCodeChangeV2(address, prevHash, prev, codeHash, code, reason)
			} else if s.hooks.OnCodeChange != nil {
				s.hooks.OnCodeChange(address, prevHash, prev, codeHash, code)
			}
		}
	}
	return prev
}

func (s *hookedStateDB) SetState(address common.Address, key common.Hash, value common.Hash) common.Hash {
	prev := s.inner.SetState(address, key, value)
	if s.hooks.OnStorageChange != nil && prev != value {
		s.hooks.OnStorageChange(address, key, prev, value)
	}
	return prev
}

func (s *hookedStateDB) SelfDestruct(address common.Address) {
	s.inner.SelfDestruct(address)
}

func (s *hookedStateDB) AddLog(log *types.Log) {
	// The inner will modify the log (add fields), so invoke that first
	s.inner.AddLog(log)
	if s.hooks.OnLog != nil {
		s.hooks.OnLog(log)
	}
}

func (s *hookedStateDB) Finalise(deleteEmptyObjects bool) {
	if s.hooks.OnBalanceChange == nil && s.hooks.OnNonceChangeV2 == nil && s.hooks.OnNonceChange == nil && s.hooks.OnCodeChangeV2 == nil && s.hooks.OnCodeChange == nil {
		// Short circuit if no relevant hooks are set.
		s.inner.Finalise(deleteEmptyObjects)
		return
	}

	// Collect all self-destructed addresses first, then sort them to ensure
	// that state change hooks will be invoked in deterministic
	// order when the accounts are deleted below
	var selfDestructedAddrs []common.Address
	for addr := range s.inner.journal.dirties {
		obj := s.inner.stateObjects[addr]
		if obj == nil || !obj.selfDestructed {
			// Not self-destructed, keep searching.
			continue
		}
		selfDestructedAddrs = append(selfDestructedAddrs, addr)
	}
	sort.Slice(selfDestructedAddrs, func(i, j int) bool {
		return bytes.Compare(selfDestructedAddrs[i][:], selfDestructedAddrs[j][:]) < 0
	})

	for _, addr := range selfDestructedAddrs {
		obj := s.inner.stateObjects[addr]
		// Bingo: state object was self-destructed, call relevant hooks.

		// If ether was sent to account post-selfdestruct, record as burnt.
		if s.hooks.OnBalanceChange != nil {
			if bal := obj.Balance(); bal.Sign() != 0 {
				s.hooks.OnBalanceChange(addr, bal.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}

		// Nonce is set to reset on self-destruct.
		if s.hooks.OnNonceChangeV2 != nil {
			s.hooks.OnNonceChangeV2(addr, obj.Nonce(), 0, tracing.NonceChangeSelfdestruct)
		} else if s.hooks.OnNonceChange != nil {
			s.hooks.OnNonceChange(addr, obj.Nonce(), 0)
		}

		// If an initcode invokes selfdestruct, do not emit a code change.
		prevCodeHash := s.inner.GetCodeHash(addr)
		if prevCodeHash == types.EmptyCodeHash {
			continue
		}
		// Otherwise, trace the change.
		if s.hooks.OnCodeChangeV2 != nil {
			s.hooks.OnCodeChangeV2(addr, prevCodeHash, s.inner.GetCode(addr), types.EmptyCodeHash, nil, tracing.CodeChangeSelfDestruct)
		} else if s.hooks.OnCodeChange != nil {
			s.hooks.OnCodeChange(addr, prevCodeHash, s.inner.GetCode(addr), types.EmptyCodeHash, nil)
		}
	}

	s.inner.Finalise(deleteEmptyObjects)
}
