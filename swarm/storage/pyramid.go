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
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

/*
   The main idea of a pyramid chunker is to process the input data without knowing the entire size apriori.
   For this to be achieved, the chunker tree is built from the ground up until the data is exhausted.
   This opens up new aveneus such as easy append and other sort of modifications to the tree thereby avoiding
   duplication of data chunks.


   Below is an example of a two level chunks tree. The leaf chunks are called data chunks and all the above
   chunks are called tree chunks. The tree chunk above data chunks is level 0 and so on until it reaches
   the root tree chunk.



                                            T10                                        <- Tree chunk lvl1
                                            |
                  __________________________|_____________________________
                 /                  |                   |                \
                /                   |                   \                 \
            __T00__             ___T01__           ___T02__           ___T03__         <- Tree chunks lvl 0
           / /     \           / /      \         / /      \         / /      \
          / /       \         / /        \       / /       \        / /        \
         D1 D2 ... D128	     D1 D2 ... D128     D1 D2 ... D128     D1 D2 ... D128      <-  Data Chunks


    The split function continuously read the data and creates data chunks and send them to storage.
    When certain no of data chunks are created (defaultBranches), a signal is sent to create a tree
    entry. When the level 0 tree entries reaches certain threshold (defaultBranches), another signal
    is sent to a tree entry one level up.. and so on... until only the data is exhausted AND only one
    tree entry is present in certain level. The key of tree entry is given out as the rootKey of the file.

*/

var (
	errLoadingTreeRootChunk = errors.New("LoadTree Error: Could not load root chunk")
	errLoadingTreeChunk     = errors.New("LoadTree Error: Could not load chunk")
)

const (
	ChunkProcessors = 8
	splitTimeout    = time.Minute * 5
)

const (
	DataChunk = 0
	TreeChunk = 1
)

type PyramidSplitterParams struct {
	SplitterParams
	getter Getter
}

func NewPyramidSplitterParams(key Key, reader io.Reader, putter Putter, getter Getter, chunkSize int64) *PyramidSplitterParams {
	hashSize := putter.RefSize()
	return &PyramidSplitterParams{
		SplitterParams: SplitterParams{
			ChunkerParams: ChunkerParams{
				chunkSize: chunkSize,
				hashSize:  hashSize,
			},
			reader: reader,
			putter: putter,
			key:    key,
		},
		getter: getter,
	}
}

/*
	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	New chunks to store are store using the putter which the caller provides.
*/
func PyramidSplit(reader io.Reader, putter Putter, getter Getter) (Key, func(), error) {
	return NewPyramidSplitter(NewPyramidSplitterParams(nil, reader, putter, getter, DefaultChunkSize)).Split()
}

func PyramidAppend(key Key, reader io.Reader, putter Putter, getter Getter) (Key, func(), error) {
	return NewPyramidSplitter(NewPyramidSplitterParams(key, reader, putter, getter, DefaultChunkSize)).Append()
}

// Entry to create a tree node
type TreeEntry struct {
	level         int
	branchCount   int64
	subtreeSize   uint64
	chunk         []byte
	key           []byte
	index         int  // used in append to indicate the index of existing tree entry
	updatePending bool // indicates if the entry is loaded from existing tree
}

func NewTreeEntry(pyramid *PyramidChunker) *TreeEntry {
	return &TreeEntry{
		level:         0,
		branchCount:   0,
		subtreeSize:   0,
		chunk:         make([]byte, pyramid.chunkSize+8),
		key:           make([]byte, pyramid.hashSize),
		index:         0,
		updatePending: false,
	}
}

// Used by the hash processor to create a data/tree chunk and send to storage
type chunkJob struct {
	key      Key
	chunk    []byte
	parentWg *sync.WaitGroup
}

type PyramidChunker struct {
	chunkSize   int64
	hashSize    int64
	branches    int64
	reader      io.Reader
	putter      Putter
	getter      Getter
	key         Key
	workerCount int64
	workerLock  sync.RWMutex
	jobC        chan *chunkJob
	wg          *sync.WaitGroup
	errC        chan error
	quitC       chan bool
	rootKey     []byte
	chunkLevel  [][]*TreeEntry
}

func NewPyramidSplitter(params *PyramidSplitterParams) (self *PyramidChunker) {
	self = &PyramidChunker{}
	self.reader = params.reader
	self.hashSize = params.hashSize
	self.branches = params.chunkSize / self.hashSize
	self.chunkSize = self.hashSize * self.branches
	self.putter = params.putter
	self.getter = params.getter
	self.key = params.key
	self.workerCount = 0
	self.jobC = make(chan *chunkJob, 2*ChunkProcessors)
	self.wg = &sync.WaitGroup{}
	self.errC = make(chan error)
	self.quitC = make(chan bool)
	self.rootKey = make([]byte, self.hashSize)
	self.chunkLevel = make([][]*TreeEntry, self.branches)
	return
}

