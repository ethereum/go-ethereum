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

package params

import (
	_ "embed"

	"github.com/ethereum/go-ethereum/common"
)

//go:embed checkpoint_mainnet.hex
var checkpointMainnet string

//go:embed checkpoint_sepolia.hex
var checkpointSepolia string

//go:embed checkpoint_holesky.hex
var checkpointHolesky string

var (
	MainnetLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95"),
		GenesisTime:           1606824023,
		Checkpoint:            common.HexToHash(checkpointMainnet),
	}).
		AddFork("GENESIS", 0, []byte{0, 0, 0, 0}).
		AddFork("ALTAIR", 74240, []byte{1, 0, 0, 0}).
		AddFork("BELLATRIX", 144896, []byte{2, 0, 0, 0}).
		AddFork("CAPELLA", 194048, []byte{3, 0, 0, 0}).
		AddFork("DENEB", 269568, []byte{4, 0, 0, 0}).
		AddFork("ELECTRA", 364032, []byte{5, 0, 0, 0})

	SepoliaLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
		GenesisTime:           1655733600,
		Checkpoint:            common.HexToHash(checkpointSepolia),
	}).
		AddFork("GENESIS", 0, []byte{144, 0, 0, 105}).
		AddFork("ALTAIR", 50, []byte{144, 0, 0, 112}).
		AddFork("BELLATRIX", 100, []byte{144, 0, 0, 113}).
		AddFork("CAPELLA", 56832, []byte{144, 0, 0, 114}).
		AddFork("DENEB", 132608, []byte{144, 0, 0, 115}).
		AddFork("ELECTRA", 222464, []byte{144, 0, 0, 116})

	HoleskyLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x9143aa7c615a7f7115e2b6aac319c03529df8242ae705fba9df39b79c59fa8b1"),
		GenesisTime:           1695902400,
		Checkpoint:            common.HexToHash(checkpointHolesky),
	}).
		AddFork("GENESIS", 0, []byte{1, 1, 112, 0}).
		AddFork("ALTAIR", 0, []byte{2, 1, 112, 0}).
		AddFork("BELLATRIX", 0, []byte{3, 1, 112, 0}).
		AddFork("CAPELLA", 256, []byte{4, 1, 112, 0}).
		AddFork("DENEB", 29696, []byte{5, 1, 112, 0}).
		AddFork("ELECTRA", 115968, []byte{6, 1, 112, 0})

	HoodiLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x212f13fc4df078b6cb7db228f1c8307566dcecf900867401a92023d7ba99cb5f"),
		GenesisTime:           1742212800,
		Checkpoint:            common.HexToHash(""),
	}).
		AddFork("GENESIS", 0, common.FromHex("0x10000910")).
		AddFork("ALTAIR", 0, common.FromHex("0x20000910")).
		AddFork("BELLATRIX", 0, common.FromHex("0x30000910")).
		AddFork("CAPELLA", 0, common.FromHex("0x40000910")).
		AddFork("DENEB", 0, common.FromHex("0x50000910")).
		AddFork("ELECTRA", 2048, common.FromHex("0x60000910")).
		AddFork("FULU", 18446744073709551615, common.FromHex("0x70000910"))
)
