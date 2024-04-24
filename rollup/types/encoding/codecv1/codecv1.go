package codecv1

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding"
)

var (
	// BLSModulus is the BLS modulus defined in EIP-4844.
	BLSModulus *big.Int

	// BlobDataProofArgs defines the argument types for `_blobDataProof` in `finalizeBatchWithProof4844`.
	BlobDataProofArgs abi.Arguments

	// MaxNumChunks is the maximum number of chunks that a batch can contain.
	MaxNumChunks int = 15
)

func init() {
	// initialize modulus
	modulus, success := new(big.Int).SetString("52435875175126190479447740508185965837690552500527637822603658699938581184513", 10)
	if !success {
		log.Crit("BLSModulus conversion failed")
	}
	BLSModulus = modulus

	// initialize arguments
	bytes32Type, err1 := abi.NewType("bytes32", "bytes32", nil)
	bytes48Type, err2 := abi.NewType("bytes48", "bytes48", nil)
	if err1 != nil || err2 != nil {
		log.Crit("Failed to initialize abi types", "err1", err1, "err2", err2)
	}

	BlobDataProofArgs = abi.Arguments{
		{Type: bytes32Type, Name: "z"},
		{Type: bytes32Type, Name: "y"},
		{Type: bytes48Type, Name: "commitment"},
		{Type: bytes48Type, Name: "proof"},
	}
}

// CodecV1Version denotes the version of the codec.
const CodecV1Version = 1

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
	// header
	Version                uint8
	BatchIndex             uint64
	L1MessagePopped        uint64
	TotalL1MessagePopped   uint64
	DataHash               common.Hash
	BlobVersionedHash      common.Hash
	ParentBatchHash        common.Hash
	SkippedL1MessageBitmap []byte

	// blob payload
	blob *kzg4844.Blob
	z    *kzg4844.Point
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
func (c *DAChunk) Encode() []byte {
	var chunkBytes []byte
	chunkBytes = append(chunkBytes, byte(len(c.Blocks)))

	for _, block := range c.Blocks {
		blockBytes := block.Encode()
		chunkBytes = append(chunkBytes, blockBytes...)
	}

	return chunkBytes
}

// Hash computes the hash of the DAChunk data.
func (c *DAChunk) Hash() (common.Hash, error) {
	var dataBytes []byte

	// concatenate block contexts
	for _, block := range c.Blocks {
		encodedBlock := block.Encode()
		// only the first 58 bytes are used in the hashing process
		dataBytes = append(dataBytes, encodedBlock[:58]...)
	}

	// concatenate l1 tx hashes
	for _, blockTxs := range c.Transactions {
		for _, txData := range blockTxs {
			if txData.Type == types.L1MessageTxType {
				txHash := strings.TrimPrefix(txData.TxHash, "0x")
				hashBytes, err := hex.DecodeString(txHash)
				if err != nil {
					return common.Hash{}, err
				}
				if len(hashBytes) != 32 {
					return common.Hash{}, fmt.Errorf("unexpected hash: %s", txData.TxHash)
				}
				dataBytes = append(dataBytes, hashBytes...)
			}
		}
	}

	hash := crypto.Keccak256Hash(dataBytes)
	return hash, nil
}

// NewDABatch creates a DABatch from the provided encoding.Batch.
func NewDABatch(batch *encoding.Batch) (*DABatch, error) {
	// this encoding can only support a fixed number of chunks per batch
	if len(batch.Chunks) > MaxNumChunks {
		return nil, fmt.Errorf("too many chunks in batch")
	}

	if len(batch.Chunks) == 0 {
		return nil, fmt.Errorf("too few chunks in batch")
	}

	// batch data hash
	dataHash, err := computeBatchDataHash(batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, err
	}

	// skipped L1 messages bitmap
	bitmapBytes, totalL1MessagePoppedAfter, err := encoding.ConstructSkippedBitmap(batch.Index, batch.Chunks, batch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, err
	}

	// blob payload
	blob, blobVersionedHash, z, err := constructBlobPayload(batch.Chunks)
	if err != nil {
		return nil, err
	}

	daBatch := DABatch{
		Version:                CodecV1Version,
		BatchIndex:             batch.Index,
		L1MessagePopped:        totalL1MessagePoppedAfter - batch.TotalL1MessagePoppedBefore,
		TotalL1MessagePopped:   totalL1MessagePoppedAfter,
		DataHash:               dataHash,
		BlobVersionedHash:      blobVersionedHash,
		ParentBatchHash:        batch.ParentBatchHash,
		SkippedL1MessageBitmap: bitmapBytes,
		blob:                   blob,
		z:                      z,
	}

	return &daBatch, nil
}

