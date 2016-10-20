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

package runtime

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// Env is a basic runtime environment required for running the EVM.
type Env struct {
	chainConfig *params.ChainConfig
	depth       int
	state       *state.StateDB

	origin   common.Address
	coinbase common.Address

	number     *big.Int
	time       *big.Int
	difficulty *big.Int
	gasLimit   *big.Int

	getHashFn func(uint64) common.Hash

	evm *vm.EVM
}

// NewEnv returns a new vm.Environment
func NewEnv(cfg *Config, state *state.StateDB) vm.Environment {
	env := &Env{
		chainConfig: cfg.ChainConfig,
		state:       state,
		origin:      cfg.Origin,
		coinbase:    cfg.Coinbase,
		number:      cfg.BlockNumber,
		time:        cfg.Time,
		difficulty:  cfg.Difficulty,
		gasLimit:    cfg.GasLimit,
	}
	env.evm = vm.New(env, vm.Config{
		Debug:     cfg.Debug,
		EnableJit: !cfg.DisableJit,
		ForceJit:  !cfg.DisableJit,
	})

	return env
}

func (self *Env) ChainConfig() *params.ChainConfig { return self.chainConfig }
func (self *Env) Vm() vm.Vm                        { return self.evm }
func (self *Env) Origin() common.Address           { return self.origin }
func (self *Env) BlockNumber() *big.Int            { return self.number }
func (self *Env) Coinbase() common.Address         { return self.coinbase }
func (self *Env) Time() *big.Int                   { return self.time }
func (self *Env) Difficulty() *big.Int             { return self.difficulty }
func (self *Env) Db() vm.Database                  { return self.state }
func (self *Env) GasLimit() *big.Int               { return self.gasLimit }
func (self *Env) VmType() vm.Type                  { return vm.StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return self.getHashFn(n)
}
func (self *Env) AddLog(log *vm.Log) {
	self.state.AddLog(log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) CanTransfer(from common.Address, balance *big.Int) bool {
	return self.state.GetBalance(from).Cmp(balance) >= 0
}
func (self *Env) SnapshotDatabase() int {
	return self.state.Snapshot()
}
func (self *Env) RevertToSnapshot(snapshot int) {
	self.state.RevertToSnapshot(snapshot)
}

func (self *Env) Transfer(from, to vm.Account, amount *big.Int) {
	core.Transfer(from, to, amount)
}

func (self *Env) Call(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.Call(self, caller, addr, data, gas, price, value)
}
func (self *Env) CallCode(caller vm.ContractRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	return core.CallCode(self, caller, addr, data, gas, price, value)
}

func (self *Env) DelegateCall(me vm.ContractRef, addr common.Address, data []byte, gas, price *big.Int) ([]byte, error) {
	return core.DelegateCall(self, me, addr, data, gas, price)
}

func (self *Env) Create(caller vm.ContractRef, data []byte, gas, price, value *big.Int) ([]byte, common.Address, error) {
	return core.Create(self, caller, data, gas, price, value)
}
