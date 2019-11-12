// Copyright 2017 The go-ethereum Authors
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

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/maticnetwork/bor/log"
)

// makeWizard creates and returns a new puppeth wizard.
func makeWizard(network string) *wizard {
	return &wizard{
		network: network,
		conf: config{
			Servers: make(map[string][]byte),
		},
		servers:  make(map[string]*sshClient),
		services: make(map[string][]string),
		in:       bufio.NewReader(os.Stdin),
	}
}

// run displays some useful infos to the user, starting on the journey of
// setting up a new or managing an existing Ethereum private network.
func (w *wizard) run() {
	fmt.Println("+-----------------------------------------------------------+")
	fmt.Println("| Welcome to puppeth, your Ethereum private network manager |")
	fmt.Println("|                                                           |")
	fmt.Println("| This tool lets you create a new Ethereum network down to  |")
	fmt.Println("| the genesis block, bootnodes, miners and ethstats servers |")
	fmt.Println("| without the hassle that it would normally entail.         |")
	fmt.Println("|                                                           |")
	fmt.Println("| Puppeth uses SSH to dial in to remote servers, and builds |")
	fmt.Println("| its network components out of Docker containers using the |")
	fmt.Println("| docker-compose toolset.                                   |")
	fmt.Println("+-----------------------------------------------------------+")
	fmt.Println()

	// Make sure we have a good network name to work with	fmt.Println()
	// Docker accepts hyphens in image names, but doesn't like it for container names
	if w.network == "" {
		fmt.Println("Please specify a network name to administer (no spaces, hyphens or capital letters please)")
		for {
			w.network = w.readString()
			if !strings.Contains(w.network, " ") && !strings.Contains(w.network, "-") && strings.ToLower(w.network) == w.network {
				fmt.Printf("\nSweet, you can set this via --network=%s next time!\n\n", w.network)
				break
			}
			log.Error("I also like to live dangerously, still no spaces, hyphens or capital letters")
		}
	}
	log.Info("Administering Ethereum network", "name", w.network)

	// Load initial configurations and connect to all live servers
	w.conf.path = filepath.Join(os.Getenv("HOME"), ".puppeth", w.network)

	blob, err := ioutil.ReadFile(w.conf.path)
	if err != nil {
		log.Warn("No previous configurations found", "path", w.conf.path)
	} else if err := json.Unmarshal(blob, &w.conf); err != nil {
		log.Crit("Previous configuration corrupted", "path", w.conf.path, "err", err)
	} else {
		// Dial all previously known servers concurrently
		var pend sync.WaitGroup
		for server, pubkey := range w.conf.Servers {
			pend.Add(1)

			go func(server string, pubkey []byte) {
				defer pend.Done()

				log.Info("Dialing previously configured server", "server", server)
				client, err := dial(server, pubkey)
				if err != nil {
					log.Error("Previous server unreachable", "server", server, "err", err)
				}
				w.lock.Lock()
				w.servers[server] = client
				w.lock.Unlock()
			}(server, pubkey)
		}
		pend.Wait()
		w.networkStats()
	}
	// Basics done, loop ad infinitum about what to do
	for {
		fmt.Println()
		fmt.Println("What would you like to do? (default = stats)")
		fmt.Println(" 1. Show network stats")
		if w.conf.Genesis == nil {
			fmt.Println(" 2. Configure new genesis")
		} else {
			fmt.Println(" 2. Manage existing genesis")
		}
		if len(w.servers) == 0 {
			fmt.Println(" 3. Track new remote server")
		} else {
			fmt.Println(" 3. Manage tracked machines")
		}
		if len(w.services) == 0 {
			fmt.Println(" 4. Deploy network components")
		} else {
			fmt.Println(" 4. Manage network components")
		}

		choice := w.read()
		switch {
		case choice == "" || choice == "1":
			w.networkStats()

		case choice == "2":
			if w.conf.Genesis == nil {
				fmt.Println()
				fmt.Println("What would you like to do? (default = create)")
				fmt.Println(" 1. Create new genesis from scratch")
				fmt.Println(" 2. Import already existing genesis")

				choice := w.read()
				switch {
				case choice == "" || choice == "1":
					w.makeGenesis()
				case choice == "2":
					w.importGenesis()
				default:
					log.Error("That's not something I can do")
				}
			} else {
				w.manageGenesis()
			}
		case choice == "3":
			if len(w.servers) == 0 {
				if w.makeServer() != "" {
					w.networkStats()
				}
			} else {
				w.manageServers()
			}
		case choice == "4":
			if len(w.services) == 0 {
				w.deployComponent()
			} else {
				w.manageComponents()
			}
		default:
			log.Error("That's not something I can do")
		}
	}
}
