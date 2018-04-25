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

	"github.com/ethereum/go-ethereum/log"
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

const (
	DefaultChunkSize int64 = 4096
)

type ChunkerParams struct {
	chunkSize int64
	hashSize  int64
}

type SplitterParams struct {
	ChunkerParams
	reader io.Reader
	putter Putter
	key    Key
}

type TreeSplitterParams struct {
	SplitterParams
	size int64
}

type JoinerParams struct {
	ChunkerParams
	key    Key
	getter Getter
	// TODO: there is a bug, so depth can only be 0 today, see: https://github.com/ethersphere/go-ethereum/issues/344
	depth int
}

type TreeChunker struct {
	branches int64
	hashFunc SwarmHasher
	dataSize int64
	data     io.Reader
	// calculated
	key         Key
	depth       int
	hashSize    int64        // self.hashFunc.New().Size()
	chunkSize   int64        // hashSize* branches
	workerCount int64        // the number of worker routines used
	workerLock  sync.RWMutex // lock for the worker count
	jobC        chan *hashJob
	wg          *sync.WaitGroup
	putter      Putter
	getter      Getter
	errC        chan error
	quitC       chan bool
}

/*
	Join reconstructs original content based on a root key.
	When joining, the caller gets returned a Lazy SectionReader, which is
	seekable and implements on-demand fetching of chunks as and where it is read.
	New chunks to retrieve are coming from the getter, which the caller provides.
	If an error is encountered during joining, it appears as a reader error.
	The SectionReader.
	As a result, partial reads from a document are possible even if other parts
	are corrupt or lost.
	The chunks are not meant to be validated by the chunker when joining. This
	is because it is left to the DPA to decide which sources are trusted.
*/
func TreeJoin(key Key, getter Getter, depth int) *LazyChunkReader {
	return NewTreeJoiner(NewJoinerParams(key, getter, depth, DefaultChunkSize)).Join()
}

/*
	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	New chunks to store are store using the putter which the caller provides.
*/
func TreeSplit(data io.Reader, size int64, putter Putter) (k Key, wait func(), err error) {
	return NewTreeSplitter(NewTreeSplitterParams(data, putter, size, DefaultChunkSize)).Split()
}

func NewJoinerParams(key Key, getter Getter, depth int, chunkSize int64) *JoinerParams {
	hashSize := int64(len(key))
	return &JoinerParams{
		ChunkerParams: ChunkerParams{
			chunkSize: chunkSize,
			hashSize:  hashSize,
		},
		key:    key,
		getter: getter,
		depth:  depth,
	}
}

func NewTreeJoiner(params *JoinerParams) *TreeChunker {
	self := &TreeChunker{}
	self.hashSize = params.hashSize
	self.branches = params.chunkSize / self.hashSize
	self.key = params.key
	self.getter = params.getter
	self.depth = params.depth
	self.chunkSize = self.hashSize * self.branches
	self.workerCount = 0
	self.jobC = make(chan *hashJob, 2*ChunkProcessors)
	self.wg = &sync.WaitGroup{}
	self.errC = make(chan error)
	self.quitC = make(chan bool)

	return self
}

func NewTreeSplitterParams(reader io.Reader, putter Putter, size int64, branches int64) *TreeSplitterParams {
	hashSize := putter.RefSize()
	return &TreeSplitterParams{
		SplitterParams: SplitterParams{
			ChunkerParams: ChunkerParams{
				chunkSize: chunkSize,
				hashSize:  hashSize,
			},
			reader: reader,
			putter: putter,
		},
		size: size,
	}
}

func NewTreeSplitter(params *TreeSplitterParams) *TreeChunker {
	self := &TreeChunker{}
	self.data = params.reader
	self.dataSize = params.size
	self.hashSize = params.hashSize
	self.branches = params.chunkSize / self.hashSize
	self.key = params.key
	self.chunkSize = self.hashSize * self.branches
	self.putter = params.putter
	self.workerCount = 0
	self.jobC = make(chan *hashJob, 2*ChunkProcessors)
	self.wg = &sync.WaitGroup{}
	self.errC = make(chan error)
	self.quitC = make(chan bool)

	return self
}

// String() for pretty printing
func (self *Chunk) String() string {
	return fmt.Sprintf("Key: %v TreeSize: %v Chunksize: %v", self.Key.Log(), self.Size, len(self.SData))
}

type hashJob struct {
	key      Key
	chunk    []byte
	size     int64
	parentWg *sync.WaitGroup
}

