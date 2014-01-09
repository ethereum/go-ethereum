package main

import (
  "fmt"
  "os"
  "os/signal"
  "flag"
  "runtime"
  _"math/big"
)

const Debug = true

var StartDBQueryInterface bool
var StartMining bool
func Init() {
  flag.BoolVar(&StartDBQueryInterface, "c", false, "console interface")
  flag.BoolVar(&StartMining, "mine", false, "start dagger mining")

  flag.Parse()
}

// Register interrupt handlers so we can stop the server
func RegisterInterupts(s *Server) {
  // Buffered chan of one is enough
  c := make(chan os.Signal, 1)
  // Notify about interrupts for now
  signal.Notify(c, os.Interrupt)
  go func() {
    for sig := range c {
      fmt.Printf("Shutting down (%v) ... \n", sig)

      s.Stop()
    }
  }()
}

func main() {
  runtime.GOMAXPROCS(runtime.NumCPU())

  InitFees()

  Init()

  if StartDBQueryInterface {
    dbInterface := NewDBInterface()
    dbInterface.Start()
  } else if StartMining {
    dagger := &Dagger{}
    dagger.Search(BigPow(2, 36))
  } else {
    fmt.Println("[DBUG]: Starting Ethereum")
    server, err := NewServer()

    if err != nil {
      fmt.Println("error NewServer:", err)
      return
    }

    RegisterInterupts(server)

    server.Start()

    // Wait for shutdown
    server.WaitForShutdown()
  }
}
