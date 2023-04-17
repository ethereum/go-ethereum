// Copyright 2016 The go-ethereum Authors
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

package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFeedOf(t *testing.T) {
	var feed FeedOf[int]
	var done, subscribed sync.WaitGroup
	subscriber := func(i int) {
		defer done.Done()

		subchan := make(chan int)
		sub := feed.Subscribe(subchan)
		timeout := time.NewTimer(2 * time.Second)
		defer timeout.Stop()
		subscribed.Done()

		select {
		case v := <-subchan:
			if v != 1 {
				t.Errorf("%d: received value %d, want 1", i, v)
			}
		case <-timeout.C:
			t.Errorf("%d: receive timeout", i)
		}

		sub.Unsubscribe()
		select {
		case _, ok := <-sub.Err():
			if ok {
				t.Errorf("%d: error channel not closed after unsubscribe", i)
			}
		case <-timeout.C:
			t.Errorf("%d: unsubscribe timeout", i)
		}
	}

	const n = 1000
	done.Add(n)
	subscribed.Add(n)
	for i := 0; i < n; i++ {
		go subscriber(i)
	}
	subscribed.Wait()
	if nsent := feed.Send(1); nsent != n {
		t.Errorf("first send delivered %d times, want %d", nsent, n)
	}
	if nsent := feed.Send(2); nsent != 0 {
		t.Errorf("second send delivered %d times, want 0", nsent)
	}
	done.Wait()
}

func TestFeedOfSubscribeSameChannel(t *testing.T) {
	var (
		feed FeedOf[int]
		done sync.WaitGroup
		ch   = make(chan int)
		sub1 = feed.Subscribe(ch)
		sub2 = feed.Subscribe(ch)
		_    = feed.Subscribe(ch)
	)
	expectSends := func(value, n int) {
		if nsent := feed.Send(value); nsent != n {
			t.Errorf("send delivered %d times, want %d", nsent, n)
		}
		done.Done()
	}
	expectRecv := func(wantValue, n int) {
		for i := 0; i < n; i++ {
			if v := <-ch; v != wantValue {
				t.Errorf("received %d, want %d", v, wantValue)
			}
		}
	}

	done.Add(1)
	go expectSends(1, 3)
	expectRecv(1, 3)
	done.Wait()

	sub1.Unsubscribe()

	done.Add(1)
	go expectSends(2, 2)
	expectRecv(2, 2)
	done.Wait()

	sub2.Unsubscribe()

	done.Add(1)
	go expectSends(3, 1)
	expectRecv(3, 1)
	done.Wait()
}

func TestFeedOfSubscribeBlockedPost(t *testing.T) {
	var (
		feed   FeedOf[int]
		nsends = 2000
		ch1    = make(chan int)
		ch2    = make(chan int)
		wg     sync.WaitGroup
	)
	defer wg.Wait()

	feed.Subscribe(ch1)
	wg.Add(nsends)
	for i := 0; i < nsends; i++ {
		go func() {
			feed.Send(99)
			wg.Done()
		}()
	}

	sub2 := feed.Subscribe(ch2)
	defer sub2.Unsubscribe()

	// We're done when ch1 has received N times.
	// The number of receives on ch2 depends on scheduling.
	for i := 0; i < nsends; {
		select {
		case <-ch1:
			i++
		case <-ch2:
		}
	}
}

func TestFeedOfSubscribeBlockedCanceled(t *testing.T) {
	t.Parallel()
	var (
		feed   FeedOf[int]
		nsends = 2000
		ch1    = make(chan int)
		ch2    = make(chan int)
		wg     sync.WaitGroup
	)
	defer wg.Wait()
	wg.Add(nsends / 2)
	ctx, cancel := context.WithCancel(context.Background())
	feed.Subscribe(ch1)
	for i := 0; i < (nsends / 2); i++ {
		go func(i int) {
			feed.SendWithCtx(ctx, true, 99)
			wg.Done()
		}(i)
	}

	sub2 := feed.Subscribe(ch2)
	defer sub2.Unsubscribe()
	for i := 0; i < nsends; {
		select {
		case _, ok := <-ch1:
			i++
			if i == nsends/2 && ok {
				cancel()
			}
		case _, ok := <-ch2:
			if !ok {
				i++
			}
		}
	}
}

