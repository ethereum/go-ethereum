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
	"io"
	"io/ioutil"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/swarm/log"
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

func NewPyramidSplitterParams(addr Address, reader io.Reader, putter Putter, getter Getter, chunkSize int64) *PyramidSplitterParams {
	hashSize := putter.RefSize()
	return &PyramidSplitterParams{
		SplitterParams: SplitterParams{
			ChunkerParams: ChunkerParams{
				chunkSize: chunkSize,
				hashSize:  hashSize,
			},
			reader: reader,
			putter: putter,
			addr:   addr,
		},
		getter: getter,
	}
}

/*
	When splitting, data is given as a SectionReader, and the key is a hashSize long byte slice (Key), the root hash of the entire content will fill this once processing finishes.
	New chunks to store are store using the putter which the caller provides.
*/
func PyramidSplit(ctx context.Context, reader io.Reader, putter Putter, getter Getter) (Address, func(context.Context) error, error) {
	return NewPyramidSplitter(NewPyramidSplitterParams(nil, reader, putter, getter, DefaultChunkSize)).Split(ctx)
}

func PyramidAppend(ctx context.Context, addr Address, reader io.Reader, putter Putter, getter Getter) (Address, func(context.Context) error, error) {
	return NewPyramidSplitter(NewPyramidSplitterParams(addr, reader, putter, getter, DefaultChunkSize)).Append(ctx)
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
	key      Address
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
	key         Address
	workerCount int64
	workerLock  sync.RWMutex
	jobC        chan *chunkJob
	wg          *sync.WaitGroup
	errC        chan error
	quitC       chan bool
	rootKey     []byte
	chunkLevel  [][]*TreeEntry
}

func NewPyramidSplitter(params *PyramidSplitterParams) (pc *PyramidChunker) {
	pc = &PyramidChunker{}
	pc.reader = params.reader
	pc.hashSize = params.hashSize
	pc.branches = params.chunkSize / pc.hashSize
	pc.chunkSize = pc.hashSize * pc.branches
	pc.putter = params.putter
	pc.getter = params.getter
	pc.key = params.addr
	pc.workerCount = 0
	pc.jobC = make(chan *chunkJob, 2*ChunkProcessors)
	pc.wg = &sync.WaitGroup{}
	pc.errC = make(chan error)
	pc.quitC = make(chan bool)
	pc.rootKey = make([]byte, pc.hashSize)
	pc.chunkLevel = make([][]*TreeEntry, pc.branches)
	return
}

func (pc *PyramidChunker) Join(addr Address, getter Getter, depth int) LazySectionReader {
	return &LazyChunkReader{
		key:       addr,
		depth:     depth,
		chunkSize: pc.chunkSize,
		branches:  pc.branches,
		hashSize:  pc.hashSize,
		getter:    getter,
	}
}

func (pc *PyramidChunker) incrementWorkerCount() {
	pc.workerLock.Lock()
	defer pc.workerLock.Unlock()
	pc.workerCount += 1
}

func (pc *PyramidChunker) getWorkerCount() int64 {
	pc.workerLock.Lock()
	defer pc.workerLock.Unlock()
	return pc.workerCount
}

func (pc *PyramidChunker) decrementWorkerCount() {
	pc.workerLock.Lock()
	defer pc.workerLock.Unlock()
	pc.workerCount -= 1
}

func (pc *PyramidChunker) Split(ctx context.Context) (k Address, wait func(context.Context) error, err error) {
	log.Debug("pyramid.chunker: Split()")

	pc.wg.Add(1)
	pc.prepareChunks(false)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
		pc.wg.Wait()

		//We close errC here because this is passed down to 8 parallel routines underneath.
		// if a error happens in one of them.. that particular routine raises error...
		// once they all complete successfully, the control comes back and we can safely close this here.
		close(pc.errC)
	}()

	defer close(pc.quitC)
	defer pc.putter.Close()

	select {
	case err := <-pc.errC:
		if err != nil {
			return nil, nil, err
		}
	case <-time.NewTimer(splitTimeout).C:
	}
	return pc.rootKey, pc.putter.Wait, nil

}

func (pc *PyramidChunker) Append(ctx context.Context) (k Address, wait func(context.Context) error, err error) {
	log.Debug("pyramid.chunker: Append()")
	// Load the right most unfinished tree chunks in every level
	pc.loadTree()

	pc.wg.Add(1)
	pc.prepareChunks(true)

	// closes internal error channel if all subprocesses in the workgroup finished
	go func() {

		// waiting for all chunks to finish
		pc.wg.Wait()

		close(pc.errC)
	}()

	defer close(pc.quitC)
	defer pc.putter.Close()

	select {
	case err := <-pc.errC:
		if err != nil {
			return nil, nil, err
		}
	case <-time.NewTimer(splitTimeout).C:
	}

	return pc.rootKey, pc.putter.Wait, nil

}

