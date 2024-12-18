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

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/params"
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
	callType CallType

	// args:start
	caller       ContractRef
	addr         common.Address
	input        []byte
	gasRemaining uint64
	value        *uint256.Int
	// args:end
}

// A CallType refers to a *CALL* [OpCode] / respective method on [EVM].
type CallType OpCode

const (
	Call         = CallType(CALL)
	CallCode     = CallType(CALLCODE)
	DelegateCall = CallType(DELEGATECALL)
	StaticCall   = CallType(STATICCALL)
)

func (t CallType) isValid() bool {
	switch t {
	case Call, CallCode, DelegateCall, StaticCall:
		return true
	default:
		return false
	}
}

// String returns a human-readable representation of the CallType.
func (t CallType) String() string {
	if t.isValid() {
		return t.OpCode().String()
	}
	return fmt.Sprintf("Unknown %T(%d)", t, t)
}

// OpCode returns t's equivalent OpCode.
func (t CallType) OpCode() OpCode {
	if t.isValid() {
		return OpCode(t)
	}
	return INVALID
}

// run runs the [PrecompiledContract], differentiating between stateful and
// regular types, updating `args.gasRemaining` in the stateful case.
func (args *evmCallArgs) run(p PrecompiledContract, input []byte) (ret []byte, err error) {
	switch p := p.(type) {
	default:
		return p.Run(input)
	case statefulPrecompile:
		env := args.env()
		ret, err := p(env, input)
		args.gasRemaining = env.Gas()
		return ret, err
	}
}

// PrecompiledStatefulContract is the stateful equivalent of a
// [PrecompiledContract].
//
// Instead of receiving and returning gas arguments, stateful precompiles use
// the respective methods on [PrecompileEnvironment]. If a call to UseGas()
// returns false, a stateful precompile SHOULD return [ErrOutOfGas].
type PrecompiledStatefulContract func(env PrecompileEnvironment, input []byte) (ret []byte, err error)

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
	// StateDB will be non-nil i.f.f !ReadOnly().
	StateDB() StateDB
	// ReadOnlyState will always be non-nil.
	ReadOnlyState() libevm.StateReader

	IncomingCallType() CallType
	Addresses() *libevm.AddressContext
	ReadOnly() bool
	// Equivalent to respective methods on [Contract].
	Gas() uint64
	UseGas(uint64) (hasEnoughGas bool)
	Value() *uint256.Int

	BlockHeader() (types.Header, error)
	BlockNumber() *big.Int
	BlockTime() uint64

	// Call is equivalent to [EVM.Call] except that the `caller` argument is
	// removed and automatically determined according to the type of call that
	// invoked the precompile.
	Call(addr common.Address, input []byte, gas uint64, value *uint256.Int, _ ...CallOption) (ret []byte, _ error)
}

func (args *evmCallArgs) env() *environment {
	var (
		self  common.Address
		value = args.value
	)
	switch args.callType {
	case StaticCall:
		value = new(uint256.Int)
		fallthrough
	case Call:
		self = args.addr

	case DelegateCall:
		value = nil
		fallthrough
	case CallCode:
		self = args.caller.Address()
	}

	// This is equivalent to the `contract` variables created by evm.*Call*()
	// methods, for non precompiles, to pass to [EVMInterpreter.Run].
	contract := NewContract(args.caller, AccountRef(self), value, args.gasRemaining)
	if args.callType == DelegateCall {
		contract = contract.AsDelegate()
	}

	return &environment{
		evm:      args.evm,
		self:     contract,
		callType: args.callType,
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
