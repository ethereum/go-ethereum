// Copyright 2024 The go-ethereum Authors
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

package ethapi

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
)

// revertError is an API error that encompasses an EVM revert with JSON error
// code and a binary data blob.
type revertError struct {
	error
	reason string // revert reason hex encoded
}

type txSyncTimeoutError struct {
	msg  string
	hash common.Hash
}

// ErrorCode returns the JSON error code for a revert.
// See: https://ethereum.org/en/developers/docs/apis/json-rpc/#error-codes
func (e *revertError) ErrorCode() int {
	return 3
}

// ErrorData returns the hex encoded revert reason.
func (e *revertError) ErrorData() interface{} {
	return e.reason
}

// newRevertError creates a revertError instance with the provided revert data.
func newRevertError(revert []byte) *revertError {
	err := vm.ErrExecutionReverted

	reason, errUnpack := abi.UnpackRevert(revert)
	if errUnpack == nil {
		err = fmt.Errorf("%w: %v", vm.ErrExecutionReverted, reason)
	}
	return &revertError{
		error:  err,
		reason: hexutil.Encode(revert),
	}
}

// TxIndexingError is an API error that indicates the transaction indexing is not
// fully finished yet with JSON error code and a binary data blob.
type TxIndexingError struct{}

// NewTxIndexingError creates a TxIndexingError instance.
func NewTxIndexingError() *TxIndexingError { return &TxIndexingError{} }

// Error implement error interface, returning the error message.
func (e *TxIndexingError) Error() string {
	return "transaction indexing is in progress"
}

// ErrorCode returns the JSON error code for a revert.
// See: https://ethereum.org/en/developers/docs/apis/json-rpc/#error-codes
func (e *TxIndexingError) ErrorCode() int {
	return -32000 // to be decided
}

// ErrorData returns the hex encoded revert reason.
func (e *TxIndexingError) ErrorData() interface{} { return "transaction indexing is in progress" }

type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
}

type invalidTxError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *invalidTxError) Error() string  { return e.Message }
func (e *invalidTxError) ErrorCode() int { return e.Code }

// JSON-RPC error codes of the eth namespace, including the eth_simulateV1
// transaction validation codes.
const (
	ErrCodeNonceTooHigh            = -38011
	ErrCodeNonceTooLow             = -38010
	ErrCodeBaseFeeTooLow           = -38012
	ErrCodeIntrinsicGas            = -38013
	ErrCodeInsufficientFunds       = -38014
	ErrCodeBlockGasLimitReached    = -38015
	ErrCodeBlockNumberInvalid      = -38020
	ErrCodeBlockTimestampInvalid   = -38021
	ErrCodeSenderIsNotEOA          = -38024
	ErrCodeMaxInitCodeSizeExceeded = -38025
	ErrCodeClientLimitExceeded     = -38026
	ErrCodeInternalError           = -32603
	ErrCodeInvalidParams           = -32602
	ErrCodeVMError                 = -32015
	ErrCodeTxSyncTimeout           = 4
)

func txValidationError(err error) *invalidTxError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeSenderIsNotEOA}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeBaseFeeTooLow}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeIntrinsicGas}
	case errors.Is(err, core.ErrInsufficientFundsForTransfer):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeInsufficientFunds}
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		return &invalidTxError{Message: err.Error(), Code: ErrCodeMaxInitCodeSizeExceeded}
	}
	return &invalidTxError{
		Message: err.Error(),
		Code:    ErrCodeInternalError,
	}
}

type invalidParamsError struct{ message string }

func (e *invalidParamsError) Error() string  { return e.message }
func (e *invalidParamsError) ErrorCode() int { return ErrCodeInvalidParams }

type clientLimitExceededError struct{ message string }

func (e *clientLimitExceededError) Error() string  { return e.message }
func (e *clientLimitExceededError) ErrorCode() int { return ErrCodeClientLimitExceeded }

type invalidBlockNumberError struct{ message string }

func (e *invalidBlockNumberError) Error() string  { return e.message }
func (e *invalidBlockNumberError) ErrorCode() int { return ErrCodeBlockNumberInvalid }

type invalidBlockTimestampError struct{ message string }

func (e *invalidBlockTimestampError) Error() string  { return e.message }
func (e *invalidBlockTimestampError) ErrorCode() int { return ErrCodeBlockTimestampInvalid }

type blockGasLimitReachedError struct{ message string }

func (e *blockGasLimitReachedError) Error() string  { return e.message }
func (e *blockGasLimitReachedError) ErrorCode() int { return ErrCodeBlockGasLimitReached }

func (e *txSyncTimeoutError) Error() string          { return e.msg }
func (e *txSyncTimeoutError) ErrorCode() int         { return ErrCodeTxSyncTimeout }
func (e *txSyncTimeoutError) ErrorData() interface{} { return e.hash.Hex() }
