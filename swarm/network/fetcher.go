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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var searchTimeout = 1 * time.Second

// Time to consider peer to be skipped.
// Also used in stream delivery.
var RequestTimeout = 10 * time.Second

var maxHopCount uint8 = 20 // maximum number of forwarded requests (hops), to make sure requests are not forwarded forever in peer loops

type RequestFunc func(context.Context, *Request) (*enode.ID, chan struct{}, error)

// Fetcher is created when a chunk is not found locally. It starts a request handler loop once and
// keeps it alive until all active requests are completed. This can happen:
//     1. either because the chunk is delivered
//     2. or becuse the requestor cancelled/timed out
// Fetcher self destroys itself after it is completed.
// TODO: cancel all forward requests after termination
type Fetcher struct {
	protoRequestFunc RequestFunc     // request function fetcher calls to issue retrieve request for a chunk
	addr             storage.Address // the address of the chunk to be fetched
	offerC           chan *enode.ID  // channel of sources (peer node id strings)
	requestC         chan uint8      // channel for incoming requests (with the hopCount value in it)
	skipCheck        bool
}

type Request struct {
	Addr        storage.Address // chunk address
	Source      *enode.ID       // nodeID of peer to request from (can be nil)
	SkipCheck   bool            // whether to offer the chunk first or deliver directly
	peersToSkip *sync.Map       // peers not to request chunk from (only makes sense if source is nil)
	HopCount    uint8           // number of forwarded requests (hops)
}

// NewRequest returns a new instance of Request based on chunk address skip check and
// a map of peers to skip.
func NewRequest(addr storage.Address, skipCheck bool, peersToSkip *sync.Map) *Request {
	return &Request{
		Addr:        addr,
		SkipCheck:   skipCheck,
		peersToSkip: peersToSkip,
	}
}

// SkipPeer returns if the peer with nodeID should not be requested to deliver a chunk.
// Peers to skip are kept per Request and for a time period of RequestTimeout.
// This function is used in stream package in Delivery.RequestFromPeers to optimize
// requests for chunks.
func (r *Request) SkipPeer(nodeID string) bool {
	val, ok := r.peersToSkip.Load(nodeID)
	if !ok {
		return false
	}
	t, ok := val.(time.Time)
	if ok && time.Now().After(t.Add(RequestTimeout)) {
		// deadine expired
		r.peersToSkip.Delete(nodeID)
		return false
	}
	return true
}

// FetcherFactory is initialised with a request function and can create fetchers
type FetcherFactory struct {
	request   RequestFunc
	skipCheck bool
}

// NewFetcherFactory takes a request function and skip check parameter and creates a FetcherFactory
func NewFetcherFactory(request RequestFunc, skipCheck bool) *FetcherFactory {
	return &FetcherFactory{
		request:   request,
		skipCheck: skipCheck,
	}
}

// New contructs a new Fetcher, for the given chunk. All peers in peersToSkip are not requested to
// deliver the given chunk. peersToSkip should always contain the peers which are actively requesting
// this chunk, to make sure we don't request back the chunks from them.
// The created Fetcher is started and returned.
func (f *FetcherFactory) New(ctx context.Context, source storage.Address, peersToSkip *sync.Map) storage.NetFetcher {
	fetcher := NewFetcher(source, f.request, f.skipCheck)
	go fetcher.run(ctx, peersToSkip)
	return fetcher
}

// NewFetcher creates a new Fetcher for the given chunk address using the given request function.
func NewFetcher(addr storage.Address, rf RequestFunc, skipCheck bool) *Fetcher {
	return &Fetcher{
		addr:             addr,
		protoRequestFunc: rf,
		offerC:           make(chan *enode.ID),
		requestC:         make(chan uint8),
		skipCheck:        skipCheck,
	}
}

// Offer is called when an upstream peer offers the chunk via syncing as part of `OfferedHashesMsg` and the node does not have the chunk locally.
func (f *Fetcher) Offer(ctx context.Context, source *enode.ID) {
	// First we need to have this select to make sure that we return if context is done
	select {
	case <-ctx.Done():
		return
	default:
	}

	// This select alone would not guarantee that we return of context is done, it could potentially
	// push to offerC instead if offerC is available (see number 2 in https://golang.org/ref/spec#Select_statements)
	select {
	case f.offerC <- source:
	case <-ctx.Done():
	}
}

// Request is called when an upstream peer request the chunk as part of `RetrieveRequestMsg`, or from a local request through FileStore, and the node does not have the chunk locally.
func (f *Fetcher) Request(ctx context.Context, hopCount uint8) {
	// First we need to have this select to make sure that we return if context is done
	select {
	case <-ctx.Done():
		return
	default:
	}

	if hopCount >= maxHopCount {
		log.Debug("fetcher request hop count limit reached", "hops", hopCount)
		return
	}

	// This select alone would not guarantee that we return of context is done, it could potentially
	// push to offerC instead if offerC is available (see number 2 in https://golang.org/ref/spec#Select_statements)
	select {
	case f.requestC <- hopCount + 1:
	case <-ctx.Done():
	}
}

