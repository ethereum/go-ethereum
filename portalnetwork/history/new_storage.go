package history

//
//import (
//	"encoding/binary"
//	"errors"
//	"fmt"
//	"path"
//	"sync/atomic"
//
//	"github.com/cockroachdb/pebble"
//	"github.com/ethereum/go-ethereum/log"
//	"github.com/ethereum/go-ethereum/metrics"
//	"github.com/ethereum/go-ethereum/p2p/enode"
//	"github.com/ethereum/go-ethereum/portalnetwork/storage"
//	"github.com/holiman/uint256"
//)
//
//const (
//	contentDeletionFraction = 0.05
//	prefixContent           = byte(0x01) // prefixContent + distance + contentId -> content
//	prefixDistanceSize      = byte(0x02) // prefixDistanceSize + distance -> total size
//)
//
//type ContentStorage struct {
//	nodeId                 enode.ID
//	storageCapacityInBytes uint64
//	radius                 atomic.Value
//	db                     *pebble.DB
//	log                    log.Logger
//}
//
//func NewHistoryStorage(config storage.PortalStorageConfig) (storage.ContentStorage, error) {
//	dbPath := path.Join(config.DataDir, config.NetworkName)
//
//	opts := &pebble.Options{
//		MaxOpenFiles: 1000,
//	}
//
//	db, err := pebble.Open(dbPath, opts)
//	if err != nil {
//		return nil, err
//	}
//
//	cs := &ContentStorage{
//		nodeId:                 config.NodeId,
//		db:                     db,
//		storageCapacityInBytes: config.StorageCapacityMB * 1000000,
//		log:                    log.New("storage", config.NetworkName),
//	}
//	cs.radius.Store(storage.MaxDistance)
//	cs.setRadiusToFarthestDistance()
//
//	return cs, nil
//}
//
//func makeKey(prefix byte, distance []byte, contentId []byte) []byte {
//	if contentId == nil {
//		key := make([]byte, 1+len(distance))
//		key[0] = prefix
//		copy(key[1:], distance)
//		return key
//	}
//	key := make([]byte, 1+len(distance)+len(contentId))
//	key[0] = prefix
//	copy(key[1:], distance)
//	copy(key[1+len(distance):], contentId)
//	return key
//}
//
//func (p *ContentStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
//	distance := xor(contentId, p.nodeId[:])
//	key := makeKey(prefixContent, distance, contentId)
//
//	batch := p.db.NewBatch()
//	defer batch.Close()
//
//	// Update content
//	if err := batch.Set(key, content, pebble.Sync); err != nil {
//		return err
//	}
//
//	// Update distance size index
//	sizeKey := makeKey(prefixDistanceSize, distance, nil)
//	var currentSize uint64
//	if value, closer, err := p.db.Get(sizeKey); err == nil {
//		currentSize = binary.BigEndian.Uint64(value)
//		closer.Close()
//	}
//
//	newSize := currentSize + uint64(len(content))
//	sizeBytes := make([]byte, 8)
//	binary.BigEndian.PutUint64(sizeBytes, newSize)
//
//	if err := batch.Set(sizeKey, sizeBytes, pebble.Sync); err != nil {
//		return err
//	}
//
//	if err := batch.Commit(pebble.Sync); err != nil {
//		return err
//	}
//
//	if size, _ := p.UsedSize(); size > p.storageCapacityInBytes {
//		if _, err := p.deleteContentFraction(contentDeletionFraction); err != nil {
//			p.log.Warn("failed to delete oversize content", "err", err)
//		}
//	}
//
//	if metrics.Enabled {
//		portalStorageMetrics.EntriesCount.Inc(1)
//		portalStorageMetrics.ContentStorageUsage.Inc(int64(len(content)))
//	}
//
//	return nil
//}
//
//func (p *ContentStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
//	distance := xor(contentId, p.nodeId[:])
//	key := makeKey(prefixContent, distance, contentId)
//
//	value, closer, err := p.db.Get(key)
//	if err == pebble.ErrNotFound {
//		return nil, storage.ErrContentNotFound
//	}
//	if err != nil {
//		return nil, err
//	}
//	defer closer.Close()
//
//	return value, nil
//}
//
//func (p *ContentStorage) deleteContentFraction(fraction float64) (deleteCount int, err error) {
//	if fraction <= 0 || fraction >= 1 {
//		return 0, errors.New("fraction should be between 0 and 1")
//	}
//
//	totalSize, err := p.ContentSize()
//	if err != nil {
//		return 0, err
//	}
//
//	targetSize := uint64(float64(totalSize) * fraction)
//	deletedSize := uint64(0)
//	count := 0
//
//	iter := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixContent},
//		UpperBound: []byte{prefixContent + 1},
//	})
//	defer iter.Close()
//
//	batch := p.db.NewBatch()
//	defer batch.Close()
//
//	for iter.Last(); iter.Valid() && deletedSize < targetSize; iter.Prev() {
//		key := iter.Key()
//		value := iter.Value()
//		distance := key[1:33]
//
//		// Delete content
//		if err := batch.Delete(key, nil); err != nil {
//			return count, err
//		}
//
//		// Update distance size index
//		sizeKey := makeKey(prefixDistanceSize, distance, nil)
//		var currentSize uint64
//		sizeValue, closer, err := p.db.Get(sizeKey)
//		if err == nil {
//			currentSize = binary.BigEndian.Uint64(sizeValue)
//			closer.Close()
//		}
//
//		newSize := currentSize - uint64(len(value))
//		if newSize == 0 {
//			if err := batch.Delete(sizeKey, nil); err != nil {
//				return count, err
//			}
//		} else {
//			sizeBytes := make([]byte, 8)
//			binary.BigEndian.PutUint64(sizeBytes, newSize)
//			if err := batch.Set(sizeKey, sizeBytes, nil); err != nil {
//				return count, err
//			}
//		}
//
//		deletedSize += uint64(len(value))
//		count++
//
//		if batch.Len() >= 1000 {
//			if err := batch.Commit(pebble.Sync); err != nil {
//				return count, err
//			}
//			batch = p.db.NewBatch()
//		}
//	}
//	if batch.Len() > 0 {
//		if err := batch.Commit(pebble.Sync); err != nil {
//			return count, err
//		}
//	}
//
//	if iter.Valid() {
//		key := iter.Key()
//		distance := key[1:33]
//		dis := uint256.NewInt(0)
//		if err := dis.UnmarshalSSZ(distance); err != nil {
//			return count, err
//		}
//		p.radius.Store(dis)
//	}
//
//	return count, nil
//}
//
//func (p *ContentStorage) UsedSize() (uint64, error) {
//	var totalSize uint64
//	iter := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixDistanceSize},
//		UpperBound: []byte{prefixDistanceSize + 1},
//	})
//	defer iter.Close()
//
//	for iter.First(); iter.Valid(); iter.Next() {
//		size := binary.BigEndian.Uint64(iter.Value())
//		totalSize += size
//	}
//
//	return totalSize, nil
//}
//
//func (p *ContentStorage) ContentSize() (uint64, error) {
//	return p.UsedSize()
//}
//
//func (p *ContentStorage) ContentCount() (uint64, error) {
//	var count uint64
//	iter, _ := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixContent},
//		UpperBound: []byte{prefixContent + 1},
//	})
//	defer iter.Close()
//
//	for iter.First(); iter.Valid(); iter.Next() {
//		count++
//	}
//
//	return count, nil
//}
//
//func (p *ContentStorage) Radius() *uint256.Int {
//	radius := p.radius.Load()
//	val := radius.(*uint256.Int)
//	return val
//}
//func (p *ContentStorage) GetLargestDistance() (*uint256.Int, error) {
//	iter := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixContent},
//		UpperBound: []byte{prefixContent + 1},
//	})
//	defer iter.Close()
//
//	if !iter.Last() {
//		return nil, fmt.Errorf("no content found")
//	}
//
//	key := iter.Key()
//	distance := key[1:33]
//
//	res := uint256.NewInt(0)
//	err := res.UnmarshalSSZ(distance)
//	return res, err
//}
//
//func (p *ContentStorage) EstimateNewRadius(currentRadius *uint256.Int) (*uint256.Int, error) {
//	currrentSize, err := p.UsedSize()
//	if err != nil {
//		return nil, err
//	}
//
//	sizeRatio := currrentSize / p.storageCapacityInBytes
//	if sizeRatio > 0 {
//		newRadius := new(uint256.Int).Div(currentRadius, uint256.NewInt(sizeRatio))
//
//		if metrics.Enabled {
//			ratio := new(uint256.Int).Mul(newRadius, uint256.NewInt(100))
//			ratio.Mod(ratio, storage.MaxDistance)
//			portalStorageMetrics.RadiusRatio.Update(ratio.Float64() / 100)
//		}
//
//		return newRadius, nil
//	}
//	return currentRadius, nil
//}
//
//func (p *ContentStorage) setRadiusToFarthestDistance() {
//	largestDistance, err := p.GetLargestDistance()
//	if err != nil {
//		p.log.Error("failed to get farthest distance", "err", err)
//		return
//	}
//	p.radius.Store(largestDistance)
//}
//func (p *ContentStorage) ForcePrune(radius *uint256.Int) error {
//	batch := p.db.NewBatch()
//	defer batch.Close()
//
//	iter := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixContent},
//		UpperBound: []byte{prefixContent + 1},
//	})
//	defer iter.Close()
//
//	var deletedSize int64
//	deleteCount := 0
//
//	for iter.First(); iter.Valid(); iter.Next() {
//		key := iter.Key()
//		value := iter.Value()
//		distance := key[1:33]
//
//		dis := uint256.NewInt(0)
//		if err := dis.UnmarshalSSZ(distance); err != nil {
//			return err
//		}
//
//		if dis.Cmp(radius) > 0 {
//			// Delete content
//			if err := batch.Delete(key, nil); err != nil {
//				return err
//			}
//
//			// Update distance size index
//			sizeKey := makeKey(prefixDistanceSize, distance, nil)
//			var currentSize uint64
//			if sizeValue, closer, err := p.db.Get(sizeKey); err == nil {
//				currentSize = binary.BigEndian.Uint64(sizeValue)
//				closer.Close()
//			}
//
//			newSize := currentSize - uint64(len(value))
//			if newSize == 0 {
//				if err := batch.Delete(sizeKey, nil); err != nil {
//					return err
//				}
//			} else {
//				sizeBytes := make([]byte, 8)
//				binary.BigEndian.PutUint64(sizeBytes, newSize)
//				if err := batch.Set(sizeKey, sizeBytes, nil); err != nil {
//					return err
//				}
//			}
//
//			deletedSize += int64(len(value))
//			deleteCount++
//		}
//
//		if batch.Len() >= 1000 {
//			if err := batch.Commit(pebble.Sync); err != nil {
//				return err
//			}
//			batch = p.db.NewBatch()
//		}
//	}
//	if batch.Len() > 0 {
//		if err := batch.Commit(pebble.Sync); err != nil {
//			return err
//		}
//	}
//
//	if metrics.Enabled {
//		portalStorageMetrics.EntriesCount.Dec(int64(deleteCount))
//		portalStorageMetrics.ContentStorageUsage.Dec(deletedSize)
//	}
//
//	return nil
//}
//
//func (p *ContentStorage) ReclaimSpace() error {
//	return p.db.Compact([]byte{prefixContent}, []byte{prefixContent + 1}, true)
//}
//
//func (p *ContentStorage) Close() error {
//	return p.db.Close()
//}
//
//func (p *ContentStorage) SizeByKey(contentId []byte) (uint64, error) {
//	distance := xor(contentId, p.nodeId[:])
//	key := makeKey(prefixContent, distance, contentId)
//
//	value, closer, err := p.db.Get(key)
//	if err == pebble.ErrNotFound {
//		return 0, nil
//	}
//	if err != nil {
//		return 0, err
//	}
//	defer closer.Close()
//
//	return uint64(len(value)), nil
//}
//
//func (p *ContentStorage) SizeByKeys(ids [][]byte) (uint64, error) {
//	var totalSize uint64
//
//	for _, id := range ids {
//		size, err := p.SizeByKey(id)
//		if err != nil {
//			return 0, err
//		}
//		totalSize += size
//	}
//
//	return totalSize, nil
//}
//
//func (p *ContentStorage) SizeOutRadius(radius *uint256.Int) (uint64, error) {
//	var totalSize uint64
//
//	iter := p.db.NewIter(&pebble.IterOptions{
//		LowerBound: []byte{prefixDistanceSize},
//		UpperBound: []byte{prefixDistanceSize + 1},
//	})
//	defer iter.Close()
//
//	for iter.First(); iter.Valid(); iter.Next() {
//		key := iter.Key()
//		distance := key[1:33]
//
//		dis := uint256.NewInt(0)
//		if err := dis.UnmarshalSSZ(distance); err != nil {
//			return 0, err
//		}
//
//		if dis.Cmp(radius) > 0 {
//			size := binary.BigEndian.Uint64(iter.Value())
//			totalSize += size
//		}
//	}
//
//	return totalSize, nil
//}
