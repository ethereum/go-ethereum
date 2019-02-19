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

package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	ch "github.com/ethereum/go-ethereum/swarm/chunk"
)

var sourcePeerID = enode.HexID("99d8594b52298567d2ca3f4c441a5ba0140ee9245e26460d01102a52773c73b9")

type mockNetFetcher struct {
	peers           *sync.Map
	sources         []*enode.ID
	peersPerRequest [][]Address
	requestCalled   bool
	offerCalled     bool
	quit            <-chan struct{}
	ctx             context.Context
	hopCounts       []uint8
	mu              sync.Mutex
}

func (m *mockNetFetcher) Offer(source *enode.ID) {
	m.offerCalled = true
	m.sources = append(m.sources, source)
}

func (m *mockNetFetcher) Request(hopCount uint8) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCalled = true
	var peers []Address
	m.peers.Range(func(key interface{}, _ interface{}) bool {
		peers = append(peers, common.FromHex(key.(string)))
		return true
	})
	m.peersPerRequest = append(m.peersPerRequest, peers)
	m.hopCounts = append(m.hopCounts, hopCount)
}

type mockNetFetchFuncFactory struct {
	fetcher *mockNetFetcher
}

func (m *mockNetFetchFuncFactory) newMockNetFetcher(ctx context.Context, _ Address, peers *sync.Map) NetFetcher {
	m.fetcher.peers = peers
	m.fetcher.quit = ctx.Done()
	m.fetcher.ctx = ctx
	return m.fetcher
}

func mustNewNetStore(t *testing.T) *NetStore {
	netStore, _ := mustNewNetStoreWithFetcher(t)
	return netStore
}

func mustNewNetStoreWithFetcher(t *testing.T) (*NetStore, *mockNetFetcher) {
	t.Helper()

	datadir, err := ioutil.TempDir("", "netstore")
	if err != nil {
		t.Fatal(err)
	}
	naddr := make([]byte, 32)
	params := NewDefaultLocalStoreParams()
	params.Init(datadir)
	params.BaseKey = naddr
	localStore, err := NewTestLocalStoreForAddr(params)
	if err != nil {
		t.Fatal(err)
	}

	fetcher := &mockNetFetcher{}
	mockNetFetchFuncFactory := &mockNetFetchFuncFactory{
		fetcher: fetcher,
	}
	netStore, err := NewNetStore(localStore, mockNetFetchFuncFactory.newMockNetFetcher)
	if err != nil {
		t.Fatal(err)
	}
	return netStore, fetcher
}

// TestNetStoreGetAndPut tests calling NetStore.Get which is blocked until the same chunk is Put.
// After the Put there should no active fetchers, and the context created for the fetcher should
// be cancelled.
func TestNetStoreGetAndPut(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c := make(chan struct{}) // this channel ensures that the gouroutine with the Put does not run earlier than the Get
	putErrC := make(chan error)
	go func() {
		<-c                                // wait for the Get to be called
		time.Sleep(200 * time.Millisecond) // and a little more so it is surely called

		// check if netStore created a fetcher in the Get call for the unavailable chunk
		if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
			putErrC <- errors.New("Expected netStore to use a fetcher for the Get call")
			return
		}

		err := netStore.Put(ctx, chunk)
		if err != nil {
			putErrC <- fmt.Errorf("Expected no err got %v", err)
			return
		}

		putErrC <- nil
	}()

	close(c)
	recChunk, err := netStore.Get(ctx, chunk.Address()) // this is blocked until the Put above is done
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	if err := <-putErrC; err != nil {
		t.Fatal(err)
	}
	// the retrieved chunk should be the same as what we Put
	if !bytes.Equal(recChunk.Address(), chunk.Address()) || !bytes.Equal(recChunk.Data(), chunk.Data()) {
		t.Fatalf("Different chunk received than what was put")
	}
	// the chunk is already available locally, so there should be no active fetchers waiting for it
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after delivery")
	}

	// A fetcher was created when the Get was called (and the chunk was not available). The chunk
	// was delivered with the Put call, so the fetcher should be cancelled now.
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}

}

