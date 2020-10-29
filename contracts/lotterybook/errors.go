// Copyright 2020 The go-ethereum Authors
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

package lotterybook

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// ErrNotEnoughDeposit is returned if the cheque drawer doesn't have enough
	// balance to spend. Whenever drawee receives this error, it should emit a
	// cash operation as soon as possible.
	ErrNotEnoughDeposit = errors.New("deposit is not enough")

	// ErrTransactionFailed is returned if the sent transaction is failed.
	ErrTransactionFailed = errors.New("transaction failed")
)

type ErrTransactionNotConfirmed struct {
	Id       common.Hash // Lottery ID
	Nonce    uint64      // Sender's nonce
	GasPrice *big.Int    // Assigned gas price
	Msg      string      // Error message
}

func (err *ErrTransactionNotConfirmed) Error() string {
	return err.Msg
}

func NewErrTransactionNotConfirmed(id common.Hash, nonce uint64, price *big.Int) error {
	return &ErrTransactionNotConfirmed{
		Id:       id,
		Nonce:    nonce,
		GasPrice: price,
		Msg:      "Transaction not confirmed",
	}
}
