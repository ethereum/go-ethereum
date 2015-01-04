package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type VMEnv struct {
	chain *core.ChainManager
	state *state.StateDB
	block *types.Block

	transactor []byte
	value      *big.Int

	depth int
	Gas   *big.Int
}

func NewEnv(chain *core.ChainManager, state *state.StateDB, block *types.Block, transactor []byte, value *big.Int) *VMEnv {
	return &VMEnv{
		chain:      chain,
		state:      state,
		block:      block,
		transactor: transactor,
		value:      value,
	}
}

func (self *VMEnv) Origin() []byte        { return self.transactor }
func (self *VMEnv) BlockNumber() *big.Int { return self.block.Number() }
func (self *VMEnv) PrevHash() []byte      { return self.block.ParentHash() }
func (self *VMEnv) Coinbase() []byte      { return self.block.Coinbase() }
func (self *VMEnv) Time() int64           { return self.block.Time() }
func (self *VMEnv) Difficulty() *big.Int  { return self.block.Difficulty() }
func (self *VMEnv) GasLimit() *big.Int    { return self.block.GasLimit() }
func (self *VMEnv) Value() *big.Int       { return self.value }
func (self *VMEnv) State() *state.StateDB { return self.state }
func (self *VMEnv) Depth() int            { return self.depth }
func (self *VMEnv) SetDepth(i int)        { self.depth = i }
func (self *VMEnv) GetHash(n uint64) []byte {
	if block := self.chain.GetBlockByNumber(n); block != nil {
		return block.Hash()
	}

	return nil
}
func (self *VMEnv) AddLog(log state.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}

func (self *VMEnv) vm(addr, data []byte, gas, price, value *big.Int) *core.Execution {
	return core.NewExecution(self, addr, data, gas, price, value)
}

func (self *VMEnv) Call(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err
}
func (self *VMEnv) CallCode(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(caller.Address(), data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *VMEnv) Create(caller vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(caller)
}
