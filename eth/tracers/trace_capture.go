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

package tracers

import (
	"bytes"
	"errors"
	"math/big"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

const (
	traceEntryLimit = 1_000_000
	traceByteLimit  = 64 << 20
)

// traceCapture derives all output families from the same execution. State diffs
// compare transaction snapshots, so reverted writes never survive in the result.
type traceCapture struct {
	state       *state.StateDB
	before      *state.StateDB
	kinds       TraceTypes
	table       vm.JumpTable
	precompiles []common.Address
	frames      []*TraceFrame
	scopes      []*traceScope
	output      hexutil.Bytes
	rootVM      *traceVM
	touched     map[common.Address]map[common.Hash]bool
	entries     int
	bytes       int
	err         error
	stop        func()
}

type traceScope struct {
	frame   *TraceFrame
	hidden  bool
	target  common.Address
	vm      *traceVM
	pending *tracePendingOp
	last    *traceVMOp
}

type tracePendingOp struct {
	op           *traceVMOp
	stackBase    int
	memoryOffset uint64
	memorySize   uint64
	callMemory   bool
	returnSize   uint64
	store        *traceStorageWrite
}

func newTraceCapture(st *state.StateDB, kinds TraceTypes, rules params.Rules) *traceCapture {
	table, _ := vm.LookupInstructionSet(rules)
	c := &traceCapture{state: st, kinds: kinds, table: table, precompiles: vm.ActivePrecompiles(rules), frames: []*TraceFrame{}, touched: make(map[common.Address]map[common.Hash]bool)}
	if kinds.has("stateDiff") {
		c.before = st.Copy()
	}
	return c
}

func (c *traceCapture) hooks() *tracing.Hooks {
	h := &tracing.Hooks{OnEnter: c.enter, OnExit: c.exit}
	if c.kinds.has("vmTrace") {
		h.OnOpcode = c.opcode
		h.OnFault = c.fault
	}
	if c.before != nil {
		h.OnBalanceChange = func(a common.Address, prev, next *big.Int, reason tracing.BalanceChangeReason) { c.touch(a) }
		h.OnNonceChangeV2 = func(a common.Address, prev, next uint64, reason tracing.NonceChangeReason) { c.touch(a) }
		h.OnCodeChangeV2 = func(a common.Address, prevHash common.Hash, prev []byte, nextHash common.Hash, next []byte, reason tracing.CodeChangeReason) {
			c.touch(a)
		}
		h.OnStorageChange = func(a common.Address, key, prev, next common.Hash) { c.touch(a); c.touched[a][key] = true }
	}
	return h
}

func (c *traceCapture) reserve(size int) bool {
	c.bytes += size
	if c.bytes > traceByteLimit || c.entries > traceEntryLimit {
		c.err = errors.New("trace output limit exceeded")
		if c.stop != nil {
			c.stop()
		}
	}
	return c.err == nil
}

func (c *traceCapture) touch(address common.Address) {
	if c.before == nil {
		return
	}
	if c.touched[address] == nil {
		c.touched[address] = make(map[common.Hash]bool)
	}
}

func (c *traceCapture) enter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	if c.err != nil {
		return
	}
	c.entries++
	if !c.reserve(0) {
		return
	}
	c.touch(from)
	c.touch(to)
	// Parity clients omit nested zero-value precompiles from the call tree.
	// Keep the execution scope: its return bytes still feed the caller's VM delta.
	hidden := false
	switch vm.OpCode(typ) {
	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		hidden = len(c.scopes) > 0 && (value == nil || value.Sign() == 0) && slices.Contains(c.precompiles, to)
	}
	path := []uint64{}
	if len(c.scopes) > 0 {
		parent := c.scopes[len(c.scopes)-1].frame
		path = append(append([]uint64{}, parent.TraceAddress...), parent.Subtraces)
		if !hidden {
			parent.Subtraces++
		}
	}
	frame := &TraceFrame{TraceAddress: path, Type: "call"}
	amount := new(big.Int)
	if value != nil {
		amount.Set(value)
	}
	switch vm.OpCode(typ) {
	case vm.CREATE, vm.CREATE2:
		frame.Type = "create"
		if c.kinds.has("trace") {
			frame.Action = traceCreateAction{from, hexutil.Uint64(gas), common.CopyBytes(input), (*hexutil.Big)(amount), strings.ToLower(vm.OpCode(typ).String())}
		}
	case vm.SELFDESTRUCT:
		frame.Type = "suicide"
		if c.kinds.has("trace") {
			frame.Action = traceSuicideAction{from, to, (*hexutil.Big)(amount)}
		}
	default:
		if c.kinds.has("trace") {
			frame.Action = traceCallAction{strings.ToLower(vm.OpCode(typ).String()), from, to, hexutil.Uint64(gas), common.CopyBytes(input), (*hexutil.Big)(amount)}
		}
	}
	if c.kinds.has("trace") && !hidden {
		if !c.reserve(len(input)) {
			return
		}
		c.frames = append(c.frames, frame)
	}
	c.scopes = append(c.scopes, &traceScope{frame: frame, target: to, hidden: hidden})
}

