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

package shed_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/syndtr/goleveldb/leveldb"
)

// Store holds fields and indexes (including their encoding functions)
// and defines operations on them by composing data from them.
// It implements storage.ChunkStore interface.
// It is just an example without any support for parallel operations
// or real world implementation.
type Store struct {
	db *shed.DB

	// fields and indexes
	schemaName     shed.StringField
	sizeCounter    shed.Uint64Field
	accessCounter  shed.Uint64Field
	retrievalIndex shed.Index
	accessIndex    shed.Index
	gcIndex        shed.Index
}

// New returns new Store. All fields and indexes are initialized
// and possible conflicts with schema from existing database is checked
// automatically.
func New(path string) (s *Store, err error) {
	db, err := shed.NewDB(path, "")
	if err != nil {
		return nil, err
	}
	s = &Store{
		db: db,
	}
	// Identify current storage schema by arbitrary name.
	s.schemaName, err = db.NewStringField("schema-name")
	if err != nil {
		return nil, err
	}
	// Global ever incrementing index of chunk accesses.
	s.accessCounter, err = db.NewUint64Field("access-counter")
	if err != nil {
		return nil, err
	}
	// Index storing actual chunk address, data and store timestamp.
	s.retrievalIndex, err = db.NewIndex("Address->StoreTimestamp|Data", shed.IndexFuncs{
		EncodeKey: func(fields shed.Item) (key []byte, err error) {
			return fields.Address, nil
		},
		DecodeKey: func(key []byte) (e shed.Item, err error) {
			e.Address = key
			return e, nil
		},
		EncodeValue: func(fields shed.Item) (value []byte, err error) {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(fields.StoreTimestamp))
			value = append(b, fields.Data...)
			return value, nil
		},
		DecodeValue: func(keyItem shed.Item, value []byte) (e shed.Item, err error) {
			e.StoreTimestamp = int64(binary.BigEndian.Uint64(value[:8]))
			e.Data = value[8:]
			return e, nil
		},
	})
	if err != nil {
		return nil, err
	}
	// Index storing access timestamp for a particular address.
	// It is needed in order to update gc index keys for iteration order.
	s.accessIndex, err = db.NewIndex("Address->AccessTimestamp", shed.IndexFuncs{
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
	// Index with keys ordered by access timestamp for garbage collection prioritization.
	s.gcIndex, err = db.NewIndex("AccessTimestamp|StoredTimestamp|Address->nil", shed.IndexFuncs{
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
	return s, nil
}

// Put stores the chunk and sets it store timestamp.
func (s *Store) Put(_ context.Context, ch storage.Chunk) (err error) {
	return s.retrievalIndex.Put(shed.Item{
		Address:        ch.Address(),
		Data:           ch.Data(),
		StoreTimestamp: time.Now().UTC().UnixNano(),
	})
}

// Get retrieves a chunk with the provided address.
// It updates access and gc indexes by removing the previous
// items from them and adding new items as keys of index entries
// are changed.
func (s *Store) Get(_ context.Context, addr storage.Address) (c storage.Chunk, err error) {
	batch := new(leveldb.Batch)

	// Get the chunk data and storage timestamp.
	item, err := s.retrievalIndex.Get(shed.Item{
		Address: addr,
	})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, storage.ErrChunkNotFound
		}
		return nil, err
	}

	// Get the chunk access timestamp.
	accessItem, err := s.accessIndex.Get(shed.Item{
		Address: addr,
	})
	switch err {
	case nil:
		// Remove gc index entry if access timestamp is found.
		err = s.gcIndex.DeleteInBatch(batch, shed.Item{
			Address:         item.Address,
			StoreTimestamp:  accessItem.AccessTimestamp,
			AccessTimestamp: item.StoreTimestamp,
		})
		if err != nil {
			return nil, err
		}
	case leveldb.ErrNotFound:
	// Access timestamp is not found. Do not do anything.
	// This is the firs get request.
	default:
		return nil, err
	}

	// Specify new access timestamp
	accessTimestamp := time.Now().UTC().UnixNano()

	// Put new access timestamp in access index.
	err = s.accessIndex.PutInBatch(batch, shed.Item{
		Address:         addr,
		AccessTimestamp: accessTimestamp,
	})
	if err != nil {
		return nil, err
	}

	// Put new access timestamp in gc index.
	err = s.gcIndex.PutInBatch(batch, shed.Item{
		Address:         item.Address,
		AccessTimestamp: accessTimestamp,
		StoreTimestamp:  item.StoreTimestamp,
	})
	if err != nil {
		return nil, err
	}

	// Increment access counter.
	// Currently this information is not used anywhere.
	_, err = s.accessCounter.IncInBatch(batch)
	if err != nil {
		return nil, err
	}

	// Write the batch.
	err = s.db.WriteBatch(batch)
	if err != nil {
		return nil, err
	}

	// Return the chunk.
	return storage.NewChunk(item.Address, item.Data), nil
}

// CollectGarbage is an example of index iteration.
// It provides no reliable garbage collection functionality.
func (s *Store) CollectGarbage() (err error) {
	const maxTrashSize = 100
	maxRounds := 10 // arbitrary number, needs to be calculated

	// Run a few gc rounds.
	for roundCount := 0; roundCount < maxRounds; roundCount++ {
		var garbageCount int
		// New batch for a new cg round.
		trash := new(leveldb.Batch)
		// Iterate through all index items and break when needed.
		err = s.gcIndex.Iterate(func(item shed.Item) (stop bool, err error) {
			// Remove the chunk.
			err = s.retrievalIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			// Remove the element in gc index.
			err = s.gcIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			// Remove the relation in access index.
			err = s.accessIndex.DeleteInBatch(trash, item)
			if err != nil {
				return false, err
			}
			garbageCount++
			if garbageCount >= maxTrashSize {
				return true, nil
			}
			return false, nil
		}, nil)
		if err != nil {
			return err
		}
		if garbageCount == 0 {
			return nil
		}
		err = s.db.WriteBatch(trash)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetSchema is an example of retrieveing the most simple
// string from a database field.
func (s *Store) GetSchema() (name string, err error) {
	name, err = s.schemaName.Get()
	if err == leveldb.ErrNotFound {
		return "", nil
	}
	return name, err
}

// GetSchema is an example of storing the most simple
// string in a database field.
func (s *Store) PutSchema(name string) (err error) {
	return s.schemaName.Put(name)
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Example_store constructs a simple storage implementation using shed package.
func Example_store() {
	dir, err := ioutil.TempDir("", "ephemeral")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	s, err := New(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	ch := storage.GenerateRandomChunk(1024)
	err = s.Put(context.Background(), ch)
	if err != nil {
		log.Fatal(err)
	}

	got, err := s.Get(context.Background(), ch.Address())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(bytes.Equal(got.Data(), ch.Data()))

	//Output: true
}
