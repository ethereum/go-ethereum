package rpc

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JekaMas/workerpool"
)

type SafePool struct {
	executionPool *atomic.Pointer[workerpool.WorkerPool]

	sync.RWMutex

	timeout time.Duration
	size    int

	// Skip sending task to execution pool
	fastPath bool
}

func NewExecutionPool(initialSize int, timeout time.Duration) *SafePool {
	sp := &SafePool{
		size:    initialSize,
		timeout: timeout,
	}

	if initialSize == 0 {
		sp.fastPath = true

		return sp
	}

	var ptr atomic.Pointer[workerpool.WorkerPool]

	p := workerpool.New(initialSize)
	ptr.Store(p)
	sp.executionPool = &ptr

	return sp
}

func (s *SafePool) Submit(ctx context.Context, fn func() error) (<-chan error, bool) {
	if s.fastPath {
		go func() {
			_ = fn()
		}()

		return nil, true
	}

	if s.executionPool == nil {
		return nil, false
	}

	pool := s.executionPool.Load()
	if pool == nil {
		return nil, false
	}

	return pool.Submit(ctx, fn, s.Timeout()), true
}

func (s *SafePool) ChangeSize(n int) {
	oldPool := s.executionPool.Swap(workerpool.New(n))

	if oldPool != nil {
		go func() {
			oldPool.StopWait()
		}()
	}

	s.Lock()
	s.size = n
	s.Unlock()
}

func (s *SafePool) ChangeTimeout(n time.Duration) {
	s.Lock()
	defer s.Unlock()

	s.timeout = n
}

func (s *SafePool) Timeout() time.Duration {
	s.RLock()
	defer s.RUnlock()

	return s.timeout
}

func (s *SafePool) Size() int {
	s.RLock()
	defer s.RUnlock()

	return s.size
}
