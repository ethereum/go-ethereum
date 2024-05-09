package history

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/util/merkle"
	"github.com/protolambda/ztyp/codec"
)

const (
	epochSize                  = 8192
	mergeBlockNumber    uint64 = 15537394
	shanghaiBlockNumber uint64 = 17_034_870
	preMergeEpochs             = (mergeBlockNumber + epochSize - 1) / epochSize
)

var (
	ErrNotPreMergeHeader           = errors.New("must be pre merge header")
	ErrPreMergeHeaderMustWithProof = errors.New("pre merge header must has accumulator proof")
)

//go:embed assets/merge_macc.txt
var masterAccumulatorHex string

//go:embed assets/historical_roots.ssz
var historicalRootsBytes []byte

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
	difficulty := uint256.MustFromBig(header.Difficulty)
	e.difficulty = uint256.NewInt(0).Add(e.difficulty, difficulty)

	difficultyBytes, err := e.difficulty.MarshalSSZ()
	if err != nil {
		return err
	}
	record := HeaderRecord{
		BlockHash:       blockHash,
		TotalDifficulty: difficultyBytes,
	}
	sszBytes, err := record.MarshalSSZ()
	if err != nil {
		return err
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
	err := a.currentEpoch.add(header)
	if err != nil {
		return err
	}
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
	if err != nil {
		return nil, err
	}
	// the epoch hash root has mix in with epochsize, so we have to add it to proof
	hashes := sszProof.Hashes
	sizeBytes := make([]byte, 32)
	binary.LittleEndian.PutUint32(sizeBytes, epochSize)
	hashes = append(hashes, sizeBytes)
	return hashes, err
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
		if header.Number.Uint64() <= mergeBlockNumber {
			return false, ErrPreMergeHeaderMustWithProof
		}
		return true, nil
	}
	return false, fmt.Errorf("unknown header proof selector %v", headerProof.Selector)
}

func (f MasterAccumulator) Contains(epochHash []byte) bool {
	for _, h := range f.HistoricalEpochs {
		if bytes.Equal(h, epochHash) {
			return true
		}
	}
	return false
}

func MixInLength(root [32]byte, length uint64) []byte {
	hash := ssz.NewHasher()
	hash.AppendBytes32(root[:])
	hash.MerkleizeWithMixin(0, length, 0)
	// length of root is 32, so we can ignore the error
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
	masterAccumulatorBytes, err := hexutil.Decode(masterAccumulatorHex)
	if err != nil {
		return masterAcc, err
	}
	err = masterAcc.UnmarshalSSZ(masterAccumulatorBytes)
	return masterAcc, err
}

type HistoricalRootsAccumulator struct {
	HistoricalRoots HistoricalRoots
}

func NewHistoricalRootsAccumulator(spec *common.Spec) (HistoricalRootsAccumulator, error) {
	historicalRoots := new(HistoricalRoots)
	reader := codec.NewDecodingReader(bytes.NewReader(historicalRootsBytes), uint64(len(historicalRootsBytes)))
	err := historicalRoots.Deserialize(spec, reader)
	return HistoricalRootsAccumulator{HistoricalRoots: *historicalRoots}, err
}

func (h HistoricalRootsAccumulator) VerifyPostMergePreCapellaHeader(blockNumber uint64, headerHash common.Root, proof *HistoricalRootsBlockProof) error {
	if blockNumber <= mergeBlockNumber {
		return errors.New("invalid historicalRootsBlockProof found for pre-merge header")
	}
	if blockNumber >= shanghaiBlockNumber {
		return errors.New("invalid historicalRootsBlockProof found for post-Shanghai header")
	}
	if !merkle.VerifyMerkleBranch(headerHash, proof.BeaconBlockBodyProof[:], 8, 412, proof.BeaconBlockBodyRoot) {
		return errors.New("merkle proof validation failed for BeaconBlockBodyProof")
	}
	if !merkle.VerifyMerkleBranch(proof.BeaconBlockBodyRoot, proof.BeaconBlockHeaderProof[:], 3, 12, proof.BeaconBlockHeaderRoot) {
		return errors.New("merkle proof validation failed for BeaconBlockHeaderProof")
	}

	blockRootIndex := proof.Slot % epochSize
	genIndex := 2*epochSize + blockRootIndex
	historicalRootIndex := proof.Slot / epochSize
	historicalRoot := h.HistoricalRoots[historicalRootIndex]

	if !merkle.VerifyMerkleBranch(proof.BeaconBlockHeaderRoot, proof.HistoricalRootsProof[:], 14, uint64(genIndex), historicalRoot) {
		return errors.New("merkle proof validation failed for HistoricalRootsProof")
	}
	return nil
}
