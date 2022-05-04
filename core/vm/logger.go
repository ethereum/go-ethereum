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
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/holiman/uint256"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
)

// Storage represents a contract's storage.
type Storage map[common.Hash]common.Hash

// Copy duplicates the current storage.
func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}
	return cpy
}

// LogConfig are the configuration options for structured logger the EVM
type LogConfig struct {
	EnableMemory     bool // enable memory capture
	DisableStack     bool // disable stack capture
	DisableStorage   bool // disable storage capture
	EnableReturnData bool // enable return data capture
	Debug            bool // print output during capture end
	Limit            int  // maximum length of output, but zero means unlimited
	// Chain overrides, can be used to execute a trace using future fork rules
	Overrides *params.ChainConfig `json:"overrides,omitempty"`
}

//go:generate gencodec -type StructLog -field-override structLogMarshaling -out gen_structlog.go

// StructLog is emitted to the EVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
type StructLog struct {
	Pc            uint64                      `json:"pc"`
	Op            OpCode                      `json:"op"`
	Gas           uint64                      `json:"gas"`
	GasCost       uint64                      `json:"gasCost"`
	Memory        bytes.Buffer                `json:"memory"`
	MemorySize    int                         `json:"memSize"`
	Stack         []uint256.Int               `json:"stack"`
	ReturnData    bytes.Buffer                `json:"returnData"`
	Storage       map[common.Hash]common.Hash `json:"-"`
	Depth         int                         `json:"depth"`
	RefundCounter uint64                      `json:"refund"`
	ExtraData     *types.ExtraData            `json:"extraData"`
	Err           error                       `json:"-"`
}

var (
	loggerPool = sync.Pool{
		New: func() interface{} {
			return &StructLog{
				Stack:     make([]uint256.Int, 0),
				ExtraData: types.NewExtraData(),
			}
		},
	}
)

func NewStructlog(pc uint64, op OpCode, gas, cost uint64, depth int) *StructLog {

	structlog := loggerPool.Get().(*StructLog)
	structlog.Pc, structlog.Op, structlog.Gas, structlog.GasCost, structlog.Depth = pc, op, gas, cost, depth

	runtime.SetFinalizer(structlog, func(logger *StructLog) {
		logger.clean()
		loggerPool.Put(logger)
	})
	return structlog
}

func (s *StructLog) clean() {
	s.Memory.Reset()
	s.Stack = s.Stack[:0]
	s.ReturnData.Reset()
	s.Storage = nil
	s.ExtraData.Clean()
}

// overrides for gencodec
type structLogMarshaling struct {
	Gas         math.HexOrDecimal64
	GasCost     math.HexOrDecimal64
	Memory      hexutil.Bytes
	ReturnData  hexutil.Bytes
	OpName      string `json:"opName"` // adds call to OpName() in MarshalJSON
	ErrorString string `json:"error"`  // adds call to ErrorString() in MarshalJSON
}

// OpName formats the operand name in a human-readable format.
func (s *StructLog) OpName() string {
	return s.Op.String()
}

// ErrorString formats the log's error as a string.
func (s *StructLog) ErrorString() string {
	if s.Err != nil {
		return s.Err.Error()
	}
	return ""
}

// EVMLogger is used to collect execution traces from an EVM transaction
// execution. CaptureState is called for each step of the VM with the
// current VM state.
// Note that reference types are actual VM data structures; make copies
// if you need to retain them beyond the current call.
type EVMLogger interface {
	CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error)
	CaptureStateAfter(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error)
	CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	CaptureExit(output []byte, gasUsed uint64, err error)
	CaptureFault(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error)
	CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error)
}

// StructLogger is an EVM state logger and implements EVMLogger.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg LogConfig
	env *EVM

	storage map[common.Address]Storage
	logs    []StructLog
	output  []byte
	err     error
}

// NewStructLogger returns a new logger
func NewStructLogger(cfg *LogConfig) *StructLogger {
	logger := &StructLogger{
		storage: make(map[common.Address]Storage),
	}
	if cfg != nil {
		logger.cfg = *cfg
	}
	return logger
}

// Reset clears the data held by the logger.
func (l *StructLogger) Reset() {
	l.storage = make(map[common.Address]Storage)
	l.output = make([]byte, 0)
	l.logs = l.logs[:0]
	l.err = nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (l *StructLogger) CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	l.env = env
}

