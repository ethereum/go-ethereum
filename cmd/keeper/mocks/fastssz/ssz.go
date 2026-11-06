// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package ssz stands in for github.com/ferranbt/fastssz in the keeper (see
// the replace directive in its go.mod). The keeper links internal/era through
// core/rawdb's chain freezer but never computes an era accumulator, and the
// real package's initialisation (64 SHA-256 zero hashes, gopkg.in/yaml.v2's
// regular expressions) costs a zkVM guest some ten million instructions. The
// API internal/era compiles against is declared here, and panics if reached.
package ssz

// HashWalker is the hashing interface a HashRoot writes itself into.
type HashWalker interface {
	Hash() []byte
	AppendUint8(i uint8)
	AppendUint32(i uint32)
	AppendUint64(i uint64)
	AppendBytes32(b []byte)
	PutUint64Array(b []uint64, maxCapacity ...uint64)
	PutUint64(i uint64)
	PutUint32(i uint32)
	PutUint16(i uint16)
	PutUint8(i uint8)
	FillUpTo32()
	Append(i []byte)
	PutBitlist(bb []byte, maxSize uint64)
	PutBool(b bool)
	PutBytes(b []byte)
	Index() int
	Merkleize(indx int)
	MerkleizeWithMixin(indx int, num, limit uint64)
}

// HashRoot is an object that can compute its SSZ hash tree root.
type HashRoot interface {
	GetTree() (*Node, error)
	HashTreeRoot() ([32]byte, error)
	HashTreeRootWith(hh HashWalker) error
}

// Node is a node of an SSZ Merkle tree.
type Node struct{}

// Hasher computes SSZ hash tree roots.
type Hasher struct{}

func unavailable() {
	panic("fastssz: mocked out of the keeper build (see cmd/keeper/mocks/fastssz)")
}

// NewHasher panics: SSZ hashing is not available in the keeper.
func NewHasher() *Hasher { unavailable(); return nil }

// HashWithDefaultHasher panics: SSZ hashing is not available in the keeper.
func HashWithDefaultHasher(v HashRoot) ([32]byte, error) { unavailable(); return [32]byte{}, nil }

func (h *Hasher) Hash() []byte                                     { unavailable(); return nil }
func (h *Hasher) AppendUint8(i uint8)                              { unavailable() }
func (h *Hasher) AppendUint32(i uint32)                            { unavailable() }
func (h *Hasher) AppendUint64(i uint64)                            { unavailable() }
func (h *Hasher) AppendBytes32(b []byte)                           { unavailable() }
func (h *Hasher) PutUint64Array(b []uint64, maxCapacity ...uint64) { unavailable() }
func (h *Hasher) PutUint64(i uint64)                               { unavailable() }
func (h *Hasher) PutUint32(i uint32)                               { unavailable() }
func (h *Hasher) PutUint16(i uint16)                               { unavailable() }
func (h *Hasher) PutUint8(i uint8)                                 { unavailable() }
func (h *Hasher) FillUpTo32()                                      { unavailable() }
func (h *Hasher) Append(i []byte)                                  { unavailable() }
func (h *Hasher) PutBitlist(bb []byte, maxSize uint64)             { unavailable() }
func (h *Hasher) PutBool(b bool)                                   { unavailable() }
func (h *Hasher) PutBytes(b []byte)                                { unavailable() }
func (h *Hasher) Index() int                                       { unavailable(); return 0 }
func (h *Hasher) Merkleize(indx int)                               { unavailable() }
func (h *Hasher) MerkleizeWithMixin(indx int, num, limit uint64)   { unavailable() }
func (h *Hasher) HashRoot() ([32]byte, error)                      { unavailable(); return [32]byte{}, nil }
