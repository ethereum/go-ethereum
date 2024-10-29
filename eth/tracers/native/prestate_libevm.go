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

package native

import (
	"math/big"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
)

// CaptureEnter implements the [vm.EVMLogger] hook for entering a new scope (via
// CALL*, CREATE or SELFDESTRUCT).
func (t *prestateTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	// Although [prestateTracer.lookupStorage] expects
	// [prestateTracer.lookupAccount] to have been called, the invariant is
	// maintained by [prestateTracer.CaptureState] when it encounters an OpCode
	// corresponding to scope entry. This, however, doesn't work when using a
	// call method exposed by [vm.PrecompileEnvironment], and is restored by a
	// call to this CaptureEnter implementation. Note that lookupAccount(x) is
	// idempotent.
	t.lookupAccount(to)
}
