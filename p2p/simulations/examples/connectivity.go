package main

import (
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	c, quitc := simulations.NewSessionController(simulations.DefaultNet)
	simulations.StartRestApiServer("8888", c)
	// wait until server shuts down
	<-quitc

}
