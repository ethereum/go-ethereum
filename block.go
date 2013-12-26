package main

import (
  _"fmt"
)

type Block struct {
  transactions []*Transaction
}

func NewBlock(/* TODO use raw data */transactions []*Transaction) *Block {
  block := &Block{
    // Slice of transactions to include in this block
    transactions: transactions,
  }

  return block
}

func (block *Block) Update() {
}
