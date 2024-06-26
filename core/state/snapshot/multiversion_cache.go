package snapshot

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const (
	batchGCNumber = 100
)

type multiVersionItem interface {
	Version() uint64
}

type multiVersionItemSlice[T multiVersionItem] struct {
	data []T
}

func newMultiVersionItemSlice[T multiVersionItem](inputData []T) *multiVersionItemSlice[T] {
	c := multiVersionItemSlice[T]{}
	c.data = inputData
	return &c
}

// SortByVersion is used to resort the multi-version slice by version.
func (a *multiVersionItemSlice[T]) SortByVersion() []T {
	sort.Sort(a)
	return a.data
}

func (a *multiVersionItemSlice[T]) Len() int {
	return len(a.data)
}

func (a *multiVersionItemSlice[T]) Swap(i, j int) {
	a.data[i], a.data[j] = a.data[j], a.data[i]
}

func (a *multiVersionItemSlice[T]) Less(i, j int) bool {
	return a.data[j].Version() > a.data[i].Version()
}

type destructCacheItem struct {
	version uint64
	root    common.Hash
}

func (di *destructCacheItem) Version() uint64 {
	return di.version
}

var _ multiVersionItem = &destructCacheItem{}

type accountCacheItem struct {
	version uint64
	root    common.Hash
	data    []byte
}

func (ai *accountCacheItem) Version() uint64 {
	return ai.version
}

var _ multiVersionItem = &accountCacheItem{}

type storageCacheItem struct {
	version uint64
	root    common.Hash
	data    []byte
}

func (si *storageCacheItem) Version() uint64 {
	return si.version
}

var _ multiVersionItem = &storageCacheItem{}

func cloneParentMap(parentMap map[common.Hash]struct{}) map[common.Hash]struct{} {
	cloneMap := make(map[common.Hash]struct{})
	for k := range parentMap {
		cloneMap[k] = struct{}{}
	}
	return cloneMap
}

type MultiVersionSnapshotCache struct {
	lock             sync.RWMutex
	destructCache    map[common.Hash][]*destructCacheItem
	accountDataCache map[common.Hash][]*accountCacheItem
	storageDataCache map[common.Hash]map[common.Hash][]*storageCacheItem
	minVersion       uint64 // bottom version
	diffLayerParent  map[common.Hash]map[common.Hash]struct{}
	cacheItemNumber  int64

	deltaRemoveQueue []*diffLayer
}

func NewMultiVersionSnapshotCache() *MultiVersionSnapshotCache {
	c := &MultiVersionSnapshotCache{
		destructCache:    make(map[common.Hash][]*destructCacheItem),
		accountDataCache: make(map[common.Hash][]*accountCacheItem),
		storageDataCache: make(map[common.Hash]map[common.Hash][]*storageCacheItem),
		minVersion:       0,
		diffLayerParent:  make(map[common.Hash]map[common.Hash]struct{}),
		cacheItemNumber:  0,
	}
	go c.loopDelayGC()
	return c
}

func (c *MultiVersionSnapshotCache) checkParent(childRoot common.Hash, parentRoot common.Hash) bool {
	if c == nil {
		return false
	}
	if _, exist := c.diffLayerParent[childRoot]; !exist {
		return false
	}
	if _, exist := c.diffLayerParent[childRoot][parentRoot]; !exist {
		return false
	}
	return true
}

