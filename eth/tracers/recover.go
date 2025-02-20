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

// NewRecoverTracer instantiates a recoverTracer.
func NewRecoverTracer(node *node.Node, child *tracing.Hooks) (*tracing.Hooks, error) {
	if child == nil {
		return nil, fmt.Errorf("child tracer is nil")
	}
	rt := &recoverTracer{node: node, child: child}

	// Build and return the Hooks object with all hook functions set to our proxy methods.
	return &tracing.Hooks{
		OnTxStart:           rt.OnTxStart,
		OnTxEnd:             rt.OnTxEnd,
		OnEnter:             rt.OnEnter,
		OnExit:              rt.OnExit,
		OnOpcode:            rt.OnOpcode,
		OnFault:             rt.OnFault,
		OnGasChange:         rt.OnGasChange,
		OnBlockchainInit:    rt.OnBlockchainInit,
		OnClose:             rt.OnClose,
		OnBlockStart:        rt.OnBlockStart,
		OnBlockEnd:          rt.OnBlockEnd,
		OnSkippedBlock:      rt.OnSkippedBlock,
		OnGenesisBlock:      rt.OnGenesisBlock,
		OnSystemCallStart:   rt.OnSystemCallStart,
		OnSystemCallStartV2: rt.OnSystemCallStartV2,
		OnSystemCallEnd:     rt.OnSystemCallEnd,
		OnBalanceChange:     rt.OnBalanceChange,
		OnNonceChange:       rt.OnNonceChange,
		OnNonceChangeV2:     rt.OnNonceChangeV2,
		OnCodeChange:        rt.OnCodeChange,
		OnStorageChange:     rt.OnStorageChange,
		OnLog:               rt.OnLog,
		OnBlockHashRead:     rt.OnBlockHashRead,
	}, nil
}

// safeCall is a helper wrapping a hook call in a defer/recover.
// If the call panics, we log the panic (and later we'll initiate a graceful shutdown).
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
		if rt.child.OnTxStart != nil {
			rt.child.OnTxStart(vm, tx, from)
		}
	})
}

func (rt *recoverTracer) OnTxEnd(receipt *types.Receipt, err error) {
	rt.safeCall("OnTxEnd", true, func() {
		if rt.child.OnTxEnd != nil {
			rt.child.OnTxEnd(receipt, err)
		}
	})
}

func (rt *recoverTracer) OnEnter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	rt.safeCall("OnEnter", true, func() {
		if rt.child.OnEnter != nil {
			rt.child.OnEnter(depth, typ, from, to, input, gas, value)
		}
	})
}

func (rt *recoverTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	rt.safeCall("OnExit", true, func() {
		if rt.child.OnExit != nil {
			rt.child.OnExit(depth, output, gasUsed, err, reverted)
		}
	})
}

func (rt *recoverTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	rt.safeCall("OnOpcode", true, func() {
		if rt.child.OnOpcode != nil {
			rt.child.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
		}
	})
}

func (rt *recoverTracer) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	rt.safeCall("OnFault", true, func() {
		if rt.child.OnFault != nil {
			rt.child.OnFault(pc, op, gas, cost, scope, depth, err)
		}
	})
}

func (rt *recoverTracer) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	rt.safeCall("OnGasChange", true, func() {
		if rt.child.OnGasChange != nil {
			rt.child.OnGasChange(old, new, reason)
		}
	})
}

func (rt *recoverTracer) OnBlockchainInit(chainConfig *params.ChainConfig) {
	rt.safeCall("OnBlockchainInit", true, func() {
		if rt.child.OnBlockchainInit != nil {
			rt.child.OnBlockchainInit(chainConfig)
		}
	})
}

