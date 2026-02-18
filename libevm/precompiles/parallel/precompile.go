// Copyright 2025-2026 the libevm authors.
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

package parallel

import "github.com/ava-labs/libevm/core/vm"

// PrecompileResult is the interface required for a [Handler] to be converted
// into a [vm.PrecompiledStatefulContract].
type PrecompileResult interface {
	// PrecompileOutput's arguments match those of
	// [vm.PrecompiledStatefulContract]. It MUST NOT re-charge the `Gas()`
	// amount returned by the [Handler], but MAY charge for other computation as
	// necessary.
	PrecompileOutput(vm.PrecompileEnvironment, []byte) ([]byte, error)
}

// AddAsPrecompile is equivalent to [AddHandler] except that the returned
// function is a [vm.PrecompiledStatefulContract] instead of a raw result
// fetcher. If the function returned by [AddHandler] returns `false` then the
// precompile returns [vm.ErrExecutionReverted].
func AddAsPrecompile[CD, D any, R PrecompileResult, A any](p *Processor, h Handler[CD, D, R, A]) vm.PrecompiledStatefulContract {
	results := AddHandler(p, h)

	return func(env vm.PrecompileEnvironment, input []byte) ([]byte, error) {
		res, ok := results(env.ReadOnlyState().TxIndex())
		if !ok {
			// TODO(arr4n) add revert data to match a Solidity-style error
			return nil, vm.ErrExecutionReverted
		}
		return res.Result.PrecompileOutput(env, input)
	}
}
