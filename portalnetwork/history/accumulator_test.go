package history

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/portalnetwork/utils"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
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

	startDifficulty := bytesToUint256(epochAcc.HeaderRecords[epochRecordIndex][32:])

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

func bytesToUint256(input []byte) *uint256.Int {
	res := utils.ReverseBytes(input)
	return uint256.MustFromBig(big.NewInt(0).SetBytes(res))
}
