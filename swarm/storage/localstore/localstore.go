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
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

var (
	// ErrInvalidMode is retuned when an unknown Mode
	// is provided to the function.
	ErrInvalidMode = errors.New("invalid mode")
	// ErrAddressLockTimeout is returned when the same chunk
	// is updated in parallel and one of the updates
	// takes longer then the configured timeout duration.
	ErrAddressLockTimeout = errors.New("address lock timeout")
)

var (
	// Default value for Capacity DB option.
	defaultCapacity uint64 = 5000000
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

	// retrieval indexes
	retrievalDataIndex   shed.Index
	retrievalAccessIndex shed.Index
	// push syncing index
	pushIndex shed.Index
	// push syncing subscriptions triggers
	pushTriggers   []chan struct{}
	pushTriggersMu sync.RWMutex

	// pull syncing index
	pullIndex shed.Index
	// pull syncing subscriptions triggers per bin
	pullTriggers   map[uint8][]chan struct{}
	pullTriggersMu sync.RWMutex

	// garbage collection index
	gcIndex shed.Index

	// field that stores number of intems in gc index
	gcSize shed.Uint64Field

	// garbage collection is triggered when gcSize exceeds
	// the capacity value
	capacity uint64

	// triggers garbage collection event loop
	collectGarbageTrigger chan struct{}

	// a buffered channel acting as a semaphore
	// to limit the maximal number of goroutines
	// created by Getters to call updateGC function
	updateGCSem chan struct{}
	// a wait group to ensure all updateGC goroutines
	// are done before closing the database
	updateGCWG sync.WaitGroup

	baseKey []byte

	batchMu sync.Mutex

	// this channel is closed when close function is called
	// to terminate other goroutines
	close chan struct{}

	// protect Close method from exiting before
	// garbage collection and gc size write workers
	// are done
	collectGarbageWorkerDone chan struct{}
}

// Options struct holds optional parameters for configuring DB.
type Options struct {
	// MockStore is a mock node store that is used to store
	// chunk data in a central store. It can be used to reduce
	// total storage space requirements in testing large number
	// of swarm nodes with chunk data deduplication provided by
	// the mock global store.
	MockStore *mock.NodeStore
	// Capacity is a limit that triggers garbage collection when
	// number of items in gcIndex equals or exceeds it.
	Capacity uint64
	// MetricsPrefix defines a prefix for metrics names.
	MetricsPrefix string
}

// New returns a new DB.  All fields and indexes are initialized
// and possible conflicts with schema from existing database is checked.
// One goroutine for writing batches is created.
func New(path string, baseKey []byte, o *Options) (db *DB, err error) {
	if o == nil {
		o = new(Options)
	}
	db = &DB{
		capacity: o.Capacity,
		baseKey:  baseKey,
		// channel collectGarbageTrigger
		// needs to be buffered with the size of 1
		// to signal another event if it
		// is triggered during already running function
		collectGarbageTrigger:    make(chan struct{}, 1),
		close:                    make(chan struct{}),
		collectGarbageWorkerDone: make(chan struct{}),
	}
	if db.capacity <= 0 {
		db.capacity = defaultCapacity
	}
	if maxParallelUpdateGC > 0 {
		db.updateGCSem = make(chan struct{}, maxParallelUpdateGC)
	}

	db.shed, err = shed.NewDB(path, o.MetricsPrefix)
	if err != nil {
		return nil, err
	}
	// Identify current storage schema by arbitrary name.
	db.schemaName, err = db.shed.NewStringField("schema-name")
	if err != nil {
		return nil, err
	}
	// Persist gc size.
	db.gcSize, err = db.shed.NewUint64Field("gc-size")
	if err != nil {
		return nil, err
	}
	// Functions for retrieval data index.
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
	// create a pull syncing triggers used by SubscribePull function
	db.pullTriggers = make(map[uint8][]chan struct{})
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
	// create a push syncing triggers used by SubscribePush function
	db.pushTriggers = make([]chan struct{}, 0)
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
	// start garbage collection worker
	go db.collectGarbageWorker()
	return db, nil
}

// Close closes the underlying database.
func (db *DB) Close() (err error) {
	close(db.close)
	db.updateGCWG.Wait()

	// wait for gc worker to
	// return before closing the shed
	select {
	case <-db.collectGarbageWorkerDone:
	case <-time.After(5 * time.Second):
		log.Error("localstore: collect garbage worker did not return after db close")
	}
	return db.shed.Close()
}

// po computes the proximity order between the address
// and database base key.
func (db *DB) po(addr chunk.Address) (bin uint8) {
	return uint8(chunk.Proximity(db.baseKey, addr))
}

// chunkToItem creates new Item with data provided by the Chunk.
func chunkToItem(ch chunk.Chunk) shed.Item {
	return shed.Item{
		Address: ch.Address(),
		Data:    ch.Data(),
	}
}

// addressToItem creates new Item with a provided address.
func addressToItem(addr chunk.Address) shed.Item {
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
