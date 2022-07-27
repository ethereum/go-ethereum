// Copyright 2022 The go-ethereum Authors
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

package native

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
	register("returnDataTracer", newReturnDataTracer)
}

type returnData struct {
	Error  string `json:"error,omitempty"`
	Reason string `json:"reason,omitempty"`
	Output string `json:"output"`
}

type returnDataTracer struct {
	env    *vm.EVM
	data   returnData
	reason error // Textual reason for the interruption
}

// newReturnDataTracer returns a native go tracer which reveals
// the return data of a tx, and implements vm.EVMLogger.
func newReturnDataTracer(_ *tracers.Context) tracers.Tracer {
	return &returnDataTracer{}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *returnDataTracer) CaptureStart(env *vm.EVM, _ common.Address, _ common.Address, _ bool, _ []byte, _ uint64, _ *big.Int) {
	t.env = env
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *returnDataTracer) CaptureEnd(output []byte, _ uint64, _ time.Duration, err error) {
	if err == nil {
		t.data.Output = bytesToHex(output)
	} else {
		t.data.Error = err.Error()
		if err == vm.ErrExecutionReverted && len(output) > 0 {
			t.data.Output = bytesToHex(output)
			if len(output) > 4 {
				unpacked, errUnpack := abi.UnpackRevert(output)
				if errUnpack == nil {
					t.data.Reason = unpacked
				}
			}
		}
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *returnDataTracer) CaptureState(_ uint64, _ vm.OpCode, _, _ uint64, _ *vm.ScopeContext, _ []byte, _ int, _ error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *returnDataTracer) CaptureFault(_ uint64, _ vm.OpCode, _, _ uint64, _ *vm.ScopeContext, _ int, _ error) {
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *returnDataTracer) CaptureEnter(_ vm.OpCode, _ common.Address, _ common.Address, _ []byte, _ uint64, _ *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *returnDataTracer) CaptureExit(_ []byte, _ uint64, _ error) {}

func (t *returnDataTracer) CaptureTxStart(_ uint64) {}

func (t *returnDataTracer) CaptureTxEnd(_ uint64) {}

// GetResult returns an error message json object.
func (t *returnDataTracer) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.data)
	if err != nil {
		return nil, err
	}
	return res, t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *returnDataTracer) Stop(err error) {
	t.reason = err
	t.env.Cancel()
}
