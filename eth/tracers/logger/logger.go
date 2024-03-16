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
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/codehash"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// Storage represents a contract's storage.
type Storage map[common.Hash]common.Hash

// Copy duplicates the current storage.
func (s Storage) Copy() Storage {
	cpy := make(Storage, len(s))
	for key, value := range s {
		cpy[key] = value
	}
	return cpy
}

// Config are the configuration options for structured logger the EVM
type Config struct {
	EnableMemory     bool // enable memory capture
	DisableStack     bool // disable stack capture
	DisableStorage   bool // disable storage capture
	EnableReturnData bool // enable return data capture
	Debug            bool // print output during capture end
	Limit            int  // maximum length of output, but zero means unlimited
	// Chain overrides, can be used to execute a trace using future fork rules
	Overrides *params.ChainConfig `json:"overrides,omitempty"`
}

//go:generate go run github.com/fjl/gencodec -type StructLog -field-override structLogMarshaling -out gen_structlog.go

// StructLog is emitted to the EVM each cycle and lists information about the current internal state
// prior to the execution of the statement.
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
	// scroll-related
	ExtraData *types.ExtraData `json:"extraData"`
}

func (s *StructLog) clean() {
	s.Memory = s.Memory[:0]
	s.Stack = s.Stack[:0]
	s.ReturnData = s.ReturnData[:0]
	s.Storage = nil
	s.ExtraData = nil
	s.Err = nil
}

func (s *StructLog) getOrInitExtraData() *types.ExtraData {
	if s.ExtraData == nil {
		s.ExtraData = &types.ExtraData{}
	}
	return s.ExtraData
}

