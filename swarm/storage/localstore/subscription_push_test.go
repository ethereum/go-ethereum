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

	chunks := make([]storage.Chunk, 0)

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			chunks = append(chunks, chunk)
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

	ch, stop := db.SubscribePush(ctx)
	defer stop()

	// receive and validate addresses from the subscription
	go func() {
		var i int // address index
		for {
			select {
			case got, ok := <-ch:
				if !ok {
					return
				}
				want := chunks[i]
				var err error
				if !bytes.Equal(got.Data(), want.Data()) {
					err = fmt.Errorf("got chunk %v data %x, want %x", i, got.Data(), want.Data())
				}
				if !bytes.Equal(got.Address(), want.Address()) {
					err = fmt.Errorf("got chunk %v address %s, want %s", i, got.Address().Hex(), want.Address().Hex())
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

	totalChunks := len(chunks)
	for i := 0; i < totalChunks; i++ {
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
		ch, stop := db.SubscribePush(ctx)
		defer stop()

		// receive and validate addresses from the subscription
		go func(j int) {
			var i int // address index
			for {
				select {
				case got, ok := <-ch:
					if !ok {
						return
					}
					want := addrs[i]
					var err error
					if !bytes.Equal(got.Address(), want) {
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
			t.Fatal(ctx.Err())
		}
	}
}
