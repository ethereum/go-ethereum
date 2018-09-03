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
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type Handler struct {
	chunkStore      *storage.NetStore
	HashSize        int
	resources       map[uint64]*resource
	resourceLock    sync.RWMutex
	storeTimeout    time.Duration
	queryMaxPeriods uint32
}

// HandlerParams pass parameters to the Handler constructor NewHandler
// Signer and TimestampProvider are mandatory parameters
type HandlerParams struct {
	QueryMaxPeriods uint32
}

// hashPool contains a pool of ready hashers
var hashPool sync.Pool
var minimumChunkLength int

// init initializes the package and hashPool
func init() {
	hashPool = sync.Pool{
		New: func() interface{} {
			return storage.MakeHashFunc(resourceHashAlgorithm)()
		},
	}
	if minimumMetadataLength < minimumUpdateDataLength {
		minimumChunkLength = minimumMetadataLength
	} else {
		minimumChunkLength = minimumUpdateDataLength
	}
}

// NewHandler creates a new Mutable Resource API
func NewHandler(params *HandlerParams) *Handler {
	rh := &Handler{
		resources:       make(map[uint64]*resource),
		storeTimeout:    defaultStoreTimeout,
		queryMaxPeriods: params.QueryMaxPeriods,
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(resourceHashAlgorithm)()
		if rh.HashSize == 0 {
			rh.HashSize = hashfunc.Size()
		}
		hashPool.Put(hashfunc)
	}

	return rh
}

// SetStore sets the store backend for the Mutable Resource API
func (h *Handler) SetStore(store *storage.NetStore) {
	h.chunkStore = store
}

// Validate is a chunk validation method
// If it looks like a resource update, the chunk address is checked against the ownerAddr of the update's signature
// It implements the storage.ChunkValidator interface
func (h *Handler) Validate(chunkAddr storage.Address, data []byte) bool {
	dataLength := len(data)
	if dataLength < minimumChunkLength || dataLength > chunk.DefaultSize+8 {
		return false
	}

	//metadata chunks have the first two bytes set to zero
	if data[0] == 0 && data[1] == 0 && dataLength >= minimumMetadataLength {
		//metadata chunk
		rootAddr, _ := metadataHash(data)
		valid := bytes.Equal(chunkAddr, rootAddr)
		if !valid {
			log.Debug("Invalid root metadata chunk with address", "addr", chunkAddr.Hex())
		}
		return valid
	}

	// if it is not a metadata chunk, check if it is a properly formatted update chunk with
	// valid signature and proof of ownership of the resource it is trying
	// to update

	// First, deserialize the chunk
	var r SignedResourceUpdate
	if err := r.fromChunk(chunkAddr, data); err != nil {
		log.Debug("Invalid resource chunk", "addr", chunkAddr.Hex(), "err", err.Error())
		return false
	}

	// check that the lookup information contained in the chunk matches the updateAddr (chunk search key)
	// that was used to retrieve this chunk
	// if this validation fails, someone forged a chunk.
	if !bytes.Equal(chunkAddr, r.updateHeader.UpdateAddr()) {
		log.Debug("period,version,rootAddr contained in update chunk do not match updateAddr", "addr", chunkAddr.Hex())
		return false
	}

	// Verify signatures and that the signer actually owns the resource
	// If it fails, it means either the signature is not valid, data is corrupted
	// or someone is trying to update someone else's resource.
	if err := r.Verify(); err != nil {
		log.Debug("Invalid signature", "err", err)
		return false
	}

	return true
}

// GetContent retrieves the data payload of the last synced update of the Mutable Resource
func (h *Handler) GetContent(rootAddr storage.Address) (storage.Address, []byte, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil || !rsrc.isSynced() {
		return nil, nil, NewError(ErrNotFound, " does not exist or is not synced")
	}
	return rsrc.lastKey, rsrc.data, nil
}

// GetLastPeriod retrieves the period of the last synced update of the Mutable Resource
func (h *Handler) GetLastPeriod(rootAddr storage.Address) (uint32, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.period, nil
}

// GetVersion retrieves the period of the last synced update of the Mutable Resource
func (h *Handler) GetVersion(rootAddr storage.Address) (uint32, error) {
	rsrc := h.get(rootAddr)
	if rsrc == nil {
		return 0, NewError(ErrNotFound, " does not exist")
	} else if !rsrc.isSynced() {
		return 0, NewError(ErrNotSynced, " is not synced")
	}
	return rsrc.version, nil
}

