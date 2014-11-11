package vm

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
)

type Environment interface {
	State() *state.State

	Origin() []byte
	BlockNumber() *big.Int
	PrevHash() []byte
	Coinbase() []byte
	Time() int64
	Difficulty() *big.Int
	BlockHash() []byte
	GasLimit() *big.Int
	Transfer(from, to Account, amount *big.Int) error
	AddLog(*state.Log)
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

	// Add default LOG. Default = big(sender.addr) + 1
	//addr := ethutil.BigD(receiver.Address())
	//tx.addLog(vm.Log{sender.Address(), [][]byte{ethutil.U256(addr.Add(addr, ethutil.Big1)).Bytes()}, nil})

	return nil
}
