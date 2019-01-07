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
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestDB_SubscribePull uploads some chunks before and after
// pull syncing subscription is created and validates if
// all addresses are received in the right order
// for expected proximity order bins.
func TestDB_SubscribePull(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var wantedChunksCount int

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 10)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(storage.MaxPO); bin++ {
		ch, stop := db.SubscribePull(ctx, bin, nil, nil)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, bin, ch, addrs, errChan)
	}

	// upload some chunks just after subscribe
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 3)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// TestDB_SubscribePull_multiple uploads chunks before and after
// multiple pull syncing subscriptions are created and
// validates if all addresses are received in the right order
// for expected proximity order bins.
func TestDB_SubscribePull_multiple(t *testing.T) {
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var wantedChunksCount int

	// prepopulate database with some chunks
	// before the subscription
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 10)

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
		for bin := uint8(0); bin <= uint8(storage.MaxPO); bin++ {
			ch, stop := db.SubscribePull(ctx, bin, nil, nil)
			defer stop()

			// receive and validate addresses from the subscription
			go readPullSubscriptionBin(ctx, bin, ch, addrs, errChan)
		}
	}

	// upload some chunks just after subscribe
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 5)

	time.Sleep(500 * time.Millisecond)

	// upload some chunks after some short time
	uploadRandomChunksBin(t, db, uploader, addrs, &wantedChunksCount, 3)

	checkErrChan(ctx, t, errChan, wantedChunksCount*subsCount)
}

// TestDB_SubscribePull_since uploads chunks before and after
// pull syncing subscriptions are created with a since argument
// and validates if all expected addresses are received in the
// right order for expected proximity order bins.
func TestDB_SubscribePull_since(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var wantedChunksCount int

	lastTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return atomic.AddInt64(&lastTimestamp, 1)
	})()

	uploadRandomChunks := func(count int, wanted bool) (last map[uint8]ChunkInfo) {
		last = make(map[uint8]ChunkInfo)
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

			if wanted {
				addrs[bin] = append(addrs[bin], chunk.Address())
				wantedChunksCount++
			}

			last[bin] = ChunkInfo{
				Address:        chunk.Address(),
				StoreTimestamp: atomic.LoadInt64(&lastTimestamp),
			}
		}
		return last
	}

	// prepopulate database with some chunks
	// before the subscription
	last := uploadRandomChunks(30, false)

	uploadRandomChunks(25, true)

	// set a timeout on subscription
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// collect all errors from validating addresses, even nil ones
	// to validate the number of addresses received by the subscription
	errChan := make(chan error)

	for bin := uint8(0); bin <= uint8(storage.MaxPO); bin++ {
		var since *ChunkInfo
		if c, ok := last[bin]; ok {
			since = &c
		}
		ch, stop := db.SubscribePull(ctx, bin, since, nil)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, bin, ch, addrs, errChan)

	}

	// upload some chunks just after subscribe
	uploadRandomChunks(15, true)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// TestDB_SubscribePull_until uploads chunks before and after
// pull syncing subscriptions are created with an until argument
// and validates if all expected addresses are received in the
// right order for expected proximity order bins.
func TestDB_SubscribePull_until(t *testing.T) {
	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var wantedChunksCount int

	lastTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return atomic.AddInt64(&lastTimestamp, 1)
	})()

	uploadRandomChunks := func(count int, wanted bool) (last map[uint8]ChunkInfo) {
		last = make(map[uint8]ChunkInfo)
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

			if wanted {
				addrs[bin] = append(addrs[bin], chunk.Address())
				wantedChunksCount++
			}

			last[bin] = ChunkInfo{
				Address:        chunk.Address(),
				StoreTimestamp: atomic.LoadInt64(&lastTimestamp),
			}
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

	for bin := uint8(0); bin <= uint8(storage.MaxPO); bin++ {
		until, ok := last[bin]
		if !ok {
			continue
		}
		ch, stop := db.SubscribePull(ctx, bin, nil, &until)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, bin, ch, addrs, errChan)
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
	t.Parallel()

	db, cleanupFunc := newTestDB(t, nil)
	defer cleanupFunc()

	uploader := db.NewPutter(ModePutUpload)

	addrs := make(map[uint8][]storage.Address)
	var wantedChunksCount int

	lastTimestamp := time.Now().UTC().UnixNano()
	defer setNow(func() (t int64) {
		return atomic.AddInt64(&lastTimestamp, 1)
	})()

	uploadRandomChunks := func(count int, wanted bool) (last map[uint8]ChunkInfo) {
		last = make(map[uint8]ChunkInfo)
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

			if wanted {
				addrs[bin] = append(addrs[bin], chunk.Address())
				wantedChunksCount++
			}

			last[bin] = ChunkInfo{
				Address:        chunk.Address(),
				StoreTimestamp: atomic.LoadInt64(&lastTimestamp),
			}
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

	for bin := uint8(0); bin <= uint8(storage.MaxPO); bin++ {
		var since *ChunkInfo
		if c, ok := upload1[bin]; ok {
			since = &c
		}
		until, ok := upload2[bin]
		if !ok {
			// no chunks un this bin uploaded in the upload2
			// skip this bin from testing
			continue
		}
		ch, stop := db.SubscribePull(ctx, bin, since, &until)
		defer stop()

		// receive and validate addresses from the subscription
		go readPullSubscriptionBin(ctx, bin, ch, addrs, errChan)
	}

	// upload some chunks just after subscribe
	uploadRandomChunks(15, false)

	checkErrChan(ctx, t, errChan, wantedChunksCount)
}

// uploadRandomChunksBin uploads random chunks to database and adds them to
// the map of addresses ber bin.
func uploadRandomChunksBin(t *testing.T, db *DB, uploader *Putter, addrs map[uint8][]storage.Address, wantedChunksCount *int, count int) {
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
		*wantedChunksCount++
	}
}

// readPullSubscriptionBin is a helper function that reads all ChunkInfos from a channel and
// sends error to errChan, even if it is nil, to count the number of ChunkInfos
// returned by the channel.
func readPullSubscriptionBin(ctx context.Context, bin uint8, ch <-chan ChunkInfo, addrs map[uint8][]storage.Address, errChan chan error) {
	var i int // address index
	for {
		select {
		case got, ok := <-ch:
			if !ok {
				return
			}
			want := addrs[bin][i]
			var err error
			if !bytes.Equal(got.Address, want) {
				err = fmt.Errorf("got chunk address %v in bin %v %s, want %s", i, bin, got.Address.Hex(), want)
			}
			i++
			// send one and only one error per received address
			errChan <- err
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
