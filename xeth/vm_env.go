package xeth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

type VMEnv struct {
	chain  *core.ChainManager
	state  *state.StateDB
	block  *types.Block
	value  *big.Int
	sender []byte

	depth int
}

func NewEnv(chain *core.ChainManager, state *state.StateDB, block *types.Block, value *big.Int, sender []byte) *VMEnv {
	return &VMEnv{
		state:  state,
		block:  block,
		value:  value,
		sender: sender,
	}
}

func (self *VMEnv) Origin() []byte        { return self.sender }
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

func (self *VMEnv) Call(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Call(addr, me)
}
func (self *VMEnv) CallCode(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error) {
	exe := self.vm(me.Address(), data, gas, price, value)
	return exe.Call(addr, me)
}

func (self *VMEnv) Create(me vm.ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(addr, data, gas, price, value)
	return exe.Create(me)
}
