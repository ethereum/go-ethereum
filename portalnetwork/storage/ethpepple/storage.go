package ethpepple

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
)

var _ storage.ContentStorage = &ContentStorage{}

type PeppleStorageConfig struct {
	StorageCapacityMB uint64
	DB                ethdb.KeyValueStore
	NodeId            enode.ID
	NetworkName       string
}

func NewPeppleDB(dataDir string, cache, handles int, namespace string) (ethdb.KeyValueStore, error) {
	db, err := pebble.New(dataDir + "/" + namespace, cache, handles, namespace, false)
	return db, err
}

type ContentStorage struct {
	nodeId                 enode.ID
	storageCapacityInBytes uint64
	radius                 atomic.Value
	// size 									 uint64
	log                    log.Logger
	db ethdb.KeyValueStore
}

func NewPeppleStorage(config PeppleStorageConfig) (storage.ContentStorage, error) {
	cs := &ContentStorage{
		nodeId:                 config.NodeId,
		db:               config.DB,
		storageCapacityInBytes: config.StorageCapacityMB * 1000_000,
		log:                    log.New("storage", config.NetworkName),
	}
	cs.radius.Store(storage.MaxDistance)
	exist, err := cs.db.Has(storage.RadisuKey); 
	if err != nil {
		return nil, err
	}
	if exist {
		radius, err := cs.db.Get(storage.RadisuKey)
		if err != nil {
			return nil, err
		}
		dis := uint256.NewInt(0)
		err = dis.UnmarshalSSZ(radius)
		if err != nil {
			return nil, err
		}
		cs.radius.Store(dis)
	}
	return cs, nil
}

// Get implements storage.ContentStorage.
func (c *ContentStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	return c.db.Get(contentId)
}

// Put implements storage.ContentStorage.
func (c *ContentStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	return c.db.Put(contentId, content)
}

// Radius implements storage.ContentStorage.
func (p *ContentStorage) Radius() *uint256.Int {
	radius := p.radius.Load()
	val := radius.(*uint256.Int)
	return val
}
