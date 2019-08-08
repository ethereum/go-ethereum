// Copyright 2019 The go-ethereum Authors
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
	"net"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/urfave/cli.v1"
)

var (
	discv4Command = cli.Command{
		Name:  "discv4",
		Usage: "Node Discovery v4 tools",
		Subcommands: []cli.Command{
			discv4PingCommand,
			discv4RequestRecordCommand,
			discv4ResolveCommand,
		},
	}
	discv4PingCommand = cli.Command{
		Name:   "ping",
		Usage:  "Sends ping to a node",
		Action: discv4Ping,
	}
	discv4RequestRecordCommand = cli.Command{
		Name:   "requestenr",
		Usage:  "Requests a node record using EIP-868 enrRequest",
		Action: discv4RequestRecord,
	}
	discv4ResolveCommand = cli.Command{
		Name:   "resolve",
		Usage:  "Finds a node in the DHT",
		Action: discv4Resolve,
		Flags:  []cli.Flag{bootnodesFlag},
	}
)

var bootnodesFlag = cli.StringFlag{
	Name:  "bootnodes",
	Usage: "Comma separated nodes used for bootstrapping",
}

func discv4Ping(ctx *cli.Context) error {
	n, disc, err := getNodeArgAndStartV4(ctx)
	if err != nil {
		return err
	}
	defer disc.Close()

	start := time.Now()
	if err := disc.Ping(n); err != nil {
		return fmt.Errorf("node didn't respond: %v", err)
	}
	fmt.Printf("node responded to ping (RTT %v).\n", time.Since(start))
	return nil
}

func discv4RequestRecord(ctx *cli.Context) error {
	n, disc, err := getNodeArgAndStartV4(ctx)
	if err != nil {
		return err
	}
	defer disc.Close()

	respN, err := disc.RequestENR(n)
	if err != nil {
		return fmt.Errorf("can't retrieve record: %v", err)
	}
	fmt.Println(respN.String())
	return nil
}

func discv4Resolve(ctx *cli.Context) error {
	n, disc, err := getNodeArgAndStartV4(ctx)
	if err != nil {
		return err
	}
	defer disc.Close()

	fmt.Println(disc.Resolve(n).String())
	return nil
}

func getNodeArgAndStartV4(ctx *cli.Context) (*enode.Node, *discover.UDPv4, error) {
	if ctx.NArg() != 1 {
		return nil, nil, fmt.Errorf("missing node as command-line argument")
	}
	n, err := parseNode(ctx.Args()[0])
	if err != nil {
		return nil, nil, err
	}
	var bootnodes []*enode.Node
	if commandHasFlag(ctx, bootnodesFlag) {
		bootnodes, err = parseBootnodes(ctx)
		if err != nil {
			return nil, nil, err
		}
	}
	disc, err := startV4(bootnodes)
	return n, disc, err
}

func parseBootnodes(ctx *cli.Context) ([]*enode.Node, error) {
	s := params.RinkebyBootnodes
	if ctx.IsSet(bootnodesFlag.Name) {
		s = strings.Split(ctx.String(bootnodesFlag.Name), ",")
	}
	nodes := make([]*enode.Node, len(s))
	var err error
	for i, record := range s {
		nodes[i], err = parseNode(record)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap node: %v", err)
		}
	}
	return nodes, nil
}

// commandHasFlag returns true if the current command supports the given flag.
func commandHasFlag(ctx *cli.Context, flag cli.Flag) bool {
	flags := ctx.FlagNames()
	sort.Strings(flags)
	i := sort.SearchStrings(flags, flag.GetName())
	return i != len(flags) && flags[i] == flag.GetName()
}

// startV4 starts an ephemeral discovery V4 node.
func startV4(bootnodes []*enode.Node) (*discover.UDPv4, error) {
	var cfg discover.Config
	cfg.Bootnodes = bootnodes
	cfg.PrivateKey, _ = crypto.GenerateKey()
	db, _ := enode.OpenDB("")
	ln := enode.NewLocalNode(db, cfg.PrivateKey)

	socket, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IP{0, 0, 0, 0}})
	if err != nil {
		return nil, err
	}
	addr := socket.LocalAddr().(*net.UDPAddr)
	ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	ln.SetFallbackUDP(addr.Port)
	return discover.ListenUDP(socket, ln, cfg)
}