// TestNetStoreGetAndPut tests calling NetStore.Put and then NetStore.Get.
// After the Put the chunk is available locally, so the Get can just retrieve it from LocalStore,
// there is no need to create fetchers.
func TestNetStoreGetAfterPut(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// First we Put the chunk, so the chunk will be available locally
	err := netStore.Put(ctx, chunk)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	// Get should retrieve the chunk from LocalStore, without creating fetcher
	recChunk, err := netStore.Get(ctx, chunk.Address())
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}
	// the retrieved chunk should be the same as what we Put
	if !bytes.Equal(recChunk.Address(), chunk.Address()) || !bytes.Equal(recChunk.Data(), chunk.Data()) {
		t.Fatalf("Different chunk received than what was put")
	}
	// no fetcher offer or request should be created for a locally available chunk
	if fetcher.offerCalled || fetcher.requestCalled {
		t.Fatal("NetFetcher.offerCalled or requestCalled not expected to be called")
	}
	// no fetchers should be created for a locally available chunk
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to not have fetcher")
	}

}

// TestNetStoreGetTimeout tests a Get call for an unavailable chunk and waits for timeout
func TestNetStoreGetTimeout(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	c := make(chan struct{}) // this channel ensures that the gouroutine does not run earlier than the Get
	fetcherErrC := make(chan error)
	go func() {
		<-c                                // wait for the Get to be called
		time.Sleep(200 * time.Millisecond) // and a little more so it is surely called

		// check if netStore created a fetcher in the Get call for the unavailable chunk
		if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
			fetcherErrC <- errors.New("Expected netStore to use a fetcher for the Get call")
			return
		}

		fetcherErrC <- nil
	}()

	close(c)
	// We call Get on this chunk, which is not in LocalStore. We don't Put it at all, so there will
	// be a timeout
	_, err := netStore.Get(ctx, chunk.Address())

	// Check if the timeout happened
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected context.DeadLineExceeded err got %v", err)
	}

	if err := <-fetcherErrC; err != nil {
		t.Fatal(err)
	}

	// A fetcher was created, check if it has been removed after timeout
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after timeout")
	}

	// Check if the fetcher context has been cancelled after the timeout
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}
}

// TestNetStoreGetCancel tests a Get call for an unavailable chunk, then cancels the context and checks
// the errors
func TestNetStoreGetCancel(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

	c := make(chan struct{}) // this channel ensures that the gouroutine with the cancel does not run earlier than the Get
	fetcherErrC := make(chan error, 1)
	go func() {
		<-c                                // wait for the Get to be called
		time.Sleep(200 * time.Millisecond) // and a little more so it is surely called
		// check if netStore created a fetcher in the Get call for the unavailable chunk
		if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
			fetcherErrC <- errors.New("Expected netStore to use a fetcher for the Get call")
			return
		}

		fetcherErrC <- nil
		cancel()
	}()

	close(c)

	// We call Get with an unavailable chunk, so it will create a fetcher and wait for delivery
	_, err := netStore.Get(ctx, chunk.Address())

	if err := <-fetcherErrC; err != nil {
		t.Fatal(err)
	}

	// After the context is cancelled above Get should return with an error
	if err != context.Canceled {
		t.Fatalf("Expected context.Canceled err got %v", err)
	}

	// A fetcher was created, check if it has been removed after cancel
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after cancel")
	}

	// Check if the fetcher context has been cancelled after the request context cancel
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}
}

// TestNetStoreMultipleGetAndPut tests four Get calls for the same unavailable chunk. The chunk is
// delivered with a Put, we have to make sure all Get calls return, and they use a single fetcher
// for the chunk retrieval
func TestNetStoreMultipleGetAndPut(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	putErrC := make(chan error)
	go func() {
		// sleep to make sure Put is called after all the Get
		time.Sleep(500 * time.Millisecond)
		// check if netStore created exactly one fetcher for all Get calls
		if netStore.fetchers.Len() != 1 {
			putErrC <- errors.New("Expected netStore to use one fetcher for all Get calls")
			return
		}
		err := netStore.Put(ctx, chunk)
		if err != nil {
			putErrC <- fmt.Errorf("Expected no err got %v", err)
			return
		}
		putErrC <- nil
	}()

	count := 4
	// call Get 4 times for the same unavailable chunk. The calls will be blocked until the Put above.
	errC := make(chan error)
	for i := 0; i < count; i++ {
		go func() {
			recChunk, err := netStore.Get(ctx, chunk.Address())
			if err != nil {
				errC <- fmt.Errorf("Expected no err got %v", err)
			}
			if !bytes.Equal(recChunk.Address(), chunk.Address()) || !bytes.Equal(recChunk.Data(), chunk.Data()) {
				errC <- errors.New("Different chunk received than what was put")
			}
			errC <- nil
		}()
	}

	if err := <-putErrC; err != nil {
		t.Fatal(err)
	}

	timeout := time.After(1 * time.Second)

	// The Get calls should return after Put, so no timeout expected
	for i := 0; i < count; i++ {
		select {
		case err := <-errC:
			if err != nil {
				t.Fatal(err)
			}
		case <-timeout:
			t.Fatalf("Timeout waiting for Get calls to return")
		}
	}

	// A fetcher was created, check if it has been removed after cancel
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after delivery")
	}

	// A fetcher was created, check if it has been removed after delivery
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}

}

