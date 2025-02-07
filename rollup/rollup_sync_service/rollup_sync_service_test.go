package rollup_sync_service

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

func TestGetCommittedBatchMetaCodecV0(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:             11155111,
				ScrollChainAddress:    common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
				L1MessageQueueAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())

	stack, err := node.New(&node.DefaultConfig)
	require.NoError(t, err, "Failed to create new P2P node")
	defer stack.Close()

	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, &l1.MockNopClient{}, &core.BlockChain{}, stack, da_syncer.Config{
		BlobScanAPIEndpoint: "http://dummy-endpoint:1234",
	})
	require.NoError(t, err)

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 911145, EndBlockNumber: 911151},
		{StartBlockNumber: 911152, EndBlockNumber: 911155},
		{StartBlockNumber: 911156, EndBlockNumber: 911159},
	}

	var chunks []*encoding.DAChunkRawTx
	for _, r := range expectedRanges {
		var blocks []encoding.DABlock
		for i := r.StartBlockNumber; i <= r.EndBlockNumber; i++ {
			blocks = append(blocks, &mockDABlock{number: i})
		}
		chunks = append(chunks, &encoding.DAChunkRawTx{Blocks: blocks})
	}

	committedBatch := mockEntryWithBlocks{
		batchIndex: 1,
		version:    encoding.CodecV0,
		chunks:     chunks,
	}

	metadata, err := service.getCommittedBatchMeta(committedBatch)
	require.NoError(t, err)

	require.Equal(t, encoding.CodecV0, encoding.CodecVersion(metadata.Version))
	require.EqualValues(t, expectedRanges, metadata.ChunkBlockRanges)
}

func TestGetCommittedBatchMetaCodecV1(t *testing.T) {
	genesisConfig := &params.ChainConfig{
		Scroll: params.ScrollConfig{
			L1Config: &params.L1Config{
				L1ChainId:             11155111,
				ScrollChainAddress:    common.HexToAddress("0x2D567EcE699Eabe5afCd141eDB7A4f2D0D6ce8a0"),
				L1MessageQueueAddress: common.HexToAddress("0x0000000000000000000000000000000000000001"),
			},
		},
	}
	db := rawdb.NewDatabase(memorydb.New())

	stack, err := node.New(&node.DefaultConfig)
	require.NoError(t, err, "Failed to create new P2P node")
	defer stack.Close()

	service, err := NewRollupSyncService(context.Background(), genesisConfig, db, &l1.MockNopClient{}, &core.BlockChain{}, stack, da_syncer.Config{
		BlobScanAPIEndpoint: "http://localhost:8080",
	})
	require.NoError(t, err)

	expectedRanges := []*rawdb.ChunkBlockRange{
		{StartBlockNumber: 100, EndBlockNumber: 142},
		{StartBlockNumber: 143, EndBlockNumber: 143},
		{StartBlockNumber: 144, EndBlockNumber: 144},
		{StartBlockNumber: 145, EndBlockNumber: 145},
		{StartBlockNumber: 146, EndBlockNumber: 146},
		{StartBlockNumber: 147, EndBlockNumber: 147},
		{StartBlockNumber: 148, EndBlockNumber: 148},
		{StartBlockNumber: 149, EndBlockNumber: 149},
		{StartBlockNumber: 150, EndBlockNumber: 150},
		{StartBlockNumber: 151, EndBlockNumber: 151},
		{StartBlockNumber: 152, EndBlockNumber: 152},
		{StartBlockNumber: 153, EndBlockNumber: 153},
		{StartBlockNumber: 154, EndBlockNumber: 154},
		{StartBlockNumber: 155, EndBlockNumber: 155},
		{StartBlockNumber: 156, EndBlockNumber: 156},
		{StartBlockNumber: 157, EndBlockNumber: 157},
		{StartBlockNumber: 158, EndBlockNumber: 158},
		{StartBlockNumber: 159, EndBlockNumber: 159},
		{StartBlockNumber: 160, EndBlockNumber: 160},
		{StartBlockNumber: 161, EndBlockNumber: 161},
		{StartBlockNumber: 162, EndBlockNumber: 162},
		{StartBlockNumber: 163, EndBlockNumber: 163},
		{StartBlockNumber: 164, EndBlockNumber: 164},
		{StartBlockNumber: 165, EndBlockNumber: 168},
		{StartBlockNumber: 169, EndBlockNumber: 169},
		{StartBlockNumber: 170, EndBlockNumber: 170},
		{StartBlockNumber: 171, EndBlockNumber: 171},
		{StartBlockNumber: 172, EndBlockNumber: 172},
		{StartBlockNumber: 173, EndBlockNumber: 173},
		{StartBlockNumber: 174, EndBlockNumber: 174},
	}

	var chunks []*encoding.DAChunkRawTx
	for _, r := range expectedRanges {
		var blocks []encoding.DABlock
		for i := r.StartBlockNumber; i <= r.EndBlockNumber; i++ {
			blocks = append(blocks, &mockDABlock{number: i})
		}
		chunks = append(chunks, &encoding.DAChunkRawTx{Blocks: blocks})
	}

	expectedVersionedHashes := []common.Hash{
		common.HexToHash("0x1"),
		common.HexToHash("0x2"),
	}

	committedBatch := mockEntryWithBlocks{
		batchIndex:      1,
		version:         encoding.CodecV1,
		chunks:          chunks,
		versionedHashes: expectedVersionedHashes,
	}

	metadata, err := service.getCommittedBatchMeta(committedBatch)
	require.NoError(t, err)

	require.Equal(t, encoding.CodecV1, encoding.CodecVersion(metadata.Version))
	require.EqualValues(t, expectedRanges, metadata.ChunkBlockRanges)
	require.EqualValues(t, expectedVersionedHashes, metadata.BlobVersionedHashes)
}

