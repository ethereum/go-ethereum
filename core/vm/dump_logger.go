// Copyright 2017 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
)

func getFile(taskName string, blockNumber uint64, perFolder, perFile uint64) (*os.File, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current work dir failed: %w", err)
	}

	logPath := path.Join(cwd, taskName, strconv.FormatUint(blockNumber/perFolder, 10), strconv.FormatUint(blockNumber/perFile, 10)+".log")
	fmt.Printf("log path: %v, block: %v\n", logPath, blockNumber)
	if err := os.MkdirAll(path.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("mkdir for all parents [%v] failed: %w", path.Dir(logPath), err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return nil, fmt.Errorf("create file %s failed: %w", logPath, err)
	}
	return file, nil
}

type ParityTraceItemAction struct {
	CallType string         `json:"callType"`
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      hexutil.Uint64 `json:"gas"`
	Input    hexutil.Bytes  `json:"input"`
	Value    hexutil.Bytes  `json:"value"`
}

type ParityTraceItemResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Output  hexutil.Bytes  `json:"output"`
}

type ParityTraceItem struct {
	Type                 string                `json:"type"`
	Action               ParityTraceItemAction `json:"action"`
	Result               ParityTraceItemResult `json:"result"`
	Subtraces            int                   `json:"subtraces"`
	TraceAddress         []int                 `json:"traceAddress"`
	Error                string                `json:"error,omitempty"`
	BlockHash            common.Hash           `json:"blockHash"`
	BlockNumber          uint64                `json:"blockNumber"`
	TransactionHash      common.Hash           `json:"transactionHash"`
	TransactionPosition  int                   `json:"transactionPosition"`
	TransactionTraceID   int                   `json:"transactionTraceID"`
	TransactionLastTrace int                   `json:"transactionLastTrace"`
}

type ParityLogContext struct {
	BlockHash   common.Hash
	BlockNumber uint64
	TxPos       int
	TxHash      common.Hash
}

type ParityLogger struct {
	context           *ParityLogContext
	encoder           *json.Encoder
	activePrecompiles []common.Address
	file              *os.File
	stack             []*ParityTraceItem
	items             []*ParityTraceItem
}

// NewParityLogger creates a new EVM tracer that prints execution steps as parity trace format
// into the provided stream.
func NewParityLogger(ctx *ParityLogContext, blockNumber uint64, perFolder, perFile uint64) (*ParityLogger, error) {
	file, err := getFile("traces", blockNumber, perFolder, perFile)
	if err != nil {
		return nil, err
	}

	l := &ParityLogger{context: ctx, encoder: json.NewEncoder(file), file: file}
	if l.context == nil {
		l.context = &ParityLogContext{}
	}
	return l, nil
}

func (l *ParityLogger) Close() error {
	return l.file.Close()
}

func (l *ParityLogger) CaptureStart(env *EVM, from, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	//rules := env.ChainConfig().Rules(env.Context.BlockNumber)
	rules := env.ChainConfig().Rules(env.Context.BlockNumber, env.Context.Random != nil)
	l.activePrecompiles = ActivePrecompiles(rules)
	l.stack = make([]*ParityTraceItem, 0, 20)
	l.items = make([]*ParityTraceItem, 0, 20)
	if create {
		l.CaptureEnter(CREATE, from, to, input, gas, value)
	} else {
		l.CaptureEnter(CALL, from, to, input, gas, value)
	}
}

func (l *ParityLogger) CaptureTxStart(gasLimit uint64) {
}

func (l *ParityLogger) CaptureTxEnd(restGas uint64) {
}

func (l *ParityLogger) CaptureFault(uint64, OpCode, uint64, uint64, *ScopeContext, int, error) {
}

// CaptureState outputs state information on the logger.
func (l *ParityLogger) CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
}

// CaptureEnd is triggered at end of execution.
func (l *ParityLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	l.CaptureExit(output, gasUsed, err)
	itemsSize := len(l.items)
	for no, item := range l.items {
		item.TransactionTraceID = no
		if no+1 == itemsSize {
			item.TransactionLastTrace = 1
		} else {
			item.TransactionLastTrace = 0
		}
		l.encoder.Encode(item)
	}
}

func getTraceType(typ OpCode) string {
	if typ == CALL || typ == DELEGATECALL || typ == CALLCODE || typ == STATICCALL {
		return "call"
	} else if typ == CREATE || typ == CREATE2 {
		return "create"
	} else if typ == SELFDESTRUCT {
		return "suicide"
	} else {
		return "unknown"
	}
}

