// Copyright 2019 The go-ethereum Authors
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

package localstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// SubscribePull returns a channel that provides chunk addresses and stored times from pull syncing index.
// Pull syncing index can be only subscribed to a particular proximity order bin. If since
// is not nil, the iteration will start from the first item stored after that timestamp. If until is not nil,
// only chunks stored up to this timestamp will be send to the channel, and the returned channel will be
// closed. The since-until interval is open on the left and closed on the right (since,until]. Returned stop
// function will terminate current and further iterations without errors, and also close the returned channel.
// Make sure that you check the second returned parameter from the channel to stop iteration when its value
// is false.
func (db *DB) SubscribePull(ctx context.Context, bin uint8, since, until *ChunkDescriptor) (c <-chan ChunkDescriptor, stop func()) {
	chunkDescriptors := make(chan ChunkDescriptor)
	trigger := make(chan struct{}, 1)

	db.pullTriggersMu.Lock()
	if _, ok := db.pullTriggers[bin]; !ok {
		db.pullTriggers[bin] = make([]chan struct{}, 0)
	}
	db.pullTriggers[bin] = append(db.pullTriggers[bin], trigger)
	db.pullTriggersMu.Unlock()

	// send signal for the initial iteration
	trigger <- struct{}{}

	stopChan := make(chan struct{})
	var stopChanOnce sync.Once

	// used to provide information from the iterator to
	// stop subscription when until chunk descriptor is reached
	var errStopSubscription = errors.New("stop subscription")

	go func() {
		// close the returned ChunkDescriptor channel at the end to
		// signal that the subscription is done
		defer close(chunkDescriptors)
		// sinceItem is the Item from which the next iteration
		// should start. The first iteration starts from the first Item.
		var sinceItem *shed.Item
		if since != nil {
			sinceItem = &shed.Item{
				Address:        since.Address,
				StoreTimestamp: since.StoreTimestamp,
			}
		}
		for {
			select {
			case <-trigger:
				// iterate until:
				// - last index Item is reached
				// - subscription stop is called
				// - context is done
				err := db.pullIndex.Iterate(func(item shed.Item) (stop bool, err error) {
					select {
					case chunkDescriptors <- ChunkDescriptor{
						Address:        item.Address,
						StoreTimestamp: item.StoreTimestamp,
					}:
						// until chunk descriptor is sent
						// break the iteration
						if until != nil &&
							(item.StoreTimestamp >= until.StoreTimestamp ||
								bytes.Equal(item.Address, until.Address)) {
							return true, errStopSubscription
						}
						// set next iteration start item
						// when its chunk is successfully sent to channel
						sinceItem = &item
						return false, nil
					case <-stopChan:
						// gracefully stop the iteration
						// on stop
						return true, nil
					case <-db.close:
						// gracefully stop the iteration
						// on database close
						return true, nil
					case <-ctx.Done():
						return true, ctx.Err()
					}
				}, &shed.IterateOptions{
					StartFrom: sinceItem,
					// sinceItem was sent as the last Address in the previous
					// iterator call, skip it in this one
					SkipStartFromItem: true,
					Prefix:            []byte{bin},
				})
				if err != nil {
					if err == errStopSubscription {
						// stop subscription without any errors
						// if until is reached
						return
					}
					log.Error("localstore pull subscription iteration", "bin", bin, "since", since, "until", until, "err", err)
					return
				}
			case <-stopChan:
				// terminate the subscription
				// on stop
				return
			case <-db.close:
				// terminate the subscription
				// on database close
				return
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil {
					log.Error("localstore pull subscription", "bin", bin, "since", since, "until", until, "err", err)
				}
				return
			}
		}
	}()

	stop = func() {
		stopChanOnce.Do(func() {
			close(stopChan)
		})

		db.pullTriggersMu.Lock()
		defer db.pullTriggersMu.Unlock()

		for i, t := range db.pullTriggers[bin] {
			if t == trigger {
				db.pullTriggers[bin] = append(db.pullTriggers[bin][:i], db.pullTriggers[bin][i+1:]...)
				break
			}
		}
	}

	return chunkDescriptors, stop
}

// ChunkDescriptor holds information required for Pull syncing. This struct
// is provided by subscribing to pull index.
type ChunkDescriptor struct {
	Address        storage.Address
	StoreTimestamp int64
}

func (c *ChunkDescriptor) String() string {
	if c == nil {
		return "none"
	}
	return fmt.Sprintf("%s stored at %v", c.Address.Hex(), c.StoreTimestamp)
}

// triggerPullSubscriptions is used internally for starting iterations
// on Pull subscriptions for a particular bin. When new item with address
// that is in particular bin for DB's baseKey is added to pull index
// this function should be called.
func (db *DB) triggerPullSubscriptions(bin uint8) {
	db.pullTriggersMu.RLock()
	triggers, ok := db.pullTriggers[bin]
	db.pullTriggersMu.RUnlock()
	if !ok {
		return
	}

	for _, t := range triggers {
		select {
		case t <- struct{}{}:
		default:
		}
	}
}