func (self *TreeChunker) incrementWorkerCount() {
	self.workerLock.Lock()
	defer self.workerLock.Unlock()
	self.workerCount += 1
}

func (self *TreeChunker) getWorkerCount() int64 {
	self.workerLock.RLock()
	defer self.workerLock.RUnlock()
	return self.workerCount
}

func (self *TreeChunker) decrementWorkerCount() {
	self.workerLock.Lock()
	defer self.workerLock.Unlock()
	self.workerCount -= 1
}

func (self *TreeChunker) Split() (k Key, wait func(), err error) {
	if self.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	self.runWorker()

	depth := 0
	treeSize := self.chunkSize

	// takes lowest depth such that chunksize*HashCount^(depth+1) > size
	// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.
	for ; treeSize < self.dataSize; treeSize *= self.branches {
		depth++
	}

	key := make([]byte, self.hashSize)
	// this waitgroup member is released after the root hash is calculated
	self.wg.Add(1)
	//launch actual recursive function passing the waitgroups
	go self.split(depth, treeSize/self.branches, key, self.dataSize, self.wg)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {
		// waiting for all threads to finish
		self.wg.Wait()
		close(self.errC)
	}()

	defer close(self.quitC)
	defer self.putter.Close()
	select {
	case err := <-self.errC:
		if err != nil {
			return nil, nil, err
		}
	case <-time.NewTimer(splitTimeout).C:
		return nil, nil, errOperationTimedOut
	}

	return key, self.putter.Wait, nil
}

func (self *TreeChunker) split(depth int, treeSize int64, key Key, size int64, parentWg *sync.WaitGroup) {

	//

	for depth > 0 && size < treeSize {
		treeSize /= self.branches
		depth--
	}

	if depth == 0 {
		// leaf nodes -> content chunks
		chunkData := make([]byte, size+8)
		binary.LittleEndian.PutUint64(chunkData[0:8], uint64(size))
		var readBytes int64
		for readBytes < size {
			n, err := self.data.Read(chunkData[8+readBytes:])
			readBytes += int64(n)
			if err != nil && !(err == io.EOF && readBytes == size) {
				self.errC <- err
				return
			}
		}
		select {
		case self.jobC <- &hashJob{key, chunkData, size, parentWg}:
		case <-self.quitC:
		}
		return
	}
	// dept > 0
	// intermediate chunk containing child nodes hashes
	branchCnt := (size + treeSize - 1) / treeSize

	var chunk = make([]byte, branchCnt*self.hashSize+8)
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
		subTreeKey := chunk[8+i*self.hashSize : 8+(i+1)*self.hashSize]

		childrenWg.Add(1)
		self.split(depth-1, treeSize/self.branches, subTreeKey, secSize, childrenWg)

		i++
		pos += treeSize
	}
	// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
	// parentWg.Add(1)
	// go func() {
	childrenWg.Wait()

	worker := self.getWorkerCount()
	if int64(len(self.jobC)) > worker && worker < ChunkProcessors {
		self.runWorker()

	}
	select {
	case self.jobC <- &hashJob{key, chunk, size, parentWg}:
	case <-self.quitC:
	}
}

func (self *TreeChunker) runWorker() {
	self.incrementWorkerCount()
	go func() {
		defer self.decrementWorkerCount()
		for {
			select {

			case job, ok := <-self.jobC:
				if !ok {
					return
				}

				h, err := self.putter.Put(job.chunk)
				if err != nil {
					self.errC <- err
					return
				}
				copy(job.key, h)
				job.parentWg.Done()
			case <-self.quitC:
				return
			}
		}
	}()
}

func (self *TreeChunker) Append() (Key, func(), error) {
	return nil, nil, errAppendOppNotSuported
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	key       Key // root key
	chunkData ChunkData
	off       int64 // offset
	chunkSize int64 // inherit from chunker
	branches  int64 // inherit from chunker
	hashSize  int64 // inherit from chunker
	depth     int
	getter    Getter
}

func (self *TreeChunker) Join() *LazyChunkReader {
	return &LazyChunkReader{
		key:       self.key,
		chunkSize: self.chunkSize,
		branches:  self.branches,
		hashSize:  self.hashSize,
		depth:     self.depth,
		getter:    self.getter,
	}
}

