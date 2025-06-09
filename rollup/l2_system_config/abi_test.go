package l2_system_config

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

func TestEventSignatures(t *testing.T) {
	assert.Equal(t, crypto.Keccak256Hash([]byte("BaseFeeOverheadUpdated(uint256,uint256)")), BaseFeeOverheadUpdatedTopic)
	assert.Equal(t, crypto.Keccak256Hash([]byte("BaseFeeScalarUpdated(uint256,uint256)")), BaseFeeScalarUpdatedTopic)
}

func bigToBytesPadded(num *big.Int) []byte {
	return common.BigToHash(num).Bytes()
}

func TestUnpackBaseFeeOverheadUpdatedLog(t *testing.T) {
	old := common.Big1
	new := common.Big2

	log := types.Log{
		Topics: []common.Hash{BaseFeeOverheadUpdatedTopic},
		Data:   append(bigToBytesPadded(old), bigToBytesPadded(new)...),
	}

	event, err := UnpackBaseFeeOverheadUpdatedEvent(log)
	assert.NoError(t, err)
	assert.Equal(t, common.Big1, event.OldBaseFeeOverhead)
	assert.Equal(t, common.Big2, event.NewBaseFeeOverhead)
}

func TestUnpackBaseFeeScalarUpdatedLog(t *testing.T) {
	old := common.Big32
	new := common.Big256

	log := types.Log{
		Topics: []common.Hash{BaseFeeScalarUpdatedTopic},
		Data:   append(bigToBytesPadded(old), bigToBytesPadded(new)...),
	}

	event, err := UnpackBaseFeeScalarUpdatedEvent(log)
	assert.NoError(t, err)
	assert.Equal(t, common.Big32, event.OldBaseFeeScalar)
	assert.Equal(t, common.Big256, event.NewBaseFeeScalar)
}
