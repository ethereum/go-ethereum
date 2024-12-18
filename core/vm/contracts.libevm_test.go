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
package vm_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/eth/tracers"
	_ "github.com/ava-labs/libevm/eth/tracers/native"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/hookstest"
	"github.com/ava-labs/libevm/libevm/legacy"
	"github.com/ava-labs/libevm/params"
)

type precompileStub struct {
	requiredGas uint64
	returnData  []byte
}

func (s *precompileStub) RequiredGas([]byte) uint64  { return s.requiredGas }
func (s *precompileStub) Run([]byte) ([]byte, error) { return s.returnData, nil }

func TestPrecompileOverride(t *testing.T) {
	type test struct {
		name        string
		addr        common.Address
		requiredGas uint64
		stubData    []byte
	}

	const gasLimit = uint64(1e7)

	tests := []test{
		{
			name:        "arbitrary values",
			addr:        common.Address{'p', 'r', 'e', 'c', 'o', 'm', 'p', 'i', 'l', 'e'},
			requiredGas: 314159,
			stubData:    []byte("the return data"),
		},
	}

	rng := rand.New(rand.NewSource(42))
	for _, addr := range vm.PrecompiledAddressesCancun {
		tests = append(tests, test{
			name:        fmt.Sprintf("existing precompile %v", addr),
			addr:        addr,
			requiredGas: rng.Uint64n(gasLimit),
			stubData:    addr[:],
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks := &hookstest.Stub{
				PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
					tt.addr: &precompileStub{
						requiredGas: tt.requiredGas,
						returnData:  tt.stubData,
					},
				},
			}
			hooks.Register(t)

			t.Run(fmt.Sprintf("%T.Call([overridden precompile address = %v])", &vm.EVM{}, tt.addr), func(t *testing.T) {
				_, evm := ethtest.NewZeroEVM(t)
				gotData, gotGasLeft, err := evm.Call(vm.AccountRef{}, tt.addr, nil, gasLimit, uint256.NewInt(0))
				require.NoError(t, err)
				assert.Equal(t, tt.stubData, gotData, "contract's return data")
				assert.Equal(t, gasLimit-tt.requiredGas, gotGasLeft, "gas left")
			})
		})
	}
}

type statefulPrecompileOutput struct {
	ChainID                 *big.Int
	Addresses               *libevm.AddressContext
	StateValue              common.Hash
	ValueReceived           *uint256.Int
	ReadOnly                bool
	BlockNumber, Difficulty *big.Int
	BlockTime               uint64
	Input                   []byte
	IncomingCallType        vm.CallType
}

func (o statefulPrecompileOutput) String() string {
	var lines []string
	out := reflect.ValueOf(o)
	for i, n := 0, out.NumField(); i < n; i++ {
		name := out.Type().Field(i).Name
		fld := out.Field(i).Interface()

		verb := "%v"
		switch fld.(type) {
		case []byte:
			verb = "%#x"
		case *libevm.AddressContext:
			verb = "%+v"
		case vm.CallType:
			verb = "%d (%[2]q)"
		}
		lines = append(lines, fmt.Sprintf("%s: "+verb, name, fld))
	}
	return strings.Join(lines, "\n")
}

func (o statefulPrecompileOutput) Bytes() []byte {
	return []byte(o.String())
}

