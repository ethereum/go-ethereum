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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

// HookedStateDB represents a statedb which emits calls to tracing-hooks
// on state operations.
type HookedStateDB struct {
	vm.StateDB
	hooks *tracing.Hooks
}

// NewHookedState wraps the given stateDb with the given hooks
func NewHookedState(stateDb vm.StateDB, hooks *tracing.Hooks) *HookedStateDB {
	s := &HookedStateDB{stateDb, hooks}
	if s.hooks == nil {
		s.hooks = new(tracing.Hooks)
	}
	return s
}

func (s *HookedStateDB) CreateAccount(addr common.Address) {
	s.StateDB.CreateAccount(addr)
}

func (s *HookedStateDB) CreateContract(addr common.Address) {
	s.StateDB.CreateContract(addr)
}

func (s *HookedStateDB) GetBalance(addr common.Address) *uint256.Int {
	return s.StateDB.GetBalance(addr)
}

func (s *HookedStateDB) GetNonce(addr common.Address) uint64 {
	return s.StateDB.GetNonce(addr)
}

func (s *HookedStateDB) GetCodeHash(addr common.Address) common.Hash {
	return s.StateDB.GetCodeHash(addr)
}

func (s *HookedStateDB) GetCode(addr common.Address) []byte {
	return s.StateDB.GetCode(addr)
}

func (s *HookedStateDB) GetCodeSize(addr common.Address) int {
	return s.StateDB.GetCodeSize(addr)
}

func (s *HookedStateDB) AddRefund(u uint64) {
	s.StateDB.AddRefund(u)
}

func (s *HookedStateDB) SubRefund(u uint64) {
	s.StateDB.SubRefund(u)
}

func (s *HookedStateDB) GetRefund() uint64 {
	return s.StateDB.GetRefund()
}

func (s *HookedStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	return s.StateDB.GetCommittedState(addr, hash)
}

func (s *HookedStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	return s.StateDB.GetState(addr, hash)
}

func (s *HookedStateDB) GetStorageRoot(addr common.Address) common.Hash {
	return s.StateDB.GetStorageRoot(addr)
}

func (s *HookedStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.StateDB.GetTransientState(addr, key)
}

func (s *HookedStateDB) SetTransientState(addr common.Address, key, value common.Hash) {
	s.StateDB.SetTransientState(addr, key, value)
}

func (s *HookedStateDB) HasSelfDestructed(addr common.Address) bool {
	return s.StateDB.HasSelfDestructed(addr)
}

func (s *HookedStateDB) Exist(addr common.Address) bool {
	return s.StateDB.Exist(addr)
}

func (s *HookedStateDB) Empty(addr common.Address) bool {
	return s.StateDB.Empty(addr)
}

func (s *HookedStateDB) AddressInAccessList(addr common.Address) bool {
	return s.StateDB.AddressInAccessList(addr)
}

func (s *HookedStateDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.StateDB.SlotInAccessList(addr, slot)
}

func (s *HookedStateDB) AddAddressToAccessList(addr common.Address) {
	s.StateDB.AddAddressToAccessList(addr)
}

func (s *HookedStateDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	s.StateDB.AddSlotToAccessList(addr, slot)
}

func (s *HookedStateDB) PointCache() *utils.PointCache {
	return s.StateDB.PointCache()
}

func (s *HookedStateDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.StateDB.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (s *HookedStateDB) RevertToSnapshot(i int) {
	s.StateDB.RevertToSnapshot(i)
}

func (s *HookedStateDB) Snapshot() int {
	return s.StateDB.Snapshot()
}

func (s *HookedStateDB) AddPreimage(hash common.Hash, bytes []byte) {
	s.StateDB.AddPreimage(hash, bytes)
}

func (s *HookedStateDB) Witness() *stateless.Witness {
	return s.StateDB.Witness()
}

func (s *HookedStateDB) AccessEvents() *vm.AccessEvents {
	return s.StateDB.AccessEvents()
}

func (s *HookedStateDB) SubBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.StateDB.SubBalance(addr, amount, reason)
	// tracked already
	// if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
	// 	newBalance := new(uint256.Int).Sub(&prev, amount)
	// 	s.hooks.OnBalanceChange(addr, prev.ToBig(), newBalance.ToBig(), reason)
	// }
	return prev
}

