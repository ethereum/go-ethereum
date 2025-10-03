// Copyright 2025 the libevm authors.
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

package reentrancy

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm"
	"github.com/ava-labs/libevm/libevm/ethtest"
	"github.com/ava-labs/libevm/libevm/hookstest"
)

func TestGuardIntegration(t *testing.T) {
	sut := common.HexToAddress("7E57ED")
	eve := common.HexToAddress("BAD")
	eveCalled := false

	zero := func() *uint256.Int {
		return uint256.NewInt(0)
	}

	returnIfGuarded := []byte("guarded")

	hooks := &hookstest.Stub{
		PrecompileOverrides: map[common.Address]libevm.PrecompiledContract{
			eve: vm.NewStatefulPrecompile(func(env vm.PrecompileEnvironment, input []byte) (ret []byte, err error) {
				eveCalled = true
				return env.Call(sut, []byte{}, env.Gas(), zero()) // i.e. reenter
			}),
			sut: vm.NewStatefulPrecompile(func(env vm.PrecompileEnvironment, input []byte) (ret []byte, err error) {
				// The argument is optional and used only to allow more than one
				// guard in a contract, tested in a separate unit test.
				if err := Guard(env, nil); err != nil {
					return returnIfGuarded, err
				}
				if env.Addresses().EVMSemantic.Caller == eve {
					// A real precompile MUST NOT panic under any circumstances.
					// It is done here to avoid a loop should the guard not
					// work.
					panic("reentrancy")
				}
				return env.Call(eve, []byte{}, env.Gas(), zero())
			}),
		},
	}
	hooks.Register(t)

	_, evm := ethtest.NewZeroEVM(t)
	got, _, err := evm.Call(vm.AccountRef{}, sut, []byte{}, 1e6, zero())
	require.True(t, eveCalled, "Malicious contract called")
	// The error is propagated Guard() -> reentered SUT -> Eve -> top-level SUT -> evm.Call()
	// This MUST NOT be [assert.ErrorIs] as such errors are never wrapped in geth.
	assert.Equal(t, err, vm.ErrExecutionReverted, "Precompile reverted")
	assert.Equal(t, returnIfGuarded, got, "Precompile reverted with expected data")
}

type envStub struct {
	self common.Address
	db   *state.StateDB
	vm.PrecompileEnvironment
}

func (s *envStub) Addresses() *libevm.AddressContext {
	return &libevm.AddressContext{
		EVMSemantic: libevm.CallerAndSelf{
			Self: s.self,
		},
	}
}

func (s *envStub) StateDB() vm.StateDB {
	return s.db
}

func TestGuard(t *testing.T) {
	db, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	require.NoError(t, err, "state.New()")
	env := &envStub{db: db}

	addr0 := common.Address{}
	addr1 := common.Address{1}
	key0 := []byte{0}
	key1 := []byte{1}

	// All tests run on the same [envStub] so are dependent on the effects of
	// the one(s) before.
	tests := []struct {
		self common.Address
		key  []byte
		want error
	}{
		{addr0, key0, nil},
		{addr0, key0, vm.ErrExecutionReverted},
		{addr0, key1, nil},
		{addr1, key0, nil},
		{addr1, key1, nil},
		{addr1, key1, vm.ErrExecutionReverted},
		{addr0, key1, vm.ErrExecutionReverted},
	}

	history := make(map[common.Hash]bool) // for better error reporting
	for _, tt := range tests {
		h := crypto.Keccak256Hash(tt.self[:], tt.key)
		already := history[h]
		history[h] = true

		env.self = tt.self
		// Tests are dependent so we don't use assert.Equalf.
		require.Equalf(t, tt.want, Guard(env, tt.key), "Guard([self=%v], %#x) when already called = %t", tt.self, tt.key, already)
	}
}
