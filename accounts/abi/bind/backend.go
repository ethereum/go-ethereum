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

	"github.com/ethereum/go-ethereum"
)

// ErrNoCode is returned by call and transact operations for which the requested
// recipient contract to operate on does not exist in the state db or does not
// have any code associated with it (i.e. suicided).
var ErrNoCode = errors.New("no contract code at given address")

// ContractBackend defines the methods needed to allow operating with contract
// on a read-write basis.
type ContractBackend interface {
	ethereum.ChainStateReader
	ethereum.ContractCaller
	ethereum.LogFilterer
	ethereum.GasPricer
	ethereum.TransactionSender
	ethereum.PendingStateReader
	ethereum.PendingContractCaller
	ethereum.GasEstimator
}