// start prepares the Fetcher
// it keeps the Fetcher alive within the lifecycle of the passed context
func (f *Fetcher) run(ctx context.Context, peers *sync.Map) {
	var (
		doRequest bool             // determines if retrieval is initiated in the current iteration
		wait      *time.Timer      // timer for search timeout
		waitC     <-chan time.Time // timer channel
		sources   []*enode.ID      // known sources, ie. peers that offered the chunk
		requested bool             // true if the chunk was actually requested
		hopCount  uint8
	)
	gone := make(chan *enode.ID) // channel to signal that a peer we requested from disconnected

	// loop that keeps the fetching process alive
	// after every request a timer is set. If this goes off we request again from another peer
	// note that the previous request is still alive and has the chance to deliver, so
	// rerequesting extends the search. ie.,
	// if a peer we requested from is gone we issue a new request, so the number of active
	// requests never decreases
	for {
		select {

		// incoming offer
		case source := <-f.offerC:
			log.Trace("new source", "peer addr", source, "request addr", f.addr)
			// 1) the chunk is offered by a syncing peer
			// add to known sources
			sources = append(sources, source)
			// launch a request to the source iff the chunk was requested (not just expected because its offered by a syncing peer)
			doRequest = requested

		// incoming request
		case hopCount = <-f.requestC:
			log.Trace("new request", "request addr", f.addr)
			// 2) chunk is requested, set requested flag
			// launch a request iff none been launched yet
			doRequest = !requested
			requested = true

			// peer we requested from is gone. fall back to another
			// and remove the peer from the peers map
		case id := <-gone:
			log.Trace("peer gone", "peer id", id.String(), "request addr", f.addr)
			peers.Delete(id.String())
			doRequest = requested

		// search timeout: too much time passed since the last request,
		// extend the search to a new peer if we can find one
		case <-waitC:
			log.Trace("search timed out: rerequesting", "request addr", f.addr)
			doRequest = requested

			// all Fetcher context closed, can quit
		case <-ctx.Done():
			log.Trace("terminate fetcher", "request addr", f.addr)
			// TODO: send cancelations to all peers left over in peers map (i.e., those we requested from)
			return
		}

		// need to issue a new request
		if doRequest {
			var err error
			sources, err = f.doRequest(ctx, gone, peers, sources, hopCount)
			if err != nil {
				log.Info("unable to request", "request addr", f.addr, "err", err)
			}
		}

		// if wait channel is not set, set it to a timer
		if requested {
			if wait == nil {
				wait = time.NewTimer(searchTimeout)
				defer wait.Stop()
				waitC = wait.C
			} else {
				// stop the timer and drain the channel if it was not drained earlier
				if !wait.Stop() {
					select {
					case <-wait.C:
					default:
					}
				}
				// reset the timer to go off after searchTimeout
				wait.Reset(searchTimeout)
			}
		}
		doRequest = false
	}
}

// doRequest attempts at finding a peer to request the chunk from
// * first it tries to request explicitly from peers that are known to have offered the chunk
// * if there are no such peers (available) it tries to request it from a peer closest to the chunk address
//   excluding those in the peersToSkip map
// * if no such peer is found an error is returned
//
// if a request is successful,
// * the peer's address is added to the set of peers to skip
// * the peer's address is removed from prospective sources, and
// * a go routine is started that reports on the gone channel if the peer is disconnected (or terminated their streamer)
func (f *Fetcher) doRequest(ctx context.Context, gone chan *enode.ID, peersToSkip *sync.Map, sources []*enode.ID, hopCount uint8) ([]*enode.ID, error) {
	var i int
	var sourceID *enode.ID
	var quit chan struct{}

	req := &Request{
		Addr:        f.addr,
		SkipCheck:   f.skipCheck,
		peersToSkip: peersToSkip,
		HopCount:    hopCount,
	}

	foundSource := false
	// iterate over known sources
	for i = 0; i < len(sources); i++ {
		req.Source = sources[i]
		var err error
		sourceID, quit, err = f.protoRequestFunc(ctx, req)
		if err == nil {
			// remove the peer from known sources
			// Note: we can modify the source although we are looping on it, because we break from the loop immediately
			sources = append(sources[:i], sources[i+1:]...)
			foundSource = true
			break
		}
	}

	// if there are no known sources, or none available, we try request from a closest node
	if !foundSource {
		req.Source = nil
		var err error
		sourceID, quit, err = f.protoRequestFunc(ctx, req)
		if err != nil {
			// if no peers found to request from
			return sources, err
		}
	}
	// add peer to the set of peers to skip from now
	peersToSkip.Store(sourceID.String(), time.Now())

	// if the quit channel is closed, it indicates that the source peer we requested from
	// disconnected or terminated its streamer
	// here start a go routine that watches this channel and reports the source peer on the gone channel
	// this go routine quits if the fetcher global context is done to prevent process leak
	go func() {
		select {
		case <-quit:
			gone <- sourceID
		case <-ctx.Done():
		}
	}()
	return sources, nil
}