func (self *PyramidChunker) Join(key Key, getter Getter, depth int) LazySectionReader {
	return &LazyChunkReader{
		key:       key,
		depth:     depth,
		chunkSize: self.chunkSize,
		branches:  self.branches,
		hashSize:  self.hashSize,
		getter:    getter,
	}
}

func (self *PyramidChunker) incrementWorkerCount() {
	self.workerLock.Lock()
	defer self.workerLock.Unlock()
	self.workerCount += 1
}

func (self *PyramidChunker) getWorkerCount() int64 {
	self.workerLock.Lock()
	defer self.workerLock.Unlock()
	return self.workerCount
}

func (self *PyramidChunker) decrementWorkerCount() {
	self.workerLock.Lock()
	defer self.workerLock.Unlock()
	self.workerCount -= 1
}

func (self *PyramidChunker) Split() (k Key, wait func(), err error) {
	log.Debug("pyramid.chunker: Split()")

	self.wg.Add(1)
	self.prepareChunks(false)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
		self.wg.Wait()

		//We close errC here because this is passed down to 8 parallel routines underneath.
		// if a error happens in one of them.. that particular routine raises error...
		// once they all complete successfully, the control comes back and we can safely close this here.
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
	}
	return self.rootKey, self.putter.Wait, nil

}

func (self *PyramidChunker) Append() (k Key, wait func(), err error) {
	log.Debug("pyramid.chunker: Append()")
	// Load the right most unfinished tree chunks in every level
	self.loadTree()

	self.wg.Add(1)
	self.prepareChunks(true)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
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
	}

	return self.rootKey, self.putter.Wait, nil

}

func (self *PyramidChunker) processor(id int64) {
	defer self.decrementWorkerCount()
	for {
		select {

		case job, ok := <-self.jobC:
			if !ok {
				return
			}
			self.processChunk(id, job)
		case <-self.quitC:
			return
		}
	}
}

func (self *PyramidChunker) processChunk(id int64, job *chunkJob) {
	log.Debug("pyramid.chunker: processChunk()", "id", id)

	ref, err := self.putter.Put(job.chunk)
	if err != nil {
		self.errC <- err
	}

	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)
	copy(job.key, ref)

	// send off new chunk to storage
	job.parentWg.Done()
}

func (self *PyramidChunker) loadTree() error {
	log.Debug("pyramid.chunker: loadTree()")
	// Get the root chunk to get the total size
	chunkData, err := self.getter.Get(Reference(self.key))
	if err != nil {
		return errLoadingTreeRootChunk
	}
	chunkSize := chunkData.Size()
	log.Trace("pyramid.chunker: root chunk", "chunk.Size", chunkSize, "self.chunkSize", self.chunkSize)

	//if data size is less than a chunk... add a parent with update as pending
	if chunkSize <= self.chunkSize {
		newEntry := &TreeEntry{
			level:         0,
			branchCount:   1,
			subtreeSize:   uint64(chunkSize),
			chunk:         make([]byte, self.chunkSize+8),
			key:           make([]byte, self.hashSize),
			index:         0,
			updatePending: true,
		}
		copy(newEntry.chunk[8:], self.key)
		self.chunkLevel[0] = append(self.chunkLevel[0], newEntry)
		return nil
	}

	var treeSize int64
	var depth int
	treeSize = self.chunkSize
	for ; treeSize < chunkSize; treeSize *= self.branches {
		depth++
	}
	log.Trace("pyramid.chunker", "depth", depth)

	// Add the root chunk entry
	branchCount := int64(len(chunkData)-8) / self.hashSize
	newEntry := &TreeEntry{
		level:         depth - 1,
		branchCount:   branchCount,
		subtreeSize:   uint64(chunkSize),
		chunk:         chunkData,
		key:           self.key,
		index:         0,
		updatePending: true,
	}
	self.chunkLevel[depth-1] = append(self.chunkLevel[depth-1], newEntry)

	// Add the rest of the tree
	for lvl := depth - 1; lvl >= 1; lvl-- {

		//TODO(jmozah): instead of loading finished branches and then trim in the end,
		//avoid loading them in the first place
		for _, ent := range self.chunkLevel[lvl] {
			branchCount = int64(len(ent.chunk)-8) / self.hashSize
			for i := int64(0); i < branchCount; i++ {
				key := ent.chunk[8+(i*self.hashSize) : 8+((i+1)*self.hashSize)]
				newChunkData, err := self.getter.Get(Reference(key))
				if err != nil {
					return errLoadingTreeChunk
				}
				newChunkSize := newChunkData.Size()
				bewBranchCount := int64(len(newChunkData)-8) / self.hashSize
				newEntry := &TreeEntry{
					level:         lvl - 1,
					branchCount:   bewBranchCount,
					subtreeSize:   uint64(newChunkSize),
					chunk:         newChunkData,
					key:           key,
					index:         0,
					updatePending: true,
				}
				self.chunkLevel[lvl-1] = append(self.chunkLevel[lvl-1], newEntry)

			}

			// We need to get only the right most unfinished branch.. so trim all finished branches
			if int64(len(self.chunkLevel[lvl-1])) >= self.branches {
				self.chunkLevel[lvl-1] = nil
			}
		}
	}

	return nil
}

