package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type VMEnv struct {
	state *state.StateDB
	block *types.Block
	tx    *types.Transaction
	depth int
}

func NewEnv(state *state.StateDB, tx *types.Transaction, block *types.Block) *VMEnv {
	return &VMEnv{
		state: state,
		block: block,
		tx:    tx,
	}
}

func (self *VMEnv) Origin() []byte        { return self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int { return self.block.Number }
func (self *VMEnv) PrevHash() []byte      { return self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte      { return self.block.Coinbase }
func (self *VMEnv) Time() int64           { return self.block.Time }
func (self *VMEnv) Difficulty() *big.Int  { return self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte     { return self.block.Hash() }
func (self *VMEnv) Value() *big.Int       { return self.tx.Value }
func (self *VMEnv) State() *state.StateDB { return self.state }
func (self *VMEnv) GasLimit() *big.Int    { return self.block.GasLimit }
func (self *VMEnv) Depth() int            { return self.depth }
func (self *VMEnv) SetDepth(i int)        { self.depth = i }
func (self *VMEnv) AddLog(log state.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *VMEnv) vm(addr, data []byte, gas, price, value *big.Int) *Execution {
	evm := vm.New(self, vm.DebugVmTy)

	return NewExecution(evm, addr, data, gas, price, value)
}

func (self *VMEnv) Call(me vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Call(addr, me)
}
func (self *VMEnv) CallCode(me vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(me.Address(), data, gas, price, value)
	return exe.Call(addr, me)
}

func (self *VMEnv) Create(me vm.ClosureRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ClosureRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(me)
}
