package lookup_test

// This file contains simple time simulation tools for testing
// and measuring time-aware algorithms

import (
	"sync"
	"time"
)

// Timer tracks information about a simulated timer
type Timer struct {
	deadline time.Time
	signal   chan time.Time
	id       int
}

// Stopwatch measures simulated execution time and manages simulated timers
type Stopwatch struct {
	t            time.Time
	resolution   time.Duration
	timers       map[int]*Timer
	timerCounter int
	stopSignal   chan struct{}
	lock         sync.RWMutex
}

// NewStopwatch returns a simulated clock that ticks on `resolution` intervals
func NewStopwatch(resolution time.Duration) *Stopwatch {
	s := &Stopwatch{
		resolution: resolution,
	}
	s.Reset()
	return s
}

// Reset clears all timers and sents the stopwatch to zero
func (s *Stopwatch) Reset() {
	s.t = time.Time{}
	s.timers = make(map[int]*Timer)
	s.Stop()
}

// Tick advances simulated time by the stopwatch's resolution and triggers
// all due timers
func (s *Stopwatch) Tick() {
	s.t = s.t.Add(s.resolution)

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

// NewTimer returns a new timer that will trigger after `duration` elapses in the
// simulation
func (s *Stopwatch) NewTimer(duration time.Duration) <-chan time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.timerCounter++
	timer := &Timer{
		deadline: s.t.Add(duration),
		signal:   make(chan time.Time, 1),
		id:       s.timerCounter,
	}

	s.timers[timer.id] = timer
	return timer.signal
}

// TimeAfter returns a simulated timer factory that can replace `time.After`
func (s *Stopwatch) TimeAfter() func(d time.Duration) <-chan time.Time {
	return func(d time.Duration) <-chan time.Time {
		return s.NewTimer(d)
	}
}

// Elapsed returns the time that has passed in the simulation
func (s *Stopwatch) Elapsed() time.Duration {
	return s.t.Sub(time.Time{})
}

// Run starts the time simulation
func (s *Stopwatch) Run() {
	go func() {
		stopSignal := make(chan struct{})
		s.lock.Lock()
		if s.stopSignal != nil {
			close(s.stopSignal)
		}
		s.stopSignal = stopSignal
		s.lock.Unlock()
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

// Stop stops the time simulation
func (s *Stopwatch) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.stopSignal != nil {
		close(s.stopSignal)
		s.stopSignal = nil
	}
}

func (s *Stopwatch) Measure(measuredFunc func()) time.Duration {
	s.Reset()
	s.Run()
	defer s.Stop()
	measuredFunc()
	return s.Elapsed()
}
