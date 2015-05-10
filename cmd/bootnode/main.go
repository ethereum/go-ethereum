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

// Command bootnode runs a bootstrap node for the Discovery Protocol.
package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

func main() {
	var (
		listenAddr  = flag.String("addr", ":30301", "listen address")
		genKey      = flag.String("genkey", "", "generate a node key and quit")
		nodeKeyFile = flag.String("nodekey", "", "private key filename")
		nodeKeyHex  = flag.String("nodekeyhex", "", "private key as hex (for testing)")
		natdesc     = flag.String("nat", "none", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")

		nodeKey *ecdsa.PrivateKey
		err     error
	)
	flag.Parse()
	logger.AddLogSystem(logger.NewStdLogSystem(os.Stdout, log.LstdFlags, logger.DebugLevel))

	if *genKey != "" {
		writeKey(*genKey)
		os.Exit(0)
	}

	natm, err := nat.Parse(*natdesc)
	if err != nil {
		log.Fatalf("-nat: %v", err)
	}
	switch {
	case *nodeKeyFile == "" && *nodeKeyHex == "":
		log.Fatal("Use -nodekey or -nodekeyhex to specify a private key")
	case *nodeKeyFile != "" && *nodeKeyHex != "":
		log.Fatal("Options -nodekey and -nodekeyhex are mutually exclusive")
	case *nodeKeyFile != "":
		if nodeKey, err = crypto.LoadECDSA(*nodeKeyFile); err != nil {
			log.Fatalf("-nodekey: %v", err)
		}
	case *nodeKeyHex != "":
		if nodeKey, err = crypto.HexToECDSA(*nodeKeyHex); err != nil {
			log.Fatalf("-nodekeyhex: %v", err)
		}
	}

	if _, err := discover.ListenUDP(nodeKey, *listenAddr, natm, ""); err != nil {
		log.Fatal(err)
	}
	select {}
}

func writeKey(target string) {
	key, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal("could not generate key: %v", err)
	}
	b := crypto.FromECDSA(key)
	if target == "-" {
		fmt.Println(hex.EncodeToString(b))
	} else {
		if err := ioutil.WriteFile(target, b, 0600); err != nil {
			log.Fatal("write error: ", err)
		}
	}
}