func (c *traceCapture) exit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if c.err != nil || len(c.scopes) == 0 {
		return
	}
	scope := c.scopes[len(c.scopes)-1]
	c.scopes = c.scopes[:len(c.scopes)-1]
	if len(c.scopes) == 0 {
		c.output = common.CopyBytes(output)
		c.reserve(len(output))
	}
	if len(c.scopes) > 0 {
		parent := c.scopes[len(c.scopes)-1]
		if parent.pending != nil {
			parent.pending.returnSize = uint64(len(output))
		}
	}
	if !c.kinds.has("trace") || scope.hidden {
		return
	}
	frame := scope.frame
	if err != nil && reverted {
		frame.Error = err.Error()
		if errors.Is(err, vm.ErrExecutionReverted) || strings.Contains(err.Error(), "execution reverted") {
			frame.Error = "Reverted"
			frame.Result = traceCallResult{hexutil.Uint64(gasUsed), common.CopyBytes(output)}
		}
	} else if frame.Type == "create" {
		frame.Result = traceCreateResult{hexutil.Uint64(gasUsed), scope.target, common.CopyBytes(output)}
	} else if frame.Type == "call" {
		frame.Result = traceCallResult{hexutil.Uint64(gasUsed), common.CopyBytes(output)}
	}
	c.reserve(len(output))
}

func (c *traceCapture) result() *TraceExecution {
	result := &TraceExecution{Output: c.output, Trace: c.frames, VMTrace: c.rootVM}
	if c.before != nil {
		result.StateDiff = c.stateDiff()
	}
	return result
}

func traceChange(before, after any, oldExists, newExists, equal bool) any {
	switch {
	case !oldExists && newExists:
		return map[string]any{"+": after}
	case oldExists && !newExists:
		return map[string]any{"-": before}
	case equal:
		return "="
	default:
		return map[string]any{"*": map[string]any{"from": before, "to": after}}
	}
}

func (c *traceCapture) stateDiff() map[common.Address]*traceAccountDiff {
	result := make(map[common.Address]*traceAccountDiff)
	for address, slots := range c.touched {
		oldExists, newExists := c.before.Exist(address), c.state.Exist(address)
		if !oldExists && !newExists {
			continue
		}
		oldBalance, newBalance := c.before.GetBalance(address), c.state.GetBalance(address)
		oldNonce, newNonce := c.before.GetNonce(address), c.state.GetNonce(address)
		oldCode, newCode := c.before.GetCode(address), c.state.GetCode(address)
		// Slots follow the account's existence: zero words on a surviving account
		// are changes, not slot creations or deletions.
		storage := make(map[common.Hash]any)
		for key := range slots {
			oldValue, newValue := c.before.GetState(address, key), c.state.GetState(address, key)
			if oldValue == newValue {
				continue
			}
			storage[key] = traceChange(oldValue, newValue, oldExists, newExists, false)
		}
		if oldExists == newExists && oldBalance.Eq(newBalance) && oldNonce == newNonce && bytes.Equal(oldCode, newCode) && len(storage) == 0 {
			continue
		}
		result[address] = &traceAccountDiff{
			Balance: traceChange((*hexutil.Big)(oldBalance.ToBig()), (*hexutil.Big)(newBalance.ToBig()), oldExists, newExists, oldBalance.Eq(newBalance)),
			Nonce:   traceChange(hexutil.Uint64(oldNonce), hexutil.Uint64(newNonce), oldExists, newExists, oldNonce == newNonce),
			Code:    traceChange(hexutil.Bytes(oldCode), hexutil.Bytes(newCode), oldExists, newExists, bytes.Equal(oldCode, newCode)),
			Storage: storage,
		}
	}
	return result
}

