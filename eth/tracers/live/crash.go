// Copyright 2024 The go-ethereum Authors
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

package live

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	tracers.LiveDirectory.Register("crash", newCrashTracer)
}

// crash is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type crash struct{}

func newCrashTracer(_ json.RawMessage) (*tracing.Hooks, error) {
	t := &crash{}
	return &tracing.Hooks{
		OnTxStart:        t.OnTxStart,
		OnTxEnd:          t.OnTxEnd,
		OnEnter:          t.OnEnter,
		OnExit:           t.OnExit,
		OnOpcode:         t.OnOpcode,
		OnFault:          t.OnFault,
		OnGasChange:      t.OnGasChange,
		OnBlockchainInit: t.OnBlockchainInit,
		OnBlockStart:     t.OnBlockStart,
		OnBlockEnd:       t.OnBlockEnd,
		OnSkippedBlock:   t.OnSkippedBlock,
		OnGenesisBlock:   t.OnGenesisBlock,
		OnBalanceChange:  t.OnBalanceChange,
		OnNonceChange:    t.OnNonceChange,
		OnCodeChange:     t.OnCodeChange,
		OnStorageChange:  t.OnStorageChange,
		OnLog:            t.OnLog,
	}, nil
}

func (t *crash) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	panic("OnOpcode")
}

func (t *crash) OnFault(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
	panic("OnFault")
}

func (t *crash) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	panic("OnEnter")
}

func (t *crash) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	panic("OnExit")
}

func (t *crash) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	panic("OnTxStart")
}

func (t *crash) OnTxEnd(receipt *types.Receipt, err error) {
	panic("OnTxEnd")
}

func (t *crash) OnBlockStart(ev tracing.BlockEvent) {
	panic("OnBlockStart")
}

func (t *crash) OnBlockEnd(err error) {
	panic("OnBlockEnd")
}

func (t *crash) OnSkippedBlock(ev tracing.BlockEvent) {
	panic("OnSkippedBlock")
}

func (t *crash) OnBlockchainInit(chainConfig *params.ChainConfig) {
	panic("OnBlockchainInit")
}

func (t *crash) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	panic("OnGenesisBlock")
}

func (t *crash) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	panic("OnBalanceChange")
}

func (t *crash) OnNonceChange(a common.Address, prev, new uint64) {
	panic("OnNonceChange")
}

func (t *crash) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	panic("OnCodeChange")
}

func (t *crash) OnStorageChange(a common.Address, k, prev, new common.Hash) {
	panic("OnStorageChange")
}

func (t *crash) OnLog(l *types.Log) {
	panic("OnLog")
}

func (t *crash) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	panic("OnGasChange")
}
