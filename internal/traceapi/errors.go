// Copyright 2026 The go-ethereum Authors
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

package traceapi

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb/database"
)

const (
	errCodeResourceNotFound = -32001
	errCodeTxRejected       = -32003

	// eth_sendRawTransaction error groups, returned by trace_rawTransaction.
	errCodeRawNonceTooLow       = 1
	errCodeRawNonceTooHigh      = 2
	errCodeRawIntrinsicGas      = 800
	errCodeRawTipAboveFeeCap    = 804
	errCodeRawFeeCapTooLow      = 806
	errCodeRawInsufficientFunds = 809
)

// errCodePrunedHistory is the code of history.PrunedHistoryError. Unavailable
// history is reported with this code but with the message of its cause.
var errCodePrunedHistory = new(history.PrunedHistoryError).ErrorCode()

type rpcError struct {
	code    int
	message string
}

func (e *rpcError) Error() string  { return e.message }
func (e *rpcError) ErrorCode() int { return e.code }

func invalidParams(format string, args ...any) error {
	return &rpcError{ethapi.ErrCodeInvalidParams, fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...any) error {
	return &rpcError{errCodeResourceNotFound, fmt.Sprintf(format, args...)}
}

func limitExceeded(format string, args ...any) error {
	return &rpcError{ethapi.ErrCodeClientLimitExceeded, fmt.Sprintf(format, args...)}
}

// itemError reports the zero-based index of the failing trace_callMany
// item in error.data.
type itemError struct {
	err   rpc.Error
	index int
}

func (e *itemError) Error() string  { return fmt.Sprintf("call %d: %v", e.index, e.err) }
func (e *itemError) ErrorCode() int { return e.err.ErrorCode() }
func (e *itemError) ErrorData() any { return map[string]int{"index": e.index} }

// callRejection maps a rejected unsigned call to the eth_simulateV1
// validation codes. Nonce and sender-code checks are skipped for unsigned
// calls, as in eth_call. Call objects invalid regardless of state are rejected
// with -32602 before execution; any other rejection of a well-formed call at
// the selected block, such as a blob fee cap below the blob base fee or a
// feature not active at that fork, is -32003 (Transaction rejected), as for
// trace_rawTransaction.
func callRejection(err error) error {
	code := errCodeTxRejected
	switch {
	case errors.Is(err, core.ErrFeeCapTooLow):
		code = ethapi.ErrCodeBaseFeeTooLow
	case errors.Is(err, core.ErrIntrinsicGas), errors.Is(err, core.ErrFloorDataGas):
		code = ethapi.ErrCodeIntrinsicGas
	case errors.Is(err, core.ErrInsufficientFunds), errors.Is(err, core.ErrInsufficientFundsForTransfer):
		code = ethapi.ErrCodeInsufficientFunds
	case errors.Is(err, vm.ErrMaxInitCodeSizeExceeded):
		code = ethapi.ErrCodeMaxInitCodeSizeExceeded
	}
	return &rpcError{code, err.Error()}
}

// rawRejection maps a rejected signed transaction to the
// eth_sendRawTransaction error groups, with -32003 for other rejections.
func rawRejection(err error) error {
	code := errCodeTxRejected
	switch {
	case errors.Is(err, core.ErrNonceTooLow):
		code = errCodeRawNonceTooLow
	case errors.Is(err, core.ErrNonceTooHigh):
		code = errCodeRawNonceTooHigh
	case errors.Is(err, core.ErrIntrinsicGas), errors.Is(err, core.ErrFloorDataGas):
		code = errCodeRawIntrinsicGas
	case errors.Is(err, core.ErrTipAboveFeeCap):
		code = errCodeRawTipAboveFeeCap
	case errors.Is(err, core.ErrFeeCapTooLow):
		code = errCodeRawFeeCapTooLow
	case errors.Is(err, core.ErrInsufficientFunds), errors.Is(err, core.ErrInsufficientFundsForTransfer):
		code = errCodeRawInsufficientFunds
	}
	return &rpcError{code, err.Error()}
}

// replayRejection reports a mined transaction that failed validation on replay.
func replayRejection(err error) error { return &rpcError{errCodeTxRejected, err.Error()} }

// stateError reports state the backend cannot provide, either when the
// state is opened or through a trie node missing during execution, as
// unavailable history.
func stateError(err error) error {
	var missing *trie.MissingNodeError
	if errors.Is(err, database.ErrStateUnavailable) || errors.As(err, &missing) {
		return &rpcError{errCodePrunedHistory, err.Error()}
	}
	return err
}
