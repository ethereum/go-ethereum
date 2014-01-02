package main

import (
  _"fmt"
  "time"
  _"bytes"
)

type Block struct {
  // The number of this block
  number        uint32
  // Hash to the previous block
  prevHash      string
  // Uncles of this block
  uncles        []*Block
  coinbase      string
  // state xxx
  state         *Trie
  difficulty    uint32
  // Creation time
  time          int64
  nonce         uint32
  // List of transactions and/or contracts
  transactions  []*Transaction

  extra         string
}

// New block takes a raw encoded string
func NewBlock(raw []byte) *Block {
  block := &Block{}
  block.UnmarshalRlp(raw)

  return block
}

// Creates a new block. This is currently for testing
func CreateTestBlock(/* TODO use raw data */transactions []*Transaction) *Block {
  block := &Block{
    // Slice of transactions to include in this block
    transactions: transactions,
    number: 1,
    prevHash: "1234",
    coinbase: "me",
    difficulty: 10,
    nonce: 0,
    time: time.Now().Unix(),
  }

  return block
}

func CreateBlock(root string, num int, prevHash string, base string, difficulty int, nonce int, extra string, txes []*Transaction) *Block {
  block := &Block{
    // Slice of transactions to include in this block
    transactions: txes,
    number: uint32(num),
    prevHash: prevHash,
    coinbase: base,
    difficulty: uint32(difficulty),
    nonce: uint32(nonce),
    time: time.Now().Unix(),
    extra: extra,
  }
  block.state = NewTrie(Db, root)
  for _, tx := range txes {
    block.state.Update(tx.recipient, string(tx.MarshalRlp()))
  }

  return block
}

func (block *Block) Update() {
}

// Returns a hash of the block
func (block *Block) Hash() []byte {
  return Sha256Bin(block.MarshalRlp())
}

func (block *Block) MarshalRlp() []byte {
  // Marshal the transactions of this block
  encTx := make([]string, len(block.transactions))
  for i, tx := range block.transactions {
    // Cast it to a string (safe)
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
    block.state.root,
    // Sha of tx
    string(Sha256Bin([]byte(Encode(encTx)))),
    block.difficulty,
    uint64(block.time),
    block.nonce,
    block.extra,
  }

  // TODO
  uncles := []interface{}{}
  // Encode a slice interface which contains the header and the list of transactions.
  return Encode([]interface{}{header, encTx, uncles})
}

func (block *Block) UnmarshalRlp(data []byte) {
  t, _ := Decode(data,0)

  // interface slice assertion
  if slice, ok := t.([]interface{}); ok {
    // interface slice assertion
    if header, ok := slice[0].([]interface{}); ok {
      if number, ok := header[0].(uint8); ok {
        block.number = uint32(number)
      }

      if prevHash, ok := header[1].([]uint8); ok {
        block.prevHash = string(prevHash)
      }

      // sha of uncles is header[2]

      if coinbase, ok := header[3].([]byte); ok {
        block.coinbase = string(coinbase)
      }

      if state, ok := header[4].([]uint8); ok {
        // XXX The database is currently a global variable defined in testing.go
        // This will eventually go away and the database will grabbed from the public server
        // interface
        block.state = NewTrie(Db, string(state))
      }

      // sha is header[5]

      // It's either 8bit or 64
      if difficulty, ok := header[6].(uint8); ok {
        block.difficulty = uint32(difficulty)
      }
      if difficulty, ok := header[6].(uint64); ok {
        block.difficulty = uint32(difficulty)
      }

      // It's either 8bit or 64
      if time, ok := header[7].(uint8); ok {
        block.time = int64(time)
      }
      if time, ok := header[7].(uint64); ok {
        block.time = int64(time)
      }

      if nonce, ok := header[8].(uint8); ok {
        block.nonce = uint32(nonce)
      }

      if extra, ok := header[9].([]byte); ok {
        block.extra = string(extra)
      }
    }

    if txSlice, ok := slice[1].([]interface{}); ok {
      // Create transaction slice equal to decoded tx interface slice
      block.transactions = make([]*Transaction, len(txSlice))

      // Unmarshal transactions
      for i, tx := range txSlice {
        if t, ok := tx.([]byte); ok {
          tx := &Transaction{}
          // Use the unmarshaled data to unmarshal the transaction
          // t is still decoded.
          tx.UnmarshalRlp(t)

          block.transactions[i] = tx
        }
      }
    }
  }
}
