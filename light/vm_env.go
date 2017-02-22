// Copyright 2016 The go-ethereum Authors
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

package light

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/net/context"
)

// VMState is a wrapper for the light state that holds the actual context and
// passes it to any state operation that requires it.
type VMState struct {
	ctx       context.Context
	state     *LightState
	snapshots []*LightState
	err       error
}

func NewVMState(ctx context.Context, state *LightState) *VMState {
	return &VMState{ctx: ctx, state: state}
}

func (s *VMState) Error() error {
	return s.err
}

func (s *VMState) AddLog(log *types.Log) {}

func (s *VMState) AddPreimage(hash common.Hash, preimage []byte) {}

// errHandler handles and stores any state error that happens during execution.
func (s *VMState) errHandler(err error) {
	if err != nil && s.err == nil {
		s.err = err
	}
}

func (self *VMState) Snapshot() int {
	self.snapshots = append(self.snapshots, self.state.Copy())
	return len(self.snapshots) - 1
}

func (self *VMState) RevertToSnapshot(idx int) {
	self.state.Set(self.snapshots[idx])
	self.snapshots = self.snapshots[:idx]
}

// CreateAccount creates creates a new account object and takes ownership.
func (s *VMState) CreateAccount(addr common.Address) {
	_, err := s.state.CreateStateObject(s.ctx, addr)
	s.errHandler(err)
}

// AddBalance adds the given amount to the balance of the specified account
func (s *VMState) AddBalance(addr common.Address, amount *big.Int) {
	err := s.state.AddBalance(s.ctx, addr, amount)
	s.errHandler(err)
}

// SubBalance adds the given amount to the balance of the specified account
func (s *VMState) SubBalance(addr common.Address, amount *big.Int) {
	err := s.state.SubBalance(s.ctx, addr, amount)
	s.errHandler(err)
}

// ForEachStorage calls a callback function for every key/value pair found
// in the local storage cache. Note that unlike core/state.StateObject,
// light.StateObject only returns cached values and doesn't download the
// entire storage tree.
func (s *VMState) ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) {
	err := s.state.ForEachStorage(s.ctx, addr, cb)
	s.errHandler(err)
}

// GetBalance retrieves the balance from the given address or 0 if the account does
// not exist
func (s *VMState) GetBalance(addr common.Address) *big.Int {
	res, err := s.state.GetBalance(s.ctx, addr)
	s.errHandler(err)
	return res
}

// GetNonce returns the nonce at the given address or 0 if the account does
// not exist
func (s *VMState) GetNonce(addr common.Address) uint64 {
	res, err := s.state.GetNonce(s.ctx, addr)
	s.errHandler(err)
	return res
}

// SetNonce sets the nonce of the specified account
func (s *VMState) SetNonce(addr common.Address, nonce uint64) {
	err := s.state.SetNonce(s.ctx, addr, nonce)
	s.errHandler(err)
}

// GetCode returns the contract code at the given address or nil if the account
// does not exist
func (s *VMState) GetCode(addr common.Address) []byte {
	res, err := s.state.GetCode(s.ctx, addr)
	s.errHandler(err)
	return res
}

// GetCodeHash returns the contract code hash at the given address
func (s *VMState) GetCodeHash(addr common.Address) common.Hash {
	res, err := s.state.GetCode(s.ctx, addr)
	s.errHandler(err)
	return crypto.Keccak256Hash(res)
}

// GetCodeSize returns the contract code size at the given address
func (s *VMState) GetCodeSize(addr common.Address) int {
	res, err := s.state.GetCode(s.ctx, addr)
	s.errHandler(err)
	return len(res)
}

// SetCode sets the contract code at the specified account
func (s *VMState) SetCode(addr common.Address, code []byte) {
	err := s.state.SetCode(s.ctx, addr, code)
	s.errHandler(err)
}

// AddRefund adds an amount to the refund value collected during a vm execution
func (s *VMState) AddRefund(gas *big.Int) {
	s.state.AddRefund(gas)
}

// GetRefund returns the refund value collected during a vm execution
func (s *VMState) GetRefund() *big.Int {
	return s.state.GetRefund()
}

// GetState returns the contract storage value at storage address b from the
// contract address a or common.Hash{} if the account does not exist
func (s *VMState) GetState(a common.Address, b common.Hash) common.Hash {
	res, err := s.state.GetState(s.ctx, a, b)
	s.errHandler(err)
	return res
}

// SetState sets the storage value at storage address key of the account addr
func (s *VMState) SetState(addr common.Address, key common.Hash, value common.Hash) {
	err := s.state.SetState(s.ctx, addr, key, value)
	s.errHandler(err)
}

// Suicide marks an account to be removed and clears its balance
func (s *VMState) Suicide(addr common.Address) bool {
	res, err := s.state.Suicide(s.ctx, addr)
	s.errHandler(err)
	return res
}

// Exist returns true if an account exists at the given address
func (s *VMState) Exist(addr common.Address) bool {
	res, err := s.state.HasAccount(s.ctx, addr)
	s.errHandler(err)
	return res
}

// Empty returns true if the account at the given address is considered empty
func (s *VMState) Empty(addr common.Address) bool {
	so, err := s.state.GetStateObject(s.ctx, addr)
	s.errHandler(err)
	return so == nil || so.empty()
}

// HasSuicided returns true if the given account has been marked for deletion
// or false if the account does not exist
func (s *VMState) HasSuicided(addr common.Address) bool {
	res, err := s.state.HasSuicided(s.ctx, addr)
	s.errHandler(err)
	return res
}
