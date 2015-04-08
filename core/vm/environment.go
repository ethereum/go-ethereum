package vm

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/rlp"
)

type Environment interface {
	State() *state.StateDB

	Origin() common.Address
	BlockNumber() *big.Int
	GetHash(n uint64) common.Hash
	Coinbase() common.Address
	Time() int64
	Difficulty() *big.Int
	GasLimit() *big.Int
	Transfer(from, to Account, amount *big.Int) error
	AddLog(*state.Log)

	VmType() Type

	Depth() int
	SetDepth(i int)

	Call(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	CallCode(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, ContextRef)
}

type Account interface {
	SubBalance(amount *big.Int)
	AddBalance(amount *big.Int)
	Balance() *big.Int
	Address() common.Address
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
	address common.Address
	topics  []common.Hash
	data    []byte
	log     uint64
}

func (self *Log) Address() common.Address {
	return self.address
}

func (self *Log) Topics() []common.Hash {
	return self.topics
}

func (self *Log) Data() []byte {
	return self.data
}

func (self *Log) Number() uint64 {
	return self.log
}

func (self *Log) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{self.address, self.topics, self.data})
}

/*
func (self *Log) RlpData() interface{} {
	return []interface{}{self.address, common.ByteSliceToInterface(self.topics), self.data}
}
*/

func (self *Log) String() string {
	return fmt.Sprintf("{%x %x %x}", self.address, self.data, self.topics)
}
