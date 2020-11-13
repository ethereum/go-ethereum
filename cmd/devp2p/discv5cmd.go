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
	"os"
	"time"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/v5test"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"gopkg.in/urfave/cli.v1"
)

var (
	discv5Command = cli.Command{
		Name:  "discv5",
		Usage: "Node Discovery v5 tools",
		Subcommands: []cli.Command{
			discv5PingCommand,
			discv5ResolveCommand,
			discv5CrawlCommand,
			discv5TestCommand,
			discv5ListenCommand,
		},
	}
	discv5PingCommand = cli.Command{
		Name:   "ping",
		Usage:  "Sends ping to a node",
		Action: discv5Ping,
	}
	discv5ResolveCommand = cli.Command{
		Name:   "resolve",
		Usage:  "Finds a node in the DHT",
		Action: discv5Resolve,
		Flags:  []cli.Flag{bootnodesFlag},
	}
	discv5CrawlCommand = cli.Command{
		Name:   "crawl",
		Usage:  "Updates a nodes.json file with random nodes found in the DHT",
		Action: discv5Crawl,
		Flags:  []cli.Flag{bootnodesFlag, crawlTimeoutFlag},
	}
	discv5TestCommand = cli.Command{
		Name:   "test",
		Usage:  "Runs protocol tests against a node",
		Action: discv5Test,
		Flags:  []cli.Flag{testPatternFlag, testListen1Flag, testListen2Flag},
	}
	discv5ListenCommand = cli.Command{
		Name:   "listen",
		Usage:  "Runs a node",
		Action: discv5Listen,
		Flags: []cli.Flag{
			bootnodesFlag,
			nodekeyFlag,
			nodedbFlag,
			listenAddrFlag,
		},
	}
)

func discv5Ping(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV5(ctx)
	defer disc.Close()

	fmt.Println(disc.Ping(n))
	return nil
}

func discv5Resolve(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	disc := startV5(ctx)
	defer disc.Close()

	fmt.Println(disc.Resolve(n))
	return nil
}

func discv5Crawl(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}
	nodesFile := ctx.Args().First()
	var inputSet nodeSet
	if common.FileExist(nodesFile) {
		inputSet = loadNodesJSON(nodesFile)
	}

	disc := startV5(ctx)
	defer disc.Close()
	c := newCrawler(inputSet, disc, disc.RandomNodes())
	c.revalidateInterval = 10 * time.Minute
	output := c.run(ctx.Duration(crawlTimeoutFlag.Name))
	writeNodesJSON(nodesFile, output)
	return nil
}

func discv5Test(ctx *cli.Context) error {
	// Disable logging unless explicitly enabled.
	if !ctx.GlobalIsSet("verbosity") && !ctx.GlobalIsSet("vmodule") {
		log.Root().SetHandler(log.DiscardHandler())
	}

	// Filter and run test cases.
	suite := &v5test.Suite{
		Dest:    getNodeArg(ctx),
		Listen1: ctx.String(testListen1Flag.Name),
		Listen2: ctx.String(testListen2Flag.Name),
	}
	tests := suite.AllTests()
	if ctx.IsSet(testPatternFlag.Name) {
		tests = utesting.MatchTests(tests, ctx.String(testPatternFlag.Name))
	}
	results := utesting.RunTests(tests, os.Stdout)
	if fails := utesting.CountFailures(results); fails > 0 {
		return fmt.Errorf("%v/%v tests passed.", len(tests)-fails, len(tests))
	}
	fmt.Printf("%v/%v passed\n", len(tests), len(tests))
	return nil
}

func discv5Listen(ctx *cli.Context) error {
	disc := startV5(ctx)
	defer disc.Close()

	fmt.Println(disc.Self())
	select {}
}

// startV5 starts an ephemeral discovery v5 node.
func startV5(ctx *cli.Context) *discover.UDPv5 {
	ln, config := makeDiscoveryConfig(ctx)
	socket := listen(ln, ctx.String(listenAddrFlag.Name))
	disc, err := discover.ListenV5(socket, ln, config)
	if err != nil {
		exit(err)
	}
	return disc
}
