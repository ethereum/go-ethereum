// Copyright 2015 The go-ethereum Authors
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

package bind

import (
	"github.com/ethereum/go-ethereum/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// SignerFn is a signer function callback when a contract requires a method to
// sign the transaction before submission.
type SignerFn = bind.SignerFn

// CallOpts is the collection of options to fine tune a contract call request.
type CallOpts = bind.CallOpts

// TransactOpts is the collection of authorization data required to create a
// valid Ethereum transaction.
type TransactOpts = bind.TransactOpts

// FilterOpts is the collection of options to fine tune filtering for events
// within a bound contract.
type FilterOpts = bind.FilterOpts

// WatchOpts is the collection of options to fine tune subscribing for events
// within a bound contract.
type WatchOpts = bind.WatchOpts

// BoundContract is the base wrapper object that reflects a contract on the
// Ethereum network. It contains a collection of methods that are used by the
// higher level contract bindings to operate.
type BoundContract = bind.BoundContract

// NewBoundContract creates a low level contract interface through which calls
// and transactions may be made through.
func NewBoundContract(address common.Address, abi abi.ABI, caller ContractCaller, transactor ContractTransactor, filterer ContractFilterer) *BoundContract {
	return bind.NewBoundContract(address, abi, caller, transactor, filterer)
}

// DeployContract deploys a contract onto the Ethereum blockchain and binds the
// deployment address with a Go wrapper.
func DeployContract(opts *TransactOpts, abi abi.ABI, bytecode []byte, backend ContractBackend, params ...interface{}) (common.Address, *types.Transaction, *BoundContract, error) {
	return bind.DeployContract(opts, abi, bytecode, backend, params)
}
