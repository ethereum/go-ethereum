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

// Package mock defines types that are used by different implementations
// of mock storages.
//
// Implementations of mock storages are located in directories
// under this package:
//
//  - db - LevelDB backend
//  - mem - in memory map backend
//  - rpc - RPC client that can connect to other backends
//
// Mock storages can implement Importer and Exporter interfaces
// for importing and exporting all chunk data that they contain.
// The exported file is a tar archive with all files named by
// hexadecimal representations of chunk keys and with content
// with JSON-encoded ExportedChunk structure. Exported format
// should be preserved across all mock store implementations.
package mock

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

// ErrNotFound indicates that the chunk is not found.
var ErrNotFound = errors.New("not found")

// NodeStore holds the node address and a reference to the GlobalStore
// in order to access and store chunk data only for one node.
type NodeStore struct {
	store GlobalStorer
	addr  common.Address
}

// NewNodeStore creates a new instance of NodeStore that keeps
// chunk data using GlobalStorer with a provided address.
func NewNodeStore(addr common.Address, store GlobalStorer) *NodeStore {
	return &NodeStore{
		store: store,
		addr:  addr,
	}
}

// Get returns chunk data for a key for a node that has the address
// provided on NodeStore initialization.
func (n *NodeStore) Get(key []byte) (data []byte, err error) {
	return n.store.Get(n.addr, key)
}

// Put saves chunk data for a key for a node that has the address
// provided on NodeStore initialization.
func (n *NodeStore) Put(key []byte, data []byte) error {
	return n.store.Put(n.addr, key, data)
}

// GlobalStorer defines methods for mock db store
// that stores chunk data for all swarm nodes.
// It is used in tests to construct mock NodeStores
// for swarm nodes and to track and validate chunks.
type GlobalStorer interface {
	Get(addr common.Address, key []byte) (data []byte, err error)
	Put(addr common.Address, key []byte, data []byte) error
	HasKey(addr common.Address, key []byte) bool
	// NewNodeStore creates an instance of NodeStore
	// to be used by a single swarm node with
	// address addr.
	NewNodeStore(addr common.Address) *NodeStore
}

// Importer defines method for importing mock store data
// from an exported tar archive.
type Importer interface {
	Import(r io.Reader) (n int, err error)
}

// Exporter defines method for exporting mock store data
// to a tar archive.
type Exporter interface {
	Export(w io.Writer) (n int, err error)
}

// ImportExporter is an interface for importing and exporting
// mock store data to and from a tar archive.
type ImportExporter interface {
	Importer
	Exporter
}

// ExportedChunk is the structure that is saved in tar archive for
// each chunk as JSON-encoded bytes.
type ExportedChunk struct {
	Data  []byte           `json:"d"`
	Addrs []common.Address `json:"a"`
}