type mockEntryWithBlocks struct {
	batchIndex      uint64
	version         encoding.CodecVersion
	chunks          []*encoding.DAChunkRawTx
	versionedHashes []common.Hash
}

func (m mockEntryWithBlocks) Type() da.Type {
	panic("implement me")
}

func (m mockEntryWithBlocks) BatchIndex() uint64 {
	return m.batchIndex
}

func (m mockEntryWithBlocks) L1BlockNumber() uint64 {
	panic("implement me")
}

func (m mockEntryWithBlocks) CompareTo(entry da.Entry) int {
	panic("implement me")
}

func (m mockEntryWithBlocks) Event() l1.RollupEvent {
	panic("implement me")
}

func (m mockEntryWithBlocks) Blocks() []*da.PartialBlock {
	panic("implement me")
}

func (m mockEntryWithBlocks) Version() encoding.CodecVersion {
	return m.version
}

func (m mockEntryWithBlocks) Chunks() []*encoding.DAChunkRawTx {
	return m.chunks
}

func (m mockEntryWithBlocks) BlobVersionedHashes() []common.Hash {
	return m.versionedHashes
}

type mockDABlock struct {
	number uint64
}

func (b *mockDABlock) Encode() []byte {
	panic("implement me")
}

func (b *mockDABlock) Decode(bytes []byte) error {
	panic("implement me")
}

func (b *mockDABlock) NumTransactions() uint16 {
	panic("implement me")
}

func (b *mockDABlock) NumL1Messages() uint16 {
	panic("implement me")
}

func (b *mockDABlock) Timestamp() uint64 {
	panic("implement me")
}

func (b *mockDABlock) BaseFee() *big.Int {
	panic("implement me")
}

func (b *mockDABlock) GasLimit() uint64 {
	panic("implement me")
}

func (b *mockDABlock) Number() uint64 {
	return b.number
}

