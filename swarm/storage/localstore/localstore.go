// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package localstore

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

var (
	// ErrInvalidMode is retuned when an unknown Mode
	// is provided to the function.
	ErrInvalidMode = errors.New("invalid mode")
	// ErrDBClosed is returned when database is closed.
	ErrDBClosed = errors.New("db closed")
	// ErrAddressLockTimeout is returned when the same chunk
	// is updated in parallel and one of the updates
	// takes longer then the configured timeout duration.
	ErrAddressLockTimeout = errors.New("address lock timeout")
)

var (
	// Default value for Capacity DB option.
	defaultCapacity int64 = 5000000
	// Limit the number of goroutines created by Getters
	// that call updateGC function. Value 0 sets no limit.
	maxParallelUpdateGC = 1000
)

// DB is the local store implementation and holds
// database related objects.
type DB struct {
	shed *shed.DB

	// schema name of loaded data
	schemaName shed.StringField
	// filed that stores number of intems in gc index
	storedGCSize shed.Uint64Field

	// this flag is for benchmarking two types of retrieval indexes
	// - single retrieval composite index retrievalCompositeIndex
	// - two separated indexes for data and access time
	//   - retrievalDataIndex
	//   - retrievalAccessIndex
	useRetrievalCompositeIndex bool
	// retrieval indexes
	retrievalCompositeIndex shed.Index
	retrievalDataIndex      shed.Index
	retrievalAccessIndex    shed.Index
	// sync indexes
	pushIndex shed.Index
	pullIndex shed.Index
	// garbage collection index
	gcIndex shed.Index
	// index that stores hashes that are not
	// counted in and saved to storedGCSize
	gcUncountedHashesIndex shed.Index

	// number of elements in garbage collection index
	gcSize int64
	// garbage collection is triggered when gcSize exceeds
	// the capacity value
	capacity int64

	// triggers garbage collection event loop
	collectGarbageTrigger chan struct{}
	// triggers write gc size event loop
	writeGCSizeTrigger chan struct{}

	// a buffered channel acting as a semaphore
	// to limit the maximal number of goroutines
	// created by Getters to call updateGC function
	updateGCSem chan struct{}
	// a wait group to ensure all updateGC goroutines
	// are done before closing the database
	updateGCWG sync.WaitGroup

	baseKey []byte

	addressLocks sync.Map

	// this channel is closed when close function is called
	// to terminate other goroutines
	close chan struct{}
}

// Options struct holds optional parameters for configuring DB.
type Options struct {
	// UseRetrievalCompositeIndex option is for benchmarking
	// two types of retrieval indexes:
	// - single retrieval composite index retrievalCompositeIndex
	// - two separated indexes for data and access time
	//   - retrievalDataIndex
	//   - retrievalAccessIndex
	// It should be temporary until the decision is reached
	// about the retrieval index structure.
	UseRetrievalCompositeIndex bool
	// MockStore is a mock node store that is used to store
	// chunk data in a central store. It can be used to reduce
	// total storage space requirements in testing large number
	// of swarm nodes with chunk data deduplication provided by
	// the mock global store.
	MockStore *mock.NodeStore
	// Capacity is a limit that triggers garbage collection when
	// number of items in gcIndex equals or exceeds it.
	Capacity int64
}

