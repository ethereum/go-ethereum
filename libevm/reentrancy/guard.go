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

// Package reentrancy provides a reentrancy guard for stateful precompiles that
// make outgoing calls to other contracts.
//
// Reentrancy occurs when the contract (C) called by a precompile (P) makes a
// further call back into P, which may result in theft of funds (see DAO hack).
// A reentrancy guard detects these recursive calls and reverts.
package reentrancy

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/libevm"
)

var slotPreimagePrefix = []byte("libevm-reentrancy-guard-")

// Guard returns [vm.ErrExecutionReverted] i.f.f. it has already been called
// with the same `key`, by the same contract, in the same transaction. It
// otherwise returns nil. The `key` MAY be nil.
//
// Contract equality is defined as the [libevm.AddressContext] "self" address
// being the same under EVM semantics.
func Guard(env vm.PrecompileEnvironment, key []byte) error {
	self := env.Addresses().EVMSemantic.Self
	slot := crypto.Keccak256Hash(slotPreimagePrefix, key)

	sdb := env.StateDB()
	if sdb.GetTransientState(self, slot) != (common.Hash{}) {
		return vm.ErrExecutionReverted
	}
	sdb.SetTransientState(self, slot, common.Hash{1})
	return nil
}

// Keep the `libevm` import to allow the linked comment on [Guard]. The package
// is imported by `vm` anyway so this is a noop but it improves developer
// experience.
var _ = (*libevm.AddressContext)(nil)
