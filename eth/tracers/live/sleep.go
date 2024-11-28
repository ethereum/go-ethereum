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

	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"time"
)

func init() {
	tracers.LiveDirectory.Register("sleep", newSleeperTracer)
}

// sleeper is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type sleeper struct{}

func newSleeperTracer(_ json.RawMessage) (*tracing.Hooks, error) {
	t := &sleeper{}
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

func (t *sleeper) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
}

func (t *sleeper) OnFault(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
}

func (t *sleeper) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *sleeper) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
}

func (t *sleeper) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
}

func (t *sleeper) OnTxEnd(receipt *types.Receipt, err error) {
}

func (t *sleeper) OnBlockStart(ev tracing.BlockEvent) {
	fmt.Printf("sleeper going to sleep for 30 minutes\n")
	time.Sleep(30 * time.Minute)
	fmt.Printf("sleeper waking up\n")
}

func (t *sleeper) OnBlockEnd(err error) {
}

func (t *sleeper) OnSkippedBlock(ev tracing.BlockEvent) {}

func (t *sleeper) OnBlockchainInit(chainConfig *params.ChainConfig) {
}

func (t *sleeper) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
}

func (t *sleeper) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
}

func (t *sleeper) OnNonceChange(a common.Address, prev, new uint64) {
}

func (t *sleeper) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
}

func (t *sleeper) OnStorageChange(a common.Address, k, prev, new common.Hash) {
}

func (t *sleeper) OnLog(l *types.Log) {

}

func (t *sleeper) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
}
