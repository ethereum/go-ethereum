// Package event implements an event multiplexer.
package event

import (
	"errors"
	"reflect"
	"sync"
)

type Subscription interface {
	Chan() <-chan interface{}
	Unsubscribe()
}

// A TypeMux dispatches events to registered receivers. Receivers can be
// registered to handle events of certain type. Any operation
// called after mux is stopped will return ErrMuxClosed.
type TypeMux struct {
	mutex   sync.RWMutex
	subm    map[reflect.Type][]*muxsub
	stopped bool
}

var ErrMuxClosed = errors.New("event: mux closed")

// NewTypeMux creates a running mux.
func NewTypeMux() *TypeMux {
	return &TypeMux{subm: make(map[reflect.Type][]*muxsub)}
}

// Subscribe creates a subscription for events of the given types. The
// subscription's channel is closed when it is unsubscribed
// or the mux is closed.
func (mux *TypeMux) Subscribe(types ...interface{}) Subscription {
	sub := newsub(mux)
	mux.mutex.Lock()
	if mux.stopped {
		mux.mutex.Unlock()
		close(sub.postC)
	} else {
		for _, t := range types {
			rtyp := reflect.TypeOf(t)
			oldsubs := mux.subm[rtyp]
			subs := make([]*muxsub, len(oldsubs)+1)
			copy(subs, oldsubs)
			subs[len(oldsubs)] = sub
			mux.subm[rtyp] = subs
		}
		mux.mutex.Unlock()
	}
	return sub
}

// Post sends an event to all receivers registered for the given type.
// It returns ErrMuxClosed if the mux has been stopped.
func (mux *TypeMux) Post(ev interface{}) error {
	rtyp := reflect.TypeOf(ev)
	mux.mutex.RLock()
	if mux.stopped {
		mux.mutex.RUnlock()
		return ErrMuxClosed
	}
	subs := mux.subm[rtyp]
	mux.mutex.RUnlock()
	for _, sub := range subs {
		sub.deliver(ev)
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
	mutex   sync.RWMutex
	closing chan struct{}

	// these two are the same channel. they are stored separately so
	// postC can be set to nil without affecting the return value of
	// Chan.
	readC <-chan interface{}
	postC chan<- interface{}
}

func newsub(mux *TypeMux) *muxsub {
	c := make(chan interface{})
	return &muxsub{
		mux:     mux,
		readC:   c,
		postC:   c,
		closing: make(chan struct{}),
	}
}

func (s *muxsub) Chan() <-chan interface{} {
	return s.readC
}

func (s *muxsub) Unsubscribe() {
	s.mux.del(s)
	s.closewait()
}

func (s *muxsub) closewait() {
	close(s.closing)
	s.mutex.Lock()
	close(s.postC)
	s.postC = nil
	s.mutex.Unlock()
}

func (s *muxsub) deliver(ev interface{}) {
	s.mutex.RLock()
	select {
	case s.postC <- ev:
	case <-s.closing:
	}
	s.mutex.RUnlock()
}