// computeBatchDataHash computes the data hash of the batch.
// Note: The batch hash and batch data hash are two different hashes,
// the former is used for identifying a badge in the contracts,
// the latter is used in the public input to the provers.
func computeBatchDataHash(chunks []*encoding.Chunk, totalL1MessagePoppedBefore uint64) (common.Hash, error) {
	var dataBytes []byte
	totalL1MessagePoppedBeforeChunk := totalL1MessagePoppedBefore

	for _, chunk := range chunks {
		daChunk, err := NewDAChunk(chunk, totalL1MessagePoppedBeforeChunk)
		if err != nil {
			return common.Hash{}, err
		}
		totalL1MessagePoppedBeforeChunk += chunk.NumL1Messages(totalL1MessagePoppedBeforeChunk)
		chunkHash, err := daChunk.Hash()
		if err != nil {
			return common.Hash{}, err
		}
		dataBytes = append(dataBytes, chunkHash.Bytes()...)
	}

	dataHash := crypto.Keccak256Hash(dataBytes)
	return dataHash, nil
}

// constructBlobPayload constructs the 4844 blob payload.
func constructBlobPayload(chunks []*encoding.Chunk) (*kzg4844.Blob, common.Hash, *kzg4844.Point, error) {
	// metadata consists of num_chunks (2 bytes) and chunki_size (4 bytes per chunk)
	metadataLength := 2 + MaxNumChunks*4

	// the raw (un-padded) blob payload
	blobBytes := make([]byte, metadataLength)

	// challenge digest preimage
	// 1 hash for metadata, 1 hash for each chunk, 1 hash for blob versioned hash
	challengePreimage := make([]byte, (1+MaxNumChunks+1)*32)

	// the chunk data hash used for calculating the challenge preimage
	var chunkDataHash common.Hash

	// blob metadata: num_chunks
	binary.BigEndian.PutUint16(blobBytes[0:], uint16(len(chunks)))

	// encode blob metadata and L2 transactions,
	// and simultaneously also build challenge preimage
	for chunkID, chunk := range chunks {
		currentChunkStartIndex := len(blobBytes)

		for _, block := range chunk.Blocks {
			for _, tx := range block.Transactions {
				if tx.Type != types.L1MessageTxType {
					// encode L2 txs into blob payload
					rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(tx)
					if err != nil {
						return nil, common.Hash{}, nil, err
					}
					blobBytes = append(blobBytes, rlpTxData...)
				}
			}
		}

		// blob metadata: chunki_size
		if chunkSize := len(blobBytes) - currentChunkStartIndex; chunkSize != 0 {
			binary.BigEndian.PutUint32(blobBytes[2+4*chunkID:], uint32(chunkSize))
		}

		// challenge: compute chunk data hash
		chunkDataHash = crypto.Keccak256Hash(blobBytes[currentChunkStartIndex:])
		copy(challengePreimage[32+chunkID*32:], chunkDataHash[:])
	}

	// if we have fewer than MaxNumChunks chunks, the rest
	// of the blob metadata is correctly initialized to 0,
	// but we need to add padding to the challenge preimage
	for chunkID := len(chunks); chunkID < MaxNumChunks; chunkID++ {
		// use the last chunk's data hash as padding
		copy(challengePreimage[32+chunkID*32:], chunkDataHash[:])
	}

	// challenge: compute metadata hash
	hash := crypto.Keccak256Hash(blobBytes[0:metadataLength])
	copy(challengePreimage[0:], hash[:])

	// convert raw data to BLSFieldElements
	blob, err := makeBlobCanonical(blobBytes)
	if err != nil {
		return nil, common.Hash{}, nil, err
	}

	// compute blob versioned hash
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, common.Hash{}, nil, fmt.Errorf("failed to create blob commitment")
	}
	blobVersionedHash := kzg4844.CalcBlobHashV1(sha256.New(), &c)

	// challenge: append blob versioned hash
	copy(challengePreimage[(1+MaxNumChunks)*32:], blobVersionedHash[:])

	// compute z = challenge_digest % BLS_MODULUS
	challengeDigest := crypto.Keccak256Hash(challengePreimage)
	pointBigInt := new(big.Int).Mod(new(big.Int).SetBytes(challengeDigest[:]), BLSModulus)
	pointBytes := pointBigInt.Bytes()

	// the challenge point z
	var z kzg4844.Point
	start := 32 - len(pointBytes)
	copy(z[start:], pointBytes)

	return blob, blobVersionedHash, &z, nil
}

// makeBlobCanonical converts the raw blob data into the canonical blob representation of 4096 BLSFieldElements.
func makeBlobCanonical(blobBytes []byte) (*kzg4844.Blob, error) {
	// blob contains 131072 bytes but we can only utilize 31/32 of these
	if len(blobBytes) > 126976 {
		return nil, fmt.Errorf("oversized batch payload")
	}

	// the canonical (padded) blob payload
	var blob kzg4844.Blob

	// encode blob payload by prepending every 31 bytes with 1 zero byte
	index := 0

	for from := 0; from < len(blobBytes); from += 31 {
		to := from + 31
		if to > len(blobBytes) {
			to = len(blobBytes)
		}
		copy(blob[index+1:], blobBytes[from:to])
		index += 32
	}

	return &blob, nil
}

