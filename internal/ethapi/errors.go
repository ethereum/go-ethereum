// Copyright 2023 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/core"
)

type callError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type invalidTxError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *invalidTxError) Error() string  { return e.Message }
func (e *invalidTxError) ErrorCode() int { return e.Code }

const (
	errCodeNonceTooHigh            = -38011
	errCodeNonceTooLow             = -38010
	errCodeIntrinsicGas            = -38013
	errCodeInsufficientFunds       = -38014
	errCodeBlockGasLimitReached    = -38015
	errCodeBlockNumberInvalid      = -38020
	errCodeBlockTimestampInvalid   = -38021
	errCodeSenderIsNotEOA          = -38024
	errCodeMaxInitCodeSizeExceeded = -38025
	errCodeClientLimitExceeded     = -38026
	errCodeInternalError           = -32603
	errCodeInvalidParams           = -32602
	errCodeReverted                = -32000
	errCodeVMError                 = -32015
)

func txValidationError(err error) *invalidTxError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &invalidTxError{Message: err.Error(), Code: errCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &invalidTxError{Message: err.Error(), Code: errCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		return &invalidTxError{Message: err.Error(), Code: errCodeSenderIsNotEOA}
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &invalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &invalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &invalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapTooLow):
		return &invalidTxError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &invalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &invalidTxError{Message: err.Error(), Code: errCodeIntrinsicGas}
	case errors.Is(err, core.ErrInsufficientFundsForTransfer):
		return &invalidTxError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrMaxInitCodeSizeExceeded):
		return &invalidTxError{Message: err.Error(), Code: errCodeMaxInitCodeSizeExceeded}
	}
	return &invalidTxError{
		Message: err.Error(),
		Code:    errCodeInternalError,
	}
}

type invalidParamsError struct{ message string }

func (e *invalidParamsError) Error() string  { return e.message }
func (e *invalidParamsError) ErrorCode() int { return errCodeInvalidParams }

type clientLimitExceededError struct{ message string }

func (e *clientLimitExceededError) Error() string  { return e.message }
func (e *clientLimitExceededError) ErrorCode() int { return errCodeClientLimitExceeded }

type invalidBlockNumberError struct{ message string }

func (e *invalidBlockNumberError) Error() string  { return e.message }
func (e *invalidBlockNumberError) ErrorCode() int { return errCodeBlockNumberInvalid }

type invalidBlockTimestampError struct{ message string }

func (e *invalidBlockTimestampError) Error() string  { return e.message }
func (e *invalidBlockTimestampError) ErrorCode() int { return errCodeBlockTimestampInvalid }

type blockGasLimitReachedError struct{ message string }

func (e *blockGasLimitReachedError) Error() string  { return e.message }
func (e *blockGasLimitReachedError) ErrorCode() int { return errCodeBlockGasLimitReached }
