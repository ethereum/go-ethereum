package firehose_test

import (
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

func TestPolygon_TracerBlockLevelOnCodeChange(t *testing.T) {
	tracer, hooks, onClose := newFirehoseTestTracer(t, tracingModelFirehose2_3)
	defer onClose()

	hooks.OnBlockchainInit(params.BorMainnetChainConfig)
	hooks.OnBlockStart(tracing.BlockEvent{Block: testBlock()})
	hooks.OnCodeChange(common.Address{}, common.BytesToHash([]byte{0x01}), []byte{0x01}, common.BytesToHash([]byte{0x02}), []byte{0x02})
	hooks.OnBlockEnd(nil)

	assertBlockEquals(t, tracer, filepath.Join("testdata", "PolygonBlockLevelCodeChange"), 1)
}

func testBlock() *types.Block {
	block := types.NewBlock(&types.Header{
		ParentHash:       common.Hash{},
		Number:           big.NewInt(1),
		Difficulty:       big.NewInt(0),
		Coinbase:         common.Address{},
		Time:             1,
		GasLimit:         210000,
		BaseFee:          big.NewInt(1000000000),
		ParentBeaconRoot: ptr(common.Hash{}),
	}, &types.Body{
		Transactions: []*types.Transaction{},
	}, nil, trie.NewStackTrie(nil))

	return block
}
