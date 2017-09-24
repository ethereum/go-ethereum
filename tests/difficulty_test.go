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
	// Values taken from
	// https://github.com/ethereum/cpp-ethereum/blob/6cb2c852700024daf79e7cb31e4c12ffb7d85537/test/unittests/libethcore/difficulty.cpp#L212
	// and
	// https://github.com/ethereum/cpp-ethereum/blob/develop/libethashseal/genesis/test/mainNetworkTest.cpp

	customMainnetConfig = &params.ChainConfig{
		ChainId:        big.NewInt(1),
		HomesteadBlock: big.NewInt(0x118c30),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(0x259518),
		EIP155Block:    big.NewInt(0x259518),
		EIP158Block:    big.NewInt(0x28d138),
		ByzantiumBlock: big.NewInt(0x2dc6c0),
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
	// Contains erroneously generated uncle-values : "0x01" -should be hashes
	dt.skipLoad("difficultyCustomMainNetwork\\.json")

	dt.walk(t, difficultyTestDir, func(t *testing.T, name string, test *difficultyTest) {
		t.Run(name, func(t *testing.T) {
			config := params.MainnetChainConfig

			switch {
			case strings.Contains(name, "Ropsten") || strings.Contains(name, "Morden"):
				config = params.TestnetChainConfig
			case strings.Contains(name, "Frontier"):
				config = frontierConfig
			case strings.Contains(name, "Homestead"):
				config = homesteadConfig
			case strings.Contains(name, "Byzantium"):
				config = byzantiumConfig
			case strings.Contains(name, "CustomMainNetwork"):
				config = customMainnetConfig
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

				t.Errorf("parent[time %v diff %v #uncles:%v] child[time %v number %v] diff %v != expected %v",
					test.ParentTimestamp, test.ParentDifficulty, test.UncleHash,
					test.CurrentTimestamp, test.CurrentBlockNumber, actual, exp)
			}
		})
	})
}