func (rt *recoverTracer) OnClose() {
	fmt.Printf("yooo on cloooosee\n")
	// OnClose is on the critical path for node shutdown. It will be called
	// if node is winding down normally or due to an earlier panic. Capture any
	// panic that might happen in child.OnClose but don't initiate a shutdown which can turn
	// into a recursive loop.
	rt.safeCall("OnClose", false, func() {
		if rt.child.OnClose != nil {
			rt.child.OnClose()
		}
	})
}

func (rt *recoverTracer) OnBlockStart(event tracing.BlockEvent) {
	rt.safeCall("OnBlockStart", true, func() {
		if rt.child.OnBlockStart != nil {
			rt.child.OnBlockStart(event)
		}
	})
}

func (rt *recoverTracer) OnBlockEnd(err error) {
	rt.safeCall("OnBlockEnd", true, func() {
		if rt.child.OnBlockEnd != nil {
			rt.child.OnBlockEnd(err)
		}
	})
}

func (rt *recoverTracer) OnSkippedBlock(event tracing.BlockEvent) {
	rt.safeCall("OnSkippedBlock", true, func() {
		if rt.child.OnSkippedBlock != nil {
			rt.child.OnSkippedBlock(event)
		}
	})
}

func (rt *recoverTracer) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	rt.safeCall("OnGenesisBlock", true, func() {
		if rt.child.OnGenesisBlock != nil {
			rt.child.OnGenesisBlock(b, alloc)
		}
	})
}

func (rt *recoverTracer) OnSystemCallStart() {
	rt.safeCall("OnSystemCallStart", true, func() {
		if rt.child.OnSystemCallStart != nil {
			rt.child.OnSystemCallStart()
		}
	})
}

func (rt *recoverTracer) OnSystemCallStartV2(ctx *tracing.VMContext) {
	rt.safeCall("OnSystemCallStartV2", true, func() {
		if rt.child.OnSystemCallStartV2 != nil {
			rt.child.OnSystemCallStartV2(ctx)
		}
	})
}

func (rt *recoverTracer) OnSystemCallEnd() {
	rt.safeCall("OnSystemCallEnd", true, func() {
		if rt.child.OnSystemCallEnd != nil {
			rt.child.OnSystemCallEnd()
		}
	})
}

func (rt *recoverTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	rt.safeCall("OnBalanceChange", true, func() {
		if rt.child.OnBalanceChange != nil {
			rt.child.OnBalanceChange(a, prev, new, reason)
		}
	})
}

func (rt *recoverTracer) OnNonceChange(a common.Address, prev, new uint64) {
	rt.safeCall("OnNonceChange", true, func() {
		if rt.child.OnNonceChange != nil {
			rt.child.OnNonceChange(a, prev, new)
		}
	})
}

func (rt *recoverTracer) OnNonceChangeV2(a common.Address, prev, new uint64, reason tracing.NonceChangeReason) {
	rt.safeCall("OnNonceChangeV2", true, func() {
		if rt.child.OnNonceChangeV2 != nil {
			rt.child.OnNonceChangeV2(a, prev, new, reason)
		}
	})
}

func (rt *recoverTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	rt.safeCall("OnCodeChange", true, func() {
		if rt.child.OnCodeChange != nil {
			rt.child.OnCodeChange(a, prevCodeHash, prevCode, codeHash, code)
		}
	})
}

func (rt *recoverTracer) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	rt.safeCall("OnStorageChange", true, func() {
		if rt.child.OnStorageChange != nil {
			rt.child.OnStorageChange(a, k, prev, new)
		}
	})
}

func (rt *recoverTracer) OnLog(l *types.Log) {
	rt.safeCall("OnLog", true, func() {
		if rt.child.OnLog != nil {
			rt.child.OnLog(l)
		}
	})
}

func (rt *recoverTracer) OnBlockHashRead(blockNumber uint64, hash common.Hash) {
	rt.safeCall("OnBlockHashRead", true, func() {
		if rt.child.OnBlockHashRead != nil {
			rt.child.OnBlockHashRead(blockNumber, hash)
		}
	})
}
