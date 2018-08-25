// Copyright 2018 The go-ethereum Authors
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

package mclock

import (
	"sync"
	"time"
)

// Simulated implements a virtual Clock for reproducible time-sensitive tests. It
// simulates a scheduler on a virtual timescale where actual processing takes zero time.
//
// The virtual clock doesn't advance on its own, call Run to advance it and execute timers.
// Since there is no way to influence the Go scheduler, testing timeout behaviour involving
// goroutines needs special care. A good way to test such timeouts is as follows: First
// perform the action that is supposed to time out. Ensure that the timer you want to test
// is created. Then run the clock until after the timeout. Finally observe the effect of
// the timeout using a channel or semaphore.
type Simulated struct {
	now       AbsTime
	scheduled []event
	mu        sync.RWMutex
	cond      *sync.Cond
}

type event struct {
	do func()
	at AbsTime
}

// Run moves the clock by the given duration, executing all timers before that duration.
func (s *Simulated) Run(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()

	end := s.now + AbsTime(d)
	for len(s.scheduled) > 0 {
		ev := s.scheduled[0]
		if ev.at > end {
			break
		}
		s.now = ev.at
		ev.do()
		s.scheduled = s.scheduled[1:]
	}
	s.now = end
}

func (s *Simulated) ActiveTimers() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.scheduled)
}

func (s *Simulated) WaitForTimers(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()

	for len(s.scheduled) < n {
		s.cond.Wait()
	}
}

// Now implements Clock.
func (s *Simulated) Now() AbsTime {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.now
}

// Sleep implements Clock.
func (s *Simulated) Sleep(d time.Duration) {
	<-s.After(d)
}

// After implements Clock.
func (s *Simulated) After(d time.Duration) <-chan time.Time {
	after := make(chan time.Time, 1)
	s.insert(d, func() {
		after <- (time.Time{}).Add(time.Duration(s.now))
	})
	return after
}

func (s *Simulated) insert(d time.Duration, do func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()

	at := s.now + AbsTime(d)
	l, h := 0, len(s.scheduled)
	ll := h
	for l != h {
		m := (l + h) / 2
		if at < s.scheduled[m].at {
			h = m
		} else {
			l = m + 1
		}
	}
	s.scheduled = append(s.scheduled, event{})
	copy(s.scheduled[l+1:], s.scheduled[l:ll])
	s.scheduled[l] = event{do: do, at: at}
	s.cond.Broadcast()
}

func (s *Simulated) init() {
	if s.cond == nil {
		s.cond = sync.NewCond(&s.mu)
	}
}
