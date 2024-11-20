package ethpepple

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

const dataDir = "./node1"

var testRadius = uint256.NewInt(100000)

func clearNodeData() {
	_ = os.RemoveAll(dataDir)
}

func getTestDb() (storage.ContentStorage, error) {
	db, err := NewPeppleDB(dataDir, 100, 100, "history")
	if err != nil {
		return nil, err
	}
	config := PeppleStorageConfig{
		DB:                db,
		StorageCapacityMB: 100,
		NodeId:            enode.ID{},
		NetworkName:       "history",
	}
	return NewPeppleStorage(config)
}

func TestReadRadius(t *testing.T) {
	db, err := getTestDb()
	assert.NoError(t, err)
	defer clearNodeData()
	assert.True(t, db.Radius().Eq(storage.MaxDistance))

	data, err := testRadius.MarshalSSZ()
	assert.NoError(t, err)
	db.Put(nil, storage.RadisuKey, data)

	store := db.(*ContentStorage)
	err = store.db.Close()
	assert.NoError(t, err)

	db, err = getTestDb()
	assert.NoError(t, err)
	assert.True(t, db.Radius().Eq(testRadius))
}

func TestStorage(t *testing.T) {
	db, err := getTestDb()
	assert.NoError(t, err)
	defer clearNodeData()
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
