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

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	// maximal time for DB.Close must return
	closeTimeout = 10 * time.Second
)

var (
	// ErrInvalidMode is retuned when an unknown Mode
	// is provided to the function.
	ErrInvalidMode = errors.New("invalid mode")
	// ErrDBClosed is returned when database is closed.
	ErrDBClosed = errors.New("db closed")
)

// DB is the local store implementation and holds
// database related objects.
type DB struct {
	shed *shed.DB

	// fields and indexes
	schemaName     shed.StringField
	sizeCounter    shed.Uint64Field
	retrievalIndex shed.Index
	pushIndex      shed.Index
	pullIndex      shed.Index
	gcIndex        shed.Index

	baseKey []byte

	batch        *batch        // current batch
	mu           sync.RWMutex  // mutex for accessing current batch
	writeTrigger chan struct{} // channel to signal current write batch
	writeDone    chan struct{} // closed when writeBatches function returns
	close        chan struct{} // closed on Close, signals other goroutines to terminate
}

// New returns a new DB.  All fields and indexes are initialized
// and possible conflicts with schema from existing database is checked.
// One goroutine for writing batches is created.
func New(path string, baseKey []byte) (db *DB, err error) {
	db = &DB{
		baseKey:      baseKey,
		batch:        newBatch(),
		writeTrigger: make(chan struct{}, 1),
		close:        make(chan struct{}),
		writeDone:    make(chan struct{}),
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
	db.sizeCounter, err = db.shed.NewUint64Field("size")
	if err != nil {
		return nil, err
	}
	db.retrievalIndex, err = db.shed.NewIndex("Hash->StoredTimestamp|AccessTimestamp|Data", shed.IndexFuncs{
		EncodeKey: func(fields shed.IndexItem) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e shed.IndexItem, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: func(fields shed.IndexItem) (value []byte, err error) {
			b := make([]byte, 16)
			binary.BigEndian.PutUint64(b[:8], uint64(fields.StoreTimestamp))
			binary.BigEndian.PutUint64(b[8:16], uint64(fields.AccessTimestamp))
			value = append(b, fields.Data...)
			return value, nil
		},
		DecodeValue: func(value []byte) (e shed.IndexItem, err error) {
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(value[8:16]))
			e.Data = value[16:]
			return e, nil
		},
	})
	if err != nil {
		return nil, err
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
		DecodeValue: func(value []byte) (e shed.IndexItem, err error) {
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
		DecodeValue: func(value []byte) (e shed.IndexItem, err error) {
			e.AccessTimestamp = int64(binary.BigEndian.Uint64(value))
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
		DecodeValue: func(value []byte) (e shed.IndexItem, err error) {
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// start goroutine that writes batches
	go db.writeBatches()
	return db, nil
}

// Close closes the underlying database.
func (db *DB) Close() (err error) {
	// signal other goroutines that
	// the database is closing
	close(db.close)
	select {
	// wait for writeBatches to write
	// the last batch
	case <-db.writeDone:
	// closing timeout
	case <-time.After(closeTimeout):
	}
	return db.shed.Close()
}

// writeBatches is a forever loop handing out the current batch apply
// the batch when the db is free.
func (db *DB) writeBatches() {
	// close the writeDone channel
	// so the DB.Close can return
	defer close(db.writeDone)

	write := func() {
		db.mu.Lock()
		b := db.batch
		db.batch = newBatch()
		db.mu.Unlock()
		b.Err = db.shed.WriteBatch(b.Batch)
		close(b.Done)
	}
	for {
		select {
		case <-db.writeTrigger:
			write()
		case <-db.close:
			// check it there is a batch
			// left to be written
			write()
			return
		}
	}
}

// po computes the proximity order between the address
// and database base key.
func (db *DB) po(addr storage.Address) (bin uint8) {
	return uint8(storage.Proximity(db.baseKey, addr))
}

// now is a helper function that returns a current unix timestamp
// in UTC timezone.
func now() (t int64) {
	return time.Now().UTC().UnixNano()
}
