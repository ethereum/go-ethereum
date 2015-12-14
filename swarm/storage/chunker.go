package storage

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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

const (
	// defaultHash           = "SHA3"   // http://golang.org/pkg/hash/#Hash
	defaultHash           = "SHA256" // http://golang.org/pkg/hash/#Hash
	defaultBranches int64 = 128
	joinTimeout           = 120 // second
	splitTimeout          = 120 // second
	// hashSize     int64 = hasherfunc.New().Size() // hasher knows about its own length in bytes
	// chunksize    int64 = branches * hashSize     // chunk is defined as this
)

/*
Tree chunker is a concrete implementation of data chunking.
This chunker works in a simple way, it builds a tree out of the document so that each node either represents a chunk of real data or a chunk of data representing an branching non-leaf node of the tree. In particular each such non-leaf chunk will represent is a concatenation of the hash of its respective children. This scheme simultaneously guarantees data integrity as well as self addressing. Abstract nodes are transparent since their represented size component is strictly greater than their maximum data size, since they encode a subtree.

If all is well it is possible to implement this by simply composing readers so that no extra allocation or buffering is necessary for the data splitting and joining. This means that in principle there can be direct IO between : memory, file system, network socket (bzz peers storage request is read from the socket). In practice there may be need for several stages of internal buffering.
The hashing itself does use extra copies and allocation though, since it does need it.
*/

type ChunkerParams struct {
	Branches     int64
	Hash         string
	JoinTimeout  time.Duration
	SplitTimeout time.Duration
}

func NewChunkerParams() *ChunkerParams {
	return &ChunkerParams{
		Branches:     defaultBranches,
		Hash:         defaultHash,
		JoinTimeout:  joinTimeout,
		SplitTimeout: splitTimeout,
	}
}

type TreeChunker struct {
	branches     int64
	hashFunc     Hasher
	joinTimeout  time.Duration
	splitTimeout time.Duration
	// calculated
	hashSize  int64 // self.hashFunc.New().Size()
	chunkSize int64 // hashSize* branches
}

func NewTreeChunker(params *ChunkerParams) (self *TreeChunker) {
	self = &TreeChunker{}
	self.hashFunc = MakeHashFunc(params.Hash)
	self.branches = params.Branches
	self.joinTimeout = params.JoinTimeout * time.Second
	self.splitTimeout = params.SplitTimeout * time.Second
	self.hashSize = int64(self.hashFunc().Size())
	self.chunkSize = self.hashSize * self.branches
	return
}

func (self *TreeChunker) KeySize() int64 {
	return self.hashSize
}

// String() for pretty printing
func (self *Chunk) String() string {
	return fmt.Sprintf("Key: %v TreeSize: %v Chunksize: %v", self.Key.Log(), self.Size, len(self.SData))
}

// The treeChunkers own Hash hashes together
// - the size (of the subtree encoded in the Chunk)
// - the Chunk, ie. the contents read from the input reader
func (self *TreeChunker) Hash(input []byte) []byte {
	hasher := self.hashFunc()
	hasher.Write(input)
	return hasher.Sum(nil)
}

func (self *TreeChunker) Split(key Key, data SectionReader, chunkC chan *Chunk, swg *sync.WaitGroup) (errC chan error) {

	if swg != nil {
		swg.Add(1)
		defer swg.Done()
	}

	if self.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	if int64(len(key)) != self.hashSize {
		panic(fmt.Sprintf("root key buffer must be allocated byte slice of length %d", self.hashSize))
	}

	wg := &sync.WaitGroup{}
	errC = make(chan error)
	rerrC := make(chan error)
	timeout := time.After(self.splitTimeout)

	wg.Add(1)
	go func() {

		depth := 0
		treeSize := self.chunkSize
		size := data.Size()
		// takes lowest depth such that chunksize*HashCount^(depth+1) > size
		// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.

		for ; treeSize < size; treeSize *= self.branches {
			depth++
		}

		// glog.V(logger.Detail).Infof("[BZZ] split request received for data (%v bytes, depth: %v)", size, depth)

		//launch actual recursive function passing the workgroup
		self.split(depth, treeSize/self.branches, key, data, chunkC, rerrC, wg, swg)
	}()

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {
		wg.Wait()
		close(rerrC)

	}()

	// waiting for request to end with wg finishing, error, or timeout
	go func() {
		select {
		case err := <-rerrC:
			if err != nil {
				errC <- err
			} // otherwise splitting is complete
		case <-timeout:
			errC <- fmt.Errorf("split time out")
		}
		close(errC)
	}()

	return
}

