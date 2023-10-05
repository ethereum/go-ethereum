package rpc

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JekaMas/workerpool"
	"github.com/ethereum/go-ethereum/metrics"
)

type SafePool struct {
	executionPool atomic.Pointer[workerpool.WorkerPool]

	sync.RWMutex

	size    int
	timeout time.Duration

	service   string       // the service using ep
	processed atomic.Int64 // keeps count of total processed requests

	close     chan struct{}
	closeOnce sync.Once

	// Skip sending task to execution pool
	fastPath bool
}

func NewExecutionPool(initialSize int, timeout time.Duration, service string, report bool) *SafePool {
	sp := &SafePool{
		size:    initialSize,
		timeout: timeout,
		service: service,
		close:   make(chan struct{}),
	}

	if initialSize == 0 {
		sp.fastPath = true

		return sp
	}

	sp.executionPool.Store(workerpool.New(initialSize))

	if metrics.Enabled && report {
		go sp.reportMetrics(3 * time.Second)
	}

	return sp
}

func (s *SafePool) Submit(ctx context.Context, fn func() error) (<-chan error, bool) {
	if s.fastPath {
		go func() {
			_ = fn()
		}()

		return nil, true
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

func (s *SafePool) Stop() {
	s.closeOnce.Do(func() {
		close(s.close)
	})

	if s.executionPool.Load() != nil {
		s.executionPool.Load().Stop()
	}
}

// reportMetrics reports the metrics after every `refresh` time interval
// regarding the execution pool.
func (s *SafePool) reportMetrics(refresh time.Duration) {
	var (
		epWorkerCountGuage           metrics.Gauge
		epWaitingQueueGuage          metrics.Gauge
		epProcessedRequestsHistogram metrics.Histogram
	)

	ticker := time.NewTicker(refresh)

	for {
		select {
		case <-ticker.C:
			ep := s.executionPool.Load()

			epWorkerCountGuage, epWaitingQueueGuage, epProcessedRequestsHistogram = newEpMetrics(s.service)

			epWorkerCountGuage.Update(ep.GetWorkerCount())
			epWaitingQueueGuage.Update(int64(ep.WaitingQueueSize()))
			epProcessedRequestsHistogram.Update(s.processed.Load())

			s.processed.Store(0)
		case <-s.close:
			ticker.Stop()

			return
		}
	}
}
