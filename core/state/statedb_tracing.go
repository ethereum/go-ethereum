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

type tracingStateDB struct {
	inner *StateDB
}

// NewTracingStateDB returns a read-only view for a StateDB or hookedStateDB.
// Reads through the view are not recorded in the block access list or code
// witness. The second return value is false for other StateDB implementations.
func NewTracingStateDB(db any) (tracing.StateDB, bool) {
	switch db := db.(type) {
	case *StateDB:
		return &tracingStateDB{inner: db}, true
	case *hookedStateDB:
		return &tracingStateDB{inner: db.inner}, true
	default:
		return nil, false
	}
}

func (s *tracingStateDB) GetBalance(addr common.Address) *uint256.Int {
	if obj := s.inner.getStateObjectNoAccess(addr); obj != nil {
		return obj.Balance()
	}
	return common.U2560
}

func (s *tracingStateDB) GetNonce(addr common.Address) uint64 {
	if obj := s.inner.getStateObjectNoAccess(addr); obj != nil {
		return obj.Nonce()
	}
	return 0
}

func (s *tracingStateDB) GetCode(addr common.Address) []byte {
	if obj := s.inner.getStateObjectNoAccess(addr); obj != nil {
		return obj.Code()
	}
	return nil
}

func (s *tracingStateDB) GetCodeHash(addr common.Address) common.Hash {
	if obj := s.inner.getStateObjectNoAccess(addr); obj != nil {
		return common.BytesToHash(obj.CodeHash())
	}
	return common.Hash{}
}

func (s *tracingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	if obj := s.inner.getStateObjectNoAccess(addr); obj != nil {
		value, dirty := obj.dirtyStorage[key]
		if dirty {
			return value
		}
		return obj.getCommittedState(key, false)
	}
	return common.Hash{}
}

func (s *tracingStateDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return s.inner.GetTransientState(addr, key)
}

func (s *tracingStateDB) Exist(addr common.Address) bool {
	return s.inner.getStateObjectNoAccess(addr) != nil
}

func (s *tracingStateDB) GetRefund() uint64 {
	return s.inner.GetRefund()
}
