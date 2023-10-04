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

const (
	errCodeNonceTooHigh      = -38011
	errCodeNonceTooLow       = -38010
	errCodeInsufficientFunds = -38014
	errCodeIntrinsicGas      = -38013
	errCodeInternalError     = -32603
	errCodeInvalidParams     = -32602
)

func callErrorFromError(err error) *callError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		return &callError{Message: err.Error(), Code: errCodeNonceTooHigh}
	case errors.Is(err, core.ErrNonceTooLow):
		return &callError{Message: err.Error(), Code: errCodeNonceTooLow}
	case errors.Is(err, core.ErrSenderNoEOA):
		// TODO
	case errors.Is(err, core.ErrFeeCapVeryHigh):
		return &callError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipVeryHigh):
		return &callError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrTipAboveFeeCap):
		return &callError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrFeeCapTooLow):
		// TODO
		return &callError{Message: err.Error(), Code: errCodeInvalidParams}
	case errors.Is(err, core.ErrInsufficientFunds):
		return &callError{Message: err.Error(), Code: errCodeInsufficientFunds}
	case errors.Is(err, core.ErrIntrinsicGas):
		return &callError{Message: err.Error(), Code: errCodeIntrinsicGas}
	case errors.Is(err, core.ErrInsufficientFundsForTransfer):
		return &callError{Message: err.Error(), Code: errCodeInsufficientFunds}
	}
	return &callError{
		Message: err.Error(),
		Code:    errCodeInternalError,
	}
}
