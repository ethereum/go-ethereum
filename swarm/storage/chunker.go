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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
)

/*
The distributed storage implemented in this package requires fix sized chunks of content.

Chunker is the interface to a component that is responsible for disassembling and assembling larger data.

TreeChunker implements a Chunker based on a tree structure defined as follows:

1 each node in the tree including the root and other branching nodes are stored as a chunk.

2 branching nodes encode data contents that includes the size of the dataslice covered by its entire subtree under the node as well as the hash keys of all its children :
data_{i} := size(subtree_{i}) || key_{j} || key_{j+1} .... || key_{j+n-1}

3 Leaf nodes encode an actual subslice of the input data.

4 if data size is not more than maximum chunksize, the data is stored in a single chunk
  key = hash(int64(size) + data)

5 if data size is more than chunksize*branches^l, but no more than chunksize*
  branches^(l+1), the data vector is split into slices of chunksize*
  branches^l length (except the last one).
  key = hash(int64(size) + key(slice0) + key(slice1) + ...)

 The underlying hash function is configurable
*/

/*
Tree chunker is a concrete implementation of data chunking.
This chunker works in a simple way, it builds a tree out of the document so that each node either represents a chunk of real data or a chunk of data representing an branching non-leaf node of the tree. In particular each such non-leaf chunk will represent is a concatenation of the hash of its respective children. This scheme simultaneously guarantees data integrity as well as self addressing. Abstract nodes are transparent since their represented size component is strictly greater than their maximum data size, since they encode a subtree.

If all is well it is possible to implement this by simply composing readers so that no extra allocation or buffering is necessary for the data splitting and joining. This means that in principle there can be direct IO between : memory, file system, network socket (bzz peers storage request is read from the socket). In practice there may be need for several stages of internal buffering.
The hashing itself does use extra copies and allocation though, since it does need it.
*/

var (
	errAppendOppNotSuported = errors.New("Append operation not supported")
	errOperationTimedOut    = errors.New("operation timed out")
)

//metrics variables
var (
	newChunkCounter = metrics.NewRegisteredCounter("storage.chunks.new", nil)
)

type TreeChunker struct {
	branches int64
	hashFunc SwarmHasher
	// calculated
	hashSize    int64        // self.hashFunc.New().Size()
	chunkSize   int64        // hashSize* branches
	workerCount int64        // the number of worker routines used
	workerLock  sync.RWMutex // lock for the worker count
}

func NewTreeChunker(params *ChunkerParams) (self *TreeChunker) {
	self = &TreeChunker{}
	self.hashFunc = MakeHashFunc(params.Hash)
	self.branches = params.Branches
	self.hashSize = int64(self.hashFunc().Size())
	self.chunkSize = self.hashSize * self.branches
	self.workerCount = 0

	return
}

// func (tc *TreeChunker) KeySize() int64 {
// 	return tc.hashSize
// }

// String is for pretty printing.
func (c *Chunk) String() string {
	return fmt.Sprintf("Key: %v TreeSize: %v Chunksize: %v", c.Key.Log(), c.Size, len(c.SData))
}

type hashJob struct {
	key      Key
	chunk    []byte
	size     int64
	parentWg *sync.WaitGroup
}

func (tc *TreeChunker) incrementWorkerCount() {
	tc.workerLock.Lock()
	defer tc.workerLock.Unlock()
	tc.workerCount++
}

func (tc *TreeChunker) getWorkerCount() int64 {
	tc.workerLock.RLock()
	defer tc.workerLock.RUnlock()
	return tc.workerCount
}

func (tc *TreeChunker) decrementWorkerCount() {
	tc.workerLock.Lock()
	defer tc.workerLock.Unlock()
	tc.workerCount--
}

func (tc *TreeChunker) Split(data io.Reader, size int64, chunkC chan *Chunk, swg, wwg *sync.WaitGroup) (Key, error) {
	if tc.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	jobC := make(chan *hashJob, 2*ChunkProcessors)
	wg := &sync.WaitGroup{}
	errC := make(chan error)
	quitC := make(chan bool)

	// wwg = workers waitgroup keeps track of hashworkers spawned by this split call
	if wwg != nil {
		wwg.Add(1)
	}

	tc.incrementWorkerCount()
	go tc.hashWorker(jobC, chunkC, errC, quitC, swg, wwg)

	depth := 0
	treeSize := tc.chunkSize

	// takes lowest depth such that chunksize*HashCount^(depth+1) > size
	// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.
	for ; treeSize < size; treeSize *= tc.branches {
		depth++
	}

	key := make([]byte, tc.hashFunc().Size())
	// this waitgroup member is released after the root hash is calculated
	wg.Add(1)
	//launch actual recursive function passing the waitgroups
	go tc.split(depth, treeSize/tc.branches, key, data, size, jobC, chunkC, errC, quitC, wg, swg, wwg)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {
		// waiting for all threads to finish
		wg.Wait()
		// if storage waitgroup is non-nil, we wait for storage to finish too
		if swg != nil {
			swg.Wait()
		}
		close(errC)
	}()

	defer close(quitC)
	select {
	case err := <-errC:
		if err != nil {
			return nil, err
		}
	case <-time.NewTimer(splitTimeout).C:
		return nil, errOperationTimedOut
	}

	return key, nil
}

