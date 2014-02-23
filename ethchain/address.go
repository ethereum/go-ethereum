package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Address struct {
	Amount *big.Int
	Nonce  uint64
}

func NewAddress(amount *big.Int) *Address {
	return &Address{Amount: amount, Nonce: 0}
}

func NewAddressFromData(data []byte) *Address {
	address := &Address{}
	address.RlpDecode(data)

	return address
}

func (a *Address) AddFee(fee *big.Int) {
	a.Amount.Add(a.Amount, fee)
}

func (a *Address) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{a.Amount, a.Nonce})
}

func (a *Address) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	a.Amount = decoder.Get(0).BigInt()
	a.Nonce = decoder.Get(1).Uint()
}

type AddrStateStore struct {
	states map[string]*AddressState
}

func NewAddrStateStore() *AddrStateStore {
	return &AddrStateStore{states: make(map[string]*AddressState)}
}

func (s *AddrStateStore) Add(addr []byte, account *Address) *AddressState {
	state := &AddressState{Nonce: account.Nonce, Account: account}
	s.states[string(addr)] = state
	return state
}

func (s *AddrStateStore) Get(addr []byte) *AddressState {
	return s.states[string(addr)]
}

type AddressState struct {
	Nonce   uint64
	Account *Address
}
