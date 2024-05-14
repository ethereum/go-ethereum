package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestVerifyHeaderWithProofs(t *testing.T) {
	headerWithProofs, err := parseHeaderWithProof()
	assert.NoError(t, err)
	masterAcc, err := NewMasterAccumulator()
	assert.NoError(t, err)
	for _, val := range headerWithProofs {
		head := types.Header{}
		err := rlp.DecodeBytes(val.Header, &head)
		assert.NoError(t, err)
		valid, err := masterAcc.VerifyHeader(head, *val.Proof)
		assert.NoError(t, err)
		assert.True(t, valid)
	}
}

func TestBuildAndVerifyProof(t *testing.T) {
	masterAcc, err := NewMasterAccumulator()
	assert.NoError(t, err)
	epochIndex := GetEpochIndex(1000003)
	epochStr := hexutil.Encode(masterAcc.HistoricalEpochs[epochIndex])
	epochAccumulator, err := getEpochAccu(epochStr)
	assert.NoError(t, err)

	for i := 1000001; i < 1000011; i++ {
		header, err := getHeader(1000003)
		assert.NoError(t, err)

		proof, err := BuildProof(*header, epochAccumulator)
		assert.NoError(t, err)

		valid, err := masterAcc.VerifyAccumulatorProof(*header, proof)
		assert.NoError(t, err)
		assert.True(t, valid)
		assert.True(t, valid)
	}
}

func TestUpdate(t *testing.T) {
	epochAcc, err := getEpochAccu("0xcddbda3fd6f764602c06803ff083dbfc73f2bb396df17a31e5457329b9a0f38d")
	assert.NoError(t, err)

	startNumber := 1000000
	epochRecordIndex := GetHeaderRecordIndex(uint64(startNumber))

	newEpochAcc := NewAccumulator()

	for i := 0; i <= int(epochRecordIndex); i++ {
		tmp := make([]byte, 64)
		copy(tmp, epochAcc.HeaderRecords[i])
		newEpochAcc.currentEpoch.records = append(newEpochAcc.currentEpoch.records, tmp)
	}
	startDifficulty := uint256.NewInt(0)
	err = startDifficulty.UnmarshalSSZ(epochAcc.HeaderRecords[epochRecordIndex][32:])

	require.NoError(t, err)

	newEpochAcc.currentEpoch.difficulty = startDifficulty

	for i := startNumber + 1; i <= 1000010; i++ {
		header, err := getHeader(uint64(i))
		assert.NoError(t, err)
		err = newEpochAcc.Update(*header)
		assert.NoError(t, err)
		currIndex := GetHeaderRecordIndex(uint64(i))
		assert.True(t, bytes.Equal(newEpochAcc.currentEpoch.records[currIndex], epochAcc.HeaderRecords[currIndex]))
	}
}

func TestVerifyPostMergePreCapellaHeader(t *testing.T) {
	acc, err := NewHistoricalRootsAccumulator(configs.Mainnet)
	require.NoError(t, err)
	require.True(t, uint64(len(acc.HistoricalRoots)) < uint64(configs.Mainnet.HISTORICAL_ROOTS_LIMIT))

	root := acc.HistoricalRoots.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	hexutil.Encode(root[:])

	require.Equal(t, hexutil.Encode(root[:]), "0x4df6b89755125d4f6c5575039a04e22301a5a49ee893c1d27e559e3eeab73da7")

	file, err := os.ReadFile("./testdata/block_proofs_bellatrix/beacon_block_proof-15539558-cdf9ed89b0c43cda17398dc4da9cfc505e5ccd19f7c39e3b43474180f1051e01.yaml")
	require.NoError(t, err)
	proof := HistoricalRootsBlockProof{}
	err = yaml.Unmarshal(file, &proof)
	require.NoError(t, err)
	// blockNumber and blockHash are from testfile
	blockHash := hexutil.MustDecode("0xcdf9ed89b0c43cda17398dc4da9cfc505e5ccd19f7c39e3b43474180f1051e01")
	err = acc.VerifyPostMergePreCapellaHeader(15539558, tree.Root(blockHash), &proof)
	require.NoError(t, err)
}

// all test blocks are in the same epoch
func parseHeaderWithProof() ([]BlockHeaderWithProof, error) {
	headWithProofBytes, err := os.ReadFile("./testdata/header_with_proofs.json")
	if err != nil {
		return nil, err
	}
	headerMap := make(map[string]map[string]string)

	err = json.Unmarshal(headWithProofBytes, &headerMap)
	if err != nil {
		return nil, err
	}
	res := make([]BlockHeaderWithProof, 0)
	for _, v := range headerMap {
		val := v["value"]
		bytes, err := hexutil.Decode(val)
		if err != nil {
			return nil, err
		}
		headWithProof := BlockHeaderWithProof{}
		err = headWithProof.UnmarshalSSZ(bytes)
		if err != nil {
			return nil, err
		}
		res = append(res, headWithProof)
	}
	return res, nil
}

func getEpochAccu(name string) (EpochAccumulator, error) {
	epochAccu := EpochAccumulator{
		HeaderRecords: make([][]byte, 0),
	}
	epochData, err := os.ReadFile(fmt.Sprintf("./testdata/%s.bin", name))
	if err != nil {
		return epochAccu, err
	}
	err = epochAccu.UnmarshalSSZ(epochData)
	return epochAccu, err
}

func getHeader(number uint64) (*types.Header, error) {
	headerFile, err := os.ReadFile("./testdata/header_rlps.json")
	if err != nil {
		return nil, err
	}
	contentMap := make(map[string]string)
	err = json.Unmarshal(headerFile, &contentMap)
	if err != nil {
		return nil, err
	}
	headerStr := contentMap[strconv.FormatUint(number, 10)]
	headerBytes, err := hexutil.Decode(headerStr)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(headerBytes)
	head := &types.Header{}
	err = rlp.Decode(reader, head)
	return head, err
}
