// Copyright 2024 The go-ethereum Authors
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

package blsync

import (
	"strings"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/api"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type Client struct {
	urls         []string
	customHeader map[string]string
	chainConfig  *lightClientConfig
	scheduler    *request.Scheduler
	blockSync    *beaconBlockSync
	engineRPC    *rpc.Client

	chainHeadSub event.Subscription
	engineClient *engineClient
}

func NewClient(ctx *cli.Context) *Client {
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
	scheduler := request.NewScheduler()
	checkpointInit := sync.NewCheckpointInit(committeeChain, chainConfig.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	beaconBlockSync := newBeaconBlockSync(headTracker)
	scheduler.RegisterTarget(headTracker)
	scheduler.RegisterTarget(committeeChain)
	scheduler.RegisterModule(checkpointInit, "checkpointInit")
	scheduler.RegisterModule(forwardSync, "forwardSync")
	scheduler.RegisterModule(headSync, "headSync")
	scheduler.RegisterModule(beaconBlockSync, "beaconBlockSync")

	return &Client{
		scheduler:    scheduler,
		urls:         ctx.StringSlice(utils.BeaconApiFlag.Name),
		customHeader: customHeader,
		chainConfig:  &chainConfig,
		blockSync:    beaconBlockSync,
	}
}

func (c *Client) SetEngineRPC(engine *rpc.Client) {
	c.engineRPC = engine
}

func (c *Client) Start() error {
	headCh := make(chan types.ChainHeadEvent, 16)
	c.chainHeadSub = c.blockSync.SubscribeChainHead(headCh)
	c.engineClient = startEngineClient(c.chainConfig, c.engineRPC, headCh)

	c.scheduler.Start()
	for _, url := range c.urls {
		beaconApi := api.NewBeaconLightApi(url, c.customHeader)
		c.scheduler.RegisterServer(request.NewServer(api.NewApiServer(beaconApi), &mclock.System{}))
	}
	return nil
}

func (c *Client) Stop() error {
	c.engineClient.stop()
	c.chainHeadSub.Unsubscribe()
	c.scheduler.Stop()
	return nil
}