func (c *MultiVersionSnapshotCache) Add(ly *diffLayer) {
	if c == nil || ly == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	log.Info("Add difflayer to snapshot multiversion cache", "root", ly.root, "version_id", ly.diffLayerID, "current_cache_item_number", c.cacheItemNumber)

	for hash := range ly.destructSet {
		if multiVersionItems, exist := c.destructCache[hash]; exist {
			multiVersionItems = append(multiVersionItems, &destructCacheItem{version: ly.diffLayerID, root: ly.root})
			c.destructCache[hash] = multiVersionItems
		} else {
			c.destructCache[hash] = []*destructCacheItem{&destructCacheItem{version: ly.diffLayerID, root: ly.root}}
		}
		c.cacheItemNumber++
		//log.Info("Add destruct to cache",
		//	"cache_account_hash", hash,
		//	"cache_version", ly.diffLayerID,
		//	"cache_root", ly.root)
	}
	// sorted by version
	for hash := range c.destructCache {
		c.destructCache[hash] = newMultiVersionItemSlice[*destructCacheItem](c.destructCache[hash]).SortByVersion()
	}

	for hash, aData := range ly.accountData {
		if multiVersionItems, exist := c.accountDataCache[hash]; exist {
			multiVersionItems = append(multiVersionItems, &accountCacheItem{version: ly.diffLayerID, root: ly.root, data: aData})
			c.accountDataCache[hash] = multiVersionItems
		} else {
			c.accountDataCache[hash] = []*accountCacheItem{&accountCacheItem{version: ly.diffLayerID, root: ly.root, data: aData}}
		}
		c.cacheItemNumber++
		//log.Info("Add account to cache",
		//	"cache_account_hash", hash,
		//	"cache_version", ly.diffLayerID,
		//	"cache_root", ly.root,
		//	"cache_data_len", len(aData))
	}

	// sorted by version
	for hash := range c.accountDataCache {
		c.accountDataCache[hash] = newMultiVersionItemSlice[*accountCacheItem](c.accountDataCache[hash]).SortByVersion()
	}

	for accountHash, slots := range ly.storageData {
		if _, exist := c.storageDataCache[accountHash]; !exist {
			c.storageDataCache[accountHash] = make(map[common.Hash][]*storageCacheItem)
		}
		for storageHash, sData := range slots {
			if multiVersionItems, exist := c.storageDataCache[accountHash][storageHash]; exist {
				multiVersionItems = append(multiVersionItems, &storageCacheItem{version: ly.diffLayerID, root: ly.root, data: sData})
				c.storageDataCache[accountHash][storageHash] = multiVersionItems
			} else {
				c.storageDataCache[accountHash][storageHash] = []*storageCacheItem{&storageCacheItem{version: ly.diffLayerID, root: ly.root, data: sData}}
			}
			c.cacheItemNumber++
			//log.Info("Add storage to cache",
			//	"cache_account_hash", accountHash,
			//	"cache_storage_hash", storageHash,
			//	"cache_version", ly.diffLayerID,
			//	"cache_root", ly.root,
			//	"cache_data_len", len(sData))
		}
	}
	// sorted by version
	for ahash := range c.storageDataCache {
		for shash := range c.storageDataCache[ahash] {
			c.storageDataCache[ahash][shash] = newMultiVersionItemSlice[*storageCacheItem](c.storageDataCache[ahash][shash]).SortByVersion()
		}
	}

	if parentDiffLayer, ok := ly.parent.(*diffLayer); ok {
		if parentLayerParent, exist := c.diffLayerParent[parentDiffLayer.root]; exist {
			clonedParentLayerParent := cloneParentMap(parentLayerParent)
			clonedParentLayerParent[ly.root] = struct{}{}
			c.diffLayerParent[ly.root] = clonedParentLayerParent
		} else {
			log.Warn("Impossible branch, maybe there is a bug.")
		}
	} else {
		c.diffLayerParent[ly.root] = make(map[common.Hash]struct{})
		c.diffLayerParent[ly.root][ly.root] = struct{}{}
	}
	diffMultiVersionCacheLengthGauge.Update(c.cacheItemNumber)
}