func (c *traceCapture) opcode(pc uint64, op byte, gas, cost uint64, context tracing.OpContext, returnData []byte, depth int, err error) {
	if c.err != nil || len(c.scopes) == 0 {
		return
	}
	scope := c.scopes[len(c.scopes)-1]
	c.finishOp(scope, context, gas)
	if scope.vm == nil {
		code := context.ContractCode()
		if !c.reserve(len(code)) {
			return
		}
		scope.vm = &traceVM{Code: common.CopyBytes(code), Ops: []*traceVMOp{}}
		if len(c.scopes) == 1 {
			c.rootVM = scope.vm
		} else {
			parent := c.scopes[len(c.scopes)-2]
			if parent.pending != nil {
				parent.pending.op.Sub = scope.vm
			}
		}
	}
	c.entries++
	if !c.reserve(0) {
		return
	}
	step := &traceVMOp{PC: pc, Cost: cost, Op: vm.OpCode(op).String()}
	scope.vm.Ops = append(scope.vm.Ops, step)
	scope.last = step
	if err != nil || cost > gas {
		return
	}
	// Terminal instructions have no pushed values or writes. Capture them while
	// the interpreter still owns its stack; OnExit runs after those pools are freed.
	switch vm.OpCode(op) {
	case vm.STOP, vm.RETURN, vm.REVERT, vm.SELFDESTRUCT:
		step.Ex = &traceVMEffects{Used: gas - cost, Push: []*hexutil.Big{}}
		return
	}
	stack := context.StackData()
	minStack, _ := c.table[op].Stack()
	pending := &tracePendingOp{op: step, stackBase: len(stack) - minStack}
	word := func(i int) uint64 {
		if len(stack) <= i {
			return 0
		}
		return stack[len(stack)-1-i].Uint64()
	}
	switch vm.OpCode(op) {
	case vm.MSTORE:
		pending.memoryOffset, pending.memorySize = word(0), 32
	case vm.MSTORE8:
		pending.memoryOffset, pending.memorySize = word(0), 1
	case vm.CALLDATACOPY, vm.CODECOPY, vm.RETURNDATACOPY, vm.MCOPY:
		pending.memoryOffset, pending.memorySize = word(0), word(2)
	case vm.EXTCODECOPY:
		pending.memoryOffset, pending.memorySize = word(1), word(3)
	case vm.CALL, vm.CALLCODE:
		pending.memoryOffset, pending.memorySize, pending.callMemory = word(5), word(6), true
	case vm.DELEGATECALL, vm.STATICCALL:
		pending.memoryOffset, pending.memorySize, pending.callMemory = word(4), word(5), true
	case vm.SSTORE:
		if len(stack) >= 2 {
			pending.store = &traceStorageWrite{(*hexutil.Big)(stack[len(stack)-1].ToBig()), (*hexutil.Big)(stack[len(stack)-2].ToBig())}
		}
	}
	scope.pending = pending
}

func (c *traceCapture) finishOp(scope *traceScope, context tracing.OpContext, gas uint64) {
	pending := scope.pending
	if pending == nil {
		return
	}
	scope.pending = nil
	stack := context.StackData()
	if pending.stackBase < 0 || pending.stackBase > len(stack) {
		c.err = errors.New("invalid trace stack transition")
		if c.stop != nil {
			c.stop()
		}
		return
	}
	effects := &traceVMEffects{Used: gas, Push: []*hexutil.Big{}, Store: pending.store}
	for i := pending.stackBase; i < len(stack); i++ {
		effects.Push = append(effects.Push, (*hexutil.Big)(stack[i].ToBig()))
	}
	offset, size := pending.memoryOffset, pending.memorySize
	if pending.callMemory && size > pending.returnSize {
		size = pending.returnSize
	}
	memory := context.MemoryData()
	if size > 0 && offset <= uint64(len(memory)) && size <= uint64(len(memory))-offset {
		if !c.reserve(int(size)) {
			return
		}
		effects.Mem = &traceMemoryWrite{offset, common.CopyBytes(memory[offset : offset+size])}
	}
	pending.op.Ex = effects
	c.reserve(len(effects.Push) * 32)
}

func (c *traceCapture) fault(pc uint64, op byte, gas, cost uint64, context tracing.OpContext, depth int, err error) {
	if c.err != nil || len(c.scopes) == 0 {
		return
	}
	if vm.OpCode(op) == vm.REVERT && (errors.Is(err, vm.ErrExecutionReverted) || strings.Contains(err.Error(), "execution reverted")) {
		return
	}
	scope := c.scopes[len(c.scopes)-1]
	if scope.last != nil && scope.last.PC == pc {
		scope.last.Ex = nil
	}
	scope.pending = nil
}
