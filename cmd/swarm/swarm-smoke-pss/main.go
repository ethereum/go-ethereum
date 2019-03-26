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
	"os"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/cmd/utils"
	gethmetrics "github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/influxdb"
	swarmmetrics "github.com/ethereum/go-ethereum/swarm/metrics"
	"github.com/ethereum/go-ethereum/swarm/tracing"

	"github.com/ethereum/go-ethereum/log"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit string // Git SHA1 commit hash of the release (set via linker flags)
)

var (
	allhosts          string
	hosts             []string
	inputSeed         int
	wsPort            int
	verbosity         int
	timeout           int
	pssMessageTimeout int
	pssMessageCount   int
	pssMessageSize    int
)

func main() {

	app := cli.NewApp()
	app.Name = "smoke-test-pss"
	app.Usage = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "hosts",
			Value:       "",
			Usage:       "comma-separated list of swarm hosts",
			Destination: &allhosts,
		},
		cli.IntFlag{
			Name:        "ws-port",
			Value:       8546,
			Usage:       "ws port",
			Destination: &wsPort,
		},
		cli.IntFlag{
			Name:        "seed",
			Value:       0,
			Usage:       "input seed in case we need deterministic upload",
			Destination: &inputSeed,
		},
		cli.IntFlag{
			Name:        "verbosity",
			Value:       1,
			Usage:       "verbosity",
			Destination: &verbosity,
		},
		cli.IntFlag{
			Name:        "timeout",
			Value:       120,
			Usage:       "timeout in seconds after which kill the process",
			Destination: &timeout,
		},
		cli.IntFlag{
			Name:        "msgtimeout",
			Value:       1,
			Usage:       "timeout in seconds until a message is received",
			Destination: &pssMessageTimeout,
		},
		cli.IntFlag{
			Name:        "msgcount",
			Value:       10,
			Usage:       "number of pss messages that should be send in the pss smoke test",
			Destination: &pssMessageCount,
		},
		cli.IntFlag{
			Name:        "msgbytes",
			Value:       128,
			Usage:       "size of a randomly generated message. defined in bytes",
			Destination: &pssMessageSize,
		},
	}

	app.Flags = append(app.Flags, []cli.Flag{
		utils.MetricsEnabledFlag,
		swarmmetrics.MetricsInfluxDBEndpointFlag,
		swarmmetrics.MetricsInfluxDBDatabaseFlag,
		swarmmetrics.MetricsInfluxDBUsernameFlag,
		swarmmetrics.MetricsInfluxDBPasswordFlag,
		swarmmetrics.MetricsInfluxDBTagsFlag,
	}...)

	app.Flags = append(app.Flags, tracing.Flags...)

	app.Commands = []cli.Command{
		{
			Name:   "asym",
			Usage:  "send and receive multiple messages across random nodes using asymmetric encryption",
			Action: wrapCliCommand("asym", pssAsymCheck),
		},
		{
			Name:   "sym",
			Usage:  "send and receive multiple messages across random nodes using symmetric encryption",
			Action: wrapCliCommand("sym", pssSymCheck),
		},
		{
			Name:   "raw",
			Usage:  "send and receive multiple raw messages across random nodes",
			Action: wrapCliCommand("raw", pssRawCheck),
		},
		{
			Name:   "all",
			Usage:  "send and receive raw, sym and asym messages across randome nodes",
			Action: wrapCliCommand("all", pssAllCheck),
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		tracing.Setup(ctx)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		return emitMetrics(ctx)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func emitMetrics(ctx *cli.Context) error {
	if gethmetrics.Enabled {
		var (
			endpoint = ctx.GlobalString(swarmmetrics.MetricsInfluxDBEndpointFlag.Name)
			database = ctx.GlobalString(swarmmetrics.MetricsInfluxDBDatabaseFlag.Name)
			username = ctx.GlobalString(swarmmetrics.MetricsInfluxDBUsernameFlag.Name)
			password = ctx.GlobalString(swarmmetrics.MetricsInfluxDBPasswordFlag.Name)
			tags     = ctx.GlobalString(swarmmetrics.MetricsInfluxDBTagsFlag.Name)
		)

		tagsMap := utils.SplitTagsFlag(tags)
		tagsMap["version"] = gitCommit
		tagsMap["msgbytes"] = strconv.Itoa(pssMessageSize)
		tagsMap["msgcount"] = strconv.Itoa(pssMessageCount)

		return influxdb.InfluxDBWithTagsOnce(gethmetrics.DefaultRegistry, endpoint, database, username, password, "swarm-smoke-pss.", tagsMap)
	}

	return nil
}
