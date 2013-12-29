package main

import (
  "fmt"
  "os"
  "os/signal"
)

const Debug = false

// Register interrupt handlers so we can stop the server
func RegisterInterupts(s *Server) {
  // Buffered chan of one is enough
  c := make(chan os.Signal, 1)
  // Notify about interrupts for now
  signal.Notify(c, os.Interrupt)
  go func() {
    for sig := range c {
      fmt.Println("Shutting down (%v) ... \n", sig)

      s.Stop()
    }
  }()
}

func main() {
  InitFees()

  bm := NewBlockManager()

  tx := NewTransaction("\x00", 20, []string{
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
  txData := tx.MarshalRlp()

  copyTx := &Transaction{}
  copyTx.UnmarshalRlp(txData)

  tx2 := NewTransaction("\x00", 20, []string{"SET 10 6", "LD 10 10"})

  blck := CreateBlock([]*Transaction{tx2, tx})

  bm.ProcessBlock( blck )

  fmt.Println("GenesisBlock:", GenisisBlock, "hashed", GenisisBlock.Hash())
}
