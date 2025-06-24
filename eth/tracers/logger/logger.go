// Copyright 2021 The go-ethereum Authors
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

package logger

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// Storage represents a contract's storage.
type Storage map[common.Hash]common.Hash

// Config are the configuration options for structured logger the EVM
type Config struct {
	EnableMemory     bool // enable memory capture
	DisableStack     bool // disable stack capture
	DisableStorage   bool // disable storage capture
	EnableReturnData bool // enable return data capture
	Limit            int  // maximum size of output, but zero means unlimited
	// Chain overrides, can be used to execute a trace using future fork rules
	Overrides *params.ChainConfig `json:"overrides,omitempty"`
}

//go:generate go run github.com/fjl/gencodec -type StructLog -field-override structLogMarshaling -out gen_structlog.go

// StructLog is emitted to the EVM each cycle and lists information about the
// current internal state prior to the execution of the statement.
type StructLog struct {
	Pc            uint64                      `json:"pc"`
	Op            vm.OpCode                   `json:"op"`
	Gas           uint64                      `json:"gas"`
	GasCost       uint64                      `json:"gasCost"`
	Memory        []byte                      `json:"memory,omitempty"`
	MemorySize    int                         `json:"memSize"`
	Stack         []uint256.Int               `json:"stack"`
	ReturnData    []byte                      `json:"returnData,omitempty"`
	Storage       map[common.Hash]common.Hash `json:"-"`
	Depth         int                         `json:"depth"`
	RefundCounter uint64                      `json:"refund"`
	Err           error                       `json:"-"`
}

