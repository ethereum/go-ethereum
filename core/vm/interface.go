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
	"github.com/ethereum/go-ethereum/params"
)

// StateDB is an EVM database for full state querying.
type StateDB interface {
	// CreateAccount creates a new account
	//
	// If an account already exists at the address, it should transfer the balance of the old
	// address to the new one.
	CreateAccount(common.Address)

	// SubBalance subtracts the amount from the address's balance
	SubBalance(common.Address, *big.Int)
	// AddBalance adds the amount to the address's balance
	AddBalance(common.Address, *big.Int)
	// GetBalance returns the balance for the given address
	//
	// For non-existent addresses it should returns zero
	// For self-destructed addresses (during the tx execution) it should return zero
	GetBalance(common.Address) *big.Int

	// GetNonce returns the nonce for the given address
	//
	// For non-existent addresses it should returns zero
	GetNonce(common.Address) uint64
	// SetNonce sets the nonce for the given address
	SetNonce(common.Address, uint64)

	// GetCodeHash returns the codeHash for the given address
	//
	// For non-existent addresses or addresses without a code (EOAs) it should returns types.EmptyCodeHash
	GetCodeHash(common.Address) common.Hash
	// GetCode returns the code for the given address,
	//
	// For non-existent addresses or addresses without a code (EOAs) it should returns nil
	GetCode(common.Address) []byte
	// SetCode sets the code for the given address
	SetCode(common.Address, []byte)
	// GetCodeSize returns the size of the code for the given address
	//
	// For non-existent addresses or addresses without a code (EOAs) it should returns zero
	GetCodeSize(common.Address) int

	// AddRefund adds the amount to the total (gas) refund
	AddRefund(uint64)
	// SubRefund subtracts the amount from the total (gas) refund
	SubRefund(uint64)
	// GetRefund returns the total (gas) refund
	GetRefund() uint64

	// GetCommittedState returns the value for the given (address, key) pair (aka slot) using committed state only
	//
	// For non-existent addresses or non-existent keys it should return empty hash (common.Hash{})
	GetCommittedState(addr common.Address, key common.Hash) common.Hash
	// GetState returns the value for the given (address, key) pair (aka slot) including recent changes
	//
	// For non-existent addresses or non-existent keys it should return empty hash (common.Hash{})
	GetState(addr common.Address, key common.Hash) common.Hash
	// SetState sets a value for the given (address, key) pair (aka slot)
	//
	// If a value already exist at the given pair, it should be replaced.
	SetState(addr common.Address, key common.Hash, value common.Hash)

	// GetCommittedState returns the value for the given (address, key) pair (aka slot) in the transient state
	//
	// check EIP-1153 for information about transient state
	GetTransientState(addr common.Address, key common.Hash) common.Hash
	// SetTransientState sets a value for the given (address, key) pair (aka slot) in the transient state
	//
	// check EIP-1153 for information about transient state
	SetTransientState(addr common.Address, key, value common.Hash)

	// SelfDestruct flags an address for destruction at the end of a transaction
	SelfDestruct(common.Address)
	// HasSelfDestructed returns true if address has been flaged for destruction (during this transaction)
	HasSelfDestructed(common.Address) bool

	// Selfdestruct6780 implements the logic of self destruct according to the EIP-6700
	Selfdestruct6780(common.Address)

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for self-destructed accounts.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP-161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	// AddressInAccessList checks if an address is in the access list
	//
	// See EIP-2930 for more details
	AddressInAccessList(addr common.Address) bool
	// SlotInAccessList checks if an slot is in the access list
	//
	// See EIP-2930 for more details
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	// AddAddressToAccessList adds the given address to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	//
	// See EIP-2930 for more details
	AddAddressToAccessList(addr common.Address)
	// AddSlotToAccessList adds the given (address,slot) to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	//
	// See EIP-2930 for more details
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	// Prepare prepares the storage for the next transaction
	// it resets the access list and transient storage,
	// it also adds addresses and slots to the list based on the rules of the network.
	Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)

	// RevertToSnapshot reverts the state to an specific snapshot
	RevertToSnapshot(int)
	// Snapshot takes a snapshot and returns an integer that can be used later with RevertToSnapshot
	//
	// Nested snapshoting should be supported.
	Snapshot() int

	// AddLog appends a log to transaction's log collection
	AddLog(*types.Log)
	// AddPreimage adds a preimage to the map of preimages
	AddPreimage(hash common.Hash, preimage []byte)
}

// CallContext provides a basic interface for the EVM calling conventions. The EVM
// depends on this context being implemented for doing subcalls and initialising new EVM contracts.
type CallContext interface {
	// Call calls another contract.
	Call(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// CallCode takes another contracts code and execute within our own context
	CallCode(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, error)
	// DelegateCall is same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *EVM, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, error)
	// Create creates a new contract
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, error)
}