// TestNetStoreFetchFuncTimeout tests a FetchFunc call for an unavailable chunk and waits for timeout
func TestNetStoreFetchFuncTimeout(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// FetchFunc is called for an unavaible chunk, so the returned wait function should not be nil
	wait := netStore.FetchFunc(ctx, chunk.Address())
	if wait == nil {
		t.Fatal("Expected wait function to be not nil")
	}

	// There should an active fetcher for the chunk after the FetchFunc call
	if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
		t.Fatalf("Expected netStore to have one fetcher for the requested chunk")
	}

	// wait function should timeout because we don't deliver the chunk with a Put
	err := wait(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("Expected context.DeadLineExceeded err got %v", err)
	}

	// the fetcher should be removed after timeout
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after timeout")
	}

	// the fetcher context should be cancelled after timeout
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}
}

// TestNetStoreFetchFuncAfterPut tests that the FetchFunc should return nil for a locally available chunk
func TestNetStoreFetchFuncAfterPut(t *testing.T) {
	netStore := mustNewNetStore(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// We deliver the created the chunk with a Put
	err := netStore.Put(ctx, chunk)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	// FetchFunc should return nil, because the chunk is available locally, no need to fetch it
	wait := netStore.FetchFunc(ctx, chunk.Address())
	if wait != nil {
		t.Fatal("Expected wait to be nil")
	}

	// No fetchers should be created at all
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to not have fetcher")
	}
}

// TestNetStoreGetCallsRequest tests if Get created a request on the NetFetcher for an unavailable chunk
func TestNetStoreGetCallsRequest(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx := context.WithValue(context.Background(), "hopcount", uint8(5))
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	// We call get for a not available chunk, it will timeout because the chunk is not delivered
	_, err := netStore.Get(ctx, chunk.Address())

	if err != context.DeadlineExceeded {
		t.Fatalf("Expected context.DeadlineExceeded err got %v", err)
	}

	// NetStore should call NetFetcher.Request and wait for the chunk
	if !fetcher.requestCalled {
		t.Fatal("Expected NetFetcher.Request to be called")
	}

	if fetcher.hopCounts[0] != 5 {
		t.Fatalf("Expected NetFetcher.Request be called with hopCount 5, got %v", fetcher.hopCounts[0])
	}
}

// TestNetStoreGetCallsOffer tests if Get created a request on the NetFetcher for an unavailable chunk
// in case of a source peer provided in the context.
func TestNetStoreGetCallsOffer(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	//  If a source peer is added to the context, NetStore will handle it as an offer
	ctx := context.WithValue(context.Background(), "source", sourcePeerID.String())
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	// We call get for a not available chunk, it will timeout because the chunk is not delivered
	_, err := netStore.Get(ctx, chunk.Address())

	if err != context.DeadlineExceeded {
		t.Fatalf("Expect error %v got %v", context.DeadlineExceeded, err)
	}

	// NetStore should call NetFetcher.Offer with the source peer
	if !fetcher.offerCalled {
		t.Fatal("Expected NetFetcher.Request to be called")
	}

	if len(fetcher.sources) != 1 {
		t.Fatalf("Expected fetcher sources length 1 got %v", len(fetcher.sources))
	}

	if fetcher.sources[0].String() != sourcePeerID.String() {
		t.Fatalf("Expected fetcher source %v got %v", sourcePeerID, fetcher.sources[0])
	}

}

