package main

import (
  "fmt"
  "time"
  _"bytes"
)

type Block struct {
  number        uint32
  prevHash      string
  uncles        []*Block
  coinbase      string
  // state xxx
  difficulty    uint32
  time          time.Time
  nonce         uint32
  transactions  []*Transaction
}

func NewBlock(/* TODO use raw data */transactions []*Transaction) *Block {
  block := &Block{
    // Slice of transactions to include in this block
    transactions: transactions,
    number: 1,
    prevHash: "1234",
    coinbase: "me",
    difficulty: 10,
    nonce: 0,

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

  /* I made up the block. It should probably contain different data or types. It sole purpose now is testing */
  header := []interface{}{
    block.number,
    block.prevHash,
    // Sha of uncles
    "",
    block.coinbase,
    // root state
    "",
    string(Sha256Bin([]byte(RlpEncode(encTx)))),
    block.difficulty,
    block.time.String(),
    block.nonce,
    // extra?
  }

  encoded := Encode([]interface{}{header, encTx})

  return encoded
}

func (block *Block) UnmarshalRlp(data []byte) {
  t, _ := Decode(data,0)
  if slice, ok := t.([]interface{}); ok {
    if header, ok := slice[0].([]interface{}); ok {
      if number, ok := header[0].(uint8); ok {
        block.number = uint32(number)
      }

      if prevHash, ok := header[1].([]byte); ok {
        block.prevHash = string(prevHash)
      }

      // sha of uncles is header[2]

      if coinbase, ok := header[3].([]byte); ok {
        block.coinbase = string(coinbase)
      }

      // state is header[header[4]

      // sha is header[5]

      // It's either 8bit or 64
      if difficulty, ok := header[6].(uint8); ok {
        block.difficulty = uint32(difficulty)
      }
      if difficulty, ok := header[6].(uint64); ok {
        block.difficulty = uint32(difficulty)
      }

      if time, ok := header[7].([]byte); ok {
        fmt.Sprintf("Time is: ", string(time))
      }

      if nonce, ok := header[8].(uint8); ok {
        block.nonce = uint32(nonce)
      }
    }

    if txSlice, ok := slice[1].([]interface{}); ok {
      block.transactions = make([]*Transaction, len(txSlice))

      for i, tx := range txSlice {
        if t, ok := tx.([]byte); ok {
          tx := &Transaction{}
          tx.UnmarshalRlp(t)

          block.transactions[i] = tx
        }
      }
    }
  }
}
