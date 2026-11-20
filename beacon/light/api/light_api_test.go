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

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

// TestHeadListenerResubscribe checks that the head listener subscribes to the event
// stream again after it ends, and that failed attempts don't make it wait longer and
// longer.
func TestHeadListenerResubscribe(t *testing.T) {
	defer func(oldMin, oldMax time.Duration) {
		minEventStreamWait, maxEventStreamWait = oldMin, oldMax
	}(minEventStreamWait, maxEventStreamWait)
	minEventStreamWait, maxEventStreamWait = time.Millisecond, 5*time.Millisecond

	// The first stream sends a head and ends, the next subscriptions fail, then a
	// stream sends the next head. A wait doubling without a cap would reach 2^20 ms
	// (17 minutes) before the last subscription.
	const failures = 20
	var subscriptions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/eth/v1/events" {
			http.NotFound(w, r)
			return
		}
		n := subscriptions.Add(1)
		if n > 1 && n <= 1+failures {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: head\ndata: {\"slot\":\"%d\",\"block\":\"%s\"}\n\n", n, common.Hash{byte(n)}.Hex())
		w.(http.Flusher).Flush()
		if n > 1 {
			<-r.Context().Done() // keep the stream open until the listener stops
		}
	}))
	defer server.Close()

	var (
		heads = make(chan uint64, 10)
		errs  atomic.Int32
	)
	stop := NewBeaconLightApi(server.URL, nil).StartHeadListener(HeadEventListener{
		OnNewHead:    func(slot uint64, blockRoot common.Hash) { heads <- slot },
		OnOptimistic: func(types.OptimisticUpdate) {},
		OnFinality:   func(types.FinalityUpdate) {},
		OnError:      func(error) { errs.Add(1) },
	})
	defer stop()

	for _, want := range []uint64{1, failures + 2} {
		select {
		case slot := <-heads:
			if slot != want {
				t.Fatalf("wrong head slot: got %d, want %d", slot, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no head event for slot %d", want)
		}
	}
	// The end of the first stream and every failed subscription are reported.
	if n := errs.Load(); n != failures+1 {
		t.Fatalf("wrong number of errors: got %d, want %d", n, failures+1)
	}
}

// TestEventStreamWait checks the waits between subscriptions to the event stream:
// doubling after attempts without events, up to the cap, and back to the start
// after an attempt with events.
func TestEventStreamWait(t *testing.T) {
	var (
		received = []bool{false, false, false, false, false, false, false, true, false}
		s        = time.Second
		want     = []time.Duration{1 * s, 2 * s, 4 * s, 8 * s, 16 * s, 30 * s, 30 * s, 1 * s, 2 * s}
		wait     time.Duration
	)
	for i := range received {
		wait = eventStreamWait(wait, received[i])
		if wait != want[i] {
			t.Fatalf("wait %d: got %v, want %v", i, wait, want[i])
		}
	}
}

// TestReadEventStreamReceived checks that only named events count as received:
// other text, such as a web page, doesn't reset the wait between subscriptions.
func TestReadEventStreamReceived(t *testing.T) {
	for _, tc := range []struct {
		body     string
		received bool
	}{
		{"<html>\n<body>not an event stream</body>\n</html>\n\n", false},
		{"event: head\ndata: {}\n\n", true},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, tc.body)
		}))
		received, err := NewBeaconLightApi(server.URL, nil).readEventStream(context.Background(), make(chan eventsource.Event, 10))
		server.Close()
		if received != tc.received || err == nil {
			t.Errorf("body %q: received %v, error %v; want received %v and the end of the stream", tc.body, received, err, tc.received)
		}
	}
}

// TestReadEventStreamErrorBody checks that a failed subscription doesn't wait long
// for an error body that doesn't end.
func TestReadEventStreamErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "unavailable")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	defer server.CloseClientConnections()

	done := make(chan error, 1)
	go func() {
		_, err := NewBeaconLightApi(server.URL, nil).readEventStream(context.Background(), make(chan eventsource.Event))
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.HasSuffix(err.Error(), "status code 503: unavailable") {
			t.Errorf("wrong error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("still reading the error body")
	}
}

// TestHeadListenerStop checks that stopping the head listener returns while a
// subscription waits for the response, and while a stream is open.
func TestHeadListenerStop(t *testing.T) {
	for _, open := range []bool{false, true} {
		t.Run(fmt.Sprintf("open=%v", open), func(t *testing.T) {
			var (
				subscribed = make(chan struct{}, 1)
				release    = make(chan struct{})
			)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/eth/v1/events" {
					http.NotFound(w, r)
					return
				}
				if open {
					w.Header().Set("Content-Type", "text/event-stream")
					w.(http.Flusher).Flush()
				}
				select {
				case subscribed <- struct{}{}:
				default:
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			}))
			defer server.Close()

			stop := NewBeaconLightApi(server.URL, nil).StartHeadListener(HeadEventListener{
				OnNewHead:    func(uint64, common.Hash) {},
				OnOptimistic: func(types.OptimisticUpdate) {},
				OnFinality:   func(types.FinalityUpdate) {},
				OnError:      func(error) {},
			})
			// On a failure, release the server and its connections first, so that
			// stopping (again: that's fine) doesn't wait on them; and don't wait long.
			defer func() {
				stopped := make(chan struct{})
				go func() {
					stop()
					close(stopped)
				}()
				select {
				case <-stopped:
				case <-time.After(5 * time.Second):
					t.Error("the listener didn't stop at the cleanup")
				}
			}()
			defer server.CloseClientConnections()
			defer close(release)
			select {
			case <-subscribed:
			case <-time.After(5 * time.Second):
				t.Fatal("no subscription")
			}
			waitBlocked(t, "readEventStream", false) // for the response, or the first event
			stopped := make(chan struct{})
			go func() {
				stop()
				close(stopped)
			}()
			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				t.Fatal("the listener didn't stop")
			}
		})
	}
}

