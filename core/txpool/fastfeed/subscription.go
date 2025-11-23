// Copyright 2024 The go-ethereum Authors
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

package fastfeed

import (
	"errors"
	"sync"
)

var (
	ErrSubscriptionClosed = errors.New("subscription closed")
)

// Subscription represents a subscription to the transaction fast feed.
type Subscription struct {
	id     int
	feed   *TxFastFeed
	events chan *TxEvent
	quit   chan struct{}
	once   sync.Once
}

// Events returns the channel that delivers transaction events.
func (s *Subscription) Events() <-chan *TxEvent {
	return s.events
}

// Unsubscribe unsubscribes from the feed and releases resources.
func (s *Subscription) Unsubscribe() {
	s.once.Do(func() {
		close(s.quit)
		close(s.events)
		
		// Remove filter
		s.feed.mu.Lock()
		delete(s.feed.filters, s.id)
		s.feed.mu.Unlock()
	})
}

// deliver reads events from the ring buffer and delivers them to the subscription channel.
func (s *Subscription) deliver() {
	for {
		select {
		case <-s.quit:
			return
		default:
		}
		
		// Try to read from ring buffer
		eventPtr := s.feed.ring.Read(s.id)
		if eventPtr == nil {
			// No data available, yield
			continue
		}
		
		// Convert pointer to event
		event := (*TxEvent)(eventPtr)
		
		// Apply filter if set
		s.feed.mu.RLock()
		filter, hasFilter := s.feed.filters[s.id]
		s.feed.mu.RUnlock()
		
		if hasFilter && !filter.Matches(event) {
			continue
		}
		
		// Copy event to avoid data races
		eventCopy := *event
		
		// Try to deliver to channel
		select {
		case s.events <- &eventCopy:
		case <-s.quit:
			return
		default:
			// Channel full, skip this event
			// In production, might want to track skipped events
		}
	}
}

