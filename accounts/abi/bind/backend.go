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

package bind

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ErrNoCode is returned by call and transact operations for which the requested
// recipient contract to operate on does not exist in the state db or does not
// have any code associated with it (i.e. suicided).
//
// Please note, this error string is part of the RPC API and is expected by the
// native contract bindings to signal this particular error. Do not change this
// as it will break all dependent code!
var ErrNoCode = errors.New("no contract code at given address")

// ContractCaller defines the methods needed to allow operating with contract on a read
// only basis.
type ContractCaller interface {
	// ContractCall executes an Ethereum contract call with the specified data as
	// the input. The pending flag requests execution against the pending block, not
	// the stable head of the chain.
	ContractCall(contract common.Address, data []byte, pending bool) ([]byte, error)
}

// ContractTransactor defines the methods needed to allow operating with contract
// on a write only basis. Beside the transacting method, the remainder are helpers
// used when the user does not provide some needed values, but rather leaves it up
// to the transactor to decide.
type ContractTransactor interface {
	// PendingAccountNonce retrieves the current pending nonce associated with an
	// account.
	PendingAccountNonce(account common.Address) (uint64, error)

	// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
	// execution of a transaction.
	SuggestGasPrice() (*big.Int, error)

	// EstimateGasLimit tries to estimate the gas needed to execute a specific
	// transaction based on the current pending state of the backend blockchain.
	// There is no guarantee that this is the true gas limit requirement as other
	// transactions may be added or removed by miners, but it should provide a basis
	// for setting a reasonable default.
	EstimateGasLimit(sender common.Address, contract *common.Address, value *big.Int, data []byte) (*big.Int, error)

	// SendTransaction injects the transaction into the pending pool for execution.
	SendTransaction(tx *types.Transaction) error
}

// ContractBackend defines the methods needed to allow operating with contract
// on a read-write basis.
type ContractBackend interface {
	ContractCaller
	ContractTransactor
}
