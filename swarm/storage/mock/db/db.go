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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
	"time"

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
	// protects nodes and keys indexes
	// in Put and Delete methods
	nodesLocks sync.Map
	keysLocks  sync.Map
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
	has, err := s.db.Has(indexForHashesPerNode(addr, key), nil)
	if err != nil {
		return nil, mock.ErrNotFound
	}
	if !has {
		return nil, mock.ErrNotFound
	}
	data, err = s.db.Get(indexDataKey(key), nil)
	if err == leveldb.ErrNotFound {
		err = mock.ErrNotFound
	}
	return
}

// Put saves the chunk data for node with address addr.
func (s *GlobalStore) Put(addr common.Address, key []byte, data []byte) error {
	unlock, err := s.lock(addr, key)
	if err != nil {
		return err
	}
	defer unlock()

	batch := new(leveldb.Batch)
	batch.Put(indexForHashesPerNode(addr, key), nil)
	batch.Put(indexForNodesWithHash(key, addr), nil)
	batch.Put(indexForNodes(addr), nil)
	batch.Put(indexForHashes(key), nil)
	batch.Put(indexDataKey(key), data)
	return s.db.Write(batch, nil)
}

// Delete removes the chunk reference to node with address addr.
func (s *GlobalStore) Delete(addr common.Address, key []byte) error {
	unlock, err := s.lock(addr, key)
	if err != nil {
		return err
	}
	defer unlock()

	batch := new(leveldb.Batch)
	batch.Delete(indexForHashesPerNode(addr, key))
	batch.Delete(indexForNodesWithHash(key, addr))

	// check if this node contains any keys, and if not
	// remove it from the
	x := indexForHashesPerNodePrefix(addr)
	if k, _ := s.db.Get(x, nil); !bytes.HasPrefix(k, x) {
		batch.Delete(indexForNodes(addr))
	}

	x = indexForNodesWithHashPrefix(key)
	if k, _ := s.db.Get(x, nil); !bytes.HasPrefix(k, x) {
		batch.Delete(indexForHashes(key))
	}
	return s.db.Write(batch, nil)
}

// HasKey returns whether a node with addr contains the key.
func (s *GlobalStore) HasKey(addr common.Address, key []byte) bool {
	has, err := s.db.Has(indexForHashesPerNode(addr, key), nil)
	if err != nil {
		has = false
	}
	return has
}

// Keys returns a paginated list of keys on all nodes.
func (s *GlobalStore) Keys(startKey []byte, limit int) (keys mock.Keys, err error) {
	return s.keys(nil, startKey, limit)
}

// Nodes returns a paginated list of all known nodes.
func (s *GlobalStore) Nodes(startAddr *common.Address, limit int) (nodes mock.Nodes, err error) {
	return s.nodes(nil, startAddr, limit)
}

// NodeKeys returns a paginated list of keys on a node with provided address.
func (s *GlobalStore) NodeKeys(addr common.Address, startKey []byte, limit int) (keys mock.Keys, err error) {
	return s.keys(&addr, startKey, limit)
}

// KeyNodes returns a paginated list of nodes that contain a particular key.
func (s *GlobalStore) KeyNodes(key []byte, startAddr *common.Address, limit int) (nodes mock.Nodes, err error) {
	return s.nodes(key, startAddr, limit)
}

// keys returns a paginated list of keys. If addr is not nil, only keys on that
// node will be returned.
func (s *GlobalStore) keys(addr *common.Address, startKey []byte, limit int) (keys mock.Keys, err error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	if limit <= 0 {
		limit = mock.DefaultLimit
	}

	prefix := []byte{indexForHashesPrefix}
	if addr != nil {
		prefix = indexForHashesPerNodePrefix(*addr)
	}
	if startKey != nil {
		if addr != nil {
			startKey = indexForHashesPerNode(*addr, startKey)
		} else {
			startKey = indexForHashes(startKey)
		}
	} else {
		startKey = prefix
	}

	ok := iter.Seek(startKey)
	if !ok {
		return keys, iter.Error()
	}
	for ; ok; ok = iter.Next() {
		k := iter.Key()
		if !bytes.HasPrefix(k, prefix) {
			break
		}
		key := append([]byte(nil), bytes.TrimPrefix(k, prefix)...)

		if len(keys.Keys) >= limit {
			keys.Next = key
			break
		}

		keys.Keys = append(keys.Keys, key)
	}
	return keys, iter.Error()
}

