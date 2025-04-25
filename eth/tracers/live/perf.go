// Copyright 2025 go-ethereum Authors
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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

func init() {
	tracers.LiveDirectory.Register("perfTracer", newPerfTracer)
}

type perfTracerConfig struct {
	CSVPath string `json:"csvPath"`
}

// perfTracer is a live tracer that measures and records transaction processing performance metrics.
// It tracks total processing time, IO time (account and storage reads), and EVM execution time for
// each transaction. The metrics are written to a CSV file.
type perfTracer struct {
	csvPath string
	writer  *csv.Writer
	file    *os.File

	// Block context
	currentBlock     *types.Block
	currentBlockHash common.Hash

	// Transaction tracking
	txStartTime time.Time
	txIndex     int

	// IO measurements
	prevAccountReads time.Duration
	prevStorageReads time.Duration

	statedb tracing.StateDB
}

func newPerfTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config perfTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}
	if config.CSVPath == "" {
		return nil, errors.New("csv path is required")
	}

	// Open CSV file
	file, err := os.OpenFile(config.CSVPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %v", err)
	}
	writer := csv.NewWriter(file)

	// Write header if file is empty
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get file info: %v", err)
	}
	if info.Size() == 0 {
		header := []string{"block_number", "block_hash", "tx_index", "tx_hash", "total_time_ns", "io_time_ns", "evm_time_ns", "gas_used"}
		if err := writer.Write(header); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write CSV header: %v", err)
		}
		writer.Flush()
	}

	t := &perfTracer{
		csvPath: config.CSVPath,
		writer:  writer,
		file:    file,
	}

	return &tracing.Hooks{
		OnBlockStart: t.OnBlockStart,
		OnTxStart:    t.OnTxStart,
		OnTxEnd:      t.OnTxEnd,
		OnBlockEnd:   t.OnBlockEnd,
		OnClose:      t.OnClose,
	}, nil
}

func (t *perfTracer) OnBlockStart(event tracing.BlockEvent) {
	t.currentBlock = event.Block
	t.currentBlockHash = event.Block.Hash()
	t.txIndex = 0
	// Reset previous IO measurements for the new block
	t.prevAccountReads = 0
	t.prevStorageReads = 0
}

func (t *perfTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.txStartTime = time.Now()
	t.statedb = vm.StateDB
}

func (t *perfTracer) OnTxEnd(receipt *types.Receipt, err error) {
	var (
		totalTime     = time.Since(t.txStartTime)
		accumulatedIO = t.statedb.GetAccumulatedIOMeasurements()
		ioTime        = (accumulatedIO.AccountReads - t.prevAccountReads) +
			(accumulatedIO.StorageReads - t.prevStorageReads)
		evmTime = totalTime - ioTime
	)

	row := []string{
		t.currentBlock.Number().String(),
		t.currentBlockHash.Hex(),
		fmt.Sprintf("%d", t.txIndex),
		receipt.TxHash.Hex(),
		fmt.Sprintf("%d", totalTime.Nanoseconds()),
		fmt.Sprintf("%d", ioTime.Nanoseconds()),
		fmt.Sprintf("%d", evmTime.Nanoseconds()),
		fmt.Sprintf("%d", receipt.GasUsed),
	}
	if err := t.writer.Write(row); err != nil {
		fmt.Printf("Failed to write CSV row: %v\n", err)
	}

	t.prevAccountReads = accumulatedIO.AccountReads
	t.prevStorageReads = accumulatedIO.StorageReads
	t.txIndex++
}

// OnBlockEnd implements tracing.BlockEndHook
func (t *perfTracer) OnBlockEnd(err error) {
	if t.writer != nil {
		t.writer.Flush()
	}
}

func (t *perfTracer) OnClose() {
	if t.writer != nil {
		t.writer.Flush()
	}
	if t.file != nil {
		t.file.Close()
	}
}
