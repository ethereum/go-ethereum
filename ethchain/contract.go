package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Contract struct {
	Amount *big.Int
	Nonce  uint64
	state  *ethutil.Trie
}

func NewContract(Amount *big.Int, root []byte) *Contract {
	contract := &Contract{Amount: Amount, Nonce: 0}
	contract.state = ethutil.NewTrie(ethutil.Config.Db, string(root))

	return contract
}

func (c *Contract) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{c.Amount, c.Nonce, c.state.Root})
}

func (c *Contract) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	c.Amount = decoder.Get(0).BigInt()
	c.Nonce = decoder.Get(1).Uint()
	c.state = ethutil.NewTrie(ethutil.Config.Db, decoder.Get(2).Interface())
}

func (c *Contract) State() *ethutil.Trie {
	return c.state
}

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
