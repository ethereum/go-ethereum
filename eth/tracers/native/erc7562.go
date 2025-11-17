// Copyright 2025 The go-ethereum Authors
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

package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"slices"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/internal"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/fjl/gencodec -type callFrameWithOpcodes -field-override callFrameWithOpcodesMarshaling -out gen_callframewithopcodes_json.go

func init() {
	tracers.DefaultDirectory.Register("erc7562Tracer", newErc7562Tracer, false)
}

type contractSizeWithOpcode struct {
	ContractSize int       `json:"contractSize"`
	Opcode       vm.OpCode `json:"opcode"`
}

type callFrameWithOpcodes struct {
	Type             vm.OpCode       `json:"-"`
	From             common.Address  `json:"from"`
	Gas              uint64          `json:"gas"`
	GasUsed          uint64          `json:"gasUsed"`
	To               *common.Address `json:"to,omitempty" rlp:"optional"`
	Input            []byte          `json:"input" rlp:"optional"`
	Output           []byte          `json:"output,omitempty" rlp:"optional"`
	Error            string          `json:"error,omitempty" rlp:"optional"`
	RevertReason     string          `json:"revertReason,omitempty"`
	Logs             []callLog       `json:"logs,omitempty" rlp:"optional"`
	Value            *big.Int        `json:"value,omitempty" rlp:"optional"`
	revertedSnapshot bool

	AccessedSlots     accessedSlots                              `json:"accessedSlots"`
	ExtCodeAccessInfo []common.Address                           `json:"extCodeAccessInfo"`
	UsedOpcodes       map[vm.OpCode]uint64                       `json:"usedOpcodes"`
	ContractSize      map[common.Address]*contractSizeWithOpcode `json:"contractSize"`
	OutOfGas          bool                                       `json:"outOfGas"`
	// Keccak preimages for the whole transaction are stored in the
	// root call frame.
	KeccakPreimages [][]byte               `json:"keccak,omitempty"`
	Calls           []callFrameWithOpcodes `json:"calls,omitempty" rlp:"optional"`
}

func (f callFrameWithOpcodes) TypeString() string {
	return f.Type.String()
}

func (f callFrameWithOpcodes) failed() bool {
	return len(f.Error) > 0 && f.revertedSnapshot
}

func (f *callFrameWithOpcodes) processOutput(output []byte, err error, reverted bool) {
	output = common.CopyBytes(output)
	// Clear error if tx wasn't reverted. This happened
	// for pre-homestead contract storage OOG.
	if err != nil && !reverted {
		err = nil
	}
	if err == nil {
		f.Output = output
		return
	}
	f.Error = err.Error()
	f.revertedSnapshot = reverted
	if f.Type == vm.CREATE || f.Type == vm.CREATE2 {
		f.To = nil
	}
	if !errors.Is(err, vm.ErrExecutionReverted) || len(output) == 0 {
		return
	}
	f.Output = output
	if len(output) < 4 {
		return
	}
	if unpacked, err := abi.UnpackRevert(output); err == nil {
		f.RevertReason = unpacked
	}
}

type callFrameWithOpcodesMarshaling struct {
	TypeString      string `json:"type"`
	Gas             hexutil.Uint64
	GasUsed         hexutil.Uint64
	Value           *hexutil.Big
	Input           hexutil.Bytes
	Output          hexutil.Bytes
	UsedOpcodes     map[hexutil.Uint64]uint64
	KeccakPreimages []hexutil.Bytes
}

type accessedSlots struct {
	Reads           map[common.Hash][]common.Hash `json:"reads"`
	Writes          map[common.Hash]uint64        `json:"writes"`
	TransientReads  map[common.Hash]uint64        `json:"transientReads"`
	TransientWrites map[common.Hash]uint64        `json:"transientWrites"`
}

type opcodeWithPartialStack struct {
	Opcode        vm.OpCode
	StackTopItems []uint256.Int
}

type erc7562Tracer struct {
	config    erc7562TracerConfig
	gasLimit  uint64
	interrupt atomic.Bool // Atomic flag to signal execution interruption
	reason    error       // Textual reason for the interruption
	env       *tracing.VMContext

	ignoredOpcodes       map[vm.OpCode]struct{}
	callstackWithOpcodes []callFrameWithOpcodes
	lastOpWithStack      *opcodeWithPartialStack
	keccakPreimages      map[string]struct{}
}