func (l *ParityLogger) CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	traceAddress := make([]int, 0, 5)
	if len(l.stack) > 0 {
		back := l.stack[len(l.stack)-1]
		traceAddress = append(traceAddress, back.TraceAddress...)
		traceAddress = append(traceAddress, back.Subtraces)
		back.Subtraces += 1
	}
	newItem := &ParityTraceItem{
		Type: getTraceType(typ),
		Action: ParityTraceItemAction{
			CallType: strings.ToLower(typ.String()),
			From:     from,
			To:       to,
			Gas:      hexutil.Uint64(gas),
			Input:    append([]byte{}, input...),
		},
		Result: ParityTraceItemResult{
			GasUsed: 0,
			Output:  nil,
		},
		Subtraces:           0,
		TraceAddress:        traceAddress,
		BlockHash:           l.context.BlockHash,
		BlockNumber:         l.context.BlockNumber,
		TransactionHash:     l.context.TxHash,
		TransactionPosition: l.context.TxPos,
	}
	if value != nil {
		newItem.Action.Value = value.Bytes()
	}

	l.items = append(l.items, newItem)
	l.stack = append(l.stack, newItem)
}

func (l *ParityLogger) isPrecompiled(addr common.Address) bool {
	for _, p := range l.activePrecompiles {
		if p == addr {
			return true
		}
	}
	return false
}

func (l *ParityLogger) CaptureExit(output []byte, gasUsed uint64, err error) {
	current := l.stack[len(l.stack)-1]
	current.Result.GasUsed = hexutil.Uint64(gasUsed)
	current.Result.Output = append([]byte{}, output...)
	if err != nil {
		current.Error = err.Error()
	}
	l.stack = l.stack[0 : len(l.stack)-1]

	// remove precompiled call
	if l.isPrecompiled(current.Action.To) {
		s := len(l.items)
		l.items = l.items[0 : s-1]
		if s > 1 {
			l.items[s-2].Subtraces -= 1
		}
	}
}

func ReceiptDumpLogger(blockNumber uint64, perFolder, perFile uint64, receipts types.Receipts) error {
	file, err := getFile("receipts", blockNumber, perFolder, perFile)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			err := encoder.Encode(log)
			if err != nil {
				return fmt.Errorf("encode log failed: %w", err)
			}
		}
	}
	return nil
}

type TxLogger struct {
	blockNumber uint64
	blockHash   common.Hash
	file        *os.File
	encoder     *json.Encoder
	signer      types.Signer
	isLondon    bool
	baseFee     *big.Int
}

func NewTxLogger(signer types.Signer, isLondon bool, baseFee *big.Int, blockHash common.Hash, blockNumber uint64, perFolder, perFile uint64) (*TxLogger, error) {
	file, err := getFile("transactions", blockNumber, perFolder, perFile)
	if err != nil {
		return nil, err
	}
	return &TxLogger{
		blockNumber: blockNumber,
		blockHash:   blockHash,
		file:        file,
		encoder:     json.NewEncoder(file),
		signer:      signer,
		isLondon:    isLondon,
		baseFee:     baseFee,
	}, nil
}

func (t *TxLogger) Dump(index int, tx *types.Transaction, receipt *types.Receipt) error {
	from, _ := types.Sender(t.signer, tx)
	// Assign the effective gas price paid
	effectiveGasPrice := hexutil.Uint64(tx.GasPrice().Uint64())
	if t.isLondon {
		gasPrice := new(big.Int).Add(t.baseFee, tx.EffectiveGasTipValue(t.baseFee))
		effectiveGasPrice = hexutil.Uint64(gasPrice.Uint64())
	}
	entry := map[string]interface{}{
		"blockNumber":       t.blockNumber,
		"blockHash":         t.blockHash,
		"transactionIndex":  index,
		"transactionHash":   tx.Hash(),
		"from":              from,
		"to":                tx.To(),
		"gas":               tx.Gas(),
		"gasUsed":           receipt.GasUsed,
		"gasPrice":          tx.GasPrice(),
		"data":              tx.Data(),
		"accessList":        tx.AccessList(),
		"nonce":             tx.Nonce(),
		"gasFeeCap":         tx.GasFeeCap(),
		"gasTipCap":         tx.GasTipCap(),
		"effectiveGasPrice": effectiveGasPrice,
		"type":              tx.Type(),
		"value":             tx.Value(),
		"status":            receipt.Status,
	}
	if err := t.encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode transaction entry %w", err)
	}
	return nil
}

func (t *TxLogger) Close() error {
	return t.file.Close()
}

func BlockDumpLogger(block *types.Block, perFolder, perFile uint64) error {
	file, err := getFile("blocks", block.NumberU64(), perFolder, perFile)
	if err != nil {
		return err
	}
	defer file.Close()

	entry := map[string]interface{}{
		"timestamp":   block.Time(),
		"blockNumber": block.NumberU64(),
		"blockHash":   block.Hash(),
		"parentHash":  block.ParentHash(),
		"gasLimit":    block.GasLimit(),
		"gasUsed":     block.GasUsed(),
		"miner":       block.Coinbase(),
		"difficulty":  block.Difficulty(),
		"nonce":       block.Nonce(),
		"size":        block.Size(),
	}
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode transaction entry %w", err)
	}

	return nil
}