func TestNewStatefulPrecompile(t *testing.T) {
	precompile := common.HexToAddress("60C0DE") // GO CODE
	rng := ethtest.NewPseudoRand(314159)
	slot := rng.Hash()

	const gasLimit = 1e6
	gasCost := rng.Uint64n(gasLimit)

	run := func(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
		if got, want := env.StateDB() != nil, !env.ReadOnly(); got != want {
			return nil, 0, fmt.Errorf("PrecompileEnvironment().StateDB() must be non-nil i.f.f. not read-only; got non-nil? %t; want %t", got, want)
		}
		hdr, err := env.BlockHeader()
		if err != nil {
			return nil, 0, err
		}

		out := &statefulPrecompileOutput{
			ChainID:          env.ChainConfig().ChainID,
			Addresses:        env.Addresses(),
			StateValue:       env.ReadOnlyState().GetState(precompile, slot),
			ValueReceived:    env.Value(),
			ReadOnly:         env.ReadOnly(),
			BlockNumber:      env.BlockNumber(),
			BlockTime:        env.BlockTime(),
			Difficulty:       hdr.Difficulty,
			Input:            input,
			IncomingCallType: env.IncomingCallType(),
		}
		return out.Bytes(), suppliedGas - gasCost, nil
	}
	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(
				// In production, the new function signature should be used, but
				// this just exercises the converter.
				legacy.PrecompiledStatefulContract(run).Upgrade(),
			),
		},
	}
	hooks.Register(t)

	header := &types.Header{
		Number:     rng.BigUint64(),
		Time:       rng.Uint64(),
		Difficulty: rng.BigUint64(),
	}
	input := rng.Bytes(8)
	stateValue := rng.Hash()
	transferValue := rng.Uint256()
	chainID := rng.BigUint64()

	caller := common.HexToAddress("CA11E12") // caller of the precompile
	eoa := common.HexToAddress("E0A")        // caller of the precompile-caller
	callerContract := vm.NewContract(vm.AccountRef(eoa), vm.AccountRef(caller), uint256.NewInt(0), 1e6)

	state, evm := ethtest.NewZeroEVM(
		t,
		ethtest.WithBlockContext(
			core.NewEVMBlockContext(header, nil, rng.AddressPtr()),
		),
		ethtest.WithChainConfig(
			&params.ChainConfig{ChainID: chainID},
		),
	)
	state.SetState(precompile, slot, stateValue)
	state.SetBalance(caller, new(uint256.Int).Not(uint256.NewInt(0)))
	evm.Origin = eoa

	tests := []struct {
		name              string
		call              func() ([]byte, uint64, error)
		wantAddresses     *libevm.AddressContext
		wantTransferValue *uint256.Int
		// Note that this only covers evm.readOnly being true because of the
		// precompile's call. See TestInheritReadOnly for alternate case.
		wantReadOnly bool
		wantCallType vm.CallType
	}{
		{
			name: "EVM.Call()",
			call: func() ([]byte, uint64, error) {
				return evm.Call(callerContract, precompile, input, gasLimit, transferValue)
			},
			wantAddresses: &libevm.AddressContext{
				Origin: eoa,
				Caller: caller,
				Self:   precompile,
			},
			wantReadOnly:      false,
			wantTransferValue: transferValue,
			wantCallType:      vm.Call,
		},
		{
			name: "EVM.CallCode()",
			call: func() ([]byte, uint64, error) {
				return evm.CallCode(callerContract, precompile, input, gasLimit, transferValue)
			},
			wantAddresses: &libevm.AddressContext{
				Origin: eoa,
				Caller: caller,
				Self:   caller,
			},
			wantReadOnly:      false,
			wantTransferValue: transferValue,
			wantCallType:      vm.CallCode,
		},
		{
			name: "EVM.DelegateCall()",
			call: func() ([]byte, uint64, error) {
				return evm.DelegateCall(callerContract, precompile, input, gasLimit)
			},
			wantAddresses: &libevm.AddressContext{
				Origin: eoa,
				Caller: eoa, // inherited from caller
				Self:   caller,
			},
			wantReadOnly:      false,
			wantTransferValue: uint256.NewInt(0),
			wantCallType:      vm.DelegateCall,
		},
		{
			name: "EVM.StaticCall()",
			call: func() ([]byte, uint64, error) {
				return evm.StaticCall(callerContract, precompile, input, gasLimit)
			},
			wantAddresses: &libevm.AddressContext{
				Origin: eoa,
				Caller: caller,
				Self:   precompile,
			},
			wantReadOnly:      true,
			wantTransferValue: uint256.NewInt(0),
			wantCallType:      vm.StaticCall,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantOutput := statefulPrecompileOutput{
				ChainID:          chainID,
				Addresses:        tt.wantAddresses,
				StateValue:       stateValue,
				ValueReceived:    tt.wantTransferValue,
				ReadOnly:         tt.wantReadOnly,
				BlockNumber:      header.Number,
				BlockTime:        header.Time,
				Difficulty:       header.Difficulty,
				Input:            input,
				IncomingCallType: tt.wantCallType,
			}

			wantGasLeft := gasLimit - gasCost

			gotReturnData, gotGasLeft, err := tt.call()
			require.NoError(t, err)
			assert.Equal(t, wantOutput.String(), string(gotReturnData))
			assert.Equal(t, wantGasLeft, gotGasLeft)
		})
	}
}

