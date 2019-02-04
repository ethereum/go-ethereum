// Copyright 2018 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	addr string
)

func chunkExplorer(c *cli.Context) error {
	log.Debug("generate endpoints...")
	generateEndpoints(scheme, cluster, appName, from, to)
	log.Debug("done.")

	var has bool
	for _, e := range endpoints {
		e = e + ":8546"
		log.Debug("dialing...." + e)
		client, _ := rpc.Dial(e)
		log.Debug("Trying...." + e)
		if err := client.Call(&has, "debugapi_hasChunk", addr); err != nil {
			log.Error("Error requesting hasChunk from endpoint", "endpoint", e, "chunkAddress", addr, "err", err)
		} else {
			if has {
				log.Info("Endpoint "+e+" reports to HAVE chunk", "chunk", addr)
			} else {
				log.Info("Endpoint "+e+" reports to NOT have chunk", "chunk", addr)
			}
		}
	}
	return nil
}
