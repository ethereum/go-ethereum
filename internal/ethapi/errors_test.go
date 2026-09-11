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

package ethapi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
)

// rpcErrorCoder mirrors the rpc package's interface for errors that carry a
// JSON-RPC error code.
type rpcErrorCoder interface {
	error
	ErrorCode() int
}

// TestTxSubmitError verifies that transaction-submission errors are mapped to
// the standardized execution-apis JSON-RPC error codes while preserving the
// original error message, and that unmapped errors are returned unchanged.
func TestTxSubmitError(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code int
	}{
		{"nonce too low", core.ErrNonceTooLow, errCodeStdNonceTooLow},
		{"nonce too high", core.ErrNonceTooHigh, errCodeStdNonceTooHigh},
		{"intrinsic gas", core.ErrIntrinsicGas, errCodeStdIntrinsicGas},
		{"gas price too low", txpool.ErrTxGasPriceTooLow, errCodeStdGasPriceTooLow},
		{"exceeds block gas limit", txpool.ErrGasLimit, errCodeStdGasExceedsBlockLimit},
		{"tip above fee cap", core.ErrTipAboveFeeCap, errCodeStdTipAboveFeeCap},
		{"gas uint overflow", core.ErrGasUintOverflow, errCodeStdGasUintOverflow},
		{"fee cap too low", core.ErrFeeCapTooLow, errCodeStdFeeCapTooLow},
		{"tip very high", core.ErrTipVeryHigh, errCodeStdTipVeryHigh},
		{"fee cap very high", core.ErrFeeCapVeryHigh, errCodeStdFeeCapVeryHigh},
		{"insufficient funds", core.ErrInsufficientFunds, errCodeStdInsufficientFunds},
		{"insufficient funds for transfer", core.ErrInsufficientFundsForTransfer, errCodeStdInsufficientFunds},
		{"already known", txpool.ErrAlreadyKnown, errCodeStdAlreadyKnown},
		{"invalid sender", txpool.ErrInvalidSender, errCodeStdInvalidSender},
		{"replacement underpriced", txpool.ErrReplaceUnderpriced, errCodeStdReplaceUnderpriced},
	} {
		// The raw sentinel error and a wrapped variant (as the pool/state
		// actually return them, e.g. "nonce too low: next nonce 5, tx nonce 0")
		// must both map to the same code via errors.Is.
		for _, err := range []error{tt.err, fmt.Errorf("%w: extra context", tt.err)} {
			got := txSubmitError(err)
			coder, ok := got.(rpcErrorCoder)
			if !ok {
				t.Fatalf("%s: txSubmitError(%q) = %T, want an error with ErrorCode()", tt.name, err, got)
			}
			if coder.ErrorCode() != tt.code {
				t.Errorf("%s: code = %d, want %d", tt.name, coder.ErrorCode(), tt.code)
			}
			if got.Error() != err.Error() {
				t.Errorf("%s: message = %q, want it preserved as %q", tt.name, got.Error(), err.Error())
			}
		}
	}
}

// TestTxSubmitErrorPassthrough verifies that errors without a catalog code are
// returned unchanged (so they keep geth's default -32000 behavior) and that a
// nil error stays nil.
func TestTxSubmitErrorPassthrough(t *testing.T) {
	if got := txSubmitError(nil); got != nil {
		t.Fatalf("txSubmitError(nil) = %v, want nil", got)
	}
	unmapped := errors.New("some unrelated failure")
	got := txSubmitError(unmapped)
	if got != unmapped {
		t.Fatalf("txSubmitError(unmapped) = %v, want the original error returned unchanged", got)
	}
	if _, ok := got.(rpcErrorCoder); ok {
		t.Fatalf("unmapped error should not carry an ErrorCode()")
	}
}
