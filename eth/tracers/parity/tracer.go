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

// Package parity implements the Parity-style trace output families as a
// tracer: the call frames of trace, the account and storage transitions of
// stateDiff and the instruction-level vmTrace.
package parity

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func init() {
	tracers.DefaultDirectory.Register("parityTracer", newTracer, false)
}

const (
	entryLimit = 1_000_000
	byteLimit  = 64 << 20
)

// Config selects the output families of a Tracer.
type Config struct {
	Trace     bool `json:"trace"`
	VMTrace   bool `json:"vmTrace"`
	StateDiff bool `json:"stateDiff"`
}

// Tracer derives all output families from the same execution of one
// transaction. State diffs compare the pre-transaction state of every account
// the transaction touches with the finalized post-state, so reverted writes
// never survive in the result.
type Tracer struct {
	config      Config
	chainConfig *params.ChainConfig
	state       tracing.StateDB
	pre         map[common.Address]*accountState // nil unless stateDiff is selected
	table       vm.JumpTable
	precompiles vm.PrecompiledContracts
	frames      []*Frame
	scopes      []*frameScope
	output      hexutil.Bytes
	rootVM      *VMTrace
	entries     int
	bytes       int
	err         error                 // output limit or stack failure, set during execution
	reason      atomic.Pointer[error] // set by Stop, possibly from another goroutine
}

type frameScope struct {
	frame   *Frame
	hidden  bool
	target  common.Address
	vm      *VMTrace
	pending *pendingOp
	last    *VMOp
}

// accountState is an account before the transaction, with the original values
// of the slots it writes.
type accountState struct {
	exists  bool
	balance *uint256.Int
	nonce   uint64
	code    []byte
	storage map[common.Hash]common.Hash
}

type pendingOp struct {
	op           *VMOp
	stackBase    int
	memoryOffset uint64
	memorySize   uint64
	store        *StorageWrite
}

// newTracer constructs the parityTracer of the tracer directory. Without a
// selected output family it records the call frames. Calls nested in a
// precompile are recognized by the precompiles of the executing fork, so
// precompiles moved by state overrides are not.
func newTracer(ctx *tracers.Context, cfg json.RawMessage, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	var config Config
	if cfg != nil {
		if err := json.Unmarshal(cfg, &config); err != nil {
			return nil, err
		}
	}
	if config == (Config{}) {
		config.Trace = true
	}
	t := New(config, chainConfig, nil)
	return &tracers.Tracer{
		Hooks: t.Hooks(),
		GetResult: func() (json.RawMessage, error) {
			result, err := t.Result()
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		},
		Stop: t.Stop,
	}, nil
}

// New returns a tracer of the selected output families. Precompiles are the
// contracts calls run as precompiles; nil selects those of the executing fork.
func New(config Config, chainConfig *params.ChainConfig, precompiles vm.PrecompiledContracts) *Tracer {
	t := &Tracer{config: config, chainConfig: chainConfig, precompiles: precompiles, frames: []*Frame{}}
	if config.StateDiff {
		t.pre = make(map[common.Address]*accountState)
	}
	return t
}

// Hooks returns the tracing hooks of the selected output families.
func (t *Tracer) Hooks() *tracing.Hooks {
	h := &tracing.Hooks{OnTxStart: t.txStart, OnEnter: t.enter, OnExit: t.exit}
	if t.config.VMTrace || t.pre != nil {
		h.OnOpcode = t.opcode
	}
	if t.config.VMTrace {
		h.OnFault = t.fault
	}
	if t.pre != nil {
		// Accounts are discovered before they change. Should a change still
		// come first, its previous value is the original one.
		h.OnBalanceChange = func(a common.Address, prev, next *big.Int, reason tracing.BalanceChangeReason) {
			if account, found := t.discover(a); found {
				account.balance = uint256.MustFromBig(prev)
			}
		}
		h.OnNonceChangeV2 = func(a common.Address, prev, next uint64, reason tracing.NonceChangeReason) {
			if account, found := t.discover(a); found {
				account.nonce = prev
			}
		}
		h.OnCodeChangeV2 = func(a common.Address, prevHash common.Hash, prev []byte, nextHash common.Hash, next []byte, reason tracing.CodeChangeReason) {
			if account, found := t.discover(a); found {
				account.code = common.CopyBytes(prev)
			}
		}
		h.OnStorageChange = func(a common.Address, key, prev, next common.Hash) {
			if account, _ := t.discover(a); account != nil {
				if _, ok := account.storage[key]; !ok {
					account.storage[key] = prev
				}
			}
		}
	}
	return h
}

