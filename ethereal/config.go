package main

import (
	"flag"
)

var Identifier string

//var StartMining bool
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
var AssetPath string

var Datadir string

func Init() {
	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.IntVar(&MaxPeer, "maxpeer", 10, "maximum desired peers")
	flag.IntVar(&RpcPort, "rpcport", 8080, "port to start json-rpc server on")
	flag.BoolVar(&StartRpc, "rpc", false, "start rpc server")
	flag.StringVar(&AssetPath, "asset_path", "", "absolute path to GUI assets directory")

	flag.BoolVar(&ShowGenesis, "genesis", false, "prints genesis header and exits")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")

	flag.StringVar(&Datadir, "datadir", ".ethereal", "specifies the datadir to use. Takes precedence over config file.")

	flag.Parse()
}