func (self *TreeChunker) split(depth int, treeSize int64, key Key, data SectionReader, chunkC chan *Chunk, errc chan error, parentWg *sync.WaitGroup, swg *sync.WaitGroup) {

	defer parentWg.Done()

	size := data.Size()
	var newChunk *Chunk
	var hash Key
	// glog.V(logger.Detail).Infof("[BZZ] depth: %v, max subtree size: %v, data size: %v", depth, treeSize, size)

	for depth > 0 && size < treeSize {
		treeSize /= self.branches
		depth--
	}

	if depth == 0 {
		// leaf nodes -> content chunks
		chunkData := make([]byte, data.Size()+8)
		binary.LittleEndian.PutUint64(chunkData[0:8], uint64(size))
		data.ReadAt(chunkData[8:], 0)
		hash = self.Hash(chunkData)
		// glog.V(logger.Detail).Infof("[BZZ] content chunk: max subtree size: %v, data size: %v", treeSize, size)
		newChunk = &Chunk{
			Key:   hash,
			SData: chunkData,
			Size:  size,
		}
	} else {
		// intermediate chunk containing child nodes hashes
		branchCnt := int64((size + treeSize - 1) / treeSize)
		// glog.V(logger.Detail).Infof("[BZZ] intermediate node: setting branches: %v, depth: %v, max subtree size: %v, data size: %v", branches, depth, treeSize, size)

		var chunk []byte = make([]byte, branchCnt*self.hashSize+8)
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
			// take the section of the data encoded in the subTree
			subTreeData := NewChunkReader(data, pos, secSize)
			// the hash of that data
			subTreeKey := chunk[8+i*self.hashSize : 8+(i+1)*self.hashSize]

			childrenWg.Add(1)
			go self.split(depth-1, treeSize/self.branches, subTreeKey, subTreeData, chunkC, errc, childrenWg, swg)

			i++
			pos += treeSize
		}
		// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
		childrenWg.Wait()
		// now we got the hashes in the chunk, then hash the chunks
		hash = self.Hash(chunk)
		newChunk = &Chunk{
			Key:   hash,
			SData: chunk,
			Size:  size,
			wg:    swg,
		}

		if swg != nil {
			swg.Add(1)
		}
	}

	// send off new chunk to storage
	if chunkC != nil {
		chunkC <- newChunk
	}
	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)x
	copy(key, hash)

}

func (self *TreeChunker) Join(key Key, chunkC chan *Chunk) SectionReader {

	return &LazyChunkReader{
		key:     key,
		chunkC:  chunkC,
		quitC:   make(chan bool),
		errC:    make(chan error),
		chunker: self,
	}
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	key     Key          // root key
	chunkC  chan *Chunk  // chunk channel to send retrieve requests on
	size    int64        // size of the entire subtree
	off     int64        // offset
	quitC   chan bool    // channel to abort retrieval
	errC    chan error   // error channel to monitor retrieve errors
	chunker *TreeChunker // needs TreeChunker params TODO: should just take
	// the chunkSize, branches etc as params
}

