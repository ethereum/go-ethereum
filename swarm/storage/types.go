// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/bmt"
	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"golang.org/x/crypto/sha3"
)

const MaxPO = 16
const AddressLength = 32

type SwarmHasher func() SwarmHash

type Address []byte

// Proximity(x, y) returns the proximity order of the MSB distance between x and y
//
// The distance metric MSB(x, y) of two equal length byte sequences x an y is the
// value of the binary integer cast of the x^y, ie., x and y bitwise xor-ed.
// the binary cast is big endian: most significant bit first (=MSB).
//
// Proximity(x, y) is a discrete logarithmic scaling of the MSB distance.
// It is defined as the reverse rank of the integer part of the base 2
// logarithm of the distance.
// It is calculated by counting the number of common leading zeros in the (MSB)
// binary representation of the x^y.
//
// (0 farthest, 255 closest, 256 self)
func Proximity(one, other []byte) (ret int) {
	b := (MaxPO-1)/8 + 1
	if b > len(one) {
		b = len(one)
	}
	m := 8
	for i := 0; i < b; i++ {
		oxo := one[i] ^ other[i]
		for j := 0; j < m; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return MaxPO
}

var ZeroAddr = Address(common.Hash{}.Bytes())

func MakeHashFunc(hash string) SwarmHasher {
	switch hash {
	case "SHA256":
		return func() SwarmHash { return &HashWithLength{crypto.SHA256.New()} }
	case "SHA3":
		return func() SwarmHash { return &HashWithLength{sha3.NewLegacyKeccak256()} }
	case "BMT":
		return func() SwarmHash {
			hasher := sha3.NewLegacyKeccak256
			hasherSize := hasher().Size()
			segmentCount := ch.DefaultSize / hasherSize
			pool := bmt.NewTreePool(hasher, segmentCount, bmt.PoolSize)
			return bmt.New(pool)
		}
	}
	return nil
}

func (a Address) Hex() string {
	return fmt.Sprintf("%064x", []byte(a[:]))
}

func (a Address) Log() string {
	if len(a[:]) < 8 {
		return fmt.Sprintf("%x", []byte(a[:]))
	}
	return fmt.Sprintf("%016x", []byte(a[:8]))
}

func (a Address) String() string {
	return fmt.Sprintf("%064x", []byte(a))
}

func (a Address) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + a.String() + `"`), nil
}

func (a *Address) UnmarshalJSON(value []byte) error {
	s := string(value)
	*a = make([]byte, 32)
	h := common.Hex2Bytes(s[1 : len(s)-1])
	copy(*a, h)
	return nil
}

type AddressCollection []Address

func NewAddressCollection(l int) AddressCollection {
	return make(AddressCollection, l)
}

func (c AddressCollection) Len() int {
	return len(c)
}

func (c AddressCollection) Less(i, j int) bool {
	return bytes.Compare(c[i], c[j]) == -1
}

