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
package libevm_test

import (
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/libevm"
)

// IMPORTANT: if any of these break then the libevm copy MUST be updated.

// These two interfaces MUST be identical.
var (
	// Each assignment demonstrates that the methods of the LHS interface are a
	// (non-strict) subset of the RHS interface's; both being possible
	// proves that they are identical.
	_ vm.PrecompiledContract     = (libevm.PrecompiledContract)(nil)
	_ libevm.PrecompiledContract = (vm.PrecompiledContract)(nil)
)

// StateReader MUST be a subset vm.StateDB.
var _ libevm.StateReader = (vm.StateDB)(nil)
