package main

import (
  "fmt"
)

func main() {
  InitFees()

  bm := NewBlockManager()

  tx := NewTransaction(0x0, 20, []string{
    "SET 10 6",
    "LD 10 10",
    "LT 10 1 20",
    "SET 255 7",
    "JMPI 20 255",
    "STOP",
    "SET 30 200",
    "LD 30 31",
    "SET 255 22",
    "JMPI 31 255",
    "SET 255 15",
    "JMP 255",
  })
  tx2 := NewTransaction(0x0, 20, []string{"SET 10 6", "LD 10 10"})

  blck := NewBlock([]*Transaction{tx2, tx})

  bm.ProcessBlock( blck )

  fmt.Printf("rlp encoded Tx %q\n", tx.Serialize())
}
