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
	"bytes"
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

// countingWriter wraps an io.Writer and records how many bytes have been
// written through it.
type countingWriter struct {
	w io.Writer
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += n
	return n, err
}

// pendingTextLog holds one text record. Its header can be updated on fault
// while its encoded snapshot preserves the state captured before execution.
type pendingTextLog struct {
	header *textLogHeader
	body   bytes.Buffer
}

type textLogHeader struct {
	pc    uint64
	op    vm.OpCode
	gas   tracing.Gas
	cost  tracing.OpCost
	depth int
	err   error
}

//go:generate go run github.com/fjl/gencodec -type StructLog -field-override structLogMarshaling -out gen_structlog.go

// StructLog is emitted to the EVM each cycle and lists information about the
// current internal state prior to the execution of the statement.
type StructLog struct {
	Pc uint64    `json:"pc"`
	Op vm.OpCode `json:"op"`

	// Execution gas, the opcode's execution charge and the budget before
	// it executes
	Gas     uint64 `json:"gas"`
	GasCost uint64 `json:"gasCost"`

	// EIP-8037 state gas, omitted when zero: the opcode's state charge and the
	// reservoir before it executes
	StateGasCost      int64  `json:"stateGasCost,omitempty"`
	StateGasReservoir uint64 `json:"stateGasReservoir,omitempty"`

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
	Gas               math.HexOrDecimal64
	GasCost           math.HexOrDecimal64
	StateGasReservoir math.HexOrDecimal64

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

// Write writes the human-readable log data into the supplied writer.
func (s *StructLog) Write(writer io.Writer) {
	s.textHeader().write(writer)
	s.writeSnapshot(writer)
}

func (s *StructLog) textHeader() *textLogHeader {
	return &textLogHeader{
		pc: s.Pc,
		op: s.Op,
		gas: tracing.Gas{
			Execution: s.Gas,
			State:     s.StateGasReservoir,
		},
		cost: tracing.OpCost{
			Execution: s.GasCost,
			State:     s.StateGasCost,
		},
		depth: s.Depth,
		err:   s.Err,
	}
}

func (h textLogHeader) write(writer io.Writer) {
	fmt.Fprintf(writer, "%-16spc=%08d gas=%v cost=%v", h.op, h.pc, h.gas.Execution, h.cost.Execution)
	if h.cost.State != 0 || h.gas.State != 0 {
		fmt.Fprintf(writer, " stateCost=%v reservoir=%v", h.cost.State, h.gas.State)
	}
	if h.err != nil {
		fmt.Fprintf(writer, " ERROR: %v", h.err)
	}
	fmt.Fprintln(writer)
}

func (s *StructLog) writeSnapshot(writer io.Writer) {
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
	Pc                uint64             `json:"pc"`
	Op                string             `json:"op"`
	Gas               uint64             `json:"gas"`
	GasCost           uint64             `json:"gasCost"`
	StateGasCost      int64              `json:"stateGasCost,omitempty"`
	StateGasReservoir uint64             `json:"stateGasReservoir,omitempty"`
	Depth             int                `json:"depth"`
	Error             string             `json:"error,omitempty,omitzero"`
	Stack             *[]string          `json:"stack,omitempty"`
	ReturnData        string             `json:"returnData,omitempty"`
	Memory            *[]string          `json:"memory,omitempty"`
	Storage           *map[string]string `json:"storage,omitempty"`
	RefundCounter     uint64             `json:"refund,omitempty"`
}

func formatMemoryWord(chunk []byte) string {
	if len(chunk) == 32 {
		return hexutil.Encode(chunk)
	}
	var word [32]byte
	copy(word[:], chunk)
	return hexutil.Encode(word[:])
}

// toLegacyJSON converts the structLog to legacy json-encoded legacy form.
func (s *StructLog) toLegacyJSON() json.RawMessage {
	msg := structLogLegacy{
		Pc:                s.Pc,
		Op:                s.Op.String(),
		Gas:               s.Gas,
		GasCost:           s.GasCost,
		StateGasCost:      s.StateGasCost,
		StateGasReservoir: s.StateGasReservoir,
		Depth:             s.Depth,
		Error:             s.ErrorString(),
		RefundCounter:     s.RefundCounter,
	}
	if s.Stack != nil {
		stack := make([]string, len(s.Stack))
		for i, stackValue := range s.Stack {
			stack[i] = stackValue.Hex()
		}
		msg.Stack = &stack
	}
	if len(s.ReturnData) > 0 {
		msg.ReturnData = hexutil.Encode(s.ReturnData)
	}
	if len(s.Memory) > 0 {
		memory := make([]string, 0, (len(s.Memory)+31)/32)
		for i := 0; i < len(s.Memory); i += 32 {
			end := i + 32
			if end > len(s.Memory) {
				end = len(s.Memory)
			}
			memory = append(memory, formatMemoryWord(s.Memory[i:end]))
		}
		msg.Memory = &memory
	}
	if len(s.Storage) > 0 {
		storage := make(map[string]string)
		for i, storageValue := range s.Storage {
			storage[i.Hex()] = storageValue.Hex()
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
// A StructLogger can stream text with one pending opcode or store JSON logs for
// later retrieval.
type StructLogger struct {
	cfg Config
	env *tracing.VMContext

	storage map[common.Address]Storage
	output  []byte
	err     error
	usedGas uint64

	writer     io.Writer         // If set, the logger will stream instead of store logs
	logs       []json.RawMessage // buffer of json-encoded logs
	resultSize int               // total bytes of trace output
	canUpdate  bool              // latest opcode has an entry eligible for a fault update
	text       *pendingTextLog   // used in writer mode

	interrupt atomic.Bool           // Atomic flag to signal execution interruption
	reason    atomic.Pointer[error] // Reason for the interruption, populated by Stop
	skip      bool                  // skip processing hooks.
}

// NewStreamingStructLogger returns a new streaming logger.
func NewStreamingStructLogger(cfg *Config, writer io.Writer) *StructLogger {
	l := NewStructLogger(cfg)
	l.writer = &countingWriter{w: writer}
	l.text = new(pendingTextLog)
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
		OnEnterV2:           l.OnEnter,
		OnExitV2:            l.OnExit,
		OnOpcodeV2:          l.OnOpcode,
		OnFaultV2:           l.OnFault,
	}
}

// OnOpcode logs a new structured log message and pushes it out to the environment
//
// OnOpcode also tracks SLOAD/SSTORE ops to track storage change.
func (l *StructLogger) OnOpcode(pc uint64, opcode byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, rData []byte, depth int, err error) {
	l.finishOpcode()

	// If tracing was interrupted, exit
	if l.interrupt.Load() {
		return
	}
	// Processing a system call.
	if l.skip {
		return
	}
	// check if already accumulated the size of the response.
	if l.cfg.Limit != 0 && l.resultSize >= l.cfg.Limit {
		return
	}
	var (
		op           = vm.OpCode(opcode)
		memory       = scope.MemoryData()
		contractAddr = scope.Address()
		stack        = scope.StackData()
		stackLen     = len(stack)
	)
	log := StructLog{
		Pc:                pc,
		Op:                op,
		Gas:               gas.Execution,
		StateGasReservoir: gas.State,
		GasCost:           cost.Execution,
		StateGasCost:      cost.State,
		MemorySize:        len(memory),
		Depth:             depth,
		RefundCounter:     l.env.StateDB.GetRefund(),
		Err:               err,
	}
	if l.cfg.EnableMemory {
		log.Memory = memory
	}
	if !l.cfg.DisableStack {
		log.Stack = stack
	}
	if l.cfg.EnableReturnData {
		log.ReturnData = rData
	}

	// Copy a snapshot of the current storage to a new container
	// A step that halts with an error does not execute the opcode,
	// so it records no storage.
	var storage Storage
	if !l.cfg.DisableStorage && err == nil && (op == vm.SLOAD || op == vm.SSTORE) {
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
	l.canUpdate = err == nil

	// create a log
	if l.writer == nil {
		entry := log.toLegacyJSON()
		l.resultSize += len(entry)
		l.logs = append(l.logs, entry)
		return
	}
	log.writeSnapshot(&l.text.body)
	l.text.header = log.textHeader()
}

// Flush writes the pending text entry. Frame and transaction callbacks flush
// it automatically. Call Flush after a direct EVM.Run without those callbacks.
func (l *StructLogger) Flush() {
	if l.text == nil || l.text.header == nil {
		return
	}
	l.text.header.write(l.writer)
	l.writer.Write(l.text.body.Bytes())
	l.text.body.Reset()
	l.text.header = nil
	l.canUpdate = false
	l.resultSize = l.writer.(*countingWriter).n
}

// finishOpcode writes any pending text and invalidates the last opcode for fault updates.
func (l *StructLogger) finishOpcode() {
	if l.text != nil {
		l.Flush()
	}
	l.canUpdate = false
}

// OnFault records the exceptional halt on the captured opcode snapshot. The
// costs in the trace are cleared while the pre-execution state is preserved.
func (l *StructLogger) OnFault(pc uint64, opcode byte, _ tracing.Gas, _ tracing.OpCost, _ tracing.OpContext, depth int, err error) {
	if !l.canUpdate || l.skip || l.interrupt.Load() || err == nil {
		return
	}
	l.canUpdate = false

	op := vm.OpCode(opcode)
	if l.writer != nil {
		l.updateTextFault(pc, op, depth, err)
	} else {
		l.updateCachedFault(pc, op, depth, err)
	}
}

func (l *StructLogger) updateTextFault(pc uint64, op vm.OpCode, depth int, err error) {
	header := l.text.header
	if header.pc != pc || header.op != op || header.depth != depth {
		return
	}
	header.err = err
	header.cost = tracing.OpCost{}
	l.Flush()
}

func (l *StructLogger) updateCachedFault(pc uint64, op vm.OpCode, depth int, err error) {
	var (
		last  = len(l.logs) - 1
		entry = l.logs[last]
		log   structLogLegacy
	)
	if err := json.Unmarshal(entry, &log); err != nil {
		return
	}
	if log.Pc != pc || log.Op != op.String() || log.Depth != depth {
		return
	}
	log.Error = err.Error()
	log.GasCost = 0
	log.StateGasCost = 0
	updated, _ := json.Marshal(&log)
	l.resultSize += len(updated) - len(entry)
	l.logs[last] = updated
}

// OnEnter writes the pending parent opcode before the child frame's steps.
func (l *StructLogger) OnEnter(int, byte, common.Address, common.Address, []byte, tracing.Gas, *big.Int) {
	if l.writer != nil {
		l.Flush()
	}
}

// OnExit is called a call frame finishes processing.
func (l *StructLogger) OnExit(depth int, output []byte, gasLeft tracing.Gas, err error, reverted bool) {
	l.finishOpcode()

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
	if p := l.reason.Load(); p != nil {
		return nil, *p
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
	l.reason.Store(&err)
	l.interrupt.Store(true)
}

func (l *StructLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	l.canUpdate = false
	l.env = env
}
func (l *StructLogger) OnSystemCallStart(env *tracing.VMContext) {
	l.finishOpcode()
	l.skip = true
}

func (l *StructLogger) OnSystemCallEnd() {
	l.skip = false
}

func (l *StructLogger) OnTxEnd(receipt *types.Receipt, err error) {
	l.finishOpcode()

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
		log.Write(writer)
	}
}

type mdLogger struct {
	out         io.Writer
	cfg         *Config
	env         *tracing.VMContext
	gas         tracing.Gas // budget the top-level frame was entered with
	isAmsterdam bool        // whether gas has a state dimension (EIP-8037)
	skip        bool
	pending     *mdOpcode
}

type mdOpcode struct {
	textLogHeader
	refund uint64
	stack  string
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
		OnTxEnd:             t.OnTxEnd,
		OnSystemCallStartV2: t.OnSystemCallStart,
		OnSystemCallEnd:     t.OnSystemCallEnd,
		OnEnterV2:           t.OnEnter,
		OnExitV2:            t.OnExit,
		OnOpcodeV2:          t.OnOpcode,
		OnFaultV2:           t.OnFault,
	}
}

func (t *mdLogger) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.flush()
	t.env = env
	t.isAmsterdam = env.Rules.IsAmsterdam
}

func (t *mdLogger) OnTxEnd(*types.Receipt, error) {
	t.flush()
}

func (t *mdLogger) OnSystemCallStart(env *tracing.VMContext) {
	t.flush()
	t.skip = true
}

func (t *mdLogger) OnSystemCallEnd() {
	t.skip = false
}

func (t *mdLogger) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas tracing.Gas, value *big.Int) {
	if t.skip {
		return
	}
	t.flush()
	if depth != 0 {
		return
	}
	t.gas = gas

	target := "to"
	if vm.OpCode(typ) == vm.CREATE {
		target = "create"
	}
	fmt.Fprintf(t.out, "Pre-execution info:\n"+
		"  - from: `%v`\n"+
		"  - %s: `%v`\n"+
		"  - data: `%#x`\n"+
		"  - gas: `%d`\n",
		from.String(), target, to.String(), input, gas.Execution)
	if t.isAmsterdam {
		fmt.Fprintf(t.out, "  - state gas reservoir: `%d`\n", gas.State)
	}
	fmt.Fprintf(t.out, "  - value: `%v` wei\n", value)

	if t.isAmsterdam {
		fmt.Fprintf(t.out, `
|  Pc   |      Op     | Cost | StateCost | Reservoir |   Refund  |   Stack   | Error |
|-------|-------------|------|-----------|-----------|-----------|-----------|-------|
`)
	} else {
		fmt.Fprintf(t.out, `
|  Pc   |      Op     | Cost |   Refund  |   Stack   | Error |
|-------|-------------|------|-----------|-----------|-------|
`)
	}
}

func (t *mdLogger) OnExit(depth int, output []byte, gasLeft tracing.Gas, err error, reverted bool) {
	if t.skip {
		return
	}
	t.flush()
	if depth == 0 {
		var used uint64
		if t.gas.Execution > gasLeft.Execution {
			used = t.gas.Execution - gasLeft.Execution
		}
		fmt.Fprintf(t.out, "\nPost-execution info:\n  - output: `%#x`\n", output)
		if t.isAmsterdam {
			// Each dimension is the budget it started with less what is left:
			// the state one can go up when refills outweigh the charges.
			fmt.Fprintf(t.out, "  - consumed execution gas: `%d`\n", used)
			fmt.Fprintf(t.out, "  - consumed state gas: `%d`\n", int64(t.gas.State)-int64(gasLeft.State))
		} else {
			fmt.Fprintf(t.out, "  - consumed gas: `%d`\n", used)
		}
		fmt.Fprintf(t.out, "  - error: `%v`\n", err)
	}
}

func (t *mdLogger) OnOpcode(pc uint64, op byte, gas tracing.Gas, cost tracing.OpCost, scope tracing.OpContext, _ []byte, depth int, err error) {
	if t.skip {
		return
	}
	t.flush()

	t.pending = &mdOpcode{
		textLogHeader: textLogHeader{
			pc:    pc,
			op:    vm.OpCode(op),
			gas:   gas,
			cost:  cost,
			depth: depth,
			err:   err,
		},
		refund: t.env.StateDB.GetRefund(),
	}
	if !t.cfg.DisableStack {
		stack := scope.StackData()
		values := make([]string, len(stack))
		for i, elem := range stack {
			values[i] = elem.Hex()
		}
		t.pending.stack = "[" + strings.Join(values, ",") + "]"
	}
	if err != nil {
		t.flush()
	}
}

func (t *mdLogger) OnFault(pc uint64, op byte, _ tracing.Gas, _ tracing.OpCost, _ tracing.OpContext, depth int, err error) {
	log := t.pending
	if log == nil || log.pc != pc || log.op != vm.OpCode(op) || log.depth != depth || err == nil {
		return
	}
	log.err = err
	log.cost = tracing.OpCost{}
	t.flush()
}

var mdErrorEscaper = strings.NewReplacer("|", `\|`, "\r", " ", "\n", " ")

func (t *mdLogger) flush() {
	log := t.pending
	if log == nil {
		return
	}
	var message string
	if log.err != nil {
		message = mdErrorEscaper.Replace(log.err.Error())
	}
	if t.isAmsterdam {
		fmt.Fprintf(t.out, "| %4d  | %10v  |  %3d |%10d |%10d |%10v |%10v | %s |\n",
			log.pc, log.op, log.cost.Execution, log.cost.State, log.gas.State, log.refund, log.stack, message)
	} else {
		fmt.Fprintf(t.out, "| %4d  | %10v  |  %3d |%10v |%10v | %s |\n",
			log.pc, log.op, log.cost.Execution, log.refund, log.stack, message)
	}
	t.pending = nil
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
