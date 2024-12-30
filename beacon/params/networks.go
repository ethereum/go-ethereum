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

package params

import (
	"github.com/ethereum/go-ethereum/common"
)

var (
	MainnetLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95"),
		GenesisTime:           1606824023,
		Checkpoint:            common.HexToHash("0x6509b691f4de4f7b083f2784938fd52f0e131675432b3fd85ea549af9aebd3d0"),
	}).
		AddFork("GENESIS", 0, []byte{0, 0, 0, 0}).
		AddFork("ALTAIR", 74240, []byte{1, 0, 0, 0}).
		AddFork("BELLATRIX", 144896, []byte{2, 0, 0, 0}).
		AddFork("CAPELLA", 194048, []byte{3, 0, 0, 0}).
		AddFork("DENEB", 269568, []byte{4, 0, 0, 0})

	SepoliaLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
		GenesisTime:           1655733600,
		Checkpoint:            common.HexToHash("0x456e85f5608afab3465a0580bff8572255f6d97af0c5f939e3f7536b5edb2d3f"),
	}).
		AddFork("GENESIS", 0, []byte{144, 0, 0, 105}).
		AddFork("ALTAIR", 50, []byte{144, 0, 0, 112}).
		AddFork("BELLATRIX", 100, []byte{144, 0, 0, 113}).
		AddFork("CAPELLA", 56832, []byte{144, 0, 0, 114}).
		AddFork("DENEB", 132608, []byte{144, 0, 0, 115})

	HoleskyLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x9143aa7c615a7f7115e2b6aac319c03529df8242ae705fba9df39b79c59fa8b1"),
		GenesisTime:           1695902400,
		Checkpoint:            common.HexToHash("0x6456a1317f54d4b4f2cb5bc9d153b5af0988fe767ef0609f0236cf29030bcff7"),
	}).
		AddFork("GENESIS", 0, []byte{1, 1, 112, 0}).
		AddFork("ALTAIR", 0, []byte{2, 1, 112, 0}).
		AddFork("BELLATRIX", 0, []byte{3, 1, 112, 0}).
		AddFork("CAPELLA", 256, []byte{4, 1, 112, 0}).
		AddFork("DENEB", 29696, []byte{5, 1, 112, 0})
)