// New creates a new metadata chunk out of the request passed in.
func (h *Handler) New(ctx context.Context, request *Request) error {

	// frequency 0 is invalid
	if request.metadata.Frequency == 0 {
		return NewError(ErrInvalidValue, "frequency cannot be 0 when creating a resource")
	}

	// make sure owner is set to something
	if request.metadata.Owner == zeroAddr {
		return NewError(ErrInvalidValue, "ownerAddr must be set to create a new metadata chunk")
	}

	// create the meta chunk and store it in swarm
	chunk, metaHash, err := request.metadata.newChunk()
	if err != nil {
		return err
	}
	if request.metaHash != nil && !bytes.Equal(request.metaHash, metaHash) ||
		request.rootAddr != nil && !bytes.Equal(request.rootAddr, chunk.Addr) {
		return NewError(ErrInvalidValue, "metaHash in UpdateRequest does not match actual metadata")
	}

	request.metaHash = metaHash
	request.rootAddr = chunk.Addr

	h.chunkStore.Put(ctx, chunk)
	log.Debug("new resource", "name", request.metadata.Name, "startTime", request.metadata.StartTime, "frequency", request.metadata.Frequency, "owner", request.metadata.Owner)

	// create the internal index for the resource and populate it with its metadata
	rsrc := &resource{
		resourceUpdate: resourceUpdate{
			updateHeader: updateHeader{
				UpdateLookup: UpdateLookup{
					rootAddr: chunk.Addr,
				},
			},
		},
		ResourceMetadata: request.metadata,
		updated:          time.Now(),
	}
	h.set(chunk.Addr, rsrc)

	return nil
}

// NewUpdateRequest prepares an UpdateRequest structure with all the necessary information to
// just add the desired data and sign it.
// The resulting structure can then be signed and passed to Handler.Update to be verified and sent
func (h *Handler) NewUpdateRequest(ctx context.Context, rootAddr storage.Address) (updateRequest *Request, err error) {

	if rootAddr == nil {
		return nil, NewError(ErrInvalidValue, "rootAddr cannot be nil")
	}

	// Make sure we have a cache of the metadata chunk
	rsrc, err := h.Load(ctx, rootAddr)
	if err != nil {
		return nil, err
	}

	now := TimestampProvider.Now()

	updateRequest = new(Request)
	updateRequest.period, err = getNextPeriod(rsrc.StartTime.Time, now.Time, rsrc.Frequency)
	if err != nil {
		return nil, err
	}

	if _, err = h.lookup(rsrc, LookupLatestVersionInPeriod(rsrc.rootAddr, updateRequest.period)); err != nil {
		if err.(*Error).code != ErrNotFound {
			return nil, err
		}
		// not finding updates means that there is a network error
		// or that the resource really does not have updates in this period.
	}

	updateRequest.multihash = rsrc.multihash
	updateRequest.rootAddr = rsrc.rootAddr
	updateRequest.metaHash = rsrc.metaHash
	updateRequest.metadata = rsrc.ResourceMetadata

	// if we already have an update for this period then increment version
	// resource object MUST be in sync for version to be correct, but we checked this earlier in the method already
	if h.hasUpdate(rootAddr, updateRequest.period) {
		updateRequest.version = rsrc.version + 1
	} else {
		updateRequest.version = 1
	}

	return updateRequest, nil
}

// Lookup retrieves a specific or latest version of the resource update with metadata chunk at params.Root
// Lookup works differently depending on the configuration of `LookupParams`
// See the `LookupParams` documentation and helper functions:
// `LookupLatest`, `LookupLatestVersionInPeriod` and `LookupVersion`
// When looking for the latest update, it starts at the next period after the current time.
// upon failure tries the corresponding keys of each previous period until one is found
// (or startTime is reached, in which case there are no updates).
func (h *Handler) Lookup(ctx context.Context, params *LookupParams) (*resource, error) {

	rsrc := h.get(params.rootAddr)
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	return h.lookup(rsrc, params)
}

