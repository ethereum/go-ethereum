package ethpepple

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

var testRadius = uint256.NewInt(100000)

func genBytes(length int) []byte {
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		res[i] = byte(i)
	}
	return res
}

func getTestDb() (storage.ContentStorage, error) {
	db := memorydb.New()
	config := PeppleStorageConfig{
		DB:                db,
		StorageCapacityMB: 1,
		NodeId:            uint256.NewInt(0).Bytes32(),
		NetworkName:       "history",
	}
	return NewPeppleStorage(config)
}

func TestReadRadius(t *testing.T) {
	db, err := getTestDb()
	assert.NoError(t, err)
	assert.True(t, db.Radius().Eq(storage.MaxDistance))

	data, err := testRadius.MarshalSSZ()
	assert.NoError(t, err)
	db.Put(nil, storage.RadisuKey, data)

	store := db.(*ContentStorage)
	err = store.db.Close()
	assert.NoError(t, err)
}

func TestStorage(t *testing.T) {
	db, err := getTestDb()
	assert.NoError(t, err)
	testcases := map[string][]byte{
		"test1": []byte("test1"),
		"test2": []byte("test2"),
		"test3": []byte("test3"),
		"test4": []byte("test4"),
	}

	for key, value := range testcases {
		db.Put(nil, []byte(key), value)
	}

	for key, value := range testcases {
		val, err := db.Get(nil, []byte(key))
		assert.NoError(t, err)
		assert.Equal(t, value, val)
	}
}

func TestXor(t *testing.T) {
	nodeId := uint256.NewInt(0).Bytes32()
	bs := make([]byte, 32)
	rand.Read(bs)
	dis := xor(bs, nodeId[:])
	assert.Equal(t, bs, dis)

	nodeId2 := uint256.NewInt(2).Bytes32()
	dis = xor(bs, nodeId2[:])
	assert.Equal(t, bs, xor(dis, nodeId2[:]))
}

// the capacity is 1MB, so prune will delete over 50Kb content
func TestPrune(t *testing.T) {
	db, err := getTestDb()
	assert.NoError(t, err)
	// the nodeId is zeros, so contentKey and contentId is the same
	testcases := []struct {
		contentKey  [32]byte
		content     []byte
		shouldPrune bool
	}{
		{
			contentKey:  uint256.NewInt(1).Bytes32(),
			content:     genBytes(900_000),
			shouldPrune: false,
		},
		{
			contentKey:  uint256.NewInt(2).Bytes32(),
			content:     genBytes(40_000),
			shouldPrune: false,
		},
		{
			contentKey:  uint256.NewInt(3).Bytes32(),
			content:     genBytes(20_000),
			shouldPrune: false,
		},
		{
			contentKey:  uint256.NewInt(4).Bytes32(),
			content:     genBytes(20_000),
			shouldPrune: false,
		},
		{
			contentKey:  uint256.NewInt(5).Bytes32(),
			content:     genBytes(20_000),
			shouldPrune: true,
		},
		{
			contentKey:  uint256.NewInt(6).Bytes32(),
			content:     genBytes(20_000),
			shouldPrune: true,
		},
		{
			contentKey:  uint256.NewInt(7).Bytes32(),
			content:     genBytes(20_000),
			shouldPrune: true,
		},
	}

	for _, val := range testcases {
		db.Put(val.contentKey[:], val.contentKey[:], val.content)
	}
	// // wait to prune done
	time.Sleep(5 * time.Second)
	for _, val := range testcases {
		content, err := db.Get(val.contentKey[:], val.contentKey[:])
		if !val.shouldPrune {
			assert.Equal(t, val.content, content)
		} else {
			assert.Error(t, err)
		}
	}
}
