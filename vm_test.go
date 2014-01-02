package main

import (
  "fmt"
  "testing"
  _"encoding/hex"
)


func TestVm(t *testing.T) {
  db, _ := NewMemDatabase()
  Db = db

  tx := NewTransaction("\x00", 20, []string{
    "PSH 10",
  })

  block := CreateBlock("", 0, "", "", 0, 0, "", []*Transaction{tx})
  db.Put(block.Hash(), block.MarshalRlp())

  bm := NewBlockManager()
  bm.ProcessBlock( block )
  tx1 := &Transaction{}
  tx1.UnmarshalRlp([]byte(block.state.Get(tx.recipient)))
  fmt.Println(tx1)
}

