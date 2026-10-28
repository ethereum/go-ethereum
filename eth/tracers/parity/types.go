// Copyright 2026 The go-ethereum Authors
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

package parity

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Result contains actual output even when no output family is selected.
type Result struct {
	Output    hexutil.Bytes                   `json:"output"`
	Trace     []*Frame                        `json:"trace"`
	StateDiff map[common.Address]*AccountDiff `json:"stateDiff"`
	VMTrace   *VMTrace                        `json:"vmTrace"`
}

// Frame is one preorder call-tree record. Location identifies its block and
// transaction, when the frame was recorded in one.
type Frame struct {
	Action       any      `json:"action"`
	Result       any      `json:"result"`
	Error        string   `json:"error,omitempty"`
	Subtraces    uint64   `json:"subtraces"`
	TraceAddress []uint64 `json:"traceAddress"`
	Type         string   `json:"type"`
	*Location
}

// Location identifies a frame's block and transaction.
type Location struct {
	BlockHash           common.Hash  `json:"blockHash"`
	BlockNumber         uint64       `json:"blockNumber"`
	TransactionHash     *common.Hash `json:"transactionHash"`
	TransactionPosition *uint64      `json:"transactionPosition"`
}

// CallAction is the action of a call frame.
type CallAction struct {
	CallType string         `json:"callType"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      hexutil.Uint64 `json:"gas"`
	Input    hexutil.Bytes  `json:"input"`
	Value    *hexutil.Big   `json:"value"`
}

// CreateAction is the action of a create frame.
type CreateAction struct {
	From           common.Address `json:"from"`
	Gas            hexutil.Uint64 `json:"gas"`
	Init           hexutil.Bytes  `json:"init"`
	Value          *hexutil.Big   `json:"value"`
	CreationMethod string         `json:"creationMethod"`
}

// SuicideAction is the action of a selfdestruct frame.
type SuicideAction struct {
	Address       common.Address `json:"address"`
	RefundAddress common.Address `json:"refundAddress"`
	Balance       *hexutil.Big   `json:"balance"`
}

// CallResult is the result of a successful or reverted call frame.
type CallResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Output  hexutil.Bytes  `json:"output"`
}

// CreateResult is the result of a successful create frame.
type CreateResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Address common.Address `json:"address"`
	Code    hexutil.Bytes  `json:"code"`
}

// AccountDiff is the transition of an account, each field a change marker.
type AccountDiff struct {
	Balance any                 `json:"balance"`
	Nonce   any                 `json:"nonce"`
	Code    any                 `json:"code"`
	Storage map[common.Hash]any `json:"storage"`
}

// VMTrace is the instruction trace of a frame that executed code.
type VMTrace struct {
	Code hexutil.Bytes `json:"code"`
	Ops  []*VMOp       `json:"ops"`
}

// VMOp is one executed instruction, with the trace of the frame it entered.
type VMOp struct {
	PC   uint64     `json:"pc"`
	Cost uint64     `json:"cost"`
	Ex   *VMEffects `json:"ex"`
	Sub  *VMTrace   `json:"sub"`
	Op   string     `json:"op"`
}

// VMEffects are the effects of an instruction that completed.
type VMEffects struct {
	Used  uint64         `json:"used"`
	Push  []*hexutil.Big `json:"push"`
	Mem   *MemoryWrite   `json:"mem"`
	Store *StorageWrite  `json:"store"`
}

// MemoryWrite is the memory range an instruction designates, after it ran.
type MemoryWrite struct {
	Off  uint64        `json:"off"`
	Data hexutil.Bytes `json:"data"`
}

// StorageWrite is the slot an SSTORE writes.
type StorageWrite struct {
	Key *hexutil.Big `json:"key"`
	Val *hexutil.Big `json:"val"`
}
