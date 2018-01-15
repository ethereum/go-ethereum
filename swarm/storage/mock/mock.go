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

// ErrNotFound is in all NodeStorer implementations
// to indicate that the chunk is not found.
var ErrNotFound = errors.New("not found")

// GlobalStorer defines methods for mock db store
// that stores chunk data for all swarm nodes.
// It is used in tests to construct mock NodeStores
// for swarm nodes and to track and validate chunks.
type GlobalStorer interface {
	Get(addr common.Address, key []byte) (data []byte, err error)
	Put(addr common.Address, key []byte, data []byte) error
	HasKey(addr common.Address, key []byte) bool
	// NewNodeStore creates an instance of NodeStorer
	// to be used by a single swarm node with
	// address addr.
	NewNodeStore(addr common.Address) NodeStorer
}

// NodeStorer defines methods that are required
// for accessing and storing chunk data.
// It is used for baypassing chunk data storing in
// storage.DbStore.
type NodeStorer interface {
	Get(key []byte) (data []byte, err error)
	Put(key []byte, data []byte) error
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