// TestEventStreamCancel checks that the event stream functions return when the
// context is closed while they wait: reading the stream, passing on an event,
// reporting an error.
func TestEventStreamCancel(t *testing.T) {
	// check serves handler, runs start, closes the context once start returns,
	// and waits for the channel start returned to be closed.
	check := func(t *testing.T, handler http.HandlerFunc, start func(t *testing.T, api *BeaconLightApi, ctx context.Context) <-chan struct{}) {
		server := httptest.NewServer(handler)
		defer server.Close()
		defer server.CloseClientConnections() // ends the open streams on a failure

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		returned := start(t, NewBeaconLightApi(server.URL, nil), ctx)
		cancel()
		select {
		case <-returned:
		case <-time.After(5 * time.Second):
			t.Fatal("didn't return after the context was closed")
		}
	}
	// stream sends events and keeps the stream open.
	stream := func(events int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, strings.Repeat("event: head\ndata: {}\n\n", events))
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}
	}
	// readStream runs readEventStream and takes its first event; with blocked,
	// it then waits until readEventStream waits to pass on the next one.
	readStream := func(blocked bool) func(t *testing.T, api *BeaconLightApi, ctx context.Context) <-chan struct{} {
		return func(t *testing.T, api *BeaconLightApi, ctx context.Context) <-chan struct{} {
			var (
				events   = make(chan eventsource.Event)
				returned = make(chan struct{})
			)
			go func() {
				api.readEventStream(ctx, events)
				close(returned)
			}()
			select {
			case <-events:
			case <-returned:
				t.Fatal("returned before passing on an event")
			case <-time.After(5 * time.Second):
				t.Fatal("no event")
			}
			// Waiting for the next event, or to pass it on.
			waitBlocked(t, "readEventStream", blocked)
			return returned
		}
	}
	// One event, taken: readEventStream waits for the next one.
	t.Run("reading", func(t *testing.T) { check(t, stream(1), readStream(false)) })
	// Two events, the second not taken: readEventStream waits to pass it on.
	t.Run("event", func(t *testing.T) { check(t, stream(2), readStream(true)) })
	// A failed subscription, its error not taken: runEventStream waits to report it.
	t.Run("error", func(t *testing.T) {
		unavailable := func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}
		check(t, unavailable, func(t *testing.T, api *BeaconLightApi, ctx context.Context) <-chan struct{} {
			returned := make(chan struct{})
			go func() {
				api.runEventStream(ctx, make(chan eventsource.Event), make(chan error))
				close(returned)
			}()
			waitBlocked(t, "runEventStream", true)
			return returned
		})
	})
}

// waitBlocked waits, for 5 s at most, until a goroutine is blocked with the method fn
// of BeaconLightApi on its stack: as its innermost frame outside the runtime (blocked
// in fn itself, in a select) if inner, anywhere otherwise (in something fn called). It
// reads the runtime's goroutine dump.
func waitBlocked(t *testing.T, fn string, inner bool) {
	t.Helper()
	buf := make([]byte, 1<<20)
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(time.Millisecond) {
		n := runtime.Stack(buf, true)
		for _, g := range strings.Split(string(buf[:n]), "\n\n") {
			if blockedIn(g, fn, inner) {
				return
			}
		}
	}
	t.Fatalf("%s isn't blocked", fn)
}

// blockedIn reports whether g, one goroutine of the runtime's dump, is blocked with
// fn on its stack (see waitBlocked).
func blockedIn(g, fn string, inner bool) bool {
	lines := strings.Split(g, "\n")
	if len(lines) < 2 || strings.Contains(lines[0], "[running") || strings.Contains(lines[0], "[runnable") {
		return false
	}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "runtime.") || strings.HasPrefix(line, "created by ") {
			continue // file positions, frames of the runtime (GOTRACEBACK=system shows them)
		}
		if strings.Contains(line, ")."+fn+"(") {
			return true
		}
		if inner {
			return false
		}
	}
	return false
}
