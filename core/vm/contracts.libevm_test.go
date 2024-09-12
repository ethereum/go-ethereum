package vm_test

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/ethereum/go-ethereum/common"
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

func TestNewStatefulPrecompile(t *testing.T) {
	rng := ethtest.NewPseudoRand(314159)
	precompile := rng.Address()
	slot := rng.Hash()

	const gasLimit = 1e6
	gasCost := rng.Uint64n(gasLimit)

	makeOutput := func(caller, self common.Address, input []byte, stateVal common.Hash) []byte {
		return []byte(fmt.Sprintf(
			"Caller: %v Precompile: %v State: %v Input: %#x",
			caller, self, stateVal, input,
		))
	}
	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			precompile: vm.NewStatefulPrecompile(
				func(state vm.StateDB, _ *params.Rules, caller, self common.Address, input []byte) ([]byte, error) {
					return makeOutput(caller, self, input, state.GetState(precompile, slot)), nil
				},
				func(b []byte) uint64 {
					return gasCost
				},
			),
		},
	}
	hooks.Register(t)

	caller := rng.Address()
	input := rng.Bytes(8)
	value := rng.Hash()

	state, evm := ethtest.NewZeroEVM(t)
	state.SetState(precompile, slot, value)
	wantReturnData := makeOutput(caller, precompile, input, value)
	wantGasLeft := gasLimit - gasCost

	gotReturnData, gotGasLeft, err := evm.Call(vm.AccountRef(caller), precompile, input, gasLimit, uint256.NewInt(0))
	require.NoError(t, err)
	assert.Equal(t, wantReturnData, gotReturnData)
	assert.Equal(t, wantGasLeft, gotGasLeft)
}

func TestCanCreateContract(t *testing.T) {
	rng := ethtest.NewPseudoRand(142857)
	account := rng.Address()
	slot := rng.Hash()

	makeErr := func(cc *libevm.AddressContext, stateVal common.Hash) error {
		return fmt.Errorf("Origin: %v Caller: %v Contract: %v State: %v", cc.Origin, cc.Caller, cc.Self, stateVal)
	}
	hooks := &hookstest.Stub{
		CanCreateContractFn: func(cc *libevm.AddressContext, s libevm.StateReader) error {
			return makeErr(cc, s.GetState(account, slot))
		},
	}
	hooks.Register(t)

	origin := rng.Address()
	caller := rng.Address()
	value := rng.Hash()
	code := rng.Bytes(8)
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
				return evm.Create(vm.AccountRef(caller), code, 1e6, uint256.NewInt(0))
			},
			wantErr: makeErr(&libevm.AddressContext{Origin: origin, Caller: caller, Self: create}, value),
		},
		{
			name: "Create2",
			create: func(evm *vm.EVM) ([]byte, common.Address, uint64, error) {
				return evm.Create2(vm.AccountRef(caller), code, 1e6, uint256.NewInt(0), new(uint256.Int).SetBytes(salt[:]))
			},
			wantErr: makeErr(&libevm.AddressContext{Origin: origin, Caller: caller, Self: create2}, value),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, evm := ethtest.NewZeroEVM(t)
			state.SetState(account, slot, value)
			evm.TxContext.Origin = origin

			_, _, _, err := tt.create(evm)
			require.EqualError(t, err, tt.wantErr.Error())
		})
	}
}