// LookupPrevious returns the resource before the one currently loaded in the resource cache
// This is useful where resource updates are used incrementally in contrast to
// merely replacing content.
// Requires a cached resource object to determine the current state of the resource.
func (h *Handler) LookupPrevious(ctx context.Context, params *LookupParams) (*resource, error) {
	rsrc := h.get(params.rootAddr)
	if rsrc == nil {
		return nil, NewError(ErrNothingToReturn, "resource not loaded")
	}
	if !rsrc.isSynced() {
		return nil, NewError(ErrNotSynced, "LookupPrevious requires synced resource.")
	} else if rsrc.period == 0 {
		return nil, NewError(ErrNothingToReturn, " not found")
	}
	var version, period uint32
	if rsrc.version > 1 {
		version = rsrc.version - 1
		period = rsrc.period
	} else if rsrc.period == 1 {
		return nil, NewError(ErrNothingToReturn, "Current update is the oldest")
	} else {
		version = 0
		period = rsrc.period - 1
	}
	return h.lookup(rsrc, NewLookupParams(rsrc.rootAddr, period, version, params.Limit))
}

// base code for public lookup methods
func (h *Handler) lookup(rsrc *resource, params *LookupParams) (*resource, error) {

	lp := *params
	// we can't look for anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before performing lookups")
	}

	var specificperiod bool
	if lp.period > 0 {
		specificperiod = true
	} else {
		// get the current time and the next period
		now := TimestampProvider.Now()

		var period uint32
		period, err := getNextPeriod(rsrc.StartTime.Time, now.Time, rsrc.Frequency)
		if err != nil {
			return nil, err
		}
		lp.period = period
	}

	// start from the last possible period, and iterate previous ones
	// (unless we want a specific period only) until we find a match.
	// If we hit startTime we're out of options
	var specificversion bool
	if lp.version > 0 {
		specificversion = true
	} else {
		lp.version = 1
	}

	var hops uint32
	if lp.Limit == 0 {
		lp.Limit = h.queryMaxPeriods
	}
	log.Trace("resource lookup", "period", lp.period, "version", lp.version, "limit", lp.Limit)
	for lp.period > 0 {
		if lp.Limit != 0 && hops > lp.Limit {
			return nil, NewErrorf(ErrPeriodDepth, "Lookup exceeded max period hops (%d)", lp.Limit)
		}
		updateAddr := lp.UpdateAddr()
		chunk, err := h.chunkStore.GetWithTimeout(context.TODO(), updateAddr, defaultRetrieveTimeout)
		if err == nil {
			if specificversion {
				return h.updateIndex(rsrc, chunk)
			}
			// check if we have versions > 1. If a version fails, the previous version is used and returned.
			log.Trace("rsrc update version 1 found, checking for version updates", "period", lp.period, "updateAddr", updateAddr)
			for {
				newversion := lp.version + 1
				updateAddr := lp.UpdateAddr()
				newchunk, err := h.chunkStore.GetWithTimeout(context.TODO(), updateAddr, defaultRetrieveTimeout)
				if err != nil {
					return h.updateIndex(rsrc, chunk)
				}
				chunk = newchunk
				lp.version = newversion
				log.Trace("version update found, checking next", "version", lp.version, "period", lp.period, "updateAddr", updateAddr)
			}
		}
		if specificperiod {
			break
		}
		log.Trace("rsrc update not found, checking previous period", "period", lp.period, "updateAddr", updateAddr)
		lp.period--
		hops++
	}
	return nil, NewError(ErrNotFound, "no updates found")
}

// Load retrieves the Mutable Resource metadata chunk stored at rootAddr
// Upon retrieval it creates/updates the index entry for it with metadata corresponding to the chunk contents
func (h *Handler) Load(ctx context.Context, rootAddr storage.Address) (*resource, error) {
	chunk, err := h.chunkStore.GetWithTimeout(ctx, rootAddr, defaultRetrieveTimeout)
	if err != nil {
		return nil, NewError(ErrNotFound, err.Error())
	}

	// create the index entry
	rsrc := &resource{}

	if err := rsrc.ResourceMetadata.binaryGet(chunk.SData); err != nil { // Will fail if this is not really a metadata chunk
		return nil, err
	}

	rsrc.rootAddr, rsrc.metaHash = metadataHash(chunk.SData)
	if !bytes.Equal(rsrc.rootAddr, rootAddr) {
		return nil, NewError(ErrCorruptData, "Corrupt metadata chunk")
	}
	h.set(rootAddr, rsrc)
	log.Trace("resource index load", "rootkey", rootAddr, "name", rsrc.ResourceMetadata.Name, "starttime", rsrc.ResourceMetadata.StartTime, "frequency", rsrc.ResourceMetadata.Frequency)
	return rsrc, nil
}