// overrides for gencodec
type structLogMarshaling struct {
	Gas         math.HexOrDecimal64
	GasCost     math.HexOrDecimal64
	Memory      hexutil.Bytes
	ReturnData  hexutil.Bytes
	Stack       []hexutil.U256
	OpName      string `json:"opName"`          // adds call to OpName() in MarshalJSON
	ErrorString string `json:"error,omitempty"` // adds call to ErrorString() in MarshalJSON
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

// WriteTo writes the human-readable log data into the supplied writer.
func (s *StructLog) WriteTo(writer io.Writer) {
	fmt.Fprintf(writer, "%-16spc=%08d gas=%v cost=%v", s.Op, s.Pc, s.Gas, s.GasCost)
	if s.Err != nil {
		fmt.Fprintf(writer, " ERROR: %v", s.Err)
	}
	fmt.Fprintln(writer)

	if len(s.Stack) > 0 {
		fmt.Fprintln(writer, "Stack:")
		for i := len(s.Stack) - 1; i >= 0; i-- {
			fmt.Fprintf(writer, "%08d  %s\n", len(s.Stack)-i-1, s.Stack[i].Hex())
		}
	}
	if len(s.Memory) > 0 {
		fmt.Fprintln(writer, "Memory:")
		fmt.Fprint(writer, hex.Dump(s.Memory))
	}
	if len(s.Storage) > 0 {
		fmt.Fprintln(writer, "Storage:")
		for h, item := range s.Storage {
			fmt.Fprintf(writer, "%x: %x\n", h, item)
		}
	}
	if len(s.ReturnData) > 0 {
		fmt.Fprintln(writer, "ReturnData:")
		fmt.Fprint(writer, hex.Dump(s.ReturnData))
	}
	fmt.Fprintln(writer)
}

// structLogLegacy stores a structured log emitted by the EVM while replaying a
// transaction in debug mode. It's the legacy format used in tracer. The differences
// between the structLog json and the 'legacy' json are:
//
// op:
// Legacy uses string (e.g. "SSTORE"), non-legacy uses a byte.
// non-legacy has an 'opName' field containing the op name.
//
// gas, gasCost:
// Legacy uses integers, non-legacy hex-strings
//
// memory:
// Legacy uses a list of 64-char strings, each representing 32-byte chunks
// of evm memory. Non-legacy just uses a string of hexdata, no chunking.
//
// storage:
// Legacy has a storage field while non-legacy doesn't.
type structLogLegacy struct {
	Pc            uint64             `json:"pc"`
	Op            string             `json:"op"`
	Gas           uint64             `json:"gas"`
	GasCost       uint64             `json:"gasCost"`
	Depth         int                `json:"depth"`
	Error         string             `json:"error,omitempty"`
	Stack         *[]string          `json:"stack,omitempty"`
	ReturnData    string             `json:"returnData,omitempty"`
	Memory        *[]string          `json:"memory,omitempty"`
	Storage       *map[string]string `json:"storage,omitempty"`
	RefundCounter uint64             `json:"refund,omitempty"`
}

// toLegacyJSON converts the structLog to legacy json-encoded legacy form.
func (s *StructLog) toLegacyJSON() json.RawMessage {
	msg := structLogLegacy{
		Pc:            s.Pc,
		Op:            s.Op.String(),
		Gas:           s.Gas,
		GasCost:       s.GasCost,
		Depth:         s.Depth,
		Error:         s.ErrorString(),
		RefundCounter: s.RefundCounter,
	}
	if s.Stack != nil {
		stack := make([]string, len(s.Stack))
		for i, stackValue := range s.Stack {
			stack[i] = stackValue.Hex()
		}
		msg.Stack = &stack
	}
	if len(s.ReturnData) > 0 {
		msg.ReturnData = hexutil.Bytes(s.ReturnData).String()
	}
	if len(s.Memory) > 0 {
		memory := make([]string, 0, (len(s.Memory)+31)/32)
		for i := 0; i+32 <= len(s.Memory); i += 32 {
			memory = append(memory, fmt.Sprintf("%x", s.Memory[i:i+32]))
		}
		msg.Memory = &memory
	}
	if len(s.Storage) > 0 {
		storage := make(map[string]string)
		for i, storageValue := range s.Storage {
			storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
		}
		msg.Storage = &storage
	}
	element, _ := json.Marshal(msg)
	return element
}

// StructLogger is an EVM state logger and implements EVMLogger.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
//
// A StructLogger can either yield it's output immediately (streaming) or store for
// later output.
type StructLogger struct {
	cfg Config
	env *tracing.VMContext

	storage map[common.Address]Storage
	output  []byte
	err     error
	usedGas uint64

	writer     io.Writer         // If set, the logger will stream instead of store logs
	logs       []json.RawMessage // buffer of json-encoded logs
	resultSize int

	interrupt atomic.Bool // Atomic flag to signal execution interruption
	reason    error       // Textual reason for the interruption
	skip      bool        // skip processing hooks.
}

// NewStreamingStructLogger returns a new streaming logger.
func NewStreamingStructLogger(cfg *Config, writer io.Writer) *StructLogger {
	l := NewStructLogger(cfg)
	l.writer = writer
	return l
}

// NewStructLogger construct a new (non-streaming) struct logger.
func NewStructLogger(cfg *Config) *StructLogger {
	logger := &StructLogger{
		storage: make(map[common.Address]Storage),
		logs:    make([]json.RawMessage, 0),
	}
	if cfg != nil {
		logger.cfg = *cfg
	}
	return logger
}

func (l *StructLogger) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart:           l.OnTxStart,
		OnTxEnd:             l.OnTxEnd,
		OnSystemCallStartV2: l.OnSystemCallStart,
		OnSystemCallEnd:     l.OnSystemCallEnd,
		OnExit:              l.OnExit,
		OnOpcode:            l.OnOpcode,
	}
}

