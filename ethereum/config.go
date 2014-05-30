package main

import (
	"flag"
	"fmt"
	"os"
)

var Identifier string
var StartMining bool
var StartRpc bool
var RpcPort int
var UseUPnP bool
var OutboundPort string
var ShowGenesis bool
var AddPeer string
var MaxPeer int
var GenAddr bool
var UseSeed bool
var ImportKey string
var ExportKey bool
var LogFile string
var NonInteractive bool
var StartJsConsole bool
var InputFile string

func Init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.IntVar(&MaxPeer, "maxpeer", 10, "maximum desired peers")
	flag.IntVar(&RpcPort, "rpcport", 8080, "port to start json-rpc server on")
	flag.BoolVar(&StartRpc, "rpc", false, "start rpc server")
	flag.BoolVar(&StartJsConsole, "js", false, "exp")

	flag.BoolVar(&StartMining, "mine", false, "start dagger mining")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")

	flag.Parse()

	InputFile = flag.Arg(0)
}
