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
	"sync"
	"time"
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
	ChunkProcessors       = 8
	DefaultBranches int64 = 128
	splitTimeout          = time.Minute * 5
)

const (
	DataChunk = 0
	TreeChunk = 1
)

type ChunkerParams struct {
	Branches int64
	Hash     string
}

func NewChunkerParams() *ChunkerParams {
	return &ChunkerParams{
		Branches: DefaultBranches,
		Hash:     SHA3Hash,
	}
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
	key       Key
	chunk     []byte
	size      int64
	parentWg  *sync.WaitGroup
	chunkType int // used to identify the tree related chunks for debugging
	chunkLvl  int // leaf-1 is level 0 and goes upwards until it reaches root
}

type PyramidChunker struct {
	hashFunc    SwarmHasher
	chunkSize   int64
	hashSize    int64
	branches    int64
	workerCount int64
	workerLock  sync.RWMutex
}

func NewPyramidChunker(params *ChunkerParams) (self *PyramidChunker) {
	self = &PyramidChunker{}
	self.hashFunc = MakeHashFunc(params.Hash)
	self.branches = params.Branches
	self.hashSize = int64(self.hashFunc().Size())
	self.chunkSize = self.hashSize * self.branches
	self.workerCount = 0
	return
}

func (self *PyramidChunker) Join(key Key, chunkC chan *Chunk) LazySectionReader {
	return &LazyChunkReader{
		key:       key,
		chunkC:    chunkC,
		chunkSize: self.chunkSize,
		branches:  self.branches,
		hashSize:  self.hashSize,
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

func (self *PyramidChunker) Split(data io.Reader, size int64, chunkC chan *Chunk, storageWG, processorWG *sync.WaitGroup) (Key, error) {
	jobC := make(chan *chunkJob, 2*ChunkProcessors)
	wg := &sync.WaitGroup{}
	errC := make(chan error)
	quitC := make(chan bool)
	rootKey := make([]byte, self.hashSize)
	chunkLevel := make([][]*TreeEntry, self.branches)

	wg.Add(1)
	go self.prepareChunks(false, chunkLevel, data, rootKey, quitC, wg, jobC, processorWG, chunkC, errC, storageWG)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
		wg.Wait()

		// if storage waitgroup is non-nil, we wait for storage to finish too
		if storageWG != nil {
			storageWG.Wait()
		}
		//We close errC here because this is passed down to 8 parallel routines underneath.
		// if a error happens in one of them.. that particular routine raises error...
		// once they all complete successfully, the control comes back and we can safely close this here.
		close(errC)
	}()

	defer close(quitC)

	select {
	case err := <-errC:
		if err != nil {
			return nil, err
		}
	case <-time.NewTimer(splitTimeout).C:
	}
	return rootKey, nil

}

func (self *PyramidChunker) Append(key Key, data io.Reader, chunkC chan *Chunk, storageWG, processorWG *sync.WaitGroup) (Key, error) {
	quitC := make(chan bool)
	rootKey := make([]byte, self.hashSize)
	chunkLevel := make([][]*TreeEntry, self.branches)

	// Load the right most unfinished tree chunks in every level
	self.loadTree(chunkLevel, key, chunkC, quitC)

	jobC := make(chan *chunkJob, 2*ChunkProcessors)
	wg := &sync.WaitGroup{}
	errC := make(chan error)

	wg.Add(1)
	go self.prepareChunks(true, chunkLevel, data, rootKey, quitC, wg, jobC, processorWG, chunkC, errC, storageWG)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
		wg.Wait()

		// if storage waitgroup is non-nil, we wait for storage to finish too
		if storageWG != nil {
			storageWG.Wait()
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
	}
	return rootKey, nil

}

func (self *PyramidChunker) processor(id int64, jobC chan *chunkJob, chunkC chan *Chunk, errC chan error, quitC chan bool, swg, wwg *sync.WaitGroup) {
	defer self.decrementWorkerCount()

	hasher := self.hashFunc()
	if wwg != nil {
		defer wwg.Done()
	}
	for {
		select {

		case job, ok := <-jobC:
			if !ok {
				return
			}
			self.processChunk(id, hasher, job, chunkC, swg)
		case <-quitC:
			return
		}
	}
}

func (self *PyramidChunker) processChunk(id int64, hasher SwarmHash, job *chunkJob, chunkC chan *Chunk, swg *sync.WaitGroup) {
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
		chunkC <- newChunk
	}
}

