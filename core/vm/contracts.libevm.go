package vm

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// evmCallArgs mirrors the parameters of the [EVM] methods Call(), CallCode(),
// DelegateCall() and StaticCall(). Its fields are identical to those of the
// parameters, prepended with the receiver name. As {Delegate,Static}Call don't
// accept a value, they MUST set the respective field to nil.
//
// Instantiation can be achieved by merely copying the parameter names, in
// order, which is trivially achieved with AST manipulation:
//
//	func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *uint256.Int) ... {
//		...
//		args := &evmCallArgs{evm, caller, addr, input, gas, value}
type evmCallArgs struct {
	evm    *EVM
	caller ContractRef
	addr   common.Address
	input  []byte
	gas    uint64
	value  *uint256.Int
}

// run runs the [PrecompiledContract], differentiating between stateful and
// regular types.
func (args *evmCallArgs) run(p PrecompiledContract, input []byte) (ret []byte, err error) {
	if p, ok := p.(statefulPrecompile); ok {
		return p.run(args.evm.StateDB, &args.evm.chainRules, args.caller.Address(), args.addr, input)
	}
	return p.Run(input)
}

// PrecompiledStatefulRun is the stateful equivalent of the Run() method of a
// [PrecompiledContract].
type PrecompiledStatefulRun func(_ StateDB, _ *params.Rules, caller, self common.Address, input []byte) ([]byte, error)

// NewStatefulPrecompile constructs a new PrecompiledContract that can be used
// via an [EVM] instance but MUST NOT be called directly; a direct call to Run()
// reserves the right to panic. See other requirements defined in the comments
// on [PrecompiledContract].
func NewStatefulPrecompile(run PrecompiledStatefulRun, requiredGas func([]byte) uint64) PrecompiledContract {
	return statefulPrecompile{
		gas: requiredGas,
		run: run,
	}
}

type statefulPrecompile struct {
	gas func([]byte) uint64
	run PrecompiledStatefulRun
}

func (p statefulPrecompile) RequiredGas(input []byte) uint64 {
	return p.gas(input)
}

func (p statefulPrecompile) Run([]byte) ([]byte, error) {
	// https://google.github.io/styleguide/go/best-practices.html#when-to-panic
	// This would indicate an API misuse and would occur in tests, not in
	// production.
	panic(fmt.Sprintf("BUG: call to %T.Run(); MUST call %T", p, p.run))
}

var (
	// These lock in the assumptions made when implementing [evmCallArgs]. If
	// these break then the struct fields SHOULD be changed to match these
	// signatures.
	_ = [](func(ContractRef, common.Address, []byte, uint64, *uint256.Int) ([]byte, uint64, error)){
		(*EVM)(nil).Call,
		(*EVM)(nil).CallCode,
	}
	_ = [](func(ContractRef, common.Address, []byte, uint64) ([]byte, uint64, error)){
		(*EVM)(nil).DelegateCall,
		(*EVM)(nil).StaticCall,
	}
)
