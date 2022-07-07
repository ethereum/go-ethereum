// Copyright 2021 The go-ethereum Authors
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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

// proxyTracer is a go implementation of the Tracer interface which
// forwards events to other tracers.
type proxy struct {
	tracer    tracers.Tracer
	env       *vm.EVM
	stop      bool   // Stop passthrough
	interrupt uint32 // Atomic flag to signal execution interruption
	reason    error  // Textual reason for the interruption
}

// newNoopTracer returns a new noop tracer.
func newProxy(ctor ctorFn) ctorFn {
	p := new(proxy)
	return func(ctx *tracers.Context) tracers.Tracer {
		p.tracer = ctor(ctx)
		return p
	}
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (p *proxy) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	if p.interrupted() {
		return
	}
	p.env = env
	p.tracer.CaptureStart(env, from, to, create, input, gas, value)
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (p *proxy) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureEnd(output, gasUsed, t, err)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (p *proxy) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureState(pc, op, gas, cost, scope, rData, depth, err)
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (p *proxy) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureFault(pc, op, gas, cost, scope, depth, err)
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (p *proxy) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureEnter(typ, from, to, input, gas, value)
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (p *proxy) CaptureExit(output []byte, gasUsed uint64, err error) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureExit(output, gasUsed, err)
}

func (p *proxy) CaptureTxStart(gasLimit uint64) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureTxStart(gasLimit)
}

func (p *proxy) CaptureTxEnd(restGas uint64) {
	if p.interrupted() {
		return
	}
	p.tracer.CaptureTxEnd(restGas)
}

// GetResult returns an empty json object.
func (p *proxy) GetResult() (json.RawMessage, error) {
	res, err := p.tracer.GetResult()
	if err != nil {
		return res, err
	}
	return res, p.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (p *proxy) Stop(err error) {
	p.reason = err
	atomic.StoreUint32(&p.interrupt, 1)
}

func (p *proxy) interrupted() bool {
	// p.interrupt needs to be atomic because Stop called from other goroutines.
	// But we set stop in the execution goroutine so its thread-safe.
	// Fast track.
	if p.stop {
		return true
	}
	// Slow track.
	if atomic.LoadUint32(&p.interrupt) > 0 {
		if p.env != nil {
			// Stop evm execution at the first opportune time.
			p.env.Cancel()
			// Disable future hooks. Note there's no guarantee that no hooks
			// will be called after this.
			p.env.Config.Debug = false
		}
		p.stop = true
		return true
	}
	return false
}