func TestInheritReadOnly(t *testing.T) {
	// The regular test of stateful precompiles only checks the read-only state
	// when called directly via vm.EVM.*Call*() methods. That approach will not
	// result in a read-only state via inheritance, which occurs when already in
	// a read-only environment there is a non-static call to a precompile.
	//
	// Test strategy:
	//
	// 1. Create a precompile that echoes its read-only status in the return
	//    data. We MUST NOT assert inside the precompile as we need proof that
	//    the precompile was actually called.
	//
	// 2. Create a bytecode contract that calls the precompile with CALL and
	//    propagates the return data. Using CALL (i.e. not STATICCALL) means
	//    that we know for certain that [forceReadOnly] isn't being used and,
	//    instead, the read-only state is being read from
	//    evm.interpreter.readOnly.
	//
	// 3. Assert that the returned input is as expected for the read-only state.

	// (1)

	precompile := common.Address{255}

	const (
		ifReadOnly = iota + 1 // see contract bytecode for rationale
		ifNotReadOnly
	)
	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(
				func(env vm.PrecompileEnvironment, input []byte) ([]byte, error) {
					if env.ReadOnly() {
						return []byte{ifReadOnly}, nil
					}
					return []byte{ifNotReadOnly}, nil
				},
			),
		},
	}
	hookstest.Register(t, params.Extras[*hookstest.Stub, *hookstest.Stub]{
		NewRules: func(_ *params.ChainConfig, r *params.Rules, _ *hookstest.Stub, blockNum *big.Int, isMerge bool, timestamp uint64) *hookstest.Stub {
			r.IsCancun = true // enable PUSH0
			return hooks
		},
	})

	// (2)
	contract := makeReturnProxy(t, precompile, vm.CALL)

	state, evm := ethtest.NewZeroEVM(t)
	rng := ethtest.NewPseudoRand(42)
	contractAddr := rng.Address()
	state.CreateAccount(contractAddr)
	state.SetCode(contractAddr, convertBytes[vm.OpCode, byte](contract...))

	// (3)

	caller := vm.AccountRef(rng.Address())
	tests := []struct {
		name string
		call func() ([]byte, uint64, error)
		want byte
	}{
		{
			name: "EVM.Call()",
			call: func() ([]byte, uint64, error) {
				return evm.Call(caller, contractAddr, []byte{}, 1e6, uint256.NewInt(0))
			},
			want: ifNotReadOnly,
		},
		{
			name: "EVM.StaticCall()",
			call: func() ([]byte, uint64, error) {
				return evm.StaticCall(vm.AccountRef(rng.Address()), contractAddr, []byte{}, 1e6)
			},
			want: ifReadOnly,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := tt.call()
			require.NoError(t, err)
			require.Equalf(t, []byte{tt.want}, got, "want %d if read-only, otherwise %d", ifReadOnly, ifNotReadOnly)
		})
	}
}

