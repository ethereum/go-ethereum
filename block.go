package main

import (
  "fmt"
  "time"
  _"bytes"
  _"encoding/hex"
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
    // Create contract if there's no recipient
    if tx.recipient == "" {
      addr := tx.Hash()

      contract := NewContract(tx.value, []byte(""))
      block.state.Update(string(addr), string(contract.MarshalRlp()))
      for i, val := range tx.data {
        contract.state.Update(string(NumberToBytes(uint64(i), 32)), val)
      }
      block.UpdateContract(addr, contract)
    }
  }

  return block
}

func (block *Block) GetContract(addr []byte) *Contract {
  data := block.state.Get(string(addr))
  if data == "" {
    return nil
  }

  contract := &Contract{}
  contract.UnmarshalRlp([]byte(data))

  return contract
}

func (block *Block) UpdateContract(addr []byte, contract *Contract) {
  block.state.Update(string(addr), string(contract.MarshalRlp()))
}


func (block *Block) PayFee(addr []byte, fee uint64) bool {
  contract := block.GetContract(addr)
  // If we can't pay the fee return
  if contract == nil || contract.amount < fee {
    fmt.Println("Contract has insufficient funds", contract.amount, fee)

    return false
  }

  contract.amount -= fee
  block.state.Update(string(addr), string(contract.MarshalRlp()))

  data := block.state.Get(string(block.coinbase))

  // Get the ether (coinbase) and add the fee (gief fee to miner)
  ether := NewEtherFromData([]byte(data))
  ether.amount += fee

  block.state.Update(string(block.coinbase), string(ether.MarshalRlp()))

  return true
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
  decoder := NewRlpDecoder(data)

  header := decoder.Get(0)
  block.number = uint32(header.Get(0).AsUint())
  block.prevHash = header.Get(1).AsString()
  // sha of uncles is header[2]
  block.coinbase = header.Get(3).AsString()
  block.state = NewTrie(Db, header.Get(4).AsString())
  block.difficulty = uint32(header.Get(5).AsUint())
  block.time = int64(header.Get(6).AsUint())
  block.nonce = uint32(header.Get(7).AsUint())
  block.extra = header.Get(8).AsString()

  txes := decoder.Get(1)
  block.transactions = make([]*Transaction, txes.Length())
  for i := 0; i < txes.Length(); i++ {
    tx := &Transaction{}
    tx.UnmarshalRlp(txes.Get(i).AsBytes())
    block.transactions[i] = tx
  }
}
