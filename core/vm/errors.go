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

package vm

import "errors"

// List execution errors
var (
	// ErrOutOfGas is returned when gas is insufficient.
	ErrOutOfGas = errors.New("out of gas")

	// ErrCodeStoreOutOfGas is returned when gas is insufficient for making contract.
	ErrCodeStoreOutOfGas = errors.New("contract creation code storage out of gas")

	// ErrDepth is returned when max call depth exceeded.
	ErrDepth = errors.New("max call depth exceeded")

	// ErrTraceLimitReached is returned when logs reached the specified limit.
	ErrTraceLimitReached = errors.New("the number of logs reached the specified limit")

	// ErrInsufficientBalance is returned when balance is insufficient for transfer.
	ErrInsufficientBalance = errors.New("insufficient balance for transfer")

	// ErrContractAddressCollision is returned when contract address collide.
	ErrContractAddressCollision = errors.New("contract address collision")

	// ErrNoCompatibleInterpreter is returned when interpreter is not compatible.
	ErrNoCompatibleInterpreter = errors.New("no compatible interpreter")

	// ErrBadPairingInput is returned if the bn256 pairing input is invalid.
	ErrBadPairingInput = errors.New("bad elliptic curve pairing size")

	// ErrWriteProtection is returned when mode is readonly, it blocks writing.
	ErrWriteProtection = errors.New("evm: write protection")

	// ErrReturnDataOutOfBounds is returned when return data is out of bounds.
	ErrReturnDataOutOfBounds = errors.New("evm: return data out of bounds")

	// ErrExecutionReverted is returned if execution is reverted.
	ErrExecutionReverted = errors.New("evm: execution reverted")

	// ErrMaxCodeSizeExceeded is returned when max code size is exceeded.
	ErrMaxCodeSizeExceeded = errors.New("evm: max code size exceeded")

	// ErrInvalidJump is returned when jump destination is invalid.
	ErrInvalidJump = errors.New("evm: invalid jump destination")

	// ErrGasUintOverflow is returned when gas uint64 is overflow.
	ErrGasUintOverflow = errors.New("gas uint64 overflow")
)