// makeReturnProxy returns the bytecode of a contract that will call `dest` with
// the specified call type and propagated the returned value.
//
// The contract does NOT check if the call reverted. In this case, the
// propagated return value will always be an empty slice. Tests using these
// proxies MUST use non-empty slices as test values.
//
// TODO(arr4n): convert this to arr4n/specops for clarity and to make it easier
// to generate a revert check.
func makeReturnProxy(t *testing.T, dest common.Address, call vm.OpCode) []vm.OpCode {
	t.Helper()
	const p0 = vm.PUSH0
	contract := []vm.OpCode{
		vm.CALLDATASIZE, p0, p0, vm.CALLDATACOPY,

		p0,              // retSize
		p0,              // retOffset
		vm.CALLDATASIZE, // argSize
		p0,              // argOffset
	}

	// See CALL signature: https://www.evm.codes/#f1?fork=cancun
	switch call {
	case vm.CALL, vm.CALLCODE:
		contract = append(contract, p0) // value
	case vm.DELEGATECALL, vm.STATICCALL:
	default:
		t.Fatalf("Bad test setup: invalid non-CALL-type opcode %s", call)
	}

	contract = append(contract, vm.PUSH20)
	contract = append(contract, convertBytes[byte, vm.OpCode](dest[:]...)...)

	contract = append(contract,
		p0, // gas
		call,

		// See function comment re ignored reverts.
		vm.RETURNDATASIZE, p0, p0, vm.RETURNDATACOPY,
		vm.RETURNDATASIZE, p0, vm.RETURN,
	)
	return contract
}

func convertBytes[From ~byte, To ~byte](buf ...From) []To {
	out := make([]To, len(buf))
	for i, b := range buf {
		out[i] = To(b)
	}
	return out
}

func TestCanCreateContract(t *testing.T) {
	rng := ethtest.NewPseudoRand(142857)
	account := rng.Address()
	slot := rng.Hash()

	const gasLimit uint64 = 1e6
	gasUsage := rng.Uint64n(gasLimit)

	makeErr := func(cc *libevm.AddressContext, stateVal common.Hash) error {
		return fmt.Errorf("Origin: %v Caller: %v Contract: %v State: %v", cc.Origin, cc.Caller, cc.Self, stateVal)
	}
	hooks := &hookstest.Stub{
		CanCreateContractFn: func(cc *libevm.AddressContext, gas uint64, s libevm.StateReader) (uint64, error) {
			return gas - gasUsage, makeErr(cc, s.GetState(account, slot))
		},
	}
	hooks.Register(t)

	origin := rng.Address()
	caller := rng.Address()
	value := rng.Hash()
	code := []byte{byte(vm.STOP)}
	salt := rng.Hash()

	create := crypto.CreateAddress(caller, 0)
	create2 := crypto.CreateAddress2(caller, salt, crypto.Keccak256(code))

	tests := []struct {
		name    string
		create  func(*vm.EVM) ([]byte, common.Address, uint64, error)
		wantErr error
	}{
		{
			name: "Create",
			create: func(evm *vm.EVM) ([]byte, common.Address, uint64, error) {
				return evm.Create(vm.AccountRef(caller), code, gasLimit, uint256.NewInt(0))
			},
			wantErr: makeErr(&libevm.AddressContext{Origin: origin, Caller: caller, Self: create}, value),
		},
		{
			name: "Create2",
			create: func(evm *vm.EVM) ([]byte, common.Address, uint64, error) {
				return evm.Create2(vm.AccountRef(caller), code, gasLimit, uint256.NewInt(0), new(uint256.Int).SetBytes(salt[:]))
			},
			wantErr: makeErr(&libevm.AddressContext{Origin: origin, Caller: caller, Self: create2}, value),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, evm := ethtest.NewZeroEVM(t)
			state.SetState(account, slot, value)
			evm.TxContext.Origin = origin

			_, _, gasRemaining, err := tt.create(evm)
			require.EqualError(t, err, tt.wantErr.Error())
			// require prints uint64s in hex
			require.Equal(t, int(gasLimit-gasUsage), int(gasRemaining), "gas remaining") //nolint:gosec // G115 won't overflow as <= 1e6
		})
	}
}