func TestValidateBatchCodecV0(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentFinalizedBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := l1.NewFinalizeBatchEvent(
		big.NewInt(0),
		common.HexToHash("0xfd3ecf106ce993adc6db68e42ce701bfe638434395abdeeb871f7bd395ae2368"),
		chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV0),
		BlobVersionedHashes: nil,
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1.BatchIndex().Uint64(), event1, parentFinalizedBatchMeta1, committedBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentFinalizedBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash(),
		TotalL1MessagePopped: 11,
		StateRoot:            event1.StateRoot(),
		WithdrawRoot:         event1.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta2, finalizedBatchMeta1)

	event2 := l1.NewFinalizeBatchEvent(
		big.NewInt(1),
		common.HexToHash("0xadb8e526c3fdc2045614158300789cd66e7a945efe5a484db00b5ef9a26016d7"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		2,
	)
	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV0),
		BlobVersionedHashes: nil,
	}

	endBlock2, finalizedBatchMeta2, err := validateBatch(event2.BatchIndex().Uint64(), event2, parentFinalizedBatchMeta2, committedBatchMeta2, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock2)

	parentFinalizedBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event2.StateRoot(),
		WithdrawRoot:         event2.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta3, finalizedBatchMeta2)
}

func TestValidateBatchCodecV1(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentFinalizedBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := l1.NewFinalizeBatchEvent(
		big.NewInt(0),
		common.HexToHash("0x73cb3310646716cb782702a0ec4ad33cf55633c85daf96b641953c5defe58031"),
		chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV1),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x0129554070e4323800ca0e5ddd17bc447854601b306a70870002a058741214b3")},
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1.BatchIndex().Uint64(), event1, parentFinalizedBatchMeta1, committedBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentFinalizedBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash(),
		TotalL1MessagePopped: 11,
		StateRoot:            event1.StateRoot(),
		WithdrawRoot:         event1.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta2, finalizedBatchMeta1)
	event2 := l1.NewFinalizeBatchEvent(
		big.NewInt(1),
		common.HexToHash("0x7f230ce84b4bf86f8ee22ffb5c145e3ef3ddf2a76da4936a33f33cebdb63a48a"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV1),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01a327088bb2b13151449d8313c281d0006d12e8453e863637b746898b6ad5a6")},
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2.BatchIndex().Uint64(), event2, parentFinalizedBatchMeta2, committedBatchMeta2, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock2)

	parentFinalizedBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event2.StateRoot(),
		WithdrawRoot:         event2.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta3, finalizedBatchMeta2)
}

func TestValidateBatchCodecV2(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentFinalizedBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := l1.NewFinalizeBatchEvent(
		big.NewInt(0),
		common.HexToHash("0xaccf37a0b974f2058692d366b2ea85502c99db4a0bcb9b77903b49bf866a463b"),
		chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV2),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x018d99636f4b20ccdc1dd11c289eb2a470e2c4dd631b1a7b48a6978805f49d18")},
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1.BatchIndex().Uint64(), event1, parentFinalizedBatchMeta1, committedBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentFinalizedBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash(),
		TotalL1MessagePopped: 11,
		StateRoot:            event1.StateRoot(),
		WithdrawRoot:         event1.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta2, finalizedBatchMeta1)
	event2 := l1.NewFinalizeBatchEvent(
		big.NewInt(1),
		common.HexToHash("0x62ec61e1fdb334868ffd471df601f6858e692af01d42b5077c805a9fd4558c91"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV2),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x015b4e3d3dcd64cc0eb6a5ad535d7a1844a8c4cdad366ec73557bcc533941370")},
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2.BatchIndex().Uint64(), event2, parentFinalizedBatchMeta2, committedBatchMeta2, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock2)

	parentFinalizedBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event2.StateRoot(),
		WithdrawRoot:         event2.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta3, finalizedBatchMeta2)
}

