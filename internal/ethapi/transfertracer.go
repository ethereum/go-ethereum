// Copyright 2023 The go-ethereum Authors
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
)

type callLog struct {
	Address common.Address `json:"address"`
	Topics  []common.Hash  `json:"topics"`
	Data    hexutil.Bytes  `json:"data"`
}

type transfer struct {
	From  common.Address `json:"from"`
	To    common.Address `json:"to"`
	Value *big.Int       `json:"value"`
}

// tracer is a simple tracer that records all ether transfers.
// This includes tx value, call value, and self destructs.
type tracer struct {
	transfers []transfer
}

func newTracer() *tracer {
	return &tracer{transfers: make([]transfer, 0)}
}

func (t *tracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	if value.Cmp(common.Big0) > 0 {
		t.transfers = append(t.transfers, transfer{From: from, To: to, Value: value})
	}
}

func (t *tracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
}

func (t *tracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

func (t *tracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

func (t *tracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	toCopy := to
	if value.Cmp(common.Big0) > 0 {
		t.transfers = append(t.transfers, transfer{From: from, To: toCopy, Value: value})
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *tracer) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (t *tracer) CaptureTxStart(gasLimit uint64) {}

func (t *tracer) CaptureTxEnd(restGas uint64) {}

func (t *tracer) Transfers() []transfer {
	return t.transfers
}
