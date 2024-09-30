// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package vm

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/params"
)

// evmCallArgs mirrors the parameters of the [EVM] methods Call(), CallCode(),
// DelegateCall() and StaticCall(). Its fields are identical to those of the
// parameters, prepended with the receiver name and call type. As
// {Delegate,Static}Call don't accept a value, they MAY set the respective field
// to nil as it will be ignored.
//
// Instantiation can be achieved by merely copying the parameter names, in
// order, which is trivially achieved with AST manipulation:
//
//	func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) ... {
//	    ...
//	    args := &evmCallArgs{evm, staticCall, caller, addr, input, gas, nil /*value*/}
type evmCallArgs struct {
	evm      *EVM
	callType callType

	// args:start
	caller ContractRef
	addr   common.Address
	input  []byte
	gas    uint64
	value  *uint256.Int
	// args:end
}

type callType uint8

const (
	call callType = iota + 1
	callCode
	delegateCall
	staticCall
)

// run runs the [PrecompiledContract], differentiating between stateful and
// regular types.
func (args *evmCallArgs) run(p PrecompiledContract, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error) {
	if p, ok := p.(statefulPrecompile); ok {
		return p(args.env(), input, suppliedGas)
	}
	// Gas consumption for regular precompiles was already handled by the native
	// RunPrecompiledContract(), which called this method.
	ret, err = p.Run(input)
	return ret, suppliedGas, err
}

// PrecompiledStatefulContract is the stateful equivalent of a
// [PrecompiledContract].
type PrecompiledStatefulContract func(env PrecompileEnvironment, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error)

// NewStatefulPrecompile constructs a new PrecompiledContract that can be used
// via an [EVM] instance but MUST NOT be called directly; a direct call to Run()
// reserves the right to panic. See other requirements defined in the comments
// on [PrecompiledContract].
func NewStatefulPrecompile(run PrecompiledStatefulContract) PrecompiledContract {
	return statefulPrecompile(run)
}

// statefulPrecompile implements the [PrecompiledContract] interface to allow a
// [PrecompiledStatefulContract] to be carried with regular geth plumbing. The
// methods are defined on this unexported type instead of directly on
// [PrecompiledStatefulContract] to hide implementation details.
type statefulPrecompile PrecompiledStatefulContract

// RequiredGas always returns zero as this gas is consumed by native geth code
// before the contract is run.
func (statefulPrecompile) RequiredGas([]byte) uint64 { return 0 }

func (p statefulPrecompile) Run([]byte) ([]byte, error) {
	// https://google.github.io/styleguide/go/best-practices.html#when-to-panic
	// This would indicate an API misuse and would occur in tests, not in
	// production.
	panic(fmt.Sprintf("BUG: call to %T.Run(); MUST call %T itself", p, p))
}

// A PrecompileEnvironment provides (a) information about the context in which a
// precompiled contract is being run; and (b) a means of calling other
// contracts.
type PrecompileEnvironment interface {
	ChainConfig() *params.ChainConfig
	Rules() params.Rules
	ReadOnly() bool
	// StateDB will be non-nil i.f.f !ReadOnly().
	StateDB() StateDB
	// ReadOnlyState will always be non-nil.
	ReadOnlyState() libevm.StateReader
	Addresses() *libevm.AddressContext

	BlockHeader() (types.Header, error)
	BlockNumber() *big.Int
	BlockTime() uint64

	// Call is equivalent to [EVM.Call] except that the `caller` argument is
	// removed and automatically determined according to the type of call that
	// invoked the precompile.
	Call(addr common.Address, input []byte, gas uint64, value *uint256.Int, _ ...CallOption) (ret []byte, gasRemaining uint64, _ error)
}

func (args *evmCallArgs) env() *environment {
	var (
		self  common.Address
		value = args.value
	)
	switch args.callType {
	case staticCall:
		value = new(uint256.Int)
		fallthrough
	case call:
		self = args.addr

	case delegateCall:
		value = nil
		fallthrough
	case callCode:
		self = args.caller.Address()
	}

	// This is equivalent to the `contract` variables created by evm.*Call*()
	// methods, for non precompiles, to pass to [EVMInterpreter.Run].
	contract := NewContract(args.caller, AccountRef(self), value, args.gas)
	if args.callType == delegateCall {
		contract = contract.AsDelegate()
	}

	return &environment{
		evm:           args.evm,
		self:          contract,
		forceReadOnly: args.readOnly(),
	}
}

func (args *evmCallArgs) readOnly() bool {
	// A switch statement provides clearer code coverage for difficult-to-test
	// cases.
	switch {
	case args.callType == staticCall:
		// evm.interpreter.readOnly is only set to true via a call to
		// EVMInterpreter.Run() so, if a precompile is called directly with
		// StaticCall(), then readOnly might not be set yet.
		return true
	case args.evm.interpreter.readOnly:
		return true
	default:
		return false
	}
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
