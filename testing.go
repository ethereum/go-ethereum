package main

/*

import (
  _"fmt"
)

// This will eventually go away
var Db *MemDatabase

func Testing() {
  db, _ := NewMemDatabase()
  Db = db

  bm := NewBlockManager()

  tx := NewTransaction("\x00", 20, []string{"PUSH"})
  txData := tx.RlpEncode()
  //fmt.Printf("%q\n", txData)

  copyTx := &Transaction{}
  copyTx.RlpDecode(txData)
  //fmt.Println(tx)
  //fmt.Println(copyTx)

  tx2 := NewTransaction("\x00", 20, []string{"SET 10 6", "LD 10 10"})

  blck := CreateTestBlock([]*Transaction{tx2, tx})

  bm.ProcessBlock( blck )
}
*/
