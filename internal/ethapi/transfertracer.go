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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// keccak256("Transfer(address,address,uint256)")
var transferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

// tracer is a simple tracer that records all logs and
// ether transfers. Transfers are recorded as if they
// were logs. Transfer events include:
// - tx value
// - call value
// - self destructs
//
// The log format for a transfer is:
// - address: 0x0000000000000000000000000000000000000000
// - data: Value
// - topics:
//   - Transfer(address,address,uint256)
//   - Sender address
//   - Recipient address
//
// TODO: embed noopTracer
type tracer struct {
	logs           []*types.Log
	traceTransfers bool
	// TODO: replace with tracers.Context once extended tracer PR is merged.
	blockNumber uint64
	blockHash   common.Hash
	txHash      common.Hash
	txIdx       uint
}

func newTracer(traceTransfers bool, blockNumber uint64, blockHash, txHash common.Hash, txIdx uint) *tracer {
	return &tracer{
		logs:           make([]*types.Log, 0),
		traceTransfers: traceTransfers,
		blockNumber:    blockNumber,
		blockHash:      blockHash,
		txHash:         txHash,
		txIdx:          txIdx,
	}
}

func (t *tracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	if value.Cmp(common.Big0) > 0 {
		t.captureTransfer(from, to, value)
	}
}

func (t *tracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
}

func (t *tracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// skip if the previous op caused an error
	if err != nil {
		return
	}
	// TODO: Use OnLog instead of CaptureState once extended tracer PR is merged.
	switch op {
	case vm.LOG0, vm.LOG1, vm.LOG2, vm.LOG3, vm.LOG4:
		size := int(op - vm.LOG0)

		stack := scope.Stack
		stackData := stack.Data()

		// Don't modify the stack
		mStart := stackData[len(stackData)-1]
		mSize := stackData[len(stackData)-2]
		topics := make([]common.Hash, size)
		for i := 0; i < size; i++ {
			topic := stackData[len(stackData)-2-(i+1)]
			topics[i] = common.Hash(topic.Bytes32())
		}

		data, err := getMemoryCopyPadded(scope.Memory, int64(mStart.Uint64()), int64(mSize.Uint64()))
		if err != nil {
			// mSize was unrealistically large
			return
		}
		t.captureLog(scope.Contract.Address(), topics, data)
	}
}

func (t *tracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

func (t *tracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	toCopy := to
	if value != nil && value.Cmp(common.Big0) > 0 {
		t.captureTransfer(from, toCopy, value)
	}
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *tracer) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (t *tracer) CaptureTxStart(gasLimit uint64) {}

func (t *tracer) CaptureTxEnd(restGas uint64) {}

func (t *tracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	t.logs = append(t.logs, &types.Log{
		Address:     address,
		Topics:      topics,
		Data:        data,
		BlockNumber: t.blockNumber,
		BlockHash:   t.blockHash,
		TxHash:      t.txHash,
		TxIndex:     t.txIdx,
		Index:       uint(len(t.logs)),
	})
}

func (t *tracer) captureTransfer(from, to common.Address, value *big.Int) {
	if !t.traceTransfers {
		return
	}
	topics := []common.Hash{
		transferTopic,
		common.BytesToHash(from.Bytes()),
		common.BytesToHash(to.Bytes()),
	}
	t.captureLog(common.Address{}, topics, common.BigToHash(value).Bytes())
}

func (t *tracer) Logs() []*types.Log {
	return t.logs
}

// TODO: remove once extended tracer PR is merged.
func getMemoryCopyPadded(m *vm.Memory, offset, size int64) ([]byte, error) {
	if offset < 0 || size < 0 {
		return nil, fmt.Errorf("offset or size must not be negative")
	}
	if int(offset+size) < m.Len() { // slice fully inside memory
		return m.GetCopy(offset, size), nil
	}
	paddingNeeded := int(offset+size) - m.Len()
	if paddingNeeded > 1024*1024 {
		return nil, fmt.Errorf("reached limit for padding memory slice: %d", paddingNeeded)
	}
	cpy := make([]byte, size)
	if overlap := int64(m.Len()) - offset; overlap > 0 {
		copy(cpy, m.GetPtr(offset, overlap))
	}
	return cpy, nil
}
