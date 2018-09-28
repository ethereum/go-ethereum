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

// Handler is the API for Mutable Resources
// It enables creating, updating, syncing and retrieving resources and their update data
package mru

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage/mru/lookup"

	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type Handler struct {
	chunkStore      *storage.NetStore
	HashSize        int
	cache           map[uint64]*cacheEntry
	cacheLock       sync.RWMutex
	storeTimeout    time.Duration
	queryMaxPeriods uint32
}

// HandlerParams pass parameters to the Handler constructor NewHandler
// Signer and TimestampProvider are mandatory parameters
type HandlerParams struct {
}

// hashPool contains a pool of ready hashers
var hashPool sync.Pool

// init initializes the package and hashPool
func init() {
	hashPool = sync.Pool{
		New: func() interface{} {
			return storage.MakeHashFunc(feedsHashAlgorithm)()
		},
	}
}

// NewHandler creates a new Mutable Resource API
func NewHandler(params *HandlerParams) *Handler {
	fh := &Handler{
		cache: make(map[uint64]*cacheEntry),
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(feedsHashAlgorithm)()
		if fh.HashSize == 0 {
			fh.HashSize = hashfunc.Size()
		}
		hashPool.Put(hashfunc)
	}

	return fh
}

// SetStore sets the store backend for the Mutable Resource API
func (h *Handler) SetStore(store *storage.NetStore) {
	h.chunkStore = store
}

// Validate is a chunk validation method
// If it looks like a resource update, the chunk address is checked against the userAddr of the update's signature
// It implements the storage.ChunkValidator interface
func (h *Handler) Validate(chunkAddr storage.Address, data []byte) bool {
	dataLength := len(data)
	if dataLength < minimumSignedUpdateLength {
		return false
	}

	// check if it is a properly formatted update chunk with
	// valid signature and proof of ownership of the resource it is trying
	// to update

	// First, deserialize the chunk
	var r Request
	if err := r.fromChunk(chunkAddr, data); err != nil {
		log.Debug("Invalid feed update chunk", "addr", chunkAddr.Hex(), "err", err.Error())
		return false
	}

	// Verify signatures and that the signer actually owns the resource
	// If it fails, it means either the signature is not valid, data is corrupted
	// or someone is trying to update someone else's resource.
	if err := r.Verify(); err != nil {
		log.Debug("Invalid feed update signature", "err", err)
		return false
	}

	return true
}

// GetContent retrieves the data payload of the last synced update of the Mutable Resource
func (h *Handler) GetContent(feed *Feed) (storage.Address, []byte, error) {
	if feed == nil {
		return nil, nil, NewError(ErrInvalidValue, "view is nil")
	}
	feedUpdate := h.get(feed)
	if feedUpdate == nil {
		return nil, nil, NewError(ErrNotFound, "resource does not exist")
	}
	return feedUpdate.lastKey, feedUpdate.data, nil
}

// NewRequest prepares a Request structure with all the necessary information to
// just add the desired data and sign it.
// The resulting structure can then be signed and passed to Handler.Update to be verified and sent
func (h *Handler) NewRequest(ctx context.Context, feed *Feed) (request *Request, err error) {
	if feed == nil {
		return nil, NewError(ErrInvalidValue, "feed cannot be nil")
	}

	now := TimestampProvider.Now().Time
	request = new(Request)
	request.Header.Version = ProtocolVersion

	query := NewQueryLatest(feed, lookup.NoClue)

	feedUpdate, err := h.Lookup(ctx, query)
	if err != nil {
		if err.(*Error).code != ErrNotFound {
			return nil, err
		}
		// not finding updates means that there is a network error
		// or that the resource really does not have updates
	}

	request.Feed = *feed

	// if we already have an update, then find next epoch
	if feedUpdate != nil {
		request.Epoch = lookup.GetNextEpoch(feedUpdate.Epoch, now)
	} else {
		request.Epoch = lookup.GetFirstEpoch(now)
	}

	return request, nil
}