// txStart discovers the accounts the transaction changes before any frame is
// entered: the sender pays for gas, the fee recipient is credited (paying a
// zero fee still touches it, so touch-clearing may delete it) and EIP-7702
// authorities are delegated.
func (t *Tracer) txStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.state = env.StateDB
	rules := t.chainConfig.Rules(env.BlockNumber, env.Random != nil, env.Time)
	t.table, _ = vm.LookupInstructionSet(rules)
	if t.precompiles == nil {
		t.precompiles = vm.ActivePrecompiledContracts(rules)
	}
	t.discover(from)
	t.discover(env.Coinbase)
	if to := tx.To(); to != nil {
		t.discover(*to)
	}
	for _, auth := range tx.SetCodeAuthorizations() {
		if authority, err := auth.Authority(); err == nil {
			t.discover(authority)
		}
	}
}

// discover records the pre-transaction state of an account at its first
// observation, and reports whether this was it.
func (t *Tracer) discover(address common.Address) (*accountState, bool) {
	if t.pre == nil || t.halted() {
		return nil, false
	}
	if account, ok := t.pre[address]; ok {
		return account, false
	}
	account := &accountState{
		exists:  t.state.Exist(address),
		balance: new(uint256.Int).Set(t.state.GetBalance(address)),
		nonce:   t.state.GetNonce(address),
		code:    common.CopyBytes(t.state.GetCode(address)),
		storage: make(map[common.Hash]common.Hash),
	}
	t.pre[address] = account
	return account, true
}

// Stop ends recording at the next event, and Result reports err.
func (t *Tracer) Stop(err error) {
	t.reason.Store(&err)
}

// halted reports whether recording has ended.
func (t *Tracer) halted() bool {
	return t.err != nil || t.reason.Load() != nil
}

func (t *Tracer) reserve(size int) bool {
	t.bytes += size
	if t.bytes > byteLimit || t.entries > entryLimit {
		t.err = errors.New("trace output limit exceeded")
	}
	return t.err == nil
}

func (t *Tracer) enter(depth int, typ byte, from, to common.Address, input []byte, gas uint64, value *big.Int) {
	if t.halted() {
		return
	}
	t.entries++
	if !t.reserve(0) {
		return
	}
	// A frame is entered before its value transfer and before a created
	// account exists.
	t.discover(from)
	t.discover(to)
	// CREATE and CREATE2 deduct the gas they make available to the child after
	// their own cost, like the CALL family does within its cost.
	switch vm.OpCode(typ) {
	case vm.CREATE, vm.CREATE2:
		if len(t.scopes) > 0 {
			if pending := t.scopes[len(t.scopes)-1].pending; pending != nil {
				pending.op.Cost += gas
			}
		}
	}
	// Parity clients omit nested zero-value precompiles from the call tree.
	// Keep the execution scope: its return bytes still feed the caller's VM delta.
	hidden := false
	switch vm.OpCode(typ) {
	case vm.CALL, vm.CALLCODE, vm.DELEGATECALL, vm.STATICCALL:
		_, precompile := t.precompiles[to]
		hidden = len(t.scopes) > 0 && (value == nil || value.Sign() == 0) && precompile
	}
	path := []uint64{}
	if len(t.scopes) > 0 {
		parent := t.scopes[len(t.scopes)-1].frame
		path = append(append([]uint64{}, parent.TraceAddress...), parent.Subtraces)
		if !hidden {
			parent.Subtraces++
		}
	}
	frame := &Frame{TraceAddress: path, Type: "call"}
	amount := new(big.Int)
	if value != nil {
		amount.Set(value)
	}
	switch vm.OpCode(typ) {
	case vm.CREATE, vm.CREATE2:
		frame.Type = "create"
		if t.config.Trace {
			frame.Action = CreateAction{from, hexutil.Uint64(gas), common.CopyBytes(input), (*hexutil.Big)(amount), strings.ToLower(vm.OpCode(typ).String())}
		}
	case vm.SELFDESTRUCT:
		frame.Type = "suicide"
		if t.config.Trace {
			frame.Action = SuicideAction{from, to, (*hexutil.Big)(amount)}
		}
	default:
		if t.config.Trace {
			frame.Action = CallAction{strings.ToLower(vm.OpCode(typ).String()), from, to, hexutil.Uint64(gas), common.CopyBytes(input), (*hexutil.Big)(amount)}
		}
	}
	if t.config.Trace && !hidden {
		if !t.reserve(len(input)) {
			return
		}
		t.frames = append(t.frames, frame)
	}
	t.scopes = append(t.scopes, &frameScope{frame: frame, target: to, hidden: hidden})
}