// nodes returns a paginated list of node addresses. If key is not nil,
// only nodes that contain that key will be returned.
func (s *GlobalStore) nodes(key []byte, startAddr *common.Address, limit int) (nodes mock.Nodes, err error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	if limit <= 0 {
		limit = mock.DefaultLimit
	}

	prefix := []byte{indexForNodesPrefix}
	if key != nil {
		prefix = indexForNodesWithHashPrefix(key)
	}
	startKey := prefix
	if startAddr != nil {
		if key != nil {
			startKey = indexForNodesWithHash(key, *startAddr)
		} else {
			startKey = indexForNodes(*startAddr)
		}
	}

	ok := iter.Seek(startKey)
	if !ok {
		return nodes, iter.Error()
	}
	for ; ok; ok = iter.Next() {
		k := iter.Key()
		if !bytes.HasPrefix(k, prefix) {
			break
		}
		addr := common.BytesToAddress(append([]byte(nil), bytes.TrimPrefix(k, prefix)...))

		if len(nodes.Addrs) >= limit {
			nodes.Next = &addr
			break
		}

		nodes.Addrs = append(nodes.Addrs, addr)
	}
	return nodes, iter.Error()
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

		key := common.Hex2Bytes(hdr.Name)

		batch := new(leveldb.Batch)
		for _, addr := range c.Addrs {
			batch.Put(indexForHashesPerNode(addr, key), nil)
			batch.Put(indexForNodesWithHash(key, addr), nil)
			batch.Put(indexForNodes(addr), nil)
		}

		batch.Put(indexForHashes(key), nil)
		batch.Put(indexDataKey(key), c.Data)

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

	snap, err := s.db.GetSnapshot()
	if err != nil {
		return 0, err
	}

	iter := snap.NewIterator(util.BytesPrefix([]byte{indexForHashesByNodePrefix}), nil)
	defer iter.Release()

	var currentKey string
	var addrs []common.Address

	saveChunk := func() error {
		hexKey := currentKey

		data, err := snap.Get(indexDataKey(common.Hex2Bytes(hexKey)), nil)
		if err != nil {
			return fmt.Errorf("get data %s: %v", hexKey, err)
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
		k := bytes.TrimPrefix(iter.Key(), []byte{indexForHashesByNodePrefix})
		i := bytes.Index(k, []byte{keyTermByte})
		if i < 0 {
			continue
		}
		hexKey := string(k[:i])

		if currentKey == "" {
			currentKey = hexKey
		}

		if hexKey != currentKey {
			if err = saveChunk(); err != nil {
				return n, err
			}

			addrs = addrs[:0]
		}

		currentKey = hexKey
		addrs = append(addrs, common.BytesToAddress(k[i+1:]))
	}

	if len(addrs) > 0 {
		if err = saveChunk(); err != nil {
			return n, err
		}
	}

	return n, iter.Error()
}

var (
	// maximal time for lock to wait until it returns error
	lockTimeout = 3 * time.Second
	// duration between two lock checks.
	lockCheckDelay = 30 * time.Microsecond
	// error returned by lock method when lock timeout is reached
	errLockTimeout = errors.New("lock timeout")
)

// lock protects parallel writes in Put and Delete methods for both
// node with provided address and for data with provided key.
func (s *GlobalStore) lock(addr common.Address, key []byte) (unlock func(), err error) {
	start := time.Now()
	nodeLockKey := addr.Hex()
	for {
		_, loaded := s.nodesLocks.LoadOrStore(nodeLockKey, struct{}{})
		if !loaded {
			break
		}
		time.Sleep(lockCheckDelay)
		if time.Since(start) > lockTimeout {
			return nil, errLockTimeout
		}
	}
	start = time.Now()
	keyLockKey := common.Bytes2Hex(key)
	for {
		_, loaded := s.keysLocks.LoadOrStore(keyLockKey, struct{}{})
		if !loaded {
			break
		}
		time.Sleep(lockCheckDelay)
		if time.Since(start) > lockTimeout {
			return nil, errLockTimeout
		}
	}
	return func() {
		s.nodesLocks.Delete(nodeLockKey)
		s.keysLocks.Delete(keyLockKey)
	}, nil
}

const (
	// prefixes for different indexes
	indexDataPrefix               = 0
	indexForNodesWithHashesPrefix = 1
	indexForHashesByNodePrefix    = 2
	indexForNodesPrefix           = 3
	indexForHashesPrefix          = 4

	// keyTermByte splits keys and node addresses
	// in database keys
	keyTermByte = 0xff
)

// indexForHashesPerNode constructs a database key to store keys used in
// NodeKeys method.
func indexForHashesPerNode(addr common.Address, key []byte) []byte {
	return append(indexForHashesPerNodePrefix(addr), key...)
}

// indexForHashesPerNodePrefix returns a prefix containing a node address used in
// NodeKeys method. Node address is hex encoded to be able to use keyTermByte
// for splitting node address and key.
func indexForHashesPerNodePrefix(addr common.Address) []byte {
	return append([]byte{indexForNodesWithHashesPrefix}, append([]byte(addr.Hex()), keyTermByte)...)
}

// indexForNodesWithHash constructs a database key to store keys used in
// KeyNodes method.
func indexForNodesWithHash(key []byte, addr common.Address) []byte {
	return append(indexForNodesWithHashPrefix(key), addr[:]...)
}

// indexForNodesWithHashPrefix returns a prefix containing a key used in
// KeyNodes method. Key is hex encoded to be able to use keyTermByte
// for splitting key and node address.
func indexForNodesWithHashPrefix(key []byte) []byte {
	return append([]byte{indexForHashesByNodePrefix}, append([]byte(common.Bytes2Hex(key)), keyTermByte)...)
}

// indexForNodes constructs a database key to store keys used in
// Nodes method.
func indexForNodes(addr common.Address) []byte {
	return append([]byte{indexForNodesPrefix}, addr[:]...)
}

// indexForHashes constructs a database key to store keys used in
// Keys method.
func indexForHashes(key []byte) []byte {
	return append([]byte{indexForHashesPrefix}, key...)
}

// indexDataKey constructs a database key for key/data storage.
func indexDataKey(key []byte) []byte {
	return append([]byte{indexDataPrefix}, key...)
}
