// Copyright 2025 The go-ethereum Authors
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

package tracers

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

// recoverTracer is a live tracer that acts as a proxy: it delegates all hook calls
// to a user‐defined ("child") tracer while catching any panics.
type recoverTracer struct {
	node  *node.Node
	child *tracing.Hooks
}

// NewRecoverTracer instantiates a recoverTracer and returns a wrapped tracing.Hooks.
// Only hook fields which are non-nil in the child are replaced with the corresponding
// recoverTracer method.
func NewRecoverTracer(node *node.Node, child *tracing.Hooks) (*tracing.Hooks, error) {
	if child == nil {
		return nil, fmt.Errorf("child tracer is nil")
	}
	rt := &recoverTracer{node: node, child: child}
	return rt.wrapHooks()
}

// wrapHooks creates a new hooks struct with safe wrappers
// for each non-nil hook in the child tracer.
func (rt *recoverTracer) wrapHooks() (*tracing.Hooks, error) {
	childVal := reflect.Indirect(reflect.ValueOf(rt.child))
	hookType := childVal.Type()

	// Create a new Hooks value.
	newHooks := reflect.New(hookType).Elem()
	// Get the recoverTracer's method set.
	rtVal := reflect.ValueOf(rt)

	// Iterate through each field of the Hooks struct.
	for i := 0; i < hookType.NumField(); i++ {
		field := hookType.Field(i)
		childField := childVal.Field(i)
		// If the child's hook is set, then wrap it.
		if !childField.IsNil() {
			// Look for the recoverTracer method with the same name.
			methodVal := rtVal.MethodByName(field.Name)
			// If the method exists and its type matches the hook's type, use it.
			if methodVal.IsValid() && methodVal.Type() == field.Type {
				fmt.Printf("Setting method %s for field %s\n", methodVal, field.Name)
				newHooks.Field(i).Set(methodVal)
			} else {
				return nil, fmt.Errorf("method %s not available on recovery tracer", field.Name)
			}
		}
	}

	// Return the wrapped Hooks.
	ret := newHooks.Addr().Interface().(*tracing.Hooks)
	return ret, nil
}

// safeCall is a helper wrapping a hook call in a defer/recover block.
// If the call panics, it logs the panic. If shutdown is true, it also initiates
// node shutdown.
func (rt *recoverTracer) safeCall(name string, shutdown bool, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(fmt.Sprintf("panic in child tracer during %s: %v", name, r))
			if shutdown {
				rt.node.Close()
			}
		}
	}()
	fn()
}

func (rt *recoverTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	rt.safeCall("OnTxStart", true, func() {
		rt.child.OnTxStart(vm, tx, from)
	})
}

func (rt *recoverTracer) OnTxEnd(receipt *types.Receipt, err error) {
	rt.safeCall("OnTxEnd", true, func() {
		rt.child.OnTxEnd(receipt, err)
	})
}

func (rt *recoverTracer) OnEnter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	rt.safeCall("OnEnter", true, func() {
		rt.child.OnEnter(depth, typ, from, to, input, gas, value)
	})
}

func (rt *recoverTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	rt.safeCall("OnExit", true, func() {
		rt.child.OnExit(depth, output, gasUsed, err, reverted)
	})
}

func (rt *recoverTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	rt.safeCall("OnOpcode", true, func() {
		rt.child.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
	})
}

func (rt *recoverTracer) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	rt.safeCall("OnFault", true, func() {
		rt.child.OnFault(pc, op, gas, cost, scope, depth, err)
	})
}

func (rt *recoverTracer) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	rt.safeCall("OnGasChange", true, func() {
		rt.child.OnGasChange(old, new, reason)
	})
}

func (rt *recoverTracer) OnBlockchainInit(chainConfig *params.ChainConfig) {
	rt.safeCall("OnBlockchainInit", true, func() {
		rt.child.OnBlockchainInit(chainConfig)
	})
}

func (rt *recoverTracer) OnClose() {
	// OnClose is on the critical path for node shutdown.
	// Capture any panic in child.OnClose, but do not re-trigger shutdown
	// to prevent potential recursive shutdown calls.
	rt.safeCall("OnClose", false, func() {
		rt.child.OnClose()
	})
}

func (rt *recoverTracer) OnBlockStart(event tracing.BlockEvent) {
	rt.safeCall("OnBlockStart", true, func() {
		rt.child.OnBlockStart(event)
	})
}

func (rt *recoverTracer) OnBlockEnd(err error) {
	rt.safeCall("OnBlockEnd", true, func() {
		rt.child.OnBlockEnd(err)
	})
}

func (rt *recoverTracer) OnSkippedBlock(event tracing.BlockEvent) {
	rt.safeCall("OnSkippedBlock", true, func() {
		rt.child.OnSkippedBlock(event)
	})
}

func (rt *recoverTracer) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	rt.safeCall("OnGenesisBlock", true, func() {
		rt.child.OnGenesisBlock(b, alloc)
	})
}

func (rt *recoverTracer) OnSystemCallStart() {
	rt.safeCall("OnSystemCallStart", true, func() {
		rt.child.OnSystemCallStart()
	})
}

func (rt *recoverTracer) OnSystemCallStartV2(ctx *tracing.VMContext) {
	rt.safeCall("OnSystemCallStartV2", true, func() {
		rt.child.OnSystemCallStartV2(ctx)
	})
}

func (rt *recoverTracer) OnSystemCallEnd() {
	rt.safeCall("OnSystemCallEnd", true, func() {
		rt.child.OnSystemCallEnd()
	})
}

func (rt *recoverTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	rt.safeCall("OnBalanceChange", true, func() {
		rt.child.OnBalanceChange(a, prev, new, reason)
	})
}

func (rt *recoverTracer) OnNonceChange(a common.Address, prev, new uint64) {
	rt.safeCall("OnNonceChange", true, func() {
		rt.child.OnNonceChange(a, prev, new)
	})
}

func (rt *recoverTracer) OnNonceChangeV2(a common.Address, prev, new uint64, reason tracing.NonceChangeReason) {
	rt.safeCall("OnNonceChangeV2", true, func() {
		rt.child.OnNonceChangeV2(a, prev, new, reason)
	})
}

func (rt *recoverTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	rt.safeCall("OnCodeChange", true, func() {
		rt.child.OnCodeChange(a, prevCodeHash, prevCode, codeHash, code)
	})
}

func (rt *recoverTracer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	rt.safeCall("OnStorageChange", true, func() {
		rt.child.OnStorageChange(a, k, prev, new)
	})
}

func (rt *recoverTracer) OnLog(l *types.Log) {
	rt.safeCall("OnLog", true, func() {
		rt.child.OnLog(l)
	})
}

func (rt *recoverTracer) OnBlockHashRead(blockNumber uint64, hash common.Hash) {
	rt.safeCall("OnBlockHashRead", true, func() {
		rt.child.OnBlockHashRead(blockNumber, hash)
	})
}
