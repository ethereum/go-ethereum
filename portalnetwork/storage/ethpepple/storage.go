package ethpepple

import (
	"encoding/binary"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
)

const (
	// minCache is the minimum amount of memory in megabytes to allocate to pebble
	// read and write caching, split half and half.
	minCache = 16

	// minHandles is the minimum number of files handles to allocate to the open
	// database files.
	minHandles = 16

	// 5% of the content will be deleted when the storage capacity is hit and radius gets adjusted.
	contentDeletionFraction = 0.05
)

var _ storage.ContentStorage = &ContentStorage{}

type PeppleStorageConfig struct {
	StorageCapacityMB uint64
	DB                *pebble.DB
	NodeId            enode.ID
	NetworkName       string
}

func NewPeppleDB(dataDir string, cache, handles int, namespace string) (*pebble.DB, error) {
	// Ensure we have some minimal caching and file guarantees
	if cache < minCache {
		cache = minCache
	}
	if handles < minHandles {
		handles = minHandles
	}
	logger := log.New("database", namespace)
	logger.Info("Allocated cache and file handles", "cache", common.StorageSize(cache*1024*1024), "handles", handles)

	// The max memtable size is limited by the uint32 offsets stored in
	// internal/arenaskl.node, DeferredBatchOp, and flushableBatchEntry.
	//
	// - MaxUint32 on 64-bit platforms;
	// - MaxInt on 32-bit platforms.
	//
	// It is used when slices are limited to Uint32 on 64-bit platforms (the
	// length limit for slices is naturally MaxInt on 32-bit platforms).
	//
	// Taken from https://github.com/cockroachdb/pebble/blob/master/internal/constants/constants.go
	maxMemTableSize := (1<<31)<<(^uint(0)>>63) - 1

	// Two memory tables is configured which is identical to leveldb,
	// including a frozen memory table and another live one.
	memTableLimit := 2
	memTableSize := cache * 1024 * 1024 / 2 / memTableLimit

	// The memory table size is currently capped at maxMemTableSize-1 due to a
	// known bug in the pebble where maxMemTableSize is not recognized as a
	// valid size.
	//
	// TODO use the maxMemTableSize as the maximum table size once the issue
	// in pebble is fixed.
	if memTableSize >= maxMemTableSize {
		memTableSize = maxMemTableSize - 1
	}
	opt := &pebble.Options{
		// Pebble has a single combined cache area and the write
		// buffers are taken from this too. Assign all available
		// memory allowance for cache.
		Cache:        pebble.NewCache(int64(cache * 1024 * 1024)),
		MaxOpenFiles: handles,

		// The size of memory table(as well as the write buffer).
		// Note, there may have more than two memory tables in the system.
		MemTableSize: uint64(memTableSize),

		// MemTableStopWritesThreshold places a hard limit on the size
		// of the existent MemTables(including the frozen one).
		// Note, this must be the number of tables not the size of all memtables
		// according to https://github.com/cockroachdb/pebble/blob/master/options.go#L738-L742
		// and to https://github.com/cockroachdb/pebble/blob/master/db.go#L1892-L1903.
		MemTableStopWritesThreshold: memTableLimit,

		// The default compaction concurrency(1 thread),
		// Here use all available CPUs for faster compaction.
		MaxConcurrentCompactions: runtime.NumCPU,

		// Per-level options. Options for at least one level must be specified. The
		// options for the last level are used for all subsequent levels.
		Levels: []pebble.LevelOptions{
			{TargetFileSize: 2 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 4 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 8 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 16 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 32 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 64 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
			{TargetFileSize: 128 * 1024 * 1024, FilterPolicy: bloom.FilterPolicy(10)},
		},
		ReadOnly: false,
	}
	// Disable seek compaction explicitly. Check https://github.com/ethereum/go-ethereum/pull/20130
	// for more details.
	opt.Experimental.ReadSamplingMultiplier = -1
	db, err := pebble.Open(dataDir+"/"+namespace, opt)
	return db, err
}

type ContentStorage struct {
	nodeId                 enode.ID
	storageCapacityInBytes uint64
	radius                 atomic.Value
	log                    log.Logger
	db                     *pebble.DB
	size                   uint64
	sizeChan               chan uint64
	sizeMutex              sync.RWMutex
	isPruning              bool
	pruneDoneChan          chan uint64 // finish prune and get the pruned size
	writeOptions           *pebble.WriteOptions
}

func NewPeppleStorage(config PeppleStorageConfig) (storage.ContentStorage, error) {
	cs := &ContentStorage{
		nodeId:                 config.NodeId,
		db:                     config.DB,
		storageCapacityInBytes: config.StorageCapacityMB * 1000_000,
		log:                    log.New("storage", config.NetworkName),
		sizeChan:               make(chan uint64, 100),
		pruneDoneChan:          make(chan uint64, 1),
		writeOptions:           &pebble.WriteOptions{Sync: false},
	}
	cs.radius.Store(storage.MaxDistance)
	radius, _, err := cs.db.Get(storage.RadisuKey)
	if err != nil && err != pebble.ErrNotFound {
		return nil, err
	}
	if err == nil {
		dis := uint256.NewInt(0)
		err = dis.UnmarshalSSZ(radius)
		if err != nil {
			return nil, err
		}
		cs.radius.Store(dis)
	}

	val, _, err := cs.db.Get(storage.SizeKey)
	if err != nil && err != pebble.ErrNotFound {
		return nil, err
	}
	if err == nil {
		size := binary.BigEndian.Uint64(val)
		// init stage, no need to use lock
		cs.size = size
	}
	go cs.saveCapacity()
	return cs, nil
}

// Get implements storage.ContentStorage.
func (c *ContentStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	distance := xor(contentId, c.nodeId[:])
	data, closer, err := c.db.Get(distance)
	if err != nil && err != pebble.ErrNotFound {
		return nil, err
	}
	if err == pebble.ErrNotFound {
		return nil, storage.ErrContentNotFound
	}
	closer.Close()
	return data, nil
}

// Put implements storage.ContentStorage.
func (c *ContentStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	length := uint64(len(contentId)) + uint64(len(content))
	c.sizeChan <- length
	distance := xor(contentId, c.nodeId[:])
	return c.db.Set(distance, content, c.writeOptions)
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
				err := c.db.Set(storage.SizeKey, buf, c.writeOptions)
				if err != nil {
					c.log.Error("save capacity failed", "error", err)
				}
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
	expectSize := uint64(float64(c.storageCapacityInBytes) * contentDeletionFraction)
	var curentSize uint64 = 0

	defer func() {
		c.pruneDoneChan <- curentSize
	}()
	// get the keys to be deleted order by distance desc
	iter, err := c.db.NewIter(nil)
	if err != nil {
		c.log.Error("get iter failed", "error", err)
		return
	}

	batch := c.db.NewBatch()
	for iter.Last(); iter.Valid(); iter.Prev() {
		if curentSize < expectSize {
			batch.Delete(iter.Key(), nil)
			curentSize += uint64(len(iter.Key())) + uint64(len(iter.Value()))
		} else {
			distance := iter.Key()
			c.db.Set(storage.RadisuKey, distance, c.writeOptions)
			dis := uint256.NewInt(0)
			err = dis.UnmarshalSSZ(distance)
			if err != nil {
				c.log.Error("unmarshal distance failed", "error", err)
			}
			c.radius.Store(dis)
			break
		}
	}
	err = batch.Commit(&pebble.WriteOptions{Sync: true})
	if err != nil {
		c.log.Error("prune batch commit failed", "error", err)
		return
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
