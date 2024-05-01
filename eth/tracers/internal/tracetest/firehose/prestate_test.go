package firehose_test

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
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

	return test
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
	if genesis.ExcessBlobGas != nil && genesis.BlobGasUsed != nil {
		excessBlobGas := eip4844.CalcExcessBlobGas(*genesis.ExcessBlobGas, *genesis.BlobGasUsed)
		context.BlobBaseFee = eip4844.CalcBlobFee(excessBlobGas)
	}
	return context
}
