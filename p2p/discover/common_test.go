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

package discover

import (
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discutil"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// This test checks basic operation of the lookup iterator.
func TestLookupIterator(t *testing.T) {
	var (
		test      = newLookupWalkerTest()
		testNodes = make([]*enode.Node, 100)
		wg        sync.WaitGroup
	)
	for i := range testNodes {
		testNodes[i] = testNode(i)
	}
	testIterator := func(it discutil.Iterator) {
		defer wg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		nodes := discutil.ReadNodes(ctx, it, 20)
		sortByID(nodes) // ReadNodes may shuffle results
		if err := checkNodesEqual(nodes, testNodes[:20]); err != nil {
			t.Error(err)
		}

		it.Close()
		n, isLive := it.NextNode(context.Background())
		if n != nil {
			t.Error("iterator returned non-nil node after close")
		}
		if isLive {
			t.Error("iterator returned isLive == true after close")
		}

		it.Close() // shouldn't crash
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go testIterator(test.newIterator())
	}

	test.serveOneLookup(testNodes[:10])
	test.serveOneLookup(testNodes[10:])
	wg.Wait()

	test.close()
}

// This test checks that the lookup iterator drops nodes when they're not being
// read fast enough.
func TestLookupIteratorDropStale(t *testing.T) {
	var (
		test       = newLookupWalkerTest()
		testNodes  = make([]*enode.Node, 2*lookupIteratorBuffer)
		lookupDone = make(chan struct{})
	)
	defer test.close()

	for i := range testNodes {
		testNodes[i] = testNode(i)
	}

	// Create iterator first so all found nodes go through its buffer.
	it := test.newIterator()

	// Serve one lookup.
	go func() {
		test.serveOneLookup(testNodes)
		close(lookupDone)
	}()

	// The first call to NextNode triggers the lookup and receives the first result
	// as soon as it becomes available.
	n, _ := it.NextNode(context.Background())
	if n != testNodes[0] {
		t.Fatalf("wrong result %d: got %v, want %v", 0, n.ID(), testNodes[0])
	}

	// Now wait for the lookup to finish and read the remaining nodes.
	<-lookupDone
	for i := 0; i < lookupIteratorBuffer; i++ {
		n, _ := it.NextNode(context.Background())
		for _, tn := range testNodes[lookupIteratorBuffer:] {
			if n == tn {
				return
			}
		}
	}
	t.Fatal("didn't find any node from second half of testNodes")
}

func testNode(id int) *enode.Node {
	var nodeID enode.ID
	binary.BigEndian.PutUint64(nodeID[:], uint64(id))
	return enode.SignNull(new(enr.Record), nodeID)
}

type lookupWalkerTest struct {
	*lookupWalker
	running int32
	nodes   chan []*enode.Node
}

func newLookupWalkerTest() *lookupWalkerTest {
	wt := &lookupWalkerTest{nodes: make(chan []*enode.Node)}
	wt.lookupWalker = newLookupWalker(wt.lookupFunc)
	return wt
}

// serveOneLookup allows one lookupFunc call to happen and makes it find
// the given nodes.
func (t *lookupWalkerTest) serveOneLookup(nodes []*enode.Node) {
	t.nodes <- nodes
	<-t.nodes
}

func (t *lookupWalkerTest) lookupFunc(callback func(*enode.Node)) {
	if atomic.AddInt32(&t.running, 1) != 1 {
		panic("spawned more than one instance of lookupFunc")
	}
	defer atomic.AddInt32(&t.running, -1)

	select {
	case nodes := <-t.nodes:
		for _, n := range nodes {
			callback(n)
		}
		t.nodes <- nil
	case <-t.lookupWalker.closeCh:
		return
	}
}
