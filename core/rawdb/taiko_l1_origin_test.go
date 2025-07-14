package rawdb

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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
		IsForcedInclusion:  true,
		Signature:          [65]byte{0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10},
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
	assert.Equal(t, testL1Origin.IsForcedInclusion, l1Origin.IsForcedInclusion)
	assert.Equal(t, testL1Origin.Signature, l1Origin.Signature)
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

func TestReadL1OriginFallbacks(t *testing.T) {
	db := NewMemoryDatabase()

	t.Run("LegacyTwo → L1Origin", func(t *testing.T) {
		// prepare a second‐legacy L1Origin
		blockID := randomBigInt()
		height := randomBigInt()
		l2Hash := randomHash()
		l1Hash := randomHash()
		buildID := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		legacyTwo := &L1OriginLegacyTwo{
			BlockID:            blockID,
			L2BlockHash:        l2Hash,
			L1BlockHeight:      height,
			L1BlockHash:        l1Hash,
			BuildPayloadArgsID: buildID,
		}

		// encode & write raw RLP
		data, err := rlp.EncodeToBytes(legacyTwo)
		require.NoError(t, err)
		require.NoError(t, db.Put(l1OriginKey(blockID), data))

		// read back via our helper
		got, err := ReadL1Origin(db, blockID)
		require.NoError(t, err)
		require.NotNil(t, got)

		// verify fields
		assert.Equal(t, blockID, got.BlockID)
		assert.Equal(t, l2Hash, got.L2BlockHash)
		assert.True(t, got.L1BlockHeight.Cmp(height) == 0)
		assert.Equal(t, l1Hash, got.L1BlockHash)
		assert.Equal(t, buildID, got.BuildPayloadArgsID)
		assert.False(t, got.IsForcedInclusion)
		assert.Equal(t, [65]byte{}, got.Signature)
	})

	t.Run("LegacyOne → L1Origin", func(t *testing.T) {
		// prepare the original legacy L1Origin
		blockID := randomBigInt()
		height := randomBigInt()
		l2Hash := randomHash()
		l1Hash := randomHash()

		legacyOne := &L1OriginLegacy{
			BlockID:       blockID,
			L2BlockHash:   l2Hash,
			L1BlockHeight: height,
			L1BlockHash:   l1Hash,
		}

		// encode & write raw RLP
		data, err := rlp.EncodeToBytes(legacyOne)
		require.NoError(t, err)
		require.NoError(t, db.Put(l1OriginKey(blockID), data))

		// read back via our helper
		got, err := ReadL1Origin(db, blockID)
		require.NoError(t, err)
		require.NotNil(t, got)

		// verify fields
		assert.Equal(t, blockID, got.BlockID)
		assert.Equal(t, l2Hash, got.L2BlockHash)
		assert.True(t, got.L1BlockHeight.Cmp(height) == 0)
		assert.Equal(t, l1Hash, got.L1BlockHash)
		// new fields should be zero-default
		assert.Equal(t, [8]byte{}, got.BuildPayloadArgsID)
		assert.False(t, got.IsForcedInclusion)
		assert.Equal(t, [65]byte{}, got.Signature)
	})
}
