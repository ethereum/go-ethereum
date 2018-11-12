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

package shed

import (
	"context"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/swarm/shed/internal"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// DB is just an example for composing indexes.
type DB struct {
	db *internal.DB

	// fields and indexes
	schemaName     internal.StringField
	sizeCounter    internal.Uint64Field
	accessCounter  internal.Uint64Field
	retrievalIndex internal.Index
	accessIndex    internal.Index
	gcIndex        internal.Index
}

func New(path string) (db *DB, err error) {
	idb, err := internal.NewDB(path)
	if err != nil {
		return nil, err
	}
	db = &DB{
		db: idb,
	}
	db.schemaName, err = idb.NewStringField("schema-name")
	if err != nil {
		return nil, err
	}
	db.sizeCounter, err = idb.NewUint64Field("size-counter")
	if err != nil {
		return nil, err
	}
	db.accessCounter, err = idb.NewUint64Field("access-counter")
	if err != nil {
		return nil, err
	}
	db.retrievalIndex, err = idb.NewIndex("Hash->StoreTimestamp|Data", internal.IndexFuncs{
		EncodeKey: func(fields internal.IndexItem) (key []byte, err error) {
			return fields.Hash, nil
		},
		DecodeKey: func(key []byte) (e internal.IndexItem, err error) {
			e.Hash = key
			return e, nil
		},
		EncodeValue: func(fields internal.IndexItem) (value []byte, err error) {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
			value = append(b, fields.Data...)
			return value, nil
		},
		DecodeValue: func(value []byte) (e internal.IndexItem, err error) {
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
			e.Data = value[8:]
			return e, nil
		},
	})
	db.accessIndex, err = idb.NewIndex("Hash->AccessTimestamp", internal.IndexFuncs{
		EncodeKey: func(fields internal.IndexItem) (key []byte, err error) {
			return fields.Hash, nil
		},
		DecodeKey: func(key []byte) (e internal.IndexItem, err error) {
			e.Hash = key
			return e, nil
		},
		EncodeValue: func(fields internal.IndexItem) (value []byte, err error) {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(fields.AccessTimestamp))
			return b, nil
		},
		DecodeValue: func(value []byte) (e internal.IndexItem, err error) {
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(value))
			return e, nil
		},
	})
	db.gcIndex, err = idb.NewIndex("AccessTimestamp|StoredTimestamp|Hash->nil", internal.IndexFuncs{
		EncodeKey: func(fields internal.IndexItem) (key []byte, err error) {
			b := make([]byte, 16, 16+len(fields.Hash))
			binary.BigEndian.PutUint64(b[:8], uint64(fields.AccessTimestamp))
			binary.BigEndian.PutUint64(b[8:16], uint64(fields.StoreTimestamp))
			key = append(b, fields.Hash...)
			return key, nil
		},
		DecodeKey: func(key []byte) (e internal.IndexItem, err error) {
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(key[:8]))
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(key[8:16]))
			e.Hash = key[16:]
			return e, nil
		},
		EncodeValue: func(fields internal.IndexItem) (value []byte, err error) {
			return nil, nil
		},
		DecodeValue: func(value []byte) (e internal.IndexItem, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Put(_ context.Context, ch storage.Chunk) (err error) {
	return db.retrievalIndex.Put(internal.IndexItem{
		Hash:           ch.Address(),
		Data:           ch.Data(),
		StoreTimestamp: time.Now().UTC().UnixNano(),
	})
}

func (db *DB) Get(_ context.Context, ref storage.Address) (c storage.Chunk, err error) {
	batch := new(leveldb.Batch)

	item, err := db.retrievalIndex.Get(internal.IndexItem{
		Hash: ref,
	})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, storage.ErrChunkNotFound
		}
		return nil, err
	}

	accessItem, err := db.accessIndex.Get(internal.IndexItem{
		Hash: ref,
	})
	switch err {
	case nil:
		err = db.gcIndex.DeleteInBatch(batch, internal.IndexItem{
			Hash:            item.Hash,
			StoreTimestamp:  accessItem.AccessTimestamp,
			AccessTimestamp: item.StoreTimestamp,
		})
		if err != nil {
			return nil, err
		}
	case leveldb.ErrNotFound:
	default:
		return nil, err
	}

	accessTimestamp := time.Now().UTC().UnixNano()

	err = db.accessIndex.PutInBatch(batch, internal.IndexItem{
		Hash:            ref,
		AccessTimestamp: accessTimestamp,
	})
	if err != nil {
		return nil, err
	}

	err = db.gcIndex.PutInBatch(batch, internal.IndexItem{
		Hash:            item.Hash,
		AccessTimestamp: accessTimestamp,
		StoreTimestamp:  item.StoreTimestamp,
	})
	if err != nil {
		return nil, err
	}

	err = db.db.WriteBatch(batch)
	if err != nil {
		return nil, err
	}

	return storage.NewChunk(item.Hash, item.Data), nil
}

func (db *DB) CollectGarbage() (err error) {
	const maxTrashSize = 100
	maxRounds := 10 // adbitrary number, needs to be calculated

	for roundCount := 0; roundCount < maxRounds; roundCount++ {
		var garbageCount int
		trash := new(leveldb.Batch)
		err = db.gcIndex.IterateAll(func(item internal.IndexItem) (stop bool, err error) {
			err = db.retrievalIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			err = db.accessIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			err = db.gcIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			garbageCount++
			if garbageCount >= maxTrashSize {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return err
		}
		if garbageCount == 0 {
			return nil
		}
		err = db.db.WriteBatch(trash)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetSchema() (name string, err error) {
	name, err = db.schemaName.Get()
	if err == leveldb.ErrNotFound {
		return "", nil
	}
	return name, err
}

func (db *DB) PutSchema(name string) (err error) {
	return db.schemaName.Put(name)
}

func (db *DB) Close() {
	db.db.Close()
}
