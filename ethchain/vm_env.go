package ethchain

import (
	"github.com/ethereum/eth-go/ethstate"
	"math/big"
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
	}
}

func (self *VMEnv) Origin() []byte         { return self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int  { return self.block.Number }
func (self *VMEnv) PrevHash() []byte       { return self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte       { return self.block.Coinbase }
func (self *VMEnv) Time() int64            { return self.block.Time }
func (self *VMEnv) Difficulty() *big.Int   { return self.block.Difficulty }
func (self *VMEnv) Value() *big.Int        { return self.tx.Value }
func (self *VMEnv) State() *ethstate.State { return self.state }