func TestValidateBatchCodecV3(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentFinalizedBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := l1.NewFinalizeBatchEvent(
		big.NewInt(0),
		common.HexToHash("0x015eb56fb95bf9a06157cfb8389ba7c2b6b08373e22581ac2ba387003708265d"),
		chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)

	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x018d99636f4b20ccdc1dd11c289eb2a470e2c4dd631b1a7b48a6978805f49d18")},
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1.BatchIndex().Uint64(), event1, parentFinalizedBatchMeta1, committedBatchMeta1, []*encoding.Chunk{chunk1, chunk2, chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock1)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentFinalizedBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash(),
		TotalL1MessagePopped: 11,
		StateRoot:            event1.StateRoot(),
		WithdrawRoot:         event1.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta2, finalizedBatchMeta1)
	event2 := l1.NewFinalizeBatchEvent(
		big.NewInt(1),
		common.HexToHash("0x382cb0d507e3d7507f556c52e05f76b05e364ad26205e7f62c95967a19c2f35d"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x015b4e3d3dcd64cc0eb6a5ad535d7a1844a8c4cdad366ec73557bcc533941370")},
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2.BatchIndex().Uint64(), event2, parentFinalizedBatchMeta2, committedBatchMeta2, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock2)

	parentFinalizedBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event2.StateRoot(),
		WithdrawRoot:         event2.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta3, finalizedBatchMeta2)
}

func TestValidateBatchUpgrades(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}

	parentFinalizedBatchMeta1 := &rawdb.FinalizedBatchMeta{}
	event1 := l1.NewFinalizeBatchEvent(
		big.NewInt(0),
		common.HexToHash("0x4605465b7470c8565b123330d7186805caf9a7f2656d8e9e744b62e14ca22c3d"),
		chunk1.Blocks[len(chunk1.Blocks)-1].Header.Root,
		chunk1.Blocks[len(chunk1.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)

	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV0),
		BlobVersionedHashes: nil,
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(event1.BatchIndex().Uint64(), event1, parentFinalizedBatchMeta1, committedBatchMeta1, []*encoding.Chunk{chunk1}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), endBlock1)

	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}

	parentFinalizedBatchMeta2 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event1.BatchHash(),
		TotalL1MessagePopped: 0,
		StateRoot:            event1.StateRoot(),
		WithdrawRoot:         event1.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta2, finalizedBatchMeta1)
	event2 := l1.NewFinalizeBatchEvent(
		big.NewInt(1),
		common.HexToHash("0xc4af33bce87aa702edc3ad4b7d34730d25719427704e250787f99e0f55049252"),
		chunk2.Blocks[len(chunk2.Blocks)-1].Header.Root,
		chunk2.Blocks[len(chunk2.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV1),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01a688c6e137310df38a62f5ad1e5119b8cb0455c386a9a4079b14fe92a239aa")},
	}
	endBlock2, finalizedBatchMeta2, err := validateBatch(event2.BatchIndex().Uint64(), event2, parentFinalizedBatchMeta2, committedBatchMeta2, []*encoding.Chunk{chunk2}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), endBlock2)

	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}

	parentFinalizedBatchMeta3 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event2.BatchHash(),
		TotalL1MessagePopped: 0,
		StateRoot:            event2.StateRoot(),
		WithdrawRoot:         event2.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta3, finalizedBatchMeta2)
	event3 := l1.NewFinalizeBatchEvent(
		big.NewInt(2),
		common.HexToHash("0x9f87f2de2019ed635f867b1e61be6a607c3174ced096f370fd18556c38833c62"),
		chunk3.Blocks[len(chunk3.Blocks)-1].Header.Root,
		chunk3.Blocks[len(chunk3.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta3 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV1),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01ea66c4de196d36e2c3a5d7c0045100b9e46ef65be8f7a921ef20e6f2e99ebd")},
	}
	endBlock3, finalizedBatchMeta3, err := validateBatch(event3.BatchIndex().Uint64(), event3, parentFinalizedBatchMeta3, committedBatchMeta3, []*encoding.Chunk{chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock3)

	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}

	parentFinalizedBatchMeta4 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event3.BatchHash(),
		TotalL1MessagePopped: 11,
		StateRoot:            event3.StateRoot(),
		WithdrawRoot:         event3.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta4, finalizedBatchMeta3)
	event4 := l1.NewFinalizeBatchEvent(
		big.NewInt(3),
		common.HexToHash("0xd33332aef8efbc9a0be4c4694088ac0dd052d2d3ad3ffda5e4c2010825e476bc"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)
	committedBatchMeta4 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x015b4e3d3dcd64cc0eb6a5ad535d7a1844a8c4cdad366ec73557bcc533941370")},
	}
	endBlock4, finalizedBatchMeta4, err := validateBatch(event4.BatchIndex().Uint64(), event4, parentFinalizedBatchMeta4, committedBatchMeta4, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock4)

	parentFinalizedBatchMeta5 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event4.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event4.StateRoot(),
		WithdrawRoot:         event4.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta5, finalizedBatchMeta4)
}