func (t *Tracer) exit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if t.halted() || len(t.scopes) == 0 {
		return
	}
	scope := t.scopes[len(t.scopes)-1]
	// A call or create that fails its depth, balance or nonce precheck keeps
	// its frame with the failure, but it starts no execution, so it has no VM
	// trace. From Amsterdam a failed create precheck never enters a frame.
	precheck := errors.Is(err, vm.ErrDepth) || errors.Is(err, vm.ErrInsufficientBalance) || errors.Is(err, vm.ErrNonceUintOverflow)
	// Every entered frame has a VM trace, even when no instruction ran (empty
	// code or a precompile). Selfdestruct does not enter a frame.
	if t.config.VMTrace && scope.vm == nil && scope.frame.Type != "suicide" && !precheck {
		t.openVM(nil)
	}
	t.scopes = t.scopes[:len(t.scopes)-1]
	if len(t.scopes) == 0 {
		t.output = common.CopyBytes(output)
		t.reserve(len(output))
	}
	if !t.config.Trace || scope.hidden {
		return
	}
	frame := scope.frame
	if err != nil && reverted {
		_, precompile := t.precompiles[scope.target]
		frame.Error = errorLabel(err, precompile && frame.Type == "call")
		if frame.Error == "Reverted" {
			frame.Result = CallResult{hexutil.Uint64(gasUsed), common.CopyBytes(output)}
		}
	} else if frame.Type == "create" {
		frame.Result = CreateResult{hexutil.Uint64(gasUsed), scope.target, common.CopyBytes(t.state.GetCode(scope.target))}
	} else if frame.Type == "call" {
		frame.Result = CallResult{hexutil.Uint64(gasUsed), common.CopyBytes(output)}
	}
	t.reserve(len(output))
}

// errorLabel maps an EVM failure to its Parity failure label. Unknown
// errors keep their text, which consumers treat as a generic failure.
func errorLabel(err error, precompile bool) string {
	var (
		invalidOp *vm.ErrInvalidOpCode
		underflow *vm.ErrStackUnderflow
		overflow  *vm.ErrStackOverflow
	)
	switch {
	case errors.Is(err, vm.ErrExecutionReverted):
		return "Reverted"
	// Dynamic gas failures are wrapped as out of gas with the cause as text.
	case errors.Is(err, vm.ErrWriteProtection) || strings.HasSuffix(err.Error(), ": "+vm.ErrWriteProtection.Error()):
		return "Mutable Call In Static Context"
	// As in Parity and EIP-170, a code-deposit failure, including code above
	// the size limit, is out of gas.
	case errors.Is(err, vm.ErrOutOfGas), errors.Is(err, vm.ErrCodeStoreOutOfGas), errors.Is(err, vm.ErrMaxCodeSizeExceeded), errors.Is(err, vm.ErrGasUintOverflow):
		return "Out of gas"
	case errors.As(err, &invalidOp):
		return "Bad instruction"
	case errors.Is(err, vm.ErrInvalidJump):
		return "Bad jump destination"
	case errors.As(err, &underflow):
		return "Stack underflow"
	case errors.As(err, &overflow):
		return "Out of stack"
	case errors.Is(err, vm.ErrReturnDataOutOfBounds):
		return "Out of bounds"
	case errors.Is(err, vm.ErrContractAddressCollision):
		return "Contract address collision"
	// OpenEthereum's label for returned code starting with 0xEF (EIP-3541).
	case errors.Is(err, vm.ErrInvalidCode):
		return "Invalid code"
	case errors.Is(err, vm.ErrNonceUintOverflow):
		return "Nonce overflow"
	case errors.Is(err, vm.ErrInsufficientBalance):
		return "Insufficient balance for transfer"
	case errors.Is(err, vm.ErrDepth):
		return "Max call depth exceeded"
	case precompile:
		return "Built-in failed"
	}
	return err.Error()
}

// Result returns the output of the traced transaction, read from its
// finalized post-state.
func (t *Tracer) Result() (*Result, error) {
	if reason := t.reason.Load(); reason != nil {
		return nil, *reason
	}
	if t.err != nil {
		return nil, t.err
	}
	result := &Result{Output: t.output, Trace: t.frames, VMTrace: t.rootVM}
	if t.pre != nil {
		result.StateDiff = t.stateDiff()
	}
	return result, nil
}

