// Copyright 2026 The go-ethereum Authors
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
	"slices"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/urfave/cli/v2"
)

var (
	discoveryCommand = &cli.Command{
		Name:  "discovery",
		Usage: "Node Discovery tools",
		Subcommands: []*cli.Command{
			discoveryListenCommand,
		},
	}
	discoveryListenCommand = &cli.Command{
		Name:   "listen",
		Usage:  "Runs a discovery node speaking all supported protocol versions on one UDP port",
		Action: discoveryListen,
		Flags: slices.Concat(discoveryNodeFlags, []cli.Flag{
			httpAddrFlag,
		}),
	}
)

func discoveryListen(ctx *cli.Context) error {
	ln, config := makeDiscoveryConfig(ctx)
	socket := listen(ctx, ln)

	// v4 is the primary listener on the real socket and forwards packets it
	// can't parse to v5 via the unhandled channel.
	unhandled := make(chan discover.ReadPacket, 100)
	v4cfg := config
	v4cfg.Unhandled = unhandled
	v4, err := discover.ListenV4(socket, ln, v4cfg)
	if err != nil {
		exit(err)
	}
	v5, err := discover.ListenV5(&discover.SharedUDPConn{UDPConn: socket, Unhandled: unhandled}, ln, config)
	if err != nil {
		exit(err)
	}
	// Close v4 before v5: when sharing the socket, v5's read loop only unblocks
	// once v4 closes the underlying socket and the unhandled channel.
	defer func() {
		v4.Close()
		v5.Close()
	}()

	// v4 and v5 share the same local node, so this is the single record both announce.
	fmt.Println(ln.Node())

	return runRPCServer(ctx.String(httpAddrFlag.Name), map[string]any{
		"discv4": &discv4API{v4},
		"discv5": &discv5API{v5},
	})
}
