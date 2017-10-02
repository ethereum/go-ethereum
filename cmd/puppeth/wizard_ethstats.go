// Copyright 2017 The go-burnout Authors
// This file is part of go-burnout.
//
// go-burnout is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-burnout is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-burnout. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"

	"github.com/burnout/go-burnout/log"
)

// deployBrnstats queries the user for various input on deploying an brnstats
// monitoring server, after which it executes it.
func (w *wizard) deployBrnstats() {
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active brnstats configurations from the server
	infos, err := checkBrnstats(client, w.network)
	if err != nil {
		infos = &brnstatsInfos{
			port:   80,
			host:   client.server,
			secret: "",
		}
	}
	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which port should brnstats listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure which virtual-host to deploy brnstats on
	if infos.host, err = w.ensureVirtualHost(client, infos.port, infos.host); err != nil {
		log.Error("Failed to decide on brnstats host", "err", err)
		return
	}
	// Port and proxy settings retrieved, figure out the secret and boot brnstats
	fmt.Println()
	if infos.secret == "" {
		fmt.Printf("What should be the secret password for the API? (must not be empty)\n")
		infos.secret = w.readString()
	} else {
		fmt.Printf("What should be the secret password for the API? (default = %s)\n", infos.secret)
		infos.secret = w.readDefaultString(infos.secret)
	}
	// Gather any blacklists to ban from reporting
	fmt.Println()
	fmt.Printf("Keep existing IP %v blacklist (y/n)? (default = yes)\n", infos.banned)
	if w.readDefaultString("y") != "y" {
		infos.banned = nil

		fmt.Println()
		fmt.Println("Which IP addresses should be blacklisted?")
		for {
			if ip := w.readIPAddress(); ip != nil {
				infos.banned = append(infos.banned, ip.String())
				continue
			}
			break
		}
	}
	// Try to deploy the brnstats server on the host
	trusted := make([]string, 0, len(w.servers))
	for _, client := range w.servers {
		if client != nil {
			trusted = append(trusted, client.address)
		}
	}
	if out, err := deployBrnstats(client, w.network, infos.port, infos.secret, infos.host, trusted, infos.banned); err != nil {
		log.Error("Failed to deploy brnstats container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	w.networkStats(false)
}
