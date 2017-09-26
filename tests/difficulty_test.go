// Copyright 2017 The go-ethereum Authors
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
//

package tests

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"strings"
)

type DifficultyTests map[string]difficultyTest

//go:generate gencodec -type difficultyTest -field-override difficultyTestMarshaling -out gen_difficultytest.go

type difficultyTest struct {
	ParentTimestamp    *big.Int    `json:"parentTimestamp"`
	ParentDifficulty   *big.Int    `json:"parentDifficulty"`
	UncleHash          common.Hash `json:"parentUncles"`
	CurrentTimestamp   *big.Int    `json:"currentTimestamp"`
	CurrentBlockNumber uint64      `json:"currentBlockNumber"`
	CurrentDifficulty  *big.Int    `json:"currentDifficulty"`
}

type difficultyTestMarshaling struct {
	ParentTimestamp    *math.HexOrDecimal256
	ParentDifficulty   *math.HexOrDecimal256
	CurrentTimestamp   *math.HexOrDecimal256
	CurrentDifficulty  *math.HexOrDecimal256
	UncleHash          common.Hash
	CurrentBlockNumber math.HexOrDecimal64
}

var (
	mainnetChainConfig = &params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: big.NewInt(1150000),
		DAOForkBlock:   big.NewInt(1920000),
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(2463000),
		EIP150Hash:     common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:    big.NewInt(2675000),
		EIP158Block:    big.NewInt(2675000),
		ByzantiumBlock: big.NewInt(4370000), // Don't enable yet

	}
	homesteadConfig = &params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(math.MaxInt64),
		EIP155Block:    big.NewInt(math.MaxInt64),
		EIP158Block:    big.NewInt(math.MaxInt64),
		ByzantiumBlock: big.NewInt(math.MaxInt64),
	}
	frontierConfig = &params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: big.NewInt(math.MaxInt64),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(math.MaxInt64),
		EIP155Block:    big.NewInt(math.MaxInt64),
		EIP158Block:    big.NewInt(math.MaxInt64),
		ByzantiumBlock: big.NewInt(math.MaxInt64),
	}
	byzantiumConfig = &params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(0),
		EIP155Block:    big.NewInt(0),
		EIP158Block:    big.NewInt(0),
		ByzantiumBlock: big.NewInt(0),
	}
)

func TestDifficulty(t *testing.T) {
	t.Parallel()

	dt := new(testMatcher)
	// Not difficulty-testes
	dt.skipLoad("hexencodetest.*")
	dt.skipLoad("crypto.*")
	dt.skipLoad("blockgenesistest\\.json")
	dt.skipLoad("genesishashestest\\.json")
	dt.skipLoad("keyaddrtest\\.json")
	dt.skipLoad("txtest\\.json")

	// files are 2 years old, strange values
	dt.skipLoad("difficultyCustomHomestead\\.json")
	dt.skipLoad("difficultyMorden\\.json")
	dt.skipLoad("difficultyOlimpic\\.json")

	dt.walk(t, difficultyTestDir, func(t *testing.T, name string, test *difficultyTest) {
		t.Run(name, func(t *testing.T) {
			config := mainnetChainConfig

			switch {
			case strings.Contains(name, "Ropsten") || strings.Contains(name, "Morden"):
				config = params.TestnetChainConfig
			case strings.Contains(name, "Frontier"):
				config = frontierConfig
			case strings.Contains(name, "Homestead"):
				config = homesteadConfig
			case strings.Contains(name, "Byzantium"):
				config = byzantiumConfig
			}
			if test.ParentDifficulty.Cmp(params.MinimumDifficulty) < 0 {
				t.Skip("difficulty below minimum")
				return
			}
			parentNumber := big.NewInt(int64(test.CurrentBlockNumber - 1))
			parent := &types.Header{
				Difficulty: test.ParentDifficulty,
				Time:       test.ParentTimestamp,
				Number:     parentNumber,
				UncleHash:  test.UncleHash,
			}

			actual := ethash.CalcDifficulty(config, test.CurrentTimestamp.Uint64(), parent)
			exp := test.CurrentDifficulty

			if actual.Cmp(exp) != 0 {
				t.Errorf("parent[time %v diff %v unclehash:%x] child[time %v number %v] diff %v != expected %v",
					test.ParentTimestamp, test.ParentDifficulty, test.UncleHash,
					test.CurrentTimestamp, test.CurrentBlockNumber, actual, exp)
			}
		})
	})
}
