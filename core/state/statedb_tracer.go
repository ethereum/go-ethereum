// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
)

// tracerView is the read-only view of the state handed to tracers.
//
// Reads are served by the state the EVM executes on, but with access list
// recording suspended. Since Amsterdam (EIP-7928) every account and slot read
// is part of the block-level access list, so a tracer reading state that
// execution itself never touched would otherwise extend the access list, and
// a valid block would fail validation.
type tracerView struct {
	db    tracing.StateDB // state the reads are served by
	inner *StateDB        // underlying state recording the access list, if any
}

// NewTracerView returns a read-only view of the given state for tracers, whose
// reads are not recorded in the block-level access list.
//
// The view wraps the state the EVM executes on, rather than the underlying
// StateDB, so that reads are served exactly as they are for the EVM.
func NewTracerView(db tracing.StateDB) tracing.StateDB {
	v := &tracerView{db: db}

	// Every state implementation able to record an access list must be listed
	// here. Others have no access list to protect.
	switch s := db.(type) {
	case *StateDB:
		v.inner = s
	case *hookedStateDB:
		v.inner = s.inner
	}
	return v
}

// suspendAccessList disables access list recording until the returned function
// is invoked. It must only wrap reads.
//
// Accounts and slots loaded in the meantime are cached as usual. This is safe,
// as the StateDB records an access before consulting its caches, so a
// subsequent read by the EVM is recorded regardless.
//
// The list is resolved on every call rather than once, since Prepare installs
// a fresh one for every transaction and system call.
func (v *tracerView) suspendAccessList() func() {
	if v.inner == nil {
		return func() {}
	}
	list := v.inner.stateAccessList
	v.inner.stateAccessList = nil
	return func() { v.inner.stateAccessList = list }
}

func (v *tracerView) GetBalance(addr common.Address) *uint256.Int {
	defer v.suspendAccessList()()
	return v.db.GetBalance(addr)
}

func (v *tracerView) GetNonce(addr common.Address) uint64 {
	defer v.suspendAccessList()()
	return v.db.GetNonce(addr)
}

func (v *tracerView) GetCode(addr common.Address) []byte {
	defer v.suspendAccessList()()
	return v.db.GetCode(addr)
}

func (v *tracerView) GetCodeHash(addr common.Address) common.Hash {
	defer v.suspendAccessList()()
	return v.db.GetCodeHash(addr)
}

func (v *tracerView) GetState(addr common.Address, key common.Hash) common.Hash {
	defer v.suspendAccessList()()
	return v.db.GetState(addr, key)
}

func (v *tracerView) Exist(addr common.Address) bool {
	defer v.suspendAccessList()()
	return v.db.Exist(addr)
}

// GetTransientState and GetRefund don't touch the access list, so they are
// forwarded as is.

func (v *tracerView) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return v.db.GetTransientState(addr, key)
}

func (v *tracerView) GetRefund() uint64 {
	return v.db.GetRefund()
}
