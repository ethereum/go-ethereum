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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

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
	// Nonce retrieves the current pending nonce associated with an account.
	AccountNonce(account common.Address) (uint64, error)

	// GasPrice retrieves the currently suggested gas price to allow a timely execution
	// of a transaction.
	GasPrice() (*big.Int, error)

	// GasLimit tries to estimate the gas needed to execute a specific transaction.
	GasLimit(sender common.Address, contract *common.Address, value *big.Int, data []byte) (*big.Int, error)

	// SendTransaction injects the transaction into the pending pool for execution.
	SendTransaction(*types.Transaction) error
}

// ContractBackend defines the methods needed to allow operating with contract
// on a read-write basis.
type ContractBackend interface {
	ContractCaller
	ContractTransactor
}
