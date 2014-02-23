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

func (c *Contract) Addr(addr []byte) *ethutil.Value {
	return ethutil.NewValueFromBytes([]byte(c.state.Get(string(addr))))
}

func (c *Contract) SetAddr(addr []byte, value interface{}) {
	c.state.Update(string(addr), string(ethutil.NewValue(value).Encode()))
}

func (c *Contract) State() *ethutil.Trie {
	return c.state
}