func (self *PyramidChunker) loadTree(chunkLevel [][]*TreeEntry, key Key, chunkC chan *Chunk, quitC chan bool) error {
	// Get the root chunk to get the total size
	chunk := retrieve(key, chunkC, quitC)
	if chunk == nil {
		return errLoadingTreeRootChunk
	}

	//if data size is less than a chunk... add a parent with update as pending
	if chunk.Size <= self.chunkSize {
		newEntry := &TreeEntry{
			level:         0,
			branchCount:   1,
			subtreeSize:   uint64(chunk.Size),
			chunk:         make([]byte, self.chunkSize+8),
			key:           make([]byte, self.hashSize),
			index:         0,
			updatePending: true,
		}
		copy(newEntry.chunk[8:], chunk.Key)
		chunkLevel[0] = append(chunkLevel[0], newEntry)
		return nil
	}

	var treeSize int64
	var depth int
	treeSize = self.chunkSize
	for ; treeSize < chunk.Size; treeSize *= self.branches {
		depth++
	}

	// Add the root chunk entry
	branchCount := int64(len(chunk.SData)-8) / self.hashSize
	newEntry := &TreeEntry{
		level:         depth - 1,
		branchCount:   branchCount,
		subtreeSize:   uint64(chunk.Size),
		chunk:         chunk.SData,
		key:           key,
		index:         0,
		updatePending: true,
	}
	chunkLevel[depth-1] = append(chunkLevel[depth-1], newEntry)

	// Add the rest of the tree
	for lvl := (depth - 1); lvl >= 1; lvl-- {

		//TODO(jmozah): instead of loading finished branches and then trim in the end,
		//avoid loading them in the first place
		for _, ent := range chunkLevel[lvl] {
			branchCount = int64(len(ent.chunk)-8) / self.hashSize
			for i := int64(0); i < branchCount; i++ {
				key := ent.chunk[8+(i*self.hashSize) : 8+((i+1)*self.hashSize)]
				newChunk := retrieve(key, chunkC, quitC)
				if newChunk == nil {
					return errLoadingTreeChunk
				}
				bewBranchCount := int64(len(newChunk.SData)-8) / self.hashSize
				newEntry := &TreeEntry{
					level:         lvl - 1,
					branchCount:   bewBranchCount,
					subtreeSize:   uint64(newChunk.Size),
					chunk:         newChunk.SData,
					key:           key,
					index:         0,
					updatePending: true,
				}
				chunkLevel[lvl-1] = append(chunkLevel[lvl-1], newEntry)

			}

			// We need to get only the right most unfinished branch.. so trim all finished branches
			if int64(len(chunkLevel[lvl-1])) >= self.branches {
				chunkLevel[lvl-1] = nil
			}
		}
	}

	return nil
}

func (self *PyramidChunker) prepareChunks(isAppend bool, chunkLevel [][]*TreeEntry, data io.Reader, rootKey []byte, quitC chan bool, wg *sync.WaitGroup, jobC chan *chunkJob, processorWG *sync.WaitGroup, chunkC chan *Chunk, errC chan error, storageWG *sync.WaitGroup) {
	defer wg.Done()

	chunkWG := &sync.WaitGroup{}
	totalDataSize := 0

	// processorWG keeps track of workers spawned for hashing chunks
	if processorWG != nil {
		processorWG.Add(1)
	}

	self.incrementWorkerCount()
	go self.processor(self.workerCount, jobC, chunkC, errC, quitC, storageWG, processorWG)

	parent := NewTreeEntry(self)
	var unFinishedChunk *Chunk

	if isAppend == true && len(chunkLevel[0]) != 0 {

		lastIndex := len(chunkLevel[0]) - 1
		ent := chunkLevel[0][lastIndex]

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

			unFinishedChunk = retrieve(lastKey, chunkC, quitC)
			if unFinishedChunk.Size < self.chunkSize {

				parent.subtreeSize = parent.subtreeSize - uint64(unFinishedChunk.Size)
				parent.branchCount = parent.branchCount - 1
			} else {
				unFinishedChunk = nil
			}
		}
	}

	for index := 0; ; index++ {

		var n int
		var err error
		chunkData := make([]byte, self.chunkSize+8)
		if unFinishedChunk != nil {
			copy(chunkData, unFinishedChunk.SData)
			n, err = data.Read(chunkData[8+unFinishedChunk.Size:])
			n += int(unFinishedChunk.Size)
			unFinishedChunk = nil
		} else {
			n, err = data.Read(chunkData[8:])
		}

		totalDataSize += n
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				if parent.branchCount == 1 {
					// Data is exactly one chunk.. pick the last chunk key as root
					chunkWG.Wait()
					lastChunksKey := parent.chunk[8 : 8+self.hashSize]
					copy(rootKey, lastChunksKey)
					break
				}
			} else {
				close(quitC)
				break
			}
		}

		// Data ended in chunk boundary.. just signal to start bulding tree
		if n == 0 {
			self.buildTree(isAppend, chunkLevel, parent, chunkWG, jobC, quitC, true, rootKey)
			break
		} else {

			pkey := self.enqueueDataChunk(chunkData, uint64(n), parent, chunkWG, jobC, quitC)

			// update tree related parent data structures
			parent.subtreeSize += uint64(n)
			parent.branchCount++

			// Data got exhausted... signal to send any parent tree related chunks
			if int64(n) < self.chunkSize {

				// only one data chunk .. so dont add any parent chunk
				if parent.branchCount <= 1 {
					chunkWG.Wait()
					copy(rootKey, pkey)
					break
				}

				self.buildTree(isAppend, chunkLevel, parent, chunkWG, jobC, quitC, true, rootKey)
				break
			}

			if parent.branchCount == self.branches {
				self.buildTree(isAppend, chunkLevel, parent, chunkWG, jobC, quitC, false, rootKey)
				parent = NewTreeEntry(self)
			}

		}

		workers := self.getWorkerCount()
		if int64(len(jobC)) > workers && workers < ChunkProcessors {
			if processorWG != nil {
				processorWG.Add(1)
			}
			self.incrementWorkerCount()
			go self.processor(self.workerCount, jobC, chunkC, errC, quitC, storageWG, processorWG)
		}

	}

}

