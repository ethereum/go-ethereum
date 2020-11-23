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

package simulation

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestPeerEvents creates simulation, adds two nodes,
// register for peer events, connects nodes in a chain
// and waits for the number of connection events to
// be received.
func TestPeerEvents(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events := sim.PeerEvents(ctx, sim.NodeIDs())

	// two nodes -> two connection events
	expectedEventCount := 2

	var wg sync.WaitGroup
	wg.Add(expectedEventCount)

	go func() {
		for e := range events {
			if e.Error != nil {
				if e.Error == context.Canceled {
					return
				}
				t.Error(e.Error)
				continue
			}
			wg.Done()
		}
	}()

	err = sim.ConnectNodesChain(sim.NodeIDs())
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()
}

func TestPeerEventsTimeout(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	_, err := sim.AddNodes(2)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	events := sim.PeerEvents(ctx, sim.NodeIDs())

	done := make(chan struct{})
	go func() {
		for e := range events {
			if e.Error == context.Canceled {
				return
			}
			if e.Error == context.DeadlineExceeded {
				close(done)
				return
			} else {
				t.Fatal(e.Error)
			}
		}
	}()

	select {
	case <-time.After(time.Second):
		t.Error("no context deadline received")
	case <-done:
		// all good, context deadline detected
	}
}
