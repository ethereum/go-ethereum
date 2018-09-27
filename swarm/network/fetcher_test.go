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

package network

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

var requestedPeerID = enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8")
var sourcePeerID = enode.HexID("99d8594b52298567d2ca3f4c441a5ba0140ee9245e26460d01102a52773c73b9")

// mockRequester pushes every request to the requestC channel when its doRequest function is called
type mockRequester struct {
	// requests []Request
	requestC  chan *Request   // when a request is coming it is pushed to requestC
	waitTimes []time.Duration // with waitTimes[i] you can define how much to wait on the ith request (optional)
	count     int             //counts the number of requests
	quitC     chan struct{}
}

func newMockRequester(waitTimes ...time.Duration) *mockRequester {
	return &mockRequester{
		requestC:  make(chan *Request),
		waitTimes: waitTimes,
		quitC:     make(chan struct{}),
	}
}

func (m *mockRequester) doRequest(ctx context.Context, request *Request) (*enode.ID, chan struct{}, error) {
	waitTime := time.Duration(0)
	if m.count < len(m.waitTimes) {
		waitTime = m.waitTimes[m.count]
		m.count++
	}
	time.Sleep(waitTime)
	m.requestC <- request

	// if there is a Source in the request use that, if not use the global requestedPeerId
	source := request.Source
	if source == nil {
		source = &requestedPeerID
	}
	return source, m.quitC, nil
}