func (tc *TreeChunker) split(depth int, treeSize int64, key Key, data io.Reader, size int64, jobC chan *hashJob, chunkC chan *Chunk, errC chan error, quitC chan bool, parentWg, swg, wwg *sync.WaitGroup) {

	//

	for depth > 0 && size < treeSize {
		treeSize /= tc.branches
		depth--
	}

	if depth == 0 {
		// leaf nodes -> content chunks
		chunkData := make([]byte, size+8)
		binary.LittleEndian.PutUint64(chunkData[0:8], uint64(size))
		var readBytes int64
		for readBytes < size {
			n, err := data.Read(chunkData[8+readBytes:])
			readBytes += int64(n)
			if err != nil && !(err == io.EOF && readBytes == size) {
				errC <- err
				return
			}
		}
		select {
		case jobC <- &hashJob{key, chunkData, size, parentWg}:
		case <-quitC:
		}
		return
	}
	// dept > 0
	// intermediate chunk containing child nodes hashes
	branchCnt := (size + treeSize - 1) / treeSize

	var chunk = make([]byte, branchCnt*tc.hashSize+8)
	var pos, i int64

	binary.LittleEndian.PutUint64(chunk[0:8], uint64(size))

	childrenWg := &sync.WaitGroup{}
	var secSize int64
	for i < branchCnt {
		// the last item can have shorter data
		if size-pos < treeSize {
			secSize = size - pos
		} else {
			secSize = treeSize
		}
		// the hash of that data
		subTreeKey := chunk[8+i*tc.hashSize : 8+(i+1)*tc.hashSize]

		childrenWg.Add(1)
		tc.split(depth-1, treeSize/tc.branches, subTreeKey, data, secSize, jobC, chunkC, errC, quitC, childrenWg, swg, wwg)

		i++
		pos += treeSize
	}
	// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
	// parentWg.Add(1)
	// go func() {
	childrenWg.Wait()

	worker := tc.getWorkerCount()
	if int64(len(jobC)) > worker && worker < ChunkProcessors {
		if wwg != nil {
			wwg.Add(1)
		}
		tc.incrementWorkerCount()
		go tc.hashWorker(jobC, chunkC, errC, quitC, swg, wwg)

	}
	select {
	case jobC <- &hashJob{key, chunk, size, parentWg}:
	case <-quitC:
	}
}

func (tc *TreeChunker) hashWorker(jobC chan *hashJob, chunkC chan *Chunk, errC chan error, quitC chan bool, swg, wwg *sync.WaitGroup) {
	defer tc.decrementWorkerCount()

	hasher := tc.hashFunc()
	if wwg != nil {
		defer wwg.Done()
	}
	for {
		select {

		case job, ok := <-jobC:
			if !ok {
				return
			}
			// now we got the hashes in the chunk, then hash the chunks
			tc.hashChunk(hasher, job, chunkC, swg)
		case <-quitC:
			return
		}
	}
}

// The treeChunkers own Hash hashes together
// - the size (of the subtree encoded in the Chunk)
// - the Chunk, ie. the contents read from the input reader
func (tc *TreeChunker) hashChunk(hasher SwarmHash, job *hashJob, chunkC chan *Chunk, swg *sync.WaitGroup) {
	hasher.ResetWithLength(job.chunk[:8]) // 8 bytes of length
	hasher.Write(job.chunk[8:])           // minus 8 []byte length
	h := hasher.Sum(nil)

	newChunk := &Chunk{
		Key:   h,
		SData: job.chunk,
		Size:  job.size,
		wg:    swg,
	}

	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)
	copy(job.key, h)
	// send off new chunk to storage
	if chunkC != nil {
		if swg != nil {
			swg.Add(1)
		}
	}
	job.parentWg.Done()

	if chunkC != nil {
		//NOTE: this increases the chunk count even if the local node already has this chunk;
		//on file upload the node will increase this counter even if the same file has already been uploaded
		//So it should be evaluated whether it is worth keeping this counter
		//and/or actually better track when the chunk is Put to the local database
		//(which may question the need for disambiguation when a completely new chunk has been created
		//and/or a chunk is being put to the local DB; for chunk tracking it may be worth distinguishing
		newChunkCounter.Inc(1)
		chunkC <- newChunk
	}
}

func (tc *TreeChunker) Append(key Key, data io.Reader, chunkC chan *Chunk, swg, wwg *sync.WaitGroup) (Key, error) {
	return nil, errAppendOppNotSuported
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	key       Key         // root key
	chunkC    chan *Chunk // chunk channel to send retrieve requests on
	chunk     *Chunk      // size of the entire subtree
	off       int64       // offset
	chunkSize int64       // inherit from chunker
	branches  int64       // inherit from chunker
	hashSize  int64       // inherit from chunker
}

