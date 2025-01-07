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
		BlockID:       randomBigInt(),
		L2BlockHash:   randomHash(),
		L1BlockHeight: randomBigInt(),
		L1BlockHash:   randomHash(),
	}
	WriteL1Origin(db, testL1Origin.BlockID, testL1Origin)
	l1Origin, err := ReadL1Origin(db, testL1Origin.BlockID)
	require.Nil(t, err)
	require.NotNil(t, l1Origin)
	assert.Equal(t, testL1Origin.BatchID, l1Origin.BatchID)
	assert.Equal(t, testL1Origin.BlockID, l1Origin.BlockID)
	assert.Equal(t, testL1Origin.L2BlockHash, l1Origin.L2BlockHash)
	assert.Equal(t, testL1Origin.L1BlockHeight, l1Origin.L1BlockHeight)
	assert.Equal(t, testL1Origin.L1BlockHash, l1Origin.L1BlockHash)
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
