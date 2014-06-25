package main

import (
	"fmt"
	"os"
  "os/user"
  "path"
  "github.com/ethereum/eth-go/ethlog"
	"flag"
	"bitbucket.org/kardianos/osext"
	"path/filepath"
	"runtime"
)

var Identifier string
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
var NonInteractive bool
var Datadir string
var LogFile string
var ConfigFile string
var DebugFile string
var LogLevel int

// flags specific to gui client
var AssetPath string

func defaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/ethereal"
		case "window":
			fallthrough
		default:
			assetPath = "."
		}
	}
	return assetPath
}

func defaultDataDir() string {
  usr, _ := user.Current()
  return path.Join(usr.HomeDir, ".ethereal")
}

var defaultConfigFile = path.Join(defaultDataDir(), "conf.ini")

func Init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\noptions precedence: default < config file < environment variables < command line", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")
	flag.BoolVar(&UseUPnP, "upnp", false, "enable UPnP support")
	flag.IntVar(&MaxPeer, "maxpeer", 10, "maximum desired peers")
	flag.IntVar(&RpcPort, "rpcport", 8080, "port to start json-rpc server on")
	flag.BoolVar(&StartRpc, "rpc", false, "start rpc server")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&UseSeed, "seed", true, "seed peers")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.BoolVar(&ExportKey, "export", false, "export private key")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&ImportKey, "import", "", "imports the given private key (hex)")
	flag.StringVar(&Datadir, "datadir", defaultDataDir(), "specifies the datadir to use")
	flag.StringVar(&ConfigFile, "conf", defaultConfigFile, "config file")
	flag.StringVar(&DebugFile, "debug", "", "debug file (no debugging if not set)")
	flag.IntVar(&LogLevel, "loglevel", int(ethlog.InfoLevel), "loglevel: 0-5: silent,error,warn,info,debug,debug detail)")

	flag.StringVar(&AssetPath, "asset_path", defaultAssetPath(), "absolute path to GUI assets directory")

	flag.Parse()
}