func (self *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	self.errC = make(chan error)
	chunk := &Chunk{
		Key: self.key,
		C:   make(chan bool), // close channel to signal data delivery
	}
	self.chunkC <- chunk // submit retrieval request, someone should be listening on the other side (or we will time out globally)
	glog.V(logger.Detail).Infof("[BZZ] readAt: reading %v into %d bytes at offset %d.", chunk.Key.Log(), len(b), off)

	// waiting for the chunk retrieval
	select {
	case <-self.quitC:
		// this is how we control process leakage (quitC is closed once join is finished (after timeout))
		// glog.V(logger.Detail).Infof("[BZZ] quit")
		return
	case <-chunk.C: // bells are ringing, data have been delivered
		// glog.V(logger.Detail).Infof("[BZZ] chunk data received for %v", chunk.Key.Log())
	}
	if len(chunk.SData) == 0 {
		// glog.V(logger.Detail).Infof("[BZZ] No payload in %v", chunk.Key.Log())
		return 0, notFound
	}
	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
	self.size = chunk.Size
	if b == nil {
		// glog.V(logger.Detail).Infof("[BZZ] Size query for %v", chunk.Key.Log())
		return
	}
	want := int64(len(b))
	if off+want > self.size {
		want = self.size - off
	}
	var treeSize int64
	var depth int
	// calculate depth and max treeSize
	treeSize = self.chunker.chunkSize
	for ; treeSize < chunk.Size; treeSize *= self.chunker.branches {
		depth++
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go self.join(b, off, off+want, depth, treeSize/self.chunker.branches, chunk, &wg)
	go func() {
		wg.Wait()
		close(self.errC)
	}()
	select {
	case err = <-self.errC:
		// glog.V(logger.Detail).Infof("[BZZ] ReadAt received %v", err)
		read = len(b)
		if off+int64(read) == self.size {
			err = io.EOF
		}
		// glog.V(logger.Detail).Infof("[BZZ] ReadAt returning at %d: %v", read, err)
	case <-self.quitC:
		// glog.V(logger.Detail).Infof("[BZZ] ReadAt aborted at %d: %v", read, err)
	}
	return
}

func (self *LazyChunkReader) join(b []byte, off int64, eoff int64, depth int, treeSize int64, chunk *Chunk, parentWg *sync.WaitGroup) {
	defer parentWg.Done()

	// glog.V(logger.Detail).Infof("[BZZ] depth: %v, loff: %v, eoff: %v, chunk.Size: %v, treeSize: %v", depth, off, eoff, chunk.Size, treeSize)

	chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))

	// find appropriate block level
	for chunk.Size < treeSize && depth > 0 {
		treeSize /= self.chunker.branches
		depth--
	}

	if depth == 0 {
		// glog.V(logger.Detail).Infof("[BZZ] depth: %v, len(b): %v, off: %v, eoff: %v, chunk.Size: %v, treeSize: %v", depth, len(b), off, eoff, chunk.Size, treeSize)
		if int64(len(b)) != eoff-off {
			//fmt.Printf("len(b) = %v  off = %v  eoff = %v", len(b), off, eoff)
			panic("len(b) does not match")
		}

		copy(b, chunk.SData[8+off:8+eoff])
		return // simply give back the chunks reader for content chunks
	}

	// subtree index
	start := off / treeSize
	end := (eoff + treeSize - 1) / treeSize
	wg := sync.WaitGroup{}

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

		wg.Add(1)
		go func(j int64) {
			childKey := chunk.SData[8+j*self.chunker.hashSize : 8+(j+1)*self.chunker.hashSize]
			// glog.V(logger.Detail).Infof("[BZZ] subtree index: %v -> %v", j, childKey.Log())

			ch := &Chunk{
				Key: childKey,
				C:   make(chan bool), // close channel to signal data delivery
			}
			// glog.V(logger.Detail).Infof("[BZZ] chunk data sent for %v (key interval in chunk %v-%v)", ch.Key.Log(), j*self.chunker.hashSize, (j+1)*self.chunker.hashSize)
			self.chunkC <- ch // submit retrieval request, someone should be listening on the other side (or we will time out globally)

			// waiting for the chunk retrieval
			select {
			case <-self.quitC:
				// this is how we control process leakage (quitC is closed once join is finished (after timeout))
				return
			case <-ch.C: // bells are ringing, data have been delivered
				// glog.V(logger.Detail).Infof("[BZZ] chunk data received")
			}
			if soff < off {
				soff = off
			}
			if len(ch.SData) == 0 {
				self.errC <- fmt.Errorf("chunk %v-%v not found", off, off+treeSize)
				return
			}
			self.join(b[soff-off:seoff-off], soff-roff, seoff-roff, depth-1, treeSize/self.chunker.branches, ch, &wg)
		}(i)
	} //for
	wg.Wait()
}