// Size is meant to be called on the LazySectionReader
func (self *LazyChunkReader) Size(quitC chan bool) (n int64, err error) {
	log.Debug("lazychunkreader.size", "key", self.key)
	if self.chunkData == nil {
		chunkData, err := self.getter.Get(Reference(self.key))
		if err != nil {
			return 0, err
		}
		if chunkData == nil {
			select {
			case <-quitC:
				return 0, errors.New("aborted")
			default:
				return 0, fmt.Errorf("root chunk not found for %v", self.key.Hex())
			}
		}
		self.chunkData = chunkData
	}
	return self.chunkData.Size(), nil
}

// read at can be called numerous times
// concurrent reads are allowed
// Size() needs to be called synchronously on the LazyChunkReader first
func (self *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	// this is correct, a swarm doc cannot be zero length, so no EOF is expected
	if len(b) == 0 {
		return 0, nil
	}
	quitC := make(chan bool)
	size, err := self.Size(quitC)
	if err != nil {
		log.Error("lazychunkreader.readat.size", "size", size, "err", err)
		return 0, err
	}

	errC := make(chan error)

	// }
	var treeSize int64
	var depth int
	// calculate depth and max treeSize
	treeSize = self.chunkSize
	for ; treeSize < size; treeSize *= self.branches {
		depth++
	}
	wg := sync.WaitGroup{}
	length := int64(len(b))
	for d := 0; d < self.depth; d++ {
		off *= self.chunkSize
		length *= self.chunkSize
	}
	wg.Add(1)
	go self.join(b, off, off+length, depth, treeSize/self.branches, self.chunkData, &wg, errC, quitC)
	go func() {
		wg.Wait()
		close(errC)
	}()

	err = <-errC
	if err != nil {
		log.Error("lazychunkreader.readat.errc", "err", err)
		close(quitC)
		return 0, err
	}
	if off+int64(len(b)) >= size {
		return int(size - off), io.EOF
	}
	return len(b), nil
}

func (self *LazyChunkReader) join(b []byte, off int64, eoff int64, depth int, treeSize int64, chunkData ChunkData, parentWg *sync.WaitGroup, errC chan error, quitC chan bool) {
	defer parentWg.Done()
	// find appropriate block level
	for chunkData.Size() < treeSize && depth > self.depth {
		treeSize /= self.branches
		depth--
	}

	// leaf chunk found
	if depth == self.depth {
		extra := 8 + eoff - int64(len(chunkData))
		if extra > 0 {
			eoff -= extra
		}
		copy(b, chunkData[8+off:8+eoff])
		return // simply give back the chunks reader for content chunks
	}

	// subtree
	start := off / treeSize
	end := (eoff + treeSize - 1) / treeSize

	// last non-leaf chunk can be shorter than default chunk size, let's not read it further then its end
	currentBranches := int64(len(chunkData)-8) / self.hashSize
	if end > currentBranches {
		end = currentBranches
	}

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
			childKey := chunkData[8+j*self.hashSize : 8+(j+1)*self.hashSize]
			chunkData, err := self.getter.Get(Reference(childKey))
			if err != nil {
				log.Error("lazychunkreader.join", "key", fmt.Sprintf("%x", childKey), "err", err)
				select {
				case errC <- fmt.Errorf("chunk %v-%v not found; key: %s", off, off+treeSize, fmt.Sprintf("%x", childKey)):
				case <-quitC:
				}
				return
			}
			if soff < off {
				soff = off
			}
			self.join(b[soff-off:seoff-off], soff-roff, seoff-roff, depth-1, treeSize/self.branches, chunkData, wg, errC, quitC)
		}(i)
	} //for
}

// Read keeps a cursor so cannot be called simulateously, see ReadAt
func (self *LazyChunkReader) Read(b []byte) (read int, err error) {
	log.Debug("lazychunkreader.read", "key", self.key)
	read, err = self.ReadAt(b, self.off)
	if err != nil && err != io.EOF {
		log.Error("lazychunkreader.readat", "read", read, "err", err)
	}

	self.off += int64(read)
	return
}

// completely analogous to standard SectionReader implementation
var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

func (s *LazyChunkReader) Seek(offset int64, whence int) (int64, error) {
	log.Debug("lazychunkreader.seek", "key", s.key, "offset", offset)
	switch whence {
	default:
		return 0, errWhence
	case 0:
		offset += 0
	case 1:
		offset += s.off
	case 2:
		if s.chunkData == nil { //seek from the end requires rootchunk for size. call Size first
			_, err := s.Size(nil)
			if err != nil {
				return 0, fmt.Errorf("can't get size: %v", err)
			}
		}
		offset += s.chunkData.Size()
	}

	if offset < 0 {
		return 0, errOffset
	}
	s.off = offset
	return offset, nil
}