// update mutable resource index map with specified content
func (h *Handler) updateIndex(rsrc *resource, chunk *storage.Chunk) (*resource, error) {

	// retrieve metadata from chunk data and check that it matches this mutable resource
	var r SignedResourceUpdate
	if err := r.fromChunk(chunk.Addr, chunk.SData); err != nil {
		return nil, err
	}
	log.Trace("resource index update", "name", rsrc.ResourceMetadata.Name, "updatekey", chunk.Addr, "period", r.period, "version", r.version)

	// update our rsrcs entry map
	rsrc.lastKey = chunk.Addr
	rsrc.period = r.period
	rsrc.version = r.version
	rsrc.updated = time.Now()
	rsrc.data = make([]byte, len(r.data))
	rsrc.multihash = r.multihash
	copy(rsrc.data, r.data)
	rsrc.Reader = bytes.NewReader(rsrc.data)
	log.Debug("resource synced", "name", rsrc.ResourceMetadata.Name, "updateAddr", chunk.Addr, "period", rsrc.period, "version", rsrc.version)
	h.set(chunk.Addr, rsrc)
	return rsrc, nil
}

// Update adds an actual data update
// Uses the Mutable Resource metadata currently loaded in the resources map entry.
// It is the caller's responsibility to make sure that this data is not stale.
// Note that a Mutable Resource update cannot span chunks, and thus has a MAX NET LENGTH 4096, INCLUDING update header data and signature. An error will be returned if the total length of the chunk payload will exceed this limit.
// Update can only check if the caller is trying to overwrite the very last known version, otherwise it just puts the update
// on the network.
func (h *Handler) Update(ctx context.Context, r *SignedResourceUpdate) (storage.Address, error) {
	return h.update(ctx, r)
}

// create and commit an update
func (h *Handler) update(ctx context.Context, r *SignedResourceUpdate) (updateAddr storage.Address, err error) {

	// we can't update anything without a store
	if h.chunkStore == nil {
		return nil, NewError(ErrInit, "Call Handler.SetStore() before updating")
	}

	rsrc := h.get(r.rootAddr)
	if rsrc != nil && rsrc.period != 0 && rsrc.version != 0 && // This is the only cheap check we can do for sure
		rsrc.period == r.period && rsrc.version >= r.version { // without having to lookup update chunks

		return nil, NewError(ErrInvalidValue, "A former update in this period is already known to exist")
	}

	chunk, err := r.toChunk() // Serialize the update into a chunk. Fails if data is too big
	if err != nil {
		return nil, err
	}

	// send the chunk
	h.chunkStore.Put(ctx, chunk)
	log.Trace("resource update", "updateAddr", r.updateAddr, "lastperiod", r.period, "version", r.version, "data", chunk.SData, "multihash", r.multihash)

	// update our resources map entry if the new update is older than the one we have, if we have it.
	if rsrc != nil && (r.period > rsrc.period || (rsrc.period == r.period && r.version > rsrc.version)) {
		rsrc.period = r.period
		rsrc.version = r.version
		rsrc.data = make([]byte, len(r.data))
		rsrc.updated = time.Now()
		rsrc.lastKey = r.updateAddr
		rsrc.multihash = r.multihash
		copy(rsrc.data, r.data)
		rsrc.Reader = bytes.NewReader(rsrc.data)
	}
	return r.updateAddr, nil
}

// Retrieves the resource index value for the given nameHash
func (h *Handler) get(rootAddr storage.Address) *resource {
	if len(rootAddr) < storage.KeyLength {
		log.Warn("Handler.get with invalid rootAddr")
		return nil
	}
	hashKey := *(*uint64)(unsafe.Pointer(&rootAddr[0]))
	h.resourceLock.RLock()
	defer h.resourceLock.RUnlock()
	rsrc := h.resources[hashKey]
	return rsrc
}

// Sets the resource index value for the given nameHash
func (h *Handler) set(rootAddr storage.Address, rsrc *resource) {
	if len(rootAddr) < storage.KeyLength {
		log.Warn("Handler.set with invalid rootAddr")
		return
	}
	hashKey := *(*uint64)(unsafe.Pointer(&rootAddr[0]))
	h.resourceLock.Lock()
	defer h.resourceLock.Unlock()
	h.resources[hashKey] = rsrc
}

// Checks if we already have an update on this resource, according to the value in the current state of the resource index
func (h *Handler) hasUpdate(rootAddr storage.Address, period uint32) bool {
	rsrc := h.get(rootAddr)
	return rsrc != nil && rsrc.period == period
}
