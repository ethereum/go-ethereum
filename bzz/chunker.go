/*
The distributed storage implemented in this package requires fix sized chunks of content
Chunker is the interface to a component that is responsible for disassembling and assembling larger data.

TreeChunker implements a Chunker based on a tree structure defined as follows:

1 if size is no more than chunksize, it is stored in a single chunk
  key = sha256(int64(size) + data)

2 if size is more than chunksize*HashCount^l, but no more than chunksize*
  HashCount^(l+1), the data vector is split into slices of chunksize*
  HashCount^l length (except the last one).
  key = sha256(int64(size) + key(slice0) + key(slice1) + ...)
*/

package bzz

import (
	"crypto"
	"encoding/binary"
	"fmt"
	"sync"
	"time"
)

const (
	hasherfunc crypto.Hash = crypto.SHA256 // http://golang.org/pkg/hash/#Hash
	branches   int64       = 4
)

var (
	// hashSize     int64 = hasherfunc.New().Size() // hasher knows about its own length in bytes
	// chunksize    int64 = branches * hashSize     // chunk is defined as this
	joinTimeout  = 120 * time.Second
	splitTimeout = 120 * time.Second
)

type Key []byte

/*
Chunker is the interface to a component that is responsible for disassembling and assembling larger data and indended to be the dependency of a DPA storage system with fixed maximum chunksize.
It relies on the underlying chunking model.
When calling Split, the caller gets returned a channel (chan *Chunk) on which it receives chunks to store. The DPA delegates to storage layers (implementing ChunkStore interface). NewChunkstore(DB) is a convenience wrapper with which all DBs (conforming to DB interface) can serve as ChunkStores. See chunkStore.go
After getting notified that all the data has been split (the error channel and chunk channel are closed), the caller can safely read or save the root key. Optionally it times out if not all chunks get stored or not the entire stream of data has been processed. By inspecting the errc channel the caller can check if any explicit errors (typically IO read/write failures) occured during splitting.

When calling Join with a root key, the data can be nil ponter in which case it will be initialized as a byte slice based reader corresponding the size of the entire subtree encoded in the chunk. The caller gets returned a channel and an error channel. The chunk channel is the one on which the caller receives placeholder chunks with missing data. The DPA is supposed to forward this to the chunk stores and notify the chunker if the data has been delivered (i.e. retrieved from memory cache, disk-persisted db or cloud based swarm delivery. The chunker then puts these together and notifies the DPA if data has been assembled by a closed error channel. Once the DPA finds the data has been joined, it is free to deliver it back to swarm in full (if the original request was via the bzz protocol) or save and serve if it it was a local client request.

*/
type Chunker interface {
	/*
	 	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	 	New chunks to store are coming to caller via the chunk channel (first return parameter)
	 	If an error is encountered during splitting, it is fed to errC error channel (second return parameter)
	   A closed error and chunk channel signal process completion at which point the key can be considered final if there were no errors.
	*/
	Split(key Key, data SectionReader) (chan *Chunk, chan error)
	/*
		Join reconstructs original content based on a root key
		When joining, data is given as a SectionWriter preallocation for which is taken care of by the caller.
		If the size of the SectionWriter is found 0 Chunker will resize it once the entire data size is known from the root chunk. If resize is not supported by the writer, an error is given.
		Any other size value is checked and if it does not fit the actual datasize, an error will be reported.
		New chunks to retrieve are coming to caller via the Chunk channel (first return parameter)
		If an error is encountered during joining, it is fed to errC error channel (second return parameter)
		A closed error and chunk channel signal process completion at which point the data can be considered final if there were no errors.
	*/
	Join(key Key, data SectionWriter) (chan *Chunk, chan error)
}

/*
Tree chunker is a concrete implementation of data chunking.
This chunker works in a simple way, it builds a tree out of the document so that each node either represents a chunk of real data or a chunk of data representing an branching non-leaf node of the tree. In particular each such non-leaf chunk will represent is a concatenation of the hash of its respective children. This scheme simultaneously guarantees data integrity as well as self addressing. Abstract nodes are transparent since their represented size component is strictly greater than their maximum data size, since they encode a subtree.

If all is well it is possible to implement this by simply composing readers and writers so that no extra allocation or buffering is necessary for the data splitting. This means that in principle there can be direct IO between : memory, file system, network socket (bzz peers storage request is read from the socket ). In practice there may be need for several stages of internal buffering.
Unfortunately the hashing itself does use extra copies and allocatetion though since it does need it.
*/
type TreeChunker struct {
	Branches     int64
	HashFunc     crypto.Hash
	JoinTimeout  time.Duration
	SplitTimeout time.Duration
	// calculated
	hashSize  int64 // self.HashFunc.New().Size()
	chunkSize int64 // hashSize* Branches
}

