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
// all chunks are received in the right order.
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

	// collect all errors from validating chunks, even nil ones
	// to validate the number of chunks received by the subscription
	errChan := make(chan error)

	sub, err := db.SubscribePush(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Stop()

	// receive and validate chunks from the subscription
	go func() {
		var i int // chunk index
		for {
			select {
			case got := <-sub.Chunks:
				want := chunks[i]
				var err error
				if !bytes.Equal(got.Data(), want.Data()) {
					err = fmt.Errorf("got chunk %v data %x, want %x", i, got.Data(), want.Data())
				}
				if !bytes.Equal(got.Address(), want.Address()) {
					err = fmt.Errorf("got chunk %v address %s, want %s", i, got.Address().Hex(), want.Address().Hex())
				}
				i++
				// send one and only one error per received chunk
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
			t.Error(ctx.Err())
		}
	}
}

// TestSubscribePush_multiple uploads chunks before and after
// multiple push syncing subscriptions are created and
// validates if all chunks are received in the right order.
func TestSubscribePush_multiple(t *testing.T) {
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

	// collect all errors from validating chunks, even nil ones
	// to validate the number of chunks received by the subscription
	errChan := make(chan error)

	subsCount := 10

	// start a number of subscriptions
	// that all of them will write every chunk error to errChan
	for j := 0; j < subsCount; j++ {
		sub, err := db.SubscribePush(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Stop()

		// receive and validate chunks from the subscription
		go func(j int) {
			var i int // chunk index
			for {
				select {
				case got := <-sub.Chunks:
					want := chunks[i]
					var err error
					if !bytes.Equal(got.Data(), want.Data()) {
						err = fmt.Errorf("got chunk %v on subscription %v data %x, want %x", i, j, got.Data(), want.Data())
					}
					if !bytes.Equal(got.Address(), want.Address()) {
						err = fmt.Errorf("got chunk %v on subscription %v address %s, want %s", i, j, got.Address().Hex(), want.Address().Hex())
					}
					i++
					// send one and only one error per received chunk
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

	// number of chunks received by all subscriptions
	totalChunks := len(chunks) * subsCount
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
// all chunks are received in the right order
// for expected proximity order bins.
func TestSubscribePull(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	chunks := make(map[uint8][]storage.Chunk)
	var uploadedChunksCount int

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(chunk.Address())
			if _, ok := chunks[bin]; !ok {
				chunks[bin] = make([]storage.Chunk, 0)
			}

			chunks[bin] = append(chunks[bin], chunk)
			uploadedChunksCount++
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating chunks, even nil ones
	// to validate the number of chunks received by the subscription
	errChan := make(chan error)

	var maxBin uint8 = 32

	for bin := uint8(0); bin < maxBin; bin++ {
		sub, err := db.SubscribePull(ctx, bin)
		if err != nil {
			t.Fatal(err)
		}
		defer sub.Stop()

		// receive and validate chunks from the subscription
		go func(bin uint8) {
			var i int // chunk index
			for {
				select {
				case got := <-sub.Chunks:
					want := chunks[bin][i]
					var err error
					if !bytes.Equal(got.Data(), want.Data()) {
						err = fmt.Errorf("got chunk %v in bin %v data %x, want %x", i, bin, got.Data(), want.Data())
					}
					if !bytes.Equal(got.Address(), want.Address()) {
						err = fmt.Errorf("got chunk %v in bin %v address %s, want %s", i, bin, got.Address().Hex(), want.Address().Hex())
					}
					i++
					// send one and only one error per received chunk
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
// validates if all chunks are received in the right order
// for expected proximity order bins.
func TestSubscribePull_multiple(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	chunks := make(map[uint8][]storage.Chunk)
	var uploadedChunksCount int

	uploadRandomChunks := func(count int) {
		for i := 0; i < count; i++ {
			chunk := generateRandomChunk()

			err := uploader.Put(chunk)
			if err != nil {
				t.Fatal(err)
			}

			bin := db.po(chunk.Address())
			if _, ok := chunks[bin]; !ok {
				chunks[bin] = make([]storage.Chunk, 0)
			}

			chunks[bin] = append(chunks[bin], chunk)
			uploadedChunksCount++
		}
	}

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunks(10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating chunks, even nil ones
	// to validate the number of chunks received by the subscription
	errChan := make(chan error)

	var maxBin uint8 = 32

	subsCount := 10

	// start a number of subscriptions
	// that all of them will write every chunk error to errChan
	for j := 0; j < subsCount; j++ {
		for bin := uint8(0); bin < maxBin; bin++ {
			sub, err := db.SubscribePull(ctx, bin)
			if err != nil {
				t.Fatal(err)
			}
			defer sub.Stop()

			// receive and validate chunks from the subscription
			go func(bin uint8, j int) {
				var i int // chunk index
				for {
					select {
					case got := <-sub.Chunks:
						want := chunks[bin][i]
						var err error
						if !bytes.Equal(got.Data(), want.Data()) {
							err = fmt.Errorf("got chunk %v in bin %v on subscription %v data %x, want %x", i, bin, j, got.Data(), want.Data())
						}
						if !bytes.Equal(got.Address(), want.Address()) {
							err = fmt.Errorf("got chunk %v in bin %v on subscription %v address %s, want %s", i, bin, j, got.Address().Hex(), want.Address().Hex())
						}
						i++
						// send one and only one error per received chunk
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
