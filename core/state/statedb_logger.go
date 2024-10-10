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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type stateDBLogger struct {
	*StateDB
	hooks *tracing.Hooks
}

func newStateDBLogger(db *StateDB, hooks *tracing.Hooks) *stateDBLogger {
	s := &stateDBLogger{db, hooks}
	if s.hooks == nil {
		s.hooks = new(tracing.Hooks)
	}
	db.SetBurnCallback(func(address common.Address, amount *uint256.Int) {
		if s.hooks.OnBalanceChange != nil {
			s.hooks.OnBalanceChange(address, amount.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestructBurn)
		}
	})
	return s
}

func (s *stateDBLogger) AddBalance(address common.Address, amount *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	prev := s.StateDB.AddBalance(address, amount, reason)
	if s.hooks.OnBalanceChange != nil {
		newBalance := new(uint256.Int).Add(&prev, amount)
		s.hooks.OnBalanceChange(address, prev.ToBig(), newBalance.ToBig(), reason)
	}
	return prev
}

func (s *stateDBLogger) SetNonce(address common.Address, nonce uint64) {
	s.StateDB.SetNonce(address, nonce)
	if s.hooks.OnNonceChange != nil {
		s.hooks.OnNonceChange(address, nonce-1, nonce)
	}
}

func (s *stateDBLogger) SetCode(address common.Address, code []byte) {
	s.StateDB.SetCode(address, code)
	if s.hooks.OnCodeChange != nil {
		s.hooks.OnCodeChange(address, types.EmptyCodeHash, nil, crypto.Keccak256Hash(code), code)
	}
}

func (s *stateDBLogger) SetState(address common.Address, key common.Hash, value common.Hash) common.Hash {
	prev := s.StateDB.SetState(address, key, value)
	if s.hooks.OnStorageChange != nil && prev != value {
		s.hooks.OnStorageChange(address, key, prev, value)
	}
	return prev
}

func (s *stateDBLogger) SelfDestruct(address common.Address) uint256.Int {
	prev := s.StateDB.SelfDestruct(address)
	if !prev.IsZero() {
		if s.hooks.OnBalanceChange != nil {
			s.hooks.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
	}
	return prev
}

func (s *stateDBLogger) Selfdestruct6780(address common.Address) uint256.Int {
	prev := s.StateDB.Selfdestruct6780(address)
	if !prev.IsZero() {
		if s.hooks.OnBalanceChange != nil {
			s.hooks.OnBalanceChange(address, prev.ToBig(), new(big.Int), tracing.BalanceDecreaseSelfdestruct)
		}
	}
	return prev
}

func (s *stateDBLogger) AddLog(log *types.Log) {
	// The inner will modify the log (add fields), so invoke that first
	s.StateDB.AddLog(log)
	if s.hooks.OnLog != nil {
		s.hooks.OnLog(log)
	}
}
