package codecv0

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding"
)

// CodecV0Version denotes the version of the codec.
const CodecV0Version = 0

// DABlock represents a Data Availability Block.
type DABlock struct {
	BlockNumber     uint64
	Timestamp       uint64
	BaseFee         *big.Int
	GasLimit        uint64
	NumTransactions uint16
	NumL1Messages   uint16
}

// DAChunk groups consecutive DABlocks with their transactions.
type DAChunk struct {
	Blocks       []*DABlock
	Transactions [][]*types.TransactionData
}

// DABatch contains metadata about a batch of DAChunks.
type DABatch struct {
	Version                uint8
	BatchIndex             uint64
	L1MessagePopped        uint64
	TotalL1MessagePopped   uint64
	DataHash               common.Hash
	ParentBatchHash        common.Hash
	SkippedL1MessageBitmap []byte
}

// NewDABlock creates a new DABlock from the given encoding.Block and the total number of L1 messages popped before.
func NewDABlock(block *encoding.Block, totalL1MessagePoppedBefore uint64) (*DABlock, error) {
	if !block.Header.Number.IsUint64() {
		return nil, errors.New("block number is not uint64")
	}

	// note: numL1Messages includes skipped messages
	numL1Messages := block.NumL1Messages(totalL1MessagePoppedBefore)
	if numL1Messages > math.MaxUint16 {
		return nil, errors.New("number of L1 messages exceeds max uint16")
	}

	// note: numTransactions includes skipped messages
	numL2Transactions := block.NumL2Transactions()
	numTransactions := numL1Messages + numL2Transactions
	if numTransactions > math.MaxUint16 {
		return nil, errors.New("number of transactions exceeds max uint16")
	}

	daBlock := DABlock{
		BlockNumber:     block.Header.Number.Uint64(),
		Timestamp:       block.Header.Time,
		BaseFee:         block.Header.BaseFee,
		GasLimit:        block.Header.GasLimit,
		NumTransactions: uint16(numTransactions),
		NumL1Messages:   uint16(numL1Messages),
	}

	return &daBlock, nil
}

// Encode serializes the DABlock into a slice of bytes.
func (b *DABlock) Encode() []byte {
	bytes := make([]byte, 60)
	binary.BigEndian.PutUint64(bytes[0:], b.BlockNumber)
	binary.BigEndian.PutUint64(bytes[8:], b.Timestamp)
	if b.BaseFee != nil {
		binary.BigEndian.PutUint64(bytes[40:], b.BaseFee.Uint64())
	}
	binary.BigEndian.PutUint64(bytes[48:], b.GasLimit)
	binary.BigEndian.PutUint16(bytes[56:], b.NumTransactions)
	binary.BigEndian.PutUint16(bytes[58:], b.NumL1Messages)
	return bytes
}

// DecodeDABlock takes a byte slice and decodes it into a DABlock.
func DecodeDABlock(bytes []byte) (*DABlock, error) {
	if len(bytes) != 60 {
		return nil, errors.New("block encoding is not 60 bytes long")
	}

	block := &DABlock{
		BlockNumber:     binary.BigEndian.Uint64(bytes[0:8]),
		Timestamp:       binary.BigEndian.Uint64(bytes[8:16]),
		BaseFee:         new(big.Int).SetUint64(binary.BigEndian.Uint64(bytes[40:48])),
		GasLimit:        binary.BigEndian.Uint64(bytes[48:56]),
		NumTransactions: binary.BigEndian.Uint16(bytes[56:58]),
		NumL1Messages:   binary.BigEndian.Uint16(bytes[58:60]),
	}

	return block, nil
}

// NewDAChunk creates a new DAChunk from the given encoding.Chunk and the total number of L1 messages popped before.
func NewDAChunk(chunk *encoding.Chunk, totalL1MessagePoppedBefore uint64) (*DAChunk, error) {
	var blocks []*DABlock
	var txs [][]*types.TransactionData

	if chunk == nil {
		return nil, errors.New("chunk is nil")
	}

	if len(chunk.Blocks) == 0 {
		return nil, errors.New("number of blocks is 0")
	}

	if len(chunk.Blocks) > 255 {
		return nil, errors.New("number of blocks exceeds 1 byte")
	}

	for _, block := range chunk.Blocks {
		b, err := NewDABlock(block, totalL1MessagePoppedBefore)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
		totalL1MessagePoppedBefore += block.NumL1Messages(totalL1MessagePoppedBefore)
		txs = append(txs, block.Transactions)
	}

	daChunk := DAChunk{
		Blocks:       blocks,
		Transactions: txs,
	}

	return &daChunk, nil
}

