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
	// ErrOutOfGas is returned when
	ErrOutOfGas = errors.New("out of gas")

	// ErrCodeStoreOutOfGas is returned when
	ErrCodeStoreOutOfGas = errors.New("contract creation code storage out of gas")

	// ErrDepth is returned when
	ErrDepth = errors.New("max call depth exceeded")

	// ErrTraceLimitReached is returned when
	ErrTraceLimitReached = errors.New("the number of logs reached the specified limit")

	// ErrInsufficientBalance is returned when
	ErrInsufficientBalance = errors.New("insufficient balance for transfer")

	// ErrContractAddressCollision is returned when
	ErrContractAddressCollision = errors.New("contract address collision")

	// ErrNoCompatibleInterpreter is returned when
	ErrNoCompatibleInterpreter = errors.New("no compatible interpreter")

	// ErrBadPairingInput is returned if the bn256 pairing input is invalid.
	ErrBadPairingInput = errors.New("bad elliptic curve pairing size")

	// ErrWriteProtection is returned when
	ErrWriteProtection = errors.New("evm: write protection")

	// ErrReturnDataOutOfBounds is returned when
	ErrReturnDataOutOfBounds = errors.New("evm: return data out of bounds")

	// ErrExecutionReverted is returned when
	ErrExecutionReverted = errors.New("evm: execution reverted")

	// ErrMaxCodeSizeExceeded is returned when
	ErrMaxCodeSizeExceeded = errors.New("evm: max code size exceeded")

	// ErrInvalidJump is returned when
	ErrInvalidJump = errors.New("evm: invalid jump destination")

	// ErrGasUintOverflow is returned when
	ErrGasUintOverflow = errors.New("gas uint64 overflow")
)