// OnOpcode logs a new structured log message and pushes it out to the environment
//
// OnOpcode also tracks SLOAD/SSTORE ops to track storage change.
func (l *StructLogger) OnOpcode(pc uint64, opcode byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	// If tracing was interrupted, exit
	if l.interrupt.Load() {
		return
	}
	// Processing a system call.
	if l.skip {
		return
	}
	// check if already accumulated the size of the response.
	if l.cfg.Limit != 0 && l.resultSize > l.cfg.Limit {
		return
	}
	var (
		op           = vm.OpCode(opcode)
		memory       = scope.MemoryData()
		contractAddr = scope.Address()
		stack        = scope.StackData()
		stackLen     = len(stack)
	)
	log := StructLog{pc, op, gas, cost, nil, len(memory), nil, nil, nil, depth, l.env.StateDB.GetRefund(), err}
	if l.cfg.EnableMemory {
		log.Memory = memory
	}
	if !l.cfg.DisableStack {
		log.Stack = scope.StackData()
	}
	if l.cfg.EnableReturnData {
		log.ReturnData = rData
	}

	// Copy a snapshot of the current storage to a new container
	var storage Storage
	if !l.cfg.DisableStorage && (op == vm.SLOAD || op == vm.SSTORE) {
		// initialise new changed values storage container for this contract
		// if not present.
		if l.storage[contractAddr] == nil {
			l.storage[contractAddr] = make(Storage)
		}
		// capture SLOAD opcodes and record the read entry in the local storage
		if op == vm.SLOAD && stackLen >= 1 {
			var (
				address = common.Hash(stack[stackLen-1].Bytes32())
				value   = l.env.StateDB.GetState(contractAddr, address)
			)
			l.storage[contractAddr][address] = value
			storage = maps.Clone(l.storage[contractAddr])
		} else if op == vm.SSTORE && stackLen >= 2 {
			// capture SSTORE opcodes and record the written entry in the local storage.
			var (
				value   = common.Hash(stack[stackLen-2].Bytes32())
				address = common.Hash(stack[stackLen-1].Bytes32())
			)
			l.storage[contractAddr][address] = value
			storage = maps.Clone(l.storage[contractAddr])
		}
	}
	log.Storage = storage

	// create a log
	if l.writer == nil {
		entry := log.toLegacyJSON()
		l.resultSize += len(entry)
		l.logs = append(l.logs, entry)
		return
	}
	log.WriteTo(l.writer)
}

// OnExit is called a call frame finishes processing.
func (l *StructLogger) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if depth != 0 {
		return
	}
	if l.skip {
		return
	}
	l.output = output
	l.err = err
	// TODO @holiman, should we output the per-scope output?
	//if l.cfg.Debug {
	//	fmt.Printf("%#x\n", output)
	//	if err != nil {
	//		fmt.Printf(" error: %v\n", err)
	//	}
	//}
}

func (l *StructLogger) GetResult() (json.RawMessage, error) {
	// Tracing aborted
	if l.reason != nil {
		return nil, l.reason
	}
	failed := l.err != nil
	returnData := common.CopyBytes(l.output)
	// Return data when successful and revert reason when reverted, otherwise empty.
	if failed && !errors.Is(l.err, vm.ErrExecutionReverted) {
		returnData = []byte{}
	}
	return json.Marshal(&ExecutionResult{
		Gas:         l.usedGas,
		Failed:      failed,
		ReturnValue: returnData,
		StructLogs:  l.logs,
	})
}

// Stop terminates execution of the tracer at the first opportune moment.
func (l *StructLogger) Stop(err error) {
	l.reason = err
	l.interrupt.Store(true)
}

func (l *StructLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	l.env = env
}
func (l *StructLogger) OnSystemCallStart(env *tracing.VMContext) {
	l.skip = true
}

func (l *StructLogger) OnSystemCallEnd() {
	l.skip = false
}

func (l *StructLogger) OnTxEnd(receipt *types.Receipt, err error) {
	if err != nil {
		// Don't override vm error
		if l.err == nil {
			l.err = err
		}
		return
	}
	if receipt != nil {
		l.usedGas = receipt.GasUsed
	}
}

// Error returns the VM error captured by the trace.
func (l *StructLogger) Error() error { return l.err }

// Output returns the VM return value captured by the trace.
func (l *StructLogger) Output() []byte { return l.output }