// Encode serializes the DAChunk into a slice of bytes.
func (c *DAChunk) Encode() ([]byte, error) {
	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(len(c.Blocks)))

	var l2TxDataBytes []byte

	for _, block := range c.Blocks {
		chunkBytes = append(chunkBytes, block.Encode()...)
	}

	for _, blockTxs := range c.Transactions {
		for _, txData := range blockTxs {
			if txData.Type == types.L1MessageTxType {
				continue
			}
			var txLen [4]byte
			rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(txData)
			if err != nil {
				return nil, err
			}
			binary.BigEndian.PutUint32(txLen[:], uint32(len(rlpTxData)))
			l2TxDataBytes = append(l2TxDataBytes, txLen[:]...)
			l2TxDataBytes = append(l2TxDataBytes, rlpTxData...)
		}
	}

	chunkBytes = append(chunkBytes, l2TxDataBytes...)
	return chunkBytes, nil
}

// Hash computes the hash of the DAChunk data.
func (c *DAChunk) Hash() (common.Hash, error) {
	chunkBytes, err := c.Encode()
	if err != nil {
		return common.Hash{}, err
	}

	if len(chunkBytes) == 0 {
		return common.Hash{}, errors.New("chunk data is empty and cannot be processed")
	}
	numBlocks := chunkBytes[0]

	// concatenate block contexts
	var dataBytes []byte
	for i := 0; i < int(numBlocks); i++ {
		// only the first 58 bytes of each BlockContext are needed for the hashing process
		dataBytes = append(dataBytes, chunkBytes[1+60*i:60*i+59]...)
	}

	// concatenate l1 and l2 tx hashes
	for _, blockTxs := range c.Transactions {
		var l1TxHashes []byte
		var l2TxHashes []byte
		for _, txData := range blockTxs {
			txHash := strings.TrimPrefix(txData.TxHash, "0x")
			hashBytes, err := hex.DecodeString(txHash)
			if err != nil {
				return common.Hash{}, fmt.Errorf("failed to decode tx hash from TransactionData: hash=%v, err=%w", txData.TxHash, err)
			}
			if txData.Type == types.L1MessageTxType {
				l1TxHashes = append(l1TxHashes, hashBytes...)
			} else {
				l2TxHashes = append(l2TxHashes, hashBytes...)
			}
		}
		dataBytes = append(dataBytes, l1TxHashes...)
		dataBytes = append(dataBytes, l2TxHashes...)
	}

	hash := crypto.Keccak256Hash(dataBytes)
	return hash, nil
}

// NewDABatch creates a DABatch from the provided encoding.Batch.
func NewDABatch(batch *encoding.Batch) (*DABatch, error) {
	// compute batch data hash
	var dataBytes []byte
	totalL1MessagePoppedBeforeChunk := batch.TotalL1MessagePoppedBefore

	for _, chunk := range batch.Chunks {
		// build data hash
		daChunk, err := NewDAChunk(chunk, totalL1MessagePoppedBeforeChunk)
		if err != nil {
			return nil, err
		}
		totalL1MessagePoppedBeforeChunk += chunk.NumL1Messages(totalL1MessagePoppedBeforeChunk)
		daChunkHash, err := daChunk.Hash()
		if err != nil {
			return nil, err
		}
		dataBytes = append(dataBytes, daChunkHash.Bytes()...)
	}

	// compute data hash
	dataHash := crypto.Keccak256Hash(dataBytes)

	// skipped L1 messages bitmap
	bitmapBytes, totalL1MessagePoppedAfter, err := encoding.ConstructSkippedBitmap(batch.Index, batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, err
	}

	daBatch := DABatch{
		Version:                CodecV0Version,
		BatchIndex:             batch.Index,
		L1MessagePopped:        totalL1MessagePoppedAfter - batch.TotalL1MessagePoppedBefore,
		TotalL1MessagePopped:   totalL1MessagePoppedAfter,
		DataHash:               dataHash,
		ParentBatchHash:        batch.ParentBatchHash,
		SkippedL1MessageBitmap: bitmapBytes,
	}

	return &daBatch, nil
}

// NewDABatchFromBytes attempts to decode the given byte slice into a DABatch.
func NewDABatchFromBytes(data []byte) (*DABatch, error) {
	if len(data) < 89 {
		return nil, fmt.Errorf("insufficient data for DABatch, expected at least 89 bytes but got %d", len(data))
	}

	b := &DABatch{
		Version:                data[0],
		BatchIndex:             binary.BigEndian.Uint64(data[1:9]),
		L1MessagePopped:        binary.BigEndian.Uint64(data[9:17]),
		TotalL1MessagePopped:   binary.BigEndian.Uint64(data[17:25]),
		DataHash:               common.BytesToHash(data[25:57]),
		ParentBatchHash:        common.BytesToHash(data[57:89]),
		SkippedL1MessageBitmap: data[89:],
	}

	return b, nil
}