// newErc7562Tracer returns a native go tracer which tracks
// call frames of a tx, and implements vm.EVMLogger.
func newErc7562Tracer(ctx *tracers.Context, cfg json.RawMessage, _ *params.ChainConfig) (*tracers.Tracer, error) {
	t, err := newErc7562TracerObject(cfg)
	if err != nil {
		return nil, err
	}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: t.OnTxStart,
			OnOpcode:  t.OnOpcode,
			OnTxEnd:   t.OnTxEnd,
			OnEnter:   t.OnEnter,
			OnExit:    t.OnExit,
			OnLog:     t.OnLog,
		},
		GetResult: t.GetResult,
		Stop:      t.Stop,
	}, nil
}

type erc7562TracerConfig struct {
	StackTopItemsSize int              `json:"stackTopItemsSize"`
	IgnoredOpcodes    []hexutil.Uint64 `json:"ignoredOpcodes"` // Opcodes to ignore during OnOpcode hook execution
	WithLog           bool             `json:"withLog"`        // If true, erc7562 tracer will collect event logs
}

func getFullConfiguration(partial erc7562TracerConfig) erc7562TracerConfig {
	config := partial

	if config.IgnoredOpcodes == nil {
		config.IgnoredOpcodes = defaultIgnoredOpcodes()
	}
	if config.StackTopItemsSize == 0 {
		config.StackTopItemsSize = 3
	}

	return config
}

func newErc7562TracerObject(cfg json.RawMessage) (*erc7562Tracer, error) {
	var config erc7562TracerConfig
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	fullConfig := getFullConfiguration(config)
	// Create a map of ignored opcodes for fast lookup
	ignoredOpcodes := make(map[vm.OpCode]struct{}, len(fullConfig.IgnoredOpcodes))
	for _, op := range fullConfig.IgnoredOpcodes {
		ignoredOpcodes[vm.OpCode(op)] = struct{}{}
	}
	// First callframe contains tx context info
	// and is populated on start and end.
	return &erc7562Tracer{
		callstackWithOpcodes: make([]callFrameWithOpcodes, 0, 1),
		config:               fullConfig,
		keccakPreimages:      make(map[string]struct{}),
		ignoredOpcodes:       ignoredOpcodes,
	}, nil
}

func (t *erc7562Tracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.env = env
	t.gasLimit = tx.Gas()
}

// OnEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *erc7562Tracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted
	if t.interrupt.Load() {
		return
	}

	toCopy := to
	call := callFrameWithOpcodes{
		Type:  vm.OpCode(typ),
		From:  from,
		To:    &toCopy,
		Input: common.CopyBytes(input),
		Gas:   gas,
		Value: value,
		AccessedSlots: accessedSlots{
			Reads:           map[common.Hash][]common.Hash{},
			Writes:          map[common.Hash]uint64{},
			TransientReads:  map[common.Hash]uint64{},
			TransientWrites: map[common.Hash]uint64{},
		},
		UsedOpcodes:       map[vm.OpCode]uint64{},
		ExtCodeAccessInfo: make([]common.Address, 0),
		ContractSize:      map[common.Address]*contractSizeWithOpcode{},
	}
	if depth == 0 {
		call.Gas = t.gasLimit
	}
	t.callstackWithOpcodes = append(t.callstackWithOpcodes, call)
}

func (t *erc7562Tracer) captureEnd(output []byte, err error, reverted bool) {
	if len(t.callstackWithOpcodes) != 1 {
		return
	}
	t.callstackWithOpcodes[0].processOutput(output, err, reverted)
}

// OnExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *erc7562Tracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if t.interrupt.Load() {
		return
	}
	if depth == 0 {
		t.captureEnd(output, err, reverted)
		return
	}

	size := len(t.callstackWithOpcodes)
	if size <= 1 {
		return
	}
	// Pop call.
	call := t.callstackWithOpcodes[size-1]
	t.callstackWithOpcodes = t.callstackWithOpcodes[:size-1]
	size -= 1

	if errors.Is(err, vm.ErrCodeStoreOutOfGas) || errors.Is(err, vm.ErrOutOfGas) {
		call.OutOfGas = true
	}
	call.GasUsed = gasUsed
	call.processOutput(output, err, reverted)
	// Nest call into parent.
	t.callstackWithOpcodes[size-1].Calls = append(t.callstackWithOpcodes[size-1].Calls, call)
}

func (t *erc7562Tracer) OnTxEnd(receipt *types.Receipt, err error) {
	if t.interrupt.Load() {
		return
	}
	// Error happened during tx validation.
	if err != nil {
		return
	}
	t.callstackWithOpcodes[0].GasUsed = receipt.GasUsed
	if t.config.WithLog {
		// Logs are not emitted when the call fails
		t.clearFailedLogs(&t.callstackWithOpcodes[0], false)
	}
}

