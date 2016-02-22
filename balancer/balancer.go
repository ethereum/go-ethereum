package balancer

import "container/heap"

const maxTaskBuffer = 3000 // maximum amount of work a worker can have in its buffer

// Task repsents a single batch of work offered to a worker.
type Task struct {
	fn func() error // work function
	c  chan error   // return channel
}

// NewTask returns a new task and sets the proper fields.
func NewTask(fn func() error, c chan error) Task {
	return Task{
		fn: fn,
		c:  c,
	}
}

// Worker is a worker that will take one it's assigned tasks
// and execute it
type Worker struct {
	id int // worker id

	// work attributes
	tasks   chan Task     // tasks to do (buffered)
	pending int           // count of pending work
	quit    chan struct{} // quit channel

	// heap attributes
	index int // index in the heap

	// temporary worker attributes
	temp  bool // is temporary worker
	start int  // start pending count
}

// work will take the oldest task and execute the function and
// yield the result back in to the return error channel.
func (w *Worker) work(done chan *Worker) {
	for {
		select {
		case task := <-w.tasks: // get task...
			task.c <- task.fn() // ...execute the task
			done <- w           // we're done
		case <-w.quit:
			return
		}
	}
}

// Pool is a pool of workers and implements containers.Heap
type Pool []*Worker

// Len returns teh length of the pool.
func (p Pool) Len() int { return len(p) }

// Less returns whether p[i} has less burden than p[j].
func (p Pool) Less(i, j int) bool { return p[i].pending < p[j].pending }

// Swap swaps p[i] with p[j] and swaps their internal indices.
func (p Pool) Swap(i, j int) {
	p[i].index = j // trade i<->j
	p[j].index = i // trade j<->i

	p[i], p[j] = p[j], p[i]
}

// Push pushes the worker x to the pool and sets the index.
func (p *Pool) Push(x interface{}) {
	w := x.(*Worker)  // cast the worker
	w.index = len(*p) // assign the new index

	*p = append(*p, x.(*Worker))
}

// Pop pops and returns the last item in p.
func (p *Pool) Pop() interface{} {
	old := *p
	n := len(old)
	x := old[n-1]
	*p = old[0 : n-1]
	return x
}

// Balancer is responsible for balancing any given tasks
// to the pool of workers. The workers are managed by the
// balancer and will try to make sure that the workers are
// equally balanced in "work to complete".
type Balancer struct {
	poolSize int // initial pool size, important for temp workers.
	pool     Pool
	done     chan *Worker
	work     chan Task
}

// New returns a new load balancer
func New(poolSize int) *Balancer {
	balancer := &Balancer{
		poolSize: poolSize,
		done:     make(chan *Worker, poolSize),
		work:     make(chan Task, poolSize*10),
		pool:     make(Pool, 0, poolSize),
	}
	heap.Init(&balancer.pool)

	// fill the pool with the given pool size
	for i := 0; i < poolSize; i++ {
		// create new worker
		worker := &Worker{id: i, tasks: make(chan Task, maxTaskBuffer)}
		// spawn worker process
		go worker.work(balancer.done)
		heap.Push(&balancer.pool, worker)
	}
	// spawn own balancer task
	go balancer.balance(balancer.work)

	return balancer
}

// Push pushes the given tasks in to the work channel.
func (b *Balancer) Push(work Task) {
	b.work <- work
}

// balance is the main thread of the balancer and handles the dispatching
// of workers and completion of them.
func (b *Balancer) balance(work chan Task) {
	for {
		select {
		case task := <-work: // get task
			b.dispatch(task) // dispatch the tasks
		case w := <-b.done: // worker is done
			b.completed(w) // handle worker
		}
	}
}

// dispatch dispatches the tasks to the least loaded worker.
func (b *Balancer) dispatch(task Task) {
	// Take least loaded worker
	w := heap.Pop(&b.pool).(*Worker)

	// If a worker is full the next write to the channel will block
	// we then instead create a new temporary worker that will take
	// over the task we originally wanted to push on the worker.
	// What this will do is create a new worker and set the state to be
	// "temporary". This means that when the "completed" handler is
	// called we check for temporaryness and discard it if it's empty.
	// Temporary workers never get precedence over non-temporary
	// workers by setting their start state to that equal of the
	// pool size - start pool size * max task buffer.
	// This may result in false positives with the falsity being that
	// len(w.tasks) is higher than the actual. This will then create
	// a temporary worker that may not have been necessary, this is
	// considered to be a reasonable tradeoff.
	if len(w.tasks) == cap(w.tasks) {
		heap.Push(&b.pool, w) // push full worker back to heap
		// set the pending state (i.e. high load)
		pending := (len(b.pool) - b.poolSize) * maxTaskBuffer
		// create the new temporary worker
		worker := &Worker{
			tasks:   make(chan Task, maxTaskBuffer),
			temp:    true,
			start:   pending,
			pending: pending,
		}
		go worker.work(b.done) // spawn worker process
		w = worker             // set new worker
	}

	// send it a task
	w.tasks <- task
	// add to its queue
	w.pending++
	// put it back in the heap
	heap.Push(&b.pool, w)
}

// completed handles the worker and puts it back in the pool
// based on it's load.
func (b *Balancer) completed(w *Worker) {
	// reduce one task
	w.pending--
	// remove it from the heap
	heap.Remove(&b.pool, w.index)

	// short circuit if temp worker depleted
	// results in temp worker being discarded
	if w.temp && w.pending == w.start {
		close(w.quit)
		return
	}
	// put it back in place
	heap.Push(&b.pool, w)
}
