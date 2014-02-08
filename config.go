package main

import (
	"flag"
)

var StartConsole bool
var StartMining bool
var UseUPnP bool
var OutboundPort string
var ShowGenesis bool
var AddPeer string
var MaxPeer int
var GenAddr bool

func Init() {
	flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
	flag.BoolVar(&StartMining, "m", false, "start dagger mining")
	flag.BoolVar(&ShowGenesis, "g", false, "prints genesis header and exits")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.StringVar(&OutboundPort, "p", "30303", "listening port")
	flag.IntVar(&MaxPeer, "x", 5, "maximum desired peers")

	flag.Parse()
}