func (c *MultiVersionSnapshotCache) loopDelayGC() {
	if c == nil {
		return
	}

	gcTicker := time.NewTicker(time.Second * 1)
	defer gcTicker.Stop()
	for {
		select {
		case <-gcTicker.C:
			c.lock.RLock()
			deltaQueueLen := len(c.deltaRemoveQueue)
			c.lock.RUnlock()
			if deltaQueueLen > 500 {
				c.lock.Lock()
				gcDifflayer := c.deltaRemoveQueue[batchGCNumber]
				if gcDifflayer.diffLayerID > c.minVersion {
					c.minVersion = gcDifflayer.diffLayerID
				}
				log.Info("Delay remove difflayer from snapshot multiversion cache",
					"root", gcDifflayer.root,
					"version_id", gcDifflayer.diffLayerID,
					"current_cache_item_number", c.cacheItemNumber,
					"deleted_difflayer_number", deltaQueueLen,
					"min_version", c.minVersion)

				for aHash, multiVersionDestructList := range c.destructCache {
					for i := 0; i < len(multiVersionDestructList); i++ {
						if multiVersionDestructList[i].version < c.minVersion {
							//log.Info("Remove destruct from cache",
							//	"cache_account_hash", aHash,
							//	"cache_version", multiVersionDestructList[i].version,
							//	"cache_root", multiVersionDestructList[i].root,
							//	"min_version", c.minVersion,
							//	"gc_diff_root", gcDifflayer.root,
							//	"gc_diff_version", gcDifflayer.diffLayerID)
							multiVersionDestructList = append(multiVersionDestructList[:i], multiVersionDestructList[i+1:]...)
							i--
							c.cacheItemNumber--
						}
					}
					if len(multiVersionDestructList) == 0 {
						delete(c.destructCache, aHash)
					} else {
						c.destructCache[aHash] = multiVersionDestructList
					}
				}

				for aHash, multiVersionAccoutList := range c.accountDataCache {
					for i := 0; i < len(multiVersionAccoutList); i++ {
						if multiVersionAccoutList[i].version < c.minVersion {
							//log.Info("Remove account from cache",
							//	"cache_account_hash", aHash,
							//	"cache_version", multiVersionAccoutList[i].version,
							//	"cache_root", multiVersionAccoutList[i].root,
							//	"cache_data_len", len(multiVersionAccoutList[i].data),
							//	"min_version", c.minVersion,
							//	"gc_diff_root", gcDifflayer.root,
							//	"gc_diff_version", gcDifflayer.diffLayerID)
							multiVersionAccoutList = append(multiVersionAccoutList[:i], multiVersionAccoutList[i+1:]...)
							i--
							c.cacheItemNumber--
						}
					}
					if len(multiVersionAccoutList) == 0 {
						delete(c.accountDataCache, aHash)
					} else {
						c.accountDataCache[aHash] = multiVersionAccoutList
					}
				}
				for aHash := range c.storageDataCache {
					for sHash, multiVersionStorageList := range c.storageDataCache[aHash] {
						for i := 0; i < len(multiVersionStorageList); i++ {
							if multiVersionStorageList[i].version < c.minVersion {
								//log.Info("Remove storage from cache",
								//	"cache_account_hash", aHash,
								//	"cache_storage_hash", sHash,
								//	"cache_version", multiVersionStorageList[i].version,
								//	"cache_root", multiVersionStorageList[i].root,
								//	"cache_data_len", len(multiVersionStorageList[i].data),
								//	"min_version", c.minVersion,
								//	"gc_diff_root", gcDifflayer.root,
								//	"gc_diff_version", gcDifflayer.diffLayerID)
								multiVersionStorageList = append(multiVersionStorageList[:i], multiVersionStorageList[i+1:]...)
								i--
								c.cacheItemNumber--
							}
						}
						if len(multiVersionStorageList) == 0 {
							delete(c.storageDataCache[aHash], sHash)
						} else {
							c.storageDataCache[aHash][sHash] = multiVersionStorageList
						}
					}
					if len(c.storageDataCache[aHash]) == 0 {
						delete(c.storageDataCache, aHash)
					}
				}

				for i := 0; i <= batchGCNumber; i++ {
					toGCDifflayerRoot := c.deltaRemoveQueue[i].root
					delete(c.diffLayerParent, toGCDifflayerRoot)
					for _, v := range c.diffLayerParent {
						delete(v, toGCDifflayerRoot)
					}
				}

				diffMultiVersionCacheLengthGauge.Update(c.cacheItemNumber)
				c.deltaRemoveQueue = c.deltaRemoveQueue[batchGCNumber+1:]
				c.lock.Unlock()
			} else {
				log.Info("Skip delay gc due to less difflayer in queue", "deleted_difflayer_number", deltaQueueLen)
			}
		}
	}
}