// CaptureState logs a new structured log message and pushes it out to the environment
//
// CaptureState also tracks SLOAD/SSTORE ops to track storage change.
func (l *StructLogger) CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
	memory := scope.Memory
	stack := scope.Stack
	contract := scope.Contract
	// create a struct log.
	structlog := NewStructlog(pc, op, gas, cost, depth)

	// check if already accumulated the specified number of logs
	if l.cfg.Limit != 0 && l.cfg.Limit <= len(l.logs) {
		return
	}
	// Copy a snapshot of the current memory state to a new buffer
	if l.cfg.EnableMemory {
		structlog.Memory.Write(memory.Data())
		structlog.MemorySize = memory.Len()
	}
	// Copy a snapshot of the current stack state to a new buffer
	if !l.cfg.DisableStack {
		structlog.Stack = append(structlog.Stack, stack.Data()...)
	}
	var (
		recordStorageDetail bool
		storageKey          common.Hash
		storageValue        common.Hash
	)
	if !l.cfg.DisableStorage {
		if op == SLOAD && stack.len() >= 1 {
			recordStorageDetail = true
			storageKey = stack.data[stack.len()-1].Bytes32()
			storageValue = l.env.StateDB.GetState(contract.Address(), storageKey)
		} else if op == SSTORE && stack.len() >= 2 {
			recordStorageDetail = true
			storageKey = stack.data[stack.len()-1].Bytes32()
			storageValue = stack.data[stack.len()-2].Bytes32()
		}
	}
	if recordStorageDetail {
		contractAddress := contract.Address()
		if l.storage[contractAddress] == nil {
			l.storage[contractAddress] = make(Storage)
		}
		l.storage[contractAddress][storageKey] = storageValue
		structlog.Storage = l.storage[contractAddress].Copy()

		if err := traceStorageProof(l, scope, structlog.ExtraData); err != nil {
			log.Error("Failed to trace data", "opcode", op.String(), "err", err)
		}
	}
	if l.cfg.EnableReturnData {
		structlog.ReturnData.Write(rData)
	}
	execFuncList, ok := OpcodeExecs[op]
	if ok {
		// execute trace func list.
		for _, exec := range execFuncList {
			if err = exec(l, scope, structlog.ExtraData); err != nil {
				log.Error("Failed to trace data", "opcode", op.String(), "err", err)
			}
		}
	}

	structlog.RefundCounter, structlog.Err = l.env.StateDB.GetRefund(), err
	l.logs = append(l.logs, *structlog)
}

func (l *StructLogger) CaptureStateAfter(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
	if !l.cfg.DisableStorage && op == SSTORE {
		logLen := len(l.logs)
		if logLen <= 0 {
			log.Error("Failed to trace after_state for sstore", "err", "empty length log")
			return
		}

		lastLog := l.logs[logLen-1]
		if lastLog.Op != SSTORE {
			log.Error("Failed to trace after_state for sstore", "err", "op mismatch")
			return
		}
		if lastLog.ExtraData == nil || len(lastLog.ExtraData.ProofList) == 0 {
			log.Error("Failed to trace after_state for sstore", "err", "empty before_state ExtraData")
			return
		}

		contractAddress := scope.Contract.Address()
		if len(lastLog.Stack) <= 0 {
			log.Error("Failed to trace after_state for sstore", "err", "empty stack for last log")
			return
		}
		storageKey := common.Hash(lastLog.Stack[len(lastLog.Stack)-1].Bytes32())
		proof, err := getWrappedProofForStorage(l, contractAddress, storageKey)
		if err != nil {
			log.Error("Failed to trace after_state storage_proof for sstore", "err", err)
		}

		l.logs[logLen-1].ExtraData.ProofList = append(lastLog.ExtraData.ProofList, proof)
	}
}

// CaptureFault implements the EVMLogger interface to trace an execution fault
// while running an opcode.
func (l *StructLogger) CaptureFault(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (l *StructLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	l.output = output
	l.err = err
	if l.cfg.Debug {
		fmt.Printf("0x%x\n", output)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
		}
	}
}

func (l *StructLogger) CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (l *StructLogger) CaptureExit(output []byte, gasUsed uint64, err error) {}

// StructLogs returns the captured log entries.
func (l *StructLogger) StructLogs() []StructLog { return l.logs }

// Error returns the VM error captured by the trace.
func (l *StructLogger) Error() error { return l.err }

// Output returns the VM return value captured by the trace.
func (l *StructLogger) Output() []byte { return l.output }

