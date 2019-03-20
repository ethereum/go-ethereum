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

package storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
	"github.com/syndtr/goleveldb/leveldb"

	lru "github.com/hashicorp/golang-lru"
)

type (
	NewNetFetcherFunc func(ctx context.Context, addr Address, peers *sync.Map) NetFetcher
)

type NetFetcher interface {
	Request(hopCount uint8)
	Offer(source *enode.ID)
}

// NetStore is an extension of local storage
// it implements the ChunkStore interface
// on request it initiates remote cloud retrieval using a fetcher
// fetchers are unique to a chunk and are stored in fetchers LRU memory cache
// fetchFuncFactory is a factory object to create a fetch function for a specific chunk address
type NetStore struct {
	mu                sync.Mutex
	store             SyncChunkStore
	fetchers          *lru.Cache
	NewNetFetcherFunc NewNetFetcherFunc
	closeC            chan struct{}
}

var fetcherTimeout = 2 * time.Minute // timeout to cancel the fetcher even if requests are coming in

// NewNetStore creates a new NetStore object using the given local store. newFetchFunc is a
// constructor function that can create a fetch function for a specific chunk address.
func NewNetStore(store SyncChunkStore, nnf NewNetFetcherFunc) (*NetStore, error) {
	fetchers, err := lru.New(defaultChunkRequestsCacheCapacity)
	if err != nil {
		return nil, err
	}
	return &NetStore{
		store:             store,
		fetchers:          fetchers,
		NewNetFetcherFunc: nnf,
		closeC:            make(chan struct{}),
	}, nil
}

// Put stores a chunk in localstore, and delivers to all requestor peers using the fetcher stored in
// the fetchers cache
func (n *NetStore) Put(ctx context.Context, ch Chunk) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// put to the chunk to the store, there should be no error
	err := n.store.Put(ctx, ch)
	if err != nil {
		return err
	}

	// if chunk is now put in the store, check if there was an active fetcher and call deliver on it
	// (this delivers the chunk to requestors via the fetcher)
	log.Trace("n.getFetcher", "ref", ch.Address())
	if f := n.getFetcher(ch.Address()); f != nil {
		log.Trace("n.getFetcher deliver", "ref", ch.Address())
		f.deliver(ctx, ch)
	}
	return nil
}

// Get retrieves the chunk from the NetStore DPA synchronously.
// It calls NetStore.get, and if the chunk is not in local Storage
// it calls fetch with the request, which blocks until the chunk
// arrived or context is done
func (n *NetStore) Get(rctx context.Context, ref Address) (Chunk, error) {
	chunk, fetch, err := n.get(rctx, ref)
	if err != nil {
		return nil, err
	}
	if chunk != nil {
		// this is not measuring how long it takes to get the chunk for the localstore, but
		// rather just adding a span for clarity when inspecting traces in Jaeger, in order
		// to make it easier to reason which is the node that actually delivered a chunk.
		_, sp := spancontext.StartSpan(
			rctx,
			"localstore.get")
		defer sp.Finish()

		return chunk, nil
	}
	return fetch(rctx)
}

func (n *NetStore) BinIndex(po uint8) uint64 {
	return n.store.BinIndex(po)
}

func (n *NetStore) Iterator(from uint64, to uint64, po uint8, f func(Address, uint64) bool) error {
	return n.store.Iterator(from, to, po, f)
}

// FetchFunc returns nil if the store contains the given address. Otherwise it returns a wait function,
// which returns after the chunk is available or the context is done
func (n *NetStore) FetchFunc(ctx context.Context, ref Address) func(context.Context) error {
	chunk, fetch, _ := n.get(ctx, ref)
	if chunk != nil {
		return nil
	}
	return func(ctx context.Context) error {
		_, err := fetch(ctx)
		return err
	}
}

// Close chunk store
func (n *NetStore) Close() {
	close(n.closeC)
	n.store.Close()

	wg := sync.WaitGroup{}
	for _, key := range n.fetchers.Keys() {
		if f, ok := n.fetchers.Get(key); ok {
			if fetch, ok := f.(*fetcher); ok {
				wg.Add(1)
				go func(fetch *fetcher) {
					defer wg.Done()
					fetch.cancel()

					select {
					case <-fetch.deliveredC:
					case <-fetch.cancelledC:
					}
				}(fetch)
			}
		}
	}
	wg.Wait()
}

// get attempts at retrieving the chunk from LocalStore
// If it is not found then using getOrCreateFetcher:
//     1. Either there is already a fetcher to retrieve it
//     2. A new fetcher is created and saved in the fetchers cache
// From here on, all Get will hit on this fetcher until the chunk is delivered
// or all fetcher contexts are done.
// It returns a chunk, a fetcher function and an error
// If chunk is nil, the returned fetch function needs to be called with a context to return the chunk.
func (n *NetStore) get(ctx context.Context, ref Address) (Chunk, func(context.Context) (Chunk, error), error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	chunk, err := n.store.Get(ctx, ref)
	if err != nil {
		// TODO: Fix comparison - we should be comparing against leveldb.ErrNotFound, this error should be wrapped.
		if err != ErrChunkNotFound && err != leveldb.ErrNotFound {
			log.Debug("Received error from LocalStore other than ErrNotFound", "err", err)
		}
		// The chunk is not available in the LocalStore, let's get the fetcher for it, or create a new one
		// if it doesn't exist yet
		f := n.getOrCreateFetcher(ctx, ref)
		// If the caller needs the chunk, it has to use the returned fetch function to get it
		return nil, f.Fetch, nil
	}

	return chunk, nil, nil
}

