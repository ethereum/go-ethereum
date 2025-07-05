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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
)

func init() {
	tracers.LiveDirectory.Register("perf", newPerfTracer)
}

type perfTracerConfig struct {
	Path string `json:"path"`
}

// perfTracer is a live tracer that measures and records transaction processing performance metrics.
// It tracks total processing time, IO time (account and storage reads), and EVM execution time for
// each transaction. The metrics are written to a JSONL file.
type perfTracer struct {
	path    string
	file    *os.File
	encoder *json.Encoder

	// Block context
	currentBlock     *types.Block
	currentBlockHash common.Hash
	blockStartTime   time.Time

	// Transaction tracking
	txStartTime time.Time
	txIndex     int

	// IO measurements
	prevAccountReads time.Duration
	prevStorageReads time.Duration

	// Transaction data collection
	txData []map[string]interface{}

	statedb tracing.StateDB
}

func newPerfTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config perfTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}
	if config.Path == "" {
		return nil, errors.New("path is required")
	}

	// Open JSONL file
	file, err := os.OpenFile(config.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSONL file: %v", err)
	}

	t := &perfTracer{
		path:    config.Path,
		file:    file,
		encoder: json.NewEncoder(file),
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
	t.txData = make([]map[string]interface{}, 0)
	t.blockStartTime = time.Now()
	// Reset previous IO measurements for the new block
	t.prevAccountReads = 0
	t.prevStorageReads = 0
}

func (t *perfTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.txStartTime = time.Now()
	t.statedb = vm.StateDB

	// The accumulated measurements include IO performed before any txs were executed.
	if t.txIndex == 0 {
		initialIO := t.statedb.GetAccumulatedIOMeasurements()
		t.prevAccountReads = initialIO.AccountReads
		t.prevStorageReads = initialIO.StorageReads
	}
}

func (t *perfTracer) OnTxEnd(receipt *types.Receipt, err error) {
	var (
		totalTime     = time.Since(t.txStartTime)
		accumulatedIO = t.statedb.GetAccumulatedIOMeasurements()
		ioTime        = (accumulatedIO.AccountReads - t.prevAccountReads) +
			(accumulatedIO.StorageReads - t.prevStorageReads)
		evmTime time.Duration
	)
	if ioTime > totalTime {
		log.Error("PerfTracer: IO time exceeds total time", "ioTime", ioTime, "totalTime", totalTime, "txIdx", t.txIndex)
	} else {
		evmTime = totalTime - ioTime
	}

	txRecord := map[string]interface{}{
		"txIndex":   fmt.Sprintf("0x%x", t.txIndex),
		"txHash":    receipt.TxHash.Hex(),
		"gasUsed":   fmt.Sprintf("0x%x", receipt.GasUsed),
		"totalTime": fmt.Sprintf("0x%x", totalTime.Nanoseconds()),
		"ioTime":    fmt.Sprintf("0x%x", ioTime.Nanoseconds()),
		"evmTime":   fmt.Sprintf("0x%x", evmTime.Nanoseconds()),
	}

	t.txData = append(t.txData, txRecord)

	t.prevAccountReads = accumulatedIO.AccountReads
	t.prevStorageReads = accumulatedIO.StorageReads
	t.txIndex++
}

func (t *perfTracer) OnBlockEnd(err error) {
	// Calculate block-level timings
	totalTime := time.Since(t.blockStartTime)
	blockEndIO := t.statedb.GetAccumulatedIOMeasurements()
	ioTime := blockEndIO.AccountReads + blockEndIO.AccountHashes + blockEndIO.AccountUpdates + blockEndIO.AccountCommits +
		blockEndIO.StorageReads + blockEndIO.StorageUpdates + blockEndIO.StorageCommits +
		blockEndIO.SnapshotCommits + blockEndIO.TrieDBCommits
	evmTime := totalTime - ioTime

	// Sanity check: IO time should not exceed total time
	if ioTime > totalTime {
		log.Error("PerfTracer: Block IO time exceeds total time",
			"blockNumber", t.currentBlock.Number(),
			"ioTime", ioTime,
			"totalTime", totalTime)
		return
	}

	// Calculate sum of transaction times and gas
	var totalTxTime time.Duration
	for _, tx := range t.txData {
		txTime, _ := strconv.ParseUint(tx["totalTime"].(string)[2:], 16, 64)
		totalTxTime += time.Duration(txTime)
	}
	if totalTxTime > totalTime {
		log.Error("PerfTracer: Sum of transaction times exceeds block total time",
			"blockNumber", t.currentBlock.Number(),
			"totalTxTime", totalTxTime,
			"blockTotalTime", totalTime)
		return
	}

	blockRecord := map[string]interface{}{
		"blockNumber":  fmt.Sprintf("0x%x", t.currentBlock.Number()),
		"blockHash":    t.currentBlockHash.Hex(),
		"gasUsed":      fmt.Sprintf("0x%x", t.currentBlock.GasUsed()),
		"totalTime":    fmt.Sprintf("0x%x", totalTime.Nanoseconds()),
		"ioTime":       fmt.Sprintf("0x%x", ioTime.Nanoseconds()),
		"evmTime":      fmt.Sprintf("0x%x", evmTime.Nanoseconds()),
		"transactions": t.txData,
	}

	if err := t.encoder.Encode(blockRecord); err != nil {
		fmt.Printf("Failed to write block record: %v\n", err)
	}

	if t.file != nil {
		t.file.Sync()
	}
}

func (t *perfTracer) OnClose() {
	if t.file != nil {
		t.file.Close()
	}
}