func (c *MultiVersionSnapshotCache) Remove(ly *diffLayer) {
	if c == nil || ly == nil {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.deltaRemoveQueue = append(c.deltaRemoveQueue, ly)
}

// QueryAccount return tuple(data-slice, need-try-disklayer, error)
func (c *MultiVersionSnapshotCache) QueryAccount(version uint64, rootHash common.Hash, ahash common.Hash) ([]byte, bool, error) {
	if c == nil {
		return nil, false, fmt.Errorf("not found, need try difflayer")
	}
	c.lock.RLock()
	defer c.lock.RUnlock()

	var (
		queryAccountItem  *accountCacheItem
		queryDestructItem *destructCacheItem
	)

	{
		if multiVersionItems, exist := c.accountDataCache[ahash]; exist && len(multiVersionItems) != 0 {
			//log.Info("Try query account cache",
			//	"query_version", version,
			//	"query_root_hash", rootHash,
			//	"query_account_hash", ahash,
			//	"multi_version_cache_len", len(multiVersionItems))
			for i := len(multiVersionItems) - 1; i >= 0; i-- {
				if multiVersionItems[i].version <= version &&
					multiVersionItems[i].version > c.minVersion &&
					c.checkParent(rootHash, multiVersionItems[i].root) {
					queryAccountItem = multiVersionItems[i]
					//log.Info("Account hit account cache",
					//	"query_version", version,
					//	"query_root_hash", rootHash,
					//	"query_account_hash", ahash,
					//	"hit_version", queryAccountItem.version,
					//	"hit_root_hash", queryAccountItem.root)
					break
				}
				//log.Info("Try hit account cache",
				//	"query_version", version,
				//	"query_root_hash", rootHash,
				//	"query_account_hash", ahash,
				//	"try_hit_version", multiVersionItems[i].version,
				//	"try_hit_root_hash", multiVersionItems[i].root,
				//	"check_version", multiVersionItems[i].version > c.minVersion,
				//	"check_parent", c.checkParent(rootHash, multiVersionItems[i].root),
				//	"check_data_len", len(multiVersionItems[i].data))
			}
		}
	}

	{
		if multiVersionItems, exist := c.destructCache[ahash]; exist && len(multiVersionItems) != 0 {
			//log.Info("Try query destruct cache",
			//	"query_version", version,
			//	"query_root_hash", rootHash,
			//	"query_account_hash", ahash,
			//	"multi_version_cache_len", len(multiVersionItems))
			for i := len(multiVersionItems) - 1; i >= 0; i-- {
				if multiVersionItems[i].version <= version &&
					multiVersionItems[i].version > c.minVersion &&
					c.checkParent(rootHash, multiVersionItems[i].root) {
					queryDestructItem = multiVersionItems[i]
					//log.Info("Account hit destruct cache",
					//	"query_version", version,
					//	"query_root_hash", rootHash,
					//	"query_account_hash", ahash,
					//	"hit_version", queryDestructItem.version,
					//	"hit_root_hash", queryDestructItem.root)
					break
				}
				//log.Info("Try hit destruct cache",
				//	"query_version", version,
				//	"query_root_hash", rootHash,
				//	"query_account_hash", ahash,
				//	"hit_version", multiVersionItems[i].version,
				//	"hit_root_hash", multiVersionItems[i].root)
			}
		}
	}
	if queryAccountItem != nil && queryDestructItem == nil {
		return queryAccountItem.data, false, nil // founded
	}

	if queryAccountItem == nil && queryDestructItem != nil {
		return nil, false, nil // deleted
	}

	if queryAccountItem == nil && queryDestructItem == nil {
		return nil, true, nil
	}

	// queryAccountItem != nil && queryDestructItem != nil
	if queryAccountItem.version >= queryDestructItem.version {
		return queryAccountItem.data, false, nil // founded
	} else {
		return nil, false, nil // deleted
	}
}

// QueryStorage return tuple(data-slice, need-try-disklayer, error)
func (c *MultiVersionSnapshotCache) QueryStorage(version uint64, rootHash common.Hash, ahash common.Hash, shash common.Hash) ([]byte, bool, error) {
	if c == nil {
		return nil, false, fmt.Errorf("not found, need try difflayer")
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	var (
		queryStorageItem  *storageCacheItem
		queryDestructItem *destructCacheItem
	)

	{
		if _, exist := c.storageDataCache[ahash]; exist {
			if multiVersionItems, exist2 := c.storageDataCache[ahash][shash]; exist2 && len(multiVersionItems) != 0 {
				//log.Info("Try query storage cache",
				//	"query_version", version,
				//	"query_root_hash", rootHash,
				//	"query_account_hash", ahash,
				//	"query_storage_hash", shash,
				//	"multi_version_cache_len", len(multiVersionItems))
				for i := len(multiVersionItems) - 1; i >= 0; i-- {
					if multiVersionItems[i].version <= version &&
						multiVersionItems[i].version > c.minVersion &&
						c.checkParent(rootHash, multiVersionItems[i].root) {
						queryStorageItem = multiVersionItems[i]
						//log.Info("Account hit storage cache",
						//	"query_version", version,
						//	"query_root_hash", rootHash,
						//	"query_account_hash", ahash,
						//	"query_storage_hash", shash,
						//	"hit_version", queryStorageItem.version,
						//	"hit_root_hash", queryStorageItem.root)
						break
					}
					//log.Info("Try hit storage cache",
					//	"query_version", version,
					//	"query_root_hash", rootHash,
					//	"query_account_hash", ahash,
					//	"query_storage_hash", shash,
					//	"hit_version", multiVersionItems[i].version,
					//	"hit_root_hash", multiVersionItems[i].root)
				}
			}
		}
	}

	{
		if multiVersionItems, exist := c.destructCache[ahash]; exist && len(multiVersionItems) != 0 {
			//log.Info("Try query destruct cache",
			//	"query_version", version,
			//	"query_root_hash", rootHash,
			//	"query_account_hash", ahash,
			//	"query_storage_hash", shash,
			//	"multi_version_cache_len", len(multiVersionItems))
			for i := len(multiVersionItems) - 1; i >= 0; i-- {
				if multiVersionItems[i].version <= version &&
					multiVersionItems[i].version > c.minVersion &&
					c.checkParent(rootHash, multiVersionItems[i].root) {
					queryDestructItem = multiVersionItems[i]
					//log.Info("Account hit destruct cache",
					//	"query_version", version,
					//	"query_root_hash", rootHash,
					//	"query_account_hash", ahash,
					//	"query_storage_hash", shash,
					//	"hit_version", queryDestructItem.version,
					//	"hit_root_hash", queryDestructItem.root)
					break
				}
				//log.Info("Try hit destruct cache",
				//	"query_version", version,
				//	"query_root_hash", rootHash,
				//	"query_account_hash", ahash,
				//	"query_storage_hash", shash,
				//	"hit_version", multiVersionItems[i].version,
				//	"hit_root_hash", multiVersionItems[i].root)
			}
		}
	}

	if queryStorageItem != nil && queryDestructItem == nil {
		return queryStorageItem.data, false, nil // founded
	}

	if queryStorageItem == nil && queryDestructItem != nil {
		return nil, false, nil // deleted
	}

	if queryStorageItem == nil && queryDestructItem == nil {
		return nil, true, nil // not founded and need try disklayer
	}

	// queryStorageItem != nil && queryDestructItem != nil
	if queryStorageItem.version >= queryDestructItem.version {
		return queryStorageItem.data, false, nil // founded
	} else {
		return nil, false, nil // deleted
	}
}
