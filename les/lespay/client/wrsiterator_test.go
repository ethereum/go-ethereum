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

func TestWrsIterator(t *testing.T) {
	ns := utils.NewNodeStateMachine(nil, nil, &mclock.Simulated{})
	st1 := ns.MustRegisterState(sfTest1)
	st2 := ns.MustRegisterState(sfTest2)
	st3 := ns.MustRegisterState(sfTest3)
	st4 := ns.MustRegisterState(sfTest4)
	enrField := ns.MustRegisterField(sfiEnr)
	weights := make([]uint64, iterTestNodeCount+1)
	wfn := func(i interface{}) uint64 {
		id := i.(enode.ID)
		idx := testNodeIndex(id)
		if idx <= 0 || idx > len(weights) {
			t.Errorf("Invalid node id %v", id)
		}
		return weights[idx]
	}
	w := NewWrsIterator(ns, st2, st3, sfTest4, sfiEnr, wfn, enode.ValidSchemesForTesting)
	ns.Start()
	for i := 1; i <= iterTestNodeCount; i++ {
		weights[i] = 1
		node := enode.SignNull(&enr.Record{}, testNodeID(i))
		ns.UpdateState(node.ID(), st1, 0, 0)
		ns.SetField(node.ID(), enrField, node.Record())
	}
	ch := make(chan *enode.Node)
	go func() {
		for w.Next() {
			ch <- w.Node()
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
	set := make(map[int]bool)
	expset := func() {
		n := next()
		if !set[n] {
			t.Errorf("Item returned by iterator not in the expected set (got %d)", n)
		}
		delete(set, n)
	}

	exp(0)
	ns.UpdateState(testNodeID(1), st2, 0, 0)
	ns.UpdateState(testNodeID(2), st2, 0, 0)
	ns.UpdateState(testNodeID(3), st2, 0, 0)
	set[1] = true
	set[2] = true
	set[3] = true
	expset()
	expset()
	expset()
	exp(0)
	ns.UpdateState(testNodeID(4), st2, 0, 0)
	ns.UpdateState(testNodeID(5), st2, 0, 0)
	ns.UpdateState(testNodeID(6), st2, 0, 0)
	ns.UpdateState(testNodeID(5), st3, 0, 0)
	set[4] = true
	set[6] = true
	expset()
	expset()
	exp(0)
	ns.UpdateState(testNodeID(1), 0, st4, 0)
	ns.UpdateState(testNodeID(2), 0, st4, 0)
	ns.UpdateState(testNodeID(3), 0, st4, 0)
	weights[2] = 0
	set[1] = true
	set[3] = true
	expset()
	expset()
	exp(0)
	weights[2] = 1
	ns.UpdateState(testNodeID(2), 0, st2, 0)
	ns.UpdateState(testNodeID(1), 0, st4, 0)
	ns.UpdateState(testNodeID(2), st2, st4, 0)
	ns.UpdateState(testNodeID(3), 0, st4, 0)
	set[1] = true
	set[2] = true
	set[3] = true
	expset()
	expset()
	expset()
	exp(0)
}
