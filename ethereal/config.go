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
var UseSeed bool
var ImportKey string
var ExportKey bool
var DataDir string
var AssetPath string

func Init() {
	flag.BoolVar(&StartConsole, "c", false, "debug and testing console")
	flag.BoolVar(&StartMining, "m", false, "start dagger mining")
	flag.BoolVar(&ShowGenesis, "g", false, "prints genesis header and exits")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&OutboundPort, "p", "30303", "listening port")
	flag.StringVar(&DataDir, "dir", ".ethereal", "ethereum data directory")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")
	flag.IntVar(&MaxPeer, "x", 5, "maximum desired peers")
	flag.StringVar(&AssetPath, "asset_path", "", "absolute path to GUI assets directory")

	flag.Parse()
}