// WriteTrace writes a formatted trace to the given writer
// @deprecated
func WriteTrace(writer io.Writer, logs []StructLog) {
	for _, log := range logs {
		log.WriteTo(writer)
	}
}

type mdLogger struct {
	out  io.Writer
	cfg  *Config
	env  *tracing.VMContext
	skip bool
}

// NewMarkdownLogger creates a logger which outputs information in a format adapted
// for human readability, and is also a valid markdown table
func NewMarkdownLogger(cfg *Config, writer io.Writer) *mdLogger {
	l := &mdLogger{out: writer, cfg: cfg}
	if l.cfg == nil {
		l.cfg = &Config{}
	}
	return l
}

func (t *mdLogger) Hooks() *tracing.Hooks {
	return &tracing.Hooks{
		OnTxStart:           t.OnTxStart,
		OnSystemCallStartV2: t.OnSystemCallStart,
		OnSystemCallEnd:     t.OnSystemCallEnd,
		OnEnter:             t.OnEnter,
		OnExit:              t.OnExit,
		OnOpcode:            t.OnOpcode,
		OnFault:             t.OnFault,
	}
}

func (t *mdLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.env = env
}

func (t *mdLogger) OnSystemCallStart(env *tracing.VMContext) {
	t.skip = true
}

func (t *mdLogger) OnSystemCallEnd() {
	t.skip = false
}

func (t *mdLogger) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.skip {
		return
	}
	if depth != 0 {
		return
	}
	if create := vm.OpCode(typ) == vm.CREATE; !create {
		fmt.Fprintf(t.out, "Pre-execution info:\n"+
			"  - from: `%v`\n"+
			"  - to: `%v`\n"+
			"  - data: `%#x`\n"+
			"  - gas: `%d`\n"+
			"  - value: `%v` wei\n",
			from.String(), to.String(), input, gas, value)
	} else {
		fmt.Fprintf(t.out, "Pre-execution info:\n"+
			"  - from: `%v`\n"+
			"  - create: `%v`\n"+
			"  - data: `%#x`\n"+
			"  - gas: `%d`\n"+
			"  - value: `%v` wei\n",
			from.String(), to.String(), input, gas, value)
	}
	fmt.Fprintf(t.out, `
|  Pc   |      Op     | Cost |   Refund  |   Stack   |
|-------|-------------|------|-----------|-----------|
`)
}

func (t *mdLogger) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if t.skip {
		return
	}
	if depth == 0 {
		fmt.Fprintf(t.out, "\nPost-execution info:\n"+
			"  - output: `%#x`\n"+
			"  - consumed gas: `%d`\n"+
			"  - error: `%v`\n",
			output, gasUsed, err)
	}
}

// OnOpcode also tracks SLOAD/SSTORE ops to track storage change.
func (t *mdLogger) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if t.skip {
		return
	}
	stack := scope.StackData()
	fmt.Fprintf(t.out, "| %4d  | %10v  |  %3d |%10v |", pc, vm.OpCode(op).String(),
		cost, t.env.StateDB.GetRefund())

	if !t.cfg.DisableStack {
		// format stack
		var a []string
		for _, elem := range stack {
			a = append(a, elem.Hex())
		}
		b := fmt.Sprintf("[%v]", strings.Join(a, ","))
		fmt.Fprintf(t.out, "%10v |", b)
	}
	fmt.Fprintln(t.out, "")
	if err != nil {
		fmt.Fprintf(t.out, "Error: %v\n", err)
	}
}

func (t *mdLogger) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	if t.skip {
		return
	}
	fmt.Fprintf(t.out, "\nError: at pc=%d, op=%v: %v\n", pc, op, err)
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64            `json:"gas"`
	Failed      bool              `json:"failed"`
	ReturnValue hexutil.Bytes     `json:"returnValue"`
	StructLogs  []json.RawMessage `json:"structLogs"`
}
