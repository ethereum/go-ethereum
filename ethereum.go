package main

import (
	"flag"
	"fmt"
	"github.com/ethereum/ethutil-go"
	"log"
	"os"
	"os/signal"
	"runtime"
)

const Debug = true

var StartConsole bool
var StartMining bool

func Init() {
	flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
	flag.BoolVar(&StartMining, "m", false, "start dagger mining")

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

	ethutil.InitFees()

	Init()

	if StartConsole {
		console := NewConsole()
		console.Start()
	} else {
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
					res := dagger.Search(ethutil.Big("01001"), ethutil.BigPow(2, 36))
					log.Println("Res dagger", res)
					//server.Broadcast("blockmine", ethutil.Encode(res.String()))
				}
			}()
		}

		server.Start()

		// Wait for shutdown
		server.WaitForShutdown()
	}
}