// Has is the storage layer entry point to query the underlying
// database to return if it has a chunk or not.
// Called from the DebugAPI
func (n *NetStore) Has(ctx context.Context, ref Address) bool {
	return n.store.Has(ctx, ref)
}

// getOrCreateFetcher attempts at retrieving an existing fetchers
// if none exists, creates one and saves it in the fetchers cache
// caller must hold the lock
func (n *NetStore) getOrCreateFetcher(ctx context.Context, ref Address) *fetcher {
	if f := n.getFetcher(ref); f != nil {
		return f
	}

	// no fetcher for the given address, we have to create a new one
	key := hex.EncodeToString(ref)
	// create the context during which fetching is kept alive
	cctx, cancel := context.WithTimeout(ctx, fetcherTimeout)
	// destroy is called when all requests finish
	destroy := func() {
		// remove fetcher from fetchers
		n.fetchers.Remove(key)
		// stop fetcher by cancelling context called when
		// all requests cancelled/timedout or chunk is delivered
		cancel()
	}
	// peers always stores all the peers which have an active request for the chunk. It is shared
	// between fetcher and the NewFetchFunc function. It is needed by the NewFetchFunc because
	// the peers which requested the chunk should not be requested to deliver it.
	peers := &sync.Map{}

	cctx, sp := spancontext.StartSpan(
		cctx,
		"netstore.fetcher",
	)

	sp.LogFields(olog.String("ref", ref.String()))
	fetcher := newFetcher(sp, ref, n.NewNetFetcherFunc(cctx, ref, peers), destroy, peers, n.closeC)
	n.fetchers.Add(key, fetcher)

	return fetcher
}

// getFetcher retrieves the fetcher for the given address from the fetchers cache if it exists,
// otherwise it returns nil
func (n *NetStore) getFetcher(ref Address) *fetcher {
	key := hex.EncodeToString(ref)
	f, ok := n.fetchers.Get(key)
	if ok {
		return f.(*fetcher)
	}
	return nil
}

// RequestsCacheLen returns the current number of outgoing requests stored in the cache
func (n *NetStore) RequestsCacheLen() int {
	return n.fetchers.Len()
}

// One fetcher object is responsible to fetch one chunk for one address, and keep track of all the
// peers who have requested it and did not receive it yet.
type fetcher struct {
	addr        Address          // address of chunk
	chunk       Chunk            // fetcher can set the chunk on the fetcher
	deliveredC  chan struct{}    // chan signalling chunk delivery to requests
	cancelledC  chan struct{}    // chan signalling the fetcher has been cancelled (removed from fetchers in NetStore)
	netFetcher  NetFetcher       // remote fetch function to be called with a request source taken from the context
	cancel      func()           // cleanup function for the remote fetcher to call when all upstream contexts are called
	peers       *sync.Map        // the peers which asked for the chunk
	requestCnt  int32            // number of requests on this chunk. If all the requests are done (delivered or context is done) the cancel function is called
	deliverOnce *sync.Once       // guarantees that we only close deliveredC once
	span        opentracing.Span // measure retrieve time per chunk
}

// newFetcher creates a new fetcher object for the fiven addr. fetch is the function which actually
// does the retrieval (in non-test cases this is coming from the network package). cancel function is
// called either
//     1. when the chunk has been fetched all peers have been either notified or their context has been done
//     2. the chunk has not been fetched but all context from all the requests has been done
// The peers map stores all the peers which have requested chunk.
func newFetcher(span opentracing.Span, addr Address, nf NetFetcher, cancel func(), peers *sync.Map, closeC chan struct{}) *fetcher {
	cancelOnce := &sync.Once{} // cancel should only be called once
	return &fetcher{
		addr:        addr,
		deliveredC:  make(chan struct{}),
		deliverOnce: &sync.Once{},
		cancelledC:  closeC,
		netFetcher:  nf,
		cancel: func() {
			cancelOnce.Do(func() {
				cancel()
			})
		},
		peers: peers,
		span:  span,
	}
}

// Fetch fetches the chunk synchronously, it is called by NetStore.Get is the chunk is not available
// locally.
func (f *fetcher) Fetch(rctx context.Context) (Chunk, error) {
	atomic.AddInt32(&f.requestCnt, 1)
	defer func() {
		// if all the requests are done the fetcher can be cancelled
		if atomic.AddInt32(&f.requestCnt, -1) == 0 {
			f.cancel()
		}
		f.span.Finish()
	}()

	// The peer asking for the chunk. Store in the shared peers map, but delete after the request
	// has been delivered
	peer := rctx.Value("peer")
	if peer != nil {
		f.peers.Store(peer, time.Now())
		defer f.peers.Delete(peer)
	}

	// If there is a source in the context then it is an offer, otherwise a request
	sourceIF := rctx.Value("source")

	hopCount, _ := rctx.Value("hopcount").(uint8)

	if sourceIF != nil {
		var source enode.ID
		if err := source.UnmarshalText([]byte(sourceIF.(string))); err != nil {
			return nil, err
		}
		f.netFetcher.Offer(&source)
	} else {
		f.netFetcher.Request(hopCount)
	}

	// wait until either the chunk is delivered or the context is done
	select {
	case <-rctx.Done():
		return nil, rctx.Err()
	case <-f.deliveredC:
		return f.chunk, nil
	case <-f.cancelledC:
		return nil, fmt.Errorf("fetcher cancelled")
	}
}

// deliver is called by NetStore.Put to notify all pending requests
func (f *fetcher) deliver(ctx context.Context, ch Chunk) {
	f.deliverOnce.Do(func() {
		f.chunk = ch
		// closing the deliveredC channel will terminate ongoing requests
		close(f.deliveredC)
		log.Trace("n.getFetcher close deliveredC", "ref", ch.Address())
	})
}
