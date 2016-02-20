package balancer

import "container/heap"

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
	id      int       // worker id
	tasks   chan Task // tasks to do (buffered)
	pending int       // count of pending work
	index   int       // index in the heap
}

// work will take the oldest task and execute the function and
// yield the result back in to the return error channel.
func (w *Worker) work(done chan *Worker) {
	for {
		task := <-w.tasks   // get task...
		task.c <- task.fn() // ...execute the task
		done <- w           // we're done
	}
}

// Pool is a pool of workers and implements containers.Heap
type Pool []*Worker

func (p Pool) Len() int           { return len(p) }
func (p Pool) Less(i, j int) bool { return p[i].pending < p[j].pending }
func (p Pool) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p *Pool) Push(x interface{}) {
	w := x.(*Worker)  // cast the worker
	w.index = len(*p) // assign the new index

	*p = append(*p, x.(*Worker))
}

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
	pool Pool
	done chan *Worker

	work chan Task
}

// New returns a new load balancer
func New(poolSize int) *Balancer {
	balancer := &Balancer{
		done: make(chan *Worker),
		pool: make(Pool, poolSize),
		work: make(chan Task),
	}

	operations := make(chan struct{}, poolSize)
	defer close(operations)

	// fill the pool with the given pool size
	for i := 0; i < poolSize; i++ {
		// create new worker
		balancer.pool[i] = &Worker{id: i, tasks: make(chan Task, 10)}
		// spawn worker process
		go func(i int) {
			operations <- struct{}{}
			balancer.pool[i].work(balancer.done)
		}(i)
	}
	// spawn own balancer task
	go balancer.balance(balancer.work)

	// wait for workers to be operations
	for i := 0; i < poolSize; i++ {
		<-operations
	}

	return balancer
}

// Push pushes the given tasks in to the work channel.
func (b *Balancer) Push(work Task) {
	go func() { b.work <- work }()
}

func (b *Balancer) balance(work chan Task) {
	go func() {
		// worker is done
		for w := range b.done {
			b.completed(w) // handle worker
		}
	}()
	go func() {
		// get task
		for task := range work {
			b.dispatch(task) // dispatch the tasks
		}
	}()
}

// dispatch dispatches the tasks to the least loaded worker.
func (b *Balancer) dispatch(task Task) {
	// Take least loaded worker
	w := heap.Pop(&b.pool).(*Worker)
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
	// put it back in place
	heap.Push(&b.pool, w)
}
