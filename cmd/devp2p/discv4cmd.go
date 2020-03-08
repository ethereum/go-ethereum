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
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
			discv4ResolveJSONCommand,
			discv4CrawlCommand,
		},
	}
	discv4PingCommand = cli.Command{
		Name:      "ping",
		Usage:     "Sends ping to a node",
		Action:    discv4Ping,
		ArgsUsage: "<node>",
	}
	discv4RequestRecordCommand = cli.Command{
		Name:      "requestenr",
		Usage:     "Requests a node record using EIP-868 enrRequest",
		Action:    discv4RequestRecord,
		ArgsUsage: "<node>",
	}
	discv4ResolveCommand = cli.Command{
		Name:      "resolve",
		Usage:     "Finds a node in the DHT",
		Action:    discv4Resolve,
		ArgsUsage: "<node>",
		Flags:     []cli.Flag{bootnodesFlag},
	}
	discv4ResolveJSONCommand = cli.Command{
		Name:      "resolve-json",
		Usage:     "Re-resolves nodes in a nodes.json file",
		Action:    discv4ResolveJSON,
		Flags:     []cli.Flag{bootnodesFlag},
		ArgsUsage: "<nodes.json file>",
	}
	discv4CrawlCommand = cli.Command{
		Name:   "crawl",
		Usage:  "Updates a nodes.json file with random nodes found in the DHT",
		Action: discv4Crawl,
		Flags:  []cli.Flag{bootnodesFlag, crawlTimeoutFlag},
	}
)

var (
	bootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated nodes used for bootstrapping",
	}
	nodekeyFlag = cli.StringFlag{
		Name:  "nodekey",
		Usage: "Hex-encoded node key",
	}
	nodedbFlag = cli.StringFlag{
		Name:  "nodedb",
		Usage: "Nodes database location",
	}
	listenAddrFlag = cli.StringFlag{
		Name:  "addr",
		Usage: "Listening address",
	}
	crawlTimeoutFlag = cli.DurationFlag{
		Name:  "timeout",
		Usage: "Time limit for the crawl.",
		Value: 30 * time.Minute,
	}
)

func discv4Ping(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV4(ctx)
	defer disc.Close()

	start := time.Now()
	if err := disc.Ping(n); err != nil {
		return fmt.Errorf("node didn't respond: %v", err)
	}
	fmt.Printf("node responded to ping (RTT %v).\n", time.Since(start))
	return nil
}

func discv4RequestRecord(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV4(ctx)
	defer disc.Close()

	respN, err := disc.RequestENR(n)
	if err != nil {
		return fmt.Errorf("can't retrieve record: %v", err)
	}
	fmt.Println(respN.String())
	return nil
}

func discv4Resolve(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV4(ctx)
	defer disc.Close()

	fmt.Println(disc.Resolve(n).String())
	return nil
}

func discv4ResolveJSON(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}
	nodesFile := ctx.Args().Get(0)
	inputSet := make(nodeSet)
	if common.FileExist(nodesFile) {
		inputSet = loadNodesJSON(nodesFile)
	}

	// Add extra nodes from command line arguments.
	var nodeargs []*enode.Node
	for i := 1; i < ctx.NArg(); i++ {
		n, err := parseNode(ctx.Args().Get(i))
		if err != nil {
			exit(err)
		}
		nodeargs = append(nodeargs, n)
	}

	// Run the crawler.
	disc := startV4(ctx)
	defer disc.Close()
	c := newCrawler(inputSet, disc, enode.IterNodes(nodeargs))
	c.revalidateInterval = 0
	output := c.run(0)
	writeNodesJSON(nodesFile, output)
	return nil
}

func discv4Crawl(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}
	nodesFile := ctx.Args().First()
	var inputSet nodeSet
	if common.FileExist(nodesFile) {
		inputSet = loadNodesJSON(nodesFile)
	}

	disc := startV4(ctx)
	defer disc.Close()
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	output := c.run(ctx.Duration(crawlTimeoutFlag.Name))
	writeNodesJSON(nodesFile, output)
	return nil
}

// startV4 starts an ephemeral discovery V4 node.
func startV4(ctx *cli.Context) *discover.UDPv4 {
	ln, config := makeDiscoveryConfig(ctx)
	socket := listen(ln, ctx.String(listenAddrFlag.Name))
	disc, err := discover.ListenV4(socket, ln, config)
	if err != nil {
		exit(err)
	}
	return disc
}

func makeDiscoveryConfig(ctx *cli.Context) (*enode.LocalNode, discover.Config) {
	var cfg discover.Config

	if ctx.IsSet(nodekeyFlag.Name) {
		key, err := crypto.HexToECDSA(ctx.String(nodekeyFlag.Name))
		if err != nil {
			exit(fmt.Errorf("-%s: %v", nodekeyFlag.Name, err))
		}
		cfg.PrivateKey = key
	} else {
		cfg.PrivateKey, _ = crypto.GenerateKey()
	}

	if commandHasFlag(ctx, bootnodesFlag) {
		bn, err := parseBootnodes(ctx)
		if err != nil {
			exit(err)
		}
		cfg.Bootnodes = bn
	}

	dbpath := ctx.String(nodedbFlag.Name)
	db, err := enode.OpenDB(dbpath)
	if err != nil {
		exit(err)
	}
	ln := enode.NewLocalNode(db, cfg.PrivateKey)
	return ln, cfg
}

func listen(ln *enode.LocalNode, addr string) *net.UDPConn {
	if addr == "" {
		addr = "0.0.0.0:0"
	}
	socket, err := net.ListenPacket("udp4", addr)
	if err != nil {
		exit(err)
	}
	usocket := socket.(*net.UDPConn)
	uaddr := socket.LocalAddr().(*net.UDPAddr)
	ln.SetFallbackIP(net.IP{127, 0, 0, 1})
	ln.SetFallbackUDP(uaddr.Port)
	return usocket
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
