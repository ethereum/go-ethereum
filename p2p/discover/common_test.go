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
		testNodes = makeTestNodes(lookupIteratorBuffer)
		wg        sync.WaitGroup
	)

	testIterator := func(it discutil.Iterator) {
		defer wg.Done()

		// Check reading nodes:
		nodes := discutil.ReadNodes(it, 20)
		sortByID(nodes)
		if err := checkNodesEqual(nodes, testNodes[:20]); err != nil {
			t.Error(err)
		}
		nodes = discutil.ReadNodes(it, 20)
		sortByID(nodes)
		if err := checkNodesEqual(nodes, testNodes[20:40]); err != nil {
			t.Error(err)
		}

		// Check close:
		it.Close()
		if it.Next() {
			t.Error("Next returned true after close")
		}
		if it.Node() != nil {
			t.Error("iterator has non-nil node after close")
		}
		it.Close() // shouldn't crash
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go testIterator(test.newIterator(nil))
	}

	test.serveOneLookup(testNodes[:10])
	test.serveOneLookup(testNodes[10:20])
	test.serveOneLookup(testNodes[20:])
	wg.Wait()

	test.close()
}

func TestLookupIteratorClose(t *testing.T) {
	test := newLookupWalkerTest()
	defer test.close()
	it := test.newIterator(nil)

	go func() {
		time.Sleep(200 * time.Millisecond)
		it.Close()
	}()
	it.Next()
}

// This test checks that the lookup iterator drops nodes when they're not being
// read fast enough.
func TestLookupIteratorDropStale(t *testing.T) {
	var (
		test       = newLookupWalkerTest()
		testNodes  = makeTestNodes(2 * lookupIteratorBuffer)
		lookupDone = make(chan struct{})
	)
	defer test.close()
	it := test.newIterator(nil)
	go func() {
		test.serveOneLookup(testNodes)
		close(lookupDone)
	}()

	// The first call to NextNode triggers the lookup and receives the first result
	// as soon as it becomes available.
	it.Next()
	if it.Node() != testNodes[0] {
		t.Fatalf("wrong result %d: got %v, want %v", 0, it.Node().ID(), testNodes[0])
	}

	// Now wait for the lookup to finish and read the remaining nodes.
	<-lookupDone
	for i := 0; i < lookupIteratorBuffer; i++ {
		it.Next()
		for _, tn := range testNodes[lookupIteratorBuffer:] {
			if it.Node() == tn {
				return
			}
		}
	}
	t.Fatal("didn't find any node from second half of testNodes")
}

// This test checks that the iterator kicks off a lookup when Next is called.
func TestLookupIteratorDrained(t *testing.T) {
	var (
		test      = newLookupWalkerTest()
		it        = test.newIterator(nil)
		testNodes = makeTestNodes(2 * lookupIteratorBuffer)
	)

	test.serveOneLookup(testNodes[:lookupIteratorBuffer])
	nodes := discutil.ReadNodes(it, lookupIteratorBuffer)
	sortByID(nodes)
	if err := checkNodesEqual(nodes, testNodes[:lookupIteratorBuffer]); err != nil {
		t.Fatal(err)
	}

	// Here the iterator buffer is drained and no lookup is running.

	// Request more nodes. This needs to start another lookup.
	go test.serveOneLookup(testNodes[lookupIteratorBuffer:])
	nodes = discutil.ReadNodes(it, 10)
	sortByID(nodes)
	if err := checkNodesEqual(nodes, testNodes[lookupIteratorBuffer:lookupIteratorBuffer+10]); err != nil {
		t.Fatal(err)
	}
}

func makeTestNodes(n int) []*enode.Node {
	nodes := make([]*enode.Node, n)
	for i := range nodes {
		var nodeID enode.ID
		binary.BigEndian.PutUint64(nodeID[:], uint64(i))
		nodes[i] = enode.SignNull(new(enr.Record), nodeID)
	}
	return nodes
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