func (self *PyramidChunker) prepareChunks(isAppend bool) {
	log.Debug("pyramid.chunker: prepareChunks", "isAppend", isAppend)
	defer self.wg.Done()

	chunkWG := &sync.WaitGroup{}

	self.incrementWorkerCount()

	go self.processor(self.workerCount)

	parent := NewTreeEntry(self)
	var unfinishedChunkData ChunkData
	var unfinishedChunkSize int64

	if isAppend && len(self.chunkLevel[0]) != 0 {
		lastIndex := len(self.chunkLevel[0]) - 1
		ent := self.chunkLevel[0][lastIndex]

		if ent.branchCount < self.branches {
			parent = &TreeEntry{
				level:         0,
				branchCount:   ent.branchCount,
				subtreeSize:   ent.subtreeSize,
				chunk:         ent.chunk,
				key:           ent.key,
				index:         lastIndex,
				updatePending: true,
			}

			lastBranch := parent.branchCount - 1
			lastKey := parent.chunk[8+lastBranch*self.hashSize : 8+(lastBranch+1)*self.hashSize]

			var err error
			unfinishedChunkData, err = self.getter.Get(lastKey)
			if err != nil {
				self.errC <- err
			}
			unfinishedChunkSize = unfinishedChunkData.Size()
			if unfinishedChunkSize < self.chunkSize {
				parent.subtreeSize = parent.subtreeSize - uint64(unfinishedChunkSize)
				parent.branchCount = parent.branchCount - 1
			} else {
				unfinishedChunkData = nil
			}
		}
	}

	for index := 0; ; index++ {
		var err error
		chunkData := make([]byte, self.chunkSize+8)

		var readBytes int

		if unfinishedChunkData != nil {
			copy(chunkData, unfinishedChunkData)
			readBytes += int(unfinishedChunkSize)
			unfinishedChunkData = nil
			log.Trace("pyramid.chunker: found unfinished chunk", "readBytes", readBytes)
		}

		var res []byte
		res, err = ioutil.ReadAll(io.LimitReader(self.reader, int64(len(chunkData)-(8+readBytes))))

		// hack for ioutil.ReadAll:
		// a successful call to ioutil.ReadAll returns err == nil, not err == EOF, whereas we
		// want to propagate the io.EOF error
		if len(res) == 0 && err == nil {
			err = io.EOF
		}
		copy(chunkData[8+readBytes:], res)

		readBytes += len(res)
		log.Trace("pyramid.chunker: copied all data", "readBytes", readBytes)

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if parent.branchCount == 1 {
					// Data is exactly one chunk.. pick the last chunk key as root
					chunkWG.Wait()
					lastChunksKey := parent.chunk[8 : 8+self.hashSize]
					copy(self.rootKey, lastChunksKey)
					break
				}
			} else {
				close(self.quitC)
				break
			}
		}

		// Data ended in chunk boundary.. just signal to start bulding tree
		if readBytes == 0 {
			self.buildTree(isAppend, parent, chunkWG, true)
			break
		} else {
			pkey := self.enqueueDataChunk(chunkData, uint64(readBytes), parent, chunkWG)

			// update tree related parent data structures
			parent.subtreeSize += uint64(readBytes)
			parent.branchCount++

			// Data got exhausted... signal to send any parent tree related chunks
			if int64(readBytes) < self.chunkSize {

				// only one data chunk .. so dont add any parent chunk
				if parent.branchCount <= 1 {
					chunkWG.Wait()
					copy(self.rootKey, pkey)
					break
				}

				self.buildTree(isAppend, parent, chunkWG, true)
				break
			}

			if parent.branchCount == self.branches {
				self.buildTree(isAppend, parent, chunkWG, false)
				parent = NewTreeEntry(self)
			}

		}

		workers := self.getWorkerCount()
		if int64(len(self.jobC)) > workers && workers < ChunkProcessors {
			self.incrementWorkerCount()
			go self.processor(self.workerCount)
		}

	}

}