// overrides for gencodec
type structLogMarshaling struct {
	Gas         math.HexOrDecimal64
	GasCost     math.HexOrDecimal64
	Memory      hexutil.Bytes
	ReturnData  hexutil.Bytes
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

// StructLogger is an EVM state logger and implements EVMLogger.
//
// StructLogger can capture state based on the given Log configuration and also keeps
// a track record of modified storage which is used in reporting snapshots of the
// contract their storage.
type StructLogger struct {
	cfg Config
	env *vm.EVM

	storage  map[common.Address]Storage
	logs     []StructLog
	output   []byte
	err      error
	gasLimit uint64
	usedGas  uint64

	interrupt atomic.Bool // Atomic flag to signal execution interruption
	reason    error       // Textual reason for the interruption

	statesAffected  map[common.Address]struct{}
	createdAccount  *types.AccountWrapper
	callStackLogInd []int
}

// NewStructLogger returns a new logger
func NewStructLogger(cfg *Config) *StructLogger {
	logger := &StructLogger{
		storage:        make(map[common.Address]Storage),
		statesAffected: make(map[common.Address]struct{}),
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
	l.statesAffected = make(map[common.Address]struct{})
	l.createdAccount = nil
	l.callStackLogInd = nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (l *StructLogger) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	l.env = env

	if create {
		// notice codeHash is set AFTER CreateTx has exited, so here codeHash is still empty
		l.createdAccount = &types.AccountWrapper{
			Address: to,
			// nonce is 1 after EIP158, so we query it from stateDb
			Nonce:   env.StateDB.GetNonce(to),
			Balance: (*hexutil.Big)(value),
		}
	}
	l.statesAffected[from] = struct{}{}
	l.statesAffected[to] = struct{}{}
}

// CaptureState logs a new structured log message and pushes it out to the environment
//
// CaptureState also tracks SLOAD/SSTORE ops to track storage change.
func (l *StructLogger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	// If tracing was interrupted, set the error and stop
	if l.interrupt.Load() {
		return
	}
	// check if already accumulated the specified number of logs
	if l.cfg.Limit != 0 && l.cfg.Limit <= len(l.logs) {
		return
	}

	memory := scope.Memory
	stack := scope.Stack
	contract := scope.Contract
	// Copy a snapshot of the current memory state to a new buffer
	var mem []byte
	if l.cfg.EnableMemory {
		mem = make([]byte, len(memory.Data()))
		copy(mem, memory.Data())
	}
	// Copy a snapshot of the current stack state to a new buffer
	var stck []uint256.Int
	if !l.cfg.DisableStack {
		stck = make([]uint256.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			stck[i] = item
		}
	}
	stackData := stack.Data()
	stackLen := len(stackData)
	// Copy a snapshot of the current storage to a new container
	var storage Storage
	if !l.cfg.DisableStorage && (op == vm.SLOAD || op == vm.SSTORE) {
		// initialise new changed values storage container for this contract
		// if not present.
		if l.storage[contract.Address()] == nil {
			l.storage[contract.Address()] = make(Storage)
		}
		// capture SLOAD opcodes and record the read entry in the local storage
		if op == vm.SLOAD && stackLen >= 1 {
			var (
				address = common.Hash(stackData[stackLen-1].Bytes32())
				value   = l.env.StateDB.GetState(contract.Address(), address)
			)
			l.storage[contract.Address()][address] = value
			storage = l.storage[contract.Address()].Copy()
		} else if op == vm.SSTORE && stackLen >= 2 {
			// capture SSTORE opcodes and record the written entry in the local storage.
			var (
				value   = common.Hash(stackData[stackLen-2].Bytes32())
				address = common.Hash(stackData[stackLen-1].Bytes32())
			)
			l.storage[contract.Address()][address] = value
			storage = l.storage[contract.Address()].Copy()
		}
	}
	var rdata []byte
	if l.cfg.EnableReturnData {
		rdata = make([]byte, len(rData))
		copy(rdata, rData)
	}
	// create a new snapshot of the EVM.
	structLog := StructLog{pc, op, gas, cost, mem, memory.Len(), stck, rdata, storage, depth, l.env.StateDB.GetRefund(), err, nil}

	if !l.cfg.DisableStorage && (op == vm.SLOAD || op == vm.SSTORE) {
		if err := traceStorage(l, scope, structLog.getOrInitExtraData()); err != nil {
			log.Error("Failed to trace data", "opcode", op.String(), "err", err)
		}
	}

	execFuncList, ok := OpcodeExecs[op]
	if ok {
		// execute trace func list.
		for _, exec := range execFuncList {
			if e := exec(l, scope, structLog.getOrInitExtraData()); e != nil {
				log.Error("Failed to trace data", "opcode", op.String(), "err", e)
			}
		}
	}
	// for each "calling" op, pick the caller's state
	switch op {
	case vm.CALL, vm.CALLCODE, vm.STATICCALL, vm.DELEGATECALL, vm.CREATE, vm.CREATE2:
		extraData := structLog.getOrInitExtraData()
		extraData.Caller = append(extraData.Caller, getWrappedAccountForAddr(l, scope.Contract.Address()))
	}
	// in reality it is impossible for CREATE to trigger ErrContractAddressCollision
	if op == vm.CREATE2 && err == nil {
		_ = stack.Data()[stackLen-1] // value
		offset := stack.Data()[stackLen-2]
		size := stack.Data()[stackLen-3]
		salt := stack.Data()[stackLen-4]
		// `CaptureState` is called **before** memory resizing
		// So sometimes we need to auto pad 0.
		code := vm.GetData(scope.Memory.Data(), offset.Uint64(), size.Uint64())

		codeAndHash := &codeAndHash{code: code}

		address := crypto.CreateAddress2(contract.Address(), salt.Bytes32(), codeAndHash.Hash().Bytes())

		contractHash := l.env.StateDB.GetKeccakCodeHash(address)
		if l.env.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != codehash.EmptyKeccakCodeHash) {
			extraData := structLog.getOrInitExtraData()
			wrappedStatus := getWrappedAccountForAddr(l, address)
			extraData.StateList = append(extraData.StateList, wrappedStatus)
			l.statesAffected[address] = struct{}{}
		}
	}

	l.logs = append(l.logs, structLog)
}

// codeAndHash is the same as codeAndHash in core/vm/evm.go
type codeAndHash struct {
	code []byte
	hash common.Hash
}

func (c *codeAndHash) Hash() common.Hash {
	if c.hash == (common.Hash{}) {
		// when calculating CREATE2 address, we use Keccak256 not Poseidon
		c.hash = crypto.Keccak256Hash(c.code)
	}
	return c.hash
}

// CaptureStateAfter for special needs, tracks SSTORE ops and records the storage change.
func (t *StructLogger) CaptureStateAfter(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault
// while running an opcode.
func (l *StructLogger) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (l *StructLogger) CaptureEnd(output []byte, gasUsed uint64, err error) {
	l.output = output
	l.err = err
	if l.cfg.Debug {
		fmt.Printf("%#x\n", output)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
		}
	}
}

func (l *StructLogger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// the last logged op should be CALL/STATICCALL/CALLCODE/CREATE/CREATE2
	lastLogPos := len(l.logs) - 1
	log.Debug("mark call stack", "pos", lastLogPos, "op", l.logs[lastLogPos].Op)
	l.callStackLogInd = append(l.callStackLogInd, lastLogPos)
	// sanity check
	if len(l.callStackLogInd) != l.env.Depth() {
		panic("unexpected evm depth in capture enter")
	}
	l.statesAffected[to] = struct{}{}
	theLog := l.logs[lastLogPos]
	theLog.getOrInitExtraData()
	// handling additional updating for CALL/STATICCALL/CALLCODE/CREATE/CREATE2 only
	// append extraData part for the log, capture the account status (the nonce / balance has been updated in capture enter)
	wrappedStatus := getWrappedAccountForAddr(l, to)
	theLog.ExtraData.StateList = append(theLog.ExtraData.StateList, wrappedStatus)
	// finally we update the caller's status (it is possible that nonce and balance being updated)
	if len(theLog.ExtraData.Caller) == 1 {
		theLog.ExtraData.Caller = append(theLog.ExtraData.Caller, getWrappedAccountForAddr(l, from))
	}
}

// CaptureExit phase, a CREATE has its target address's code being set and queryable
func (l *StructLogger) CaptureExit(output []byte, gasUsed uint64, err error) {
	stackH := len(l.callStackLogInd)
	if stackH == 0 {
		panic("unexpected capture exit occur")
	}

	theLogPos := l.callStackLogInd[stackH-1]
	l.callStackLogInd = l.callStackLogInd[:stackH-1]
	theLog := l.logs[theLogPos]
	// update "forecast" data
	if err != nil {
		theLog.ExtraData.CallFailed = true
	}

	// handling updating for CREATE only
	switch theLog.Op {
	case vm.CREATE, vm.CREATE2:
		// append extraData part for the log whose op is CREATE(2), capture the account status (the codehash would be updated in capture exit)
		dataLen := len(theLog.ExtraData.StateList)
		if dataLen == 0 {
			panic("unexpected data capture for target op")
		}

		lastAccData := theLog.ExtraData.StateList[dataLen-1]
		wrappedStatus := getWrappedAccountForAddr(l, lastAccData.Address)
		theLog.ExtraData.StateList = append(theLog.ExtraData.StateList, wrappedStatus)
		code := getCodeForAddr(l, lastAccData.Address)
		theLog.ExtraData.CodeList = append(theLog.ExtraData.CodeList, hexutil.Encode(code))
	default:
		//do nothing for other op code
		return
	}

}

func (l *StructLogger) GetResult() (json.RawMessage, error) {
	result, err := l.getResult()
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

func (l *StructLogger) GetResultWithL1DataFee(l1DataFee *big.Int) (json.RawMessage, error) {
	result, err := l.getResult()
	if err != nil {
		return nil, err
	}

	result.L1DataFee = (*hexutil.Big)(l1DataFee)
	return json.Marshal(result)
}

func (l *StructLogger) getResult() (*types.ExecutionResult, error) {
	// Tracing aborted
	if l.reason != nil {
		return nil, l.reason
	}
	failed := l.err != nil
	returnData := common.CopyBytes(l.output)
	// Return data when successful and revert reason when reverted, otherwise empty.
	returnVal := fmt.Sprintf("%x", returnData)
	if failed && l.err != vm.ErrExecutionReverted {
		returnVal = ""
	}
	return &types.ExecutionResult{
		Gas:         l.usedGas,
		Failed:      failed,
		ReturnValue: returnVal,
		StructLogs:  FormatLogs(l.StructLogs()),
	}, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (l *StructLogger) Stop(err error) {
	l.reason = err
	l.interrupt.Store(true)
}

func (l *StructLogger) CaptureTxStart(gasLimit uint64) {
	l.gasLimit = gasLimit
}

func (l *StructLogger) CaptureTxEnd(restGas uint64) {
	l.usedGas = l.gasLimit - restGas
}

// StructLogs returns the captured log entries.
func (l *StructLogger) StructLogs() []StructLog { return l.logs }

// Error returns the VM error captured by the trace.
func (l *StructLogger) Error() error { return l.err }

// Output returns the VM return value captured by the trace.
func (l *StructLogger) Output() []byte { return l.output }

// UpdatedAccounts is used to collect all "touched" accounts
func (l *StructLogger) UpdatedAccounts() map[common.Address]struct{} {
	return l.statesAffected
}

// UpdatedStorages is used to collect all "touched" storage slots
func (l *StructLogger) UpdatedStorages() map[common.Address]Storage {
	return l.storage
}

// CreatedAccount return the account data in case it is a create tx
func (l *StructLogger) CreatedAccount() *types.AccountWrapper { return l.createdAccount }

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
		if len(log.Memory) > 0 {
			fmt.Fprintln(writer, "Memory:")
			fmt.Fprint(writer, hex.Dump(log.Memory))
		}
		if len(log.Storage) > 0 {
			fmt.Fprintln(writer, "Storage:")
			for h, item := range log.Storage {
				fmt.Fprintf(writer, "%x: %x\n", h, item)
			}
		}
		if len(log.ReturnData) > 0 {
			fmt.Fprintln(writer, "ReturnData:")
			fmt.Fprint(writer, hex.Dump(log.ReturnData))
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
	cfg *Config
	env *vm.EVM
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

func (t *mdLogger) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	if !create {
		fmt.Fprintf(t.out, "From: `%v`\nTo: `%v`\nData: `%#x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	} else {
		fmt.Fprintf(t.out, "From: `%v`\nCreate at: `%v`\nData: `%#x`\nGas: `%d`\nValue `%v` wei\n",
			from.String(), to.String(),
			input, gas, value)
	}

	fmt.Fprintf(t.out, `
|  Pc   |      Op     | Cost |   Stack   |   RStack  |  Refund |
|-------|-------------|------|-----------|-----------|---------|
`)
}

// CaptureState also tracks SLOAD/SSTORE ops to track storage change.
func (t *mdLogger) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	stack := scope.Stack
	fmt.Fprintf(t.out, "| %4d  | %10v  |  %3d |", pc, op, cost)

	if !t.cfg.DisableStack {
		// format stack
		var a []string
		for _, elem := range stack.Data() {
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

func (t *mdLogger) CaptureStateAfter(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

func (t *mdLogger) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, depth int, err error) {
	fmt.Fprintf(t.out, "\nError: at pc=%d, op=%v: %v\n", pc, op, err)
}

func (t *mdLogger) CaptureEnd(output []byte, gasUsed uint64, err error) {
	fmt.Fprintf(t.out, "\nOutput: `%#x`\nConsumed gas: `%d`\nError: `%v`\n",
		output, gasUsed, err)
}

func (t *mdLogger) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *mdLogger) CaptureExit(output []byte, gasUsed uint64, err error) {}

func (*mdLogger) CaptureTxStart(gasLimit uint64) {}

func (*mdLogger) CaptureTxEnd(restGas uint64) {}

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
			Error:         trace.ErrorString(),
			RefundCounter: trace.RefundCounter,
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = stackValue.Hex()
			}
			formatted[index].Stack = &stack
		}
		if trace.ReturnData != nil && len(trace.ReturnData) > 0 {
			formatted[index].ReturnData = hexutil.Bytes(trace.ReturnData).String()
		}
		if trace.Memory != nil {
			memory := make([]string, 0, (len(trace.Memory)+31)/32)
			for i := 0; i+32 <= len(trace.Memory); i += 32 {
				memory = append(memory, fmt.Sprintf("%x", trace.Memory[i:i+32]))
			}
			formatted[index].Memory = &memory
		}
		if trace.Storage != nil {
			storage := make(map[string]string)
			for i, storageValue := range trace.Storage {
				storage[fmt.Sprintf("%x", i)] = fmt.Sprintf("%x", storageValue)
			}
			formatted[index].Storage = &storage
		}
	}
	return formatted
}
