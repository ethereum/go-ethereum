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
//
// Accounts and slots loaded through the view are cached as usual. This is
// safe, as the StateDB records an access before consulting its caches, so a
// subsequent read by the EVM is recorded regardless.
type tracerView struct {
	db      tracing.StateDB
	suspend func() func()
}

// NewTracerView returns a read-only view of the given state for tracers, whose
// reads are not recorded in the block-level access list.
//
// The suspend function is invoked around every read that touches the access
// list. It must disable the recording of accesses until the function it returns
// is invoked.
func NewTracerView(db tracing.StateDB, suspend func() func()) tracing.StateDB {
	return &tracerView{db: db, suspend: suspend}
}

func (v *tracerView) GetBalance(addr common.Address) *uint256.Int {
	defer v.suspend()()
	return v.db.GetBalance(addr)
}

func (v *tracerView) GetNonce(addr common.Address) uint64 {
	defer v.suspend()()
	return v.db.GetNonce(addr)
}

func (v *tracerView) GetCode(addr common.Address) []byte {
	defer v.suspend()()
	return v.db.GetCode(addr)
}

func (v *tracerView) GetCodeHash(addr common.Address) common.Hash {
	defer v.suspend()()
	return v.db.GetCodeHash(addr)
}

func (v *tracerView) GetState(addr common.Address, key common.Hash) common.Hash {
	defer v.suspend()()
	return v.db.GetState(addr, key)
}

func (v *tracerView) Exist(addr common.Address) bool {
	defer v.suspend()()
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