func (t *erc7562Tracer) OnLog(log1 *types.Log) {
	// Only logs need to be captured via opcode processing
	if !t.config.WithLog {
		return
	}
	// Skip if tracing was interrupted
	if t.interrupt.Load() {
		return
	}
	l := callLog{
		Address:  log1.Address,
		Topics:   log1.Topics,
		Data:     log1.Data,
		Position: hexutil.Uint(len(t.callstackWithOpcodes[len(t.callstackWithOpcodes)-1].Calls)),
	}
	t.callstackWithOpcodes[len(t.callstackWithOpcodes)-1].Logs = append(t.callstackWithOpcodes[len(t.callstackWithOpcodes)-1].Logs, l)
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *erc7562Tracer) GetResult() (json.RawMessage, error) {
	if t.interrupt.Load() {
		return nil, t.reason
	}
	if len(t.callstackWithOpcodes) != 1 {
		return nil, errors.New("incorrect number of top-level calls")
	}

	keccak := make([][]byte, 0, len(t.callstackWithOpcodes[0].KeccakPreimages))
	for k := range t.keccakPreimages {
		keccak = append(keccak, []byte(k))
	}
	t.callstackWithOpcodes[0].KeccakPreimages = keccak
	slices.SortFunc(keccak, func(a, b []byte) int {
		return bytes.Compare(a, b)
	})

	enc, err := json.Marshal(t.callstackWithOpcodes[0])
	if err != nil {
		return nil, err
	}

	return enc, t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *erc7562Tracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}

// clearFailedLogs clears the logs of a callframe and all its children
// in case of execution failure.
func (t *erc7562Tracer) clearFailedLogs(cf *callFrameWithOpcodes, parentFailed bool) {
	failed := cf.failed() || parentFailed
	// Clear own logs
	if failed {
		cf.Logs = nil
	}
	for i := range cf.Calls {
		t.clearFailedLogs(&cf.Calls[i], failed)
	}
}

func (t *erc7562Tracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if t.interrupt.Load() {
		return
	}
	var (
		opcode          = vm.OpCode(op)
		opcodeWithStack *opcodeWithPartialStack
		stackSize       = len(scope.StackData())
		stackLimit      = min(stackSize, t.config.StackTopItemsSize)
		stackTopItems   = make([]uint256.Int, stackLimit)
	)
	for i := 0; i < stackLimit; i++ {
		stackTopItems[i] = *peepStack(scope.StackData(), i)
	}
	opcodeWithStack = &opcodeWithPartialStack{
		Opcode:        opcode,
		StackTopItems: stackTopItems,
	}
	t.handleReturnRevert(opcode)
	size := len(t.callstackWithOpcodes)
	currentCallFrame := &t.callstackWithOpcodes[size-1]
	if t.lastOpWithStack != nil {
		t.handleExtOpcodes(opcode, currentCallFrame)
	}
	t.handleAccessedContractSize(opcode, scope, currentCallFrame)
	if t.lastOpWithStack != nil {
		t.handleGasObserved(opcode, currentCallFrame)
	}
	t.storeUsedOpcode(opcode, currentCallFrame)
	t.handleStorageAccess(opcode, scope, currentCallFrame)
	t.storeKeccak(opcode, scope)
	t.lastOpWithStack = opcodeWithStack
}

func (t *erc7562Tracer) handleReturnRevert(opcode vm.OpCode) {
	if opcode == vm.REVERT || opcode == vm.RETURN {
		t.lastOpWithStack = nil
	}
}

func (t *erc7562Tracer) handleGasObserved(opcode vm.OpCode, currentCallFrame *callFrameWithOpcodes) {
	// [OP-012]
	pendingGasObserved := t.lastOpWithStack.Opcode == vm.GAS && !isCall(opcode)
	if pendingGasObserved {
		currentCallFrame.UsedOpcodes[vm.GAS]++
	}
}

func (t *erc7562Tracer) storeUsedOpcode(opcode vm.OpCode, currentCallFrame *callFrameWithOpcodes) {
	// ignore "unimportant" opcodes
	if opcode != vm.GAS && !t.isIgnoredOpcode(opcode) {
		currentCallFrame.UsedOpcodes[opcode]++
	}
}