func TestActivePrecompilesOverride(t *testing.T) {
	newRules := func() params.Rules {
		return new(params.ChainConfig).Rules(big.NewInt(0), false, 0)
	}
	defaultActive := vm.ActivePrecompiles(newRules())

	rng := ethtest.NewPseudoRand(0xDecafC0ffeeBad)
	precompiles := make([]common.Address, rng.Intn(10)+5)
	for i := range precompiles {
		precompiles[i] = rng.Address()
	}
	hooks := &hookstest.Stub{
		ActivePrecompilesFn: func(active []common.Address) []common.Address {
			assert.Equal(t, defaultActive, active, "ActivePrecompiles() hook receives default addresses")
			return precompiles
		},
	}
	hooks.Register(t)

	require.Equal(t, precompiles, vm.ActivePrecompiles(newRules()), "vm.ActivePrecompiles() returns overridden addresses")
}

func TestPrecompileMakeCall(t *testing.T) {
	// There is one test per *CALL* op code:
	//
	// 1. `eoa` makes a call to a bytecode contract, `caller`;
	// 2. `caller` calls `sut`, the precompile under test, via the test's *CALL* op code;
	// 3. `sut` makes a Call() to `dest`, which reflects env data for testing.
	//
	// This acts as a full integration test of a precompile being invoked before
	// making an "outbound" call.
	eoa := common.HexToAddress("E0A")
	caller := common.HexToAddress("CA11E12")
	sut := common.HexToAddress("7E57ED")
	dest := common.HexToAddress("DE57")

	rng := ethtest.NewPseudoRand(142857)
	precompileCallData := rng.Bytes(8)

	// If the SUT precompile receives this as its calldata then it will use the
	// vm.WithUNSAFECallerAddressProxying() option.
	unsafeCallerProxyOptSentinel := []byte("override-caller sentinel")

	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			sut: vm.NewStatefulPrecompile(func(env vm.PrecompileEnvironment, input []byte) (ret []byte, err error) {
				var opts []vm.CallOption
				if bytes.Equal(input, unsafeCallerProxyOptSentinel) {
					opts = append(opts, vm.WithUNSAFECallerAddressProxying())
				}
				// We are ultimately testing env.Call(), hence why this is the SUT.
				return env.Call(dest, precompileCallData, env.Gas(), uint256.NewInt(0), opts...)
			}),
			dest: vm.NewStatefulPrecompile(func(env vm.PrecompileEnvironment, input []byte) (ret []byte, err error) {
				out := &statefulPrecompileOutput{
					Addresses: env.Addresses(),
					ReadOnly:  env.ReadOnly(),
					Input:     input, // expected to be callData
				}
				return out.Bytes(), nil
			}),
		},
	}
	hookstest.Register(t, params.Extras[*hookstest.Stub, *hookstest.Stub]{
		NewRules: func(_ *params.ChainConfig, r *params.Rules, _ *hookstest.Stub, blockNum *big.Int, isMerge bool, timestamp uint64) *hookstest.Stub {
			r.IsCancun = true // enable PUSH0
			return hooks
		},
	})

	tests := []struct {
		incomingCallType vm.OpCode
		eoaTxCallData    []byte
		// Unlike TestNewStatefulPrecompile, which tests the AddressContext of
		// the precompile itself, these test the AddressContext of a contract
		// called by the precompile.
		want statefulPrecompileOutput
	}{
		{
			incomingCallType: vm.CALL,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: sut,
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.CALL,
			eoaTxCallData:    unsafeCallerProxyOptSentinel,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // overridden by CallOption
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.CALLCODE,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // SUT runs as its own caller because of CALLCODE
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.CALLCODE,
			eoaTxCallData:    unsafeCallerProxyOptSentinel,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // CallOption is a NOOP
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.DELEGATECALL,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // as with CALLCODE
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.DELEGATECALL,
			eoaTxCallData:    unsafeCallerProxyOptSentinel,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // CallOption is a NOOP
					Self:   dest,
				},
				Input: precompileCallData,
			},
		},
		{
			incomingCallType: vm.STATICCALL,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: sut,
					Self:   dest,
				},
				Input: precompileCallData,
				// This demonstrates that even though the precompile makes a
				// (non-static) CALL, the read-only state is inherited. Yes,
				// this is _another_ way to get a read-only state, different to
				// the other tests.
				ReadOnly: true,
			},
		},
		{
			incomingCallType: vm.STATICCALL,
			eoaTxCallData:    unsafeCallerProxyOptSentinel,
			want: statefulPrecompileOutput{
				Addresses: &libevm.AddressContext{
					Origin: eoa,
					Caller: caller, // overridden by CallOption
					Self:   dest,
				},
				Input:    precompileCallData,
				ReadOnly: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.incomingCallType.String(), func(t *testing.T) {
			t.Logf("calldata = %q", tt.eoaTxCallData)
			state, evm := ethtest.NewZeroEVM(t)
			evm.Origin = eoa
			state.CreateAccount(caller)
			proxy := makeReturnProxy(t, sut, tt.incomingCallType)
			state.SetCode(caller, convertBytes[vm.OpCode, byte](proxy...))

			got, _, err := evm.Call(vm.AccountRef(eoa), caller, tt.eoaTxCallData, 1e6, uint256.NewInt(0))
			require.NoError(t, err)
			require.Equal(t, tt.want.String(), string(got))
		})
	}
}