func TestValidateBatchInFinalizeByBundle(t *testing.T) {
	block1 := readBlockFromJSON(t, "./testdata/blockTrace_02.json")
	block2 := readBlockFromJSON(t, "./testdata/blockTrace_03.json")
	block3 := readBlockFromJSON(t, "./testdata/blockTrace_04.json")
	block4 := readBlockFromJSON(t, "./testdata/blockTrace_05.json")

	chunk1 := &encoding.Chunk{Blocks: []*encoding.Block{block1}}
	chunk2 := &encoding.Chunk{Blocks: []*encoding.Block{block2}}
	chunk3 := &encoding.Chunk{Blocks: []*encoding.Block{block3}}
	chunk4 := &encoding.Chunk{Blocks: []*encoding.Block{block4}}
	event := l1.NewFinalizeBatchEvent(
		big.NewInt(3),
		common.HexToHash("0xaa6dc7cc432c8d46a9373e1e96d829a1e24e52fe0468012ff062793ea8f5b55e"),
		chunk4.Blocks[len(chunk4.Blocks)-1].Header.Root,
		chunk4.Blocks[len(chunk4.Blocks)-1].WithdrawRoot,
		common.HexToHash("0x1"),
		common.HexToHash("0x1"),
		1,
	)

	committedBatchMeta1 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01bbc6b98d7d3783730b6208afac839ad37dcf211b9d9e7c83a5f9d02125ddd7")},
	}

	committedBatchMeta2 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x01c81e5696e00f1e6e7d76c197f74ed51650147c49c4e6e5b0b702cdcc54352a")},
	}

	committedBatchMeta3 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x012e15203534ae3f4cbe1b0f58fe6db6e5c29432115a8ece6ef5550bf2ffce4c")},
	}

	committedBatchMeta4 := &rawdb.CommittedBatchMeta{
		Version:             uint8(encoding.CodecV3),
		BlobVersionedHashes: []common.Hash{common.HexToHash("0x015b4e3d3dcd64cc0eb6a5ad535d7a1844a8c4cdad366ec73557bcc533941370")},
	}

	endBlock1, finalizedBatchMeta1, err := validateBatch(0, event, &rawdb.FinalizedBatchMeta{}, committedBatchMeta1, []*encoding.Chunk{chunk1}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), endBlock1)

	endBlock2, finalizedBatchMeta2, err := validateBatch(1, event, finalizedBatchMeta1, committedBatchMeta2, []*encoding.Chunk{chunk2}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), endBlock2)

	endBlock3, finalizedBatchMeta3, err := validateBatch(2, event, finalizedBatchMeta2, committedBatchMeta3, []*encoding.Chunk{chunk3}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(13), endBlock3)

	endBlock4, finalizedBatchMeta4, err := validateBatch(3, event, finalizedBatchMeta3, committedBatchMeta4, []*encoding.Chunk{chunk4}, nil)
	assert.NoError(t, err)
	assert.Equal(t, uint64(17), endBlock4)

	parentFinalizedBatchMeta5 := &rawdb.FinalizedBatchMeta{
		BatchHash:            event.BatchHash(),
		TotalL1MessagePopped: 42,
		StateRoot:            event.StateRoot(),
		WithdrawRoot:         event.WithdrawRoot(),
	}
	assert.Equal(t, parentFinalizedBatchMeta5, finalizedBatchMeta4)
}

func readBlockFromJSON(t *testing.T, filename string) *encoding.Block {
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	block := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(data, block))
	return block
}
