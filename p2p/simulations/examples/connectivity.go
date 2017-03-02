package main

import (
	"runtime"

	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/simulations"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	glog.SetV(6)
	glog.SetToStderr(true)

	c, quitc := simulations.NewSessionController(simulations.DefaultNet)
	simulations.StartRestApiServer("8888", c)
	// wait until server shuts down
	<-quitc

}
