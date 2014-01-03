package main

import (
  _"fmt"
)

type Contract struct {
  t uint32 // contract is always 1
  amount uint64 // ???
  state  *Trie
}

func NewContract(amount uint64, root []byte) *Contract {
  contract := &Contract{t: 1, amount: amount}
  contract.state = NewTrie(Db, string(root))

  return  contract
}

func (c *Contract) MarshalRlp() []byte {
  return Encode([]interface{}{c.t, c.amount, c.state.root})
}

func (c *Contract) UnmarshalRlp(data []byte) {
  t, _ := Decode(data, 0)

  if slice, ok := t.([]interface{}); ok {
    if t, ok := slice[0].(uint8); ok {
      c.t = uint32(t)
    }

    if amount, ok := slice[1].(uint8); ok {
      c.amount = uint64(amount)
    } else if amount, ok := slice[1].(uint16); ok {
      c.amount = uint64(amount)
    } else if amount, ok := slice[1].(uint32); ok {
      c.amount = uint64(amount)
    } else if amount, ok := slice[1].(uint64); ok {
      c.amount = amount
    }

    if root, ok := slice[2].([]uint8); ok {
      c.state = NewTrie(Db, string(root))
    }
  }
}

type Ether struct {
  t uint32
  amount uint64
  nonce string
}

func NewEtherFromData(data []byte) *Ether {
  ether := &Ether{}
  ether.UnmarshalRlp(data)

  return ether
}

func (e *Ether) MarshalRlp() []byte {
  return Encode([]interface{}{e.t, e.amount, e.nonce})
}

func (e *Ether) UnmarshalRlp(data []byte) {
  t, _ := Decode(data, 0)

  if slice, ok := t.([]interface{}); ok {
    if t, ok := slice[0].(uint8); ok {
      e.t = uint32(t)
    }

    if amount, ok := slice[1].(uint8); ok {
      e.amount = uint64(amount)
    }

    if nonce, ok := slice[2].([]uint8); ok {
      e.nonce = string(nonce)
    }
  }
}
