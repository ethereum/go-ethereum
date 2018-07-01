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
	"time"

	"github.com/ethereum/go-ethereum/swarm/log"
)

var (
	// NetStore.Get timeout for get and get retries
	// This is the maximum period that the Get will block.
	// If it is reached, Get will return ErrChunkNotFound.
	netStoreRetryTimeout = 30 * time.Second
	// Minimal period between calling get method on NetStore
	// on retry. It protects calling get very frequently if
	// it returns ErrChunkNotFound very fast.
	netStoreMinRetryDelay = 3 * time.Second
	// Timeout interval before retrieval is timed out.
	// It is used in NetStore.get on waiting for ReqC to be
	// closed on a single retrieve request.
	searchTimeout = 10 * time.Second
)

// NetStore implements the ChunkStore interface,
// this chunk access layer assumed 2 chunk stores
// local storage eg. LocalStore and network storage eg., NetStore
// access by calling network is blocking with a timeout
type NetStore struct {
	localStore *LocalStore
	retrieve   func(chunk *Chunk) error
}

func NewNetStore(localStore *LocalStore, retrieve func(chunk *Chunk) error) *NetStore {
	return &NetStore{localStore, retrieve}
}

// Get is the entrypoint for local retrieve requests
// waits for response or times out
//
// Get uses get method to retrieve request, but retries if the
// ErrChunkNotFound is returned by get, until the netStoreRetryTimeout
// is reached.
func (ns *NetStore) Get(addr Address) (chunk *Chunk, err error) {
	timer := time.NewTimer(netStoreRetryTimeout)
	defer timer.Stop()

	// result and resultC provide results from the goroutine
	// where NetStore.get is called.
	type result struct {
		chunk *Chunk
		err   error
	}
	resultC := make(chan result)

	// quitC ensures that retring goroutine is terminated
	// when this function returns.
	quitC := make(chan struct{})
	defer close(quitC)

	// do retries in a goroutine so that the timer can
	// force this method to return after the netStoreRetryTimeout.
	go func() {
		// limiter ensures that NetStore.get is not called more frequently
		// then netStoreMinRetryDelay. If NetStore.get takes longer
		// then netStoreMinRetryDelay, the next retry call will be
		// without a delay.
		limiter := time.NewTimer(netStoreMinRetryDelay)
		defer limiter.Stop()

		for {
			chunk, err := ns.get(addr, 0)
			if err != ErrChunkNotFound {
				// break retry only if the error is nil
				// or other error then ErrChunkNotFound
				select {
				case <-quitC:
					// Maybe NetStore.Get function has returned
					// by the timer.C while we were waiting for the
					// results. Terminate this goroutine.
				case resultC <- result{chunk: chunk, err: err}:
					// Send the result to the parrent goroutine.
				}
				return

			}
			select {
			case <-quitC:
				// NetStore.Get function has returned, possibly
				// by the timer.C, which makes this goroutine
				// not needed.
				return
			case <-limiter.C:
			}
			// Reset the limiter for the next iteration.
			limiter.Reset(netStoreMinRetryDelay)
			log.Debug("NetStore.Get retry chunk", "key", addr)
		}
	}()

	select {
	case r := <-resultC:
		return r.chunk, r.err
	case <-timer.C:
		return nil, ErrChunkNotFound
	}
}

// GetWithTimeout makes a single retrieval attempt for a chunk with a explicit timeout parameter
func (ns *NetStore) GetWithTimeout(addr Address, timeout time.Duration) (chunk *Chunk, err error) {
	return ns.get(addr, timeout)
}

func (ns *NetStore) get(addr Address, timeout time.Duration) (chunk *Chunk, err error) {
	if timeout == 0 {
		timeout = searchTimeout
	}
	if ns.retrieve == nil {
		chunk, err = ns.localStore.Get(addr)
		if err == nil {
			return chunk, nil
		}
		if err != ErrFetching {
			return nil, err
		}
	} else {
		var created bool
		chunk, created = ns.localStore.GetOrCreateRequest(addr)

		if chunk.ReqC == nil {
			return chunk, nil
		}

		if created {
			err := ns.retrieve(chunk)
			if err != nil {
				// mark chunk request as failed so that we can retry it later
				chunk.SetErrored(ErrChunkUnavailable)
				return nil, err
			}
		}
	}

	t := time.NewTicker(timeout)
	defer t.Stop()

	select {
	case <-t.C:
		// mark chunk request as failed so that we can retry
		chunk.SetErrored(ErrChunkNotFound)
		return nil, ErrChunkNotFound
	case <-chunk.ReqC:
	}
	chunk.SetErrored(nil)
	return chunk, nil
}

// Put is the entrypoint for local store requests coming from storeLoop
func (ns *NetStore) Put(chunk *Chunk) {
	ns.localStore.Put(chunk)
}

// Close chunk store
func (ns *NetStore) Close() {
	ns.localStore.Close()
}