func TestPrecompileCallWithTracer(t *testing.T) {
	// The native pre-state tracer, when logging storage, assumes an invariant
	// that is broken by a precompile calling another contract. This is a test
	// of the fix, ensuring that an SLOADed value is properly handled by the
	// tracer.

	rng := ethtest.NewPseudoRand(42 * 142857)
	precompile := rng.Address()
	contract := rng.Address()

	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(func(env vm.PrecompileEnvironment, input []byte) (ret []byte, err error) {
				return env.Call(contract, nil, env.Gas(), uint256.NewInt(0))
			}),
		},
	}
	hooks.Register(t)

	state, evm := ethtest.NewZeroEVM(t)
	evm.GasPrice = big.NewInt(1)

	state.CreateAccount(contract)
	var zeroHash common.Hash
	value := rng.Hash()
	state.SetState(contract, zeroHash, value)
	state.SetCode(contract, convertBytes[vm.OpCode, byte](vm.PC, vm.SLOAD))

	const tracerName = "prestateTracer"
	tracer, err := tracers.DefaultDirectory.New(tracerName, nil, nil)
	require.NoErrorf(t, err, "tracers.DefaultDirectory.New(%q)", tracerName)
	evm.Config.Tracer = tracer

	_, _, err = evm.Call(vm.AccountRef(rng.Address()), precompile, []byte{}, 1e6, uint256.NewInt(0))
	require.NoError(t, err, "evm.Call([precompile that calls regular contract])")

	gotJSON, err := tracer.GetResult()
	require.NoErrorf(t, err, "%T.GetResult()", tracer)
	var got map[common.Address]struct{ Storage map[common.Hash]common.Hash }
	require.NoErrorf(t, json.Unmarshal(gotJSON, &got), "json.Unmarshal(%T.GetResult(), %T)", tracer, &got)
	require.Equal(t, value, got[contract].Storage[zeroHash], "value loaded with SLOAD")
}

//nolint:testableexamples // Including output would only make the example more complicated and hide the true intent
func ExamplePrecompileEnvironment() {
	// To determine the actual caller of a precompile, as against the effective
	// caller (under EVM rules, as exposed by `Addresses().Caller`):
	actualCaller := func(env vm.PrecompileEnvironment) common.Address {
		if env.IncomingCallType() == vm.DelegateCall {
			// DelegateCall acts as if it were its own caller.
			return env.Addresses().Self
		}
		// CallCode could return either `Self` or `Caller` as it acts as its
		// caller but doesn't inherit the caller's caller as DelegateCall does.
		// Having it handled here is arbitrary from a behavioural perspective
		// and is done only to simplify the code.
		//
		// Call and StaticCall don't affect self/caller semantics in any way.
		return env.Addresses().Caller
	}

	// actualCaller would typically be a top-level function. It's only a
	// variable to include it in this example function.
	_ = actualCaller
}