func (pc *PyramidChunker) processor(id int64) {
	defer pc.decrementWorkerCount()
	for {
		select {

		case job, ok := <-pc.jobC:
			if !ok {
				return
			}
			pc.processChunk(id, job)
		case <-pc.quitC:
			return
		}
	}
}

func (pc *PyramidChunker) processChunk(id int64, job *chunkJob) {
	log.Debug("pyramid.chunker: processChunk()", "id", id)

	ref, err := pc.putter.Put(context.TODO(), job.chunk)
	if err != nil {
		pc.errC <- err
	}

	// report hash of this chunk one level up (keys corresponds to the proper subslice of the parent chunk)
	copy(job.key, ref)

	// send off new chunk to storage
	job.parentWg.Done()
}

func (pc *PyramidChunker) loadTree() error {
	log.Debug("pyramid.chunker: loadTree()")
	// Get the root chunk to get the total size
	chunkData, err := pc.getter.Get(context.TODO(), Reference(pc.key))
	if err != nil {
		return errLoadingTreeRootChunk
	}
	chunkSize := chunkData.Size()
	log.Trace("pyramid.chunker: root chunk", "chunk.Size", chunkSize, "pc.chunkSize", pc.chunkSize)

	//if data size is less than a chunk... add a parent with update as pending
	if chunkSize <= pc.chunkSize {
		newEntry := &TreeEntry{
			level:         0,
			branchCount:   1,
			subtreeSize:   uint64(chunkSize),
			chunk:         make([]byte, pc.chunkSize+8),
			key:           make([]byte, pc.hashSize),
			index:         0,
			updatePending: true,
		}
		copy(newEntry.chunk[8:], pc.key)
		pc.chunkLevel[0] = append(pc.chunkLevel[0], newEntry)
		return nil
	}

	var treeSize int64
	var depth int
	treeSize = pc.chunkSize
	for ; treeSize < chunkSize; treeSize *= pc.branches {
		depth++
	}
	log.Trace("pyramid.chunker", "depth", depth)

	// Add the root chunk entry
	branchCount := int64(len(chunkData)-8) / pc.hashSize
	newEntry := &TreeEntry{
		level:         depth - 1,
		branchCount:   branchCount,
		subtreeSize:   uint64(chunkSize),
		chunk:         chunkData,
		key:           pc.key,
		index:         0,
		updatePending: true,
	}
	pc.chunkLevel[depth-1] = append(pc.chunkLevel[depth-1], newEntry)

	// Add the rest of the tree
	for lvl := depth - 1; lvl >= 1; lvl-- {

		//TODO(jmozah): instead of loading finished branches and then trim in the end,
		//avoid loading them in the first place
		for _, ent := range pc.chunkLevel[lvl] {
			branchCount = int64(len(ent.chunk)-8) / pc.hashSize
			for i := int64(0); i < branchCount; i++ {
				key := ent.chunk[8+(i*pc.hashSize) : 8+((i+1)*pc.hashSize)]
				newChunkData, err := pc.getter.Get(context.TODO(), Reference(key))
				if err != nil {
					return errLoadingTreeChunk
				}
				newChunkSize := newChunkData.Size()
				bewBranchCount := int64(len(newChunkData)-8) / pc.hashSize
				newEntry := &TreeEntry{
					level:         lvl - 1,
					branchCount:   bewBranchCount,
					subtreeSize:   uint64(newChunkSize),
					chunk:         newChunkData,
					key:           key,
					index:         0,
					updatePending: true,
				}
				pc.chunkLevel[lvl-1] = append(pc.chunkLevel[lvl-1], newEntry)

			}

			// We need to get only the right most unfinished branch.. so trim all finished branches
			if int64(len(pc.chunkLevel[lvl-1])) >= pc.branches {
				pc.chunkLevel[lvl-1] = nil
			}
		}
	}

	return nil
}

