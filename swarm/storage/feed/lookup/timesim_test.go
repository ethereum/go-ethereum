package lookup_test

import (
	"sync"
	"sync/atomic"
	"time"
)

type Timer struct {
	deadline time.Time
	signal   chan time.Time
	id       int32
}

type Stopwatch struct {
	t            time.Time
	r            time.Duration
	timers       map[int32]*Timer
	timerCounter int32
	stopSignal   chan struct{}
	lock         sync.RWMutex
}

func NewStopwatch(resolution time.Duration) *Stopwatch {
	s := &Stopwatch{
		r: resolution,
	}
	s.Reset()
	return s
}

func (s *Stopwatch) Reset() {
	s.t = time.Time{}
	s.timers = make(map[int32]*Timer)
	s.Stop()
}

func (s *Stopwatch) Tick() {
	s.t = s.t.Add(s.r)
	s.lock.Lock()
	defer s.lock.Unlock()
	for id, timer := range s.timers {
		if s.t.After(timer.deadline) || s.t.Equal(timer.deadline) {
			timer.signal <- s.t
			close(timer.signal)
			delete(s.timers, id)
		}
	}
}

func (s *Stopwatch) GetTimer(duration time.Duration) <-chan time.Time {
	timer := &Timer{
		deadline: s.t.Add(duration),
		signal:   make(chan time.Time, 1),
		id:       atomic.AddInt32(&s.timerCounter, 1),
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	s.timers[timer.id] = timer
	return timer.signal
}

func (s *Stopwatch) TimeAfter() func(d time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		return s.GetTimer(d)
	}
}

func (s *Stopwatch) Elapsed() time.Duration {
	return s.t.Sub(time.Time{})
}

func (s *Stopwatch) Run() {
	go func() {
		stopSignal := make(chan struct{})
		s.stopSignal = stopSignal
		for {
			select {
			case <-time.After(1 * time.Millisecond):
				s.Tick()
			case <-stopSignal:
				return
			}
		}
	}()
}

func (s *Stopwatch) Stop() {
	if s.stopSignal != nil {
		close(s.stopSignal)
		s.stopSignal = nil
	}
}
