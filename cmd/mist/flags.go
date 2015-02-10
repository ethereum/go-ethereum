/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"

	"bitbucket.org/kardianos/osext"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/vm"
)

var (
	Identifier      string
	KeyRing         string
	KeyStore        string
	PMPGateway      string
	StartRpc        bool
	StartWebSockets bool
	RpcPort         int
	WsPort          int
	NatType         string
	OutboundPort    string
	ShowGenesis     bool
	AddPeer         string
	MaxPeer         int
	GenAddr         bool
	BootNodes       string
	NodeKey         *ecdsa.PrivateKey
	SecretFile      string
	ExportDir       string
	NonInteractive  bool
	Datadir         string
	LogFile         string
	ConfigFile      string
	DebugFile       string
	LogLevel        int
	VmType          int
)

// flags specific to gui client
var AssetPath string

//TODO: If we re-use the one defined in cmd.go the binary osx image crashes. If somebody finds out why we can dry this up.
func defaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "cmd", "mist") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/mist"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
}
func defaultDataDir() string {
	usr, _ := user.Current()
	return path.Join(usr.HomeDir, ".ethereum")
}

var defaultConfigFile = path.Join(defaultDataDir(), "conf.ini")

func Init() {
	// TODO: move common flag processing to cmd/utils
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\noptions precedence: default < config file < environment variables < command line\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&VmType, "vm", 0, "Virtual Machine type: 0-1: standard, debug")
	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&KeyRing, "keyring", "", "identifier for keyring to use")
	flag.StringVar(&KeyStore, "keystore", "db", "system to store keyrings: db|file (db)")
	flag.IntVar(&RpcPort, "rpcport", 8545, "port to start json-rpc server on")
	flag.IntVar(&WsPort, "wsport", 40404, "port to start websocket rpc server on")
	flag.BoolVar(&StartRpc, "rpc", true, "start rpc server")
	flag.BoolVar(&StartWebSockets, "ws", false, "start websocket server")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.StringVar(&NatType, "nat", "", "NAT support (UPNP|PMP) (none)")
	flag.StringVar(&SecretFile, "import", "", "imports the file given (hex or mnemonic formats)")
	flag.StringVar(&ExportDir, "export", "", "exports the session keyring to files in the directory given")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&Datadir, "datadir", defaultDataDir(), "specifies the datadir to use")
	flag.StringVar(&ConfigFile, "conf", defaultConfigFile, "config file")
	flag.StringVar(&DebugFile, "debug", "", "debug file (no debugging if not set)")
	flag.IntVar(&LogLevel, "loglevel", int(logger.InfoLevel), "loglevel: 0-5: silent,error,warn,info,debug,debug detail)")

	flag.StringVar(&AssetPath, "asset_path", defaultAssetPath(), "absolute path to GUI assets directory")

	// Network stuff
	var (
		nodeKeyFile = flag.String("nodekey", "", "network private key file")
		nodeKeyHex  = flag.String("nodekeyhex", "", "network private key (for testing)")
	)
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")
	flag.StringVar(&PMPGateway, "pmp", "", "Gateway IP for NAT-PMP")
	flag.StringVar(&BootNodes, "bootnodes", "", "space-separated node URLs for discovery bootstrap")
	flag.IntVar(&MaxPeer, "maxpeer", 30, "maximum desired peers")

	flag.Parse()

	var err error
	switch {
	case *nodeKeyFile != "" && *nodeKeyHex != "":
		log.Fatal("Options -nodekey and -nodekeyhex are mutually exclusive")
	case *nodeKeyFile != "":
		if NodeKey, err = crypto.LoadECDSA(*nodeKeyFile); err != nil {
			log.Fatalf("-nodekey: %v", err)
		}
	case *nodeKeyHex != "":
		if NodeKey, err = crypto.HexToECDSA(*nodeKeyHex); err != nil {
			log.Fatalf("-nodekeyhex: %v", err)
		}
	}

	if VmType >= int(vm.MaxVmTy) {
		log.Fatal("Invalid VM type ", VmType)
	}
}
