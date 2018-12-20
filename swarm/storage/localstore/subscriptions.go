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

package localstore

import (
	"bytes"
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var ErrSubscriptionFeedClosed = errors.New("subscription feed closed")

// SubscribePull returns a Subscription for pull syncing index.
// Pull syncing index can be only subscribed to a particular
// proximity order bin.
func (db *DB) SubscribePull(ctx context.Context, bin uint8) (s *Subscription, err error) {
	return db.pullFeed.subscribe(ctx, []byte{bin})
}

// SubscribePush returns a Subscription for push syncing index.
func (db *DB) SubscribePush(ctx context.Context) (s *Subscription, err error) {
	return db.pushFeed.subscribe(ctx, nil)
}

// Subscription provides stream of Chunks in a particular order
// through the Chunks channel. That channel will not be closed
// when the last Chunk is read, but will block until the new Chunk
// is added to database index. Subscription should be used for
// getting Chunks and waiting for new ones. It provides methods
// to control and get information about subscription state.
type Subscription struct {
	// Chunks is the read-only channel that provides stream of chunks.
	// This is the subscription main purpose.
	Chunks <-chan storage.Chunk

	// subscribe to set of keys only with this prefix
	prefix []byte
	// signals subscription to gracefully stop
	stopChan chan struct{}
	// protects stopChan form multiple closing
	stopOnce sync.Once
	// provides information if subscription is done
	doneChan chan struct{}
	// trigger signals a new index iteration
	// when index receives new items
	trigger chan struct{}
	// an error from the subscription, if any
	err error
	// protects err field
	mu sync.RWMutex
}

// Done returns a read-only channel that will be closed
// when the subscription is stopped or encountered an error.
func (s *Subscription) Done() <-chan struct{} {
	return s.doneChan
}

// Err returns an error that subscription encountered.
// It should be usually called after the Done is read from.
// It is safe to call this function multiple times.
func (s *Subscription) Err() (err error) {
	s.mu.RLock()
	err = s.err
	s.mu.RUnlock()
	return err
}

// Stop terminates the subscription without any error.
// It is safe to call this function multiple times.
func (s *Subscription) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopChan)
	})
}

// feed is a collection of Chunks subscriptions of order given
// by sort index and Chunk data provided by data index.
// It provides methods to create, trigger and remove subscriptions.
// It is the internal core component for push and pull
// index subscriptions.
type feed struct {
	// index on which keys the order of Chunks will be
	// provided by subscriptions
	sortIndex shed.Index
	// index that contains chunk data
	dataIndex shed.Index
	// collection fo subscriptions on this feed
	subscriptions []*Subscription
	// protects subscriptions slice
	mu sync.Mutex
	// closed when subscription is closed
	closeChan chan struct{}
	// protects closeChan form multiple closing
	closeOnce sync.Once
}

// newFeed creates a new feed with from sort and data indexes.
// Sort index provides ordering of Chunks and data index
// provides Chunk data.
func newFeed(sortIndex, dataIndex shed.Index) (f *feed) {
	return &feed{
		sortIndex:     sortIndex,
		dataIndex:     dataIndex,
		subscriptions: make([]*Subscription, 0),
		closeChan:     make(chan struct{}),
	}
}

// subscribe creates a new subscription on the feed.
// It creates a new goroutine which will iterate over existing sort index keys
// and creates new iterators when trigger method is called.
func (f *feed) subscribe(ctx context.Context, prefix []byte) (s *Subscription, err error) {
	// prevent new subscription after the feed is closed
	select {
	case <-f.closeChan:
		return nil, ErrSubscriptionFeedClosed
	default:
	}
	chunks := make(chan storage.Chunk)
	s = &Subscription{
		Chunks:   chunks,
		prefix:   prefix,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		trigger:  make(chan struct{}, 1),
	}
	f.mu.Lock()
	f.subscriptions = append(f.subscriptions, s)
	f.mu.Unlock()

	// send signal for the initial iteration
	s.trigger <- struct{}{}

	go func() {
		// this error will be set in deferred unsubscribe
		// function call and set as Subscription.err value
		var err error
		defer func() {
			f.unsubscribe(s, err)
		}()

		// startFrom is the Item from which the next iteration
		// should start. The first iteration starts from the first Item.
		var startFrom *shed.Item
		for {
			select {
			case <-s.trigger:
				// iterate until:
				// - last index Item is reached
				// - subscription stop is called
				// - context is done
				err = f.sortIndex.Iterate(func(item shed.Item) (stop bool, err error) {
					// get chunk data
					dataItem, err := f.dataIndex.Get(item)
					if err != nil {
						return true, err
					}

					select {
					case chunks <- storage.NewChunk(dataItem.Address, dataItem.Data):
						// set next iteration start item
						// when its chunk is successfully sent to channel
						startFrom = &item
						return false, nil
					case <-s.stopChan:
						// gracefully stop the iteration
						return true, nil
					case <-ctx.Done():
						return true, ctx.Err()
					}
				}, &shed.IterateOptions{
					StartFrom: startFrom,
					// startFrom was sent as the last Chunk in the previous
					// iterator call, skip it in this one
					SkipStartFromItem: true,
					Prefix:            prefix,
				})
				if err != nil {
					return
				}
			case <-s.stopChan:
				// gracefully stop the iteration
				return
			case <-ctx.Done():
				if err == nil {
					err = ctx.Err()
				}
				return
			}
		}
	}()

	return s, nil
}

// unsubscribe removes a subscription from the feed.
// This function is called when subscription goroutine terminates
// to cleanup feed subscriptions and set error on subscription.
func (f *feed) unsubscribe(s *Subscription, err error) {
	s.mu.Lock()
	s.err = err
	s.mu.Unlock()

	f.mu.Lock()
	defer f.mu.Unlock()
	for i, sub := range f.subscriptions {
		if sub == s {
			f.subscriptions = append(f.subscriptions[:i], f.subscriptions[i+1:]...)
		}
	}

	// signal that the subscription is done
	close(s.doneChan)
}

// close stops all subscriptions and prevents any new subscriptions
// to be made by closing the closeChan.
func (f *feed) close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range f.subscriptions {
		s.Stop()
	}
	f.closeOnce.Do(func() {
		close(f.closeChan)
	})
}

// trigger signals all subscriptions with tprovided prefix
// that they should continue iterating over index keys
// where they stopped in the last iteration. This method
// should be called when new data is put to the index.
func (f *feed) trigger(prefix []byte) {
	for _, s := range f.subscriptions {
		if bytes.Equal(prefix, s.prefix) {
			select {
			case s.trigger <- struct{}{}:
			default:
			}
		}
	}
}