// Lookup retrieves a specific or latest version of the resource
// Lookup works differently depending on the configuration of `ID`
// See the `ID` documentation and helper functions:
// `LookupLatest` and `LookupBefore`
// When looking for the latest update, it starts at the next period after the current time.
// upon failure tries the corresponding keys of each previous period until one is found
// (or startTime is reached, in which case there are no updates).
func (h *Handler) Lookup(ctx context.Context, query *Query) (*cacheEntry, error) {

	timeLimit := query.TimeLimit
	if timeLimit == 0 { // if time limit is set to zero, the user wants to get the latest update
		timeLimit = TimestampProvider.Now().Time
	}

	if query.Hint == lookup.NoClue { // try to use our cache
		entry := h.get(&query.Feed)
		if entry != nil && entry.Epoch.Time <= timeLimit { // avoid bad hints
			query.Hint = entry.Epoch
		}
	}

	// we can't look for anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before performing lookups")
	}

	var id ID
	id.Feed = query.Feed
	var readCount int

	// Invoke the lookup engine.
	// The callback will be called every time the lookup algorithm needs to guess
	requestPtr, err := lookup.Lookup(timeLimit, query.Hint, func(epoch lookup.Epoch, now uint64) (interface{}, error) {
		readCount++
		id.Epoch = epoch
		ctx, cancel := context.WithTimeout(ctx, defaultRetrieveTimeout)
		defer cancel()

		chunk, err := h.chunkStore.Get(ctx, id.Addr())
		if err != nil { // TODO: check for catastrophic errors other than chunk not found
			return nil, nil
		}

		var request Request
		if err := request.fromChunk(chunk.Address(), chunk.Data()); err != nil {
			return nil, nil
		}
		if request.Time <= timeLimit {
			return &request, nil
		}
		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	log.Info(fmt.Sprintf("Resource lookup finished in %d lookups", readCount))

	request, _ := requestPtr.(*Request)
	if request == nil {
		return nil, NewError(ErrNotFound, "no updates found")
	}
	return h.updateCache(request)

}

// update mutable resource cache map with specified content
func (h *Handler) updateCache(request *Request) (*cacheEntry, error) {

	updateAddr := request.Addr()
	log.Trace("feed cache update", "topic", request.Topic.Hex(), "updateaddr", updateAddr, "epoch time", request.Epoch.Time, "epoch level", request.Epoch.Level)

	feedUpdate := h.get(&request.Feed)
	if feedUpdate == nil {
		feedUpdate = &cacheEntry{}
		h.set(&request.Feed, feedUpdate)
	}

	// update our rsrcs entry map
	feedUpdate.lastKey = updateAddr
	feedUpdate.Update = request.Update
	feedUpdate.Reader = bytes.NewReader(feedUpdate.data)
	return feedUpdate, nil
}

// Update adds an actual data update
// Uses the Mutable Resource metadata currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
// Note that a Mutable Resource update cannot span chunks, and thus has a MAX NET LENGTH 4096, INCLUDING update header data and signature. An error will be returned if the total length of the chunk payload will exceed this limit.
// Update can only check if the caller is trying to overwrite the very last known version, otherwise it just puts the update
// on the network.
func (h *Handler) Update(ctx context.Context, r *Request) (updateAddr storage.Address, err error) {

	// we can't update anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before updating")
	}

	feedUpdate := h.get(&r.Feed)
	if feedUpdate != nil && feedUpdate.Epoch.Equals(r.Epoch) { // This is the only cheap check we can do for sure
		return nil, NewError(ErrInvalidValue, "A former update in this epoch is already known to exist")
	}

	chunk, err := r.toChunk() // Serialize the update into a chunk. Fails if data is too big
	if err != nil {
		return nil, err
	}

	// send the chunk
	h.chunkStore.Put(ctx, chunk)
	log.Trace("feed update", "updateAddr", r.idAddr, "epoch time", r.Epoch.Time, "epoch level", r.Epoch.Level, "data", chunk.Data())
	// update our resources map cache entry if the new update is older than the one we have, if we have it.
	if feedUpdate != nil && r.Epoch.After(feedUpdate.Epoch) {
		feedUpdate.Epoch = r.Epoch
		feedUpdate.data = make([]byte, len(r.data))
		feedUpdate.lastKey = r.idAddr
		copy(feedUpdate.data, r.data)
		feedUpdate.Reader = bytes.NewReader(feedUpdate.data)
	}

	return r.idAddr, nil
}

// Retrieves the resource cache value for the given nameHash
func (h *Handler) get(view *Feed) *cacheEntry {
	mapKey := view.mapKey()
	h.cacheLock.RLock()
	defer h.cacheLock.RUnlock()
	feedUpdate := h.cache[mapKey]
	return feedUpdate
}

// Sets the resource cache value for the given View
func (h *Handler) set(view *Feed, feedUpdate *cacheEntry) {
	mapKey := view.mapKey()
	h.cacheLock.Lock()
	defer h.cacheLock.Unlock()
	h.cache[mapKey] = feedUpdate
}
