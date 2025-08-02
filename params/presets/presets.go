// Copyright 2025 The go-ethereum Authors
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

package presets

import (
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
)

var mainnetTD, _ = new(big.Int).SetString("58_750_000_000_000_000_000_000", 0)

var Mainnet = params.NewConfig2(
	params.Activations{
		forks.Homestead:        1_150_000,
		forks.DAO:              1_920_000,
		forks.TangerineWhistle: 2_463_000,
		forks.SpuriousDragon:   2_675_000,
		forks.Byzantium:        4_370_000,
		forks.Constantinople:   7_280_000,
		forks.Petersburg:       7_280_000,
		forks.Istanbul:         9_069_000,
		forks.MuirGlacier:      9_200_000,
		forks.Berlin:           12_244_000,
		forks.London:           12_965_000,
		forks.ArrowGlacier:     13_773_000,
		forks.GrayGlacier:      15_050_000,
		forks.Paris:            15_537_393,
		// time-based forks
		forks.Shanghai: 1681338455,
		forks.Cancun:   1710338135,
		forks.Prague:   1746612311,
	},
	params.ChainID.V(big.NewInt(1)),
	params.TerminalTotalDifficulty.V(mainnetTD),
	params.DepositContractAddress.V(common.HexToAddress("0x00000000219ab540356cbb839cbe05303d7705fa")),
	params.DAOForkSupport.V(true),
	params.BlobSchedule.V(map[forks.Fork]params.BlobConfig{
		forks.Cancun: *params.DefaultCancunBlobConfig,
		forks.Prague: *params.DefaultPragueBlobConfig,
	}),
)

var sepoliaTD, _ = new(big.Int).SetString("17_000_000_000_000_000", 0)

// SepoliaChainConfig contains the chain parameters to run a node on the Sepolia test network.
var Sepolia = params.NewConfig2(
	params.Activations{
		forks.Homestead:        0,
		forks.TangerineWhistle: 0,
		forks.SpuriousDragon:   0,
		forks.Byzantium:        0,
		forks.Constantinople:   0,
		forks.Petersburg:       0,
		forks.Istanbul:         0,
		forks.MuirGlacier:      0,
		forks.Berlin:           0,
		forks.London:           0,
		forks.Paris:            1735371,
		// time-based forks
		forks.Shanghai: 1677557088,
		forks.Cancun:   1706655072,
		forks.Prague:   1741159776,
	},
	params.ChainID.V(big.NewInt(11155111)),
	params.TerminalTotalDifficulty.V(sepoliaTD),
	params.DepositContractAddress.V(common.HexToAddress("0x7f02c3e3c98b133055b8b348b2ac625669ed295d")),
	params.BlobSchedule.V(map[forks.Fork]params.BlobConfig{
		forks.Cancun: *params.DefaultCancunBlobConfig,
		forks.Prague: *params.DefaultPragueBlobConfig,
	}),
)

// AllEthashProtocolChanges2 contains every protocol change (EIPs) introduced
// and accepted by the Ethereum core developers into the Ethash consensus.
var AllEthashProtocolChanges = params.NewConfig2(
	params.Activations{
		forks.Homestead:        0,
		forks.TangerineWhistle: 0,
		forks.SpuriousDragon:   0,
		forks.Byzantium:        0,
		forks.Constantinople:   0,
		forks.Petersburg:       0,
		forks.Istanbul:         0,
		forks.MuirGlacier:      0,
		forks.Berlin:           0,
		forks.London:           0,
		forks.ArrowGlacier:     0,
		forks.GrayGlacier:      0,
	},
	params.ChainID.V(big.NewInt(1337)),
	params.TerminalTotalDifficulty.V(big.NewInt(math.MaxInt64)),
)

// TestChainConfig contains every protocol change (EIPs) introduced
// and accepted by the Ethereum core developers for testing purposes.
var TestChainConfig = params.NewConfig2(
	params.Activations{
		forks.Homestead:        0,
		forks.TangerineWhistle: 0,
		forks.SpuriousDragon:   0,
		forks.Byzantium:        0,
		forks.Constantinople:   0,
		forks.Petersburg:       0,
		forks.Istanbul:         0,
		forks.MuirGlacier:      0,
		forks.Berlin:           0,
		forks.London:           0,
		forks.ArrowGlacier:     0,
		forks.GrayGlacier:      0,
	},
	params.ChainID.V(big.NewInt(1)),
	params.TerminalTotalDifficulty.V(big.NewInt(math.MaxInt64)),
)

// MergedTestChainConfig2 contains every protocol change (EIPs) introduced
// and accepted by the Ethereum core developers for testing purposes.
var MergedTestChainConfig = params.NewConfig2(
	params.Activations{
		forks.Homestead:        0,
		forks.TangerineWhistle: 0,
		forks.SpuriousDragon:   0,
		forks.Byzantium:        0,
		forks.Constantinople:   0,
		forks.Petersburg:       0,
		forks.Istanbul:         0,
		forks.MuirGlacier:      0,
		forks.Berlin:           0,
		forks.London:           0,
		forks.ArrowGlacier:     0,
		forks.GrayGlacier:      0,
		forks.Paris:            0,
		forks.Shanghai:         0,
		forks.Cancun:           0,
		forks.Prague:           0,
		forks.Osaka:            0,
	},
	params.ChainID.V(big.NewInt(1)),
	params.TerminalTotalDifficulty.V(big.NewInt(0)),
	params.BlobSchedule.V(map[forks.Fork]params.BlobConfig{
		forks.Cancun: *params.DefaultCancunBlobConfig,
		forks.Prague: *params.DefaultPragueBlobConfig,
		forks.Osaka:  *params.DefaultOsakaBlobConfig,
	}),
)