func (pc *PyramidChunker) prepareChunks(isAppend bool) {
	log.Debug("pyramid.chunker: prepareChunks", "isAppend", isAppend)
	defer pc.wg.Done()

	chunkWG := &sync.WaitGroup{}

	pc.incrementWorkerCount()

	go pc.processor(pc.workerCount)

	parent := NewTreeEntry(pc)
	var unfinishedChunkData ChunkData
	var unfinishedChunkSize int64

	if isAppend && len(pc.chunkLevel[0]) != 0 {
		lastIndex := len(pc.chunkLevel[0]) - 1
		ent := pc.chunkLevel[0][lastIndex]

		if ent.branchCount < pc.branches {
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
			lastKey := parent.chunk[8+lastBranch*pc.hashSize : 8+(lastBranch+1)*pc.hashSize]

			var err error
			unfinishedChunkData, err = pc.getter.Get(context.TODO(), lastKey)
			if err != nil {
				pc.errC <- err
			}
			unfinishedChunkSize = unfinishedChunkData.Size()
			if unfinishedChunkSize < pc.chunkSize {
				parent.subtreeSize = parent.subtreeSize - uint64(unfinishedChunkSize)
				parent.branchCount = parent.branchCount - 1
			} else {
				unfinishedChunkData = nil
			}
		}
	}

	for index := 0; ; index++ {
		var err error
		chunkData := make([]byte, pc.chunkSize+8)

		var readBytes int

		if unfinishedChunkData != nil {
			copy(chunkData, unfinishedChunkData)
			readBytes += int(unfinishedChunkSize)
			unfinishedChunkData = nil
			log.Trace("pyramid.chunker: found unfinished chunk", "readBytes", readBytes)
		}

		var res []byte
		res, err = ioutil.ReadAll(io.LimitReader(pc.reader, int64(len(chunkData)-(8+readBytes))))

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

				pc.cleanChunkLevels()

				// Check if we are appending or the chunk is the only one.
				if parent.branchCount == 1 && (pc.depth() == 0 || isAppend) {
					// Data is exactly one chunk.. pick the last chunk key as root
					chunkWG.Wait()
					lastChunksKey := parent.chunk[8 : 8+pc.hashSize]
					copy(pc.rootKey, lastChunksKey)
					break
				}
			} else {
				close(pc.quitC)
				break
			}
		}

		// Data ended in chunk boundary.. just signal to start bulding tree
		if readBytes == 0 {
			pc.buildTree(isAppend, parent, chunkWG, true, nil)
			break
		} else {
			pkey := pc.enqueueDataChunk(chunkData, uint64(readBytes), parent, chunkWG)

			// update tree related parent data structures
			parent.subtreeSize += uint64(readBytes)
			parent.branchCount++

			// Data got exhausted... signal to send any parent tree related chunks
			if int64(readBytes) < pc.chunkSize {

				pc.cleanChunkLevels()

				// only one data chunk .. so dont add any parent chunk
				if parent.branchCount <= 1 {
					chunkWG.Wait()

					if isAppend || pc.depth() == 0 {
						// No need to build the tree if the depth is 0
						// or we are appending.
						// Just use the last key.
						copy(pc.rootKey, pkey)
					} else {
						// We need to build the tree and and provide the lonely
						// chunk key to replace the last tree chunk key.
						pc.buildTree(isAppend, parent, chunkWG, true, pkey)
					}
					break
				}

				pc.buildTree(isAppend, parent, chunkWG, true, nil)
				break
			}

			if parent.branchCount == pc.branches {
				pc.buildTree(isAppend, parent, chunkWG, false, nil)
				parent = NewTreeEntry(pc)
			}

		}

		workers := pc.getWorkerCount()
		if int64(len(pc.jobC)) > workers && workers < ChunkProcessors {
			pc.incrementWorkerCount()
			go pc.processor(pc.workerCount)
		}

	}

}

