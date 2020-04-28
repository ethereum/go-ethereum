// Copyright 2020 The go-ethereum Authors
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

package client

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

var (
	testSetup = &nodestate.Setup{}
	sfTest1   = testSetup.NewFlag("test1")
	sfTest2   = testSetup.NewFlag("test2")
	sfTest3   = testSetup.NewFlag("test3")
	sfTest4   = testSetup.NewFlag("test4")
)

const iterTestNodeCount = 6

func testNodeID(i int) enode.ID {
	return enode.ID{42, byte(i % 256), byte(i / 256)}
}

func testNodeIndex(id enode.ID) int {
	if id[0] != 42 {
		return -1
	}
	return int(id[1]) + int(id[2])*256
}

type dummyIdentity enode.ID

func (id dummyIdentity) Verify(r *enr.Record, sig []byte) error { return nil }
func (id dummyIdentity) NodeAddr(r *enr.Record) []byte          { return id[:] }

func testNode(i int) *enode.Node {
	r := &enr.Record{}
	identity := dummyIdentity(testNodeID(i))
	r.SetSig(identity, []byte{42})
	n, _ := enode.New(identity, r)
	return n
}

func TestQueueIterator(t *testing.T) {
	ns := nodestate.NewNodeStateMachine(nil, nil, &mclock.Simulated{}, testSetup)
	qi := NewQueueIterator(ns, sfTest2, sfTest3, sfTest4)
	ns.Start()
	for i := 1; i <= iterTestNodeCount; i++ {
		ns.SetState(testNode(i), sfTest1, nodestate.Flags{}, 0)
	}
	ch := make(chan *enode.Node)
	go func() {
		for qi.Next() {
			ch <- qi.Node()
		}
		close(ch)
	}()
	next := func() int {
		select {
		case node := <-ch:
			return testNodeIndex(node.ID())
		case <-time.After(time.Millisecond * 200):
			return 0
		}
	}
	exp := func(i int) {
		n := next()
		if n != i {
			t.Errorf("Wrong item returned by iterator (expected %d, got %d)", i, n)
		}
	}
	exp(0)
	ns.SetState(testNode(1), sfTest2, nodestate.Flags{}, 0)
	ns.SetState(testNode(2), sfTest2, nodestate.Flags{}, 0)
	ns.SetState(testNode(3), sfTest2, nodestate.Flags{}, 0)
	exp(1)
	exp(2)
	exp(3)
	exp(0)
	ns.SetState(testNode(4), sfTest2, nodestate.Flags{}, 0)
	ns.SetState(testNode(5), sfTest2, nodestate.Flags{}, 0)
	ns.SetState(testNode(6), sfTest2, nodestate.Flags{}, 0)
	ns.SetState(testNode(5), sfTest3, nodestate.Flags{}, 0)
	exp(4)
	exp(6)
	exp(0)
	ns.SetState(testNode(1), nodestate.Flags{}, sfTest4, 0)
	ns.SetState(testNode(2), nodestate.Flags{}, sfTest4, 0)
	ns.SetState(testNode(3), nodestate.Flags{}, sfTest4, 0)
	ns.SetState(testNode(2), sfTest3, nodestate.Flags{}, 0)
	ns.SetState(testNode(2), nodestate.Flags{}, sfTest3, 0)
	exp(1)
	exp(3)
	exp(2)
	exp(0)
}
