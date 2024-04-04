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
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"

	"github.com/scroll-tech/go-ethereum/rollup/types/encoding"
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
	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		t.Fatalf("Failed to new P2P node: %v", err)
	}
	defer stack.Close()
	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, l1Client, bc, stack)
	if err != nil {
		t.Fatalf("Failed to new rollup sync service: %v", err)
	}

	assert.NotNil(t, service)
	service.Start()
	time.Sleep(10 * time.Millisecond)
	service.Stop()
}

func TestDecodeChunkRangesCodecv0(t *testing.T) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	require.NoError(t, err)

	service := &RollupSyncService{
		scrollChainABI: scrollChainABI,
	}

	data, err := os.ReadFile("./testdata/commitBatch_input_codecv0.json")
	require.NoError(t, err, "Failed to read json file")

	type tx struct {
		Input string `json:"input"`
	}
	var commitBatch tx
	err = json.Unmarshal(data, &commitBatch)
	require.NoError(t, err, "Failed to unmarshal transaction json")

	testTxData, err := hex.DecodeString(commitBatch.Input[2:])
	if err != nil {
		t.Fatalf("Failed to decode string: %v", err)
	}

	ranges, err := service.decodeChunkBlockRanges(testTxData)
	if err != nil {
		t.Fatalf("Failed to decode chunk ranges: %v", err)
	}

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 4435142, EndBlockNumber: 4435142},
		{StartBlockNumber: 4435143, EndBlockNumber: 4435144},
		{StartBlockNumber: 4435145, EndBlockNumber: 4435145},
		{StartBlockNumber: 4435146, EndBlockNumber: 4435146},
		{StartBlockNumber: 4435147, EndBlockNumber: 4435147},
		{StartBlockNumber: 4435148, EndBlockNumber: 4435148},
		{StartBlockNumber: 4435149, EndBlockNumber: 4435150},
		{StartBlockNumber: 4435151, EndBlockNumber: 4435151},
		{StartBlockNumber: 4435152, EndBlockNumber: 4435152},
		{StartBlockNumber: 4435153, EndBlockNumber: 4435153},
		{StartBlockNumber: 4435154, EndBlockNumber: 4435154},
		{StartBlockNumber: 4435155, EndBlockNumber: 4435155},
		{StartBlockNumber: 4435156, EndBlockNumber: 4435156},
		{StartBlockNumber: 4435157, EndBlockNumber: 4435157},
		{StartBlockNumber: 4435158, EndBlockNumber: 4435158},
	}

	if len(expectedRanges) != len(ranges) {
		t.Fatalf("Expected range length %v, got %v", len(expectedRanges), len(ranges))
	}

	for i := range ranges {
		if *expectedRanges[i] != *ranges[i] {
			t.Errorf("Mismatch at index %d: expected %v, got %v", i, *expectedRanges[i], *ranges[i])
		}
	}
}

func TestDecodeChunkRangesCodecv1(t *testing.T) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	require.NoError(t, err)

	service := &RollupSyncService{
		scrollChainABI: scrollChainABI,
	}

	data, err := os.ReadFile("./testdata/commitBatch_input_codecv1.json")
	require.NoError(t, err, "Failed to read json file")

	type tx struct {
		Input string `json:"input"`
	}
	var commitBatch tx
	err = json.Unmarshal(data, &commitBatch)
	require.NoError(t, err, "Failed to unmarshal transaction json")

	testTxData, err := hex.DecodeString(commitBatch.Input[2:])
	if err != nil {
		t.Fatalf("Failed to decode string: %v", err)
	}

	ranges, err := service.decodeChunkBlockRanges(testTxData)
	if err != nil {
		t.Fatalf("Failed to decode chunk ranges: %v", err)
	}

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 1690, EndBlockNumber: 1780},
		{StartBlockNumber: 1781, EndBlockNumber: 1871},
		{StartBlockNumber: 1872, EndBlockNumber: 1962},
		{StartBlockNumber: 1963, EndBlockNumber: 2053},
		{StartBlockNumber: 2054, EndBlockNumber: 2144},
		{StartBlockNumber: 2145, EndBlockNumber: 2235},
		{StartBlockNumber: 2236, EndBlockNumber: 2326},
		{StartBlockNumber: 2327, EndBlockNumber: 2417},
		{StartBlockNumber: 2418, EndBlockNumber: 2508},
	}

	if len(expectedRanges) != len(ranges) {
		t.Fatalf("Expected range length %v, got %v", len(expectedRanges), len(ranges))
	}

	for i := range ranges {
		if *expectedRanges[i] != *ranges[i] {
			t.Errorf("Mismatch at index %d: expected %v, got %v", i, *expectedRanges[i], *ranges[i])
		}
	}
}

