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
	"path"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/vm"
)

var (
	Identifier       string
	KeyRing          string
	DiffTool         bool
	DiffType         string
	KeyStore         string
	StartRpc         bool
	StartWebSockets  bool
	RpcListenAddress string
	RpcPort          int
	WsPort           int
	OutboundPort     string
	ShowGenesis      bool
	AddPeer          string
	MaxPeer          int
	GenAddr          bool
	BootNodes        string
	NodeKey          *ecdsa.PrivateKey
	NAT              nat.Interface
	SecretFile       string
	ExportDir        string
	NonInteractive   bool
	Datadir          string
	LogFile          string
	ConfigFile       string
	DebugFile        string
	LogLevel         int
	LogFormat        string
	Dump             bool
	DumpHash         string
	DumpNumber       int
	VmType           int
	ImportChain      string
	SHH              bool
	Dial             bool
	PrintVersion     bool
)

// flags specific to cli client
var (
	StartMining    bool
	StartJsConsole bool
	InputFile      string
)

var defaultConfigFile = path.Join(ethutil.DefaultDataDir(), "conf.ini")

func Init() {
	// TODO: move common flag processing to cmd/util
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s [options] [filename]:\noptions precedence: default < config file < environment variables < command line\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.IntVar(&VmType, "vm", 0, "Virtual Machine type: 0-1: standard, debug")
	flag.StringVar(&Identifier, "id", "", "Custom client identifier")
	flag.StringVar(&KeyRing, "keyring", "", "identifier for keyring to use")
	flag.StringVar(&KeyStore, "keystore", "db", "system to store keyrings: db|file (db)")

	flag.StringVar(&RpcListenAddress, "rpcaddr", "127.0.0.1", "address for json-rpc server to listen on")
	flag.IntVar(&RpcPort, "rpcport", 8545, "port to start json-rpc server on")
	flag.IntVar(&WsPort, "wsport", 40404, "port to start websocket rpc server on")
	flag.BoolVar(&StartRpc, "rpc", false, "start rpc server")
	flag.BoolVar(&StartWebSockets, "ws", false, "start websocket server")
	flag.BoolVar(&NonInteractive, "y", false, "non-interactive mode (say yes to confirmations)")
	flag.BoolVar(&GenAddr, "genaddr", false, "create a new priv/pub key")
	flag.StringVar(&SecretFile, "import", "", "imports the file given (hex or mnemonic formats)")
	flag.StringVar(&ExportDir, "export", "", "exports the session keyring to files in the directory given")
	flag.StringVar(&LogFile, "logfile", "", "log file (defaults to standard output)")
	flag.StringVar(&Datadir, "datadir", ethutil.DefaultDataDir(), "specifies the datadir to use")
	flag.StringVar(&ConfigFile, "conf", defaultConfigFile, "config file")
	flag.StringVar(&DebugFile, "debug", "", "debug file (no debugging if not set)")
	flag.IntVar(&LogLevel, "loglevel", int(logger.InfoLevel), "loglevel: 0-5: silent,error,warn,info,debug,debug detail)")
	flag.StringVar(&LogFormat, "logformat", "std", "logformat: std,raw)")
	flag.BoolVar(&DiffTool, "difftool", false, "creates output for diff'ing. Sets LogLevel=0")
	flag.StringVar(&DiffType, "diff", "all", "sets the level of diff output [vm, all]. Has no effect if difftool=false")
	flag.BoolVar(&ShowGenesis, "genesis", false, "Dump the genesis block")
	flag.StringVar(&ImportChain, "chain", "", "Imports given chain")

	flag.BoolVar(&Dump, "dump", false, "output the ethereum state in JSON format. Sub args [number, hash]")
	flag.StringVar(&DumpHash, "hash", "", "specify arg in hex")
	flag.IntVar(&DumpNumber, "number", -1, "specify arg in number")

	flag.BoolVar(&StartMining, "mine", false, "start dagger mining")
	flag.BoolVar(&StartJsConsole, "js", false, "launches javascript console")
	flag.BoolVar(&PrintVersion, "version", false, "prints version number")

	// Network stuff
	var (
		nodeKeyFile = flag.String("nodekey", "", "network private key file")
		nodeKeyHex  = flag.String("nodekeyhex", "", "network private key (for testing)")
		natstr      = flag.String("nat", "any", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")
	)
	flag.BoolVar(&Dial, "dial", true, "dial out connections (default on)")
	//flag.BoolVar(&SHH, "shh", true, "run whisper protocol (default on)")
	flag.StringVar(&OutboundPort, "port", "30303", "listening port")

	flag.StringVar(&BootNodes, "bootnodes", "", "space-separated node URLs for discovery bootstrap")
	flag.IntVar(&MaxPeer, "maxpeer", 30, "maximum desired peers")

	flag.Parse()

	// When the javascript console is started log to a file instead
	// of stdout
	if StartJsConsole {
		LogFile = path.Join(Datadir, "ethereum.log")
	}

	var err error
	if NAT, err = nat.Parse(*natstr); err != nil {
		log.Fatalf("-nat: %v", err)
	}
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

	InputFile = flag.Arg(0)
}
