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
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var (
	// keccak256("Transfer(address,address,uint256)")
	transferTopic = common.HexToHash("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	// ERC-7528
	transferAddress = common.HexToAddress("0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE")
)

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
type tracer struct {
	// logs keeps logs for all open call frames.
	// This lets us clear logs for failed calls.
	logs           [][]*types.Log
	count          int
	traceTransfers bool
	blockNumber    uint64
	blockHash      common.Hash
	txHash         common.Hash
	txIdx          uint
}

func newTracer(traceTransfers bool, blockNumber uint64, blockHash, txHash common.Hash, txIndex uint) *tracer {
	return &tracer{
		traceTransfers: traceTransfers,
		blockNumber:    blockNumber,
		blockHash:      blockHash,
		txHash:         txHash,
		txIdx:          txIndex,
	}
}

func (t *tracer) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnEnter: t.onEnter,
		OnExit:  t.onExit,
		OnLog:   t.onLog,
	}
}

func (t *tracer) onEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.logs = append(t.logs, make([]*types.Log, 0))
	if vm.OpCode(typ) != vm.DELEGATECALL && value != nil && value.Cmp(common.Big0) > 0 {
		t.captureTransfer(from, to, value)
	}
}

func (t *tracer) onExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth == 0 {
		t.onEnd(reverted)
		return
	}
	size := len(t.logs)
	if size <= 1 {
		return
	}
	// pop call
	call := t.logs[size-1]
	t.logs = t.logs[:size-1]
	size--

	// Clear logs if call failed.
	if !reverted {
		t.logs[size-1] = append(t.logs[size-1], call...)
	}
}

func (t *tracer) onEnd(reverted bool) {
	if reverted {
		t.logs[0] = nil
	}
}

func (t *tracer) onLog(log *types.Log) {
	t.captureLog(log.Address, log.Topics, log.Data)
}

func (t *tracer) captureLog(address common.Address, topics []common.Hash, data []byte) {
	t.logs[len(t.logs)-1] = append(t.logs[len(t.logs)-1], &types.Log{
		Address:     address,
		Topics:      topics,
		Data:        data,
		BlockNumber: t.blockNumber,
		BlockHash:   t.blockHash,
		TxHash:      t.txHash,
		TxIndex:     t.txIdx,
		Index:       uint(t.count),
	})
	t.count++
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
	t.captureLog(transferAddress, topics, common.BigToHash(value).Bytes())
}

// reset prepares the tracer for the next transaction.
func (t *tracer) reset(txHash common.Hash, txIdx uint) {
	t.logs = nil
	t.txHash = txHash
	t.txIdx = txIdx
}

func (t *tracer) Logs() []*types.Log {
	return t.logs[0]
}
