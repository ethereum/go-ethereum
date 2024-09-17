package vm_test

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/libevm"
	"github.com/ethereum/go-ethereum/libevm/ethtest"
	"github.com/ethereum/go-ethereum/libevm/hookstest"
	"github.com/ethereum/go-ethereum/params"
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
	Caller, Self            common.Address
	StateValue              common.Hash
	ReadOnly                bool
	BlockNumber, Difficulty *big.Int
	BlockTime               uint64
	Input                   []byte
}

func (o statefulPrecompileOutput) String() string {
	var lines []string
	out := reflect.ValueOf(o)
	for i, n := 0, out.NumField(); i < n; i++ {
		name := out.Type().Field(i).Name
		fld := out.Field(i).Interface()

		verb := "%v"
		if _, ok := fld.([]byte); ok {
			verb = "%#x"
		}
		lines = append(lines, fmt.Sprintf("%s: "+verb, name, fld))
	}
	return strings.Join(lines, "\n")
}

func TestNewStatefulPrecompile(t *testing.T) {
	rng := ethtest.NewPseudoRand(314159)
	precompile := rng.Address()
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

		addrs := env.Addresses()
		out := &statefulPrecompileOutput{
			Caller:      addrs.Caller,
			Self:        addrs.Self,
			StateValue:  env.ReadOnlyState().GetState(precompile, slot),
			ReadOnly:    env.ReadOnly(),
			BlockNumber: env.BlockNumber(),
			BlockTime:   env.BlockTime(),
			Difficulty:  hdr.Difficulty,
			Input:       input,
		}
		return []byte(out.String()), suppliedGas - gasCost, nil
	}
	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(run),
		},
	}
	hooks.Register(t)

	header := &types.Header{
		Number:     rng.BigUint64(),
		Time:       rng.Uint64(),
		Difficulty: rng.BigUint64(),
	}
	caller := rng.Address()
	input := rng.Bytes(8)
	value := rng.Hash()

	state, evm := ethtest.NewZeroEVM(t, ethtest.WithBlockContext(
		core.NewEVMBlockContext(header, nil, rng.AddressPtr()),
	))
	state.SetState(precompile, slot, value)

	tests := []struct {
		name string
		call func() ([]byte, uint64, error)
		// Note that this only covers evm.readWrite being set to forceReadOnly,
		// via StaticCall(). See TestInheritReadOnly for alternate case.
		wantReadOnly bool
	}{
		{
			name: "EVM.Call()",
			call: func() ([]byte, uint64, error) {
				return evm.Call(vm.AccountRef(caller), precompile, input, gasLimit, uint256.NewInt(0))
			},
			wantReadOnly: false,
		},
		{
			name: "EVM.CallCode()",
			call: func() ([]byte, uint64, error) {
				return evm.CallCode(vm.AccountRef(caller), precompile, input, gasLimit, uint256.NewInt(0))
			},
			wantReadOnly: false,
		},
		{
			name: "EVM.DelegateCall()",
			call: func() ([]byte, uint64, error) {
				return evm.DelegateCall(vm.AccountRef(caller), precompile, input, gasLimit)
			},
			wantReadOnly: false,
		},
		{
			name: "EVM.StaticCall()",
			call: func() ([]byte, uint64, error) {
				return evm.StaticCall(vm.AccountRef(caller), precompile, input, gasLimit)
			},
			wantReadOnly: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantReturnData := statefulPrecompileOutput{
				Caller:      caller,
				Self:        precompile,
				StateValue:  value,
				ReadOnly:    tt.wantReadOnly,
				BlockNumber: header.Number,
				BlockTime:   header.Time,
				Difficulty:  header.Difficulty,
				Input:       input,
			}.String()
			wantGasLeft := gasLimit - gasCost

			gotReturnData, gotGasLeft, err := tt.call()
			require.NoError(t, err)
			assert.Equal(t, wantReturnData, string(gotReturnData))
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

	var precompile common.Address
	const precompileAddr = 255
	precompile[common.AddressLength-1] = precompileAddr

	const (
		ifReadOnly = iota + 1 // see contract bytecode for rationale
		ifNotReadOnly
	)
	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(
				func(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
					if env.ReadOnly() {
						return []byte{ifReadOnly}, suppliedGas, nil
					}
					return []byte{ifNotReadOnly}, suppliedGas, nil
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

	// See CALL signature: https://www.evm.codes/#f1?fork=cancun
	const p0 = vm.PUSH0
	contract := []vm.OpCode{
		vm.PUSH1, 1, // retSize (bytes)
		p0, // retOffset
		p0, // argSize
		p0, // argOffset
		p0, // value
		vm.PUSH1, precompileAddr,
		p0, // gas
		vm.CALL,
		// It's ok to ignore the return status. If the CALL failed then we'll
		// return []byte{0} next, and both non-failure return buffers are
		// non-zero because of the `iota + 1`.
		vm.PUSH1, 1, // size (byte)
		p0,
		vm.RETURN,
	}

	state, evm := ethtest.NewZeroEVM(t)
	rng := ethtest.NewPseudoRand(42)
	contractAddr := rng.Address()
	state.CreateAccount(contractAddr)
	state.SetCode(contractAddr, contractCode(contract))

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

// contractCode converts a slice of op codes into a byte buffer for storage as
// contract code.
func contractCode(ops []vm.OpCode) []byte {
	ret := make([]byte, len(ops))
	for i, o := range ops {
		ret[i] = byte(o)
	}
	return ret
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
