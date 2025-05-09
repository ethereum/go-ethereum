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

package blsync

import (
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/urfave/cli/v2"
)

// lightClientConfig contains beacon light client configuration
type lightClientConfig struct {
	*types.ChainConfig
	Checkpoint common.Hash
}

var (
	MainnetConfig = lightClientConfig{
		ChainConfig: (&types.ChainConfig{
			GenesisValidatorsRoot: common.HexToHash("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95"),
			GenesisTime:           1606824023,
		}).
			AddFork("GENESIS", 0, []byte{0, 0, 0, 0}).
			AddFork("ALTAIR", 74240, []byte{1, 0, 0, 0}).
			AddFork("BELLATRIX", 144896, []byte{2, 0, 0, 0}).
			AddFork("CAPELLA", 194048, []byte{3, 0, 0, 0}).
			AddFork("DENEB", 269568, []byte{4, 0, 0, 0}),
		Checkpoint: common.HexToHash("0x388be41594ec7d6a6894f18c73f3469f07e2c19a803de4755d335817ed8e2e5a"),
	}

	SepoliaConfig = lightClientConfig{
		ChainConfig: (&types.ChainConfig{
			GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
			GenesisTime:           1655733600,
		}).
			AddFork("GENESIS", 0, []byte{144, 0, 0, 105}).
			AddFork("ALTAIR", 50, []byte{144, 0, 0, 112}).
			AddFork("BELLATRIX", 100, []byte{144, 0, 0, 113}).
			AddFork("CAPELLA", 56832, []byte{144, 0, 0, 114}).
			AddFork("DENEB", 132608, []byte{144, 0, 0, 115}),
		Checkpoint: common.HexToHash("0x1005a6d9175e96bfbce4d35b80f468e9bff0b674e1e861d16e09e10005a58e81"),
	}
)

func makeChainConfig(ctx *cli.Context) lightClientConfig {
	var config lightClientConfig
	customConfig := ctx.IsSet(utils.BeaconConfigFlag.Name)
	utils.CheckExclusive(ctx, utils.MainnetFlag, utils.SepoliaFlag, utils.BeaconConfigFlag)
	switch {
	case ctx.Bool(utils.MainnetFlag.Name):
		config = MainnetConfig
	case ctx.Bool(utils.SepoliaFlag.Name):
		config = SepoliaConfig
	default:
		if !customConfig {
			config = MainnetConfig
		}
	}
	// Genesis root and time should always be specified together with custom chain config
	if customConfig {
		if !ctx.IsSet(utils.BeaconGenesisRootFlag.Name) {
			utils.Fatalf("Custom beacon chain config is specified but genesis root is missing")
		}
		if !ctx.IsSet(utils.BeaconGenesisTimeFlag.Name) {
			utils.Fatalf("Custom beacon chain config is specified but genesis time is missing")
		}
		if !ctx.IsSet(utils.BeaconCheckpointFlag.Name) {
			utils.Fatalf("Custom beacon chain config is specified but checkpoint is missing")
		}
		config.ChainConfig = &types.ChainConfig{
			GenesisTime: ctx.Uint64(utils.BeaconGenesisTimeFlag.Name),
		}
		if c, err := hexutil.Decode(ctx.String(utils.BeaconGenesisRootFlag.Name)); err == nil && len(c) <= 32 {
			copy(config.GenesisValidatorsRoot[:len(c)], c)
		} else {
			utils.Fatalf("Invalid hex string", "beacon.genesis.gvroot", ctx.String(utils.BeaconGenesisRootFlag.Name), "error", err)
		}
		if err := config.ChainConfig.LoadForks(ctx.String(utils.BeaconConfigFlag.Name)); err != nil {
			utils.Fatalf("Could not load beacon chain config file", "file name", ctx.String(utils.BeaconConfigFlag.Name), "error", err)
		}
	} else {
		if ctx.IsSet(utils.BeaconGenesisRootFlag.Name) {
			utils.Fatalf("Genesis root is specified but custom beacon chain config is missing")
		}
		if ctx.IsSet(utils.BeaconGenesisTimeFlag.Name) {
			utils.Fatalf("Genesis time is specified but custom beacon chain config is missing")
		}
	}
	// Checkpoint is required with custom chain config and is optional with pre-defined config
	if ctx.IsSet(utils.BeaconCheckpointFlag.Name) {
		if c, err := hexutil.Decode(ctx.String(utils.BeaconCheckpointFlag.Name)); err == nil && len(c) <= 32 {
			copy(config.Checkpoint[:len(c)], c)
		} else {
			utils.Fatalf("Invalid hex string", "beacon.checkpoint", ctx.String(utils.BeaconCheckpointFlag.Name), "error", err)
		}
	}
	return config
}
