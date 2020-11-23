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
	"testing"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

func TestConnectToPivotNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	pid, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	sim.SetPivotNode(pid)

	id, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectToPivotNode(id)
	if err != nil {
		t.Fatal(err)
	}

	if sim.Net.GetConn(id, pid) == nil {
		t.Error("node did not connect to pivot node")
	}
}

func TestConnectToLastNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	n := 10

	ids, err := sim.AddNodes(n)
	if err != nil {
		t.Fatal(err)
	}

	id, err := sim.AddNode()
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectToLastNode(id)
	if err != nil {
		t.Fatal(err)
	}

	for _, i := range ids[:n-2] {
		if sim.Net.GetConn(id, i) != nil {
			t.Error("node connected to the node that is not the last")
		}
	}

	if sim.Net.GetConn(id, ids[n-1]) == nil {
		t.Error("node did not connect to the last node")
	}
}

func TestConnectToRandomNode(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	n := 10

	ids, err := sim.AddNodes(n)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectToRandomNode(ids[0])
	if err != nil {
		t.Fatal(err)
	}

	var cc int
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sim.Net.GetConn(ids[i], ids[j]) != nil {
				cc++
			}
		}
	}

	if cc != 1 {
		t.Errorf("expected one connection, got %v", cc)
	}
}

func TestConnectNodesFull(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(12)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectNodesFull(ids)
	if err != nil {
		t.Fatal(err)
	}

	testFull(t, sim, ids)
}

func testFull(t *testing.T, sim *Simulation, ids []discover.NodeID) {
	n := len(ids)
	var cc int
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sim.Net.GetConn(ids[i], ids[j]) != nil {
				cc++
			}
		}
	}

	want := n * (n - 1) / 2

	if cc != want {
		t.Errorf("expected %v connection, got %v", want, cc)
	}
}

func TestConnectNodesChain(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectNodesChain(ids)
	if err != nil {
		t.Fatal(err)
	}

	testChain(t, sim, ids)
}

func testChain(t *testing.T, sim *Simulation, ids []discover.NodeID) {
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := sim.Net.GetConn(ids[i], ids[j])
			if i == j-1 {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func TestConnectNodesRing(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	err = sim.ConnectNodesRing(ids)
	if err != nil {
		t.Fatal(err)
	}

	testRing(t, sim, ids)
}

func testRing(t *testing.T, sim *Simulation, ids []discover.NodeID) {
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := sim.Net.GetConn(ids[i], ids[j])
			if i == j-1 || (i == 0 && j == n-1) {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func TestConnectToNodesStar(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	centerIndex := 2

	err = sim.ConnectNodesStar(ids[centerIndex], ids)
	if err != nil {
		t.Fatal(err)
	}

	testStar(t, sim, ids, centerIndex)
}

func testStar(t *testing.T, sim *Simulation, ids []discover.NodeID, centerIndex int) {
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := sim.Net.GetConn(ids[i], ids[j])
			if i == centerIndex || j == centerIndex {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func TestConnectToNodesStarPivot(t *testing.T) {
	sim := New(noopServiceFuncMap)
	defer sim.Close()

	ids, err := sim.AddNodes(10)
	if err != nil {
		t.Fatal(err)
	}

	if len(sim.Net.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	pivotIndex := 4

	sim.SetPivotNode(ids[pivotIndex])

	err = sim.ConnectNodesStarPivot(ids)
	if err != nil {
		t.Fatal(err)
	}

	testStar(t, sim, ids, pivotIndex)
}
