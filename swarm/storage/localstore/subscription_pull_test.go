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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/shed"
)

// TestDB_SubscribePull uploads some chunks before and after
// pull syncing subscription is created and validates if
// all addresses are received in the right order
// for expected proximity order bins.
func TestDB_SubscribePull(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)
	var addrsMu sync.Mutex
	var wantedChunksCount int

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
		ch, stop := db.SubscribePull(ctx, bin, 0, 0)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, db, bin, ch, addrs, &addrsMu, errChan)
	}

	// upload some chunks just after subscribe
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 5)

	time.Sleep(200 * time.Millisecond)

	// upload some chunks after some short time
	// to ensure that subscription will include them
	// in a dynamic environment
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 3)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// TestDB_SubscribePull_multiple uploads chunks before and after
// multiple pull syncing subscriptions are created and
// validates if all addresses are received in the right order
// for expected proximity order bins.
func TestDB_SubscribePull_multiple(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)
	var addrsMu sync.Mutex
	var wantedChunksCount int

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	subsCount := 10

	// start a number of subscriptions
	// that all of them will write every address error to errChan
	for j := 0; j < subsCount; j++ {
		for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
			ch, stop := db.SubscribePull(ctx, bin, 0, 0)
			defer stop()

			// receive and validate addresses from the subscription
			go readPullSubscriptionBin(ctx, db, bin, ch, addrs, &addrsMu, errChan)
		}
	}

	// upload some chunks just after subscribe
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 5)

	time.Sleep(200 * time.Millisecond)

	// upload some chunks after some short time
	// to ensure that subscription will include them
	// in a dynamic environment
	uploadRandomChunksBin(t, db, addrs, &addrsMu, &wantedChunksCount, 3)

	checkErrChan(ctx, t, errChan, wantedChunksCount*subsCount)
}

// TestDB_SubscribePull_since uploads chunks before and after
// pull syncing subscriptions are created with a since argument
// and validates if all expected addresses are received in the
// right order for expected proximity order bins.
func TestDB_SubscribePull_since(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)
	var addrsMu sync.Mutex
	var wantedChunksCount int

	binIDCounter := make(map[uint8]uint64)
	var binIDCounterMu sync.RWMutex

	uploadRandomChunks := func(count int, wanted bool) (first map[uint8]uint64) {
		addrsMu.Lock()
		defer addrsMu.Unlock()

		first = make(map[uint8]uint64)
		for i := 0; i < count; i++ {
			ch := generateTestRandomChunk()

			_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(ch.Address())

			binIDCounterMu.RLock()
			binIDCounter[bin]++
			binIDCounterMu.RUnlock()

			if wanted {
				if _, ok := addrs[bin]; !ok {
					addrs[bin] = make([]chunk.Address, 0)
				}
				addrs[bin] = append(addrs[bin], ch.Address())
				wantedChunksCount++

				if _, ok := first[bin]; !ok {
					first[bin] = binIDCounter[bin]
				}
			}
		}
		return first
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(30, false)

	first := uploadRandomChunks(25, true)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
		since, ok := first[bin]
		if !ok {
			continue
		}
		ch, stop := db.SubscribePull(ctx, bin, since, 0)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, db, bin, ch, addrs, &addrsMu, errChan)

	}

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// TestDB_SubscribePull_until uploads chunks before and after
// pull syncing subscriptions are created with an until argument
// and validates if all expected addresses are received in the
// right order for expected proximity order bins.
func TestDB_SubscribePull_until(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)
	var addrsMu sync.Mutex
	var wantedChunksCount int

	binIDCounter := make(map[uint8]uint64)
	var binIDCounterMu sync.RWMutex

	uploadRandomChunks := func(count int, wanted bool) (last map[uint8]uint64) {
		addrsMu.Lock()
		defer addrsMu.Unlock()

		last = make(map[uint8]uint64)
		for i := 0; i < count; i++ {
			ch := generateTestRandomChunk()

			_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(ch.Address())

			if _, ok := addrs[bin]; !ok {
				addrs[bin] = make([]chunk.Address, 0)
			}
			if wanted {
				addrs[bin] = append(addrs[bin], ch.Address())
				wantedChunksCount++
			}

			binIDCounterMu.RLock()
			binIDCounter[bin]++
			binIDCounterMu.RUnlock()

			last[bin] = binIDCounter[bin]
		}
		return last
	}

	// prepopulate database with some chunks
	// before the subscription
	last := uploadRandomChunks(30, true)

	uploadRandomChunks(25, false)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
		until, ok := last[bin]
		if !ok {
			continue
		}
		ch, stop := db.SubscribePull(ctx, bin, 0, until)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, db, bin, ch, addrs, &addrsMu, errChan)
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(15, false)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// TestDB_SubscribePull_sinceAndUntil uploads chunks before and
// after pull syncing subscriptions are created with since
// and until arguments, and validates if all expected addresses
// are received in the right order for expected proximity order bins.
func TestDB_SubscribePull_sinceAndUntil(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)
	var addrsMu sync.Mutex
	var wantedChunksCount int

	binIDCounter := make(map[uint8]uint64)
	var binIDCounterMu sync.RWMutex

	uploadRandomChunks := func(count int, wanted bool) (last map[uint8]uint64) {
		addrsMu.Lock()
		defer addrsMu.Unlock()

		last = make(map[uint8]uint64)
		for i := 0; i < count; i++ {
			ch := generateTestRandomChunk()

			_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(ch.Address())

			if _, ok := addrs[bin]; !ok {
				addrs[bin] = make([]chunk.Address, 0)
			}
			if wanted {
				addrs[bin] = append(addrs[bin], ch.Address())
				wantedChunksCount++
			}

			binIDCounterMu.RLock()
			binIDCounter[bin]++
			binIDCounterMu.RUnlock()

			last[bin] = binIDCounter[bin]
		}
		return last
	}

	// all chunks from upload1 are not expected
	// as upload1 chunk is used as since for subscriptions
	upload1 := uploadRandomChunks(100, false)

	// all chunks from upload2 are expected
	// as upload2 chunk is used as until for subscriptions
	upload2 := uploadRandomChunks(100, true)

	// upload some chunks before subscribe but after
	// wanted chunks
	uploadRandomChunks(8, false)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
		since, ok := upload1[bin]
		if ok {
			// start from the next uploaded chunk
			since++
		}
		until, ok := upload2[bin]
		if !ok {
			// no chunks un this bin uploaded in the upload2
			// skip this bin from testing
			continue
		}
		ch, stop := db.SubscribePull(ctx, bin, since, until)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, db, bin, ch, addrs, &addrsMu, errChan)
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(15, false)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// uploadRandomChunksBin uploads random chunks to database and adds them to
// the map of addresses ber bin.
func uploadRandomChunksBin(t *testing.T, db *DB, addrs map[uint8][]chunk.Address, addrsMu *sync.Mutex, wantedChunksCount *int, count int) {
	addrsMu.Lock()
	defer addrsMu.Unlock()

	for i := 0; i < count; i++ {
		ch := generateTestRandomChunk()

		_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
		if err != nil {
			t.Fatal(err)
		}

		bin := db.po(ch.Address())
		if _, ok := addrs[bin]; !ok {
			addrs[bin] = make([]chunk.Address, 0)
		}
		addrs[bin] = append(addrs[bin], ch.Address())

		*wantedChunksCount++
	}
}