func (self *TreeChunker) Init() {
	if self.HashFunc == 0 {
		self.HashFunc = hasherfunc
	}
	if self.Branches == 0 {
		self.Branches = branches
	}
	if self.JoinTimeout == 0 {
		self.JoinTimeout = joinTimeout
	}
	if self.SplitTimeout == 0 {
		self.SplitTimeout = splitTimeout
	}
	self.hashSize = int64(self.HashFunc.New().Size())
	self.chunkSize = self.hashSize * self.Branches
	dpaLogger.Debugf("Chunker initialised: branches: %v, hashsize: %v, chunksize: %v, join timeout: %v , split timeout: %v", self.Branches, self.hashSize, self.chunkSize, self.JoinTimeout, self.SplitTimeout)

}

type Chunk struct {
	Data SectionReader // nil if request, to be supplied by dpa
	Size int64         // size of the data covered by the subtree encoded in this chunk
	// not the size of data, which is Data.Size() see SectionReader
	// 0 if request, to be supplied by dpa
	Key Key       // always
	C   chan bool // to signal data delivery by the dpa
	wg  sync.WaitGroup
}

func (self *Chunk) String() string {
	var size int64
	if self.Data != nil {
		size = self.Data.Size()
	}
	return fmt.Sprintf("Key: [%x..] TreeSize: %v Chunksize: %v", self.Key[:4], self.Size, size)
}

// The treeChunkers own Hash hashes together
// - the size (of the subtree encoded in the Chunk)
// - the Chunk, ie. the contents read from the input reader
func (self *TreeChunker) Hash(size int64, input SectionReader) []byte {
	hasher := self.HashFunc.New()
	binary.Write(hasher, binary.LittleEndian, size)
	input.WriteTo(hasher) // SectionReader implements io.WriterTo
	return hasher.Sum(nil)
}

func (self *TreeChunker) Split(key Key, data SectionReader) (chunkC chan *Chunk, errC chan error) {
	wg := &sync.WaitGroup{}
	chunkC = make(chan *Chunk)
	errC = make(chan error)
	rerrC := make(chan error)
	timeout := time.After(splitTimeout)
	if key == nil {
		dpaLogger.Debugf("please allocate byte slice for root key")
		return
	}
	wg.Add(1)
	dpaLogger.Debugf("add one")

	go func() {

		depth := 0
		treeSize := self.chunkSize
		size := data.Size()
		// takes lowest depth such that chunksize*HashCount^(depth+1) > size
		// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.

		for ; treeSize < size; treeSize *= self.Branches {
			depth++
		}

		dpaLogger.Debugf("split request received for data (%v bytes, depth: %v)", size, depth)

		//launch actual recursive function passing the workgroup
		self.split(depth, treeSize/self.Branches, key, data, chunkC, rerrC, wg)
	}()

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		dpaLogger.Debugf("waiting for splitter to finish")
		wg.Wait()
		dpaLogger.Debugf("splitter finished. closing rerrC")
		close(rerrC)

	}()

	// waiting for request to end with wg finishing, error, or timeout
	go func() {
		dpaLogger.Debugf("waiting for rerrC to close")

		select {
		case err := <-rerrC:
			dpaLogger.Debugf("action on rerrC")

			if err != nil {
				dpaLogger.Debugf("error on rerrC")

				errC <- err
			} // otherwise splitting is complete
		case <-timeout:
			errC <- fmt.Errorf("split time out")
		}
		close(chunkC)
		close(errC)
	}()

	return
}

func (self *TreeChunker) split(depth int, treeSize int64, key Key, data SectionReader, chunkC chan *Chunk, errc chan error, parentWg *sync.WaitGroup) {

	defer parentWg.Done()

	size := data.Size()
	var newChunk *Chunk
	var hash Key
	dpaLogger.Debugf("depth: %v, max subtree size: %v, data size: %v", depth, treeSize, size)

	switch {
	case depth == 0:
		if size > self.chunkSize {
			panic("ouch")
		}
		// leaf nodes -> content chunks
		hash = self.Hash(size, data)
		dpaLogger.Debugf("content chunk: max subtree size: %v, data size: %v", treeSize, size)
		newChunk = &Chunk{
			Key:  hash,
			Data: data,
			Size: size,
		}
	case size < treeSize:
		// last item on this level (== size % self.Branches ^ (depth + 1) )
		self.split(depth-1, treeSize/self.Branches, key, data, chunkC, errc, parentWg)
		return
	default:
		// intermediate chunk containing child nodes hashes
		branches := int64((size-1)/treeSize) + 1
		dpaLogger.Debugf("intermediate node: setting branches: %v, depth: %v, max subtree size: %v, data size: %v", branches, depth, treeSize, size)

		var chunk []byte = make([]byte, branches*self.hashSize)
		var pos, i int64

		childrenWg := &sync.WaitGroup{}
		var secSize int64
		for i < branches {
			// the last item can have shorter data
			if size-pos < treeSize {
				secSize = size - pos
			} else {
				secSize = treeSize
			}
			// take the section of the data corresponding encoded in the subTree
			subTreeData := NewChunkReader(data, pos, secSize)
			// the hash of that data
			subTreeKey := chunk[i*self.hashSize : (i+1)*self.hashSize]

			childrenWg.Add(1)
			go self.split(depth-1, treeSize/self.Branches, subTreeKey, subTreeData, chunkC, errc, childrenWg)

			i++
			pos += treeSize
		}
		// wait for all the children to complete calculating their hashes and copying them onto sections of the chunk
		childrenWg.Wait()
		// now we got the hashes in the chunk, then hash the chunk
		chunkReader := NewChunkReaderFromBytes(chunk) // bytes.Reader almost implements SectionReader
		hash = self.Hash(treeSize, chunkReader)
		newChunk = &Chunk{
			Key:  hash,
			Data: chunkReader,
			Size: treeSize,
		}
	}
	// send off new chunk to storage
	dpaLogger.Debugf("sending chunk on chunk channel")

	chunkC <- newChunk
	dpaLogger.Debugf("sent chunk on chunk channel")

	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)
	dpaLogger.Debugf("copying parent key ")

	copy(key, hash)
	dpaLogger.Debugf("copied parent key ")

}

