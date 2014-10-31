package chain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethstate"
	"github.com/ethereum/go-ethereum/vm"
)

type VMEnv struct {
	state *ethstate.State
	block *Block
	tx    *Transaction
}

func NewEnv(state *ethstate.State, tx *Transaction, block *Block) *VMEnv {
	return &VMEnv{
		state: state,
		block: block,
		tx:    tx,
	}
}

func (self *VMEnv) Origin() []byte         { return self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int  { return self.block.Number }
func (self *VMEnv) PrevHash() []byte       { return self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte       { return self.block.Coinbase }
func (self *VMEnv) Time() int64            { return self.block.Time }
func (self *VMEnv) Difficulty() *big.Int   { return self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte      { return self.block.Hash() }
func (self *VMEnv) Value() *big.Int        { return self.tx.Value }
func (self *VMEnv) State() *ethstate.State { return self.state }
func (self *VMEnv) GasLimit() *big.Int     { return self.block.GasLimit }
func (self *VMEnv) AddLog(log ethstate.Log) {
	self.state.AddLog(log)
}
func (self *VMEnv) Transfer(from, to vm.Account, amount *big.Int) error {
	return vm.Transfer(from, to, amount)
}
