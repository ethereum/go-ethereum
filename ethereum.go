package main

import (
  "fmt"
  "os"
  "os/signal"
  "flag"
  "runtime"
  "log"
  _"math/big"
)

const Debug = true

var StartConsole bool
var StartMining bool
func Init() {
  flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
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

  if StartConsole {
    console := NewConsole()
    console.Start()
  } else{
    log.Println("Starting Ethereum")
    server, err := NewServer()

    if err != nil {
      log.Println(err)
      return
    }

    RegisterInterupts(server)

    if StartMining {
      log.Println("Mining started")
      dagger := &Dagger{}

      go func() {
        for {
          res := dagger.Search(Big("0"), BigPow(2, 36))
          server.Broadcast("block", Encode(res.String()))
        }
      }()
    }

    server.Start()

    err = server.ConnectToPeer("localhost:12345")
    if err != nil {
      log.Println(err)
      server.Stop()
      return
    }


    // Wait for shutdown
    server.WaitForShutdown()
  }
}
