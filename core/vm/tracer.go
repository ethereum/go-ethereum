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

package vm

import (
	"math/big"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type TracingEnabled byte
type TracingDisabled uint16

type TracingSwitch interface {
	TracingEnabled | TracingDisabled
}

func tracingEnabled[TS TracingSwitch]() bool {
	var t TS
	return unsafe.Sizeof(t) == 1
}

// tracer is a wrapper that gives nil-safe access to a tracing.Hooks
// and also enables VM tracing to be disabled at compile time
type tracer[TS TracingSwitch] struct{ hooks *tracing.Hooks }

func (t tracer[TS]) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnTxStart != nil {
		t.hooks.OnTxStart(vm, tx, from)
	}
}

func (t tracer[TS]) OnTxEnd(receipt *types.Receipt, err error) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnTxEnd != nil {
		t.hooks.OnTxEnd(receipt, err)
	}
}

func (t tracer[TS]) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnEnter != nil {
		t.hooks.OnEnter(depth, typ, from, to, input, gas, value)
	}
}

func (t tracer[TS]) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnExit != nil {
		t.hooks.OnExit(depth, output, gasUsed, err, reverted)
	}
}

func (t tracer[TS]) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnOpcode != nil {
		t.hooks.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
	}
}

func (t tracer[TS]) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnFault != nil {
		t.hooks.OnFault(pc, op, gas, cost, scope, depth, err)
	}
}

func (t tracer[TS]) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnGasChange != nil {
		t.hooks.OnGasChange(old, new, reason)
	}
}

func (t tracer[TS]) OnBlockchainInit(chainConfig *params.ChainConfig) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnBlockchainInit != nil {
		t.hooks.OnBlockchainInit(chainConfig)
	}
}

func (t tracer[TS]) OnClose() {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnClose != nil {
		t.hooks.OnClose()
	}
}

func (t tracer[TS]) OnBlockStart(event tracing.BlockEvent) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnBlockStart != nil {
		t.hooks.OnBlockStart(event)
	}
}

func (t tracer[TS]) OnBlockEnd(err error) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnBlockEnd != nil {
		t.hooks.OnBlockEnd(err)
	}
}

func (t tracer[TS]) OnSkippedBlock(event tracing.BlockEvent) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnSkippedBlock != nil {
		t.hooks.OnSkippedBlock(event)
	}
}

func (t tracer[TS]) OnGenesisBlock(genesis *types.Block, alloc types.GenesisAlloc) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnGenesisBlock != nil {
		t.hooks.OnGenesisBlock(genesis, alloc)
	}
}

func (t tracer[TS]) OnSystemCallStart() {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnSystemCallStart != nil {
		t.hooks.OnSystemCallStart()
	}
}

func (t tracer[TS]) OnSystemCallStartV2(vm *tracing.VMContext) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnSystemCallStartV2 != nil {
		t.hooks.OnSystemCallStartV2(vm)
	}
}

func (t tracer[TS]) OnSystemCallEnd() {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnSystemCallEnd != nil {
		t.hooks.OnSystemCallEnd()
	}
}

func (t tracer[TS]) OnBalanceChange(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnBalanceChange != nil {
		t.hooks.OnBalanceChange(addr, prev, new, reason)
	}
}

func (t tracer[TS]) OnNonceChange(addr common.Address, prev, new uint64) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnNonceChange != nil {
		t.hooks.OnNonceChange(addr, prev, new)
	}
}

func (t tracer[TS]) OnNonceChangeV2(addr common.Address, prev, new uint64, reason tracing.NonceChangeReason) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnNonceChangeV2 != nil {
		t.hooks.OnNonceChangeV2(addr, prev, new, reason)
	}
}

func (t tracer[TS]) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnCodeChange != nil {
		t.hooks.OnCodeChange(addr, prevCodeHash, prevCode, codeHash, code)
	}
}

func (t tracer[TS]) OnStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnStorageChange != nil {
		t.hooks.OnStorageChange(addr, slot, prev, new)
	}
}

func (t tracer[TS]) OnLog(log *types.Log) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnLog != nil {
		t.hooks.OnLog(log)
	}
}

func (t tracer[TS]) OnBlockHashRead(blockNumber uint64, hash common.Hash) {
	if tracingEnabled[TS]() && t.hooks != nil && t.hooks.OnBlockHashRead != nil {
		t.hooks.OnBlockHashRead(blockNumber, hash)
	}
}