// Join implements the Joiner interface.
func (tc *TreeChunker) Join(key Key, chunkC chan *Chunk) LazySectionReader {
	return &LazyChunkReader{
		key:       key,
		chunkC:    chunkC,
		chunkSize: tc.chunkSize,
		branches:  tc.branches,
		hashSize:  tc.hashSize,
	}
}

// Size is meant to be called on the LazySectionReader
func (r *LazyChunkReader) Size(quitC chan bool) (n int64, err error) {
	if r.chunk != nil {
		return r.chunk.Size, nil
	}
	chunk := retrieve(r.key, r.chunkC, quitC)
	if chunk == nil {
		select {
		case <-quitC:
			return 0, errors.New("aborted")
		default:
			return 0, fmt.Errorf("root chunk not found for %v", r.key.Hex())
		}
	}
	r.chunk = chunk
	return chunk.Size, nil
}

// ReadAt can be called numerous times and concurrent reads are allowed
// Size() needs to be called synchronously on the LazyChunkReader first
func (r *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	// this is correct, a swarm doc cannot be zero length, so no EOF is expected
	if len(b) == 0 {
		return 0, nil
	}
	quitC := make(chan bool)
	size, err := r.Size(quitC)
	if err != nil {
		return 0, err
	}

	errC := make(chan error)

	// }
	var treeSize int64
	var depth int
	// calculate depth and max treeSize
	treeSize = r.chunkSize
	for ; treeSize < size; treeSize *= r.branches {
		depth++
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go r.join(b, off, off+int64(len(b)), depth, treeSize/r.branches, r.chunk, &wg, errC, quitC)
	go func() {
		wg.Wait()
		close(errC)
	}()

	err = <-errC
	if err != nil {
		close(quitC)

		return 0, err
	}
	if off+int64(len(b)) >= size {
		return len(b), io.EOF
	}
	return len(b), nil
}

func (r *LazyChunkReader) join(b []byte, off int64, eoff int64, depth int, treeSize int64, chunk *Chunk, parentWg *sync.WaitGroup, errC chan error, quitC chan bool) {
	defer parentWg.Done()
	// return NewDPA(&LocalStore{})

	// chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	// find appropriate block level
	for chunk.Size < treeSize && depth > 0 {
		treeSize /= r.branches
		depth--
	}

	// leaf chunk found
	if depth == 0 {
		extra := 8 + eoff - int64(len(chunk.SData))
		if extra > 0 {
			eoff -= extra
		}
		copy(b, chunk.SData[8+off:8+eoff])
		return // simply give back the chunks reader for content chunks
	}

	// subtree
	start := off / treeSize
	end := (eoff + treeSize - 1) / treeSize

	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for i := start; i < end; i++ {
		soff := i * treeSize
		roff := soff
		seoff := soff + treeSize

		if soff < off {
			soff = off
		}
		if seoff > eoff {
			seoff = eoff
		}
		if depth > 1 {
			wg.Wait()
		}
		wg.Add(1)
		go func(j int64) {
			childKey := chunk.SData[8+j*r.hashSize : 8+(j+1)*r.hashSize]
			chunk := retrieve(childKey, r.chunkC, quitC)
			if chunk == nil {
				select {
				case errC <- fmt.Errorf("chunk %v-%v not found", off, off+treeSize):
				case <-quitC:
				}
				return
			}
			if soff < off {
				soff = off
			}
			r.join(b[soff-off:seoff-off], soff-roff, seoff-roff, depth-1, treeSize/r.branches, chunk, wg, errC, quitC)
		}(i)
	} //for
}

// the helper method submits chunks for a key to a oueue (DPA) and
// block until they time out or arrive
// abort if quitC is readable
func retrieve(key Key, chunkC chan *Chunk, quitC chan bool) *Chunk {
	chunk := &Chunk{
		Key: key,
		C:   make(chan bool), // close channel to signal data delivery
	}
	// submit chunk for retrieval
	select {
	case chunkC <- chunk: // submit retrieval request, someone should be listening on the other side (or we will time out globally)
	case <-quitC:
		return nil
	}
	// waiting for the chunk retrieval
	select { // chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	case <-quitC:
		// this is how we control process leakage (quitC is closed once join is finished (after timeout))
		return nil
	case <-chunk.C: // bells are ringing, data have been delivered
	}
	if len(chunk.SData) == 0 {
		return nil // chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	}
	return chunk
}

// Read keeps a cursor so cannot be called simulateously, see ReadAt
func (r *LazyChunkReader) Read(b []byte) (read int, err error) {
	read, err = r.ReadAt(b, r.off)

	r.off += int64(read)
	return
}

// completely analogous to standard SectionReader implementation
var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

func (r *LazyChunkReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errWhence
	case 0:
		offset += 0
	case 1:
		offset += r.off
	case 2:
		if r.chunk == nil { //seek from the end requires rootchunk for size. call Size first
			_, err := r.Size(nil)
			if err != nil {
				return 0, fmt.Errorf("can't get size: %v", err)
			}
		}
		offset += r.chunk.Size
	}

	if offset < 0 {
		return 0, errOffset
	}
	r.off = offset
	return offset, nil
}