// WriteTrace writes a formatted trace to the given writer
func WriteTrace(writer io.Writer, logs []StructLog) {
	for _, log := range logs {
		fmt.Fprintf(writer, "%-16spc=%08d gas=%v cost=%v", log.Op, log.Pc, log.Gas, log.GasCost)
		if log.Err != nil {
			fmt.Fprintf(writer, " ERROR: %v", log.Err)
		}
		fmt.Fprintln(writer)

		if len(log.Stack) > 0 {
			fmt.Fprintln(writer, "Stack:")
			for i := len(log.Stack) - 1; i >= 0; i-- {
				fmt.Fprintf(writer, "%08d  %s\n", len(log.Stack)-i-1, log.Stack[i].Hex())
			}
		}
		if log.Memory.Len() > 0 {
			fmt.Fprintln(writer, "Memory:")
			fmt.Fprint(writer, hex.Dump(log.Memory.Bytes()))
		}
		if len(log.Storage) > 0 {
			fmt.Fprintln(writer, "Storage:")
			for h, item := range log.Storage {
				fmt.Fprintf(writer, "%x: %x\n", h, item)
			}
		}
		if log.ReturnData.Len() > 0 {
			fmt.Fprintln(writer, "ReturnData:")
			fmt.Fprint(writer, hex.Dump(log.ReturnData.Bytes()))
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

type mdLogger struct {
	out io.Writer
	cfg *LogConfig
	env *EVM
}

// NewMarkdownLogger creates a logger which outputs information in a format adapted
// for human readability, and is also a valid markdown table
func NewMarkdownLogger(cfg *LogConfig, writer io.Writer) *mdLogger {
	l := &mdLogger{out: writer, cfg: cfg}
	if l.cfg == nil {
		l.cfg = &LogConfig{}
	}
	return l
}

func (t *mdLogger) CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	if !create {
		fmt.Fprintf(t.out, "From: `%v`\nTo: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	} else {
		fmt.Fprintf(t.out, "From: `%v`\nCreate at: `%v`\nData: `0x%x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	}

	fmt.Fprintf(t.out, `
|  Pc   |      Op     | Cost |   Stack   |   RStack  |  Refund |
|-------|-------------|------|-----------|-----------|---------|
`)
}

// CaptureState also tracks SLOAD/SSTORE ops to track storage change.
func (t *mdLogger) CaptureState(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
	stack := scope.Stack
	fmt.Fprintf(t.out, "| %4d  | %10v  |  %3d |", pc, op, cost)

	if !t.cfg.DisableStack {
		// format stack
		var a []string
		for _, elem := range stack.data {
			a = append(a, elem.Hex())
		}
		b := fmt.Sprintf("[%v]", strings.Join(a, ","))
		fmt.Fprintf(t.out, "%10v |", b)
	}
	fmt.Fprintf(t.out, "%10v |", t.env.StateDB.GetRefund())
	fmt.Fprintln(t.out, "")
	if err != nil {
		fmt.Fprintf(t.out, "Error: %v\n", err)
	}
}

// CaptureStateAfter for special needs, tracks SSTORE ops and records the storage change.
func (t *mdLogger) CaptureStateAfter(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
}

func (t *mdLogger) CaptureFault(pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error) {
	fmt.Fprintf(t.out, "\nError: at pc=%d, op=%v: %v\n", pc, op, err)
}

func (t *mdLogger) CaptureEnd(output []byte, gasUsed uint64, tm time.Duration, err error) {
	fmt.Fprintf(t.out, "\nOutput: `0x%x`\nConsumed gas: `%d`\nError: `%v`\n",
		output, gasUsed, err)
}

func (t *mdLogger) CaptureEnter(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *mdLogger) CaptureExit(output []byte, gasUsed uint64, err error) {}

// FormatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []StructLog) []types.StructLogRes {
	formatted := make([]types.StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = types.StructLogRes{
			Pc:            trace.Pc,
			Op:            trace.Op.String(),
			Gas:           trace.Gas,
			GasCost:       trace.GasCost,
			Depth:         trace.Depth,
			RefundCounter: trace.RefundCounter,
			Error:         trace.ErrorString(),
		}
		if len(trace.Stack) != 0 {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = stackValue.Hex()
			}
			formatted[index].Stack = &stack
		}
		if trace.Memory.Len() != 0 {
			memory := make([]string, 0, (trace.Memory.Len()+31)/32)
			for i := 0; i+32 <= trace.Memory.Len(); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory.Bytes()[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if len(trace.Storage) != 0 {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
		if trace.ExtraData != nil {
			formatted[index].ExtraData = trace.ExtraData.SealExtraData()
		}
	}
	return formatted
}
