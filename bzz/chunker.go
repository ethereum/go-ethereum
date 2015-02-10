/*
The distributed storage implemented in this package requires fix sized chunks of content
Chunker is the interface to a component that is responsible for disassembling and assembling larger data.

TreeChunker implements a Chunker based on a tree structure defined as follows:

1 each node in the tree including the root and other branching nodes are stored as a chunk.

2 branching nodes encode data contents that includes the size of the dataslice covered by its entire subtree under the node as well as the hash keys of all its children :
data_{i} := size(subtree_{i}) || key_{j} || key_{j+1} .... || key_{j+n-1}

3 Leaf nodes encode an actual subslice of the input data.

4 if data size is not more than maximum chunksize, the data is stored in a single chunk
  key = sha256(int64(size) + data)

2 if data size is more than chunksize*Branches^l, but no more than chunksize*
  Branches^(l+1), the data vector is split into slices of chunksize*
  Branches^l length (except the last one).
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
	branches   int64       = 128
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
When calling Split, the caller provides a channel (chan *Chunk) on which it receives chunks to store. The DPA delegates to storage layers (implementing ChunkStore interface). NewChunkstore(DB) is a convenience wrapper with which all DBs (conforming to DB interface) can serve as ChunkStores. See chunkStore.go
After getting notified that all the data has been split (the error channel is closed), the caller can safely read or save the root key. Optionally it times out if not all chunks get stored or not the entire stream of data has been processed. By inspecting the errc channel the caller can check if any explicit errors (typically IO read/write failures) occured during splitting.

When calling Join with a root key, the caller gets returned a lazy reader. The caller again provides a channel and receives an error channel. The chunk channel is the one on which the caller receives placeholder chunks with missing data. The DPA is supposed to forward this to the chunk stores and notify the chunker if the data has been delivered (i.e. retrieved from memory cache, disk-persisted db or cloud based swarm delivery). The chunker then puts these together and notifies the DPA if data has been assembled by a closed error channel. Once the DPA finds the data has been joined, it is free to deliver it back to swarm in full (if the original request was via the bzz protocol) or save and serve if it it was a local client request.

*/
type Chunker interface {
	/*
	 	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	 	New chunks to store are coming to caller via the chunk storage channel, which the caller provides.
	 	The caller gets returned an error channel, if an error is encountered during splitting, it is fed to errC error channel.
	   A closed error signals process completion at which point the key can be considered final if there were no errors.
	*/
	Split(key Key, data SectionReader, chunkC chan *Chunk) chan error
	/*
		Join reconstructs original content based on a root key.
		When joining, the caller gets returned a LazySectionReader and an error channel.
		New chunks to retrieve are coming to caller via the Chunk channel, which the caller provides.
		If an error is encountered during joining, it is fed to errC error channel.
		A closed error signals process completion at which point the data can be considered final and fully reconstructed if there were no errors.
		The LazySectionReader provides on-demand fetching of content including chunks.
		Lifecycle of the reader can be modified with SetTimeout()
	*/
	Join(key Key, chunkC chan *Chunk) (LazySectionReader, chan error)

	// returns the key length
	KeySize() int64
}

