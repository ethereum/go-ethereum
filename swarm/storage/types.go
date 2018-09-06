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
	"hash"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/swarm/bmt"
	"github.com/ethereum/go-ethereum/swarm/chunk"
)

const MaxPO = 16
const KeyLength = 32

type Hasher func() hash.Hash
type SwarmHasher func() SwarmHash

// Peer is the recorded as Source on the chunk
// should probably not be here? but network should wrap chunk object
type Peer interface{}

type Address []byte

func (a Address) Size() uint {
	return uint(len(a))
}

func (a Address) isEqual(y Address) bool {
	return bytes.Equal(a, y)
}

func (a Address) bits(i, j uint) uint {
	ii := i >> 3
	jj := i & 7
	if ii >= a.Size() {
		return 0
	}

	if jj+j <= 8 {
		return uint((a[ii] >> jj) & ((1 << j) - 1))
	}

	res := uint(a[ii] >> jj)
	jj = 8 - jj
	j -= jj
	for j != 0 {
		ii++
		if j < 8 {
			res += uint(a[ii]&((1<<j)-1)) << jj
			return res
		}
		res += uint(a[ii]) << jj
		jj += 8
		j -= 8
	}
	return res
}

func Proximity(one, other []byte) (ret int) {
	b := (MaxPO-1)/8 + 1
	if b > len(one) {
		b = len(one)
	}
	m := 8
	for i := 0; i < b; i++ {
		oxo := one[i] ^ other[i]
		if i == b-1 {
			m = MaxPO % 8
		}
		for j := 0; j < m; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return MaxPO
}

func IsZeroAddr(addr Address) bool {
	return len(addr) == 0 || bytes.Equal(addr, ZeroAddr)
}

var ZeroAddr = Address(common.Hash{}.Bytes())

func MakeHashFunc(hash string) SwarmHasher {
	switch hash {
	case "SHA256":
		return func() SwarmHash { return &HashWithLength{crypto.SHA256.New()} }
	case "SHA3":
		return func() SwarmHash { return &HashWithLength{sha3.NewKeccak256()} }
	case "BMT":
		return func() SwarmHash {
			hasher := sha3.NewKeccak256
			hasherSize := hasher().Size()
			segmentCount := chunk.DefaultSize / hasherSize
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
	return fmt.Sprintf("%064x", []byte(a)[:])
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

// Chunk also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size()
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	Addr  Address // always
	SData []byte  // nil if request, to be supplied by dpa
	Size  int64   // size of the data covered by the subtree encoded in this chunk
	//Source   Peer           // peer
	C          chan bool // to signal data delivery by the dpa
	ReqC       chan bool // to signal the request done
	dbStoredC  chan bool // never remove a chunk from memStore before it is written to dbStore
	dbStored   bool
	dbStoredMu *sync.Mutex
	errored    error // flag which is set when the chunk request has errored or timeouted
	erroredMu  sync.Mutex
}

func (c *Chunk) SetErrored(err error) {
	c.erroredMu.Lock()
	defer c.erroredMu.Unlock()

	c.errored = err
}

func (c *Chunk) GetErrored() error {
	c.erroredMu.Lock()
	defer c.erroredMu.Unlock()

	return c.errored
}

func NewChunk(addr Address, reqC chan bool) *Chunk {
	return &Chunk{
		Addr:       addr,
		ReqC:       reqC,
		dbStoredC:  make(chan bool),
		dbStoredMu: &sync.Mutex{},
	}
}

func (c *Chunk) markAsStored() {
	c.dbStoredMu.Lock()
	defer c.dbStoredMu.Unlock()

	if !c.dbStored {
		close(c.dbStoredC)
		c.dbStored = true
	}
}

func (c *Chunk) WaitToStore() error {
	<-c.dbStoredC
	return c.GetErrored()
}

func GenerateRandomChunk(dataSize int64) *Chunk {
	return GenerateRandomChunks(dataSize, 1)[0]
}

func GenerateRandomChunks(dataSize int64, count int) (chunks []*Chunk) {
	var i int
	hasher := MakeHashFunc(DefaultHash)()
	if dataSize > chunk.DefaultSize {
		dataSize = chunk.DefaultSize
	}

	for i = 0; i < count; i++ {
		chunks = append(chunks, NewChunk(nil, nil))
		chunks[i].SData = make([]byte, dataSize+8)
		rand.Read(chunks[i].SData)
		binary.LittleEndian.PutUint64(chunks[i].SData[:8], uint64(dataSize))
		hasher.ResetWithLength(chunks[i].SData[:8])
		hasher.Write(chunks[i].SData[8:])
		chunks[i].Addr = make([]byte, 32)
		copy(chunks[i].Addr, hasher.Sum(nil))
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
	Hash                       SwarmHasher `toml:"-"`
	DbCapacity                 uint64
	CacheCapacity              uint
	ChunkRequestsCacheCapacity uint
	BaseKey                    []byte
}

func NewDefaultStoreParams() *StoreParams {
	return NewStoreParams(defaultLDBCapacity, defaultCacheCapacity, defaultChunkRequestsCacheCapacity, nil, nil)
}

func NewStoreParams(ldbCap uint64, cacheCap uint, requestsCap uint, hash SwarmHasher, basekey []byte) *StoreParams {
	if basekey == nil {
		basekey = make([]byte, 32)
	}
	if hash == nil {
		hash = MakeHashFunc(DefaultHash)
	}
	return &StoreParams{
		Hash:                       hash,
		DbCapacity:                 ldbCap,
		CacheCapacity:              cacheCap,
		ChunkRequestsCacheCapacity: requestsCap,
		BaseKey:                    basekey,
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
func (c ChunkData) Size() int64 {
	return int64(binary.LittleEndian.Uint64(c[:8]))
}

func (c ChunkData) Data() []byte {
	return c[8:]
}

type ChunkValidator interface {
	Validate(addr Address, data []byte) bool
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
func (v *ContentAddressValidator) Validate(addr Address, data []byte) bool {
	if l := len(data); l < 9 || l > chunk.DefaultSize+8 {
		return false
	}

	hasher := v.Hasher()
	hasher.ResetWithLength(data[:8])
	hasher.Write(data[8:])
	hash := hasher.Sum(nil)

	return bytes.Equal(hash, addr[:])
}
