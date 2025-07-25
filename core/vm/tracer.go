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

const tracingEnabledSize = 1

type TracingEnabled byte
type TracingDisabled uint16

type TracingSwitch interface {
	TracingEnabled | TracingDisabled
}

// tracer is a wrapper that gives nil-safe access to a tracing.Hooks
// and also enables VM tracing to be disabled at compile time
type tracer[TS TracingSwitch] struct {
	hooks tracing.Hooks
}

func NewTracer[TS TracingSwitch](hooks *tracing.Hooks) tracer[TS] {
	var t tracer[TS]
	if hooks != nil {
		t.hooks = *hooks
	}
	return t
}

func (t *tracer[TS]) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnTxStart != nil {
		t.hooks.OnTxStart(vm, tx, from)
	}
}

func (t *tracer[TS]) OnTxEnd(receipt *types.Receipt, err error) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnTxEnd != nil {
		t.hooks.OnTxEnd(receipt, err)
	}
}

func (t *tracer[TS]) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnEnter != nil {
		t.hooks.OnEnter(depth, typ, from, to, input, gas, value)
	}
}

func (t *tracer[TS]) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnExit != nil {
		t.hooks.OnExit(depth, output, gasUsed, err, reverted)
	}
}

func (t *tracer[TS]) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnOpcode != nil {
		t.hooks.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
	}
}

func (t *tracer[TS]) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnFault != nil {
		t.hooks.OnFault(pc, op, gas, cost, scope, depth, err)
	}
}

func (t *tracer[TS]) OnGasChange(oldGas, newGas uint64, reason tracing.GasChangeReason) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnGasChange != nil {
		t.hooks.OnGasChange(oldGas, newGas, reason)
	}
}

func (t *tracer[TS]) OnBlockchainInit(chainConfig *params.ChainConfig) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnBlockchainInit != nil {
		t.hooks.OnBlockchainInit(chainConfig)
	}
}

func (t *tracer[TS]) OnClose() {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnClose != nil {
		t.hooks.OnClose()
	}
}

func (t *tracer[TS]) OnBlockStart(event tracing.BlockEvent) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnBlockStart != nil {
		t.hooks.OnBlockStart(event)
	}
}

func (t *tracer[TS]) OnBlockEnd(err error) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnBlockEnd != nil {
		t.hooks.OnBlockEnd(err)
	}
}

func (t *tracer[TS]) OnSkippedBlock(event tracing.BlockEvent) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnSkippedBlock != nil {
		t.hooks.OnSkippedBlock(event)
	}
}

func (t *tracer[TS]) OnGenesisBlock(genesis *types.Block, alloc types.GenesisAlloc) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnGenesisBlock != nil {
		t.hooks.OnGenesisBlock(genesis, alloc)
	}
}

func (t *tracer[TS]) OnSystemCallStart() {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnSystemCallStart != nil {
		t.hooks.OnSystemCallStart()
	}
}

func (t *tracer[TS]) OnSystemCallStartV2(vm *tracing.VMContext) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnSystemCallStartV2 != nil {
		t.hooks.OnSystemCallStartV2(vm)
	}
}

func (t *tracer[TS]) OnSystemCallEnd() {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnSystemCallEnd != nil {
		t.hooks.OnSystemCallEnd()
	}
}

func (t *tracer[TS]) OnBalanceChange(addr common.Address, prevBalance, newBalance *big.Int, reason tracing.BalanceChangeReason) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnBalanceChange != nil {
		t.hooks.OnBalanceChange(addr, prevBalance, newBalance, reason)
	}
}

func (t *tracer[TS]) OnNonceChange(addr common.Address, prevNonce, newNonce uint64) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnNonceChange != nil {
		t.hooks.OnNonceChange(addr, prevNonce, newNonce)
	}
}

func (t *tracer[TS]) OnNonceChangeV2(addr common.Address, prevNonce, newNonce uint64, reason tracing.NonceChangeReason) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnNonceChangeV2 != nil {
		t.hooks.OnNonceChangeV2(addr, prevNonce, newNonce, reason)
	}
}

func (t *tracer[TS]) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnCodeChange != nil {
		t.hooks.OnCodeChange(addr, prevCodeHash, prevCode, codeHash, code)
	}
}

func (t *tracer[TS]) OnStorageChange(addr common.Address, slot common.Hash, prevValue, newValue common.Hash) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnStorageChange != nil {
		t.hooks.OnStorageChange(addr, slot, prevValue, newValue)
	}
}

func (t *tracer[TS]) OnLog(log *types.Log) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnLog != nil {
		t.hooks.OnLog(log)
	}
}

func (t *tracer[TS]) OnBlockHashRead(blockNumber uint64, hash common.Hash) {
	if unsafe.Sizeof(*new(TS)) == tracingEnabledSize && t.hooks.OnBlockHashRead != nil {
		t.hooks.OnBlockHashRead(blockNumber, hash)
	}
}
