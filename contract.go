package main

import (
)

type Contract struct {
  active int
  amount uint32 // ???
  state  *Trie
}
func NewContract(amount uint32, root []byte) *Contract {
  contract := &Contract{active: 1, amount: amount}
  contract.state = NewTrie(Db, string(root))

  return  contract
}
func (c *Contract) MarshalRlp() []byte {
  // Prepare the transaction for serialization
  preEnc := []interface{}{uint32(c.active), c.amount, c.state.root}

  return Encode(preEnc)
}

func (c *Contract) UnmarshalRlp(data []byte) {
  t, _ := Decode(data, 0)

  if slice, ok := t.([]interface{}); ok {
    if active, ok := slice[0].(uint8); ok {
      c.active = int(active)
    }

    if amount, ok := slice[1].(uint8); ok {
      c.amount = uint32(amount)
    }

    if root, ok := slice[2].([]uint8); ok {
      c.state = NewTrie(Db, string(root))
    }
  }
}
