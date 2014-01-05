package main

import (
  "fmt"
)

// This will eventually go away
var Db *MemDatabase

func Testing() {
  db, _ := NewMemDatabase()
  Db = db

  bm := NewBlockManager()

  tx := NewTransaction("\x00", 20, []string{"PSH 10"})
  txData := tx.MarshalRlp()

  copyTx := &Transaction{}
  copyTx.UnmarshalRlp(txData)
  fmt.Println(tx)
  fmt.Println(copyTx)

  tx2 := NewTransaction("\x00", 20, []string{"SET 10 6", "LD 10 10"})

  blck := CreateTestBlock([]*Transaction{tx2, tx})

  bm.ProcessBlock( blck )

  fmt.Println("GenesisBlock:", GenisisBlock, "hash", string(GenisisBlock.Hash()))
}
