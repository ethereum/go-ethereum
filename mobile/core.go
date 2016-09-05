// Copyright 2016 The go-ethereum Authors
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

// Contains all the wrappers from the core package.

package geth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

// MainnetChainConfig is the chain configurations for the main Ethereum network.
var MainnetChainConfig = &ChainConfig{
	HomesteadBlock: params.MainNetHomesteadBlock.Int64(),
	DAOForkBlock:   params.MainNetDAOForkBlock.Int64(),
	DAOForkSupport: true,
}

// MainnetGenesis is the JSON spec to use for the main Ethereum network. It is
// actually empty since that defaults to the hard coded binary genesis block.
var MainnetGenesis = ""

// TestnetChainConfig is the chain configurations for the Ethereum test network.
var TestnetChainConfig = &ChainConfig{
	HomesteadBlock: params.TestNetHomesteadBlock.Int64(),
	DAOForkBlock:   0,
	DAOForkSupport: false,
}

// TestnetGenesis is the JSON spec to use for the Ethereum test network.
var TestnetGenesis = core.TestNetGenesisBlock()

// ChainConfig is the core config which determines the blockchain settings.
type ChainConfig struct {
	HomesteadBlock int64 // Homestead switch block
	DAOForkBlock   int64 // TheDAO hard-fork switch block
	DAOForkSupport bool  // Whether the nodes supports or opposes the DAO hard-fork
}

// NewChainConfig creates a new chain configuration that transitions immediately
// to homestead and has no notion of the DAO fork (ideal for a private network).
func NewChainConfig() *ChainConfig {
	return new(ChainConfig)
}
