// Copyright 2014 The go-ethereum Authors
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

// Package event implements an event multiplexer.
package event

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Event is a time-tagged notification pushed to subscribers.
type Event struct {
	Time time.Time
	Data interface{}
}

// Subscription is implemented by event subscriptions.
type Subscription interface {
	// Chan returns a channel that carries events.
	// Implementations should return the same channel
	// for any subsequent calls to Chan.
	Chan() <-chan *Event

	// Unsubscribe stops delivery of events to a subscription.
	// The event channel is closed.
	// Unsubscribe can be called more than once.
	Unsubscribe()
}

// A TypeMux dispatches events to registered receivers. Receivers can be
// registered to handle events of certain type. Any operation
// called after mux is stopped will return ErrMuxClosed.
//
// The zero value is ready to use.
type TypeMux struct {
	mutex   sync.RWMutex
	subm    map[reflect.Type][]*muxsub
	stopped bool
}

// ErrMuxClosed is returned when Posting on a closed TypeMux.
var ErrMuxClosed = errors.New("event: mux closed")

// Subscribe creates a subscription for events of the given types. The
// subscription's channel is closed when it is unsubscribed
// or the mux is closed.
func (mux *TypeMux) Subscribe(types ...interface{}) Subscription {
	sub := newsub(mux)
	mux.mutex.Lock()
	defer mux.mutex.Unlock()
	if mux.stopped {
		// set the status to closed so that calling Unsubscribe after this
		// call will short curuit
		sub.closed = true
		close(sub.postC)
	} else {
		if mux.subm == nil {
			mux.subm = make(map[reflect.Type][]*muxsub)
		}
		for _, t := range types {
			rtyp := reflect.TypeOf(t)
			oldsubs := mux.subm[rtyp]
			if find(oldsubs, sub) != -1 {
				panic(fmt.Sprintf("event: duplicate type %s in Subscribe", rtyp))
			}
			subs := make([]*muxsub, len(oldsubs)+1)
			copy(subs, oldsubs)
			subs[len(oldsubs)] = sub
			mux.subm[rtyp] = subs
		}
	}
	return sub
}

// Post sends an event to all receivers registered for the given type.
// It returns ErrMuxClosed if the mux has been stopped.
func (mux *TypeMux) Post(ev interface{}) error {
	event := &Event{
		Time: time.Now(),
		Data: ev,
	}
	rtyp := reflect.TypeOf(ev)
	mux.mutex.RLock()
	if mux.stopped {
		mux.mutex.RUnlock()
		return ErrMuxClosed
	}
	subs := mux.subm[rtyp]
	mux.mutex.RUnlock()
	for _, sub := range subs {
		sub.deliver(event)
	}
	return nil
}

// Stop closes a mux. The mux can no longer be used.
// Future Post calls will fail with ErrMuxClosed.
// Stop blocks until all current deliveries have finished.
func (mux *TypeMux) Stop() {
	mux.mutex.Lock()
	for _, subs := range mux.subm {
		for _, sub := range subs {
			sub.closewait()
		}
	}
	mux.subm = nil
	mux.stopped = true
	mux.mutex.Unlock()
}

func (mux *TypeMux) del(s *muxsub) {
	mux.mutex.Lock()
	for typ, subs := range mux.subm {
		if pos := find(subs, s); pos >= 0 {
			if len(subs) == 1 {
				delete(mux.subm, typ)
			} else {
				mux.subm[typ] = posdelete(subs, pos)
			}
		}
	}
	s.mux.mutex.Unlock()
}

func find(slice []*muxsub, item *muxsub) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func posdelete(slice []*muxsub, pos int) []*muxsub {
	news := make([]*muxsub, len(slice)-1)
	copy(news[:pos], slice[:pos])
	copy(news[pos:], slice[pos+1:])
	return news
}

type muxsub struct {
	mux     *TypeMux
	created time.Time
	closeMu sync.Mutex
	closing chan struct{}
	closed  bool

	// these two are the same channel. they are stored separately so
	// postC can be set to nil without affecting the return value of
	// Chan.
	postMu sync.RWMutex
	readC  <-chan *Event
	postC  chan<- *Event
}

func newsub(mux *TypeMux) *muxsub {
	c := make(chan *Event)
	return &muxsub{
		mux:     mux,
		created: time.Now(),
		readC:   c,
		postC:   c,
		closing: make(chan struct{}),
	}
}

func (s *muxsub) Chan() <-chan *Event {
	return s.readC
}

func (s *muxsub) Unsubscribe() {
	s.mux.del(s)
	s.closewait()
}

func (s *muxsub) closewait() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return
	}
	close(s.closing)
	s.closed = true

	s.postMu.Lock()
	close(s.postC)
	s.postC = nil
	s.postMu.Unlock()
}

func (s *muxsub) deliver(event *Event) {
	// Short circuit delivery if stale event
	if s.created.After(event.Time) {
		return
	}
	// Otherwise deliver the event
	s.postMu.RLock()
	defer s.postMu.RUnlock()

	select {
	case s.postC <- event:
	case <-s.closing:
	}
}
