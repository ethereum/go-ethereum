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

	// ErrGasLimitReached is returned by the gas pool if the amount of gas required
	// by a transaction is higher than what's left in the block.
	ErrGasLimitReached = errors.New("gas limit reached")

	// ErrBlacklistedHash is returned if a block to import is on the blacklist.
	ErrBlacklistedHash = errors.New("blacklisted hash")

	// ErrNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	ErrNonceTooHigh = errors.New("nonce too high")

	// ErrNoGenesis is returned when there is no Genesis Block.
	ErrNoGenesis = errors.New("genesis not found in chain")

	// ErrGenesisNoConfig is retrurned when setup Genensis Block, if there is no
	// ChainConfig.
	ErrGenesisNoConfig = errors.New("genesis has no chain configuration")

	// ErrInsufficientBalanceForGas is returned if balance is insufficient to pay
	// for gas
	ErrInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")

	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	ErrInvalidSender = errors.New("invalid sender")

	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrUnderpriced is returned if a transaction's gas price is below the minimum
	// configured for the transaction pool.
	ErrUnderpriced = errors.New("transaction underpriced")

	// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
	// with a different one without the required price bump.
	ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	// ErrIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	ErrIntrinsicGas = errors.New("intrinsic gas too low")

	// ErrGasLimit is returned if a transaction's requested gas limit exceeds the
	// maximum allowance of the current block.
	ErrGasLimit = errors.New("exceeds block gas limit")

	// ErrNegativeValue is a sanity error to ensure noone is able to specify a
	// transaction with a negative value.
	ErrNegativeValue = errors.New("negative value")

	// ErrOversizedData is returned if the input data of a transaction is greater
	// than some meaningful limit a user might use. This is not a consensus error
	// making the transaction invalid, rather a DOS protection.
	ErrOversizedData = errors.New("oversized data")

	// ErrNoActiveJournal is returned if a transaction is attempted to be inserted
	// into the journal, but no such file is currently open.
	ErrNoActiveJournal = errors.New("no active journal")

	// ErrGhostStateAttack is returned when sidechain ghost-state attack
	ErrGhostStateAttack = errors.New("sidechain ghost-state attack")

	// ErrMissingParent is returned when parent block is nil
	ErrMissingParent = errors.New("missing parent")

	// ErrPrematureAbort is returned when chain is terminating
	ErrPrematureAbort = errors.New("aborted")
)
