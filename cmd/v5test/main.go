// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// bootnode runs a bootstrap node for the Ethereum Discovery Protocol.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

func main() {
	var (
		listenPort = flag.Int("addr", 31000, "beginning of listening port range")
		natdesc    = flag.String("nat", "none", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")
		count      = flag.Int("count", 1, "number of v5 topic discovery test nodes (adds default bootnodes to form a test network)")
		regtopic   = flag.String("reg", "", "topic to register on the network")
		looktopic  = flag.String("search", "", "topic to search on the network")
	)
	flag.Var(glog.GetVerbosity(), "verbosity", "log verbosity (0-9)")
	flag.Var(glog.GetVModule(), "vmodule", "log verbosity pattern")
	glog.SetToStderr(true)
	flag.Parse()

	natm, err := nat.Parse(*natdesc)
	if err != nil {
		utils.Fatalf("-nat: %v", err)
	}

	for i := 0; i < *count; i++ {
		listenAddr := ":" + strconv.Itoa(*listenPort+i)

		nodeKey, err := crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("could not generate key: %v", err)
		}

		if net, err := discv5.ListenUDP(nodeKey, listenAddr, natm, ""); err != nil {
			utils.Fatalf("%v", err)
		} else {
			if err := net.SetFallbackNodes(discv5.BootNodes); err != nil {
				utils.Fatalf("%v", err)
			}
			go func() {
				if *looktopic == "" {
					for i := 0; i < 20; i++ {
						time.Sleep(time.Millisecond * time.Duration(2000+rand.Intn(2001)))
						net.BucketFill()
					}
				}
				switch {
				case *regtopic != "":
					// register topic
					fmt.Println("Starting topic register")
					stop := make(chan struct{})
					net.RegisterTopic(discv5.Topic(*regtopic), stop)
				case *looktopic != "":
					// search topic
					fmt.Println("Starting topic search")
					stop := make(chan struct{})
					found := make(chan string, 100)
					go net.SearchTopic(discv5.Topic(*looktopic), stop, found)
					for s := range found {
						fmt.Println(time.Now(), s)
					}
				default:
					// just keep doing lookups
					for {
						time.Sleep(time.Millisecond * time.Duration(40000+rand.Intn(40001)))
						net.BucketFill()
					}
				}
			}()
		}
		fmt.Printf("Started test node #%d with public key %v\n", i, discv5.PubkeyID(&nodeKey.PublicKey))
	}

	select {}
}
