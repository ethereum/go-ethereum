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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
)

// maxTrackedPayloads is the maximum number of prepared payloads the execution
// engine tracks before evicting old ones. Ideally we should only ever track the
// latest one; but have a slight wiggle room for non-ideal conditions.
const maxTrackedPayloads = 10

// maxTrackedHeaders is the maximum number of executed payloads the execution
// engine tracks before evicting old ones. Ideally we should only ever track the
// latest one; but have a slight wiggle room for non-ideal conditions.
const maxTrackedHeaders = 10

// payloadQueueItem represents an id->payload tuple to store until it's retrieved
// or evicted.
type payloadQueueItem struct {
	id      beacon.PayloadID
	payload *beacon.ExecutableDataV1
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
func (q *payloadQueue) put(id beacon.PayloadID, data *beacon.ExecutableDataV1) {
	q.lock.Lock()
	defer q.lock.Unlock()

	copy(q.payloads[1:], q.payloads)
	q.payloads[0] = &payloadQueueItem{
		id:      id,
		payload: data,
	}
}

// get retrieves a previously stored payload item or nil if it does not exist.
func (q *payloadQueue) get(id beacon.PayloadID) *beacon.ExecutableDataV1 {
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
