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

package lightclient

import (
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Client struct {
	scheduler        *request.Scheduler
	canonicalChain   *canonicalChain
	blocksAndHeaders *blocksAndHeaders
}

func NewClient(config light.ClientConfig, db ethdb.Database) *Client {
	// create data structures
	var (
		committeeChain = light.NewCommitteeChain(db, config)
		headTracker    = light.NewHeadTracker(committeeChain, config.Threshold)
	)
	// set up scheduler and sync modules
	//chainHeadFeed := new(event.Feed)
	scheduler := request.NewScheduler()
	client := &Client{
		scheduler:        scheduler,
		canonicalChain:   newCanonicalChain(),
		blocksAndHeaders: newBlocksAndHeaders(),
	}

	checkpointInit := sync.NewCheckpointInit(committeeChain, config.Checkpoint)
	forwardSync := sync.NewForwardUpdateSync(committeeChain)
	headSync := sync.NewHeadSync(headTracker, committeeChain)
	chainOdr := newChainOdr(headTracker, client.canonicalChain, client.blocksAndHeaders)
	scheduler.RegisterTarget(headTracker)
	scheduler.RegisterTarget(committeeChain)
	scheduler.RegisterTarget(client.canonicalChain)
	scheduler.RegisterTarget(client.blocksAndHeaders)
	scheduler.RegisterModule(checkpointInit, "checkpointInit")
	scheduler.RegisterModule(forwardSync, "forwardSync")
	scheduler.RegisterModule(headSync, "headSync")
	scheduler.RegisterModule(chainOdr, "chainOdr")
	return client
}

func (c *Client) Start() {
	c.scheduler.Start()
}

func (c *Client) Stop() {
	c.scheduler.Stop()
}
