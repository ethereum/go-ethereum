package main

import (
  _"fmt"
  "testing"
)


func TestVm(t *testing.T) {
  InitFees()

  db, _ := NewMemDatabase()
  Db = db

  ctrct := NewTransaction("", 200000000, []string{
    "PUSH", "1a2f2e",
    "PUSH", "hallo",
    "POP",   // POP hallo
    "PUSH", "3",
    "LOAD",  // Load hallo back on the stack

    "PUSH", "1",
    "PUSH", "2",
    "ADD",

    "PUSH", "2",
    "PUSH", "1",
    "SUB",

    "PUSH", "100000000000000000000000",
    "PUSH", "10000000000000",
    "SDIV",

    "PUSH", "105",
    "PUSH", "200",
    "MOD",

    "PUSH", "100000000000000000000000",
    "PUSH", "10000000000000",
    "SMOD",

    "PUSH", "5",
    "PUSH", "10",
    "LT",

    "PUSH", "5",
    "PUSH", "5",
    "LE",

    "PUSH", "50",
    "PUSH", "5",
    "GT",

    "PUSH", "5",
    "PUSH", "5",
    "GE",

    "PUSH", "10",
    "PUSH", "10",
    "NOT",

    "MYADDRESS",
    "TXSENDER",

    "STOP",
  })
  tx := NewTransaction("1e8a42ea8cce13", 100, []string{})

  block := CreateBlock("", 0, "", "c014ba53", 0, 0, "", []*Transaction{ctrct, tx})
  db.Put(block.Hash(), block.MarshalRlp())

  bm := NewBlockManager()
  bm.ProcessBlock( block )
}

