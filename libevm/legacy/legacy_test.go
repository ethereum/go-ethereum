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

package legacy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/core/vm"
)

// stubPrecompileEnvironment implements [vm.PrecompileEnvironment] for testing.
type stubPrecompileEnvironment struct {
	vm.PrecompileEnvironment
	gas uint64
}

func (s *stubPrecompileEnvironment) Gas() uint64 {
	return s.gas
}

func (s *stubPrecompileEnvironment) UseGas(gas uint64) (hasEnoughGas bool) {
	if s.gas < gas {
		return false
	}
	s.gas -= gas
	return true
}

func TestPrecompiledStatefulContract_Upgrade(t *testing.T) {
	t.Parallel()

	errTest := errors.New("test error")

	tests := map[string]struct {
		suppliedGas   uint64
		precompileRet []byte
		remainingGas  uint64
		precompileErr error
		wantErr       error
		wantGas       uint64
	}{
		"call_error": {
			suppliedGas:   10,
			precompileRet: []byte{2},
			remainingGas:  6,
			precompileErr: errTest,
			wantErr:       errTest,
			wantGas:       6,
		},
		"remaining_gas_exceeds_supplied_gas": {
			suppliedGas:   10,
			precompileRet: []byte{2},
			remainingGas:  11,
			wantErr:       errRemainingGasExceedsSuppliedGas,
			wantGas:       10,
		},
		"zero_remaining_gas": {
			suppliedGas:   10,
			precompileRet: []byte{2},
			remainingGas:  0,
			wantGas:       0,
		},
		"used_one_gas": {
			suppliedGas:   10,
			precompileRet: []byte{2},
			remainingGas:  9,
			wantGas:       9,
		},
	}

	for name, test := range tests {
		testCase := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := PrecompiledStatefulContract(func(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error) {
				return testCase.precompileRet, testCase.remainingGas, testCase.precompileErr
			})

			upgraded := c.Upgrade()

			env := &stubPrecompileEnvironment{
				gas: testCase.suppliedGas,
			}
			input := []byte("unused")

			ret, err := upgraded(env, input)
			require.ErrorIs(t, err, testCase.wantErr)
			assert.Equal(t, testCase.precompileRet, ret, "bytes returned by upgraded contract")
			assert.Equalf(t, testCase.wantGas, env.gas, "remaining gas in %T", env)
		})
	}
}
