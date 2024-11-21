package ethpepple

import (
	"bytes"
	"container/heap"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
)

const contentDeletionFraction = 0.05 // 5% of the content will be deleted when the storage capacity is hit and radius gets adjusted.

var _ storage.ContentStorage = &ContentStorage{}

type PeppleStorageConfig struct {
	StorageCapacityMB uint64
	DB                ethdb.KeyValueStore
	NodeId            enode.ID
	NetworkName       string
}

func NewPeppleDB(dataDir string, cache, handles int, namespace string) (ethdb.KeyValueStore, error) {
	db, err := pebble.New(dataDir+"/"+namespace, cache, handles, namespace, false)
	return db, err
}

type ContentStorage struct {
	nodeId                 enode.ID
	storageCapacityInBytes uint64
	radius                 atomic.Value
	log                    log.Logger
	db                     ethdb.KeyValueStore
	size                   uint64
	sizeChan               chan uint64
	sizeMutex              sync.RWMutex
	isPruning              bool
	pruneDoneChan          chan uint64 // finish prune and get the pruned size
}

func NewPeppleStorage(config PeppleStorageConfig) (storage.ContentStorage, error) {
	cs := &ContentStorage{
		nodeId:                 config.NodeId,
		db:                     config.DB,
		storageCapacityInBytes: config.StorageCapacityMB * 1000_000,
		log:                    log.New("storage", config.NetworkName),
		sizeChan:               make(chan uint64, 100),
		pruneDoneChan:          make(chan uint64, 1),
	}
	cs.radius.Store(storage.MaxDistance)
	exist, err := cs.db.Has(storage.RadisuKey)
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

	exist, err = cs.db.Has(storage.SizeKey)
	if err != nil {
		return nil, err
	}
	if exist {
		val, err := cs.db.Get(storage.SizeKey)
		if err != nil {
			return nil, err
		}
		size := binary.BigEndian.Uint64(val)
		// init stage, no need to use lock
		cs.size = size
	}
	go cs.saveCapacity()
	return cs, nil
}

// Get implements storage.ContentStorage.
func (c *ContentStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	return c.db.Get(contentId)
}

// Put implements storage.ContentStorage.
func (c *ContentStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	length := uint64(len(contentId)) + uint64(len(content))
	c.sizeChan <- length
	return c.db.Put(contentId, content)
}

// Radius implements storage.ContentStorage.
func (c *ContentStorage) Radius() *uint256.Int {
	radius := c.radius.Load()
	val := radius.(*uint256.Int)
	return val
}

func (c *ContentStorage) saveCapacity() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	sizeChanged := false
	buf := make([]byte, 8) // uint64

	for {
		select {
		case <-ticker.C:
			if sizeChanged {
				binary.BigEndian.PutUint64(buf, c.size)
				c.db.Put(storage.SizeKey, buf)
				sizeChanged = false
			}
		case size := <-c.sizeChan:
			c.log.Debug("reveice size %v", size)
			c.sizeMutex.Lock()
			c.size += size
			c.sizeMutex.Unlock()
			sizeChanged = true
			if c.size > c.storageCapacityInBytes {
				if !c.isPruning {
					c.isPruning = true
					go c.prune()
				}
			}
		case prunedSize := <-c.pruneDoneChan:
			c.isPruning = false
			c.size -= prunedSize
			sizeChanged = true
		}
	}
}

func (c *ContentStorage) prune() {
	var distance = []byte{}

	h := &MaxHeap{}
	heap.Init(h)

	expectSize := uint64(float64(c.storageCapacityInBytes) * contentDeletionFraction)

	var curentSize uint64 = 0

	defer func() {
		c.pruneDoneChan <- curentSize
	}()
	// get the keys to be deleted order by distance desc
	iterator := c.db.NewIterator(nil, nil)
	defer iterator.Release()
	for iterator.Next() {
		key := iterator.Key()
		if bytes.Equal(key, storage.SizeKey) || bytes.Equal(key, storage.RadisuKey) {
			continue
		}
		val := iterator.Value()
		size := uint64(len(val))

		distance := xor(key, c.nodeId[:])
		heap.Push(h, Item{
			Distance:  distance,
			ValueSize: size,
		})
		if h.Len() > maxItem {
			heap.Remove(h, h.Len()-1)
		}
	}
	iterator.Release()
	// delete the keys
	for h.Len() > 0 {
		if curentSize > expectSize {
			break
		}
		item := heap.Pop(h)
		val := item.(Item)
		distance = val.Distance
		key := xor(val.Distance, c.nodeId[:])
		if err := c.db.Delete(key); err != nil {
			c.log.Error("failed to delete key %v, err: %v", key, err)
			continue
		}
		curentSize += val.ValueSize
	}

	dis := uint256.NewInt(0)
	err := dis.UnmarshalSSZ(distance)
	if err != nil {
		c.log.Error("failed to parse the radius key %v, err is %v", distance, err)
	}
	c.radius.Store(dis)
	err = c.db.Put(storage.RadisuKey, distance)

	if err != nil {
		c.log.Error("failed to save the radius key %v, err is %v", distance, err)
	}
}

func xor(contentId, nodeId []byte) []byte {
	// length of contentId maybe not 32bytes
	padding := make([]byte, 32)
	if len(contentId) != len(nodeId) {
		copy(padding, contentId)
	} else {
		padding = contentId
	}
	res := make([]byte, len(padding))
	for i := range padding {
		res[i] = padding[i] ^ nodeId[i]
	}
	return res
}