// readPullSubscriptionBin is a helper function that reads all chunk.Descriptors from a channel and
// sends error to errChan, even if it is nil, to count the number of chunk.Descriptors
// returned by the channel.
func readPullSubscriptionBin(ctx context.Context, db *DB, bin uint8, ch <-chan chunk.Descriptor, addrs map[uint8][]chunk.Address, addrsMu *sync.Mutex, errChan chan error) {
	var i int // address index
	for {
		select {
		case got, ok := <-ch:
			if !ok {
				return
			}
			var err error
			addrsMu.Lock()
			if i+1 > len(addrs[bin]) {
				err = fmt.Errorf("got more chunk addresses %v, then expected %v, for bin %v", i+1, len(addrs[bin]), bin)
			} else {
				addr := addrs[bin][i]
				if !bytes.Equal(got.Address, addr) {
					err = fmt.Errorf("got chunk bin id %v in bin %v %v, want %v", i, bin, got.Address.Hex(), addr.Hex())
				} else {
					want, err := db.retrievalDataIndex.Get(shed.Item{
						Address: addr,
					})
					if err != nil {
						err = fmt.Errorf("got chunk (bin id %v in bin %v) from retrieval index %s: %v", i, bin, addrs[bin][i].Hex(), err)
					} else {
						if got.BinID != want.BinID {
							err = fmt.Errorf("got chunk bin id %v in bin %v %v, want %v", i, bin, got, want)
						}
					}
				}
			}
			addrsMu.Unlock()
			i++
			// send one and only one error per received address
			select {
			case errChan <- err:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// checkErrChan expects the number of wantedChunksCount errors from errChan
// and calls t.Error for the ones that are not nil.
func checkErrChan(ctx context.Context, t *testing.T, errChan chan error, wantedChunksCount int) {
	t.Helper()

	for i := 0; i < wantedChunksCount; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

// TestDB_LastPullSubscriptionBinID validates that LastPullSubscriptionBinID
// is returning the last chunk descriptor for proximity order bins by
// doing a few rounds of chunk uploads.
func TestDB_LastPullSubscriptionBinID(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	addrs := make(map[uint8][]chunk.Address)

	binIDCounter := make(map[uint8]uint64)
	var binIDCounterMu sync.RWMutex

	last := make(map[uint8]uint64)

	// do a few rounds of uploads and check if
	// last pull subscription chunk is correct
	for _, count := range []int{1, 3, 10, 11, 100, 120} {

		// upload
		for i := 0; i < count; i++ {
			ch := generateTestRandomChunk()

			_, err := db.Put(context.Background(), chunk.ModePutUpload, ch)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(ch.Address())

			if _, ok := addrs[bin]; !ok {
				addrs[bin] = make([]chunk.Address, 0)
			}
			addrs[bin] = append(addrs[bin], ch.Address())

			binIDCounterMu.RLock()
			binIDCounter[bin]++
			binIDCounterMu.RUnlock()

			last[bin] = binIDCounter[bin]
		}

		// check
		for bin := uint8(0); bin <= uint8(chunk.MaxPO); bin++ {
			want, ok := last[bin]
			got, err := db.LastPullSubscriptionBinID(bin)
			if ok {
				if err != nil {
					t.Errorf("got unexpected error value %v", err)
				}
			}
			if got != want {
				t.Errorf("got last bin id %v, want %v", got, want)
			}
		}
	}
}

// TestAddressInBin validates that function addressInBin
// returns a valid address for every proximity order bin.
func TestAddressInBin(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	for po := uint8(0); po < chunk.MaxPO; po++ {
		addr := db.addressInBin(po)

		got := db.po(addr)

		if got != uint8(po) {
			t.Errorf("got po %v, want %v", got, po)
		}
	}
}
