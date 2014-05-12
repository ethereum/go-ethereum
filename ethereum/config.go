package main

import (
	"flag"
)

var StartConsole bool
var StartMining bool
var StartRpc bool
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

func Init() {
	flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
	flag.BoolVar(&StartMining, "m", false, "start dagger mining")
	flag.BoolVar(&ShowGenesis, "g", false, "prints genesis header and exits")
	//flag.BoolVar(&UseGui, "gui", true, "use the gui")
	flag.BoolVar(&StartRpc, "r", false, "start rpc server")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.BoolVar(&UseSeed, "seed", false, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&OutboundPort, "p", "30303", "listening port")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&DataDir, "dir", ".ethereum", "ethereum data directory")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")
	flag.IntVar(&MaxPeer, "x", 5, "maximum desired peers")

	flag.Parse()
}
