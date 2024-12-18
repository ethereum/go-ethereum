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

// Package legacy provides converters between legacy types and their refactored
// equivalents.
package legacy

import "github.com/ava-labs/libevm/core/vm"

// PrecompiledStatefulContract is the legacy signature of
// [vm.PrecompiledStatefulContract], which explicitly accepts and returns gas
// values. Instances SHOULD NOT use the [vm.PrecompileEnvironment]
// gas-management methods as this may result in unexpected behaviour.
type PrecompiledStatefulContract func(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) (ret []byte, remainingGas uint64, err error)

// Upgrade converts the legacy precompile signature into the now-required form.
func (c PrecompiledStatefulContract) Upgrade() vm.PrecompiledStatefulContract {
	return func(env vm.PrecompileEnvironment, input []byte) ([]byte, error) {
		gas := env.Gas()
		ret, remainingGas, err := c(env, input, gas)
		if used := gas - remainingGas; used < gas {
			env.UseGas(used)
		}
		return ret, err
	}
}
