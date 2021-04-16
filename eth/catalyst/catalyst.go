// Copyright 2020 The go-ethereum Authors
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
	"errors"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// Register adds catalyst APIs to the node.
func Register(stack *node.Node, backend *eth.Ethereum) error {
	if backend.BlockChain().Config().CatalystBlock == nil {
		return errors.New("can't enable catalyst service without catalyst fork block in chain config")
	}

	log.Warn("Catalyst mode enabled")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   newConsensusAPI(backend),
			Public:    true,
		},
	})
	return nil
}