func TestFeedOfUnsubscribeBlockedPost(t *testing.T) {
	var (
		feed   FeedOf[int]
		nsends = 200
		chans  = make([]chan int, 2000)
		subs   = make([]Subscription, len(chans))
		bchan  = make(chan int)
		bsub   = feed.Subscribe(bchan)
		wg     sync.WaitGroup
	)
	for i := range chans {
		chans[i] = make(chan int, nsends)
	}

	// Queue up some Sends. None of these can make progress while bchan isn't read.
	wg.Add(nsends)
	for i := 0; i < nsends; i++ {
		go func() {
			feed.Send(99)
			wg.Done()
		}()
	}
	// Subscribe the other channels.
	for i, ch := range chans {
		subs[i] = feed.Subscribe(ch)
	}
	// Unsubscribe them again.
	for _, sub := range subs {
		sub.Unsubscribe()
	}
	// Unblock the Sends.
	bsub.Unsubscribe()
	wg.Wait()
}

// Checks that unsubscribing a channel during Send works even if that
// channel has already been sent on.
func TestFeedOfUnsubscribeSentChan(t *testing.T) {
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		ch2  = make(chan int)
		sub1 = feed.Subscribe(ch1)
		sub2 = feed.Subscribe(ch2)
		wg   sync.WaitGroup
	)
	defer sub2.Unsubscribe()

	wg.Add(1)
	go func() {
		feed.Send(0)
		wg.Done()
	}()

	// Wait for the value on ch1.
	<-ch1
	// Unsubscribe ch1, removing it from the send cases.
	sub1.Unsubscribe()

	// Receive ch2, finishing Send.
	<-ch2
	wg.Wait()

	// Send again. This should send to ch2 only, so the wait group will unblock
	// as soon as a value is received on ch2.
	wg.Add(1)
	go func() {
		feed.Send(0)
		wg.Done()
	}()
	<-ch2
	wg.Wait()
}

// Checks that unsubscribing a channel during Send works even if that
// channel has already been used with ctx.
func TestFeedOfUnsubscribeSentChanWithCtx(t *testing.T) {
	t.Parallel()
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		ch2  = make(chan int)
		sub1 = feed.Subscribe(ch1)
		sub2 = feed.Subscribe(ch2)
		wg   sync.WaitGroup
	)
	defer sub2.Unsubscribe()

	wg.Add(1)
	go func() {
		feed.SendWithCtx(context.Background(), false, 0)
		wg.Done()
	}()

	// Wait for the value on ch1.
	_, ok := <-ch1
	if !ok {
		t.Fatal("should not be dropped")
	}
	// Unsubscribe ch1, removing it from the send cases.
	sub1.Unsubscribe()

	// Receive ch2, finishing Send.
	_, ok = <-ch2
	if !ok {
		t.Fatal("should not be dropped")
	}
	wg.Wait()

	// Send again. This should send to ch2 only, so the wait group will unblock
	// as soon as a value is received on ch2.
	wg.Add(1)
	go func() {
		feed.Send(0)
		wg.Done()
	}()
	<-ch2
	wg.Wait()
}

// Checks that unsubscribing a channel during Send works even if that
// channel has already been sent on with background ctx.
func TestFeedOfDropSlowConsumer(t *testing.T) {
	t.Parallel()
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		ch2  = make(chan int)
		sub1 = feed.Subscribe(ch1)
		sub2 = feed.Subscribe(ch2)
		wg   sync.WaitGroup
	)
	defer sub2.Unsubscribe()
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		feed.SendWithCtx(ctx, true, 0)
		wg.Done()
	}()

	// Wait for the value on ch1.
	<-ch1
	// Unsubscribe ch1, removing it from the send cases.
	sub1.Unsubscribe()
	cancel() // drop ch2
	// make sure that ch2 is not active otherwise there is 50% chance to be selected instead.
	time.Sleep(1 * time.Second)
	_, ok := <-ch2
	if ok {
		t.Fatal("should be dropped")
	}
	wg.Wait()
}

// Checks that normal send is still possible after canceling previous ctx.
func TestFeedOfSendNormalAfterCancel(t *testing.T) {
	t.Parallel()
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		ch2  = make(chan int)
		sub1 = feed.Subscribe(ch1)
		sub2 = feed.Subscribe(ch2)
		wg   sync.WaitGroup
	)
	defer sub1.Unsubscribe()
	defer sub2.Unsubscribe()
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		feed.SendWithCtx(ctx, true, 0)
		wg.Done()
	}()

	<-ch1
	<-ch2
	cancel()
	wg.Wait()

	wg.Add(1)
	go func() {
		feed.Send(0)
		wg.Done()
	}()

	_, ok := <-ch1
	if !ok {
		t.Log("shouldn't be dropped")
	}
	_, ok = <-ch2
	if !ok {
		t.Log("shouldn't be dropped")
	}

	wg.Wait()
}

