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
	"fmt"

	"github.com/ubiq/go-ubiq/log"
)

// ensureVirtualHost checks whether a reverse-proxy is running on the specified
// host machine, and if yes requests a virtual host from the user to host a
// specific web service on. If no proxy exists, the method will offer to deploy
// one.
//
// If the user elects not to use a reverse proxy, an empty hostname is returned!
func (w *wizard) ensureVirtualHost(client *sshClient, port int, def string) (string, error) {
	if proxy, _ := checkNginx(client, w.network); proxy != nil {
		// Reverse proxy is running, if ports match, we need a virtual host
		if proxy.port == port {
			fmt.Println()
			fmt.Printf("Shared port, which domain to assign? (default = %s)\n", def)
			return w.readDefaultString(def), nil
		}
	}
	// Reverse proxy is not running, offer to deploy a new one
	fmt.Println()
	fmt.Println("Allow sharing the port with other services (y/n)? (default = yes)")
	if w.readDefaultString("y") == "y" {
		if out, err := deployNginx(client, w.network, port); err != nil {
			log.Error("Failed to deploy reverse-proxy", "err", err)
			if len(out) > 0 {
				fmt.Printf("%s\n", out)
			}
			return "", err
		}
		// Reverse proxy deployed, ask again for the virtual-host
		fmt.Println()
		fmt.Printf("Proxy deployed, which domain to assign? (default = %s)\n", def)
		return w.readDefaultString(def), nil
	}
	// Reverse proxy not requested, deploy as a standalone service
	return "", nil
}