// Encode serializes the DABatch into bytes.
func (b *DABatch) Encode() []byte {
	batchBytes := make([]byte, 89+len(b.SkippedL1MessageBitmap))
	batchBytes[0] = b.Version
	binary.BigEndian.PutUint64(batchBytes[1:], b.BatchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.L1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.TotalL1MessagePopped)
	copy(batchBytes[25:], b.DataHash[:])
	copy(batchBytes[57:], b.ParentBatchHash[:])
	copy(batchBytes[89:], b.SkippedL1MessageBitmap[:])
	return batchBytes
}

// Hash computes the hash of the serialized DABatch.
func (b *DABatch) Hash() common.Hash {
	bytes := b.Encode()
	return crypto.Keccak256Hash(bytes)
}

// DecodeFromCalldata attempts to decode a DABatch and an array of DAChunks from the provided calldata byte slice.
func DecodeFromCalldata(data []byte) (*DABatch, []*DAChunk, error) {
	// TODO: implement this function.
	return nil, nil, nil
}

// CalldataNonZeroByteGas is the gas consumption per non zero byte in calldata.
const CalldataNonZeroByteGas = 16

// GetKeccak256Gas calculates the gas cost for computing the keccak256 hash of a given size.
func GetKeccak256Gas(size uint64) uint64 {
	return GetMemoryExpansionCost(size) + 30 + 6*((size+31)/32)
}

// GetMemoryExpansionCost calculates the cost of memory expansion for a given memoryByteSize.
func GetMemoryExpansionCost(memoryByteSize uint64) uint64 {
	memorySizeWord := (memoryByteSize + 31) / 32
	memoryCost := (memorySizeWord*memorySizeWord)/512 + (3 * memorySizeWord)
	return memoryCost
}

// EstimateBlockL1CommitCalldataSize calculates the calldata size in l1 commit for this block approximately.
// TODO: The calculation could be more accurate by using 58 + len(l2TxDataBytes) (see Chunk).
// This needs to be adjusted in the future.
func EstimateBlockL1CommitCalldataSize(b *encoding.Block) (uint64, error) {
	var size uint64
	for _, txData := range b.Transactions {
		if txData.Type == types.L1MessageTxType {
			continue
		}
		size += 4 // 4 bytes payload length
		txPayloadLength, err := getTxPayloadLength(txData)
		if err != nil {
			return 0, err
		}
		size += txPayloadLength
	}
	size += 60 // 60 bytes BlockContext
	return size, nil
}

// EstimateBlockL1CommitGas calculates the total L1 commit gas for this block approximately.
func EstimateBlockL1CommitGas(b *encoding.Block) (uint64, error) {
	var total uint64
	var numL1Messages uint64
	for _, txData := range b.Transactions {
		if txData.Type == types.L1MessageTxType {
			numL1Messages++
			continue
		}

		txPayloadLength, err := getTxPayloadLength(txData)
		if err != nil {
			return 0, err
		}
		total += CalldataNonZeroByteGas * txPayloadLength // an over-estimate: treat each byte as non-zero
		total += CalldataNonZeroByteGas * 4               // 4 bytes payload length
		total += GetKeccak256Gas(txPayloadLength)         // l2 tx hash
	}

	// 60 bytes BlockContext calldata
	total += CalldataNonZeroByteGas * 60

	// sload
	total += 2100 * numL1Messages // numL1Messages times cold sload in L1MessageQueue

	// staticcall
	total += 100 * numL1Messages // numL1Messages times call to L1MessageQueue
	total += 100 * numL1Messages // numL1Messages times warm address access to L1MessageQueue

	total += GetMemoryExpansionCost(36) * numL1Messages // staticcall to proxy
	total += 100 * numL1Messages                        // read admin in proxy
	total += 100 * numL1Messages                        // read impl in proxy
	total += 100 * numL1Messages                        // access impl
	total += GetMemoryExpansionCost(36) * numL1Messages // delegatecall to impl

	return total, nil
}

// EstimateChunkL1CommitCalldataSize calculates the calldata size needed for committing a chunk to L1 approximately.
func EstimateChunkL1CommitCalldataSize(c *encoding.Chunk) (uint64, error) {
	var totalL1CommitCalldataSize uint64
	for _, block := range c.Blocks {
		blockL1CommitCalldataSize, err := EstimateBlockL1CommitCalldataSize(block)
		if err != nil {
			return 0, err
		}
		totalL1CommitCalldataSize += blockL1CommitCalldataSize
	}
	return totalL1CommitCalldataSize, nil
}

