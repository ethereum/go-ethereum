package vm

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
)

type Environment interface {
	//State() *state.State

	Origin() []byte
	BlockNumber() *big.Int
	PrevHash() []byte
	Coinbase() []byte
	Time() int64
	Difficulty() *big.Int
	BlockHash() []byte
	GasLimit() *big.Int

	Transfer(from, to Account, amount *big.Int) error
	AddLog(addr []byte, topics [][]byte, data []byte)
	DeleteAccount(addr []byte)
	SetState(addr, key, value []byte)
	GetState(addr, key []byte) []byte
	Balance(addr []byte) *big.Int
	AddBalance(addr []byte, balance *big.Int)
	GetCode(addr []byte) []byte
	Refund(addr []byte, gas, price *big.Int)
}

type Object interface {
	GetStorage(key *big.Int) *ethutil.Value
	SetStorage(key *big.Int, value *ethutil.Value)
}

type Account interface {
	SubBalance(amount *big.Int)
	AddBalance(amount *big.Int)
	Balance() *big.Int
}

// generic transfer method
func Transfer(from, to Account, amount *big.Int) error {
	if from.Balance().Cmp(amount) < 0 {
		return errors.New("Insufficient balance in account")
	}

	from.SubBalance(amount)
	to.AddBalance(amount)

	return nil
}
