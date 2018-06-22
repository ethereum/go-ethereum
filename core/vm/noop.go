// Copyright 2016 The go-ethereum Authors
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

package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// NoopCanTransfer dummy function
func NoopCanTransfer(db StateDB, from common.Address, balance *big.Int) bool {
	return true
}

// NoopTransfer dummy function
func NoopTransfer(db StateDB, from, to common.Address, amount *big.Int) {}

// NoopEVMCallContext represents the EVM's call context
type NoopEVMCallContext struct{}

// Call dummy function
func (NoopEVMCallContext) Call(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}

// CallCode dummy function
func (NoopEVMCallContext) CallCode(caller ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error) {
	return nil, nil
}

// Create dummy function
func (NoopEVMCallContext) Create(caller ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error) {
	return nil, common.Address{}, nil
}

// DelegateCall dummy function
func (NoopEVMCallContext) DelegateCall(me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error) {
	return nil, nil
}

// NoopStateDB is an dummy state database
type NoopStateDB struct{}

// CreateAccount dummy method
func (NoopStateDB) CreateAccount(common.Address) {}

// SubBalance dummy method
func (NoopStateDB) SubBalance(common.Address, *big.Int) {}

// AddBalance dummy method
func (NoopStateDB) AddBalance(common.Address, *big.Int) {}

// GetBalance dummy method
func (NoopStateDB) GetBalance(common.Address) *big.Int { return nil }

// GetNonce dummy method
func (NoopStateDB) GetNonce(common.Address) uint64 { return 0 }

// SetNonce dummy method
func (NoopStateDB) SetNonce(common.Address, uint64) {}

// GetCodeHash dummy method
func (NoopStateDB) GetCodeHash(common.Address) common.Hash { return common.Hash{} }

// GetCode dummy method
func (NoopStateDB) GetCode(common.Address) []byte { return nil }

// SetCode dummy method
func (NoopStateDB) SetCode(common.Address, []byte) {}

// GetCodeSize dummy method
func (NoopStateDB) GetCodeSize(common.Address) int { return 0 }

// AddRefund dummy method
func (NoopStateDB) AddRefund(uint64) {}

// GetRefund dummy method
func (NoopStateDB) GetRefund() uint64 { return 0 }

// GetState dummy method
func (NoopStateDB) GetState(common.Address, common.Hash) common.Hash { return common.Hash{} }

// SetState dummy method
func (NoopStateDB) SetState(common.Address, common.Hash, common.Hash) {}

// Suicide dummy method
func (NoopStateDB) Suicide(common.Address) bool { return false }

// HasSuicided dummy method
func (NoopStateDB) HasSuicided(common.Address) bool { return false }

// Exist dummy method
func (NoopStateDB) Exist(common.Address) bool { return false }

// Empty dummy method
func (NoopStateDB) Empty(common.Address) bool { return false }

// RevertToSnapshot dummy method
func (NoopStateDB) RevertToSnapshot(int) {}

// Snapshot dummy method
func (NoopStateDB) Snapshot() int { return 0 }

// AddLog dummy method
func (NoopStateDB) AddLog(*types.Log) {}

// AddPreimage dummy method
func (NoopStateDB) AddPreimage(common.Hash, []byte) {}

// ForEachStorage dummy method
func (NoopStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {}
