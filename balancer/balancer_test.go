package balancer

import (
	"container/heap"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestTemporaryWorker(t *testing.T) {
	size := 5
	maxTaskBuffer = 1
	balancer := New(4)

	qchan := make([]chan struct{}, size)
	errch := make(chan error, size)

	for i := 0; i < size; i++ {
		i := i

		qchan[i] = make(chan struct{})
		balancer.dispatch(NewTask(func() error {
			<-qchan[i] // blocks
			return nil
		}, errch))
	}

	if len(balancer.pool) != balancer.poolSize+1 {
		t.Fatal("expected pool to be size", balancer.poolSize+1, "got", len(balancer.pool))
	}

	lastWorker := balancer.pool[len(balancer.pool)-1]
	if !lastWorker.temp {
		t.Error("expected last worker to be temp worker")
	}

	if lastWorker.start != (len(balancer.pool)-balancer.poolSize)*maxTaskBuffer {
		t.Error("expected temp worker start", balancer.poolSize, "got", lastWorker.start)
	}

	// clean up
	for i := 0; i < cap(qchan); i++ {
		close(qchan[i])
	}
}

func TestLoad(t *testing.T) {
	maxTaskBuffer = 2
	size := 5
	balancer := New(4)

	qchan := make([]chan struct{}, size)
	errch := make(chan error, size)

	for i := 0; i < size; i++ {
		i := i

		qchan[i] = make(chan struct{})
		balancer.dispatch(NewTask(func() error {
			<-qchan[i] // blocks
			return nil
		}, errch))
	}

	var foundTwo bool
	for _, worker := range balancer.pool {
		if worker.pending == 2 {
			foundTwo = true
		}
	}
	if !foundTwo {
		t.Error("expected to have at least one item with pending 2")
	}

	// clean up
	for i := 0; i < cap(qchan); i++ {
		close(qchan[i])
	}
}

func TestPool(t *testing.T) {
	pool := make(Pool, 0)
	heap.Init(&pool)

	var workers = []*Worker{
		&Worker{id: 0, pending: 0},
		&Worker{id: 1, pending: 1},
	}

	for _, worker := range workers {
		heap.Push(&pool, worker)
	}

	worker := heap.Pop(&pool).(*Worker)
	if worker.id != 0 {
		t.Error("expected worker 0 to be popped")
	}

	worker.pending = 2
	heap.Push(&pool, worker)

	if pool[worker.index] != worker {
		t.Error("expected", worker.index, "to be at index but got different worker")
	}

	worker = heap.Pop(&pool).(*Worker)
	if worker.id != 1 {
		t.Error("expected worker 1 to be popped")
	}
	heap.Push(&pool, worker)

	if pool[worker.index] != worker {
		t.Error("expected", worker.index, "to be at index but got different worker")
	}
}

func makeTxs(size int, b *testing.B) []types.Transactions {
	key, _ := crypto.GenerateKey()

	batches := make([]types.Transactions, b.N)
	for i := 0; i < b.N; i++ {
		txs := make(types.Transactions, size)
		for j := range txs {
			var err error
			txs[j], err = types.NewTransaction(0, common.Address{}, new(big.Int), new(big.Int), new(big.Int), nil).SignECDSA(key)
			if err != nil {
				b.Fatal(err)
			}
		}
		batches[i] = txs
	}
	return batches
}

const benchTxSize = 1024

func BenchmarkTxsRaw(b *testing.B) {
	batches := makeTxs(benchTxSize, b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txs := batches[i]
		b.StartTimer()

		for _, tx := range txs {
			if _, err := tx.From(); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkTxsLoadBalancer(b *testing.B) {
	balancer := New(4)
	batches := makeTxs(benchTxSize, b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		txs := batches[i]

		var (
			size      = 8
			batchsize = len(txs) / size
			ch        = make(chan error, size)
		)

		for j := 0; j < size; j++ {
			j := j
			task := Task{
				fn: func() error {
					for _, tx := range txs[j*size : j*size+batchsize] {
						if _, err := tx.From(); err != nil {
							return err
						}
					}
					return nil
				},
				c: ch,
			}

			balancer.Push(task)
		}

		for j := 0; j < size; j++ {
			if err := <-ch; err != nil {
				b.Error(err)
			}
		}
		close(ch)
	}
}
