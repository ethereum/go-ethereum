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
	"fmt"
	"io"
	"math"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

const (
	processors = 8
)

type Tree struct {
	Chunks int64
	Levels []map[int64]*Node
	Lock   sync.RWMutex
}

type Node struct {
	Pending  int64
	Size     uint64
	Children []common.Hash
	Last     bool
}

func (self *Node) String() string {
	var children []string
	for _, node := range self.Children {
		children = append(children, node.Hex())
	}
	return fmt.Sprintf("pending: %v, size: %v, last :%v, children: %v", self.Pending, self.Size, self.Last, strings.Join(children, ", "))
}

type Task struct {
	Index int64 // Index of the chunk being processed
	Size  uint64
	Data  []byte // Binary blob of the chunk
	Last  bool
}

type PyramidChunker struct {
	hashFunc    Hasher
	chunkSize   int64
	hashSize    int64
	branches    int64
	workerCount int
}

func NewPyramidChunker(params *ChunkerParams) (self *PyramidChunker) {
	self = &PyramidChunker{}
	self.hashFunc = MakeHashFunc(params.Hash)
	self.branches = params.Branches
	self.hashSize = int64(self.hashFunc().Size())
	self.chunkSize = self.hashSize * self.branches
	self.workerCount = 1
	return
}

func (self *PyramidChunker) Split(data io.Reader, size int64, chunkC chan *Chunk, swg, wwg *sync.WaitGroup) (Key, error) {

	chunks := (size + self.chunkSize - 1) / self.chunkSize
	depth := int(math.Ceil(math.Log(float64(chunks))/math.Log(float64(self.branches)))) + 1

	results := Tree{
		Chunks: chunks,
		Levels: make([]map[int64]*Node, depth),
	}
	for i := 0; i < depth; i++ {
		results.Levels[i] = make(map[int64]*Node)
	}
	// Create a pool of workers to crunch through the file
	tasks := make(chan *Task, 2*processors)
	pend := new(sync.WaitGroup)
	abortC := make(chan bool)
	for i := 0; i < processors; i++ {
		pend.Add(1)
		go self.processor(pend, swg, tasks, chunkC, &results)
	}
	// Feed the chunks into the task pool
	read := 0
	for index := 0; ; index++ {
		buffer := make([]byte, self.chunkSize+8)
		n, err := data.Read(buffer[8:])
		read += n
		last := int64(read) == size || err == io.ErrUnexpectedEOF || err == io.EOF
		if err != nil && !last {
			close(abortC)
			break
		}
		binary.LittleEndian.PutUint64(buffer[:8], uint64(n))
		pend.Add(1)
		select {
		case tasks <- &Task{Index: int64(index), Size: uint64(n), Data: buffer[:n+8], Last: last}:
		case <-abortC:
			return nil, err
		}
		if last {
			break
		}
	}
	// Wait for the workers and return
	close(tasks)
	pend.Wait()

	key := results.Levels[0][0].Children[0][:]
	return key, nil
}

func (self *PyramidChunker) processor(pend, swg *sync.WaitGroup, tasks chan *Task, chunkC chan *Chunk, results *Tree) {
	defer pend.Done()

	// Start processing leaf chunks ad infinitum
	hasher := self.hashFunc()
	for task := range tasks {
		depth, pow := len(results.Levels)-1, self.branches
		size := task.Size
		data := task.Data
		var node *Node
		for depth >= 0 {
			// New chunk received, reset the hasher and start processing
			hasher.Reset()
			if node == nil { // Leaf node, hash the data chunk
				hasher.Write(task.Data)
			} else { // Internal node, hash the children
				size = node.Size
				data = make([]byte, hasher.Size()*len(node.Children)+8)
				binary.LittleEndian.PutUint64(data[:8], size)

				hasher.Write(data[:8])
				for i, hash := range node.Children {
					copy(data[i*hasher.Size()+8:], hash[:])
					hasher.Write(hash[:])
				}
			}
			hash := hasher.Sum(nil)
			last := task.Last || (node != nil) && node.Last
			// Insert the subresult into the memoization tree
			results.Lock.Lock()
			if node = results.Levels[depth][task.Index/pow]; node == nil {
				// Figure out the pending tasks
				pending := self.branches
				if task.Index/pow == results.Chunks/pow {
					pending = (results.Chunks + pow/self.branches - 1) / (pow / self.branches) % self.branches
				}
				node = &Node{pending, 0, make([]common.Hash, pending), last}
				results.Levels[depth][task.Index/pow] = node
			}
			node.Pending--
			i := task.Index / (pow / self.branches) % self.branches
			if last {
				node.Last = true
			}
			copy(node.Children[i][:], hash)
			node.Size += size
			left := node.Pending
			if chunkC != nil {
				if swg != nil {
					swg.Add(1)
				}
				select {
				case chunkC <- &Chunk{Key: hash, SData: data, wg: swg}:
					// case <- self.quitC
				}
			}
			if depth+1 < len(results.Levels) {
				delete(results.Levels[depth+1], task.Index/(pow/self.branches))
			}

			results.Lock.Unlock()
			// If there's more work to be done, leave for others
			if left > 0 {
				break
			}
			// We're the last ones in this batch, merge the children together
			depth--
			pow *= self.branches
		}
		pend.Done()
	}
}
