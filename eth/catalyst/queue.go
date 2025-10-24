// Copyright 2022 The go-ethereum Authors
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

package catalyst

import (
	"sync"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/miner"
)

// maxTrackedPayloads is the maximum number of prepared payloads the execution
// engine tracks before evicting old ones. Ideally we should only ever track the
// latest one; but have a slight wiggle room for non-ideal conditions.
const maxTrackedPayloads = 10

// maxTrackedHeaders is the maximum number of executed payloads the execution
// engine tracks before evicting old ones. These are tracked outside the chain
// during initial sync to allow ForkchoiceUpdate to reference past blocks via
// hashes only. For the sync target it would be enough to track only the latest
// header, but snap sync also needs the latest finalized height for the ancient
// limit.
const maxTrackedHeaders = 96

// maxTrackedInclusionLists is the maximum number of inclusion lists the execution
// engine tracks before evicting old ones.
const maxTrackedInclusionLists = 8

// payloadQueueItem represents an id->payload tuple to store until it's retrieved
// or evicted.
type payloadQueueItem struct {
	id      engine.PayloadID
	payload *miner.Payload
}

// payloadQueue tracks the latest handful of constructed payloads to be retrieved
// by the beacon chain if block production is requested.
type payloadQueue struct {
	payloads []*payloadQueueItem
	lock     sync.RWMutex
}

// newPayloadQueue creates a pre-initialized queue with a fixed number of slots
// all containing empty items.
func newPayloadQueue() *payloadQueue {
	return &payloadQueue{
		payloads: make([]*payloadQueueItem, maxTrackedPayloads),
	}
}

// put inserts a new payload into the queue at the given id.
func (q *payloadQueue) put(id engine.PayloadID, payload *miner.Payload) {
	q.lock.Lock()
	defer q.lock.Unlock()

	copy(q.payloads[1:], q.payloads)
	q.payloads[0] = &payloadQueueItem{
		id:      id,
		payload: payload,
	}
}

// get retrieves a previously stored payload item or nil if it does not exist.
func (q *payloadQueue) get(id engine.PayloadID, full bool) *engine.ExecutionPayloadEnvelope {
	q.lock.RLock()
	defer q.lock.RUnlock()

	for _, item := range q.payloads {
		if item == nil {
			return nil // no more items
		}
		if item.id == id {
			if !full {
				return item.payload.Resolve()
			}
			return item.payload.ResolveFull()
		}
	}
	return nil
}

// peek retrieves a previously stored payload itself or nil if it does not exist.
func (q *payloadQueue) peek(id engine.PayloadID) *miner.Payload {
	q.lock.RLock()
	defer q.lock.RUnlock()

	for _, item := range q.payloads {
		if item == nil {
			return nil // no more items
		}
		if item.id == id {
			return item.payload
		}
	}
	return nil
}

// has checks if a particular payload is already tracked.
func (q *payloadQueue) has(id engine.PayloadID) bool {
	q.lock.RLock()
	defer q.lock.RUnlock()

	for _, item := range q.payloads {
		if item == nil {
			return false
		}
		if item.id == id {
			return true
		}
	}
	return false
}

// headerQueueItem represents an hash->header tuple to store until it's retrieved
// or evicted.
type headerQueueItem struct {
	hash   common.Hash
	header *types.Header
}

// headerQueue tracks the latest handful of constructed headers to be retrieved
// by the beacon chain if block production is requested.
type headerQueue struct {
	headers []*headerQueueItem
	lock    sync.RWMutex
}

// newHeaderQueue creates a pre-initialized queue with a fixed number of slots
// all containing empty items.
func newHeaderQueue() *headerQueue {
	return &headerQueue{
		headers: make([]*headerQueueItem, maxTrackedHeaders),
	}
}

// put inserts a new header into the queue at the given hash.
func (q *headerQueue) put(hash common.Hash, data *types.Header) {
	q.lock.Lock()
	defer q.lock.Unlock()

	copy(q.headers[1:], q.headers)
	q.headers[0] = &headerQueueItem{
		hash:   hash,
		header: data,
	}
}

// get retrieves a previously stored header item or nil if it does not exist.
func (q *headerQueue) get(hash common.Hash) *types.Header {
	q.lock.RLock()
	defer q.lock.RUnlock()

	for _, item := range q.headers {
		if item == nil {
			return nil // no more items
		}
		if item.hash == hash {
			return item.header
		}
	}
	return nil
}

// inclusionListQueueItem represents an hash->inclusionList tuple to store until it's retrieved
// or evicted.
type inclusionListQueueItem struct {
	parentHash    common.Hash
	inclusionList types.InclusionList
}

// inclusionListQueue tracks the latest handful of constructed inclusion lists to be retrieved
// by the beacon chain if inclusion list production is requested.
type inclusionListQueue struct {
	inclusionLists []*inclusionListQueueItem
	lock           sync.RWMutex
}

// newinclusionListQueue creates a pre-initialized queue with a fixed number of slots
// all containing empty items.
func newInclusionListQueue() *inclusionListQueue {
	return &inclusionListQueue{
		inclusionLists: make([]*inclusionListQueueItem, maxTrackedInclusionLists),
	}
}

// put inserts a new inclusion list into the queue at the given parent hash that
// the inclusion list is built upon.
func (q *inclusionListQueue) put(parentHash common.Hash, inclusionList types.InclusionList) {
	q.lock.Lock()
	defer q.lock.Unlock()

	copy(q.inclusionLists[1:], q.inclusionLists)
	q.inclusionLists[0] = &inclusionListQueueItem{
		parentHash,
		inclusionList,
	}
}

// get retrieves a previously stored inclusion list item or nil if it does not exist.
func (q *inclusionListQueue) get(parentHash common.Hash) types.InclusionList {
	q.lock.RLock()
	defer q.lock.RUnlock()

	for _, item := range q.inclusionLists {
		if item == nil {
			return nil // no more items
		}
		if item.parentHash == parentHash {
			return item.inclusionList
		}
	}
	return nil
}
