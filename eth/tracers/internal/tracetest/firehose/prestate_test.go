package firehose_test

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/cli/server/chains"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

func readPrestateData(t *testing.T, path string) *prestateData {
	t.Helper()

	// Call tracer test found, read if from disk
	blob, err := os.ReadFile(path)
	require.NoError(t, err)

	test := new(prestateData)
	require.NoError(t, json.Unmarshal(blob, test))

	var genesisWithTD struct {
		Genesis struct {
			TotalDifficulty *math.HexOrDecimal256 `json:"totalDifficulty"`
		} `json:"genesis"`
	}
	if err := json.Unmarshal(blob, &genesisWithTD); err == nil {
		test.TotalDifficulty = (*big.Int)(genesisWithTD.Genesis.TotalDifficulty)
	}

	// Polygon overrides time based hard fork with block number based hard fork, let's try to handle that
	// here.
	//
	// We read the "old" genesis Ethereum format which is time based, and we try to infer
	// if we should set the time based hard fork values or not based on current time.
	var genesisTimeBased struct {
		Genesis struct {
			Config struct {
				ShanghaiTime *math.HexOrDecimal64 `json:"shanghaiTime"`
				CancunTime   *math.HexOrDecimal64 `json:"cancunTime"`
				PragueTime   *math.HexOrDecimal64 `json:"pragueTime"`
			} `json:"config"`
		} `json:"genesis"`
	}

	if err := json.Unmarshal(blob, &genesisTimeBased); err == nil {
		timeBasedConfig := genesisTimeBased.Genesis.Config
		blockBasedConfig := test.Genesis.Config

		blockBasedConfig.CancunBlock = timeBasedToBlockBasedHardFork(timeBasedConfig.CancunTime)
		blockBasedConfig.ShanghaiBlock = timeBasedToBlockBasedHardFork(timeBasedConfig.ShanghaiTime)
		blockBasedConfig.PragueBlock = timeBasedToBlockBasedHardFork(timeBasedConfig.PragueTime)
	}

	chain, err := chains.GetChain("mainnet")
	require.NoError(t, err)

	test.Genesis.Config.Bor = chain.Genesis.Config.Bor

	return test
}

func timeBasedToBlockBasedHardFork(in *math.HexOrDecimal64) *big.Int {
	if in == nil {
		return nil
	}

	// If the fork is active, we return 0 as the block number to activate to
	if time.Now().Unix() > int64(*in) {
		return big.NewInt(0)
	}

	// Otherwise, we return the block number that would be mined at that time
	return nil
}

var _ core.ChainContext = (*prestateData)(nil)

type prestateData struct {
	Genesis         *core.Genesis   `json:"genesis"`
	Context         *callContext    `json:"context"`
	Input           string          `json:"input"`
	TotalDifficulty *big.Int        `json:"-"`
	TracerConfig    json.RawMessage `json:"tracerConfig"`

	// Populated after loading
	genesisBlock *types.Block
}

// Config implements core.ChainContext.
func (p *prestateData) Config() *params.ChainConfig {
	return p.Genesis.Config
}

// Engine implements core.ChainContext.
func (p *prestateData) Engine() consensus.Engine {
	return ethash.NewFullFaker()
}

// GetHeader implements core.ChainContext.
func (p *prestateData) GetHeader(hash common.Hash, number uint64) *types.Header {
	if p.Genesis == nil {
		return nil
	}

	if p.genesisBlock == nil {
		p.genesisBlock = p.Genesis.ToBlock()
	}

	if hash == p.genesisBlock.Hash() {
		return p.genesisBlock.Header()
	}

	if number == p.genesisBlock.NumberU64() {
		return p.genesisBlock.Header()
	}

	return nil
}

type callContext struct {
	Number     math.HexOrDecimal64   `json:"number"`
	Difficulty *math.HexOrDecimal256 `json:"difficulty"`
	Time       math.HexOrDecimal64   `json:"timestamp"`
	GasLimit   math.HexOrDecimal64   `json:"gasLimit"`
	Miner      common.Address        `json:"miner"`
	BaseFee    *math.HexOrDecimal256 `json:"baseFeePerGas"`
}

func (c *callContext) toBlockContext(genesis *core.Genesis) vm.BlockContext {
	context := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		Coinbase:    c.Miner,
		BlockNumber: new(big.Int).SetUint64(uint64(c.Number)),
		Time:        uint64(c.Time),
		Difficulty:  (*big.Int)(c.Difficulty),
		GasLimit:    uint64(c.GasLimit),
	}
	if genesis.Config.IsLondon(context.BlockNumber) {
		context.BaseFee = (*big.Int)(c.BaseFee)
	}

	if genesis.Config.TerminalTotalDifficulty != nil && genesis.Config.TerminalTotalDifficulty.Sign() == 0 {
		context.Random = &genesis.Mixhash
	}

	if genesis.ExcessBlobGas != nil && genesis.BlobGasUsed != nil {
		header := &types.Header{Number: genesis.Config.CancunBlock, Time: 0}
		excess := eip4844.CalcExcessBlobGas(genesis.Config, header, genesis.Timestamp)
		header.ExcessBlobGas = &excess
		context.BlobBaseFee = eip4844.CalcBlobFee(genesis.Config, header)
	}
	return context
}
