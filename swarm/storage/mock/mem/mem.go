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

// Package mem implements a mock store that keeps all chunk data in memory.
// While it can be used for testing on smaller scales, the main purpose of this
// package is to provide the simplest reference implementation of a mock store.
package mem

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

// GlobalStore stores all chunk data and also keys and node addresses relations.
// It implements mock.GlobalStore interface.
type GlobalStore struct {
	// holds a slice of keys per node
	nodeKeys map[common.Address][][]byte
	// holds which key is stored on which nodes
	keyNodes map[string][]common.Address
	// all node addresses
	nodes []common.Address
	// all keys
	keys [][]byte
	// all keys data
	data map[string][]byte
	mu   sync.RWMutex
}

// NewGlobalStore creates a new instance of GlobalStore.
func NewGlobalStore() *GlobalStore {
	return &GlobalStore{
		nodeKeys: make(map[common.Address][][]byte),
		keyNodes: make(map[string][]common.Address),
		nodes:    make([]common.Address, 0),
		keys:     make([][]byte, 0),
		data:     make(map[string][]byte),
	}
}

// NewNodeStore returns a new instance of NodeStore that retrieves and stores
// chunk data only for a node with address addr.
func (s *GlobalStore) NewNodeStore(addr common.Address) *mock.NodeStore {
	return mock.NewNodeStore(addr, s)
}

// Get returns chunk data if the chunk with key exists for node
// on address addr.
func (s *GlobalStore) Get(addr common.Address, key []byte) (data []byte, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, has := s.nodeKeyIndex(addr, key); !has {
		return nil, mock.ErrNotFound
	}

	data, ok := s.data[string(key)]
	if !ok {
		return nil, mock.ErrNotFound
	}
	return data, nil
}

// Put saves the chunk data for node with address addr.
func (s *GlobalStore) Put(addr common.Address, key []byte, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if i, found := s.nodeKeyIndex(addr, key); !found {
		s.nodeKeys[addr] = append(s.nodeKeys[addr], nil)
		copy(s.nodeKeys[addr][i+1:], s.nodeKeys[addr][i:])
		s.nodeKeys[addr][i] = key
	}

	if i, found := s.keyNodeIndex(key, addr); !found {
		k := string(key)
		s.keyNodes[k] = append(s.keyNodes[k], addr)
		copy(s.keyNodes[k][i+1:], s.keyNodes[k][i:])
		s.keyNodes[k][i] = addr
	}

	if i, found := s.nodeIndex(addr); !found {
		s.nodes = append(s.nodes, addr)
		copy(s.nodes[i+1:], s.nodes[i:])
		s.nodes[i] = addr
	}

	if i, found := s.keyIndex(key); !found {
		s.keys = append(s.keys, nil)
		copy(s.keys[i+1:], s.keys[i:])
		s.keys[i] = key
	}

	s.data[string(key)] = data

	return nil
}

// Delete removes the chunk data for node with address addr.
func (s *GlobalStore) Delete(addr common.Address, key []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if i, has := s.nodeKeyIndex(addr, key); has {
		s.nodeKeys[addr] = append(s.nodeKeys[addr][:i], s.nodeKeys[addr][i+1:]...)
	}

	k := string(key)
	if i, on := s.keyNodeIndex(key, addr); on {
		s.keyNodes[k] = append(s.keyNodes[k][:i], s.keyNodes[k][i+1:]...)
	}

	if len(s.nodeKeys[addr]) == 0 {
		if i, found := s.nodeIndex(addr); found {
			s.nodes = append(s.nodes[:i], s.nodes[i+1:]...)
		}
	}

	if len(s.keyNodes[k]) == 0 {
		if i, found := s.keyIndex(key); found {
			s.keys = append(s.keys[:i], s.keys[i+1:]...)
		}
	}
	return nil
}

// HasKey returns whether a node with addr contains the key.
func (s *GlobalStore) HasKey(addr common.Address, key []byte) (yes bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, yes = s.nodeKeyIndex(addr, key)
	return yes
}

// keyIndex returns the index of a key in keys slice.
func (s *GlobalStore) keyIndex(key []byte) (index int, found bool) {
	l := len(s.keys)
	index = sort.Search(l, func(i int) bool {
		return bytes.Compare(s.keys[i], key) >= 0
	})
	found = index < l && bytes.Equal(s.keys[index], key)
	return index, found
}

// nodeIndex returns the index of a node address in nodes slice.
func (s *GlobalStore) nodeIndex(addr common.Address) (index int, found bool) {
	l := len(s.nodes)
	index = sort.Search(l, func(i int) bool {
		return bytes.Compare(s.nodes[i][:], addr[:]) >= 0
	})
	found = index < l && bytes.Equal(s.nodes[index][:], addr[:])
	return index, found
}

// nodeKeyIndex returns the index of a key in nodeKeys slice.
func (s *GlobalStore) nodeKeyIndex(addr common.Address, key []byte) (index int, found bool) {
	l := len(s.nodeKeys[addr])
	index = sort.Search(l, func(i int) bool {
		return bytes.Compare(s.nodeKeys[addr][i], key) >= 0
	})
	found = index < l && bytes.Equal(s.nodeKeys[addr][index], key)
	return index, found
}