func (self *PyramidChunker) buildTree(isAppend bool, chunkLevel [][]*TreeEntry, ent *TreeEntry, chunkWG *sync.WaitGroup, jobC chan *chunkJob, quitC chan bool, last bool, rootKey []byte) {
	chunkWG.Wait()
	self.enqueueTreeChunk(chunkLevel, ent, chunkWG, jobC, quitC, last)

	compress := false
	endLvl := self.branches
	for lvl := int64(0); lvl < self.branches; lvl++ {
		lvlCount := int64(len(chunkLevel[lvl]))
		if lvlCount >= self.branches {
			endLvl = lvl + 1
			compress = true
			break
		}
	}

	if compress == false && last == false {
		return
	}

	// Wait for all the keys to be processed before compressing the tree
	chunkWG.Wait()

	for lvl := int64(ent.level); lvl < endLvl; lvl++ {

		lvlCount := int64(len(chunkLevel[lvl]))
		if lvlCount == 1 && last == true {
			copy(rootKey, chunkLevel[lvl][0].key)
			return
		}

		for startCount := int64(0); startCount < lvlCount; startCount += self.branches {

			endCount := startCount + self.branches
			if endCount > lvlCount {
				endCount = lvlCount
			}

			var nextLvlCount int64
			var tempEntry *TreeEntry
			if len(chunkLevel[lvl+1]) > 0 {
				nextLvlCount = int64(len(chunkLevel[lvl+1]) - 1)
				tempEntry = chunkLevel[lvl+1][nextLvlCount]
			}
			if isAppend == true && tempEntry != nil && tempEntry.updatePending == true {
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
					updateEntry.subtreeSize += chunkLevel[lvl][index].subtreeSize
					copy(updateEntry.chunk[8+(index*self.hashSize):8+((index+1)*self.hashSize)], chunkLevel[lvl][index].key[:self.hashSize])
				}

				self.enqueueTreeChunk(chunkLevel, updateEntry, chunkWG, jobC, quitC, last)

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
					entry := chunkLevel[lvl][i]
					newEntry.subtreeSize += entry.subtreeSize
					copy(newEntry.chunk[8+(index*self.hashSize):8+((index+1)*self.hashSize)], entry.key[:self.hashSize])
					index++
				}

				self.enqueueTreeChunk(chunkLevel, newEntry, chunkWG, jobC, quitC, last)

			}

		}

		if isAppend == false {
			chunkWG.Wait()
			if compress == true {
				chunkLevel[lvl] = nil
			}
		}
	}

}

func (self *PyramidChunker) enqueueTreeChunk(chunkLevel [][]*TreeEntry, ent *TreeEntry, chunkWG *sync.WaitGroup, jobC chan *chunkJob, quitC chan bool, last bool) {
	if ent != nil {

		// wait for data chunks to get over before processing the tree chunk
		if last == true {
			chunkWG.Wait()
		}

		binary.LittleEndian.PutUint64(ent.chunk[:8], ent.subtreeSize)
		ent.key = make([]byte, self.hashSize)
		chunkWG.Add(1)
		select {
		case jobC <- &chunkJob{ent.key, ent.chunk[:ent.branchCount*self.hashSize+8], int64(ent.subtreeSize), chunkWG, TreeChunk, 0}:
		case <-quitC:
		}

		// Update or append based on weather it is a new entry or being reused
		if ent.updatePending == true {
			chunkWG.Wait()
			chunkLevel[ent.level][ent.index] = ent
		} else {
			chunkLevel[ent.level] = append(chunkLevel[ent.level], ent)
		}

	}
}

func (self *PyramidChunker) enqueueDataChunk(chunkData []byte, size uint64, parent *TreeEntry, chunkWG *sync.WaitGroup, jobC chan *chunkJob, quitC chan bool) Key {
	binary.LittleEndian.PutUint64(chunkData[:8], size)
	pkey := parent.chunk[8+parent.branchCount*self.hashSize : 8+(parent.branchCount+1)*self.hashSize]

	chunkWG.Add(1)
	select {
	case jobC <- &chunkJob{pkey, chunkData[:size+8], int64(size), chunkWG, DataChunk, -1}:
	case <-quitC:
	}

	return pkey

}
