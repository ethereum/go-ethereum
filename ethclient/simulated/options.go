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

package simulated

import (
	"math/big"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
)

// WithBlockGasLimit configures the simulated backend to target a specific gas limit
// when producing blocks.
func WithBlockGasLimit(gaslimit uint64) func(nodeConf *node.Config, ethConf *ethconfig.Config) {
	return func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.Genesis.GasLimit = gaslimit
		ethConf.Miner.GasCeil = gaslimit
	}
}

// WithCallGasLimit configures the simulated backend to cap eth_calls to a specific
// gas limit when running client operations.
func WithCallGasLimit(gaslimit uint64) func(nodeConf *node.Config, ethConf *ethconfig.Config) {
	return func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.RPCGasCap = gaslimit
	}
}

// WithMinerMinTip configures the simulated backend to require a specific minimum
// gas tip for a transaction to be included.
//
// 0 is not possible as a live Geth node would reject that due to DoS protection,
// so the simulated backend will replicate that behavior for consisntency.
func WithMinerMinTip(tip *big.Int) func(nodeConf *node.Config, ethConf *ethconfig.Config) {
	if tip == nil || tip.Cmp(new(big.Int)) <= 0 {
		panic("invalid miner minimum tip")
	}
	return func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		ethConf.Miner.GasPrice = tip
	}
}
