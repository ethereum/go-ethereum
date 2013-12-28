package main

import (
  "fmt"
  "time"
)

type Block struct {
  RlpSerializer

  number        uint32
  prevHash      string
  uncles        []*Block
  coinbase      string
  // state xxx
  difficulty    int
  time          time.Time
  nonce         int
  transactions  []*Transaction
}

func NewBlock(/* TODO use raw data */transactions []*Transaction) *Block {
  block := &Block{
    // Slice of transactions to include in this block
    transactions: transactions,

    time: time.Now(),
  }

  return block
}

func (block *Block) Update() {
}

func (block *Block) Hash() string {
  return Sha256Hex(block.MarshalRlp())
}

func (block *Block) MarshalRlp() []byte {
  // Encoding method requires []interface{} type. It's actual a slice of strings
  encTx := make([]string, len(block.transactions))
  for i, tx := range block.transactions {
    encTx[i] = string(tx.MarshalRlp())
  }

  header := []interface{}{
    block.number,
    //block.prevHash,
    // Sha of uncles
    //block.coinbase,
    // root state
    //Sha256Bin([]byte(RlpEncode(encTx))),
    //block.difficulty,
    //block.time,
    //block.nonce,
    // extra?
  }

  return Encode([]interface{}{header, encTx})
}

func (block *Block) UnmarshalRlp(data []byte) {
  fmt.Printf("%q\n", data)
  t, _ := Decode(data,0)
  if slice, ok := t.([]interface{}); ok {
    if txes, ok := slice[1].([]interface{}); ok {
      fmt.Println(txes[0])
    }
  }
}
