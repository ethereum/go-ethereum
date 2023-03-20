package blocknative

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
)

func init() {
	tracers.DefaultDirectory.Register("txnOpCodeTracer", NewTxnOpCodeTracer, false)
}

// txnOpCodeTracer is a go implementation of the Tracer interface which
// only returns a restricted trace of a transaction consisting of transaction
// op codes and relevant gas data.
// This is intended for Blocknative usage.
type txnOpCodeTracer struct {
	env       *vm.EVM     // EVM context for execution of transaction to occur within
	trace     Trace       // Accumulated execution data the caller is interested in
	callStack []CallFrame // Data structure for op codes making up our trace
	interrupt uint32      // Atomic flag to signal execution interruption
	reason    error       // Textual reason for the interruption (not always specific for us)
	opts      TracerOpts
	beginTime time.Time // Time object for start of trace for stats
}

// NewTxnOpCodeTracer returns a new txnOpCodeTracer tracer with the given
// options applied.
func NewTxnOpCodeTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {

	// First callframe contains tx context info
	// and is populated on start and end.
	t := &txnOpCodeTracer{callStack: make([]CallFrame, 1)}

	// Decode raw json opts into our struct.
	if cfg != nil {
		if err := json.Unmarshal(cfg, &t.opts); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// GetResult returns an empty json object.
func (t *txnOpCodeTracer) GetResult() (json.RawMessage, error) {

	// This block used to trip on subtraces being discovered, for this tracer we do not need this,
	// however we would like to keep this here in a possible future where we do care about such cases.

	// if len(t.callStack) != 1 {
	// 	return nil, errors.New("incorrect number of top-level calls")
	// }

	// Only want the top level trace, all other indexes hold subtraces to which we do not particularly need
	t.trace.CallFrame = t.callStack[0]

	res, err := json.Marshal(t.trace)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *txnOpCodeTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env

	// If we want NetBalChanges, start by tracking the top level addresses
	if t.opts.NetBalChanges {
		t.lookupAccount(from)
		t.lookupAccount(to)
		t.lookupAccount(env.Context.Coinbase) // Todo alex: Do we care about where the mining reward theoretically goes to?

		// The recipient balance includes the value transferred.
		// todo alex: This would mean in the before for the from address,
		toBal := new(big.Int).Sub(t.trace.NetBalChanges.Pre[to].Balance, value)
		t.trace.NetBalChanges.Pre[to].Balance = toBal

		// Todo alex: Does capture start occur before the initial gas negation? I think I need to re-add here!
	}

	// Blocks only contain `Random` post-merge, but we still have pre-merge tests.
	random := ""
	if env.Context.Random != nil {
		random = bytesToHex(env.Context.Random.Bytes())
	}

	// Populate the block context from the vm environment.
	t.trace.BlockContext.Number = env.Context.BlockNumber.Uint64()
	t.trace.BlockContext.BaseFee = env.Context.BaseFee.Uint64()
	t.trace.BlockContext.Time = env.Context.Time
	t.trace.BlockContext.Coinbase = addrToHex(env.Context.Coinbase)
	t.trace.BlockContext.GasLimit = env.Context.GasLimit
	t.trace.BlockContext.Random = random

	// Start tracing timer
	t.beginTime = time.Now()

	// This is the initial call
	t.callStack[0] = CallFrame{
		Type:  "CALL",
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
	}
	if create {
		// TODO: Here we can note creation of contracts for potential future tracing
		t.callStack[0].Type = "CREATE"
	}
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *txnOpCodeTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	// Collect final gasUsed
	t.callStack[0].GasUsed = uintToHex(gasUsed)

	// Add total time duration for this trace request
	t.trace.Time = fmt.Sprintf("%v", time.Since(t.beginTime))

	// If the user wants the logs, grab them from the state
	if t.opts.Logs {
		for _, stateLog := range t.env.StateDB.Logs() {
			t.trace.Logs = append(t.trace.Logs, CallLog{
				Address: stateLog.Address,
				Data:    bytesToHex(stateLog.Data),
				Topics:  stateLog.Topics,
			})
		}
	}

	// This is the final output of a call
	if err != nil {
		t.callStack[0].Error = err.Error()
		if err.Error() == "execution reverted" && len(output) > 0 {
			t.callStack[0].Output = bytesToHex(output)

			// This revert reason is found via the standard introduced in v0.8.4
			// It uses a ABI with the method Error(string)
			// This is the top level call, internal txns may fail while top level succeeds still
			revertReason, _ := abi.UnpackRevert(output)
			t.callStack[0].ErrorReason = revertReason
		}
	} else {
		// TODO: This output is for the originally called contract, we can use the ABI to decode this for useful information
		// ie: there are custom error types in ABIs since 0.8.4 which will turn up here
		t.callStack[0].Output = bytesToHex(output)
	}
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *txnOpCodeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	defer func() {
		if r := recover(); r != nil {
			t.callStack[depth].Error = "internal failure"
			log.Warn("Panic during trace. Recovered.", "err", r)
		}
	}()

	// If we want NetBalChanges, track storage altering opcodes. Here we add the addresses to our map to check later.
	// Todo Alex: Can I remove this in space of a better way of tracking touched addresses and not opcode comparison?
	if t.opts.NetBalChanges {
		stack := scope.Stack
		stackData := stack.Data()
		stackLen := len(stackData)
		caller := scope.Contract.Address()
		switch {
		case stackLen >= 1 && (op == vm.SLOAD || op == vm.SSTORE):
			slot := common.Hash(stackData[stackLen-1].Bytes32())
			t.lookupStorage(caller, slot)
		case stackLen >= 1 && (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE):
			addr := common.Address(stackData[stackLen-1].Bytes20())
			t.lookupAccount(addr)
		case stackLen >= 5 && (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE):
			addr := common.Address(stackData[stackLen-2].Bytes20())
			t.lookupAccount(addr)
		}
	}
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *txnOpCodeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
	// The err here is generated by geth, not by contract error logging
}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *txnOpCodeTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Skip if tracing was interrupted
	if atomic.LoadUint32(&t.interrupt) > 0 {
		t.env.Cancel()
		return
	}

	// Apart from the starting call detected by CaptureStart, here we track every new transaction opcode
	call := CallFrame{
		Type:  typ.String(),
		From:  addrToHex(from),
		To:    addrToHex(to),
		Input: bytesToHex(input),
		Gas:   uintToHex(gas),
		Value: bigToHex(value),
	}
	t.callStack = append(t.callStack, call)

	// Todo: Can add a decode request here from OWL in future
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *txnOpCodeTracer) CaptureExit(output []byte, gasUsed uint64, err error) {

	size := len(t.callStack)
	if size <= 1 {
		return
	}
	// pop call
	call := t.callStack[size-1]
	t.callStack = t.callStack[:size-1]
	size -= 1

	call.GasUsed = uintToHex(gasUsed)
	if err == nil {
		call.Output = bytesToHex(output)
	} else {
		call.Error = err.Error()
		if err.Error() == "execution reverted" && len(output) > 0 {
			call.Output = bytesToHex(output)
			revertReason, _ := abi.UnpackRevert(output)
			call.ErrorReason = revertReason
		}

		if call.Type == "CREATE" || call.Type == "CREATE2" {
			call.To = ""
		}
	}
	t.callStack[size-1].Calls = append(t.callStack[size-1].Calls, call)
}

func (*txnOpCodeTracer) CaptureTxStart(gasLimit uint64) {
}

// SetStateRoot implements core.stateRootSetter and stores the given root in the trace's BlockContext.
func (t *txnOpCodeTracer) SetStateRoot(root common.Hash) {
	t.trace.BlockContext.StateRoot = bytesToHex(root.Bytes())
}

func (t *txnOpCodeTracer) CaptureTxEnd(restGas uint64) {
	// If we want NetBalChanges,
	if t.opts.NetBalChanges {
		for addr, state := range t.trace.NetBalChanges.Pre {
			// Todo alex: If we end up tracking creations / deletions, we would prune some here

			// Keep track if we end up finding an altered address
			modified := false

			// Keep track of potential Eth balance changes, and storage changes
			// Later in a final post-processing step we will decode these for user known formats
			postAccount := &account{Storage: make(map[common.Hash]common.Hash)}
			newBalance := t.env.StateDB.GetBalance(addr)
			newCode := t.env.StateDB.GetCode(addr)

			if newBalance.Cmp(t.trace.NetBalChanges.Pre[addr].Balance) != 0 {
				modified = true
				postAccount.Balance = newBalance
			}
			if !bytes.Equal(newCode, t.trace.NetBalChanges.Pre[addr].Code) {
				modified = true
				postAccount.Code = newCode
			}

			for key, val := range state.Storage {
				// don't include the empty slot
				if val == (common.Hash{}) {
					delete(t.trace.NetBalChanges.Pre[addr].Storage, key)
				}

				newVal := t.env.StateDB.GetState(addr, key)
				if val == newVal {
					// Omit unchanged slots
					delete(t.trace.NetBalChanges.Pre[addr].Storage, key)
				} else {
					modified = true
					if newVal != (common.Hash{}) {
						postAccount.Storage[key] = newVal
					}
				}
			}

			if modified {
				t.trace.NetBalChanges.Post[addr] = postAccount
			} else {
				// if state is not modified, then no need to include into the pre state
				delete(t.trace.NetBalChanges.Pre, addr)
			}
		}
	}
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *txnOpCodeTracer) Stop(err error) {
	t.reason = err
	atomic.StoreUint32(&t.interrupt, 1)
}

// LookupAccount fetches details of an account and adds it to the prestate
// if it doesn't exist there.
func (t *txnOpCodeTracer) lookupAccount(addr common.Address) {
	if _, ok := t.trace.NetBalChanges.Pre[addr]; ok {
		return
	}

	t.trace.NetBalChanges.Pre[addr] = &account{
		Balance: t.env.StateDB.GetBalance(addr),
		Code:    t.env.StateDB.GetCode(addr),
		Storage: make(map[common.Hash]common.Hash),
	}
}

// LookupStorage fetches the requested storage slot and adds
// it to the prestate of the given contract. It assumes `lookupAccount`
// has been performed on the contract before.
func (t *txnOpCodeTracer) lookupStorage(addr common.Address, key common.Hash) {
	if _, ok := t.trace.NetBalChanges.Pre[addr].Storage[key]; ok {
		return
	}
	t.trace.NetBalChanges.Pre[addr].Storage[key] = t.env.StateDB.GetState(addr, key)
}
