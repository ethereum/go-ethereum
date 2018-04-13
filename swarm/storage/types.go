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
	"crypto"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/bmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
)

const MaxPO = 7
const KeyLength = 32

type Hasher func() hash.Hash
type SwarmHasher func() SwarmHash

// Peer is the recorded as Source on the chunk
// should probably not be here? but network should wrap chunk object
type Peer interface{}

type Key []byte

func (x Key) Size() uint {
	return uint(len(x))
}

func (x Key) isEqual(y Key) bool {
	return bytes.Equal(x, y)
}

func (h Key) bits(i, j uint) uint {
	ii := i >> 3
	jj := i & 7
	if ii >= h.Size() {
		return 0
	}

	if jj+j <= 8 {
		return uint((h[ii] >> jj) & ((1 << j) - 1))
	}

	res := uint(h[ii] >> jj)
	jj = 8 - jj
	j -= jj
	for j != 0 {
		ii++
		if j < 8 {
			res += uint(h[ii]&((1<<j)-1)) << jj
			return res
		}
		res += uint(h[ii]) << jj
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

func IsZeroKey(key Key) bool {
	return len(key) == 0 || bytes.Equal(key, ZeroKey)
}

var ZeroKey = Key(common.Hash{}.Bytes())

func MakeHashFunc(hash string) SwarmHasher {
	switch hash {
	case "SHA256":
		return func() SwarmHash { return &HashWithLength{crypto.SHA256.New()} }
	case "SHA3":
		return func() SwarmHash { return &HashWithLength{sha3.NewKeccak256()} }
	case "BMT":
		return func() SwarmHash {
			hasher := sha3.NewKeccak256
			pool := bmt.NewTreePool(hasher, bmt.DefaultSegmentCount, bmt.DefaultPoolSize)
			return bmt.New(pool)
		}
	}
	return nil
}

func (key Key) Hex() string {
	return fmt.Sprintf("%064x", []byte(key[:]))
}

func (key Key) Log() string {
	if len(key[:]) < 8 {
		return fmt.Sprintf("%x", []byte(key[:]))
	}
	return fmt.Sprintf("%016x", []byte(key[:8]))
}

func (key Key) String() string {
	return fmt.Sprintf("%064x", []byte(key)[:])
}

func (key Key) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + key.String() + `"`), nil
}

func (key *Key) UnmarshalJSON(value []byte) error {
	s := string(value)
	*key = make([]byte, 32)
	h := common.Hex2Bytes(s[1 : len(s)-1])
	copy(*key, h)
	return nil
}

type KeyCollection []Key

func NewKeyCollection(l int) KeyCollection {
	return make(KeyCollection, l)
}

func (c KeyCollection) Len() int {
	return len(c)
}

func (c KeyCollection) Less(i, j int) bool {
	return bytes.Compare(c[i], c[j]) == -1
}

func (c KeyCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// Chunk also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size()
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	Key   Key    // always
	SData []byte // nil if request, to be supplied by dpa
	Size  int64  // size of the data covered by the subtree encoded in this chunk
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

func NewChunk(key Key, reqC chan bool) *Chunk {
	return &Chunk{
		Key:        key,
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

func (c *Chunk) WaitToStore() {
	<-c.dbStoredC
}

func FakeChunk(size int64, count int, chunks []*Chunk) int {
	var i int
	hasher := MakeHashFunc(SHA3Hash)()
	if size > DefaultChunkSize {
		size = DefaultChunkSize
	}

	for i = 0; i < count; i++ {
		hasher.Reset()
		chunks[i].SData = make([]byte, size)
		rand.Read(chunks[i].SData)
		binary.LittleEndian.PutUint64(chunks[i].SData[:8], uint64(size))
		hasher.Write(chunks[i].SData)
		chunks[i].Key = make([]byte, 32)
		copy(chunks[i].Key, hasher.Sum(nil))
	}

	return i
}

// Size, Seek, Read, ReadAt
type LazySectionReader interface {
	Size(chan bool) (int64, error)
	io.Seeker
	io.Reader
	io.ReaderAt
}

type LazyTestSectionReader struct {
	*io.SectionReader
}

func (self *LazyTestSectionReader) Size(chan bool) (int64, error) {
	return self.SectionReader.Size(), nil
}

type StoreParams struct {
	Hash          SwarmHasher `toml:"-"`
	DbCapacity    uint64
	CacheCapacity uint
	BaseKey       []byte
}

func NewStoreParams(capacity uint64, hash SwarmHasher, basekey []byte) *StoreParams {
	if basekey == nil {
		basekey = make([]byte, 32)
	}
	if hash == nil {
		hash = MakeHashFunc("SHA3")
	}
	if capacity == 0 {
		capacity = defaultDbCapacity
	}
	return &StoreParams{
		Hash:          hash,
		DbCapacity:    capacity,
		CacheCapacity: defaultCacheCapacity,
		BaseKey:       basekey,
	}
}

type ChunkData []byte

type Reference []byte

// Putter is responsible to store data and create a reference for it
type Putter interface {
	Put(ChunkData) (Reference, error)
	// RefSize returns the length of the Reference created by this Putter
	RefSize() int64
	// Close is to indicate that no more chunk data will be Put on this Putter
	Close()
	// Wait returns if all data has been store and the Close() was called.
	Wait()
}

// Getter is an interface to retrieve a chunk's data by its reference
type Getter interface {
	Get(Reference) (ChunkData, error)
}

// NOTE: this returns invalid data if chunk is encrypted
func (c ChunkData) Size() int64 {
	return int64(binary.LittleEndian.Uint64(c[:8]))
}

func (c ChunkData) Data() []byte {
	return c[8:]
}

type ChunkValidator interface {
	Validate(key Key, data []byte) bool
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
func (self *ContentAddressValidator) Validate(key Key, data []byte) bool {
	hasher := self.Hasher()
	hasher.ResetWithLength(data[:8])
	hasher.Write(data[8:])
	hash := hasher.Sum(nil)

	if !bytes.Equal(hash, key[:]) {
		log.Error("invalid content address", "expected", fmt.Sprintf("%x", hash), "have", key)
		return false
	}
	return true
}
