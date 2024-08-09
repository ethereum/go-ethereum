// Copyright 2024 The go-ethereum Authors
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
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/fjl/gencodec -type OverrideAccount -field-override overrideMarshaling -out state_override_json.go

// OverrideAccount specifies the fields of an account that should be overridden
// during the execution of a message call.
//
// Note: The 'state' and 'stateDiff' fields are mutually exclusive and cannot
// be specified simultaneously. If 'state' is set, the message execution will
// only use the data provided in the given state. Otherwise, if 'stateDiff'
// is set, all the differences will be applied first, and then the call message
// will be executed.
type OverrideAccount struct {
	Nonce     *uint64                     `json:"nonce"`
	Code      *[]byte                     `json:"code"`
	Balance   *uint256.Int                `json:"balance"`
	State     map[common.Hash]common.Hash `json:"state"`
	StateDiff map[common.Hash]common.Hash `json:"stateDiff"`
}

// nolint
type overrideMarshaling struct {
	Nonce   *hexutil.Uint64
	Code    *hexutil.Bytes
	Balance *hexutil.U256
}

// copy returns a deep-copied override object.
func (o OverrideAccount) copy() OverrideAccount {
	var obj OverrideAccount
	if o.Nonce != nil {
		nonce := *o.Nonce
		obj.Nonce = &nonce
	}
	if o.Code != nil {
		code := slices.Clone(*o.Code)
		obj.Code = &code
	}
	if o.Balance != nil {
		obj.Balance = new(uint256.Int).Set(o.Balance)
	}
	if o.State != nil {
		obj.State = maps.Clone(o.State)
	}
	if o.StateDiff != nil {
		obj.StateDiff = maps.Clone(o.StateDiff)
	}
	return obj
}

// overrideReader implements the Reader interface, serving as a wrapper around a
// provided state reader, but incorporating with overridden states.
type overrideReader struct {
	reader    Reader
	overrides map[common.Address]OverrideAccount
}

// newOverrideReader creates a reader with customized state overrides.
func newOverrideReader(overrides map[common.Address]OverrideAccount, reader Reader) *overrideReader {
	return &overrideReader{
		reader:    reader,
		overrides: overrides,
	}
}

// Account implementing Reader interface, retrieving the account associated with
// a particular address.
//
// - Returns a nil account if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned account is safe to modify after the call
func (r *overrideReader) Account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.reader.Account(addr)
	if err != nil {
		return nil, err
	}
	// apply the overrides if it's specified
	override, ok := r.overrides[addr]
	if ok {
		if account == nil {
			account = types.NewEmptyStateAccount()
		}
		if override.Nonce != nil {
			account.Nonce = *override.Nonce
		}
		if override.Balance != nil {
			account.Balance = new(uint256.Int).Set(override.Balance)
		}
		if override.Code != nil {
			account.CodeHash = crypto.Keccak256(*override.Code)
		}
		// TODO what about the storage root then, should we compute the
		// storage root of overridden state here?
	}
	return account, nil
}

// Storage implementing Reader interface, retrieving the storage slot associated
// with a particular account address and slot key.
//
// - Returns an empty slot if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned storage slot is safe to modify after the call
func (r *overrideReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	override, ok := r.overrides[addr]
	if ok {
		if override.State != nil {
			return override.State[slot], nil
		}
		if override.StateDiff != nil {
			if val, ok := override.StateDiff[slot]; ok {
				return val, nil
			}
		}
	}
	return r.reader.Storage(addr, slot)
}

// Stats returns the statistics of the reader, specifically detailing the time
// spent on account reading and storage reading.
func (r *overrideReader) Stats() (time.Duration, time.Duration) { return 0, 0 }

// Copy implementing Reader interface, returning a deep-copied state reader.
func (r *overrideReader) Copy() Reader {
	overrides := make(map[common.Address]OverrideAccount)
	for addr, override := range r.overrides {
		overrides[addr] = override.copy()
	}
	return &overrideReader{
		overrides: overrides,
		reader:    r.reader.Copy(),
	}
}

// OverrideDatabase implements the Database interface, serving as a wrapper
// around a standard state database, but incorporating overridden states.
type OverrideDatabase struct {
	Database
	overrides map[common.Address]OverrideAccount
}

// NewOverrideDatabase creates a state database with provided state overrides.
func NewOverrideDatabase(db Database, overrides map[common.Address]OverrideAccount) *OverrideDatabase {
	// Allocate an empty override set just in case the provided one
	// is nil. Don't panic for lazy users.
	if overrides == nil {
		overrides = make(map[common.Address]OverrideAccount)
	}
	return &OverrideDatabase{
		Database:  db,
		overrides: overrides,
	}
}

// Reader returns a state reader associated with the specified state root.
func (db *OverrideDatabase) Reader(root common.Hash) (Reader, error) {
	reader, err := db.Database.Reader(root)
	if err != nil {
		return nil, err
	}
	return newOverrideReader(db.overrides, reader), nil
}

// ContractCode retrieves a particular contract's code.
func (db *OverrideDatabase) ContractCode(addr common.Address, codeHash common.Hash) ([]byte, error) {
	override, ok := db.overrides[addr]
	if ok && override.Code != nil {
		return *override.Code, nil
	}
	return db.Database.ContractCode(addr, codeHash)
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *OverrideDatabase) ContractCodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	override, ok := db.overrides[addr]
	if ok && override.Code != nil {
		return len(*override.Code), nil
	}
	return db.Database.ContractCodeSize(addr, codeHash)
}

// The stateWrapReader wraps a live state database as the state source.
type stateWrapReader struct {
	state *StateDB
}

// Account implementing Reader interface, retrieving the account associated with
// a particular address.
//
// - Returns a nil account if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned account is safe to modify after the call
func (r *stateWrapReader) Account(addr common.Address) (*types.StateAccount, error) {
	obj := r.state.getStateObject(addr)
	if obj == nil {
		return nil, nil
	}
	return obj.data.Copy(), nil
}

// Storage implementing Reader interface, retrieving the storage slot associated
// with a particular account address and slot key.
//
// - Returns an empty slot if it does not exist
// - Returns an error only if an unexpected issue occurs
// - The returned storage slot is safe to modify after the call
func (r *stateWrapReader) Storage(addr common.Address, slot common.Hash) (common.Hash, error) {
	return r.state.GetState(addr, slot), nil
}

// Stats returns the statistics of the reader, specifically detailing the time
// spent on account reading and storage reading.
func (r *stateWrapReader) Stats() (time.Duration, time.Duration) { return 0, 0 }

// Copy implementing Reader interface, returning a deep-copied state reader.
func (r *stateWrapReader) Copy() Reader {
	return &stateWrapReader{state: r.state.Copy()}
}

// stateWrap is an internal struct wrapping a live state instance as the state
// data source. This can be useful in scenarios where we need to create the
// snapshot of the current state and apply some additional overrides on top
// (e.g. state override in RPC call serving).
type stateWrap struct {
	Database
	state *StateDB
}

// Reader returns a state reader associated with the specified state root.
func (db *stateWrap) Reader(root common.Hash) (Reader, error) {
	if root != db.state.originalRoot {
		return nil, errors.New("state root is not matched")
	}
	return &stateWrapReader{state: db.state}, nil
}

// OverrideState applies the state overrides into the provided live state database.
func OverrideState(state *StateDB, overrides map[common.Address]OverrideAccount) (*StateDB, error) {
	db := NewOverrideDatabase(&stateWrap{
		Database: state.db,
		state:    state.Copy(),
	}, overrides)
	return New(state.originalRoot, db)
}