func (c AddressCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Chunk interface implemented by context.Contexts and data chunks
type Chunk interface {
	Address() Address
	Data() []byte
}

type chunk struct {
	addr  Address
	sdata []byte
	span  int64
}

func NewChunk(addr Address, data []byte) *chunk {
	return &chunk{
		addr:  addr,
		sdata: data,
		span:  -1,
	}
}

func (c *chunk) Address() Address {
	return c.addr
}

func (c *chunk) Data() []byte {
	return c.sdata
}

// String() for pretty printing
func (self *chunk) String() string {
	return fmt.Sprintf("Address: %v TreeSize: %v Chunksize: %v", self.addr.Log(), self.span, len(self.sdata))
}

func GenerateRandomChunk(dataSize int64) Chunk {
	hasher := MakeHashFunc(DefaultHash)()
	sdata := make([]byte, dataSize+8)
	rand.Read(sdata[8:])
	binary.LittleEndian.PutUint64(sdata[:8], uint64(dataSize))
	hasher.ResetWithLength(sdata[:8])
	hasher.Write(sdata[8:])
	return NewChunk(hasher.Sum(nil), sdata)
}

func GenerateRandomChunks(dataSize int64, count int) (chunks []Chunk) {
	for i := 0; i < count; i++ {
		ch := GenerateRandomChunk(dataSize)
		chunks = append(chunks, ch)
	}
	return chunks
}

// Size, Seek, Read, ReadAt
type LazySectionReader interface {
	Context() context.Context
	Size(context.Context, chan bool) (int64, error)
	io.Seeker
	io.Reader
	io.ReaderAt
}

type LazyTestSectionReader struct {
	*io.SectionReader
}

func (r *LazyTestSectionReader) Size(context.Context, chan bool) (int64, error) {
	return r.SectionReader.Size(), nil
}

func (r *LazyTestSectionReader) Context() context.Context {
	return context.TODO()
}

type StoreParams struct {
	Hash          SwarmHasher `toml:"-"`
	DbCapacity    uint64
	CacheCapacity uint
	BaseKey       []byte
}

func NewDefaultStoreParams() *StoreParams {
	return NewStoreParams(defaultLDBCapacity, defaultCacheCapacity, nil, nil)
}

func NewStoreParams(ldbCap uint64, cacheCap uint, hash SwarmHasher, basekey []byte) *StoreParams {
	if basekey == nil {
		basekey = make([]byte, 32)
	}
	if hash == nil {
		hash = MakeHashFunc(DefaultHash)
	}
	return &StoreParams{
		Hash:          hash,
		DbCapacity:    ldbCap,
		CacheCapacity: cacheCap,
		BaseKey:       basekey,
	}
}

type ChunkData []byte

type Reference []byte

// Putter is responsible to store data and create a reference for it
type Putter interface {
	Put(context.Context, ChunkData) (Reference, error)
	// RefSize returns the length of the Reference created by this Putter
	RefSize() int64
	// Close is to indicate that no more chunk data will be Put on this Putter
	Close()
	// Wait returns if all data has been store and the Close() was called.
	Wait(context.Context) error
}

// Getter is an interface to retrieve a chunk's data by its reference
type Getter interface {
	Get(context.Context, Reference) (ChunkData, error)
}

// NOTE: this returns invalid data if chunk is encrypted
func (c ChunkData) Size() uint64 {
	return binary.LittleEndian.Uint64(c[:8])
}

type ChunkValidator interface {
	Validate(chunk Chunk) bool
}

// Provides method for validation of content address in chunks
// Holds the corresponding hasher to create the address
type ContentAddressValidator struct {
	Hasher SwarmHasher
}

// Constructor
func NewContentAddressValidator(hasher SwarmHasher) *ContentAddressValidator {
	return &ContentAddressValidator{
		Hasher: hasher,
	}
}

// Validate that the given key is a valid content address for the given data
func (v *ContentAddressValidator) Validate(chunk Chunk) bool {
	data := chunk.Data()
	if l := len(data); l < 9 || l > ch.DefaultSize+8 {
		// log.Error("invalid chunk size", "chunk", addr.Hex(), "size", l)
		return false
	}

	hasher := v.Hasher()
	hasher.ResetWithLength(data[:8])
	hasher.Write(data[8:])
	hash := hasher.Sum(nil)

	return bytes.Equal(hash, chunk.Address())
}

type ChunkStore interface {
	Put(ctx context.Context, ch Chunk) (err error)
	Get(rctx context.Context, ref Address) (ch Chunk, err error)
	Close()
}

// SyncChunkStore is a ChunkStore which supports syncing
type SyncChunkStore interface {
	ChunkStore
	BinIndex(po uint8) uint64
	Iterator(from uint64, to uint64, po uint8, f func(Address, uint64) bool) error
	FetchFunc(ctx context.Context, ref Address) func(context.Context) error
}

// FakeChunkStore doesn't store anything, just implements the ChunkStore interface
// It can be used to inject into a hasherStore if you don't want to actually store data just do the
// hashing
type FakeChunkStore struct {
}

// Put doesn't store anything it is just here to implement ChunkStore
func (f *FakeChunkStore) Put(_ context.Context, ch Chunk) error {
	return nil
}

// Gut doesn't store anything it is just here to implement ChunkStore
func (f *FakeChunkStore) Get(_ context.Context, ref Address) (Chunk, error) {
	panic("FakeChunkStore doesn't support Get")
}

// Close doesn't store anything it is just here to implement ChunkStore
func (f *FakeChunkStore) Close() {
}
