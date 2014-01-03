package main

import (
  _"fmt"
  "testing"
)


func TestVm(t *testing.T) {
  db, _ := NewMemDatabase()
  Db = db

  ctrct := NewTransaction("", 20, []string{
    "PUSH",
    "1a2f2e",
    "PUSH",
    "hallo",
    "POP",   // POP hallo
    "PUSH",
    "3",
    "LOAD",  // Load hallo back on the stack
    "STOP",
  })
  tx := NewTransaction("1e8a42ea8cce13", 100, []string{})

  block := CreateBlock("", 0, "", "", 0, 0, "", []*Transaction{ctrct, tx})
  db.Put(block.Hash(), block.MarshalRlp())

  bm := NewBlockManager()
  bm.ProcessBlock( block )
}

