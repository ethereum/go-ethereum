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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	opentracing "github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
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
)

type ChunkerParams struct {
	chunkSize int64
	hashSize  int64
}

type SplitterParams struct {
	ChunkerParams
	reader io.Reader
	putter Putter
	addr   Address
}

type TreeSplitterParams struct {
	SplitterParams
	size int64
}

type JoinerParams struct {
	ChunkerParams
	addr   Address
	getter Getter
	// TODO: there is a bug, so depth can only be 0 today, see: https://github.com/ethersphere/go-ethereum/issues/344
	depth int
	ctx   context.Context
}

type TreeChunker struct {
	ctx context.Context

	branches int64
	hashFunc SwarmHasher
	dataSize int64
	data     io.Reader
	// calculated
	addr        Address
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
func TreeJoin(ctx context.Context, addr Address, getter Getter, depth int) *LazyChunkReader {
	jp := &JoinerParams{
		ChunkerParams: ChunkerParams{
			chunkSize: ch.DefaultSize,
			hashSize:  int64(len(addr)),
		},
		addr:   addr,
		getter: getter,
		depth:  depth,
		ctx:    ctx,
	}

	return NewTreeJoiner(jp).Join(ctx)
}

/*
	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	New chunks to store are store using the putter which the caller provides.
*/
func TreeSplit(ctx context.Context, data io.Reader, size int64, putter Putter) (k Address, wait func(context.Context) error, err error) {
	tsp := &TreeSplitterParams{
		SplitterParams: SplitterParams{
			ChunkerParams: ChunkerParams{
				chunkSize: ch.DefaultSize,
				hashSize:  putter.RefSize(),
			},
			reader: data,
			putter: putter,
		},
		size: size,
	}
	return NewTreeSplitter(tsp).Split(ctx)
}

func NewTreeJoiner(params *JoinerParams) *TreeChunker {
	tc := &TreeChunker{}
	tc.hashSize = params.hashSize
	tc.branches = params.chunkSize / params.hashSize
	tc.addr = params.addr
	tc.getter = params.getter
	tc.depth = params.depth
	tc.chunkSize = params.chunkSize
	tc.workerCount = 0
	tc.jobC = make(chan *hashJob, 2*ChunkProcessors)
	tc.wg = &sync.WaitGroup{}
	tc.errC = make(chan error)
	tc.quitC = make(chan bool)

	tc.ctx = params.ctx

	return tc
}

func NewTreeSplitter(params *TreeSplitterParams) *TreeChunker {
	tc := &TreeChunker{}
	tc.data = params.reader
	tc.dataSize = params.size
	tc.hashSize = params.hashSize
	tc.branches = params.chunkSize / params.hashSize
	tc.addr = params.addr
	tc.chunkSize = params.chunkSize
	tc.putter = params.putter
	tc.workerCount = 0
	tc.jobC = make(chan *hashJob, 2*ChunkProcessors)
	tc.wg = &sync.WaitGroup{}
	tc.errC = make(chan error)
	tc.quitC = make(chan bool)

	return tc
}

type hashJob struct {
	key      Address
	chunk    []byte
	size     int64
	parentWg *sync.WaitGroup
}

func (tc *TreeChunker) incrementWorkerCount() {
	tc.workerLock.Lock()
	defer tc.workerLock.Unlock()
	tc.workerCount += 1
}

func (tc *TreeChunker) getWorkerCount() int64 {
	tc.workerLock.RLock()
	defer tc.workerLock.RUnlock()
	return tc.workerCount
}

func (tc *TreeChunker) decrementWorkerCount() {
	tc.workerLock.Lock()
	defer tc.workerLock.Unlock()
	tc.workerCount -= 1
}

func (tc *TreeChunker) Split(ctx context.Context) (k Address, wait func(context.Context) error, err error) {
	if tc.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	tc.runWorker(ctx)

	depth := 0
	treeSize := tc.chunkSize

	// takes lowest depth such that chunksize*HashCount^(depth+1) > size
	// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.
	for ; treeSize < tc.dataSize; treeSize *= tc.branches {
		depth++
	}

	key := make([]byte, tc.hashSize)
	// this waitgroup member is released after the root hash is calculated
	tc.wg.Add(1)
	//launch actual recursive function passing the waitgroups
	go tc.split(ctx, depth, treeSize/tc.branches, key, tc.dataSize, tc.wg)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {
		// waiting for all threads to finish
		tc.wg.Wait()
		close(tc.errC)
	}()

	defer close(tc.quitC)
	defer tc.putter.Close()
	select {
	case err := <-tc.errC:
		if err != nil {
			return nil, nil, err
		}
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	return key, tc.putter.Wait, nil
}

func (tc *TreeChunker) split(ctx context.Context, depth int, treeSize int64, addr Address, size int64, parentWg *sync.WaitGroup) {

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
			n, err := tc.data.Read(chunkData[8+readBytes:])
			readBytes += int64(n)
			if err != nil && !(err == io.EOF && readBytes == size) {
				tc.errC <- err
				return
			}
		}
		select {
		case tc.jobC <- &hashJob{addr, chunkData, size, parentWg}:
		case <-tc.quitC:
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
		subTreeAddress := chunk[8+i*tc.hashSize : 8+(i+1)*tc.hashSize]

		childrenWg.Add(1)
		tc.split(ctx, depth-1, treeSize/tc.branches, subTreeAddress, secSize, childrenWg)

		i++
		pos += treeSize
	}
	// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
	// parentWg.Add(1)
	// go func() {
	childrenWg.Wait()

	worker := tc.getWorkerCount()
	if int64(len(tc.jobC)) > worker && worker < ChunkProcessors {
		tc.runWorker(ctx)

	}
	select {
	case tc.jobC <- &hashJob{addr, chunk, size, parentWg}:
	case <-tc.quitC:
	}
}

func (tc *TreeChunker) runWorker(ctx context.Context) {
	tc.incrementWorkerCount()
	go func() {
		defer tc.decrementWorkerCount()
		for {
			select {

			case job, ok := <-tc.jobC:
				if !ok {
					return
				}

				h, err := tc.putter.Put(ctx, job.chunk)
				if err != nil {
					tc.errC <- err
					return
				}
				copy(job.key, h)
				job.parentWg.Done()
			case <-tc.quitC:
				return
			}
		}
	}()
}

func (tc *TreeChunker) Append() (Address, func(), error) {
	return nil, nil, errAppendOppNotSuported
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	ctx       context.Context
	addr      Address // root address
	chunkData ChunkData
	off       int64 // offset
	chunkSize int64 // inherit from chunker
	branches  int64 // inherit from chunker
	hashSize  int64 // inherit from chunker
	depth     int
	getter    Getter
}

func (tc *TreeChunker) Join(ctx context.Context) *LazyChunkReader {
	return &LazyChunkReader{
		addr:      tc.addr,
		chunkSize: tc.chunkSize,
		branches:  tc.branches,
		hashSize:  tc.hashSize,
		depth:     tc.depth,
		getter:    tc.getter,
		ctx:       tc.ctx,
	}
}

func (r *LazyChunkReader) Context() context.Context {
	return r.ctx
}

// Size is meant to be called on the LazySectionReader
func (r *LazyChunkReader) Size(ctx context.Context, quitC chan bool) (n int64, err error) {
	metrics.GetOrRegisterCounter("lazychunkreader.size", nil).Inc(1)

	var sp opentracing.Span
	var cctx context.Context
	cctx, sp = spancontext.StartSpan(
		ctx,
		"lcr.size")
	defer sp.Finish()

	log.Debug("lazychunkreader.size", "addr", r.addr)
	if r.chunkData == nil {

		startTime := time.Now()
		chunkData, err := r.getter.Get(cctx, Reference(r.addr))
		if err != nil {
			metrics.GetOrRegisterResettingTimer("lcr.getter.get.err", nil).UpdateSince(startTime)
			return 0, err
		}
		metrics.GetOrRegisterResettingTimer("lcr.getter.get", nil).UpdateSince(startTime)
		r.chunkData = chunkData
		s := r.chunkData.Size()
		log.Debug("lazychunkreader.size", "key", r.addr, "size", s)
		if s < 0 {
			return 0, errors.New("corrupt size")
		}
		return int64(s), nil
	}
	s := r.chunkData.Size()
	log.Debug("lazychunkreader.size", "key", r.addr, "size", s)

	return int64(s), nil
}

// read at can be called numerous times
// concurrent reads are allowed
// Size() needs to be called synchronously on the LazyChunkReader first
func (r *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	metrics.GetOrRegisterCounter("lazychunkreader.readat", nil).Inc(1)

	var sp opentracing.Span
	var cctx context.Context
	cctx, sp = spancontext.StartSpan(
		r.ctx,
		"lcr.read")
	defer sp.Finish()

	defer func() {
		sp.LogFields(
			olog.Int("off", int(off)),
			olog.Int("read", read))
	}()

	// this is correct, a swarm doc cannot be zero length, so no EOF is expected
	if len(b) == 0 {
		return 0, nil
	}
	quitC := make(chan bool)
	size, err := r.Size(cctx, quitC)
	if err != nil {
		log.Debug("lazychunkreader.readat.size", "size", size, "err", err)
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
	length := int64(len(b))
	for d := 0; d < r.depth; d++ {
		off *= r.chunkSize
		length *= r.chunkSize
	}
	wg.Add(1)
	go r.join(b, off, off+length, depth, treeSize/r.branches, r.chunkData, &wg, errC, quitC)
	go func() {
		wg.Wait()
		close(errC)
	}()

	err = <-errC
	if err != nil {
		log.Debug("lazychunkreader.readat.errc", "err", err)
		close(quitC)
		return 0, err
	}
	if off+int64(len(b)) >= size {
		log.Debug("lazychunkreader.readat.return at end", "size", size, "off", off)
		return int(size - off), io.EOF
	}
	log.Debug("lazychunkreader.readat.errc", "buff", len(b))
	return len(b), nil
}

func (r *LazyChunkReader) join(b []byte, off int64, eoff int64, depth int, treeSize int64, chunkData ChunkData, parentWg *sync.WaitGroup, errC chan error, quitC chan bool) {
	defer parentWg.Done()
	// find appropriate block level
	for chunkData.Size() < uint64(treeSize) && depth > r.depth {
		treeSize /= r.branches
		depth--
	}

	// leaf chunk found
	if depth == r.depth {
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
	currentBranches := int64(len(chunkData)-8) / r.hashSize
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
			childAddress := chunkData[8+j*r.hashSize : 8+(j+1)*r.hashSize]
			startTime := time.Now()
			chunkData, err := r.getter.Get(r.ctx, Reference(childAddress))
			if err != nil {
				metrics.GetOrRegisterResettingTimer("lcr.getter.get.err", nil).UpdateSince(startTime)
				log.Debug("lazychunkreader.join", "key", fmt.Sprintf("%x", childAddress), "err", err)
				select {
				case errC <- fmt.Errorf("chunk %v-%v not found; key: %s", off, off+treeSize, fmt.Sprintf("%x", childAddress)):
				case <-quitC:
				}
				return
			}
			metrics.GetOrRegisterResettingTimer("lcr.getter.get", nil).UpdateSince(startTime)
			if l := len(chunkData); l < 9 {
				select {
				case errC <- fmt.Errorf("chunk %v-%v incomplete; key: %s, data length %v", off, off+treeSize, fmt.Sprintf("%x", childAddress), l):
				case <-quitC:
				}
				return
			}
			if soff < off {
				soff = off
			}
			r.join(b[soff-off:seoff-off], soff-roff, seoff-roff, depth-1, treeSize/r.branches, chunkData, wg, errC, quitC)
		}(i)
	} //for
}

// Read keeps a cursor so cannot be called simulateously, see ReadAt
func (r *LazyChunkReader) Read(b []byte) (read int, err error) {
	log.Debug("lazychunkreader.read", "key", r.addr)
	metrics.GetOrRegisterCounter("lazychunkreader.read", nil).Inc(1)

	read, err = r.ReadAt(b, r.off)
	if err != nil && err != io.EOF {
		log.Debug("lazychunkreader.readat", "read", read, "err", err)
		metrics.GetOrRegisterCounter("lazychunkreader.read.err", nil).Inc(1)
	}

	metrics.GetOrRegisterCounter("lazychunkreader.read.bytes", nil).Inc(int64(read))

	r.off += int64(read)
	return read, err
}

// completely analogous to standard SectionReader implementation
var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

func (r *LazyChunkReader) Seek(offset int64, whence int) (int64, error) {
	log.Debug("lazychunkreader.seek", "key", r.addr, "offset", offset)
	switch whence {
	default:
		return 0, errWhence
	case 0:
		offset += 0
	case 1:
		offset += r.off
	case 2:
		if r.chunkData == nil { //seek from the end requires rootchunk for size. call Size first
			_, err := r.Size(context.TODO(), nil)
			if err != nil {
				return 0, fmt.Errorf("can't get size: %v", err)
			}
		}
		offset += int64(r.chunkData.Size())
	}

	if offset < 0 {
		return 0, errOffset
	}
	r.off = offset
	return offset, nil
}
