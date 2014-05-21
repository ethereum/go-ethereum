package main

import (
	"flag"
	"fmt"
	"os"
)

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
var DataDir string
var NonInteractive bool
var StartJsConsole bool
var InputFile string

func Init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVar(&StartMining, "m", false, "start dagger mining")
	flag.BoolVar(&ShowGenesis, "g", false, "prints genesis header and exits")
	flag.BoolVar(&StartRpc, "r", false, "start rpc server")
	flag.IntVar(&RpcPort, "rpcport", 8080, "port to start json-rpc server on")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&OutboundPort, "p", "30303", "listening port")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&DataDir, "dir", ".ethereum", "ethereum data directory")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")
	flag.IntVar(&MaxPeer, "x", 10, "maximum desired peers")
	flag.BoolVar(&StartJsConsole, "js", false, "exp")
	//flag.StringVar(&InputFile, "e", "", "Run javascript file")

	flag.Parse()

	InputFile = flag.Arg(0)
}