func (self *TreeChunker) Join(key Key, data SectionWriter) (chunkC chan *Chunk, errC chan error) {
	// initialise return parameters
	errC = make(chan error)
	chunkC = make(chan *Chunk)
	// timer to time out the operation (needed within so as to avoid process leakage)
	timeout := time.After(joinTimeout)
	wg := &sync.WaitGroup{}
	// initialise internal error channel
	rerrC := make(chan error)
	quitC := make(chan bool)

	wg.Add(1)
	go func() {
		// create the 'chunk' for root chunk of the data tree
		chunk := &Chunk{
			Key: key,
			C:   make(chan bool, 1),
		}
		// request data
		dpaLogger.Debugf("request root chunk for key %x", key[:4])
		chunkC <- chunk
		// wait for reponse, if no root, we cannot go on
		select {
		case <-chunk.C: // bells ringing data delivered
			dpaLogger.Debugf("request root chunk data has come, size %v", chunk.Size)
		case <-timeout:
			err := fmt.Errorf("split time out waiting for root")
			rerrC <- err
			wg.Done()
			return
		}

		if data.Size() < chunk.Size {
			dpaLogger.Debugf("trying to resize writer to size %v for join data", chunk.Size)

			var err error
			if resizeable, ok := data.(Resizeable); ok {
				err = resizeable.Resize(chunk.Size)
			} else {
				err = fmt.Errorf("writer does not support resizing and has insufficient size %v (need %v)", data.Size(), chunk.Size)
			}
			if err != nil {
				dpaLogger.Debugf("%v", err)
				rerrC <- err
				wg.Done()
				return
			}
		}
		// calculate depth and max treeSize
		var depth int
		var treeSize int64 = self.chunkSize

		for ; treeSize < chunk.Size; treeSize *= self.Branches {
			depth++
		}
		// launch recursive call on root chunk
		self.join(depth, treeSize, chunk, data, chunkC, rerrC, wg, quitC)
	}()

	// waits for all the processes to finish and signals by closing internal rerrc
	go func() {
		wg.Wait()
		close(rerrC)
	}()

	go func() {
		select {
		case err := <-rerrC:
			if err != nil {
				errC <- err
			} // otherwise channel is closed, data joining complete
		case <-timeout:
			errC <- fmt.Errorf("join time out")
			close(quitC)
		}
		// this will indicate to the caller that processing is finished (with or without error)
		close(errC)
		close(chunkC)
	}()

	return
}

func (self *TreeChunker) join(depth int, treeSize int64, chunk *Chunk, data SectionWriter, chunkC chan *Chunk, errC chan error, wg *sync.WaitGroup, quitC chan bool) {

	defer wg.Done()

	select {
	case <-quitC:
	case <-chunk.C: // bells are ringing, data have been delivered
		dpaLogger.Debugf("received chunk data: %v", chunk)
		switch {
		case chunk.Size <= treeSize && depth == 0:
			dpaLogger.Debugf("reading into data")
			// we received a chunk for a leaf node representing actual content
			if _, err := data.ReadFrom(chunk.Data); err != nil {
				errC <- err
			}
			return
		case chunk.Size < treeSize:
			// this must be a last item on its level
			self.join(depth-1, treeSize/self.Branches, chunk, data, chunkC, errC, wg, quitC)
			return
		default:
			// intermediate chunk, chunk containing hashes of child nodes
			var pos, i int64
			for pos < chunk.Size {
				// create partial Chunk in order to send a retrieval request
				subtree := &Chunk{
					Key: make([]byte, self.hashSize), // preallocate hashSize long slice for key
					C:   make(chan bool, 1),          // close channel to signal data delivery
				}
				// read the Hash of the subtree from the relevant section of the Chunk into the allocated byte slice in subtree.Key
				chunk.Data.ReadAt(subtree.Key, i*self.hashSize)
				// call recursively on the subtree
				subTreeData := NewChunkWriter(data, pos, treeSize)
				wg.Add(1)
				go self.join(depth-1, treeSize/self.Branches, subtree, subTreeData, chunkC, errC, wg, quitC)
				// submit request
				chunkC <- subtree
				i++
				pos += subtree.Size
			}

		}
	}
}