/*
Tree chunker is a concrete implementation of data chunking.
This chunker works in a simple way, it builds a tree out of the document so that each node either represents a chunk of real data or a chunk of data representing an branching non-leaf node of the tree. In particular each such non-leaf chunk will represent is a concatenation of the hash of its respective children. This scheme simultaneously guarantees data integrity as well as self addressing. Abstract nodes are transparent since their represented size component is strictly greater than their maximum data size, since they encode a subtree.

If all is well it is possible to implement this by simply composing readers so that no extra allocation or buffering is necessary for the data splitting and joining. This means that in principle there can be direct IO between : memory, file system, network socket (bzz peers storage request is read from the socket ). In practice there may be need for several stages of internal buffering.
Unfortunately the hashing itself does use extra copies and allocation though since it does need it.
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
	// dpaLogger.Debugf("Chunker initialised: branches: %v, hashsize: %v, chunksize: %v, join timeout: %v , split timeout: %v", self.Branches, self.hashSize, self.chunkSize, self.JoinTimeout, self.SplitTimeout)

}

func (self *TreeChunker) KeySize() int64 {
	return self.hashSize
}

// String() for pretty printing
func (self *Chunk) String() string {
	var size int64
	var slice []byte
	var n int
	var err error
	return fmt.Sprintf("Key: [%x..] TreeSize: %v Chunksize: %v Data: %x\n", self.Key[:4], self.Size, size, self.Data[:n])
}

// The treeChunkers own Hash hashes together
// - the size (of the subtree encoded in the Chunk)
// - the Chunk, ie. the contents read from the input reader
func (self *TreeChunker) Hash(size int64, input []byte) []byte {
	hasher := self.HashFunc.New()
	binary.Write(hasher, binary.LittleEndian, size)
	hasher.Write(input)
	return hasher.Sum(nil)
}

func (self *TreeChunker) Split(key Key, data SectionReader, chunkC chan *Chunk) (errC chan error) {

	if self.chunkSize <= 0 {
		panic("chunker must be initialised")
	}

	if int64(len(key)) != self.hashSize {
		panic(fmt.Sprintf("root key buffer must be allocated byte slice of length %d", self.hashSize))
	}

	wg := &sync.WaitGroup{}
	errC = make(chan error)
	rerrC := make(chan error)
	timeout := time.After(self.SplitTimeout)

	wg.Add(1)
	go func() {

		depth := 0
		treeSize := self.chunkSize
		size := data.Size()
		// takes lowest depth such that chunksize*HashCount^(depth+1) > size
		// power series, will find the order of magnitude of the data size in base hashCount or numbers of levels of branching in the resulting tree.

		for ; treeSize < size; treeSize *= self.Branches {
			depth++
		}

		// dpaLogger.Debugf("split request received for data (%v bytes, depth: %v)", size, depth)

		//launch actual recursive function passing the workgroup
		self.split(depth, treeSize/self.Branches, key, data, chunkC, rerrC, wg)
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

func (self *TreeChunker) split(depth int, treeSize int64, key Key, data SectionReader, chunkC chan *Chunk, errc chan error, parentWg *sync.WaitGroup) {

	defer parentWg.Done()

	size := data.Size()
	var newChunk *Chunk
	var hash Key
	// dpaLogger.Debugf("depth: %v, max subtree size: %v, data size: %v", depth, treeSize, size)

	for depth > 0 && size < treeSize {
		treeSize /= self.Branches
		depth--
	}

	if depth == 0 {
		// leaf nodes -> content chunks
		chunkData := make([]byte, data.Size())
		data.ReadAt(chunkData, 0)
		hash = self.Hash(size, chunkData)
		// dpaLogger.Debugf("content chunk: max subtree size: %v, data size: %v", treeSize, size)
		newChunk = &Chunk{
			Key:  hash,
			Data: chunkData,
			Size: size,
		}
	} else {
		// intermediate chunk containing child nodes hashes
		branches := int64((size-1)/treeSize) + 1
		// dpaLogger.Debugf("intermediate node: setting branches: %v, depth: %v, max subtree size: %v, data size: %v", branches, depth, treeSize, size)

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
		chunkData := make([]byte, chunkReader.Size())
		chunkReader.ReadAt(chunkData, 0)

		hash = self.Hash(size, chunkData)
		newChunk = &Chunk{
			Key:  hash,
			Data: chunkData,
			Size: size,
		}
	}
	// send off new chunk to storage
	if chunkC != nil {
		chunkC <- newChunk
	}
	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)x
	copy(key, hash)

}

func (self *TreeChunker) Join(key Key, chunkC chan *Chunk) (data LazySectionReader, errC chan error) {
	// initialise return parameters
	errC = make(chan error)
	// timer to time out the operation (needed within so as to avoid process leakage)
	timeout := time.After(self.JoinTimeout)
	wg := &sync.WaitGroup{}
	// initialise internal error channel
	rerrC := make(chan error)
	quitC := make(chan bool)

	// launch lazy recursive call on root chunk
	notReadyReader := &NotReadyReader{}
	reader := &EmbeddedReader{
		LazySectionReader: &lazyReader{notReadyReader},
	}
	fun := self.join(0, 0, key, chunkC, rerrC, wg, quitC)
	data = reader

	// processing is triggered by reads on the LazySectionReader
	wg.Add(1)
	notReadyReader.r = reader
	notReadyReader.initF = func() {
		reader.lock.Lock()
		if fun != nil {
			reader.LazySectionReader = fun() // replace the reader while caller is waiting
			wg.Done()                        // just to have one in waitgroup until reader funcs are called
			fun = nil                        // to be only called once
		}
		reader.lock.Unlock()
	}

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
	}()

	return
}

func (self *TreeChunker) join(depth int, treeSize int64, key Key, chunkC chan *Chunk, errC chan error, wg *sync.WaitGroup, quitC chan bool) (readerF func() LazySectionReader) {
	wg.Add(1)
	readerF = func() (r LazySectionReader) {
		defer wg.Done()
		chunk := &Chunk{
			Key: key,
			C:   make(chan bool, 1), // close channel to signal data delivery
		}
		chunkC <- chunk // submit retrieval request, someone should be listening on the other side (or we will time out globally)

		// waiting for the chunk retrieval
		select {
		case <-quitC:
			// this is how we control process leakage (quitC is closed once join is finished (after timeout))
			return
		case <-chunk.C: // bells are ringing, data have been delivered
		}

		//		if data == nil {
		//			return
		//		}

		// calculate depth and max treeSize
		var depth int
		var treeSize int64 = self.hashSize
		for ; treeSize*self.Branches < chunk.Size; treeSize *= self.Branches {
			depth++
		}

		if depth == 0 {
			r = LazyReader(NewByteSliceReader(chunk.Data))
			return // simply give back the chunks reader for content chunks
		}

		// find appropriate block level
		for chunk.Size < treeSize {
			treeSize /= self.Branches
			depth--
		}
		// boooo
		// intermediate chunk, chunk containing hashes of child nodes
		var i int64
		var childKey Key
		var readerF func() (r LazySectionReader)
		var readerFs [](func() (r LazySectionReader))
		branches := int64((chunk.Size-1)/treeSize) + 1
		// dpaLogger.DebugDetailf("tree node - size %v, chunk size: %v, subtreeSize %v, branches %v", chunk.Size, chunk.Reader.Size(), treeSize, branches)

		// iterate through the chunk containing the keys of children
		// create lazy init functions that give back readers
		for i < branches {
			// create partial Chunk in order to send a retrieval request
			childKey = make([]byte, self.hashSize) // preallocate hashSize long slice for key
			// read the Hash of the subtree from the relevant section of the Chunk into the allocated byte slice in subtree.Key
			if _, err := chunk.Reader.ReadAt(childKey, i*self.hashSize); err != nil {
				// dpaLogger.DebugDetailf("Read error: %v", err)
				errC <- err
				break
			}
			// call lazy reader function recursively on the subtree
			readerF = self.join(depth-1, treeSize/self.Branches, childKey, chunkC, errC, wg, quitC)
			readerFs = append(readerFs, readerF)
			i++
		}
		// new reader created on demand:
		r = &LazyChunkReader{
			readerFs: readerFs,
			sections: make([]LazySectionReader, branches),
			size:     chunk.Size,
			treeSize: treeSize,
		}
		return
	}
	return
}
