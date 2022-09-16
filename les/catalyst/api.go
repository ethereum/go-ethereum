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

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"github.com/ethereumfair/go-ethereum/les"
	"github.com/ethereumfair/go-ethereum/log"
	"github.com/ethereumfair/go-ethereum/node"
	"github.com/ethereumfair/go-ethereum/rpc"
)

// Register adds catalyst APIs to the light client.
func Register(stack *node.Node, backend *les.LightEthereum) error {
	log.Warn("Catalyst mode enabled", "protocol", "les")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       NewConsensusAPI(backend),
			Authenticated: true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	les *les.LightEthereum
}

// NewConsensusAPI creates a new consensus api for the given backend.
// The underlying blockchain needs to have a valid terminal total difficulty set.
func NewConsensusAPI(les *les.LightEthereum) *ConsensusAPI {
	if les.BlockChain().Config().TerminalTotalDifficulty == nil {
		log.Warn("Catalyst started without valid total difficulty")
	}
	return &ConsensusAPI{les: les}
}