func (t *erc7562Tracer) handleStorageAccess(opcode vm.OpCode, scope tracing.OpContext, currentCallFrame *callFrameWithOpcodes) {
	if opcode == vm.SLOAD || opcode == vm.SSTORE || opcode == vm.TLOAD || opcode == vm.TSTORE {
		slot := common.BytesToHash(peepStack(scope.StackData(), 0).Bytes())
		addr := scope.Address()

		if opcode == vm.SLOAD {
			// read slot values before this UserOp was created
			// (so saving it if it was written before the first read)
			_, rOk := currentCallFrame.AccessedSlots.Reads[slot]
			_, wOk := currentCallFrame.AccessedSlots.Writes[slot]
			if !rOk && !wOk {
				currentCallFrame.AccessedSlots.Reads[slot] = append(currentCallFrame.AccessedSlots.Reads[slot], t.env.StateDB.GetState(addr, slot))
			}
		} else if opcode == vm.SSTORE {
			currentCallFrame.AccessedSlots.Writes[slot]++
		} else if opcode == vm.TLOAD {
			currentCallFrame.AccessedSlots.TransientReads[slot]++
		} else {
			currentCallFrame.AccessedSlots.TransientWrites[slot]++
		}
	}
}

func (t *erc7562Tracer) storeKeccak(opcode vm.OpCode, scope tracing.OpContext) {
	if opcode == vm.KECCAK256 {
		dataOffset := peepStack(scope.StackData(), 0).Uint64()
		dataLength := peepStack(scope.StackData(), 1).Uint64()
		preimage, err := internal.GetMemoryCopyPadded(scope.MemoryData(), int64(dataOffset), int64(dataLength))
		if err != nil {
			log.Warn("erc7562Tracer: failed to copy keccak preimage from memory", "err", err)
			return
		}
		t.keccakPreimages[string(preimage)] = struct{}{}
	}
}

func (t *erc7562Tracer) handleExtOpcodes(opcode vm.OpCode, currentCallFrame *callFrameWithOpcodes) {
	if isEXT(t.lastOpWithStack.Opcode) {
		addr := common.HexToAddress(t.lastOpWithStack.StackTopItems[0].Hex())

		// only store the last EXTCODE* opcode per address - could even be a boolean for our current use-case
		// [OP-051]

		if !(t.lastOpWithStack.Opcode == vm.EXTCODESIZE && opcode == vm.ISZERO) {
			currentCallFrame.ExtCodeAccessInfo = append(currentCallFrame.ExtCodeAccessInfo, addr)
		}
	}
}

func (t *erc7562Tracer) handleAccessedContractSize(opcode vm.OpCode, scope tracing.OpContext, currentCallFrame *callFrameWithOpcodes) {
	// [OP-041]
	if isEXTorCALL(opcode) {
		n := 0
		if !isEXT(opcode) {
			n = 1
		}
		addr := common.BytesToAddress(peepStack(scope.StackData(), n).Bytes())
		if _, ok := currentCallFrame.ContractSize[addr]; !ok {
			currentCallFrame.ContractSize[addr] = &contractSizeWithOpcode{
				ContractSize: len(t.env.StateDB.GetCode(addr)),
				Opcode:       opcode,
			}
		}
	}
}

func peepStack(stackData []uint256.Int, n int) *uint256.Int {
	return &stackData[len(stackData)-n-1]
}

func isEXTorCALL(opcode vm.OpCode) bool {
	return isEXT(opcode) || isCall(opcode)
}

func isEXT(opcode vm.OpCode) bool {
	return opcode == vm.EXTCODEHASH ||
		opcode == vm.EXTCODESIZE ||
		opcode == vm.EXTCODECOPY
}

func isCall(opcode vm.OpCode) bool {
	return opcode == vm.CALL ||
		opcode == vm.CALLCODE ||
		opcode == vm.DELEGATECALL ||
		opcode == vm.STATICCALL
}

// Check if this opcode is ignored for the purposes of generating the used opcodes report
func (t *erc7562Tracer) isIgnoredOpcode(opcode vm.OpCode) bool {
	if _, ok := t.ignoredOpcodes[opcode]; ok {
		return true
	}
	return false
}

func defaultIgnoredOpcodes() []hexutil.Uint64 {
	ignored := make([]hexutil.Uint64, 0, 64)

	// Allow all PUSHx, DUPx and SWAPx opcodes as they have sequential codes
	for op := vm.PUSH0; op < vm.SWAP16; op++ {
		ignored = append(ignored, hexutil.Uint64(op))
	}

	for _, op := range []vm.OpCode{
		vm.POP, vm.ADD, vm.SUB, vm.MUL,
		vm.DIV, vm.EQ, vm.LT, vm.GT,
		vm.SLT, vm.SGT, vm.SHL, vm.SHR,
		vm.AND, vm.OR, vm.NOT, vm.ISZERO,
	} {
		ignored = append(ignored, hexutil.Uint64(op))
	}

	return ignored
}