// New returns a new DB.  All fields and indexes are initialized
// and possible conflicts with schema from existing database is checked.
// One goroutine for writing batches is created.
func New(path string, baseKey []byte, o *Options) (db *DB, err error) {
	if o == nil {
		o = new(Options)
	}
	db = &DB{
		capacity:                   o.Capacity,
		baseKey:                    baseKey,
		useRetrievalCompositeIndex: o.UseRetrievalCompositeIndex,
		// channels collectGarbageTrigger and writeGCSizeTrigger
		// need to be buffered with the size of 1
		// to signal another event if it
		// is triggered during already running function
		collectGarbageTrigger: make(chan struct{}, 1),
		writeGCSizeTrigger:    make(chan struct{}, 1),
		close:                 make(chan struct{}),
	}
	if db.capacity <= 0 {
		db.capacity = defaultCapacity
	}
	if maxParallelUpdateGC > 0 {
		db.updateGCSem = make(chan struct{}, maxParallelUpdateGC)
	}

	db.shed, err = shed.NewDB(path)
	if err != nil {
		return nil, err
	}
	// Identify current storage schema by arbitrary name.
	db.schemaName, err = db.shed.NewStringField("schema-name")
	if err != nil {
		return nil, err
	}
	// Persist gc size.
	db.storedGCSize, err = db.shed.NewUint64Field("gc-size")
	if err != nil {
		return nil, err
	}
	if db.useRetrievalCompositeIndex {
		var (
			encodeValueFunc func(fields shed.Item) (value []byte, err error)
			decodeValueFunc func(keyItem shed.Item, value []byte) (e shed.Item, err error)
		)
		if o.MockStore != nil {
			encodeValueFunc = func(fields shed.Item) (value []byte, err error) {
				b := make([]byte, 16)
				binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
				binary.BigEndian.PutUint64(b[8:16], uint64(fields.AccessTimestamp))
				err = o.MockStore.Put(fields.Address, fields.Data)
				if err != nil {
					return nil, err
				}
				return b, nil
			}
			decodeValueFunc = func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.AccessTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
				e.Data, err = o.MockStore.Get(keyItem.Address)
				return e, err
			}
		} else {
			encodeValueFunc = func(fields shed.Item) (value []byte, err error) {
				b := make([]byte, 16)
				binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
				binary.BigEndian.PutUint64(b[8:16], uint64(fields.AccessTimestamp))
				value = append(b, fields.Data...)
				return value, nil
			}
			decodeValueFunc = func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.AccessTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
				e.Data = value[16:]
				return e, nil
			}
		}
		// Index storing chunk data with stored and access timestamps.
		db.retrievalCompositeIndex, err = db.shed.NewIndex("Hash->StoredTimestamp|AccessTimestamp|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.Item) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.Item, err error) {
				e.Address = key
				return e, nil
			},
			EncodeValue: encodeValueFunc,
			DecodeValue: decodeValueFunc,
		})
		if err != nil {
			return nil, err
		}
	} else {
		var (
			encodeValueFunc func(fields shed.Item) (value []byte, err error)
			decodeValueFunc func(keyItem shed.Item, value []byte) (e shed.Item, err error)
		)
		if o.MockStore != nil {
			encodeValueFunc = func(fields shed.Item) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
				err = o.MockStore.Put(fields.Address, fields.Data)
				if err != nil {
					return nil, err
				}
				return b, nil
			}
			decodeValueFunc = func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.Data, err = o.MockStore.Get(keyItem.Address)
				return e, err
			}
		} else {
			encodeValueFunc = func(fields shed.Item) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
				value = append(b, fields.Data...)
				return value, nil
			}
			decodeValueFunc = func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.Data = value[8:]
				return e, nil
			}
		}
		// Index storing actual chunk address, data and store timestamp.
		db.retrievalDataIndex, err = db.shed.NewIndex("Address->StoreTimestamp|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.Item) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.Item, err error) {
				e.Address = key
				return e, nil
			},
			EncodeValue: encodeValueFunc,
			DecodeValue: decodeValueFunc,
		})
		if err != nil {
			return nil, err
		}
		// Index storing access timestamp for a particular address.
		// It is needed in order to update gc index keys for iteration order.
		db.retrievalAccessIndex, err = db.shed.NewIndex("Address->AccessTimestamp", shed.IndexFuncs{
			EncodeKey: func(fields shed.Item) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.Item, err error) {
				e.Address = key
				return e, nil
			},
			EncodeValue: func(fields shed.Item) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.AccessTimestamp))
				return b, nil
			},
			DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
				e.AccessTimestamp = int64(binary.BigEndian.Uint64(value))
				return e, nil
			},
		})
		if err != nil {
			return nil, err
		}
	}
	// pull index allows history and live syncing per po bin
	db.pullIndex, err = db.shed.NewIndex("PO|StoredTimestamp|Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			key = make([]byte, 41)
			key[0] = db.po(fields.Address)
			binary.BigEndian.PutUint64(key[1:9], uint64(fields.StoreTimestamp))
			copy(key[9:], fields.Address[:])
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key[9:]
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[1:9]))
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// push index contains as yet unsynced chunks
	db.pushIndex, err = db.shed.NewIndex("StoredTimestamp|Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			key = make([]byte, 40)
			binary.BigEndian.PutUint64(key[:8], uint64(fields.StoreTimestamp))
			copy(key[8:], fields.Address[:])
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key[8:]
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// gc index for removable chunk ordered by ascending last access time
	db.gcIndex, err = db.shed.NewIndex("AccessTimestamp|StoredTimestamp|Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			b := make([]byte, 16, 16+len(fields.Address))
			binary.BigEndian.PutUint64(b[:8], uint64(fields.AccessTimestamp))
			binary.BigEndian.PutUint64(b[8:16], uint64(fields.StoreTimestamp))
			key = append(b, fields.Address...)
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[8:16]))
			e.Address = key[16:]
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// gc uncounted hashes index keeps hashes that are in gc index
	// but not counted in and saved to storedGCSize
	db.gcUncountedHashesIndex, err = db.shed.NewIndex("Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}

	// count number of elements in garbage collection index
	gcSize, err := db.storedGCSize.Get()
	if err != nil {
		return nil, err
	}
	// get number of uncounted hashes
	gcUncountedSize, err := db.gcUncountedHashesIndex.Count()
	if err != nil {
		return nil, err
	}
	gcSize += uint64(gcUncountedSize)
	// remove uncounted hashes from the index
	err = db.gcUncountedHashesIndex.IterateAll(func(item shed.Item) (stop bool, err error) {
		return false, db.gcUncountedHashesIndex.Delete(item)
	})
	if err != nil {
		return nil, err
	}
	// save the total gcSize after uncounted hashes are removed
	err = db.storedGCSize.Put(gcSize)
	if err != nil {
		return nil, err
	}
	db.incGCSize(int64(gcSize))

	// start worker to write gc size
	go db.writeGCSizeWorker()
	// start garbage collection worker
	go db.collectGarbageWorker()
	return db, nil
}

