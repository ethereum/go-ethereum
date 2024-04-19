package beacon

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	_ "github.com/mattn/go-sqlite3"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/stretchr/testify/require"
)

var zeroNodeId = uint256.NewInt(0).Bytes32()

const dbName = "beacon.sqlite"

func defaultContentIdFunc(contentKey []byte) []byte {
	digest := sha256.Sum256(contentKey)
	return digest[:]
}

func TestGetAndPut(t *testing.T) {
	testDir := "./"
	beaconStorage, err := genStorage(testDir)
	require.NoError(t, err)
	defer clearNodeData(testDir)

	testData, err := getTestData()
	require.NoError(t, err)

	for _, entry := range testData {
		key := entry.key
		value := entry.value

		contentId := defaultContentIdFunc(key)
		_, err = beaconStorage.Get(key, contentId)
		require.Equal(t, storage.ErrContentNotFound, err)

		err = beaconStorage.Put(key, contentId, value)
		require.NoError(t, err)

		res, err := beaconStorage.Get(key, contentId)
		require.NoError(t, err)
		require.Equal(t, value, res)
	}
}

func genStorage(testDir string) (storage.ContentStorage, error) {
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path.Join(testDir, dbName))
	if err != nil {
		return nil, err
	}
	config := &storage.PortalStorageConfig{
		StorageCapacityMB: 1000,
		DB:                db,
		NodeId:            enode.ID(zeroNodeId),
		Spec:              configs.Mainnet,
	}
	return NewBeaconStorage(*config)
}

type entry struct {
	key   []byte
	value []byte
}

func getTestData() ([]entry, error) {
	baseDir := "./testdata"
	items, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	entries := make([]entry, 0)

	for _, item := range items {
		if !item.IsDir() {
			f, err := os.ReadFile(fmt.Sprintf("%s/%s", baseDir, item.Name()))
			if err != nil {
				return nil, err
			}
			var result map[string]map[string]string
			err = json.Unmarshal(f, &result)
			if err != nil {
				return nil, err
			}
			for _, v := range result {
				entries = append(entries, entry{
					key:   hexutil.MustDecode(v["content_key"]),
					value: hexutil.MustDecode(v["content_value"]),
				})
			}
		}
	}
	return entries, nil
}

func clearNodeData(nodeDataDir string) {
	err := os.Remove(fmt.Sprintf("%s%s", nodeDataDir, dbName))
	if err != nil {
		fmt.Println(err)
	}
}
