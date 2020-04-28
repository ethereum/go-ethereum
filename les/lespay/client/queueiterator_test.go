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
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

var (
	sfTest1   = utils.NewFlag("test1")
	sfTest2   = utils.NewFlag("test2")
	sfTest3   = utils.NewFlag("test3")
	sfTest4   = utils.NewFlag("test4")
	testSetup = utils.NodeStateSetup{Flags: []nodestate.Flags{sfTest1, sfTest2, sfTest3, sfTest4}}
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
	ns := utils.NewNodeStateMachine(nil, nil, &mclock.Simulated{}, testSetup)
	st1 := ns.StateMask(sfTest1)
	st2 := ns.StateMask(sfTest2)
	st3 := ns.StateMask(sfTest3)
	st4 := ns.StateMask(sfTest4)
	qi := NewQueueIterator(ns, st2, st3, sfTest4)
	ns.Start()
	for i := 1; i <= iterTestNodeCount; i++ {
		ns.SetState(testNode(i), st1, 0, 0)
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
	ns.SetState(testNode(1), st2, 0, 0)
	ns.SetState(testNode(2), st2, 0, 0)
	ns.SetState(testNode(3), st2, 0, 0)
	exp(1)
	exp(2)
	exp(3)
	exp(0)
	ns.SetState(testNode(4), st2, 0, 0)
	ns.SetState(testNode(5), st2, 0, 0)
	ns.SetState(testNode(6), st2, 0, 0)
	ns.SetState(testNode(5), st3, 0, 0)
	exp(4)
	exp(6)
	exp(0)
	ns.SetState(testNode(1), 0, st4, 0)
	ns.SetState(testNode(2), 0, st4, 0)
	ns.SetState(testNode(3), 0, st4, 0)
	ns.SetState(testNode(2), st3, 0, 0)
	ns.SetState(testNode(2), 0, st3, 0)
	exp(1)
	exp(3)
	exp(2)
	exp(0)
}