// TestFetcherSingleRequest creates a Fetcher using mockRequester, and run it with a sample set of peers to skip.
// mockRequester pushes a Request on a channel every time the request function is called. Using
// this channel we test if calling Fetcher.Request calls the request function, and whether it uses
// the correct peers to skip which we provided for the fetcher.run function.
func TestFetcherSingleRequest(t *testing.T) {
	requester := newMockRequester()
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peers := []string{"a", "b", "c", "d"}
	peersToSkip := &sync.Map{}
	for _, p := range peers {
		peersToSkip.Store(p, time.Now())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fetcher.run(ctx, peersToSkip)

	rctx := context.Background()
	fetcher.Request(rctx, 0)

	select {
	case request := <-requester.requestC:
		// request should contain all peers from peersToSkip provided to the fetcher
		for _, p := range peers {
			if _, ok := request.peersToSkip.Load(p); !ok {
				t.Fatalf("request.peersToSkip misses peer")
			}
		}

		// source peer should be also added to peersToSkip eventually
		time.Sleep(100 * time.Millisecond)
		if _, ok := request.peersToSkip.Load(requestedPeerID.String()); !ok {
			t.Fatalf("request.peersToSkip does not contain peer returned by the request function")
		}

		// hopCount in the forwarded request should be incremented
		if request.HopCount != 1 {
			t.Fatalf("Expected request.HopCount 1 got %v", request.HopCount)
		}

		// fetch should trigger a request, if it doesn't happen in time, test should fail
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("fetch timeout")
	}
}

// TestCancelStopsFetcher tests that a cancelled fetcher does not initiate further requests even if its fetch function is called
func TestFetcherCancelStopsFetcher(t *testing.T) {
	requester := newMockRequester()
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peersToSkip := &sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())

	// we start the fetcher, and then we immediately cancel the context
	go fetcher.run(ctx, peersToSkip)
	cancel()

	rctx, rcancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer rcancel()
	// we call Request with an active context
	fetcher.Request(rctx, 0)

	// fetcher should not initiate request, we can only check by waiting a bit and making sure no request is happening
	select {
	case <-requester.requestC:
		t.Fatalf("cancelled fetcher initiated request")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestFetchCancelStopsRequest tests that calling a Request function with a cancelled context does not initiate a request
func TestFetcherCancelStopsRequest(t *testing.T) {
	requester := newMockRequester(100 * time.Millisecond)
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peersToSkip := &sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// we start the fetcher with an active context
	go fetcher.run(ctx, peersToSkip)

	rctx, rcancel := context.WithCancel(context.Background())
	rcancel()

	// we call Request with a cancelled context
	fetcher.Request(rctx, 0)

	// fetcher should not initiate request, we can only check by waiting a bit and making sure no request is happening
	select {
	case <-requester.requestC:
		t.Fatalf("cancelled fetch function initiated request")
	case <-time.After(200 * time.Millisecond):
	}

	// if there is another Request with active context, there should be a request, because the fetcher itself is not cancelled
	rctx = context.Background()
	fetcher.Request(rctx, 0)

	select {
	case <-requester.requestC:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected request")
	}
}

// TestOfferUsesSource tests Fetcher Offer behavior.
// In this case there should be 1 (and only one) request initiated from the source peer, and the
// source nodeid should appear in the peersToSkip map.
func TestFetcherOfferUsesSource(t *testing.T) {
	requester := newMockRequester(100 * time.Millisecond)
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peersToSkip := &sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the fetcher
	go fetcher.run(ctx, peersToSkip)

	rctx := context.Background()
	// call the Offer function with the source peer
	fetcher.Offer(rctx, &sourcePeerID)

	// fetcher should not initiate request
	select {
	case <-requester.requestC:
		t.Fatalf("fetcher initiated request")
	case <-time.After(200 * time.Millisecond):
	}

	// call Request after the Offer
	rctx = context.Background()
	fetcher.Request(rctx, 0)

	// there should be exactly 1 request coming from fetcher
	var request *Request
	select {
	case request = <-requester.requestC:
		if *request.Source != sourcePeerID {
			t.Fatalf("Expected source id %v got %v", sourcePeerID, request.Source)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("fetcher did not initiate request")
	}

	select {
	case <-requester.requestC:
		t.Fatalf("Fetcher number of requests expected 1 got 2")
	case <-time.After(200 * time.Millisecond):
	}

	// source peer should be added to peersToSkip eventually
	time.Sleep(100 * time.Millisecond)
	if _, ok := request.peersToSkip.Load(sourcePeerID.String()); !ok {
		t.Fatalf("SourcePeerId not added to peersToSkip")
	}
}

func TestFetcherOfferAfterRequestUsesSourceFromContext(t *testing.T) {
	requester := newMockRequester(100 * time.Millisecond)
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peersToSkip := &sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the fetcher
	go fetcher.run(ctx, peersToSkip)

	// call Request first
	rctx := context.Background()
	fetcher.Request(rctx, 0)

	// there should be a request coming from fetcher
	var request *Request
	select {
	case request = <-requester.requestC:
		if request.Source != nil {
			t.Fatalf("Incorrect source peer id, expected nil got %v", request.Source)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("fetcher did not initiate request")
	}

	// after the Request call Offer
	fetcher.Offer(context.Background(), &sourcePeerID)

	// there should be a request coming from fetcher
	select {
	case request = <-requester.requestC:
		if *request.Source != sourcePeerID {
			t.Fatalf("Incorrect source peer id, expected %v got %v", sourcePeerID, request.Source)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("fetcher did not initiate request")
	}

	// source peer should be added to peersToSkip eventually
	time.Sleep(100 * time.Millisecond)
	if _, ok := request.peersToSkip.Load(sourcePeerID.String()); !ok {
		t.Fatalf("SourcePeerId not added to peersToSkip")
	}
}

// TestFetcherRetryOnTimeout tests that fetch retries after searchTimeOut has passed
func TestFetcherRetryOnTimeout(t *testing.T) {
	requester := newMockRequester()
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	peersToSkip := &sync.Map{}

	// set searchTimeOut to low value so the test is quicker
	defer func(t time.Duration) {
		searchTimeout = t
	}(searchTimeout)
	searchTimeout = 250 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start the fetcher
	go fetcher.run(ctx, peersToSkip)

	// call the fetch function with an active context
	rctx := context.Background()
	fetcher.Request(rctx, 0)

	// after 100ms the first request should be initiated
	time.Sleep(100 * time.Millisecond)

	select {
	case <-requester.requestC:
	default:
		t.Fatalf("fetch did not initiate request")
	}

	// after another 100ms no new request should be initiated, because search timeout is 250ms
	time.Sleep(100 * time.Millisecond)

	select {
	case <-requester.requestC:
		t.Fatalf("unexpected request from fetcher")
	default:
	}

	// after another 300ms search timeout is over, there should be a new request
	time.Sleep(300 * time.Millisecond)

	select {
	case <-requester.requestC:
	default:
		t.Fatalf("fetch did not retry request")
	}
}

// TestFetcherFactory creates a FetcherFactory and checks if the factory really creates and starts
// a Fetcher when it return a fetch function. We test the fetching functionality just by checking if
// a request is initiated when the fetch function is called
func TestFetcherFactory(t *testing.T) {
	requester := newMockRequester(100 * time.Millisecond)
	addr := make([]byte, 32)
	fetcherFactory := NewFetcherFactory(requester.doRequest, false)

	peersToSkip := &sync.Map{}

	fetcher := fetcherFactory.New(context.Background(), addr, peersToSkip)

	fetcher.Request(context.Background(), 0)

	// check if the created fetchFunction really starts a fetcher and initiates a request
	select {
	case <-requester.requestC:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("fetch timeout")
	}

}

func TestFetcherRequestQuitRetriesRequest(t *testing.T) {
	requester := newMockRequester()
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	// make sure searchTimeout is long so it is sure the request is not retried because of timeout
	defer func(t time.Duration) {
		searchTimeout = t
	}(searchTimeout)
	searchTimeout = 10 * time.Second

	peersToSkip := &sync.Map{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go fetcher.run(ctx, peersToSkip)

	rctx := context.Background()
	fetcher.Request(rctx, 0)

	select {
	case <-requester.requestC:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("request is not initiated")
	}

	close(requester.quitC)

	select {
	case <-requester.requestC:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("request is not initiated after failed request")
	}
}

// TestRequestSkipPeer checks if PeerSkip function will skip provided peer
// and not skip unknown one.
func TestRequestSkipPeer(t *testing.T) {
	addr := make([]byte, 32)
	peers := []enode.ID{
		enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8"),
		enode.HexID("99d8594b52298567d2ca3f4c441a5ba0140ee9245e26460d01102a52773c73b9"),
	}

	peersToSkip := new(sync.Map)
	peersToSkip.Store(peers[0].String(), time.Now())
	r := NewRequest(addr, false, peersToSkip)

	if !r.SkipPeer(peers[0].String()) {
		t.Errorf("peer not skipped")
	}

	if r.SkipPeer(peers[1].String()) {
		t.Errorf("peer skipped")
	}
}

// TestRequestSkipPeerExpired checks if a peer to skip is not skipped
// after RequestTimeout has passed.
func TestRequestSkipPeerExpired(t *testing.T) {
	addr := make([]byte, 32)
	peer := enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8")

	// set RequestTimeout to a low value and reset it after the test
	defer func(t time.Duration) { RequestTimeout = t }(RequestTimeout)
	RequestTimeout = 250 * time.Millisecond

	peersToSkip := new(sync.Map)
	peersToSkip.Store(peer.String(), time.Now())
	r := NewRequest(addr, false, peersToSkip)

	if !r.SkipPeer(peer.String()) {
		t.Errorf("peer not skipped")
	}

	time.Sleep(500 * time.Millisecond)

	if r.SkipPeer(peer.String()) {
		t.Errorf("peer skipped")
	}
}

// TestRequestSkipPeerPermanent checks if a peer to skip is not skipped
// after RequestTimeout is not skipped if it is set for a permanent skipping
// by value to peersToSkip map is not time.Duration.
func TestRequestSkipPeerPermanent(t *testing.T) {
	addr := make([]byte, 32)
	peer := enode.HexID("3431c3939e1ee2a6345e976a8234f9870152d64879f30bc272a074f6859e75e8")

	// set RequestTimeout to a low value and reset it after the test
	defer func(t time.Duration) { RequestTimeout = t }(RequestTimeout)
	RequestTimeout = 250 * time.Millisecond

	peersToSkip := new(sync.Map)
	peersToSkip.Store(peer.String(), true)
	r := NewRequest(addr, false, peersToSkip)

	if !r.SkipPeer(peer.String()) {
		t.Errorf("peer not skipped")
	}

	time.Sleep(500 * time.Millisecond)

	if !r.SkipPeer(peer.String()) {
		t.Errorf("peer not skipped")
	}
}

func TestFetcherMaxHopCount(t *testing.T) {
	requester := newMockRequester()
	addr := make([]byte, 32)
	fetcher := NewFetcher(addr, requester.doRequest, true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peersToSkip := &sync.Map{}

	go fetcher.run(ctx, peersToSkip)

	rctx := context.Background()
	fetcher.Request(rctx, maxHopCount)

	// if hopCount is already at max no request should be initiated
	select {
	case <-requester.requestC:
		t.Fatalf("cancelled fetcher initiated request")
	case <-time.After(200 * time.Millisecond):
	}
}
