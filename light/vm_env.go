// Copyright 2015 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/net/context"
)

// VMEnv is the light client version of the vm execution environment.
// Unlike other structures, VMEnv holds a context that is applied by state
// retrieval requests through the entire execution. If any state operation
// returns an error, the execution fails.
type VMEnv struct {
	vm.Environment
	ctx    context.Context
	chainConfig *core.ChainConfig
	evm    *vm.EVM
	state  *VMState
	header *types.Header
	msg    core.Message
	depth  int
	chain  *LightChain
	typ    vm.Type
	// structured logging
	logs []vm.StructLog
	err  error
}

// NewEnv creates a new execution environment based on an ODR capable light state
func NewEnv(ctx context.Context, state *LightState, chainConfig *core.ChainConfig, chain *LightChain, msg core.Message, header *types.Header, cfg vm.Config) *VMEnv {
	env := &VMEnv{
		chainConfig: chainConfig,
		chain:  chain,
		header: header,
		msg:    msg,
		typ:    vm.StdVmTy,
	}
	env.state = &VMState{ctx: ctx, state: state, env: env}

	// if no log collector is present set self as the collector
	if cfg.Logger.Collector == nil {
		cfg.Logger.Collector = env
	}

	env.evm = vm.New(env, cfg)
	return env
}

func (self *VMEnv) RuleSet() vm.RuleSet      { return self.chainConfig }
func (self *VMEnv) Vm() vm.Vm                { return self.evm }
func (self *VMEnv) Origin() common.Address   { f, _ := self.msg.From(); return f }
func (self *VMEnv) BlockNumber() *big.Int    { return self.header.Number }
func (self *VMEnv) Coinbase() common.Address { return self.header.Coinbase }
func (self *VMEnv) Time() *big.Int           { return self.header.Time }
func (self *VMEnv) Difficulty() *big.Int     { return self.header.Difficulty }
func (self *VMEnv) GasLimit() *big.Int       { return self.header.GasLimit }
func (self *VMEnv) Value() *big.Int          { return self.msg.Value() }
func (self *VMEnv) Db() vm.Database          { return self.state }
func (self *VMEnv) Depth() int               { return self.depth }
func (self *VMEnv) SetDepth(i int)           { self.depth = i }
func (self *VMEnv) VmType() vm.Type          { return self.typ }
func (self *VMEnv) SetVmType(t vm.Type)      { self.typ = t }
func (self *VMEnv) GetHash(n uint64) common.Hash {
	for header := self.chain.GetHeader(self.header.ParentHash, self.header.Number.Uint64()-1); header != nil; header = self.chain.GetHeader(header.ParentHash, header.Number.Uint64()-1) {
		if header.GetNumberU64() == n {
			return header.Hash()
		}
	}

	return common.Hash{}
}

func (self *VMEnv) AddLog(log *vm.Log) {
	//self.state.AddLog(log)
}
func (self *VMEnv) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}

func (self *VMEnv) MakeSnapshot() vm.Database {
	return &VMState{ctx: self.ctx, state: self.state.state.Copy(), env: self}
}

func (self *VMEnv) SetSnapshot(copy vm.Database) {
	self.state.state.Set(copy.(*VMState).state)
}

func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) {
	core.Transfer(from, to, amount)
}

func (self *VMEnv) Call(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.Call(self, me, addr, data, gas, price, value)
}
func (self *VMEnv) CallCode(me vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.CallCode(self, me, addr, data, gas, price, value)
}

func (self *VMEnv) Create(me vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return core.Create(self, me, data, gas, price, value)
}

func (self *VMEnv) StructLogs() []vm.StructLog {
	return self.logs
}

func (self *VMEnv) AddStructLog(log vm.StructLog) {
	self.logs = append(self.logs, log)
}

// Error returns the error (if any) that happened during execution.
func (self *VMEnv) Error() error {
	return self.err
}

// VMState is a wrapper for the light state that holds the actual context and
// passes it to any state operation that requires it.
type VMState struct {
	vm.Database
	ctx   context.Context
	state *LightState
	env   *VMEnv
}

// errHandler handles and stores any state error that happens during execution.
func (s *VMState) errHandler(err error) {
	if err != nil && s.env.err == nil {
		s.env.err = err
	}
}

// GetAccount returns the account object of the given account or nil if the
// account does not exist
func (s *VMState) GetAccount(addr common.Address) vm.Account {
	so, err := s.state.GetStateObject(s.ctx, addr)
	s.errHandler(err)
	if err != nil {
		// return a dummy state object to avoid panics
		so = s.state.newStateObject(addr)
	}
	return so
}

// CreateAccount creates creates a new account object and takes ownership.
func (s *VMState) CreateAccount(addr common.Address) vm.Account {
	so, err := s.state.CreateStateObject(s.ctx, addr)
	s.errHandler(err)
	if err != nil {
		// return a dummy state object to avoid panics
		so = s.state.newStateObject(addr)
	}
	return so
}

// AddBalance adds the given amount to the balance of the specified account
func (s *VMState) AddBalance(addr common.Address, amount *big.Int) {
	err := s.state.AddBalance(s.ctx, addr, amount)
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

// Delete marks an account to be removed and clears its balance
func (s *VMState) Delete(addr common.Address) bool {
	res, err := s.state.Delete(s.ctx, addr)
	s.errHandler(err)
	return res
}

// Exist returns true if an account exists at the given address
func (s *VMState) Exist(addr common.Address) bool {
	res, err := s.state.HasAccount(s.ctx, addr)
	s.errHandler(err)
	return res
}

// IsDeleted returns true if the given account has been marked for deletion
// or false if the account does not exist
func (s *VMState) IsDeleted(addr common.Address) bool {
	res, err := s.state.IsDeleted(s.ctx, addr)
	s.errHandler(err)
	return res
}