func TestFeedOfUnsubscribeFromInbox(t *testing.T) {
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		ch2  = make(chan int)
		sub1 = feed.Subscribe(ch1)
		sub2 = feed.Subscribe(ch1)
		sub3 = feed.Subscribe(ch2)
	)
	if len(feed.inbox) != 3 {
		t.Errorf("inbox length != 3 after subscribe")
	}
	if len(feed.sendCases) != 2 {
		t.Errorf("sendCases is non-empty after unsubscribe")
	}

	sub1.Unsubscribe()
	sub2.Unsubscribe()
	sub3.Unsubscribe()
	if len(feed.inbox) != 0 {
		t.Errorf("inbox is non-empty after unsubscribe")
	}
	if len(feed.sendCases) != 2 {
		t.Errorf("sendCases is non-empty after unsubscribe")
	}
}

func TestFeedOfUnsubscribeAfterDrop(t *testing.T) {
	t.Parallel()
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		sub1 = feed.Subscribe(ch1)
	)

	ctx, cancel := context.WithCancel(context.Background())
	go feed.SendWithCtx(ctx, true, 0)
	cancel()
	time.Sleep(1 * time.Second)
	sub1.Unsubscribe() //should not panic
	t.Log(sub1.Err())
	if len(feed.inbox) != 0 {
		t.Errorf("inbox is non-empty after unsubscribe")
	}
	if len(feed.sendCases) != 2 {
		t.Errorf("sendCases is non-empty after unsubscribe")
	}
}

func TestFeedOfDropAfterUnsubscribe(t *testing.T) {
	t.Parallel()
	var (
		feed FeedOf[int]
		ch1  = make(chan int)
		sub1 = feed.Subscribe(ch1)
	)

	ctx, cancel := context.WithCancel(context.Background())
	go feed.SendWithCtx(ctx, true, 0)
	time.Sleep(1 * time.Second)
	sub1.Unsubscribe()
	cancel() // shouldn't have anything to drop
}

func TestFeedOfNdropAmount(t *testing.T) {
	var (
		feed   FeedOf[int]
		nsends = 2000
		chans  = make([]chan int, nsends)
		subs   = make([]Subscription, len(chans))
		wg     sync.WaitGroup
	)
	for i := range chans {
		chans[i] = make(chan int)
	}

	// Subscribe the other channels.
	for i, ch := range chans {
		subs[i] = feed.Subscribe(ch)
	}

	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	var sent, dropped atomic.Uint32
	go func() {
		nsent, ndropped := feed.SendWithCtx(ctx, true, 0)
		sent.Store(uint32(nsent))
		dropped.Store(uint32(ndropped))
		wg.Done()
	}()

	for i := 0; i < nsends/2; i++ {
		ch := chans[i]
		_, ok := <-ch
		if !ok {
			t.Fatal("should not be dropped")
		}
	}
	cancel()
	wg.Wait()
	for i := nsends / 2; i < nsends; i++ {
		ch := chans[i]
		_, ok := <-ch
		if ok {
			t.Fatal("Should be dropped")
		}
	}
	if sent.Load() != uint32(nsends)/2 {
		t.Fatal("send amount mismatch")
	}

	if dropped.Load() != uint32(nsends)/2 {
		t.Fatal("drop amount mismatch")
	}
}

func BenchmarkFeedOfSend1000(b *testing.B) {
	var (
		done  sync.WaitGroup
		feed  FeedOf[int]
		nsubs = 1000
	)
	subscriber := func(ch <-chan int) {
		for i := 0; i < b.N; i++ {
			<-ch
		}
		done.Done()
	}
	done.Add(nsubs)
	for i := 0; i < nsubs; i++ {
		ch := make(chan int, 200)
		feed.Subscribe(ch)
		go subscriber(ch)
	}

	// The actual benchmark.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if feed.Send(i) != nsubs {
			panic("wrong number of sends")
		}
	}

	b.StopTimer()
	done.Wait()
}

func BenchmarkFeedOfSend1000Ctx(b *testing.B) {
	var (
		done  sync.WaitGroup
		feed  FeedOf[int]
		nsubs = 1000
	)
	subscriber := func(ch <-chan int) {
		for i := 0; i < b.N; i++ {
			<-ch
		}
		done.Done()
	}
	done.Add(nsubs)
	for i := 0; i < nsubs; i++ {
		ch := make(chan int, 200)
		feed.Subscribe(ch)
		go subscriber(ch)
	}

	ctx := context.Background()
	// The actual benchmark.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if sent, _ := feed.SendWithCtx(ctx, false, i); sent != nsubs {
			panic("wrong number of sends")
		}
	}

	b.StopTimer()
	done.Wait()
}
