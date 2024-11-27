package ethpepple

import (
	"testing"

	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func genBytes(length int) []byte {
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		res[i] = byte(i)
	}
	return res
}

func TestNewPeppleDB(t *testing.T) {
	db, err := NewPeppleDB(t.TempDir(), 16, 16, "test")
	assert.NoError(t, err)
	defer db.Close()

	assert.NotNil(t, db)
}

func setupTestStorage(t *testing.T) storage.ContentStorage {
	db, err := NewPeppleDB(t.TempDir(), 16, 16, "test")
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	config := PeppleStorageConfig{
		StorageCapacityMB: 1,
		DB:                db,
		NodeId:            uint256.NewInt(0).Bytes32(),
		NetworkName:       "test",
	}

	storage, err := NewPeppleStorage(config)
	assert.NoError(t, err)
	return storage
}

func TestContentStoragePutAndGet(t *testing.T) {
	db := setupTestStorage(t)

	testCases := []struct {
		contentKey []byte
		contentId  []byte
		content    []byte
	}{
		{[]byte("key1"), []byte("id1"), []byte("content1")},
		{[]byte("key2"), []byte("id2"), []byte("content2")},
	}

	for _, tc := range testCases {
		err := db.Put(tc.contentKey, tc.contentId, tc.content)
		assert.NoError(t, err)

		got, err := db.Get(tc.contentKey, tc.contentId)
		assert.NoError(t, err)
		assert.Equal(t, tc.content, got)
	}
}

func TestRadius(t *testing.T) {
	db := setupTestStorage(t)
	radius := db.Radius()
	assert.NotNil(t, radius)
	assert.True(t, radius.Eq(storage.MaxDistance))
}

func TestXOR(t *testing.T) {
	testCases := []struct {
		contentId []byte
		nodeId    []byte
		expected  []byte
	}{
		{
			contentId: []byte{0x01},
			nodeId:    make([]byte, 32),
			expected:  append([]byte{0x01}, make([]byte, 31)...),
		},
		{
			contentId: []byte{0xFF},
			nodeId:    []byte{0x0F},
			expected:  []byte{0xF0},
		},
	}

	for _, tc := range testCases {
		result := xor(tc.contentId, tc.nodeId)
		assert.Equal(t, tc.expected, result)
	}
}

// the capacity is 1MB, so prune will delete over 50Kb content
func TestPrune(t *testing.T) {
	db := setupTestStorage(t)
	// the nodeId is zeros, so contentKey and contentId is the same
	testcases := []struct {
		contentKey  [32]byte
		content     []byte
		outOfRadius bool
		err         error
	}{
		{
			contentKey: uint256.NewInt(1).Bytes32(),
			content:    genBytes(900_000),
		},
		{
			contentKey: uint256.NewInt(2).Bytes32(),
			content:    genBytes(40_000),
		},
		{
			contentKey: uint256.NewInt(3).Bytes32(),
			content:    genBytes(20_000),
			err:        storage.ErrContentNotFound,
		},
		{
			contentKey: uint256.NewInt(4).Bytes32(),
			content:    genBytes(20_000),
			err:        storage.ErrContentNotFound,
		},
		{
			contentKey: uint256.NewInt(5).Bytes32(),
			content:    genBytes(20_000),
			err:        storage.ErrContentNotFound,
		},
		{
			contentKey:  uint256.NewInt(6).Bytes32(),
			content:     genBytes(20_000),
			err:         storage.ErrInsufficientRadius,
			outOfRadius: true,
		},
		{
			contentKey:  uint256.NewInt(7).Bytes32(),
			content:     genBytes(20_000),
			err:         storage.ErrInsufficientRadius,
			outOfRadius: true,
		},
	}

	for _, val := range testcases {
		err := db.Put(val.contentKey[:], val.contentKey[:], val.content)
		if err != nil {
			assert.Equal(t, val.err, err)
		}
	}
	for _, val := range testcases {
		content, err := db.Get(val.contentKey[:], val.contentKey[:])
		if err == nil {
			assert.Equal(t, val.content, content)
		} else if !val.outOfRadius {
			assert.Equal(t, val.err, err)
		}
	}
	radius := db.Radius()
	data, err := radius.MarshalSSZ()
	assert.NoError(t, err)
	actual := uint256.NewInt(2).Bytes32()
	assert.Equal(t, data, actual[:])
}
