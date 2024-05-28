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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

// This test checks that revalidation can handle a node disappearing while
// a request is active.
func TestRevalidationNodeRemoved(t *testing.T) {
	var (
		clock     mclock.Simulated
		transport = newPingRecorder()
		tab, db   = newInactiveTestTable(transport, Config{Clock: &clock})
		tr        = &tab.revalidation
	)
	defer db.Close()

	// Fill a bucket.
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
