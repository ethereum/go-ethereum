package main

import (
  "fmt"
  "testing"
)


func TestVm(t *testing.T) {
  db, _ := NewMemDatabase()
  Db = db

  tx := NewTransaction("", 20, []string{
    "PSH 10",
  })

  block := CreateBlock("", 0, "", "", 0, 0, "", []*Transaction{tx})
  db.Put(block.Hash(), block.MarshalRlp())

  bm := NewBlockManager()
  bm.ProcessBlock( block )
  contract := block.GetContract(tx.Hash())
  fmt.Println(contract)
  fmt.Println("it is", contract.state.Get(string(Encode(0))))
}

