package main

import (
  "fmt"
  "os"
  "os/signal"
  "flag"
)

const Debug = true

var StartDBQueryInterface bool
func Init() {
  flag.BoolVar(&StartDBQueryInterface, "db", false, "start db query interface")

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
      fmt.Println("Shutting down (%v) ... \n", sig)

      s.Stop()
    }
  }()
}

func main() {
  InitFees()

  Init()

  if StartDBQueryInterface {
    dbInterface := NewDBInterface()
    dbInterface.Start()
  } else {
    Testing()
  }
}