func TestGetChunkRangesCodecv0(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:          11155111,
				ScrollChainAddress: common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())

	rlpData, err := os.ReadFile("./testdata/commitBatch_codecv0.rlp")
	if err != nil {
		t.Fatalf("Failed to read RLP data: %v", err)
	}
	l1Client := &mockEthClient{
		commitBatchRLP: rlpData,
	}
	bc := &core.BlockChain{}
	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		t.Fatalf("Failed to new P2P node: %v", err)
	}
	defer stack.Close()
	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, l1Client, bc, stack)
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

func TestGetChunkRangesCodecv1(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:          11155111,
				ScrollChainAddress: common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())

	rlpData, err := os.ReadFile("./testdata/commitBatch_codecv1.rlp")
	if err != nil {
		t.Fatalf("Failed to read RLP data: %v", err)
	}
	l1Client := &mockEthClient{
		commitBatchRLP: rlpData,
	}
	bc := &core.BlockChain{}
	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		t.Fatalf("Failed to new P2P node: %v", err)
	}
	defer stack.Close()
	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, l1Client, bc, stack)
	if err != nil {
		t.Fatalf("Failed to new rollup sync service: %v", err)
	}

	vLog := &types.Log{
		TxHash: common.HexToHash("0x1"),
	}
	ranges, err := service.getChunkRanges(1, vLog)
	require.NoError(t, err)

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 1, EndBlockNumber: 11},
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

func TestValidateBatchCodecv0(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(0),
		BatchHash:    common.HexToHash("0xfd3ecf106ce993adc6db68e42ce701bfe638434395abdeeb871f7bd395ae2368"),
		StateRoot:    chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot: chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1, parentBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, &params.ChainConfig{}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash,
		TotalL1MessagePopped: 11,
		StateRoot:            chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot:         chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}
	assert.Equal(t, parentBatchMeta2, finalizedBatchMeta1)
	event2 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(1),
		BatchHash:    common.HexToHash("0xadb8e526c3fdc2045614158300789cd66e7a945efe5a484db00b5ef9a26016d7"),
		StateRoot:    chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		WithdrawRoot: chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2, parentBatchMeta2, []*encoding.Chunk{chunk4}, &params.ChainConfig{}, nil)
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

func TestValidateBatchCodecv1(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(0),
		BatchHash:    common.HexToHash("0x73cb3310646716cb782702a0ec4ad33cf55633c85daf96b641953c5defe58031"),
		StateRoot:    chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot: chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1, parentBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, &params.ChainConfig{BernoulliBlock: big.NewInt(0)}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash,
		TotalL1MessagePopped: 11,
		StateRoot:            chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot:         chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}
	assert.Equal(t, parentBatchMeta2, finalizedBatchMeta1)
	event2 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(1),
		BatchHash:    common.HexToHash("0x8eb3f63fbf286bb51a49879bfc653c53c890621542c640e5b6163cffb5a47aa6"),
		StateRoot:    chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		WithdrawRoot: chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2, parentBatchMeta2, []*encoding.Chunk{chunk4}, &params.ChainConfig{BernoulliBlock: big.NewInt(0)}, nil)
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

func TestValidateBatchFromCodecv0ToCodecv1(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(0),
		BatchHash:    common.HexToHash("0xfd3ecf106ce993adc6db68e42ce701bfe638434395abdeeb871f7bd395ae2368"),
		StateRoot:    chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot: chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1, parentBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, &params.ChainConfig{BernoulliBlock: big.NewInt(16)}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash,
		TotalL1MessagePopped: 11,
		StateRoot:            chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		WithdrawRoot:         chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
	}
	assert.Equal(t, parentBatchMeta2, finalizedBatchMeta1)
	event2 := &L1FinalizeBatchEvent{
		BatchIndex:   big.NewInt(1),
		BatchHash:    common.HexToHash("0x425ab2830087e2642f0407550d65f108ee93533063ef0bfab1263b0b3c8a4c9e"),
		StateRoot:    chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		WithdrawRoot: chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2, parentBatchMeta2, []*encoding.Chunk{chunk4}, &params.ChainConfig{BernoulliBlock: big.NewInt(16)}, nil)
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

func readBlockFromJSON(t *testing.T, filename string) *encoding.Block {
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	block := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(data, block))
	return block
}
