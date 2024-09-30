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

var _ PrecompileEnvironment = (*environment)(nil)

type environment struct {
	evm           *EVM
	self          *Contract
	forceReadOnly bool
}

func (e *environment) ChainConfig() *params.ChainConfig  { return e.evm.chainConfig }
func (e *environment) Rules() params.Rules               { return e.evm.chainRules }
func (e *environment) ReadOnly() bool                    { return e.forceReadOnly || e.evm.interpreter.readOnly }
func (e *environment) ReadOnlyState() libevm.StateReader { return e.evm.StateDB }
func (e *environment) BlockNumber() *big.Int             { return new(big.Int).Set(e.evm.Context.BlockNumber) }
func (e *environment) BlockTime() uint64                 { return e.evm.Context.Time }

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

func (e *environment) Call(addr common.Address, input []byte, gas uint64, value *uint256.Int, opts ...CallOption) ([]byte, uint64, error) {
	return e.callContract(call, addr, input, gas, value, opts...)
}

func (e *environment) callContract(typ callType, addr common.Address, input []byte, gas uint64, value *uint256.Int, opts ...CallOption) ([]byte, uint64, error) {
	// Depth and read-only setting are handled by [EVMInterpreter.Run], which
	// isn't used for precompiles, so we need to do it ourselves to maintain the
	// expected invariants.
	in := e.evm.interpreter

	in.evm.depth++
	defer func() { in.evm.depth-- }()

	if e.forceReadOnly && !in.readOnly { // i.e. the precompile was StaticCall()ed
		in.readOnly = true
		defer func() { in.readOnly = false }()
	}

	var caller ContractRef = e.self
	for _, o := range opts {
		switch o := o.(type) {
		case callOptUNSAFECallerAddressProxy:
			// Note that, in addition to being unsafe, this breaks an EVM
			// assumption that the caller ContractRef is always a *Contract.
			caller = AccountRef(e.self.CallerAddress)
		case nil:
		default:
			return nil, gas, fmt.Errorf("unsupported option %T", o)
		}
	}

	switch typ {
	case call:
		if in.readOnly && !value.IsZero() {
			return nil, gas, ErrWriteProtection
		}
		return e.evm.Call(caller, addr, input, gas, value)
	case callCode, delegateCall, staticCall:
		// TODO(arr4n): these cases should be very similar to CALL, hence the
		// early abstraction, to signal to future maintainers. If implementing
		// them, there's likely no need to honour the
		// [callOptUNSAFECallerAddressProxy] because it's purely for backwards
		// compatibility.
		fallthrough
	default:
		return nil, gas, fmt.Errorf("unimplemented precompile call type %v", typ)
	}
}