func diffMarker(before, after any, oldExists, newExists, equal bool) any {
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

func (t *Tracer) stateDiff() map[common.Address]*AccountDiff {
	result := make(map[common.Address]*AccountDiff)
	for address, pre := range t.pre {
		oldExists, newExists := pre.exists, t.state.Exist(address)
		if !oldExists && !newExists {
			continue
		}
		oldBalance, newBalance := pre.balance, t.state.GetBalance(address)
		oldNonce, newNonce := pre.nonce, t.state.GetNonce(address)
		oldCode, newCode := pre.code, t.state.GetCode(address)
		// Slots follow the account's existence: zero words on a surviving account
		// are changes, not slot creations or deletions. A deleted account's
		// marker implies all of its storage is wiped, so it lists no slots.
		storage := make(map[common.Hash]any)
		for key, oldValue := range pre.storage {
			if !newExists {
				break
			}
			newValue := t.state.GetState(address, key)
			if oldValue == newValue {
				continue
			}
			storage[key] = diffMarker(oldValue, newValue, oldExists, newExists, false)
		}
		if oldExists == newExists && oldBalance.Eq(newBalance) && oldNonce == newNonce && bytes.Equal(oldCode, newCode) && len(storage) == 0 {
			continue
		}
		result[address] = &AccountDiff{
			Balance: diffMarker((*hexutil.Big)(oldBalance.ToBig()), (*hexutil.Big)(newBalance.ToBig()), oldExists, newExists, oldBalance.Eq(newBalance)),
			Nonce:   diffMarker(hexutil.Uint64(oldNonce), hexutil.Uint64(newNonce), oldExists, newExists, oldNonce == newNonce),
			Code:    diffMarker(hexutil.Bytes(oldCode), hexutil.Bytes(newCode), oldExists, newExists, bytes.Equal(oldCode, newCode)),
			Storage: storage,
		}
	}
	return result
}

func (t *Tracer) opcode(pc uint64, op byte, gas, cost uint64, context tracing.OpContext, returnData []byte, depth int, err error) {
	// A selfdestruct credits its beneficiary before its frame is entered.
	if t.pre != nil && vm.OpCode(op) == vm.SELFDESTRUCT {
		if stack := context.StackData(); len(stack) > 0 {
			t.discover(common.Address(stack[len(stack)-1].Bytes20()))
		}
	}
	if t.halted() || len(t.scopes) == 0 || !t.config.VMTrace {
		return
	}
	scope := t.scopes[len(t.scopes)-1]
	t.finishOp(scope, context, gas)
	if scope.vm == nil && !t.openVM(context.ContractCode()) {
		return
	}
	// Geth executes an implicit STOP past the end of the code, which is not an
	// instruction of the code. Operations rejected before execution are omitted.
	if pc >= uint64(len(context.ContractCode())) || rejectedOp(err) {
		return
	}
	t.entries++
	if !t.reserve(0) {
		return
	}
	step := &VMOp{PC: pc, Cost: cost, Op: vm.OpCode(op).String()}
	scope.vm.Ops = append(scope.vm.Ops, step)
	scope.last = step
	if err != nil || cost > gas {
		return
	}
	// Terminal instructions have no pushed values or writes. Capture them while
	// the interpreter still owns its stack; OnExit runs after those pools are freed.
	switch vm.OpCode(op) {
	case vm.STOP, vm.RETURN, vm.REVERT, vm.SELFDESTRUCT:
		step.Ex = &VMEffects{Used: gas - cost, Push: []*hexutil.Big{}}
		return
	}
	stack := context.StackData()
	pending := &pendingOp{op: step, stackBase: len(stack) - t.stackInputs(vm.OpCode(op), context.ContractCode(), pc)}
	word := func(i int) uint64 {
		if len(stack) <= i {
			return 0
		}
		return stack[len(stack)-1-i].Uint64()
	}
	// mem is the post-operation contents of the range the operands designate.
	switch vm.OpCode(op) {
	case vm.MSTORE, vm.MLOAD:
		pending.memoryOffset, pending.memorySize = word(0), 32
	case vm.MSTORE8:
		pending.memoryOffset, pending.memorySize = word(0), 1
	case vm.CALLDATACOPY, vm.CODECOPY, vm.RETURNDATACOPY, vm.MCOPY:
		pending.memoryOffset, pending.memorySize = word(0), word(2)
	case vm.EXTCODECOPY:
		pending.memoryOffset, pending.memorySize = word(1), word(3)
	case vm.CALL, vm.CALLCODE:
		pending.memoryOffset, pending.memorySize = word(5), word(6)
	case vm.DELEGATECALL, vm.STATICCALL:
		pending.memoryOffset, pending.memorySize = word(4), word(5)
	case vm.SSTORE:
		if len(stack) >= 2 {
			pending.store = &StorageWrite{(*hexutil.Big)(stack[len(stack)-1].ToBig()), (*hexutil.Big)(stack[len(stack)-2].ToBig())}
		}
	}
	scope.pending = pending
}

// stackInputs returns the number of stack words an operation reads. The
// EIP-8024 instructions read to the depth of their immediate, beyond the
// minimum of the instruction table.
func (t *Tracer) stackInputs(op vm.OpCode, code []byte, pc uint64) int {
	var immediate byte
	if pc+1 < uint64(len(code)) {
		immediate = code[pc+1]
	}
	switch op {
	case vm.DUPN:
		return vm.DecodeSingle(immediate)
	case vm.SWAPN:
		return vm.DecodeSingle(immediate) + 1
	case vm.EXCHANGE:
		n, m := vm.DecodePair(immediate)
		return max(n, m) + 1
	}
	inputs, _ := t.table[op].Stack()
	return inputs
}

// openVM starts the innermost scope's VM trace and attaches it to the parent's
// pending call or create instruction.
func (t *Tracer) openVM(code []byte) bool {
	if !t.reserve(len(code)) {
		return false
	}
	scope := t.scopes[len(t.scopes)-1]
	scope.vm = &VMTrace{Code: common.CopyBytes(code), Ops: []*VMOp{}}
	if len(t.scopes) == 1 {
		t.rootVM = scope.vm
	} else {
		parent := t.scopes[len(t.scopes)-2]
		if parent.pending != nil {
			parent.pending.op.Sub = scope.vm
		}
	}
	return true
}

func (t *Tracer) finishOp(scope *frameScope, context tracing.OpContext, gas uint64) {
	pending := scope.pending
	if pending == nil {
		return
	}
	scope.pending = nil
	stack := context.StackData()
	if pending.stackBase < 0 || pending.stackBase > len(stack) {
		t.err = errors.New("invalid trace stack transition")
		return
	}
	effects := &VMEffects{Used: gas, Push: []*hexutil.Big{}, Store: pending.store}
	for i := pending.stackBase; i < len(stack); i++ {
		effects.Push = append(effects.Push, (*hexutil.Big)(stack[i].ToBig()))
	}
	offset, size := pending.memoryOffset, pending.memorySize
	memory := context.MemoryData()
	if size > 0 && offset <= uint64(len(memory)) && size <= uint64(len(memory))-offset {
		if !t.reserve(int(size)) {
			return
		}
		effects.Mem = &MemoryWrite{offset, common.CopyBytes(memory[offset : offset+size])}
	}
	pending.op.Ex = effects
	t.reserve(len(effects.Push) * 32)
}

func (t *Tracer) fault(pc uint64, op byte, gas, cost uint64, context tracing.OpContext, depth int, err error) {
	if t.halted() || len(t.scopes) == 0 {
		return
	}
	if vm.OpCode(op) == vm.REVERT && (errors.Is(err, vm.ErrExecutionReverted) || strings.Contains(err.Error(), "execution reverted")) {
		return
	}
	scope := t.scopes[len(t.scopes)-1]
	if scope.last != nil && scope.last.PC == pc {
		if rejectedOp(err) {
			// Undefined opcodes pass the gas phase and fail when executed.
			scope.vm.Ops = scope.vm.Ops[:len(scope.vm.Ops)-1]
			scope.last = nil
		} else {
			// The operation began executing: keep its cost, without effects.
			scope.last.Ex = nil
		}
	}
	scope.pending = nil
}

// rejectedOp reports failures that reject an operation before it
// executes: undefined opcodes and stack bounds.
func rejectedOp(err error) bool {
	var (
		invalidOp *vm.ErrInvalidOpCode
		underflow *vm.ErrStackUnderflow
		overflow  *vm.ErrStackOverflow
	)
	return errors.As(err, &invalidOp) || errors.As(err, &underflow) || errors.As(err, &overflow)
}
