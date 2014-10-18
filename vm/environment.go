package vm

import (
	"math/big"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type Environment interface {
	State() *ethstate.State

	Origin() []byte
	BlockNumber() *big.Int
	PrevHash() []byte
	Coinbase() []byte
	Time() int64
	Difficulty() *big.Int
	BlockHash() []byte
	GasLimit() *big.Int
}

type Object interface {
	GetStorage(key *big.Int) *ethutil.Value
	SetStorage(key *big.Int, value *ethutil.Value)
}
