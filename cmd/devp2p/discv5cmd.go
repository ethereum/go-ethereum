// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/v5test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/urfave/cli/v2"
)

var (
	discv5Command = &cli.Command{
		Name:  "discv5",
		Usage: "Node Discovery v5 tools",
		Subcommands: []*cli.Command{
			discv5PingCommand,
			discv5ResolveCommand,
			discv5CrawlCommand,
			discv5TestCommand,
			discv5ListenCommand,
		},
	}
	discv5PingCommand = &cli.Command{
		Name:   "ping",
		Usage:  "Sends ping to a node",
		Action: discv5Ping,
		Flags:  discoveryNodeFlags,
	}
	discv5ResolveCommand = &cli.Command{
		Name:   "resolve",
		Usage:  "Finds a node in the DHT",
		Action: discv5Resolve,
		Flags:  discoveryNodeFlags,
	}
	discv5CrawlCommand = &cli.Command{
		Name:   "crawl",
		Usage:  "Updates a nodes.json file with random nodes found in the DHT",
		Action: discv5Crawl,
		Flags: slices.Concat(discoveryNodeFlags, []cli.Flag{
			crawlTimeoutFlag,
			crawlParallelismFlag,
		}),
	}
	discv5TestCommand = &cli.Command{
		Name:   "test",
		Usage:  "Runs protocol tests against a node",
		Action: discv5Test,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
			testListen1Flag,
			testListen2Flag,
			testExpectedIPFlag,
			testExpectedIP6Flag,
		},
	}
	discv5ListenCommand = &cli.Command{
		Name:   "listen",
		Usage:  "Runs a node",
		Action: discv5Listen,
		Flags:  discoveryNodeFlags,
	}
)

func discv5Ping(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc, _ := startV5(ctx)
	defer disc.Close()

	_, err := disc.Ping(n)
	fmt.Println(err)
	return nil
}

func discv5Resolve(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc, _ := startV5(ctx)
	defer disc.Close()

	fmt.Println(disc.Resolve(n))
	return nil
}

func discv5Crawl(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return errors.New("need nodes file as argument")
	}
	nodesFile := ctx.Args().First()
	inputSet := make(nodeSet)
	if common.FileExist(nodesFile) {
		inputSet = loadNodesJSON(nodesFile)
	}

	disc, config := startV5(ctx)
	defer disc.Close()

	c, err := newCrawler(inputSet, config.Bootnodes, disc, disc.RandomNodes())
	if err != nil {
		return err
	}
	c.revalidateInterval = 10 * time.Minute
	output := c.run(ctx.Duration(crawlTimeoutFlag.Name), ctx.Int(crawlParallelismFlag.Name))
	writeNodesJSON(nodesFile, output)
	return nil
}

// discv5Test runs the protocol test suite.
func discv5Test(ctx *cli.Context) error {
	expectIP, err := parseExpectedIP(ctx.String(testExpectedIPFlag.Name), false)
	if err != nil {
		return err
	}
	expectIP6, err := parseExpectedIP(ctx.String(testExpectedIP6Flag.Name), true)
	if err != nil {
		return err
	}
	suite := &v5test.Suite{
		Dest:      getNodeArg(ctx),
		Listen1:   ctx.String(testListen1Flag.Name),
		Listen2:   ctx.String(testListen2Flag.Name),
		ExpectIP:  expectIP,
		ExpectIP6: expectIP6,
	}
	tests := suite.AllTests()
	if (expectIP.IsValid() || expectIP6.IsValid()) && ctx.IsSet(testPatternFlag.Name) {
		matches := utesting.MatchTests(tests, ctx.String(testPatternFlag.Name))
		if !slices.ContainsFunc(matches, func(test utesting.Test) bool {
			return test.Name == "FindnodeZeroDistance"
		}) {
			return errors.New("--expect-ip and --expect-ip6 require running FindnodeZeroDistance")
		}
	}
	return runTests(ctx, tests)
}

func parseExpectedIP(raw string, ipv6 bool) (netip.Addr, error) {
	if raw == "" {
		return netip.Addr{}, nil
	}
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		return netip.Addr{}, err
	}
	if ipv6 {
		if !addr.Is6() || addr.Is4In6() {
			return netip.Addr{}, fmt.Errorf("invalid expected IPv6 address %q", raw)
		}
	} else if !addr.Is4() {
		return netip.Addr{}, fmt.Errorf("invalid expected IPv4 address %q", raw)
	}
	return addr, nil
}

func discv5Listen(ctx *cli.Context) error {
	disc, _ := startV5(ctx)
	defer disc.Close()

	fmt.Println(disc.Self())
	select {}
}

// startV5 starts an ephemeral discovery v5 node.
func startV5(ctx *cli.Context) (*discover.UDPv5, discover.Config) {
	ln, config := makeDiscoveryConfig(ctx)
	socket := listen(ctx, ln)
	disc, err := discover.ListenV5(socket, ln, config)
	if err != nil {
		exit(err)
	}
	return disc, config
}