// keyNodeIndex returns the index of a node address in keyNodes slice.
func (s *GlobalStore) keyNodeIndex(key []byte, addr common.Address) (index int, found bool) {
	k := string(key)
	l := len(s.keyNodes[k])
	index = sort.Search(l, func(i int) bool {
		return bytes.Compare(s.keyNodes[k][i][:], addr[:]) >= 0
	})
	found = index < l && s.keyNodes[k][index] == addr
	return index, found
}

// Keys returns a paginated list of keys on all nodes.
func (s *GlobalStore) Keys(startKey []byte, limit int) (keys mock.Keys, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var i int
	if startKey != nil {
		i, _ = s.keyIndex(startKey)
	}
	total := len(s.keys)
	max := maxIndex(i, limit, total)
	keys.Keys = make([][]byte, 0, max-i)
	for ; i < max; i++ {
		keys.Keys = append(keys.Keys, append([]byte(nil), s.keys[i]...))
	}
	if total > max {
		keys.Next = s.keys[max]
	}
	return keys, nil
}

// Nodes returns a paginated list of all known nodes.
func (s *GlobalStore) Nodes(startAddr *common.Address, limit int) (nodes mock.Nodes, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var i int
	if startAddr != nil {
		i, _ = s.nodeIndex(*startAddr)
	}
	total := len(s.nodes)
	max := maxIndex(i, limit, total)
	nodes.Addrs = make([]common.Address, 0, max-i)
	for ; i < max; i++ {
		nodes.Addrs = append(nodes.Addrs, s.nodes[i])
	}
	if total > max {
		nodes.Next = &s.nodes[max]
	}
	return nodes, nil
}

// NodeKeys returns a paginated list of keys on a node with provided address.
func (s *GlobalStore) NodeKeys(addr common.Address, startKey []byte, limit int) (keys mock.Keys, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var i int
	if startKey != nil {
		i, _ = s.nodeKeyIndex(addr, startKey)
	}
	total := len(s.nodeKeys[addr])
	max := maxIndex(i, limit, total)
	keys.Keys = make([][]byte, 0, max-i)
	for ; i < max; i++ {
		keys.Keys = append(keys.Keys, append([]byte(nil), s.nodeKeys[addr][i]...))
	}
	if total > max {
		keys.Next = s.nodeKeys[addr][max]
	}
	return keys, nil
}

// KeyNodes returns a paginated list of nodes that contain a particular key.
func (s *GlobalStore) KeyNodes(key []byte, startAddr *common.Address, limit int) (nodes mock.Nodes, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var i int
	if startAddr != nil {
		i, _ = s.keyNodeIndex(key, *startAddr)
	}
	total := len(s.keyNodes[string(key)])
	max := maxIndex(i, limit, total)
	nodes.Addrs = make([]common.Address, 0, max-i)
	for ; i < max; i++ {
		nodes.Addrs = append(nodes.Addrs, s.keyNodes[string(key)][i])
	}
	if total > max {
		nodes.Next = &s.keyNodes[string(key)][max]
	}
	return nodes, nil
}

// maxIndex returns the end index for one page listing
// based on the start index, limit and total number of elements.
func maxIndex(start, limit, total int) (max int) {
	if limit <= 0 {
		limit = mock.DefaultLimit
	}
	if limit > mock.MaxLimit {
		limit = mock.MaxLimit
	}
	max = total
	if start+limit < max {
		max = start + limit
	}
	return max
}

// Import reads tar archive from a reader that contains exported chunk data.
// It returns the number of chunks imported and an error.
func (s *GlobalStore) Import(r io.Reader) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		s.keyNodes[string(key)] = c.Addrs
		for _, addr := range c.Addrs {
			if i, has := s.nodeKeyIndex(addr, key); !has {
				s.nodeKeys[addr] = append(s.nodeKeys[addr], nil)
				copy(s.nodeKeys[addr][i+1:], s.nodeKeys[addr][i:])
				s.nodeKeys[addr][i] = key
			}
			if i, found := s.nodeIndex(addr); !found {
				s.nodes = append(s.nodes, addr)
				copy(s.nodes[i+1:], s.nodes[i:])
				s.nodes[i] = addr
			}
		}
		if i, found := s.keyIndex(key); !found {
			s.keys = append(s.keys, nil)
			copy(s.keys[i+1:], s.keys[i:])
			s.keys[i] = key
		}
		s.data[string(key)] = c.Data
		n++
	}
	return n, err
}

// Export writes to a writer a tar archive with all chunk data from
// the store. It returns the number of chunks exported and an error.
func (s *GlobalStore) Export(w io.Writer) (n int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tw := tar.NewWriter(w)
	defer tw.Close()

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	encoder := json.NewEncoder(buf)
	for key, addrs := range s.keyNodes {
		buf.Reset()
		if err = encoder.Encode(mock.ExportedChunk{
			Addrs: addrs,
			Data:  s.data[key],
		}); err != nil {
			return n, err
		}

		data := buf.Bytes()
		hdr := &tar.Header{
			Name: common.Bytes2Hex([]byte(key)),
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return n, err
		}
		if _, err := tw.Write(data); err != nil {
			return n, err
		}
		n++
	}
	return n, err
}
