package storage

import (
	"io"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
	Children []common.Hash
	Last     bool
}

type Task struct {
	Index int64  // Index of the chunk being processed
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
	glog.V(logger.Detail).Infof("chunks: %v, depth: %v", chunks, depth)

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
		go self.processor(pend, tasks, &results)
	}
	// Feed the chunks into the task pool
	for index := 0; ; index++ {
		buffer := make([]byte, self.chunkSize+8)
		n, err := io.ReadFull(data, buffer)
		last := err == io.ErrUnexpectedEOF
		if err != nil && !last {
			glog.V(logger.Info).Infof("error: %v", err)

			close(abortC)
		}
		pend.Add(1)
		// glog.V(logger.Info).Infof("-> task %v (%v)", index, n)
		select {
		case tasks <- &Task{Index: int64(index), Data: buffer[:n+8], Last: last}:
		case <-abortC:
			return nil, err
		}
		if last {
			// glog.V(logger.Info).Infof("last task %v (%v)", index, n)
			break
		}
	}
	// Wait for the workers and return
	close(tasks)
	pend.Wait()

	// glog.V(logger.Info).Infof("len: %v", results.Levels[0][0])
	key := results.Levels[0][0].Children[0][:]
	return key, nil
}

func (self *PyramidChunker) processor(pend *sync.WaitGroup, tasks chan *Task, results *Tree) {
	defer pend.Done()

	// glog.V(logger.Info).Infof("processor started")
	// Start processing leaf chunks ad infinitum
	hasher := self.hashFunc()
	for task := range tasks {
		depth, pow := len(results.Levels)-1, self.branches
		// glog.V(logger.Info).Infof("task: %v, last: %v", task.Index, task.Last)

		var node *Node
		for depth >= 0 {
			// New chunk received, reset the hasher and start processing
			hasher.Reset()

			if node == nil { // Leaf node, hash the data chunk
				hasher.Write(task.Data)
			} else { // Internal node, hash the children
				for _, hash := range node.Children {
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
				node = &Node{pending, make([]common.Hash, pending), last}
				results.Levels[depth][task.Index/pow] = node
			}
			node.Pending--
			i := task.Index / (pow / self.branches) % self.branches
			if last {
				node.Pending -= self.branches - i
				node.Children = node.Children[:i+1]
				node.Last = true
			}
			copy(node.Children[i][:], hash)
			left := node.Pending

			if depth+1 < len(results.Levels) {
				delete(results.Levels[depth+1], task.Index/(pow/self.branches))
			}
			results.Lock.Unlock()
			// If there's more work to be done, leave for others
			// glog.V(logger.Info).Infof("left %v", left)
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
