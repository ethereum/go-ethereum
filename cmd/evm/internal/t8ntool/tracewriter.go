// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package t8ntool

import (
	"encoding/json"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/directory"
	"github.com/ethereum/go-ethereum/log"
)

// traceWriter is an vm.EVMLogger which also holds an inner logger/tracer.
// When the TxEnd event happens, the inner tracer result is written to the file, and
// the file is closed.
type traceWriter struct {
	inner vm.EVMLogger
	f     io.WriteCloser
}

// Compile-time interface check
var _ = vm.EVMLogger((*traceWriter)(nil))

func (t *traceWriter) CaptureTxEnd(receipt *types.Receipt, err error) {
	t.inner.CaptureTxEnd(receipt, err)
	defer t.f.Close()

	if tracer, ok := t.inner.(directory.Tracer); ok {
		result, err := tracer.GetResult()
		if err != nil {
			log.Warn("Error in tracer", "err", err)
			return
		}
		err = json.NewEncoder(t.f).Encode(result)
		if err != nil {
			log.Warn("Error writing tracer output", "err", err)
			return
		}
	}
}

func (t *traceWriter) CaptureTxStart(env *vm.EVM, tx *types.Transaction, from common.Address) {
	t.inner.CaptureTxStart(env, tx, from)
}
func (t *traceWriter) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.inner.CaptureStart(from, to, create, input, gas, value)
}

func (t *traceWriter) CaptureEnd(output []byte, gasUsed uint64, err error, reverted bool) {
	t.inner.CaptureEnd(output, gasUsed, err, reverted)
}

func (t *traceWriter) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.inner.CaptureEnter(typ, from, to, input, gas, value)
}

func (t *traceWriter) CaptureExit(output []byte, gasUsed uint64, err error, reverted bool) {
	t.inner.CaptureExit(output, gasUsed, err, reverted)
}

func (t *traceWriter) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	t.inner.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
}
func (t *traceWriter) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	t.inner.CaptureFault(pc, op, gas, cost, scope, depth, err)
}

func (t *traceWriter) CaptureKeccakPreimage(hash common.Hash, data []byte) {
	t.inner.CaptureKeccakPreimage(hash, data)
}

func (t *traceWriter) OnGasChange(old, new uint64, reason vm.GasChangeReason) {
	t.inner.OnGasChange(old, new, reason)
}
