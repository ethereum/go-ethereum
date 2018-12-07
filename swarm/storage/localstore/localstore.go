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
	"sync/atomic"
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
	// ErrUpdateLockTimeout is returned when the same chunk
	// is updated in parallel and one of the updates
	// takes longer then the configured timeout duration.
	ErrUpdateLockTimeout = errors.New("update lock timeout")
)

// DB is the local store implementation and holds
// database related objects.
type DB struct {
	shed *shed.DB

	// fields
	schemaName shed.StringField

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

	// number of elements in garbage collection index
	gcSize int64

	baseKey []byte

	updateLocks sync.Map
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
}

// New returns a new DB.  All fields and indexes are initialized
// and possible conflicts with schema from existing database is checked.
// One goroutine for writing batches is created.
func New(path string, baseKey []byte, o *Options) (db *DB, err error) {
	if o == nil {
		o = new(Options)
	}
	db = &DB{
		baseKey:                    baseKey,
		useRetrievalCompositeIndex: o.UseRetrievalCompositeIndex,
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
	if db.useRetrievalCompositeIndex {
		var (
			encodeValueFunc func(fields shed.IndexItem) (value []byte, err error)
			decodeValueFunc func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error)
		)
		if o.MockStore != nil {
			encodeValueFunc = func(fields shed.IndexItem) (value []byte, err error) {
				b := make([]byte, 16)
				binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
				binary.BigEndian.PutUint64(b[8:16], uint64(fields.AccessTimestamp))
				err = o.MockStore.Put(fields.Address, fields.Data)
				if err != nil {
					return nil, err
				}
				return b, nil
			}
			decodeValueFunc = func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.AccessTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
				e.Data, err = o.MockStore.Get(keyIndexItem.Address)
				return e, err
			}
		} else {
			encodeValueFunc = func(fields shed.IndexItem) (value []byte, err error) {
				b := make([]byte, 16)
				binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
				binary.BigEndian.PutUint64(b[8:16], uint64(fields.AccessTimestamp))
				value = append(b, fields.Data...)
				return value, nil
			}
			decodeValueFunc = func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.AccessTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
				e.Data = value[16:]
				return e, nil
			}
		}
		// Index storing chunk data with stored and access timestamps.
		db.retrievalCompositeIndex, err = db.shed.NewIndex("Hash->StoredTimestamp|AccessTimestamp|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
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
			encodeValueFunc func(fields shed.IndexItem) (value []byte, err error)
			decodeValueFunc func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error)
		)
		if o.MockStore != nil {
			encodeValueFunc = func(fields shed.IndexItem) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
				err = o.MockStore.Put(fields.Address, fields.Data)
				if err != nil {
					return nil, err
				}
				return b, nil
			}
			decodeValueFunc = func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.Data, err = o.MockStore.Get(keyIndexItem.Address)
				return e, err
			}
		} else {
			encodeValueFunc = func(fields shed.IndexItem) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
				value = append(b, fields.Data...)
				return value, nil
			}
			decodeValueFunc = func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
				e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
				e.Data = value[8:]
				return e, nil
			}
		}
		// Index storing actual chunk address, data and store timestamp.
		db.retrievalDataIndex, err = db.shed.NewIndex("Address->StoreTimestamp|Data", shed.IndexFuncs{
			EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
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
			EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
				return fields.Address, nil
			},
			DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
				e.Address = key
				return e, nil
			},
			EncodeValue: func(fields shed.IndexItem) (value []byte, err error) {
				b := make([]byte, 8)
				binary.BigEndian.PutUint64(b, uint64(fields.AccessTimestamp))
				return b, nil
			},
			DecodeValue: func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
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
		EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
			key = make([]byte, 41)
			key[0] = db.po(fields.Address)
			binary.BigEndian.PutUint64(key[1:9], uint64(fields.StoreTimestamp))
			copy(key[9:], fields.Address[:])
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
			e.Address = key[9:]
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[1:9]))
			return e, nil
		},
		EncodeValue: func(fields shed.IndexItem) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// push index contains as yet unsynced chunks
	db.pushIndex, err = db.shed.NewIndex("StoredTimestamp|Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
			key = make([]byte, 40)
			binary.BigEndian.PutUint64(key[:8], uint64(fields.StoreTimestamp))
			copy(key[8:], fields.Address[:])
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
			e.Address = key[8:]
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			return e, nil
		},
		EncodeValue: func(fields shed.IndexItem) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// gc index for removable chunk ordered by ascending last access time
	db.gcIndex, err = db.shed.NewIndex("AccessTimestamp|StoredTimestamp|Hash->nil", shed.IndexFuncs{
		EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
			b := make([]byte, 16, 16+len(fields.Address))
			binary.BigEndian.PutUint64(b[:8], uint64(fields.AccessTimestamp))
			binary.BigEndian.PutUint64(b[8:16], uint64(fields.StoreTimestamp))
			key = append(b, fields.Address...)
			return key, nil
		},
		DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[8:16]))
			e.Address = key[16:]
			return e, nil
		},
		EncodeValue: func(fields shed.IndexItem) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(keyIndexItem shed.IndexItem, value []byte) (e shed.IndexItem, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// count number of elements in garbage collection index
	var gcSize int64
	db.gcIndex.IterateAll(func(_ shed.IndexItem) (stop bool, err error) {
		gcSize++
		return false, nil
	})
	atomic.AddInt64(&db.gcSize, gcSize)
	return db, nil
}

// Close closes the underlying database.
func (db *DB) Close() (err error) {
	return db.shed.Close()
}

// po computes the proximity order between the address
// and database base key.
func (db *DB) po(addr storage.Address) (bin uint8) {
	return uint8(storage.Proximity(db.baseKey, addr))
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
