package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type Contract struct {
	Amount *big.Int
	Nonce  uint64
	//state  *ethutil.Trie
	state      *State
	address    []byte
	script     []byte
	initScript []byte
}

func NewContract(address []byte, Amount *big.Int, root []byte) *Contract {
	contract := &Contract{address: address, Amount: Amount, Nonce: 0}
	contract.state = NewState(ethutil.NewTrie(ethutil.Config.Db, string(root)))

	return contract
}

func NewContractFromBytes(address, data []byte) *Contract {
	contract := &Contract{address: address}
	contract.RlpDecode(data)

	return contract
}

func (c *Contract) Addr(addr []byte) *ethutil.Value {
	return ethutil.NewValueFromBytes([]byte(c.state.trie.Get(string(addr))))
}

func (c *Contract) SetAddr(addr []byte, value interface{}) {
	c.state.trie.Update(string(addr), string(ethutil.NewValue(value).Encode()))
}

func (c *Contract) State() *State {
	return c.state
}

func (c *Contract) GetMem(num *big.Int) *ethutil.Value {
	nb := ethutil.BigToBytes(num, 256)

	return c.Addr(nb)
}

func (c *Contract) GetInstr(pc *big.Int) *ethutil.Value {
	if int64(len(c.script)-1) < pc.Int64() {
		return ethutil.NewValue(0)
	}

	return ethutil.NewValueFromBytes([]byte{c.script[pc.Int64()]})
}

func (c *Contract) SetMem(num *big.Int, val *ethutil.Value) {
	addr := ethutil.BigToBytes(num, 256)
	c.state.trie.Update(string(addr), string(val.Encode()))
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *Contract) ReturnGas(val *big.Int, state *State) {
	c.Amount.Add(c.Amount, val)
}

func (c *Contract) Address() []byte {
	return c.address
}

func (c *Contract) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{c.Amount, c.Nonce, c.state.trie.Root})
}

func (c *Contract) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	c.Amount = decoder.Get(0).BigInt()
	c.Nonce = decoder.Get(1).Uint()
	c.state = NewState(ethutil.NewTrie(ethutil.Config.Db, decoder.Get(2).Interface()))
}

func MakeContract(tx *Transaction, state *State) *Contract {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.Hash()[12:]

		value := tx.Value
		contract := NewContract(addr, value, []byte(""))
		state.trie.Update(string(addr), string(contract.RlpEncode()))
		contract.script = tx.Data
		contract.initScript = tx.Init

		/*
			for i, val := range tx.Data {
					if len(val) > 0 {
						bytNum := ethutil.BigToBytes(big.NewInt(int64(i)), 256)
						contract.state.trie.Update(string(bytNum), string(ethutil.Encode(val)))
					}
			}
		*/
		state.trie.Update(string(addr), string(contract.RlpEncode()))

		return contract
	}

	return nil
}
