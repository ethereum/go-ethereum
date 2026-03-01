//go:build evmone && cgo

package vm

import (
	"errors"
	"math"
	"runtime/cgo"
)

// EVMC status code constants (must match evmc_status_code enum in evmc.h).
const (
	evmcSuccess              int32 = 0
	evmcFailure              int32 = 1
	evmcRevert               int32 = 2
	evmcOutOfGas             int32 = 3
	evmcInvalidInstruction   int32 = 4
	evmcUndefinedInstruction int32 = 5
	evmcStackOverflowStatus  int32 = 6
	evmcStackUnderflowStatus int32 = 7
	evmcBadJumpDestination   int32 = 8
	evmcInvalidMemoryAccess  int32 = 9
	evmcCallDepthExceeded    int32 = 10
	evmcStaticModeViolation  int32 = 11
)

// Run loops and evaluates the contract's code with the given input data and returns
// the return byte-slice and an error if one occurred.
//
// When built with the evmone tag, this method executes bytecode via the evmone
// C++ EVM, falling back to the Go interpreter for tracing, Verkle, or ExtraEips.
func (evm *EVM) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	// Fall back to Go interpreter for cases evmone cannot handle:
	// 1. Tracing: evmone doesn't support per-opcode Go callbacks
	// 2. Verkle/EIP-4762: requires per-chunk gas charging not in EVMC
	// 3. ExtraEips: evmone doesn't support arbitrary custom EIPs
	if evm.Config.Tracer != nil || evm.chainRules.IsEIP4762 || len(evm.Config.ExtraEips) > 0 {
		return evm.runGoInterpreter(contract, input, readOnly)
	}

	// Increment the call depth which is restricted to 1024
	evm.depth++
	defer func() { evm.depth-- }()

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This also makes sure that the readOnly flag isn't removed for child calls.
	if readOnly && !evm.readOnly {
		evm.readOnly = true
		defer func() { evm.readOnly = false }()
	}

	// Reset the previous call's return data.
	evm.returnData = nil

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	contract.Input = input

	// Initialize the singleton evmone VM.
	initEvmone()

	// Map chain rules to EVMC revision.
	rev := evmcRevision(evm.chainRules)

	// Create and pin the host context.
	hostCtx := &evmcHostContext{
		evm:      evm,
		contract: contract,
	}
	handle := pinHostContext(hostCtx)
	defer cgo.Handle(handle).Delete()

	// EVMC uses int64 for gas; cap to avoid overflow when Go uses uint64 gas limits.
	gas := int64(contract.Gas)
	if contract.Gas > uint64(math.MaxInt64) {
		gas = math.MaxInt64
	}

	// Execute via evmone.
	result := executeEvmone(
		handle,
		rev,
		gas,
		contract.Address(),
		contract.Caller(),
		input,
		contract.Code,
		contract.Value(),
		int32(evm.depth),
		readOnly,
	)

	ret = result.output

	// Update gas accounting: evmone returns gas_left.
	if result.gasLeft >= 0 {
		contract.Gas = uint64(result.gasLeft)
	} else {
		contract.Gas = 0
	}

	// Propagate gas refund from evmone to StateDB.
	// evmone tracks SSTORE refunds internally; we must relay them to the
	// Go state so that state_transition.calcRefund() can apply them.
	if result.gasRefund > 0 {
		evm.StateDB.AddRefund(uint64(result.gasRefund))
	} else if result.gasRefund < 0 {
		evm.StateDB.SubRefund(uint64(-result.gasRefund))
	}

	// Map EVMC status to go-ethereum errors.
	switch result.statusCode {
	case evmcSuccess:
		return ret, nil
	case evmcRevert:
		return ret, ErrExecutionReverted
	case evmcOutOfGas:
		return nil, ErrOutOfGas
	case evmcInvalidInstruction:
		return nil, &ErrInvalidOpCode{}
	case evmcUndefinedInstruction:
		return nil, &ErrInvalidOpCode{}
	case evmcStackOverflowStatus:
		return nil, &ErrStackOverflow{}
	case evmcStackUnderflowStatus:
		return nil, &ErrStackUnderflow{}
	case evmcBadJumpDestination:
		return nil, ErrInvalidJump
	case evmcStaticModeViolation:
		return nil, ErrWriteProtection
	case evmcCallDepthExceeded:
		return nil, ErrDepth
	case evmcInvalidMemoryAccess:
		return nil, ErrReturnDataOutOfBounds
	default:
		return nil, errors.New("evmone: execution failure")
	}
}
