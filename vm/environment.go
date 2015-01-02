package vm

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type Environment interface {
	State() *state.StateDB

	Origin() []byte
	BlockNumber() *big.Int
	PrevHash() []byte
	Coinbase() []byte
	Time() int64
	Difficulty() *big.Int
	BlockHash() []byte
	GasLimit() *big.Int
	Transfer(from, to Account, amount *big.Int) error
	AddLog(state.Log)

	Depth() int
	SetDepth(i int)

	Call(me ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error)
	CallCode(me ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me ContextRef, addr, data []byte, gas, price, value *big.Int) ([]byte, error, ContextRef)
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

type Log struct {
	address []byte
	topics  [][]byte
	data    []byte
}

func (self *Log) Address() []byte {
	return self.address
}

func (self *Log) Topics() [][]byte {
	return self.topics
}

func (self *Log) Data() []byte {
	return self.data
}

func (self *Log) RlpData() interface{} {
	return []interface{}{self.address, ethutil.ByteSliceToInterface(self.topics), self.data}
}

func (self *Log) String() string {
	return fmt.Sprintf("[A=%x T=%x D=%x]", self.address, self.topics, self.data)
}