// TestNetStoreFetcherCountPeers tests multiple NetStore.Get calls with peer in the context.
// There is no Put call, so the Get calls timeout
func TestNetStoreFetcherCountPeers(t *testing.T) {

	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	addr := randomAddr()
	peers := []string{randomAddr().Hex(), randomAddr().Hex(), randomAddr().Hex()}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	errC := make(chan error)
	nrGets := 3

	// Call Get 3 times with a peer in context
	for i := 0; i < nrGets; i++ {
		peer := peers[i]
		go func() {
			ctx := context.WithValue(ctx, "peer", peer)
			_, err := netStore.Get(ctx, addr)
			errC <- err
		}()
	}

	// All 3 Get calls should timeout
	for i := 0; i < nrGets; i++ {
		err := <-errC
		if err != context.DeadlineExceeded {
			t.Fatalf("Expected \"%v\" error got \"%v\"", context.DeadlineExceeded, err)
		}
	}

	// fetcher should be closed after timeout
	select {
	case <-fetcher.quit:
	case <-time.After(3 * time.Second):
		t.Fatalf("mockNetFetcher not closed after timeout")
	}

	// All 3 peers should be given to NetFetcher after the 3 Get calls
	if len(fetcher.peersPerRequest) != nrGets {
		t.Fatalf("Expected 3 got %v", len(fetcher.peersPerRequest))
	}

	for i, peers := range fetcher.peersPerRequest {
		if len(peers) < i+1 {
			t.Fatalf("Expected at least %v got %v", i+1, len(peers))
		}
	}
}

// TestNetStoreFetchFuncCalledMultipleTimes calls the wait function given by FetchFunc three times,
// and checks there is still exactly one fetcher for one chunk. Afthe chunk is delivered, it checks
// if the fetcher is closed.
func TestNetStoreFetchFuncCalledMultipleTimes(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// FetchFunc should return a non-nil wait function, because the chunk is not available
	wait := netStore.FetchFunc(ctx, chunk.Address())
	if wait == nil {
		t.Fatal("Expected wait function to be not nil")
	}

	// There should be exactly one fetcher for the chunk
	if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
		t.Fatalf("Expected netStore to have one fetcher for the requested chunk")
	}

	// Call wait three times in parallel
	count := 3
	errC := make(chan error)
	for i := 0; i < count; i++ {
		go func() {
			errC <- wait(ctx)
		}()
	}

	// sleep a little so the wait functions are called above
	time.Sleep(100 * time.Millisecond)

	// there should be still only one fetcher, because all wait calls are for the same chunk
	if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
		t.Fatal("Expected netStore to have one fetcher for the requested chunk")
	}

	// Deliver the chunk with a Put
	err := netStore.Put(ctx, chunk)
	if err != nil {
		t.Fatalf("Expected no err got %v", err)
	}

	// wait until all wait calls return (because the chunk is delivered)
	for i := 0; i < count; i++ {
		err := <-errC
		if err != nil {
			t.Fatal(err)
		}
	}

	// There should be no more fetchers for the delivered chunk
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after delivery")
	}

	// The context for the fetcher should be cancelled after delivery
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}
}

// TestNetStoreFetcherLifeCycleWithTimeout is similar to TestNetStoreFetchFuncCalledMultipleTimes,
// the only difference is that we don't deilver the chunk, just wait for timeout
func TestNetStoreFetcherLifeCycleWithTimeout(t *testing.T) {
	netStore, fetcher := mustNewNetStoreWithFetcher(t)

	chunk := GenerateRandomChunk(ch.DefaultSize)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// FetchFunc should return a non-nil wait function, because the chunk is not available
	wait := netStore.FetchFunc(ctx, chunk.Address())
	if wait == nil {
		t.Fatal("Expected wait function to be not nil")
	}

	// There should be exactly one fetcher for the chunk
	if netStore.fetchers.Len() != 1 || netStore.getFetcher(chunk.Address()) == nil {
		t.Fatalf("Expected netStore to have one fetcher for the requested chunk")
	}

	// Call wait three times in parallel
	count := 3
	errC := make(chan error)
	for i := 0; i < count; i++ {
		go func() {
			rctx, rcancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer rcancel()
			err := wait(rctx)
			if err != context.DeadlineExceeded {
				errC <- fmt.Errorf("Expected err %v got %v", context.DeadlineExceeded, err)
				return
			}
			errC <- nil
		}()
	}

	// wait until all wait calls timeout
	for i := 0; i < count; i++ {
		err := <-errC
		if err != nil {
			t.Fatal(err)
		}
	}

	// There should be no more fetchers after timeout
	if netStore.fetchers.Len() != 0 {
		t.Fatal("Expected netStore to remove the fetcher after delivery")
	}

	// The context for the fetcher should be cancelled after timeout
	select {
	case <-fetcher.ctx.Done():
	default:
		t.Fatal("Expected fetcher context to be cancelled")
	}
}

func randomAddr() Address {
	addr := make([]byte, 32)
	rand.Read(addr)
	return Address(addr)
}
