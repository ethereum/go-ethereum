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
	"github.com/ava-labs/libevm/common/math"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/options"
	"github.com/ava-labs/libevm/params"
)

var _ PrecompileEnvironment = (*environment)(nil)

type environment struct {
	evm      *EVM
	self     *Contract
	callType CallType
}

func (e *environment) Gas() uint64            { return e.self.Gas }
func (e *environment) UseGas(gas uint64) bool { return e.self.UseGas(gas) }
func (e *environment) Value() *uint256.Int    { return new(uint256.Int).Set(e.self.Value()) }

func (e *environment) ChainConfig() *params.ChainConfig  { return e.evm.chainConfig }
func (e *environment) Rules() params.Rules               { return e.evm.chainRules }
func (e *environment) ReadOnlyState() libevm.StateReader { return e.evm.StateDB }
func (e *environment) IncomingCallType() CallType        { return e.callType }
func (e *environment) BlockNumber() *big.Int             { return new(big.Int).Set(e.evm.Context.BlockNumber) }
func (e *environment) BlockTime() uint64                 { return e.evm.Context.Time }

func (e *environment) refundGas(add uint64) error {
	gas, overflow := math.SafeAdd(e.self.Gas, add)
	if overflow {
		return ErrGasUintOverflow
	}
	e.self.Gas = gas
	return nil
}

func (e *environment) ReadOnly() bool {
	// A switch statement provides clearer code coverage for difficult-to-test
	// cases.
	switch {
	case e.callType == StaticCall:
		// evm.interpreter.readOnly is only set to true via a call to
		// EVMInterpreter.Run() so, if a precompile is called directly with
		// StaticCall(), then readOnly might not be set yet.
		return true
	case e.evm.interpreter.readOnly:
		return true
	default:
		return false
	}
}

func (e *environment) Addresses() *libevm.AddressContext {
	return &libevm.AddressContext{
		Origin: e.evm.Origin,
		Caller: e.self.CallerAddress,
		Self:   e.self.Address(),
	}
}

func (e *environment) StateDB() StateDB {
	if e.ReadOnly() {
		return nil
	}
	return e.evm.StateDB
}

func (e *environment) BlockHeader() (types.Header, error) {
	hdr := e.evm.Context.Header
	if hdr == nil {
		// Although [core.NewEVMBlockContext] sets the field and is in the
		// typical hot path (e.g. miner), there are other ways to create a
		// [vm.BlockContext] (e.g. directly in tests) that may result in no
		// available header.
		return types.Header{}, fmt.Errorf("nil %T in current %T", hdr, e.evm.Context)
	}
	return *hdr, nil
}

func (e *environment) Call(addr common.Address, input []byte, gas uint64, value *uint256.Int, opts ...CallOption) ([]byte, error) {
	return e.callContract(Call, addr, input, gas, value, opts...)
}

func (e *environment) callContract(typ CallType, addr common.Address, input []byte, gas uint64, value *uint256.Int, opts ...CallOption) (retData []byte, retErr error) {
	// Depth and read-only setting are handled by [EVMInterpreter.Run], which
	// isn't used for precompiles, so we need to do it ourselves to maintain the
	// expected invariants.
	in := e.evm.interpreter

	in.evm.depth++
	defer func() { in.evm.depth-- }()

	if e.ReadOnly() && !in.readOnly { // i.e. the precompile was StaticCall()ed
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	var caller ContractRef = e.self
	if options.As[callConfig](opts...).unsafeCallerAddressProxying {
		// Note that, in addition to being unsafe, this breaks an EVM
		// assumption that the caller ContractRef is always a *Contract.
		caller = AccountRef(e.self.CallerAddress)
		if e.callType == DelegateCall {
			// self was created with AsDelegate(), which means that
			// CallerAddress was inherited.
			caller = AccountRef(e.self.Address())
		}
	}

	if in.readOnly && value != nil && !value.IsZero() {
		return nil, ErrWriteProtection
	}
	if !e.UseGas(gas) {
		return nil, ErrOutOfGas
	}

	if t := e.evm.Config.Tracer; t != nil {
		var bigVal *big.Int
		if value != nil {
			bigVal = value.ToBig()
		}
		t.CaptureEnter(typ.OpCode(), caller.Address(), addr, input, gas, bigVal)

		startGas := gas
		defer func() {
			t.CaptureEnd(retData, startGas-e.Gas(), retErr)
		}()
	}

	switch typ {
	case Call:
		ret, returnGas, callErr := e.evm.Call(caller, addr, input, gas, value)
		if err := e.refundGas(returnGas); err != nil {
			return nil, err
		}
		return ret, callErr
	case CallCode, DelegateCall, StaticCall:
		// TODO(arr4n): these cases should be very similar to CALL, hence the
		// early abstraction, to signal to future maintainers. If implementing
		// them, there's likely no need to honour the
		// [callOptUNSAFECallerAddressProxy] because it's purely for backwards
		// compatibility.
		fallthrough
	default:
		return nil, fmt.Errorf("unimplemented precompile call type %v", typ)
	}
}
