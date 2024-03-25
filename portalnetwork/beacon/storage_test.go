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
	"github.com/ethereum/go-ethereum/portalnetwork/utils"
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
	key, value, err := getClientBootstrap()
	require.NoError(t, err)

	contentId := defaultContentIdFunc(key)
	_, err = beaconStorage.Get(key, contentId)
	require.Equal(t, storage.ErrContentNotFound, err)

	err = beaconStorage.Put(key, contentId, value)
	require.NoError(t, err)

	res, err := beaconStorage.Get(key, contentId)
	require.NoError(t, err)
	require.Equal(t, value, res)

	key, value, err = getClientUpdatesByRange()
	require.NoError(t, err)

	contentId = defaultContentIdFunc(key)
	_, err = beaconStorage.Get(key, contentId)
	require.Equal(t, storage.ErrContentNotFound, err)

	err = beaconStorage.Put(key, contentId, value)
	require.NoError(t, err)

	res, err = beaconStorage.Get(key, contentId)
	require.NoError(t, err)
	require.Equal(t, value, res)
}

func genStorage(testDir string) (storage.ContentStorage, error) {
	utils.EnsureDir(testDir)
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

func getClientBootstrap() ([]byte, []byte, error) {
	filePath := "testdata/light_client_bootstrap.json"

	f, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	var result map[string]map[string]string
	err = json.Unmarshal(f, &result)
	if err != nil {
		return nil, nil, err
	}
	contentKey := hexutil.MustDecode(result["6718368"]["content_key"])
	contentValue := hexutil.MustDecode(result["6718368"]["content_value"])
	return contentKey, contentValue, nil
}

func getClientUpdatesByRange() ([]byte, []byte, error) {
	filePath := "testdata/light_client_updates_by_range.json"

	f, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	var result map[string]map[string]string
	err = json.Unmarshal(f, &result)
	if err != nil {
		return nil, nil, err
	}
	contentKey := hexutil.MustDecode(result["6684738"]["content_key"])
	contentValue := hexutil.MustDecode(result["6684738"]["content_value"])
	return contentKey, contentValue, nil
}

func clearNodeData(nodeDataDir string) {
	err := os.Remove(fmt.Sprintf("%s%s", nodeDataDir, dbName))
	if err != nil {
		fmt.Println(err)
	}
}
