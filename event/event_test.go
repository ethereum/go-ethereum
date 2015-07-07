// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package event

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

type testEvent int

func TestSub(t *testing.T) {
	mux := new(TypeMux)
	defer mux.Stop()

	sub := mux.Subscribe(testEvent(0))
	go func() {
		if err := mux.Post(testEvent(5)); err != nil {
			t.Errorf("Post returned unexpected error: %v", err)
		}
	}()
	ev := <-sub.Chan()

	if ev.(testEvent) != testEvent(5) {
		t.Errorf("Got %v (%T), expected event %v (%T)",
			ev, ev, testEvent(5), testEvent(5))
	}
}

func TestMuxErrorAfterStop(t *testing.T) {
	mux := new(TypeMux)
	mux.Stop()

	sub := mux.Subscribe(testEvent(0))
	if _, isopen := <-sub.Chan(); isopen {
		t.Errorf("subscription channel was not closed")
	}
	if err := mux.Post(testEvent(0)); err != ErrMuxClosed {
		t.Errorf("Post error mismatch, got: %s, expected: %s", err, ErrMuxClosed)
	}
}

func TestUnsubscribeUnblockPost(t *testing.T) {
	mux := new(TypeMux)
	defer mux.Stop()

	sub := mux.Subscribe(testEvent(0))
	unblocked := make(chan bool)
	go func() {
		mux.Post(testEvent(5))
		unblocked <- true
	}()

	select {
	case <-unblocked:
		t.Errorf("Post returned before Unsubscribe")
	default:
		sub.Unsubscribe()
		<-unblocked
	}
}

func TestSubscribeDuplicateType(t *testing.T) {
	mux := new(TypeMux)
	expected := "event: duplicate type event.testEvent in Subscribe"

	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("Subscribe didn't panic for duplicate type")
		} else if err != expected {
			t.Errorf("panic mismatch: got %#v, expected %#v", err, expected)
		}
	}()
	mux.Subscribe(testEvent(1), testEvent(2))
}

func TestMuxConcurrent(t *testing.T) {
	rand.Seed(time.Now().Unix())
	mux := new(TypeMux)
	defer mux.Stop()

	recv := make(chan int)
	poster := func() {
		for {
			err := mux.Post(testEvent(0))
			if err != nil {
				return
			}
		}
	}
	sub := func(i int) {
		time.Sleep(time.Duration(rand.Intn(99)) * time.Millisecond)
		sub := mux.Subscribe(testEvent(0))
		<-sub.Chan()
		sub.Unsubscribe()
		recv <- i
	}

	go poster()
	go poster()
	go poster()
	nsubs := 1000
	for i := 0; i < nsubs; i++ {
		go sub(i)
	}

	// wait until everyone has been served
	counts := make(map[int]int, nsubs)
	for i := 0; i < nsubs; i++ {
		counts[<-recv]++
	}
	for i, count := range counts {
		if count != 1 {
			t.Errorf("receiver %d called %d times, expected only 1 call", i, count)
		}
	}
}

func emptySubscriber(mux *TypeMux, types ...interface{}) {
	s := mux.Subscribe(testEvent(0))
	go func() {
		for _ = range s.Chan() {
		}
	}()
}

func BenchmarkPost3(b *testing.B) {
	var mux = new(TypeMux)
	defer mux.Stop()
	emptySubscriber(mux, testEvent(0))
	emptySubscriber(mux, testEvent(0))
	emptySubscriber(mux, testEvent(0))

	for i := 0; i < b.N; i++ {
		mux.Post(testEvent(0))
	}
}

func BenchmarkPostConcurrent(b *testing.B) {
	var mux = new(TypeMux)
	defer mux.Stop()
	emptySubscriber(mux, testEvent(0))
	emptySubscriber(mux, testEvent(0))
	emptySubscriber(mux, testEvent(0))

	var wg sync.WaitGroup
	poster := func() {
		for i := 0; i < b.N; i++ {
			mux.Post(testEvent(0))
		}
		wg.Done()
	}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go poster()
	}
	wg.Wait()
}

// for comparison
func BenchmarkChanSend(b *testing.B) {
	c := make(chan interface{})
	closed := make(chan struct{})
	go func() {
		for _ = range c {
		}
	}()

	for i := 0; i < b.N; i++ {
		select {
		case c <- i:
		case <-closed:
		}
	}
}
