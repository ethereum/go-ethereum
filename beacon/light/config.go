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

package light

import (
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

type ChainConfig struct {
	*types.ChainConfig
	Checkpoint common.Hash
}

type ClientConfig struct {
	ChainConfig
	Threshold int
	TimeCheck bool
}

var (
	MainnetConfig = ChainConfig{
		ChainConfig: (&types.ChainConfig{
			GenesisValidatorsRoot: common.HexToHash("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95"),
			GenesisTime:           1606824023,
		}).
			AddFork("GENESIS", 0, []byte{0, 0, 0, 0}).
			AddFork("ALTAIR", 74240, []byte{1, 0, 0, 0}).
			AddFork("BELLATRIX", 144896, []byte{2, 0, 0, 0}).
			AddFork("CAPELLA", 194048, []byte{3, 0, 0, 0}),
		Checkpoint: common.HexToHash("0x388be41594ec7d6a6894f18c73f3469f07e2c19a803de4755d335817ed8e2e5a"),
	}

	SepoliaConfig = ChainConfig{
		ChainConfig: (&types.ChainConfig{
			GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
			GenesisTime:           1655733600,
		}).
			AddFork("GENESIS", 0, []byte{144, 0, 0, 105}).
			AddFork("ALTAIR", 50, []byte{144, 0, 0, 112}).
			AddFork("BELLATRIX", 100, []byte{144, 0, 0, 113}).
			AddFork("CAPELLA", 56832, []byte{144, 0, 0, 114}),
		Checkpoint: common.HexToHash("0x1005a6d9175e96bfbce4d35b80f468e9bff0b674e1e861d16e09e10005a58e81"),
	}

	GoerliConfig = ChainConfig{
		ChainConfig: (&types.ChainConfig{
			GenesisValidatorsRoot: common.HexToHash("0x043db0d9a83813551ee2f33450d23797757d430911a9320530ad8a0eabc43efb"),
			GenesisTime:           1614588812,
		}).
			AddFork("GENESIS", 0, []byte{0, 0, 16, 32}).
			AddFork("ALTAIR", 36660, []byte{1, 0, 16, 32}).
			AddFork("BELLATRIX", 112260, []byte{2, 0, 16, 32}).
			AddFork("CAPELLA", 162304, []byte{3, 0, 16, 32}),
		Checkpoint: common.HexToHash("0x53a0f4f0a378e2c4ae0a9ee97407eb69d0d737d8d8cd0a5fb1093f42f7b81c49"),
	}
)
