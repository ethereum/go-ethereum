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

// Package flowcontrol implements a client side flow control mechanism
package flowcontrol

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

type testNode struct {
	node                *ClientNode
	bufLimit, bandwidth uint64
	waitUntil           mclock.AbsTime
	index, totalCost    uint64
}

const (
	testMaxCost = 1000000
	testLength  = 100000
)

func (n *testNode) send(t *testing.T, now mclock.AbsTime) bool {
	if now < n.waitUntil {
		return false
	}
	n.index++
	if ok, _, _ := n.node.AcceptRequest(n.index, testMaxCost); !ok {
		t.Fatalf("Rejected request after expected waiting time has passed")
	}
	rcost := uint64(rand.Int63n(testMaxCost))
	bv, _ := n.node.RequestProcessed(n.index, testMaxCost, rcost)
	if bv < testMaxCost {
		n.waitUntil = now + mclock.AbsTime((testMaxCost-bv)*1001000/n.bandwidth)
	}
	//n.waitUntil = now + mclock.AbsTime(float64(testMaxCost)*1001000/float64(n.bandwidth)*(1-float64(bv)/float64(n.bufLimit)))
	n.totalCost += rcost
	return true
}

func TestConstantTotalBandwidth(t *testing.T) {
	testConstantTotalBandwidth(t, 10, 1, 0)
	testConstantTotalBandwidth(t, 10, 1, 1)
	testConstantTotalBandwidth(t, 30, 1, 0)
	testConstantTotalBandwidth(t, 30, 2, 3)
	testConstantTotalBandwidth(t, 100, 1, 0)
	testConstantTotalBandwidth(t, 100, 3, 5)
	testConstantTotalBandwidth(t, 100, 5, 10)
}

func testConstantTotalBandwidth(t *testing.T, nodeCount, maxCapacityNodes, randomSend int) {
	clock := &mclock.Simulated{}
	nodes := make([]*testNode, nodeCount)
	var totalBandwidth uint64
	for i, _ := range nodes {
		nodes[i] = &testNode{bandwidth: uint64(50000 + rand.Intn(100000))}
		totalBandwidth += nodes[i].bandwidth
	}
	m := NewClientManager(PieceWiseLinear{{0, totalBandwidth}}, clock)
	for _, n := range nodes {
		n.bufLimit = n.bandwidth * 6000 //uint64(2000+rand.Intn(10000))
		n.node = NewClientNode(m, &ServerParams{BufLimit: n.bufLimit, MinRecharge: n.bandwidth})
	}
	maxNodes := make([]int, maxCapacityNodes)
	for i, _ := range maxNodes {
		// we don't care if some indexes are selected multiple times
		// in that case we have fewer max nodes
		maxNodes[i] = rand.Intn(nodeCount)
	}

	for i := 0; i < testLength; i++ {
		now := clock.Now()
		for _, idx := range maxNodes {
			for nodes[idx].send(t, now) {
			}
		}
		if rand.Intn(testLength) < maxCapacityNodes*3 {
			maxNodes[rand.Intn(maxCapacityNodes)] = rand.Intn(nodeCount)
		}

		sendCount := randomSend
		for sendCount > 0 {
			if nodes[rand.Intn(nodeCount)].send(t, now) {
				sendCount--
			}
		}

		clock.Run(time.Millisecond)
	}

	var totalCost uint64
	for _, n := range nodes {
		totalCost += n.totalCost
	}
	ratio := float64(totalCost) / float64(totalBandwidth) / testLength
	if ratio < 0.98 || ratio > 1.02 {
		t.Errorf("totalCost/totalBandwidth/testLength ratio incorrect (expected: 1, got: %f)", ratio)
	}

}
