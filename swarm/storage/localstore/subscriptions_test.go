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
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestSubscribePush uploads some chunks before and after
// push syncing subscription is created and validates if
// all addresses are received in the right order.
func TestSubscribePush(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make([]storage.Address, 0)

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			addrs = append(addrs, chunk.Address())
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	sub, err := db.SubscribePush(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Stop()

	// receive and validate addresses from the subscription
	go func() {
		var i int // address index
		for {
			select {
			case got := <-sub.Addrs:
				want := addrs[i]
				var err error
				if !bytes.Equal(got, want) {
					err = fmt.Errorf("got chunk %v address %s, want %s", i, got, want)
				}
				i++
				// send one and only one error per received address
				errChan <- err
			case <-ctx.Done():
				return
			}
		}
	}()

	// upload some chunks just after subscribe
	uploadRandomChunks(5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunks(3)

	totalChunks := len(addrs)
	for i := 0; i < totalChunks; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	}
}

// TestSubscribePush_multiple uploads chunks before and after
// multiple push syncing subscriptions are created and
// validates if all addresses are received in the right order.
func TestSubscribePush_multiple(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make([]storage.Address, 0)

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			addrs = append(addrs, chunk.Address())
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	subsCount := 10

	// start a number of subscriptions
	// that all of them will write every addresses error to errChan
	for j := 0; j < subsCount; j++ {
		sub, err := db.SubscribePush(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Stop()

		// receive and validate addresses from the subscription
		go func(j int) {
			var i int // address index
			for {
				select {
				case got := <-sub.Addrs:
					want := addrs[i]
					var err error
					if !bytes.Equal(got, want) {
						err = fmt.Errorf("got chunk %v address on subscription %v %s, want %s", i, j, got, want)
					}
					i++
					// send one and only one error per received address
					errChan <- err
				case <-ctx.Done():
					return
				}
			}
		}(j)
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunks(3)

	// number of addresses received by all subscriptions
	totalChunks := len(addrs) * subsCount
	for i := 0; i < totalChunks; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	}
}

// TestSubscribePull uploads some chunks before and after
// pull syncing subscription is created and validates if
// all addresses are received in the right order
// for expected proximity order bins.
func TestSubscribePull(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var uploadedChunksCount int

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(chunk.Address())
			if _, ok := addrs[bin]; !ok {
				addrs[bin] = make([]storage.Address, 0)
			}

			addrs[bin] = append(addrs[bin], chunk.Address())
			uploadedChunksCount++
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin < uint8(storage.MaxPO); bin++ {
		sub, err := db.SubscribePull(ctx, bin)
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Stop()

		// receive and validate addresses from the subscription
		go func(bin uint8) {
			var i int // address index
			for {
				select {
				case got := <-sub.Addrs:
					want := addrs[bin][i]
					var err error
					if !bytes.Equal(got, want) {
						err = fmt.Errorf("got chunk address %v in bin %v %s, want %s", i, bin, got, want)
					}
					i++
					// send one and only one error per received address
					errChan <- err
				case <-ctx.Done():
					return
				}
			}
		}(bin)
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunks(3)

	for i := 0; i < uploadedChunksCount; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	}
}

// TestSubscribePull_multiple uploads chunks before and after
// multiple pull syncing subscriptions are created and
// validates if all addresses are received in the right order
// for expected proximity order bins.
func TestSubscribePull_multiple(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var uploadedChunksCount int

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(chunk.Address())
			if _, ok := addrs[bin]; !ok {
				addrs[bin] = make([]storage.Address, 0)
			}

			addrs[bin] = append(addrs[bin], chunk.Address())
			uploadedChunksCount++
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

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
		for bin := uint8(0); bin < uint8(storage.MaxPO); bin++ {
			sub, err := db.SubscribePull(ctx, bin)
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Stop()

			// receive and validate addresses from the subscription
			go func(bin uint8, j int) {
				var i int // address index
				for {
					select {
					case got := <-sub.Addrs:
						want := addrs[bin][i]
						var err error
						if !bytes.Equal(got, want) {
							err = fmt.Errorf("got chunk address %v in bin %v on subscription %v %s, want %s", i, bin, j, got, want)
						}
						i++
						// send one and only one error per received address
						errChan <- err
					case <-ctx.Done():
						return
					}
				}
			}(bin, j)
		}
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunks(3)

	totalChunks := uploadedChunksCount * subsCount

	for i := 0; i < totalChunks; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				t.Error(err)
			}
		case <-ctx.Done():
			t.Error(ctx.Err())
		}
	}
}
