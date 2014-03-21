package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Account struct {
	address []byte
	Amount  *big.Int
	Nonce   uint64
}

func NewAccount(address []byte, amount *big.Int) *Account {
	return &Account{address, amount, 0}
}

func NewAccountFromData(address, data []byte) *Account {
	account := &Account{address: address}
	account.RlpDecode(data)

	return account
}

func (a *Account) AddFee(fee *big.Int) {
	a.AddFunds(fee)
}

func (a *Account) AddFunds(funds *big.Int) {
	a.Amount.Add(a.Amount, funds)
}

func (a *Account) Address() []byte {
	return a.address
}

// Implements Callee
func (a *Account) ReturnGas(value *big.Int, state *State) {
	// Return the value back to the sender
	a.AddFunds(value)
	state.UpdateAccount(a.address, a)
}

func (a *Account) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{a.Amount, a.Nonce})
}

func (a *Account) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	a.Amount = decoder.Get(0).BigInt()
	a.Nonce = decoder.Get(1).Uint()
}

type AddrStateStore struct {
	states map[string]*AccountState
}

func NewAddrStateStore() *AddrStateStore {
	return &AddrStateStore{states: make(map[string]*AccountState)}
}

func (s *AddrStateStore) Add(addr []byte, account *Account) *AccountState {
	state := &AccountState{Nonce: account.Nonce, Account: account}
	s.states[string(addr)] = state
	return state
}

func (s *AddrStateStore) Get(addr []byte) *AccountState {
	return s.states[string(addr)]
}

type AccountState struct {
	Nonce   uint64
	Account *Account
}