// Close closes the underlying database.
func (db *DB) Close() (err error) {
	close(db.close)
	db.updateGCWG.Wait()
	return db.shed.Close()
}

// po computes the proximity order between the address
// and database base key.
func (db *DB) po(addr storage.Address) (bin uint8) {
	return uint8(storage.Proximity(db.baseKey, addr))
}

var (
	// Maximal time for lockAddr to wait until it
	// returns error.
	addressLockTimeout = 3 * time.Second
	// duration between two lock checks in lockAddr.
	addressLockCheckDelay = 30 * time.Microsecond
)

// lockAddr sets the lock on a particular address
// using addressLocks sync.Map and returns unlock function.
// If the address is locked this function will check it
// in a for loop for addressLockTimeout time, after which
// it will return ErrAddressLockTimeout error.
func (db *DB) lockAddr(addr storage.Address) (unlock func(), err error) {
	start := time.Now()
	lockKey := hex.EncodeToString(addr)
	for {
		_, loaded := db.addressLocks.LoadOrStore(lockKey, struct{}{})
		if !loaded {
			break
		}
		time.Sleep(addressLockCheckDelay)
		if time.Since(start) > addressLockTimeout {
			return nil, ErrAddressLockTimeout
		}
	}
	return func() { db.addressLocks.Delete(lockKey) }, nil
}

// chunkToItem creates new Item with data provided by the Chunk.
func chunkToItem(ch storage.Chunk) shed.Item {
	return shed.Item{
		Address: ch.Address(),
		Data:    ch.Data(),
	}
}

// addressToItem creates new Item with a provided address.
func addressToItem(addr storage.Address) shed.Item {
	return shed.Item{
		Address: addr,
	}
}

// now is a helper function that returns a current unix timestamp
// in UTC timezone.
// It is set in the init function for usage in production, and
// optionally overridden in tests for data validation.
var now func() int64

func init() {
	// set the now function
	now = func() (t int64) {
		return time.Now().UTC().UnixNano()
	}
}