func (pc *PyramidChunker) buildTree(isAppend bool, ent *TreeEntry, chunkWG *sync.WaitGroup, last bool, lonelyChunkKey []byte) {
	chunkWG.Wait()
	pc.enqueueTreeChunk(ent, chunkWG, last)

	compress := false
	endLvl := pc.branches
	for lvl := int64(0); lvl < pc.branches; lvl++ {
		lvlCount := int64(len(pc.chunkLevel[lvl]))
		if lvlCount >= pc.branches {
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

		lvlCount := int64(len(pc.chunkLevel[lvl]))
		if lvlCount == 1 && last {
			copy(pc.rootKey, pc.chunkLevel[lvl][0].key)
			return
		}

		for startCount := int64(0); startCount < lvlCount; startCount += pc.branches {

			endCount := startCount + pc.branches
			if endCount > lvlCount {
				endCount = lvlCount
			}

			var nextLvlCount int64
			var tempEntry *TreeEntry
			if len(pc.chunkLevel[lvl+1]) > 0 {
				nextLvlCount = int64(len(pc.chunkLevel[lvl+1]) - 1)
				tempEntry = pc.chunkLevel[lvl+1][nextLvlCount]
			}
			if isAppend && tempEntry != nil && tempEntry.updatePending {
				updateEntry := &TreeEntry{
					level:         int(lvl + 1),
					branchCount:   0,
					subtreeSize:   0,
					chunk:         make([]byte, pc.chunkSize+8),
					key:           make([]byte, pc.hashSize),
					index:         int(nextLvlCount),
					updatePending: true,
				}
				for index := int64(0); index < lvlCount; index++ {
					updateEntry.branchCount++
					updateEntry.subtreeSize += pc.chunkLevel[lvl][index].subtreeSize
					copy(updateEntry.chunk[8+(index*pc.hashSize):8+((index+1)*pc.hashSize)], pc.chunkLevel[lvl][index].key[:pc.hashSize])
				}

				pc.enqueueTreeChunk(updateEntry, chunkWG, last)

			} else {

				noOfBranches := endCount - startCount
				newEntry := &TreeEntry{
					level:         int(lvl + 1),
					branchCount:   noOfBranches,
					subtreeSize:   0,
					chunk:         make([]byte, (noOfBranches*pc.hashSize)+8),
					key:           make([]byte, pc.hashSize),
					index:         int(nextLvlCount),
					updatePending: false,
				}

				index := int64(0)
				for i := startCount; i < endCount; i++ {
					entry := pc.chunkLevel[lvl][i]
					newEntry.subtreeSize += entry.subtreeSize
					copy(newEntry.chunk[8+(index*pc.hashSize):8+((index+1)*pc.hashSize)], entry.key[:pc.hashSize])
					index++
				}
				// Lonely chunk key is the key of the last chunk that is only one on the last branch.
				// In this case, ignore the its tree chunk key and replace it with the lonely chunk key.
				if lonelyChunkKey != nil {
					// Overwrite the last tree chunk key with the lonely data chunk key.
					copy(newEntry.chunk[int64(len(newEntry.chunk))-pc.hashSize:], lonelyChunkKey[:pc.hashSize])
				}

				pc.enqueueTreeChunk(newEntry, chunkWG, last)

			}

		}

		if !isAppend {
			chunkWG.Wait()
			if compress {
				pc.chunkLevel[lvl] = nil
			}
		}
	}

}

func (pc *PyramidChunker) enqueueTreeChunk(ent *TreeEntry, chunkWG *sync.WaitGroup, last bool) {
	if ent != nil && ent.branchCount > 0 {

		// wait for data chunks to get over before processing the tree chunk
		if last {
			chunkWG.Wait()
		}

		binary.LittleEndian.PutUint64(ent.chunk[:8], ent.subtreeSize)
		ent.key = make([]byte, pc.hashSize)
		chunkWG.Add(1)
		select {
		case pc.jobC <- &chunkJob{ent.key, ent.chunk[:ent.branchCount*pc.hashSize+8], chunkWG}:
		case <-pc.quitC:
		}

		// Update or append based on weather it is a new entry or being reused
		if ent.updatePending {
			chunkWG.Wait()
			pc.chunkLevel[ent.level][ent.index] = ent
		} else {
			pc.chunkLevel[ent.level] = append(pc.chunkLevel[ent.level], ent)
		}

	}
}

func (pc *PyramidChunker) enqueueDataChunk(chunkData []byte, size uint64, parent *TreeEntry, chunkWG *sync.WaitGroup) Address {
	binary.LittleEndian.PutUint64(chunkData[:8], size)
	pkey := parent.chunk[8+parent.branchCount*pc.hashSize : 8+(parent.branchCount+1)*pc.hashSize]

	chunkWG.Add(1)
	select {
	case pc.jobC <- &chunkJob{pkey, chunkData[:size+8], chunkWG}:
	case <-pc.quitC:
	}

	return pkey

}

// depth returns the number of chunk levels.
// It is used to detect if there is only one data chunk
// left for the last branch.
func (pc *PyramidChunker) depth() (d int) {
	for _, l := range pc.chunkLevel {
		if l == nil {
			return
		}
		d++
	}
	return
}

// cleanChunkLevels removes gaps (nil levels) between chunk levels
// that are not nil.
func (pc *PyramidChunker) cleanChunkLevels() {
	for i, l := range pc.chunkLevel {
		if l == nil {
			pc.chunkLevel = append(pc.chunkLevel[:i], append(pc.chunkLevel[i+1:], nil)...)
		}
	}
}
