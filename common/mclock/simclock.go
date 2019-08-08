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
	lastId    uint64
}

type event struct {
	do func()
	at AbsTime
	id uint64
}

// SimulatedEvent implements Event for a virtual clock.
type SimulatedEvent struct {
	at AbsTime
	id uint64
	s  *Simulated
}

// Run moves the clock by the given duration, executing all timers before that duration.
func (s *Simulated) Run(d time.Duration) {
	s.mu.Lock()
	s.init()

	end := s.now + AbsTime(d)
	var do []func()
	for len(s.scheduled) > 0 {
		ev := s.scheduled[0]
		if ev.at > end {
			break
		}
		s.now = ev.at
		do = append(do, ev.do)
		s.scheduled = s.scheduled[1:]
	}
	s.now = end
	s.mu.Unlock()

	for _, fn := range do {
		fn()
	}
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
	s.AfterFunc(d, func() {
		after <- (time.Time{}).Add(time.Duration(s.now))
	})
	return after
}

// AfterFunc implements Clock.
func (s *Simulated) AfterFunc(d time.Duration, do func()) Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.init()

	at := s.now + AbsTime(d)
	s.lastId++
	id := s.lastId
	l, h := 0, len(s.scheduled)
	ll := h
	for l != h {
		m := (l + h) / 2
		if (at < s.scheduled[m].at) || ((at == s.scheduled[m].at) && (id < s.scheduled[m].id)) {
			h = m
		} else {
			l = m + 1
		}
	}
	s.scheduled = append(s.scheduled, event{})
	copy(s.scheduled[l+1:], s.scheduled[l:ll])
	e := event{do: do, at: at, id: id}
	s.scheduled[l] = e
	s.cond.Broadcast()
	return &SimulatedEvent{at: at, id: id, s: s}
}

func (s *Simulated) init() {
	if s.cond == nil {
		s.cond = sync.NewCond(&s.mu)
	}
}

// Cancel implements Event.
func (e *SimulatedEvent) Cancel() bool {
	s := e.s
	s.mu.Lock()
	defer s.mu.Unlock()

	l, h := 0, len(s.scheduled)
	ll := h
	for l != h {
		m := (l + h) / 2
		if e.id == s.scheduled[m].id {
			l = m
			break
		}
		if (e.at < s.scheduled[m].at) || ((e.at == s.scheduled[m].at) && (e.id < s.scheduled[m].id)) {
			h = m
		} else {
			l = m + 1
		}
	}
	if l >= ll || s.scheduled[l].id != e.id {
		return false
	}
	copy(s.scheduled[l:ll-1], s.scheduled[l+1:])
	s.scheduled = s.scheduled[:ll-1]
	return true
}
