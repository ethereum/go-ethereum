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

// Package db implements a mock store that keeps all chunk data in LevelDB database.
package db

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

// GlobalStore contains the LevelDB database that is storing
// chunk data for all swarm nodes.
// Closing the GlobalStore with Close method is required to
// release resources used by the database.
type GlobalStore struct {
	db *leveldb.DB
}

// NewGlobalStore creates a new instance of GlobalStore.
func NewGlobalStore(path string) (s *GlobalStore, err error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &GlobalStore{
		db: db,
	}, nil
}

// Close releases the resources used by the underlying LevelDB.
func (s *GlobalStore) Close() error {
	return s.db.Close()
}

// NewNodeStore returns a new instance of NodeStore that retrieves and stores
// chunk data only for a node with address addr.
func (s *GlobalStore) NewNodeStore(addr common.Address) *mock.NodeStore {
	return mock.NewNodeStore(addr, s)
}

// Get returns chunk data if the chunk with key exists for node
// on address addr.
func (s *GlobalStore) Get(addr common.Address, key []byte) (data []byte, err error) {
	has, err := s.db.Has(nodeDBKey(addr, key), nil)
	if err != nil {
		return nil, mock.ErrNotFound
	}
	if !has {
		return nil, mock.ErrNotFound
	}
	data, err = s.db.Get(dataDBKey(key), nil)
	if err == leveldb.ErrNotFound {
		err = mock.ErrNotFound
	}
	return
}

// Put saves the chunk data for node with address addr.
func (s *GlobalStore) Put(addr common.Address, key []byte, data []byte) error {
	batch := new(leveldb.Batch)
	batch.Put(nodeDBKey(addr, key), nil)
	batch.Put(dataDBKey(key), data)
	return s.db.Write(batch, nil)
}

// HasKey returns whether a node with addr contains the key.
func (s *GlobalStore) HasKey(addr common.Address, key []byte) bool {
	has, err := s.db.Has(nodeDBKey(addr, key), nil)
	if err != nil {
		has = false
	}
	return has
}

// Import reads tar archive from a reader that contains exported chunk data.
// It returns the number of chunks imported and an error.
func (s *GlobalStore) Import(r io.Reader) (n int, err error) {
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return n, err
		}

		data, err := ioutil.ReadAll(tr)
		if err != nil {
			return n, err
		}

		var c mock.ExportedChunk
		if err = json.Unmarshal(data, &c); err != nil {
			return n, err
		}

		batch := new(leveldb.Batch)
		for _, addr := range c.Addrs {
			batch.Put(nodeDBKeyHex(addr, hdr.Name), nil)
		}

		batch.Put(dataDBKey(common.Hex2Bytes(hdr.Name)), c.Data)
		if err = s.db.Write(batch, nil); err != nil {
			return n, err
		}

		n++
	}
	return n, err
}

// Export writes to a writer a tar archive with all chunk data from
// the store. It returns the number fo chunks exported and an error.
func (s *GlobalStore) Export(w io.Writer) (n int, err error) {
	tw := tar.NewWriter(w)
	defer tw.Close()

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	encoder := json.NewEncoder(buf)

	iter := s.db.NewIterator(util.BytesPrefix(nodeKeyPrefix), nil)
	defer iter.Release()

	var currentKey string
	var addrs []common.Address

	saveChunk := func(hexKey string) error {
		key := common.Hex2Bytes(hexKey)

		data, err := s.db.Get(dataDBKey(key), nil)
		if err != nil {
			return err
		}

		buf.Reset()
		if err = encoder.Encode(mock.ExportedChunk{
			Addrs: addrs,
			Data:  data,
		}); err != nil {
			return err
		}

		d := buf.Bytes()
		hdr := &tar.Header{
			Name: hexKey,
			Mode: 0644,
			Size: int64(len(d)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(d); err != nil {
			return err
		}
		n++
		return nil
	}

	for iter.Next() {
		k := bytes.TrimPrefix(iter.Key(), nodeKeyPrefix)
		i := bytes.Index(k, []byte("-"))
		if i < 0 {
			continue
		}
		hexKey := string(k[:i])

		if currentKey == "" {
			currentKey = hexKey
		}

		if hexKey != currentKey {
			if err = saveChunk(currentKey); err != nil {
				return n, err
			}

			addrs = addrs[:0]
		}

		currentKey = hexKey
		addrs = append(addrs, common.BytesToAddress(k[i:]))
	}

	if len(addrs) > 0 {
		if err = saveChunk(currentKey); err != nil {
			return n, err
		}
	}

	return n, err
}

var (
	nodeKeyPrefix = []byte("node-")
	dataKeyPrefix = []byte("data-")
)

// nodeDBKey constructs a database key for key/node mappings.
func nodeDBKey(addr common.Address, key []byte) []byte {
	return nodeDBKeyHex(addr, common.Bytes2Hex(key))
}

// nodeDBKeyHex constructs a database key for key/node mappings
// using the hexadecimal string representation of the key.
func nodeDBKeyHex(addr common.Address, hexKey string) []byte {
	return append(append(nodeKeyPrefix, []byte(hexKey+"-")...), addr[:]...)
}

// dataDBkey constructs a database key for key/data storage.
func dataDBKey(key []byte) []byte {
	return append(dataKeyPrefix, key...)
}
