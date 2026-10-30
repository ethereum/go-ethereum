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

package downloader

import (
	"testing"
	"time"
)

// TestUnsubscribeWhileBroadcasting verifies that SyncStatusSubscription.Unsubscribe
// does not deadlock when the API event loop is blocked in a broadcast send to the
// subscription's status channel. This is the situation where the subscription
// owner has stopped reading events (e.g. the RPC client disconnected) and the
// event loop is blocked writing to the status channel, so without interleaved
// draining in Unsubscribe both goroutines block forever.
func TestUnsubscribeWhileBroadcasting(t *testing.T) {
	api := &DownloaderAPI{
		installSyncSubscription:   make(chan chan interface{}),
		uninstallSyncSubscription: make(chan *uninstallSyncSubscriptionRequest),
	}
	sub := &SyncStatusSubscription{api: api, c: make(chan interface{})}

	// Emulate the DownloaderAPI event loop: it gets blocked in a broadcast send
	// to the status channel, then processes the uninstall request once unblocked.
	broadcasted := make(chan struct{})
	go func() {
		sub.c <- &SyncingResult{Syncing: true} // blocks until Unsubscribe drains it
		close(broadcasted)
		req := <-api.uninstallSyncSubscription
		close(req.uninstalled)
	}()

	done := make(chan struct{})
	go func() {
		sub.Unsubscribe()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Unsubscribe deadlocked while the event loop was broadcasting")
	}
	select {
	case <-broadcasted:
	case <-time.After(time.Second):
		t.Fatal("broadcasting goroutine was not unblocked")
	}
}

// TestUnsubscribeDropsEvents verifies that Unsubscribe consumes the events
// pending on the status channel and still uninstalls the subscription when
// the event loop is processing a broadcast around the same time.
func TestUnsubscribeDropsEvents(t *testing.T) {
	api := &DownloaderAPI{
		installSyncSubscription:   make(chan chan interface{}),
		uninstallSyncSubscription: make(chan *uninstallSyncSubscriptionRequest),
	}
	sub := &SyncStatusSubscription{api: api, c: make(chan interface{})}

	// Emulate the DownloaderAPI event loop: it broadcasts one event, then
	// processes the uninstall request. The broadcast is only delivered once
	// Unsubscribe drains the status channel.
	broadcasted := make(chan struct{})
	go func() {
		sub.c <- false
		close(broadcasted)
		req := <-api.uninstallSyncSubscription
		close(req.uninstalled)
	}()

	done := make(chan struct{})
	go func() {
		sub.Unsubscribe()
		close(done)
	}()

	select {
	case <-broadcasted:
	case <-time.After(3 * time.Second):
		t.Fatal("pending broadcast was not consumed by Unsubscribe")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Unsubscribe did not return after uninstall confirmation")
	}
}
