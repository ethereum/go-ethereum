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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

type Hasher func() hash.Hash

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

/*
func proximity(one, other []byte) (ret int) {
	retbig, _ := binary.Varint(other)
	ret = int(int8(retbig))
	return
}*/
func Proximity(one, other []byte) (ret int) {
	for i := 0; i < len(one); i++ {
		oxo := one[i] ^ other[i]
		for j := 0; j < 8; j++ {
			if (uint8(oxo)>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return len(one) * 8
}

func IsZeroKey(key Key) bool {
	return len(key) == 0 || bytes.Equal(key, ZeroKey)
}

var ZeroKey = Key(common.Hash{}.Bytes())

func MakeHashFunc(hash string) Hasher {
	switch hash {
	case "SHA256":
		return crypto.SHA256.New
	case "SHA3":
		return sha3.NewKeccak256
	}
	return nil
}

func (key Key) Hex() string {
	return fmt.Sprintf("%064x", []byte(key[:]))
}

func (key Key) Log() string {
	if len(key[:]) < 4 {
		return fmt.Sprintf("%x", []byte(key[:]))
	}
	return fmt.Sprintf("%08x", []byte(key[:4]))
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
	if bytes.Compare(c[i], c[j]) == -1 {
		return true
	}
	return false
}

func (c KeyCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// each chunk when first requested opens a record associated with the request
// next time a request for the same chunk arrives, this record is updated
// this request status keeps track of the request ID-s as well as the requesting
// peers and has a channel that is closed when the chunk is retrieved. Multiple
// local callers can wait on this channel (or combined with a timeout, block with a
// select).
type RequestStatus struct {
	Key        Key
	Source     Peer
	C          chan bool
	Requesters map[uint64][]interface{}
}

func newRequestStatus(key Key) *RequestStatus {
	return &RequestStatus{
		Key:        key,
		Requesters: make(map[uint64][]interface{}),
		C:          make(chan bool),
	}
}

// Chunk also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size()
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	Key      Key             // always
	SData    []byte          // nil if request, to be supplied by dpa
	Size     int64           // size of the data covered by the subtree encoded in this chunk
	Source   Peer            // peer
	C        chan bool       // to signal data delivery by the dpa
	Req      *RequestStatus  // request Status needed by netStore
	wg       *sync.WaitGroup // wg to synchronize
	dbStored chan bool       // never remove a chunk from memStore before it is written to dbStore
}

func NewChunk(key Key, rs *RequestStatus) *Chunk {
	return &Chunk{Key: key, Req: rs}
}

func FakeChunk(size int64, count int, chunks []Chunk) int {
	var i int
	hasher := MakeHashFunc(defaultHash)()
	chunksize := getDefaultChunkSize()
	if size > chunksize {
		size = chunksize
	}

	for i = 0; i < count; i++ {
		/*
			hasher.Reset()
			data := make([]byte, size)
			rand.Read(data)
			binary.LittleEndian.PutUint64(data[8:], uint64(size))
			hasher.Write(data)
			chunks[i].SData = make([]byte, chunksize)
			copy(chunks[i].SData, hasher.Sum(nil))
		*/
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

func getDefaultChunkSize() int64 {
	return defaultBranches * int64(MakeHashFunc(defaultHash)().Size())

}

/*
The ChunkStore interface is implemented by :

- MemStore: a memory cache
- DbStore: local disk/db store
- LocalStore: a combination (sequence of) memStore and dbStore
- NetStore: cloud storage abstraction layer
- DPA: local requests for swarm storage and retrieval
*/
type ChunkStore interface {
	Put(*Chunk) // effectively there is no error even if there is an error
	Get(Key) (*Chunk, error)
	Close()
}

/*
Chunker is the interface to a component that is responsible for disassembling and assembling larger data and indended to be the dependency of a DPA storage system with fixed maximum chunksize.

It relies on the underlying chunking model.

When calling Split, the caller provides a channel (chan *Chunk) on which it receives chunks to store. The DPA delegates to storage layers (implementing ChunkStore interface).

Split returns an error channel, which the caller can monitor.
After getting notified that all the data has been split (the error channel is closed), the caller can safely read or save the root key. Optionally it times out if not all chunks get stored or not the entire stream of data has been processed. By inspecting the errc channel the caller can check if any explicit errors (typically IO read/write failures) occurred during splitting.

When calling Join with a root key, the caller gets returned a seekable lazy reader. The caller again provides a channel on which the caller receives placeholder chunks with missing data. The DPA is supposed to forward this to the chunk stores and notify the chunker if the data has been delivered (i.e. retrieved from memory cache, disk-persisted db or cloud based swarm delivery). As the seekable reader is used, the chunker then puts these together the relevant parts on demand.
*/
type Splitter interface {
	/*
	   When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	   New chunks to store are coming to caller via the chunk storage channel, which the caller provides.
	   wg is a Waitgroup (can be nil) that can be used to block until the local storage finishes
	   The caller gets returned an error channel, if an error is encountered during splitting, it is fed to errC error channel.
	   A closed error signals process completion at which point the key can be considered final if there were no errors.
	*/
	Split(io.Reader, int64, chan *Chunk, *sync.WaitGroup, *sync.WaitGroup) (Key, error)
}

type Joiner interface {
	/*
	   Join reconstructs original content based on a root key.
	   When joining, the caller gets returned a Lazy SectionReader, which is
	   seekable and implements on-demand fetching of chunks as and where it is read.
	   New chunks to retrieve are coming to caller via the Chunk channel, which the caller provides.
	   If an error is encountered during joining, it appears as a reader error.
	   The SectionReader.
	   As a result, partial reads from a document are possible even if other parts
	   are corrupt or lost.
	   The chunks are not meant to be validated by the chunker when joining. This
	   is because it is left to the DPA to decide which sources are trusted.
	*/
	Join(key Key, chunkC chan *Chunk) LazySectionReader
}

type Chunker interface {
	Joiner
	Splitter
	// returns the key length
	// KeySize() int64
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
