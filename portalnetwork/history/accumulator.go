package history

import (
	_ "embed"
	"encoding/binary"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
)

const (
	epochSize               = 8192
	mergeBlockNumber uint64 = 15537394
	preMergeEpochs          = (mergeBlockNumber + epochSize - 1) / epochSize
)

var (
	ErrNotPreMergeHeader           = errors.New("must be pre merge header")
	ErrPreMergeHeaderMustWithProof = errors.New("pre merge header must has accumulator proof")
)

//go:embed assets/merge_macc.bin
var masterAccumulatorBytes []byte

var zeroRecordBytes = make([]byte, 64)

type AccumulatorProof [][]byte

type epoch struct {
	records    [][]byte
	difficulty *uint256.Int
}

func newEpoch() *epoch {
	return &epoch{
		records:    make([][]byte, 0, epochSize),
		difficulty: uint256.NewInt(0),
	}
}

func (e *epoch) add(header types.Header) error {
	blockHash := header.Hash().Bytes()
	difficulty := uint256.MustFromBig(header.Number)
	e.difficulty = uint256.NewInt(0).Add(e.difficulty, difficulty)
	record := HeaderRecord{
		BlockHash:       blockHash,
		TotalDifficulty: e.difficulty.Bytes(),
	}
	sszBytes, err := record.MarshalSSZ()
	if err != nil {
		return nil
	}
	e.records = append(e.records, sszBytes)
	return nil
}

type Accumulator struct {
	historicalEpochs [][]byte
	currentEpoch     *epoch
}

type BlockEpochData struct {
	epochHash          []byte
	blockRelativeIndex uint64
}

func NewAccumulator() *Accumulator {
	return &Accumulator{
		historicalEpochs: make([][]byte, 0, int(preMergeEpochs)),
		currentEpoch:     newEpoch(),
	}
}

func (a *Accumulator) Update(header types.Header) error {
	if header.Number.Uint64() >= mergeBlockNumber {
		return ErrNotPreMergeHeader
	}

	if len(a.currentEpoch.records) == epochSize {
		epochAccu := EpochAccumulator{
			HeaderRecords: a.currentEpoch.records,
		}
		root, err := epochAccu.HashTreeRoot()
		if err != nil {
			return err
		}
		a.historicalEpochs = append(a.historicalEpochs, MixInLength(root, epochSize))
		a.currentEpoch = newEpoch()
	}
	a.currentEpoch.add(header)
	return nil
}

func (a *Accumulator) Finish() (*MasterAccumulator, error) {
	// padding with zero bytes
	for len(a.currentEpoch.records) < epochSize {
		a.currentEpoch.records = append(a.currentEpoch.records, zeroRecordBytes)
	}
	epochAccu := EpochAccumulator{
		HeaderRecords: a.currentEpoch.records,
	}
	root, err := epochAccu.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	a.historicalEpochs = append(a.historicalEpochs, MixInLength(root, epochSize))
	return &MasterAccumulator{
		HistoricalEpochs: a.historicalEpochs,
	}, nil
}

func GetEpochIndex(blockNumber uint64) uint64 {
	return blockNumber / epochSize
}

func GetEpochIndexByHeader(header types.Header) uint64 {
	return GetEpochIndex(header.Number.Uint64())
}

func GetHeaderRecordIndex(blockNumber uint64) uint64 {
	return blockNumber % epochSize
}

func GetHeaderRecordIndexByHeader(header types.Header) uint64 {
	return GetHeaderRecordIndex(header.Number.Uint64())
}

func BuildProof(header types.Header, epochAccumulator EpochAccumulator) (AccumulatorProof, error) {
	tree, err := epochAccumulator.GetTree()
	if err != nil {
		return nil, err
	}
	index := GetHeaderRecordIndexByHeader(header)
	// maybe the calculation of index should impl in ssz
	proofIndex := epochSize*2 + index*2
	sszProof, err := tree.Prove(int(proofIndex))
	// the epoch hash root has mix in with epochsize, so we have to add it to proof
	hashes := sszProof.Hashes
	sizeBytes := make([]byte, 32)
	binary.LittleEndian.PutUint32(sizeBytes, epochSize)
	hashes = append(hashes, sizeBytes)
	return AccumulatorProof(hashes), err
}

func BuildHeaderWithProof(header types.Header, epochAccumulator EpochAccumulator) (*BlockHeaderWithProof, error) {
	proof, err := BuildProof(header, epochAccumulator)
	if err != nil {
		return nil, err
	}
	rlpBytes, err := rlp.EncodeToBytes(header)
	if err != nil {
		return nil, err
	}
	return &BlockHeaderWithProof{
		Header: rlpBytes,
		Proof: &BlockHeaderProof{
			Selector: accumulatorProof,
			Proof:    proof,
		},
	}, nil
}

func (f MasterAccumulator) GetBlockEpochDataForBlockNumber(blockNumber uint64) BlockEpochData {
	epochIndex := GetEpochIndex(blockNumber)
	return BlockEpochData{
		epochHash:          f.HistoricalEpochs[epochIndex],
		blockRelativeIndex: GetHeaderRecordIndex(blockNumber),
	}
}

func (f MasterAccumulator) VerifyAccumulatorProof(header types.Header, proof AccumulatorProof) (bool, error) {
	if header.Number.Uint64() > mergeBlockNumber {
		return false, ErrNotPreMergeHeader
	}

	epochIndex := GetEpochIndexByHeader(header)
	root := f.HistoricalEpochs[epochIndex]
	valid := verifyProof(root, header, proof)
	return valid, nil
}

func (f MasterAccumulator) VerifyHeader(header types.Header, headerProof BlockHeaderProof) (bool, error) {
	switch headerProof.Selector {
	case accumulatorProof:
		return f.VerifyAccumulatorProof(header, headerProof.Proof)
	case none:
		if header.Number.Uint64() > mergeBlockNumber {
			return false, ErrNotPreMergeHeader
		}
		return false, ErrPreMergeHeaderMustWithProof
	}
	return false, nil
}

func MixInLength(root [32]byte, length uint64) []byte {
	hash := ssz.NewHasher()
	hash.AppendBytes32(root[:])
	hash.MerkleizeWithMixin(0, length, 0)
	// length of root is 32, so we can ignore the err
	newRoot, _ := hash.HashRoot()
	return newRoot[:]
}

func verifyProof(root []byte, header types.Header, proof AccumulatorProof) bool {
	leaf := header.Hash()

	recordIndex := GetHeaderRecordIndexByHeader(header)
	index := epochSize*2*2 + recordIndex*2
	sszProof := &ssz.Proof{
		Index:  int(index),
		Leaf:   leaf[:],
		Hashes: proof,
	}
	valid, err := ssz.VerifyProof(root, sszProof)
	if err != nil {
		return false
	}
	return valid
}

func NewMasterAccumulator() (MasterAccumulator, error) {
	var masterAcc = MasterAccumulator{
		HistoricalEpochs: make([][]byte, 0),
	}
	err := masterAcc.UnmarshalSSZ(masterAccumulatorBytes)
	return masterAcc, err
}
