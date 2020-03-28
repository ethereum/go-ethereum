// Copyright 2014 The go-ethereum Authors
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

package core

import "errors"

var (
	// ErrKnownBlock is returned when a block to import is already known locally.
	ErrKnownBlock = errors.New("block already known")

	// ErrBlacklistedHash is returned if a block to import is on the blacklist.
	ErrBlacklistedHash = errors.New("blacklisted hash")

	// ErrNoGenesis is returned when there is no Genesis Block.
	ErrNoGenesis = errors.New("genesis not found in chain")
)

// State transition consensus errors, any of them encountered during
// the block processing can lead to consensus issue.
var (
	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	ErrNonceTooHigh = errors.New("nonce too high")

	// ErrGasLimitReached is returned by the gas pool if the amount of gas required
	// by a transaction is higher than what's left in the block.
	ErrGasLimitReached = errors.New("gas limit reached")

	// ErrInsufficientBalanceForTransfer is returned if the transaction sender doesn't
	// have enough balance for transfer(topmost call only).
	ErrInsufficientBalanceForTransfer = errors.New("insufficient balance for transfer")

	// ErrInsufficientBalanceForFee is returned if transaction sender doesn't have
	// enough balance to cover transaction fee.
	ErrInsufficientBalanceForFee = errors.New("insufficient balance to pay fee")

	// ErrGasOverflow is returned when calculating gas usage.
	ErrGasOverflow = errors.New("gas overflow")

	// ErrInsufficientIntrinsicGas is returned when the gas limit speicified in transaction
	// is not enought to cover intrinsic gas usage.
	ErrInsufficientIntrinsicGas = errors.New("insufficient intrinsic gas")
)
