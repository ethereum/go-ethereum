package vm

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
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
	AddStructLog(StructLog)
	StructLogs() []StructLog

	VmType() Type

	Depth() int
	SetDepth(i int)

	Call(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	CallCode(me ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error)
	Create(me ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, ContextRef)
}

type StructLog struct {
	Pc     uint64
	Op     OpCode
	Gas    *big.Int
	Memory []byte
	Stack  []*big.Int
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
