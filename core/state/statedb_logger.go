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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

type stateDBLogger struct {
	inner  *StateDB
	logger *tracing.Hooks
}

func newStateDBLogger(db *StateDB, logger *tracing.Hooks) *stateDBLogger {
	s := &stateDBLogger{db, logger}

	db.SetBurnCallback(func(address common.Address, amount *uint256.Int) {
		if s.logger != nil && s.logger.OnBalanceChange != nil {
			s.logger.OnBalanceChange(address, amount.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
		}
	})
	return s
}

func (s *stateDBLogger) CreateAccount(address common.Address) {
	s.inner.CreateAccount(address)
}

func (s *stateDBLogger) CreateContract(address common.Address) {
	s.inner.CreateContract(address)
}

func (s *stateDBLogger) SubBalance(address common.Address, u *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	return s.inner.SubBalance(address, u, reason)
}

func (s *stateDBLogger) AddBalance(address common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.inner.AddBalance(address, amount, reason)
	if s.logger != nil && s.logger.OnBalanceChange != nil {
		s.logger.OnBalanceChange(address, prev.ToBig(), amount.ToBig(), reason)
	}
	return prev
}

func (s *stateDBLogger) GetBalance(address common.Address) *uint256.Int {
	return s.inner.GetBalance(address)
}

func (s *stateDBLogger) GetNonce(address common.Address) uint64 {
	return s.inner.GetNonce(address)
}

func (s *stateDBLogger) SetNonce(address common.Address, nonce uint64) {
	s.inner.SetNonce(address, nonce)
	if s.logger != nil && s.logger.OnNonceChange != nil {
		s.logger.OnNonceChange(address, nonce-1, nonce)
	}
}

func (s *stateDBLogger) GetCodeHash(address common.Address) common.Hash {
	return s.inner.GetCodeHash(address)
}

func (s *stateDBLogger) GetCode(address common.Address) []byte {
	return s.inner.GetCode(address)
}

func (s *stateDBLogger) SetCode(address common.Address, code []byte) {
	s.inner.SetCode(address, code)
	if s.logger != nil && s.logger.OnCodeChange != nil {
		s.logger.OnCodeChange(address, types.EmptyCodeHash, nil, crypto.Keccak256Hash(code), code)
	}
}

func (s *stateDBLogger) GetCodeSize(address common.Address) int {
	return s.inner.GetCodeSize(address)
}

func (s *stateDBLogger) AddRefund(u uint64) {
	s.inner.AddRefund(u)
}

func (s *stateDBLogger) SubRefund(u uint64) {
	s.inner.SubRefund(u)
}

func (s *stateDBLogger) GetRefund() uint64 {
	return s.inner.GetRefund()
}

func (s *stateDBLogger) GetCommittedState(address common.Address, key common.Hash) common.Hash {
	return s.inner.GetCommittedState(address, key)
}

func (s *stateDBLogger) GetState(address common.Address, key common.Hash) common.Hash {
	return s.inner.GetState(address, key)
}

func (s *stateDBLogger) SetState(address common.Address, key common.Hash, value common.Hash) common.Hash {
	prev := s.inner.SetState(address, key, value)
	if s.logger != nil && s.logger.OnStorageChange != nil && prev != value {
		s.logger.OnStorageChange(address, key, prev, value)
	}
	return prev
}

func (s *stateDBLogger) GetStorageRoot(address common.Address) common.Hash {
	return s.inner.GetStorageRoot(address)
}

func (s *stateDBLogger) GetTransientState(address common.Address, key common.Hash) common.Hash {
	return s.inner.GetTransientState(address, key)
}

func (s *stateDBLogger) SetTransientState(address common.Address, key, value common.Hash) {
	s.inner.SetTransientState(address, key, value)
}

func (s *stateDBLogger) SelfDestruct(address common.Address) uint256.Int {
	prev := s.inner.SelfDestruct(address)
	if !prev.IsZero() {
		if s.logger != nil && s.logger.OnBalanceChange != nil {
			s.logger.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
	}
	return prev

}

func (s *stateDBLogger) HasSelfDestructed(address common.Address) bool {
	return s.inner.HasSelfDestructed(address)
}

func (s *stateDBLogger) Selfdestruct6780(address common.Address) uint256.Int {
	prev := s.inner.Selfdestruct6780(address)
	if !prev.IsZero() {
		if s.logger != nil && s.logger.OnBalanceChange != nil {
			s.logger.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
	}
	return prev
}

func (s *stateDBLogger) Exist(address common.Address) bool {
	return s.inner.Exist(address)
}

func (s *stateDBLogger) Empty(address common.Address) bool {
	return s.inner.Empty(address)
}

func (s *stateDBLogger) AddressInAccessList(address common.Address) bool {
	return s.inner.AddressInAccessList(address)
}

func (s *stateDBLogger) SlotInAccessList(address common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return s.inner.SlotInAccessList(address, slot)
}

func (s *stateDBLogger) AddAddressToAccessList(address common.Address) {
	s.inner.AddAddressToAccessList(address)
}

func (s *stateDBLogger) AddSlotToAccessList(address common.Address, slot common.Hash) {
	s.inner.AddSlotToAccessList(address, slot)
}

func (s *stateDBLogger) PointCache() *utils.PointCache {
	return s.inner.PointCache()
}

func (s *stateDBLogger) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	s.inner.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (s *stateDBLogger) RevertToSnapshot(i int) {
	s.inner.RevertToSnapshot(i)
}

func (s *stateDBLogger) Snapshot() int {
	return s.inner.Snapshot()
}

func (s *stateDBLogger) AddLog(log *types.Log) {
	// The inner will modify the log (add fields), so invoke that first
	s.inner.AddLog(log)
	if s.logger != nil && s.logger.OnLog != nil {
		s.logger.OnLog(log)
	}
}

func (s *stateDBLogger) AddPreimage(hash common.Hash, bytes []byte) {
	s.inner.AddPreimage(hash, bytes)
}

func (s *stateDBLogger) Witness() *stateless.Witness {
	return s.inner.Witness()
}

func (s *stateDBLogger) Finalise(deleteEmptyObjects bool) {
	s.inner.Finalise(deleteEmptyObjects)
}

// GetLogs returns the logs matching the specified transaction hash, and annotates
// them with the given blockNumber and blockHash.
func (s *stateDBLogger) GetLogs(txHash common.Hash, blockNumber uint64, blockHash common.Hash) []*types.Log {
	return s.inner.GetLogs(txHash, blockNumber, blockHash)
}

// TxIndex returns the current transaction index set by SetTxContext.
func (s *stateDBLogger) TxIndex() int {
	return s.inner.TxIndex()
}
func (s *stateDBLogger) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	return s.inner.IntermediateRoot(deleteEmptyObjects)
}

func (s stateDBLogger) SetTxContext(txHash common.Hash, txIndex int) {
	s.inner.SetTxContext(txHash, txIndex)
}