func (s *HookedStateDB) AddBalance(addr common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.StateDB.AddBalance(addr, amount, reason)
	// tracked already
	// if s.hooks.OnBalanceChange != nil && !amount.IsZero() {
	// 	newBalance := new(uint256.Int).Add(&prev, amount)
	// 	s.hooks.OnBalanceChange(addr, prev.ToBig(), newBalance.ToBig(), reason)
	// }
	return prev
}

func (s *HookedStateDB) SetNonce(address common.Address, nonce uint64, reason tracing.NonceChangeReason) {
	prev := s.StateDB.GetNonce(address)
	s.StateDB.SetNonce(address, nonce, reason)
	if s.hooks.OnNonceChangeV2 != nil {
		s.hooks.OnNonceChangeV2(address, prev, nonce, reason)
	} else if s.hooks.OnNonceChange != nil {
		s.hooks.OnNonceChange(address, prev, nonce)
	}
}

func (s *HookedStateDB) SetCode(address common.Address, code []byte) []byte {
	prev := s.StateDB.SetCode(address, code)
	if s.hooks.OnCodeChange != nil {
		prevHash := types.EmptyCodeHash
		if len(prev) != 0 {
			prevHash = crypto.Keccak256Hash(prev)
		}
		s.hooks.OnCodeChange(address, prevHash, prev, crypto.Keccak256Hash(code), code)
	}
	return prev
}

func (s *HookedStateDB) SetState(address common.Address, key common.Hash, value common.Hash) common.Hash {
	prev := s.StateDB.SetState(address, key, value)
	if s.hooks.OnStorageChange != nil && prev != value {
		s.hooks.OnStorageChange(address, key, prev, value)
	}
	return prev
}

func (s *HookedStateDB) SelfDestruct(address common.Address) uint256.Int {
	var prevCode []byte
	var prevCodeHash common.Hash

	if s.hooks.OnCodeChange != nil {
		prevCode = s.StateDB.GetCode(address)
		prevCodeHash = s.StateDB.GetCodeHash(address)
	}

	prev := s.StateDB.SelfDestruct(address)

	if s.hooks.OnBalanceChange != nil && !prev.IsZero() {
		s.hooks.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
	}

	if s.hooks.OnCodeChange != nil && len(prevCode) > 0 {
		s.hooks.OnCodeChange(address, prevCodeHash, prevCode, types.EmptyCodeHash, nil)
	}

	return prev
}

func (s *HookedStateDB) SelfDestruct6780(address common.Address) (uint256.Int, bool) {
	var prevCode []byte
	var prevCodeHash common.Hash

	if s.hooks.OnCodeChange != nil {
		prevCodeHash = s.StateDB.GetCodeHash(address)
		prevCode = s.StateDB.GetCode(address)
	}

	prev, changed := s.StateDB.SelfDestruct6780(address)

	if s.hooks.OnBalanceChange != nil && changed && !prev.IsZero() {
		s.hooks.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
	}

	if s.hooks.OnCodeChange != nil && changed && len(prevCode) > 0 {
		s.hooks.OnCodeChange(address, prevCodeHash, prevCode, types.EmptyCodeHash, nil)
	}

	return prev, changed
}

func (s *HookedStateDB) AddLog(log *types.Log) {
	// The inner will modify the log (add fields), so invoke that first
	s.StateDB.AddLog(log)
	if s.hooks.OnLog != nil {
		s.hooks.OnLog(log)
	}
}

func (s *HookedStateDB) Finalise(deleteEmptyObjects bool) {
	defer s.StateDB.Finalise(deleteEmptyObjects)
	if s.hooks.OnBalanceChange == nil {
		return
	}
	statedb, ok := s.StateDB.(*StateDB)
	if !ok {
		return
	}
	for addr := range statedb.journal.dirties {
		obj := statedb.stateObjects[addr]
		if obj != nil && obj.selfDestructed {
			// If ether was sent to account post-selfdestruct it is burnt.
			if bal := obj.Balance(); bal.Sign() != 0 {
				s.hooks.OnBalanceChange(addr, bal.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
			}
		}
	}
}
