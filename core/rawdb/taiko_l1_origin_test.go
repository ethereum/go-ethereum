package rawdb

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// randomBigInt generates a random big integer.
func randomBigInt() *big.Int {
	randomBigInt, err := rand.Int(rand.Reader, common.Big256)
	if err != nil {
		log.Crit(err.Error())
	}

	return randomBigInt
}

// randomHash generates a random blob of data and returns it as a hash.
func randomHash() common.Hash {
	var hash common.Hash
	if n, err := rand.Read(hash[:]); n != common.HashLength || err != nil {
		panic(err)
	}
	return hash
}

func TestL1Origin(t *testing.T) {
	db := NewMemoryDatabase()
	testL1Origin := &L1Origin{
		BlockID:     randomBigInt(),
		L2BlockHash: randomHash(),
		// L1BlockHeight is intentionally set to nil to represent a value of zero for legacy behavior.
		L1BlockHeight:      nil,
		L1BlockHash:        randomHash(),
		BuildPayloadArgsID: [8]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8},
	}
	WriteL1Origin(db, testL1Origin.BlockID, testL1Origin)
	l1Origin, err := ReadL1Origin(db, testL1Origin.BlockID)
	require.Nil(t, err)
	require.NotNil(t, l1Origin)
	assert.Equal(t, testL1Origin.BlockID, l1Origin.BlockID)
	assert.Equal(t, testL1Origin.L2BlockHash, l1Origin.L2BlockHash)
	assert.True(t, l1Origin.L1BlockHeight.Cmp(common.Big0) == 0)
	assert.Equal(t, testL1Origin.L1BlockHash, l1Origin.L1BlockHash)
	assert.Equal(t, testL1Origin.BuildPayloadArgsID, l1Origin.BuildPayloadArgsID)
}

func TestHeadL1Origin(t *testing.T) {
	db := NewMemoryDatabase()
	testBlockID := randomBigInt()
	WriteHeadL1Origin(db, testBlockID)
	blockID, err := ReadHeadL1Origin(db)
	require.Nil(t, err)
	require.NotNil(t, blockID)
	assert.Equal(t, testBlockID, blockID)
}