func (self *PyramidChunker) buildTree(isAppend bool, ent *TreeEntry, chunkWG *sync.WaitGroup, last bool) {
	chunkWG.Wait()
	self.enqueueTreeChunk(ent, chunkWG, last)

	compress := false
	endLvl := self.branches
	for lvl := int64(0); lvl < self.branches; lvl++ {
		lvlCount := int64(len(self.chunkLevel[lvl]))
		if lvlCount >= self.branches {
			endLvl = lvl + 1
			compress = true
			break
		}
	}

	if !compress && !last {
		return
	}

	// Wait for all the keys to be processed before compressing the tree
	chunkWG.Wait()

	for lvl := int64(ent.level); lvl < endLvl; lvl++ {

		lvlCount := int64(len(self.chunkLevel[lvl]))
		if lvlCount == 1 && last {
			copy(self.rootKey, self.chunkLevel[lvl][0].key)
			return
		}

		for startCount := int64(0); startCount < lvlCount; startCount += self.branches {

			endCount := startCount + self.branches
			if endCount > lvlCount {
				endCount = lvlCount
			}

			var nextLvlCount int64
			var tempEntry *TreeEntry
			if len(self.chunkLevel[lvl+1]) > 0 {
				nextLvlCount = int64(len(self.chunkLevel[lvl+1]) - 1)
				tempEntry = self.chunkLevel[lvl+1][nextLvlCount]
			}
			if isAppend && tempEntry != nil && tempEntry.updatePending {
				updateEntry := &TreeEntry{
					level:         int(lvl + 1),
					branchCount:   0,
					subtreeSize:   0,
					chunk:         make([]byte, self.chunkSize+8),
					key:           make([]byte, self.hashSize),
					index:         int(nextLvlCount),
					updatePending: true,
				}
				for index := int64(0); index < lvlCount; index++ {
					updateEntry.branchCount++
					updateEntry.subtreeSize += self.chunkLevel[lvl][index].subtreeSize
					copy(updateEntry.chunk[8+(index*self.hashSize):8+((index+1)*self.hashSize)], self.chunkLevel[lvl][index].key[:self.hashSize])
				}

				self.enqueueTreeChunk(updateEntry, chunkWG, last)

			} else {

				noOfBranches := endCount - startCount
				newEntry := &TreeEntry{
					level:         int(lvl + 1),
					branchCount:   noOfBranches,
					subtreeSize:   0,
					chunk:         make([]byte, (noOfBranches*self.hashSize)+8),
					key:           make([]byte, self.hashSize),
					index:         int(nextLvlCount),
					updatePending: false,
				}

				index := int64(0)
				for i := startCount; i < endCount; i++ {
					entry := self.chunkLevel[lvl][i]
					newEntry.subtreeSize += entry.subtreeSize
					copy(newEntry.chunk[8+(index*self.hashSize):8+((index+1)*self.hashSize)], entry.key[:self.hashSize])
					index++
				}

				self.enqueueTreeChunk(newEntry, chunkWG, last)

			}

		}

		if !isAppend {
			chunkWG.Wait()
			if compress {
				self.chunkLevel[lvl] = nil
			}
		}
	}

}

func (self *PyramidChunker) enqueueTreeChunk(ent *TreeEntry, chunkWG *sync.WaitGroup, last bool) {
	if ent != nil && ent.branchCount > 0 {

		// wait for data chunks to get over before processing the tree chunk
		if last {
			chunkWG.Wait()
		}

		binary.LittleEndian.PutUint64(ent.chunk[:8], ent.subtreeSize)
		ent.key = make([]byte, self.hashSize)
		chunkWG.Add(1)
		select {
		case self.jobC <- &chunkJob{ent.key, ent.chunk[:ent.branchCount*self.hashSize+8], chunkWG}:
		case <-self.quitC:
		}

		// Update or append based on weather it is a new entry or being reused
		if ent.updatePending {
			chunkWG.Wait()
			self.chunkLevel[ent.level][ent.index] = ent
		} else {
			self.chunkLevel[ent.level] = append(self.chunkLevel[ent.level], ent)
		}

	}
}

func (self *PyramidChunker) enqueueDataChunk(chunkData []byte, size uint64, parent *TreeEntry, chunkWG *sync.WaitGroup) Key {
	binary.LittleEndian.PutUint64(chunkData[:8], size)
	pkey := parent.chunk[8+parent.branchCount*self.hashSize : 8+(parent.branchCount+1)*self.hashSize]

	chunkWG.Add(1)
	select {
	case self.jobC <- &chunkJob{pkey, chunkData[:size+8], chunkWG}:
	case <-self.quitC:
	}

	return pkey

}
