package rollup_sync_service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/params"
)

func TestRollupSyncServiceStartAndStop(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:          11155111,
				ScrollChainAddress: common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())
	l1Client := &mockEthClient{}
	bc := &core.BlockChain{}
	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, l1Client, bc, 1)
	if err != nil {
		t.Fatalf("Failed to new rollup sync service: %v", err)
	}

	assert.NotNil(t, service)
	service.Start()
	time.Sleep(10 * time.Millisecond)
	service.Stop()
}

func TestDecodeChunkRanges(t *testing.T) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	require.NoError(t, err)

	service := &RollupSyncService{
		scrollChainABI: scrollChainABI,
	}

	data, err := os.ReadFile("./testdata/commit_batch_transaction.json")
	require.NoError(t, err, "Failed to read json file")

	type transactionJson struct {
		CallData string `json:"calldata"`
	}
	var txObj transactionJson
	err = json.Unmarshal(data, &txObj)
	require.NoError(t, err, "Failed to unmarshal transaction json")

	testTxData, err := hex.DecodeString(txObj.CallData[2:])
	if err != nil {
		t.Fatalf("Failed to decode string: %v", err)
	}

	ranges, err := service.decodeChunkBlockRanges(testTxData)
	if err != nil {
		t.Fatalf("Failed to decode chunk ranges: %v", err)
	}

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 335921, EndBlockNumber: 335928},
		{StartBlockNumber: 335929, EndBlockNumber: 335933},
		{StartBlockNumber: 335934, EndBlockNumber: 335938},
		{StartBlockNumber: 335939, EndBlockNumber: 335942},
		{StartBlockNumber: 335943, EndBlockNumber: 335945},
		{StartBlockNumber: 335946, EndBlockNumber: 335949},
		{StartBlockNumber: 335950, EndBlockNumber: 335956},
		{StartBlockNumber: 335957, EndBlockNumber: 335962},
	}

	if len(expectedRanges) != len(ranges) {
		t.Fatalf("Expected range length %v, got %v", len(expectedRanges), len(ranges))
	}

	for i := range ranges {
		if *expectedRanges[i] != *ranges[i] {
			t.Fatalf("Mismatch at index %d: expected %v, got %v", i, *expectedRanges[i], *ranges[i])
		}
	}
}

func TestGetChunkRanges(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:          11155111,
				ScrollChainAddress: common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())

	rlpData, err := os.ReadFile("./testdata/commit_batch_tx.rlp")
	if err != nil {
		t.Fatalf("Failed to read RLP data: %v", err)
	}
	l1Client := &mockEthClient{
		commitBatchRLP: rlpData,
	}
	bc := &core.BlockChain{}
	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, l1Client, bc, 1)
	if err != nil {
		t.Fatalf("Failed to new rollup sync service: %v", err)
	}

	vLog := &types.Log{
		TxHash: common.HexToHash("0x0"),
	}
	ranges, err := service.getChunkRanges(1, vLog)
	require.NoError(t, err)

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 911145, EndBlockNumber: 911151},
		{StartBlockNumber: 911152, EndBlockNumber: 911155},
		{StartBlockNumber: 911156, EndBlockNumber: 911159},
	}

	if len(expectedRanges) != len(ranges) {
		t.Fatalf("Expected range length %v, got %v", len(expectedRanges), len(ranges))
	}

	for i := range ranges {
		if *expectedRanges[i] != *ranges[i] {
			t.Fatalf("Mismatch at index %d: expected %v, got %v", i, *expectedRanges[i], *ranges[i])
		}
	}
}

func TestValidateBatch(t *testing.T) {
	templateBlockTrace1, err := os.ReadFile("./testdata/blockTrace_02.json")
	require.NoError(t, err)
	wrappedBlock1 := &WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace1, wrappedBlock1)
	require.NoError(t, err)
	chunk1 := &Chunk{Blocks: []*WrappedBlock{wrappedBlock1}}

	templateBlockTrace2, err := os.ReadFile("./testdata/blockTrace_03.json")
	require.NoError(t, err)
	wrappedBlock2 := &WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace2, wrappedBlock2)
	require.NoError(t, err)
	chunk2 := &Chunk{Blocks: []*WrappedBlock{wrappedBlock2}}

	templateBlockTrace3, err := os.ReadFile("./testdata/blockTrace_04.json")
	require.NoError(t, err)
	wrappedBlock3 := &WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace3, wrappedBlock3)
	require.NoError(t, err)
	chunk3 := &Chunk{Blocks: []*WrappedBlock{wrappedBlock3}}

	parentBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(0),
		BatchHash:    common.HexToHash("0xd0f52bc254646e639bf24cc34606319a111975b2fdc431b1381eb6199bc09790"),
		StateRoot:    chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot: chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}
	endBlock1, finalizedBatchMeta1, err := validateBatch(event1, parentBatchMeta1, []*Chunk{chunk1, chunk2, chunk3})
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	templateBlockTrace4, err := os.ReadFile("./testdata/blockTrace_05.json")
	require.NoError(t, err)
	wrappedBlock4 := &WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace4, wrappedBlock4)
	require.NoError(t, err)
	chunk4 := &Chunk{Blocks: []*WrappedBlock{wrappedBlock4}}

	parentBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash,
		TotalL1MessagePopped: 11,
		StateRoot:            chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot:         chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}
	assert.Equal(t, parentBatchMeta2, finalizedBatchMeta1)
	event2 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(1),
		BatchHash:    common.HexToHash("0xfb77bf8f3bf449126ebbf403fdccfcf78636e34d72d62eed8da0e8c9fd38fa63"),
		StateRoot:    chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		WithdrawRoot: chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2, parentBatchMeta2, []*Chunk{chunk4})
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock2)

	parentBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash,
		TotalL1MessagePopped: 42,
		StateRoot:            chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		WithdrawRoot:         chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
	}
	assert.Equal(t, parentBatchMeta3, finalizedBatchMeta2)
}