// EstimateChunkL1CommitGas calculates the total L1 commit gas for this chunk approximately.
func EstimateChunkL1CommitGas(c *encoding.Chunk) (uint64, error) {
	var totalTxNum uint64
	var totalL1CommitGas uint64
	for _, block := range c.Blocks {
		totalTxNum += uint64(len(block.Transactions))
		blockL1CommitGas, err := EstimateBlockL1CommitGas(block)
		if err != nil {
			return 0, err
		}
		totalL1CommitGas += blockL1CommitGas
	}

	numBlocks := uint64(len(c.Blocks))
	totalL1CommitGas += 100 * numBlocks                         // numBlocks times warm sload
	totalL1CommitGas += CalldataNonZeroByteGas                  // numBlocks field of chunk encoding in calldata
	totalL1CommitGas += CalldataNonZeroByteGas * numBlocks * 60 // numBlocks of BlockContext in chunk

	totalL1CommitGas += GetKeccak256Gas(58*numBlocks + 32*totalTxNum) // chunk hash
	return totalL1CommitGas, nil
}

// EstimateBatchL1CommitGas calculates the total L1 commit gas for this batch approximately.
func EstimateBatchL1CommitGas(b *encoding.Batch) (uint64, error) {
	var totalL1CommitGas uint64

	// Add extra gas costs
	totalL1CommitGas += 100000                 // constant to account for ops like _getAdmin, _implementation, _requireNotPaused, etc
	totalL1CommitGas += 4 * 2100               // 4 one-time cold sload for commitBatch
	totalL1CommitGas += 20000                  // 1 time sstore
	totalL1CommitGas += 21000                  // base fee for tx
	totalL1CommitGas += CalldataNonZeroByteGas // version in calldata

	// adjusting gas:
	// add 1 time cold sload (2100 gas) for L1MessageQueue
	// add 1 time cold address access (2600 gas) for L1MessageQueue
	// minus 1 time warm sload (100 gas) & 1 time warm address access (100 gas)
	totalL1CommitGas += (2100 + 2600 - 100 - 100)
	totalL1CommitGas += GetKeccak256Gas(89 + 32)           // parent batch header hash, length is estimated as 89 (constant part)+ 32 (1 skippedL1MessageBitmap)
	totalL1CommitGas += CalldataNonZeroByteGas * (89 + 32) // parent batch header in calldata

	// adjust batch data hash gas cost
	totalL1CommitGas += GetKeccak256Gas(uint64(32 * len(b.Chunks)))

	totalL1MessagePoppedBefore := b.TotalL1MessagePoppedBefore

	for _, chunk := range b.Chunks {
		chunkL1CommitGas, err := EstimateChunkL1CommitGas(chunk)
		if err != nil {
			return 0, err
		}
		totalL1CommitGas += chunkL1CommitGas

		totalL1MessagePoppedInChunk := chunk.NumL1Messages(totalL1MessagePoppedBefore)
		totalL1MessagePoppedBefore += totalL1MessagePoppedInChunk

		totalL1CommitGas += CalldataNonZeroByteGas * (32 * (totalL1MessagePoppedInChunk + 255) / 256)
		totalL1CommitGas += GetKeccak256Gas(89 + 32*(totalL1MessagePoppedInChunk+255)/256)

		totalL1CommitCalldataSize, err := EstimateChunkL1CommitCalldataSize(chunk)
		if err != nil {
			return 0, err
		}
		totalL1CommitGas += GetMemoryExpansionCost(totalL1CommitCalldataSize)
	}

	return totalL1CommitGas, nil
}

// EstimateBatchL1CommitCalldataSize calculates the calldata size in l1 commit for this batch approximately.
func EstimateBatchL1CommitCalldataSize(b *encoding.Batch) (uint64, error) {
	var totalL1CommitCalldataSize uint64
	for _, chunk := range b.Chunks {
		chunkL1CommitCalldataSize, err := EstimateChunkL1CommitCalldataSize(chunk)
		if err != nil {
			return 0, err
		}
		totalL1CommitCalldataSize += chunkL1CommitCalldataSize
	}
	return totalL1CommitCalldataSize, nil
}

func getTxPayloadLength(txData *types.TransactionData) (uint64, error) {
	rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(txData)
	if err != nil {
		return 0, err
	}
	return uint64(len(rlpTxData)), nil
}
