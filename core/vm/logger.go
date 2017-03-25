// Copyright 2015 The go-ethereum Authors
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
	"encoding/hex"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
)

type Storage map[common.Hash]common.Hash

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		cpy[key] = value
	}

	return cpy
}

// LogConfig are the configuration options for structured logger the EVM
type LogConfig struct {
	DisableMemory  bool // disable memory capture
	DisableStack   bool // disable stack capture
	DisableStorage bool // disable storage capture
	FullStorage    bool // show full storage (slow)
	Limit          int  // maximum length of output, but zero means unlimited
}

// StructLog is emitted to the EVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc      uint64
	Op      OpCode
	Gas     uint64
	GasCost uint64
	Memory  []byte
	Stack   []*big.Int
	Storage map[common.Hash]common.Hash
	Depth   int
	Err     error
}

// Tracer is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type Tracer interface {
	CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, contract *Contract, depth int, err error) error
}

// StructLogger is an EVM state logger and implements Tracer.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg LogConfig

	logs          []StructLog
	changedValues map[common.Address]Storage
}

// NewLogger returns a new logger
func NewStructLogger(cfg *LogConfig) *StructLogger {
	logger := &StructLogger{
		changedValues: make(map[common.Address]Storage),
	}
	if cfg != nil {
		logger.cfg = *cfg
	}
	return logger
}

// captureState logs a new structured log message and pushes it out to the environment
//
// captureState also tracks SSTORE ops to track dirty values.
func (l *StructLogger) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, memory *Memory, stack *Stack, contract *Contract, depth int, err error) error {
	// check if already accumulated the specified number of logs
	if l.cfg.Limit != 0 && l.cfg.Limit <= len(l.logs) {
		return ErrTraceLimitReached
	}

	// initialise new changed values storage container for this contract
	// if not present.
	if l.changedValues[contract.Address()] == nil {
		l.changedValues[contract.Address()] = make(Storage)
	}

	// capture SSTORE opcodes and determine the changed value and store
	// it in the local storage container. NOTE: we do not need to do any
	// range checks here because that's already handler prior to calling
	// this function.
	switch op {
	case SSTORE:
		var (
			value   = common.BigToHash(stack.data[stack.len()-2])
			address = common.BigToHash(stack.data[stack.len()-1])
		)
		l.changedValues[contract.Address()][address] = value
	}

	// copy a snapstot of the current memory state to a new buffer
	var mem []byte
	if !l.cfg.DisableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}

	// copy a snapshot of the current stack state to a new buffer
	var stck []*big.Int
	if !l.cfg.DisableStack {
		stck = make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = new(big.Int).Set(item)
		}
	}

	// Copy the storage based on the settings specified in the log config. If full storage
	// is disabled (default) we can use the simple Storage.Copy method, otherwise we use
	// the state object to query for all values (slow process).
	var storage Storage
	if !l.cfg.DisableStorage {
		if l.cfg.FullStorage {
			storage = make(Storage)
			// Get the contract account and loop over each storage entry. This may involve looping over
			// the trie and is a very expensive process.

			env.StateDB.ForEachStorage(contract.Address(), func(key, value common.Hash) bool {
				storage[key] = value
				// Return true, indicating we'd like to continue.
				return true
			})
		} else {
			// copy a snapshot of the current storage to a new container.
			storage = l.changedValues[contract.Address()].Copy()
		}
	}
	// create a new snaptshot of the EVM.
	log := StructLog{pc, op, gas, cost, mem, stck, storage, env.depth, err}

	l.logs = append(l.logs, log)
	return nil
}

// StructLogs returns a list of captured log entries
func (l *StructLogger) StructLogs() []StructLog {
	return l.logs
}

// WriteTrace writes a formatted trace to the given writer
func WriteTrace(writer io.Writer, logs []StructLog) {
	for _, log := range logs {
		fmt.Fprintf(writer, "%-10spc=%08d gas=%v cost=%v", log.Op, log.Pc, log.Gas, log.GasCost)
		if log.Err != nil {
			fmt.Fprintf(writer, " ERROR: %v", log.Err)
		}
		fmt.Fprintf(writer, "\n")

		for i := len(log.Stack) - 1; i >= 0; i-- {
			fmt.Fprintf(writer, "%08d  %x\n", len(log.Stack)-i-1, math.PaddedBigBytes(log.Stack[i], 32))
		}

		fmt.Fprint(writer, hex.Dump(log.Memory))

		for h, item := range log.Storage {
			fmt.Fprintf(writer, "%x: %x\n", h, item)
		}
		fmt.Fprintln(writer)
	}
}

// WriteLogs writes vm logs in a readable format to the given writer
func WriteLogs(writer io.Writer, logs []*types.Log) {
	for _, log := range logs {
		fmt.Fprintf(writer, "LOG%d: %x bn=%d txi=%x\n", len(log.Topics), log.Address, log.BlockNumber, log.TxIndex)

		for i, topic := range log.Topics {
			fmt.Fprintf(writer, "%08d  %x\n", i, topic)
		}

		fmt.Fprint(writer, hex.Dump(log.Data))
		fmt.Fprintln(writer)
	}
}
