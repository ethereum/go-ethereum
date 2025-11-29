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

//go:embed checkpoint_hoodi.hex
var checkpointHoodi string

var (
	MainnetLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x4b363db94e286120d76eb905340fdd4e54bfe9f06bf33ff6cf5ad27f511bfe95"),
		GenesisTime:           1606824023,
		Checkpoint:            common.HexToHash(checkpointMainnet),
	}).
		AddFork("GENESIS", 0, common.FromHex("0x00000000")).
		AddFork("ALTAIR", 74240, common.FromHex("0x01000000")).
		AddFork("BELLATRIX", 144896, common.FromHex("0x02000000")).
		AddFork("CAPELLA", 194048, common.FromHex("0x03000000")).
		AddFork("DENEB", 269568, common.FromHex("0x04000000")).
		AddFork("ELECTRA", 364032, common.FromHex("0x05000000")).
		AddFork("FULU", 411392, common.FromHex("0x06000000"))

	SepoliaLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0xd8ea171f3c94aea21ebc42a1ed61052acf3f9209c00e4efbaaddac09ed9b8078"),
		GenesisTime:           1655733600,
		Checkpoint:            common.HexToHash(checkpointSepolia),
	}).
		AddFork("GENESIS", 0, common.FromHex("0x90000069")).
		AddFork("ALTAIR", 50, common.FromHex("0x90000070")).
		AddFork("BELLATRIX", 100, common.FromHex("0x90000071")).
		AddFork("CAPELLA", 56832, common.FromHex("0x90000072")).
		AddFork("DENEB", 132608, common.FromHex("0x90000073")).
		AddFork("ELECTRA", 222464, common.FromHex("0x90000074")).
		AddFork("FULU", 272640, common.FromHex("0x90000075"))

	HoleskyLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x9143aa7c615a7f7115e2b6aac319c03529df8242ae705fba9df39b79c59fa8b1"),
		GenesisTime:           1695902400,
		Checkpoint:            common.HexToHash(checkpointHolesky),
	}).
		AddFork("GENESIS", 0, common.FromHex("0x01017000")).
		AddFork("ALTAIR", 0, common.FromHex("0x02017000")).
		AddFork("BELLATRIX", 0, common.FromHex("0x03017000")).
		AddFork("CAPELLA", 256, common.FromHex("0x04017000")).
		AddFork("DENEB", 29696, common.FromHex("0x05017000")).
		AddFork("ELECTRA", 115968, common.FromHex("0x06017000")).
		AddFork("FULU", 165120, common.FromHex("0x07017000"))

	HoodiLightConfig = (&ChainConfig{
		GenesisValidatorsRoot: common.HexToHash("0x212f13fc4df078b6cb7db228f1c8307566dcecf900867401a92023d7ba99cb5f"),
		GenesisTime:           1742212800,
		Checkpoint:            common.HexToHash(checkpointHoodi),
	}).
		AddFork("GENESIS", 0, common.FromHex("0x10000910")).
		AddFork("ALTAIR", 0, common.FromHex("0x20000910")).
		AddFork("BELLATRIX", 0, common.FromHex("0x30000910")).
		AddFork("CAPELLA", 0, common.FromHex("0x40000910")).
		AddFork("DENEB", 0, common.FromHex("0x50000910")).
		AddFork("ELECTRA", 2048, common.FromHex("0x60000910")).
		AddFork("FULU", 50688, common.FromHex("0x70000910"))
)
