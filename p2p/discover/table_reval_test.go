// Copyright 2024 The go-ethereum Authors
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

package discover

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// This test checks that revalidation can handle a node disappearing while
// a request is active.
func TestRevalidation_nodeRemoved(t *testing.T) {
	var (
		clock     mclock.Simulated
		transport = newPingRecorder()
		tab, db   = newInactiveTestTable(transport, Config{Clock: &clock})
		tr        = &tab.revalidation
	)
	defer db.Close()

	// Add a node to the table.
	node := nodeAtDistance(tab.self().ID(), 255, net.IP{77, 88, 99, 1})
	tab.handleAddNode(addNodeOp{node: node})

	// Start a revalidation request. Schedule once to get the next start time,
	// then advance the clock to that point and schedule again to start.
	next := tr.run(tab, clock.Now())
	clock.Run(time.Duration(next + 1))
	tr.run(tab, clock.Now())
	if len(tr.activeReq) != 1 {
		t.Fatal("revalidation request did not start:", tr.activeReq)
	}

	// Delete the node.
	tab.deleteInBucket(tab.bucket(node.ID()), node.ID())

	// Now finish the revalidation request.
	var resp revalidationResponse
	select {
	case resp = <-tab.revalResponseCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for revalidation")
	}
	tr.handleResponse(tab, resp)

	// Ensure the node was not re-added to the table.
	if tab.getNode(node.ID()) != nil {
		t.Fatal("node was re-added to Table")
	}
	if tr.fast.contains(node.ID()) || tr.slow.contains(node.ID()) {
		t.Fatal("removed node contained in revalidation list")
	}
}

// This test checks that nodes with an updated endpoint remain in the fast revalidation list.
func TestRevalidation_endpointUpdate(t *testing.T) {
	var (
		clock     mclock.Simulated
		transport = newPingRecorder()
		tab, db   = newInactiveTestTable(transport, Config{Clock: &clock})
		tr        = &tab.revalidation
	)
	defer db.Close()

	// Add node to table.
	node := nodeAtDistance(tab.self().ID(), 255, net.IP{77, 88, 99, 1})
	tab.handleAddNode(addNodeOp{node: node})

	// Update the record in transport, including endpoint update.
	record := node.Record()
	record.Set(enr.IP{100, 100, 100, 100})
	record.Set(enr.UDP(9999))
	nodev2 := enode.SignNull(record, node.ID())
	transport.updateRecord(nodev2)

	// Start a revalidation request. Schedule once to get the next start time,
	// then advance the clock to that point and schedule again to start.
	next := tr.run(tab, clock.Now())
	clock.Run(time.Duration(next + 1))
	tr.run(tab, clock.Now())
	if len(tr.activeReq) != 1 {
		t.Fatal("revalidation request did not start:", tr.activeReq)
	}

	// Now finish the revalidation request.
	var resp revalidationResponse
	select {
	case resp = <-tab.revalResponseCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for revalidation")
	}
	tr.handleResponse(tab, resp)

	if tr.fast.nodes[0].ID() != node.ID() {
		t.Fatal("node not contained in fast revalidation list")
	}
	if tr.fast.nodes[0].isValidatedLive {
		t.Fatal("node is marked live after endpoint change")
	}
}

// TestRevalidation_concurrentAddAndRun reproduces the data race between the
// Table.loop goroutine (which calls tableRevalidation.run) and the doRefresh
// goroutine (which reaches tableRevalidation.nodeAdded via handleAddNode) on
// revalidationList.nextTime and revalidationList.nodes. See issue #31460.
//
// Without proper locking, this test reliably flags a race under "go test -race".
func TestRevalidation_concurrentAddAndRun(t *testing.T) {
	var (
		transport = newPingRecorder()
		// Real clock + small ping interval so that list.schedule produces
		// nextTime values close to now, and tr.run repeatedly enters the body.
		tab, db = newInactiveTestTable(transport, Config{PingInterval: time.Millisecond})
	)
	defer db.Close()

	// Seed the fast list so tr.run has something to iterate over.
	for i := 0; i < 16; i++ {
		n := nodeAtDistance(tab.self().ID(), 255, net.IP{10, 0, 0, byte(i)})
		tab.mutex.Lock()
		tab.handleAddNode(addNodeOp{node: n})
		tab.mutex.Unlock()
	}

	// A barrier so both goroutines start their loops together.
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	const iterations = 2000

	// Background goroutine: simulate doRefresh -> loadSeedNodes -> handleAddNode,
	// which reaches tr.nodeAdded and appends to list.nodes (and may write
	// list.nextTime via list.schedule).
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			n := nodeAtDistance(tab.self().ID(), 200, net.IP{11, 0, byte(i >> 8), byte(i)})
			tab.mutex.Lock()
			tab.handleAddNode(addNodeOp{node: n})
			tab.mutex.Unlock()
		}
	}()

	// Foreground goroutine: simulate Table.loop repeatedly calling tr.run,
	// which reads list.nextTime and list.nodes and may write list.nextTime
	// via list.schedule.
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			tab.revalidation.run(tab, mclock.System{}.Now())
		}
	}()

	close(start)
	wg.Wait()
}
