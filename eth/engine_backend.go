// Copyright 2026 The go-ethereum Authors
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

package eth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
)

// NewEngineBackend wraps an existing chain and its database in the parts of a
// backend that the engine API needs to import payloads and update the
// forkchoice: the chain, the database and an idle full-sync downloader.
// Nothing else is created: no node, networking, transaction pool, miner, blob
// cache or log index, all of which stay nil. The engine API reaches them only
// to build payloads (forkchoice updates with payload attributes), to update
// custody columns, to serve blobs and to exchange capabilities, so callers
// must not use those, and must not call Stop. It is meant for short-lived
// engine API callers such as evm enginetest, which release it by terminating
// the downloader and stopping the chain.
func NewEngineBackend(chain *core.BlockChain, db ethdb.Database) *Ethereum {
	config := ethconfig.Defaults
	eth := &Ethereum{
		config:     &config,
		chainDb:    db,
		blockchain: chain,
	}
	eth.handler = &handler{
		downloader: downloader.New(eth.chainDb, ethconfig.FullSync, chain, func(string) {}, func() {}, false),
	}
	return eth
}