// NewDABatchFromBytes attempts to decode the given byte slice into a DABatch.
// Note: This function only populates the batch header, it leaves the blob-related fields empty.
func NewDABatchFromBytes(data []byte) (*DABatch, error) {
	if len(data) < 121 {
		return nil, fmt.Errorf("insufficient data for DABatch, expected at least 121 bytes but got %d", len(data))
	}

	b := &DABatch{
		Version:                data[0],
		BatchIndex:             binary.BigEndian.Uint64(data[1:9]),
		L1MessagePopped:        binary.BigEndian.Uint64(data[9:17]),
		TotalL1MessagePopped:   binary.BigEndian.Uint64(data[17:25]),
		DataHash:               common.BytesToHash(data[25:57]),
		BlobVersionedHash:      common.BytesToHash(data[57:89]),
		ParentBatchHash:        common.BytesToHash(data[89:121]),
		SkippedL1MessageBitmap: data[121:],
	}

	return b, nil
}

// Encode serializes the DABatch into bytes.
func (b *DABatch) Encode() []byte {
	batchBytes := make([]byte, 121+len(b.SkippedL1MessageBitmap))
	batchBytes[0] = b.Version
	binary.BigEndian.PutUint64(batchBytes[1:], b.BatchIndex)
	binary.BigEndian.PutUint64(batchBytes[9:], b.L1MessagePopped)
	binary.BigEndian.PutUint64(batchBytes[17:], b.TotalL1MessagePopped)
	copy(batchBytes[25:], b.DataHash[:])
	copy(batchBytes[57:], b.BlobVersionedHash[:])
	copy(batchBytes[89:], b.ParentBatchHash[:])
	copy(batchBytes[121:], b.SkippedL1MessageBitmap[:])
	return batchBytes
}

// Hash computes the hash of the serialized DABatch.
func (b *DABatch) Hash() common.Hash {
	bytes := b.Encode()
	return crypto.Keccak256Hash(bytes)
}

// BlobDataProof computes the abi-encoded blob verification data.
func (b *DABatch) BlobDataProof() ([]byte, error) {
	if b.blob == nil {
		return nil, errors.New("called BlobDataProof with empty blob")
	}
	if b.z == nil {
		return nil, errors.New("called BlobDataProof with empty z")
	}

	commitment, err := kzg4844.BlobToCommitment(b.blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment")
	}

	proof, y, err := kzg4844.ComputeProof(b.blob, *b.z)
	if err != nil {
		log.Crit("failed to create KZG proof at point", "err", err, "z", hex.EncodeToString(b.z[:]))
	}

	// Memory layout of ``_blobDataProof``:
	// | z       | y       | kzg_commitment | kzg_proof |
	// |---------|---------|----------------|-----------|
	// | bytes32 | bytes32 | bytes48        | bytes48   |

	values := []interface{}{*b.z, y, commitment, proof}
	return BlobDataProofArgs.Pack(values...)
}

// Blob returns the blob of the batch.
func (b *DABatch) Blob() *kzg4844.Blob {
	return b.blob
}

// DecodeFromCalldata attempts to decode a DABatch and an array of DAChunks from the provided calldata byte slice.
func DecodeFromCalldata(data []byte) (*DABatch, []*DAChunk, error) {
	// TODO: implement this function.
	return nil, nil, nil
}

// EstimateChunkL1CommitBlobSize estimates the size of the L1 commit blob for a single chunk.
func EstimateChunkL1CommitBlobSize(c *encoding.Chunk) (uint64, error) {
	metadataSize := uint64(2 + 4*MaxNumChunks) // over-estimate: adding metadata length
	chunkDataSize, err := chunkL1CommitBlobDataSize(c)
	if err != nil {
		return 0, err
	}
	paddedSize := ((metadataSize + chunkDataSize + 30) / 31) * 32
	return paddedSize, nil
}

// EstimateBatchL1CommitBlobSize estimates the total size of the L1 commit blob for a batch.
func EstimateBatchL1CommitBlobSize(b *encoding.Batch) (uint64, error) {
	metadataSize := uint64(2 + 4*MaxNumChunks)
	var batchDataSize uint64
	for _, c := range b.Chunks {
		chunkDataSize, err := chunkL1CommitBlobDataSize(c)
		if err != nil {
			return 0, err
		}
		batchDataSize += chunkDataSize
	}
	paddedSize := ((metadataSize + batchDataSize + 30) / 31) * 32
	return paddedSize, nil
}

func chunkL1CommitBlobDataSize(c *encoding.Chunk) (uint64, error) {
	var dataSize uint64
	for _, block := range c.Blocks {
		for _, tx := range block.Transactions {
			if tx.Type != types.L1MessageTxType {
				rlpTxData, err := encoding.ConvertTxDataToRLPEncoding(tx)
				if err != nil {
					return 0, err
				}
				dataSize += uint64(len(rlpTxData))
			}
		}
	}
	return dataSize, nil
}
