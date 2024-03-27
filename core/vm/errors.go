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

import (
	"errors"
	"fmt"
	"math"
)

// List evm execution errors
var (
	ErrOutOfGas                 = errors.New("out of gas")
	ErrCodeStoreOutOfGas        = errors.New("contract creation code storage out of gas")
	ErrDepth                    = errors.New("max call depth exceeded")
	ErrInsufficientBalance      = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision = errors.New("contract address collision")
	ErrExecutionReverted        = errors.New("execution reverted")
	ErrMaxCodeSizeExceeded      = errors.New("max code size exceeded")
	ErrMaxInitCodeSizeExceeded  = errors.New("max initcode size exceeded")
	ErrInvalidJump              = errors.New("invalid jump destination")
	ErrWriteProtection          = errors.New("write protection")
	ErrReturnDataOutOfBounds    = errors.New("return data out of bounds")
	ErrGasUintOverflow          = errors.New("gas uint64 overflow")
	ErrInvalidCode              = errors.New("invalid code: must not begin with 0xef")
	ErrNonceUintOverflow        = errors.New("nonce uint64 overflow")

	// errStopToken is an internal token indicating interpreter loop termination,
	// never returned to outside callers.
	errStopToken = errors.New("stop token")
)

// ErrStackUnderflow wraps an evm error when the items on the stack less
// than the minimal requirement.
type ErrStackUnderflow struct {
	stackLen int
	required int
}

func (e *ErrStackUnderflow) Error() string {
	return fmt.Sprintf("stack underflow (%d <=> %d)", e.stackLen, e.required)
}

// ErrStackOverflow wraps an evm error when the items on the stack exceeds
// the maximum allowance.
type ErrStackOverflow struct {
	stackLen int
	limit    int
}

func (e *ErrStackOverflow) Error() string {
	return fmt.Sprintf("stack limit reached %d (%d)", e.stackLen, e.limit)
}

// ErrInvalidOpCode wraps an evm error when an invalid opcode is encountered.
type ErrInvalidOpCode struct {
	opcode OpCode
}

func (e *ErrInvalidOpCode) Error() string { return fmt.Sprintf("invalid opcode: %s", e.opcode) }

// rpcError is the same interface as the one defined in rpc/errors.go
// but we do not want to depend on rpc package here so we redefine it.
//
// It's used to ensure that the VMError implements the RPC error interface.
type rpcError interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

var _ rpcError = (*VMError)(nil)

// VMError wraps a VM error with an additional stable error code. The error
// field is the original error that caused the VM error and must be one of the
// VM error defined at the top of this file.
//
// If the error is not one of the known error above, the error code will be
// set to VMErrorCodeUnknown.
type VMError struct {
	error
	code int
}

func VMErrorFromErr(err error) error {
	if err == nil {
		return nil
	}

	return &VMError{
		error: err,
		code:  vmErrorCodeFromErr(err),
	}
}

func (e *VMError) Error() string {
	return e.error.Error()
}

func (e *VMError) Unwrap() error {
	return e.error
}

func (e *VMError) ErrorCode() int {
	return e.code
}

const (
	// We start the error code at 1 so that we can use 0 later for some possible extension. There
	// is no unspecified value for the code today because it should always be set to a valid value
	// that could be VMErrorCodeUnknown if the error is not mapped to a known error code.

	VMErrorCodeOutOfGas = 1 + iota
	VMErrorCodeCodeStoreOutOfGas
	VMErrorCodeDepth
	VMErrorCodeInsufficientBalance
	VMErrorCodeContractAddressCollision
	VMErrorCodeExecutionReverted
	VMErrorCodeMaxCodeSizeExceeded
	VMErrorCodeInvalidJump
	VMErrorCodeWriteProtection
	VMErrorCodeReturnDataOutOfBounds
	VMErrorCodeGasUintOverflow
	VMErrorCodeInvalidCode
	VMErrorCodeNonceUintOverflow
	VMErrorCodeStackUnderflow
	VMErrorCodeStackOverflow
	VMErrorCodeInvalidOpCode

	// VMErrorCodeUnknown explicitly marks an error as unknown, this is useful when error is converted
	// from an actual `error` in which case if the mapping is not known, we can use this value to indicate that.
	VMErrorCodeUnknown = math.MaxInt - 1
)

func vmErrorCodeFromErr(err error) int {
	switch {
	case errors.Is(err, ErrOutOfGas):
		return VMErrorCodeOutOfGas
	case errors.Is(err, ErrCodeStoreOutOfGas):
		return VMErrorCodeCodeStoreOutOfGas
	case errors.Is(err, ErrDepth):
		return VMErrorCodeDepth
	case errors.Is(err, ErrInsufficientBalance):
		return VMErrorCodeInsufficientBalance
	case errors.Is(err, ErrContractAddressCollision):
		return VMErrorCodeContractAddressCollision
	case errors.Is(err, ErrExecutionReverted):
		return VMErrorCodeExecutionReverted
	case errors.Is(err, ErrMaxCodeSizeExceeded):
		return VMErrorCodeMaxCodeSizeExceeded
	case errors.Is(err, ErrInvalidJump):
		return VMErrorCodeInvalidJump
	case errors.Is(err, ErrWriteProtection):
		return VMErrorCodeWriteProtection
	case errors.Is(err, ErrReturnDataOutOfBounds):
		return VMErrorCodeReturnDataOutOfBounds
	case errors.Is(err, ErrGasUintOverflow):
		return VMErrorCodeGasUintOverflow
	case errors.Is(err, ErrInvalidCode):
		return VMErrorCodeInvalidCode
	case errors.Is(err, ErrNonceUintOverflow):
		return VMErrorCodeNonceUintOverflow

	default:
		// Dynamic errors
		if v := (*ErrStackUnderflow)(nil); errors.As(err, &v) {
			return VMErrorCodeStackUnderflow
		}

		if v := (*ErrStackOverflow)(nil); errors.As(err, &v) {
			return VMErrorCodeStackOverflow
		}

		if v := (*ErrInvalidOpCode)(nil); errors.As(err, &v) {
			return VMErrorCodeInvalidOpCode
		}

		return VMErrorCodeUnknown
	}
}
