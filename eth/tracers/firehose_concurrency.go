package tracers

import (
	pbeth "github.com/streamingfast/firehose-ethereum/types/pb/sf/ethereum/type/v2"
	"sync"
)

type blockPrintJob struct {
	block    *pbeth.Block
	finality *FinalityStatus
}

type outputJob struct {
	blockNum uint64
	data     []byte
}

type ConcurrentFlushQueue struct {
	bufferSize int

	startSignal    chan uint64
	jobQueue       chan *blockPrintJob
	outputQueue    chan *outputJob
	printBlockFunc func(block *pbeth.Block, finality *FinalityStatus)
	outputFunc     func([]byte)

	jobWG     sync.WaitGroup
	outputWG  sync.WaitGroup
	closeOnce sync.Once
}

func NewConcurrentFlushQueue(bufferSize int, printBlockFunc func(*pbeth.Block, *FinalityStatus), outputFunc func([]byte)) *ConcurrentFlushQueue {
	return &ConcurrentFlushQueue{
		startSignal:    make(chan uint64, 1),
		jobQueue:       make(chan *blockPrintJob, bufferSize),
		outputQueue:    make(chan *outputJob, bufferSize),
		outputFunc:     outputFunc,
		bufferSize:     bufferSize,
		printBlockFunc: printBlockFunc,
	}
}

func (q *ConcurrentFlushQueue) Start(concurrency int) {
	for i := 0; i < concurrency; i++ {
		q.jobWG.Add(1)
		go q.worker()
	}

	q.outputWG.Add(1)
	go q.outputOrderer()
}

func (q *ConcurrentFlushQueue) Enqueue(block *pbeth.Block, finality *FinalityStatus) {
	select {
	case q.startSignal <- block.Number:
	default:
	}

	q.jobQueue <- &blockPrintJob{
		block:    block,
		finality: finality,
	}
}

// CloseChannels signals goroutines to shut down and waits for them.
// It blocks until all concurrent block flushing operations are completed, ensuring a clean
// shutdown of the printing pipeline.
func (q *ConcurrentFlushQueue) CloseChannels() {
	q.closeOnce.Do(func() {
		close(q.jobQueue)
		q.jobWG.Wait()
		close(q.outputQueue)
		q.outputWG.Wait()
	})
}

// Instantiates a worker that listens for jobs
func (q *ConcurrentFlushQueue) worker() {
	defer q.jobWG.Done()
	for job := range q.jobQueue {
		q.printBlockFunc(job.block, job.finality)
	}
}

// Channel ensuring that blocks are linearly flushed out in order
func (q *ConcurrentFlushQueue) outputOrderer() {
	defer q.outputWG.Done()
	buffer := make(map[uint64][]byte)
	next := <-q.startSignal

	for job := range q.outputQueue {
		buffer[job.blockNum] = job.data
		for {
			data, ok := buffer[next]
			if !ok {
				break
			}
			q.outputFunc(data)
			delete(buffer, next)
			next++
		}
	}
}
