// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/api"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var (
	verbosityFlag = &cli.IntFlag{
		Name:     "verbosity",
		Usage:    "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value:    3,
		Category: flags.LoggingCategory,
	}
	vmoduleFlag = &cli.StringFlag{
		Name:     "vmodule",
		Usage:    "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value:    "",
		Hidden:   true,
		Category: flags.LoggingCategory,
	}
)

func main() {
	app := flags.NewApp("beacon light syncer tool")
	app.Flags = []cli.Flag{
		utils.BeaconApiFlag,
		utils.BeaconApiHeaderFlag,
		utils.BeaconThresholdFlag,
		utils.BeaconNoFilterFlag,
		utils.BeaconConfigFlag,
		utils.BeaconGenesisRootFlag,
		utils.BeaconGenesisTimeFlag,
		utils.BeaconCheckpointFlag,
		//TODO datadir for optional permanent database
		utils.MainnetFlag,
		utils.SepoliaFlag,
		utils.GoerliFlag,
		utils.BlsyncApiFlag,
		utils.BlsyncJWTSecretFlag,
		verbosityFlag,
		vmoduleFlag,
	}
	app.Action = blsync

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func blsync(ctx *cli.Context) error {
	usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorable(os.Stderr)
	}
	verbosity := log.FromLegacyLevel(ctx.Int(verbosityFlag.Name))
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(output, verbosity, usecolor)))

	if !ctx.IsSet(utils.BeaconApiFlag.Name) {
		utils.Fatalf("Beacon node light client API URL not specified")
	}
	var (
		chainConfig  = makeChainConfig(ctx)
		customHeader = make(map[string]string)
	)

	for _, s := range ctx.StringSlice(utils.BeaconApiHeaderFlag.Name) {
		kv := strings.Split(s, ":")
		if len(kv) != 2 {
			utils.Fatalf("Invalid custom API header entry: %s", s)
		}
		customHeader[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}

	// create data structures
	var (
		db             = memorydb.New()
		threshold      = ctx.Int(utils.BeaconThresholdFlag.Name)
		committeeChain = light.NewCommitteeChain(db, chainConfig.ChainConfig, threshold, !ctx.Bool(utils.BeaconNoFilterFlag.Name))
		headTracker    = light.NewHeadTracker(committeeChain, threshold)
	)
	headSync := sync.NewHeadSync(headTracker, committeeChain)

	// set up scheduler and sync modules
	scheduler := request.NewScheduler(&mclock.System{})

	checkpointInit := sync.NewCheckpointInit(committeeChain, chainConfig.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	beaconBlockSync := newBeaconBlockSync(headTracker)
	scheduler.RegisterTarget(headTracker)
	scheduler.RegisterTarget(committeeChain)
	scheduler.RegisterModule(checkpointInit, "checkpointInit")
	scheduler.RegisterModule(forwardSync, "forwardSync")
	scheduler.RegisterModule(headSync, "headSync")
	scheduler.RegisterModule(beaconBlockSync, "beaconBlockSync")
	go updateEngineApi(makeRPCClient(ctx), beaconBlockSync.headCh)
	// start
	scheduler.Start()
	// register server(s)
	for _, url := range ctx.StringSlice(utils.BeaconApiFlag.Name) {
		beaconApi := api.NewBeaconLightApi(url, customHeader)
		scheduler.RegisterServer(api.NewApiServer(beaconApi))
	}
	// run until stopped
	<-ctx.Done()
	scheduler.Stop()
	close(beaconBlockSync.headCh)
	return nil
}
