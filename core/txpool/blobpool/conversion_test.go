// Copyright 2026 The go-ethereum Authors
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

package blobpool

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestConversionQueueBasic(t *testing.T) {
	queue := newConversionQueue()
	defer queue.close()

	key, _ := crypto.GenerateKey()
	tx := makeTx(0, 1, 1, 1, key)

	ptx, err := queue.convert(tx)
	if err != nil {
		t.Fatalf("Expected successful conversion, got error: %v", err)
	}
	if ptx == nil {
		t.Fatal("Expected a converted transaction, got nil")
	}
	if ptx.Tx.Hash() != tx.Hash() {
		t.Errorf("Converted tx hash mismatch: have %s, want %s", ptx.Tx.Hash(), tx.Hash())
	}
	if len(ptx.CellSidecar.Cells) == 0 {
		t.Error("Expected cells to be computed during conversion")
	}
}

func TestConversionQueueClosed(t *testing.T) {
	queue := newConversionQueue()
	queue.close()

	key, _ := crypto.GenerateKey()
	tx := makeTx(0, 1, 1, 1, key)

	if _, err := queue.convert(tx); err == nil {
		t.Fatal("Expected error when converting on closed queue, got nil")
	}
}

func TestConversionQueueDoubleClose(t *testing.T) {
	queue := newConversionQueue()
	queue.close()
	queue.close() // Should not panic
}

func TestConversionQueueSerialBackgroundTasks(t *testing.T) {
	queue := newConversionQueue()

	firstStarted := make(chan struct{})
	firstRelease := make(chan struct{})
	if err := queue.launchConversion(func() {
		close(firstStarted)
		<-firstRelease
	}); err != nil {
		t.Fatalf("Failed to launch first conversion: %v", err)
	}
	<-firstStarted

	secondStarted := make(chan struct{})
	if err := queue.launchConversion(func() { close(secondStarted) }); err != nil {
		t.Fatalf("Failed to launch second conversion: %v", err)
	}
	select {
	case <-secondStarted:
		close(firstRelease)
		queue.close()
		t.Fatal("Second conversion started before first conversion finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(firstRelease)
	select {
	case <-secondStarted:
	case <-time.After(time.Second):
		queue.close()
		t.Fatal("Second conversion did not start after first conversion finished")
	}
	queue.close()
}

func TestConversionQueueCloseWaitsForBackgroundTask(t *testing.T) {
	queue := newConversionQueue()

	started := make(chan struct{})
	release := make(chan struct{})
	if err := queue.launchConversion(func() {
		close(started)
		<-release
	}); err != nil {
		t.Fatalf("Failed to launch conversion: %v", err)
	}
	<-started

	closed := make(chan struct{})
	go func() {
		queue.close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Queue closed before the running conversion finished")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Queue did not close after the running conversion finished")
	}
}

func TestConversionQueueAutoRestartBatch(t *testing.T) {
	queue := newConversionQueue()
	defer queue.close()

	key, _ := crypto.GenerateKey()

	// Create a heavy transaction to ensure the first batch runs long enough
	// for subsequent tasks to be queued while it is active.
	heavy := makeMultiBlobTx(0, 1, 1, 1, int(params.BlobTxMaxBlobs), 0, key)

	var wg sync.WaitGroup
	wg.Add(1)
	heavyDone := make(chan error, 1)
	go func() {
		defer wg.Done()
		_, err := queue.convert(heavy)
		heavyDone <- err
	}()

	// Give the conversion worker a head start so that the following tasks are
	// enqueued while the first batch is running.
	time.Sleep(200 * time.Millisecond)

	tx1 := makeTx(1, 1, 1, 1, key)
	tx2 := makeTx(2, 1, 1, 1, key)

	wg.Add(2)
	done1 := make(chan error, 1)
	done2 := make(chan error, 1)
	go func() { defer wg.Done(); _, err := queue.convert(tx1); done1 <- err }()
	go func() { defer wg.Done(); _, err := queue.convert(tx2); done2 <- err }()

	for _, c := range []struct {
		name string
		done chan error
	}{{"tx1", done1}, {"tx2", done2}, {"heavy", heavyDone}} {
		select {
		case err := <-c.done:
			if err != nil {
				t.Fatalf("%s conversion error: %v", c.name, err)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("timeout waiting for %s conversion", c.name)
		}
	}
	wg.Wait()
}
