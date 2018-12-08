// Copyright 2017 The go-ethereum Authors
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

// Package light implements on-demand retrieval capable state and chain objects
// for the Ethereum Light Client.
package les

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestFreeClientPoolL10C100(t *testing.T) {
	testFreeClientPool(t, 10, 100)
}

func TestFreeClientPoolL40C200(t *testing.T) {
	testFreeClientPool(t, 40, 200)
}

func TestFreeClientPoolL100C300(t *testing.T) {
	testFreeClientPool(t, 100, 300)
}

const testFreeClientPoolTicks = 500000

func testFreeClientPool(t *testing.T, connLimit, clientCount int) {
	var (
		clock     mclock.Simulated
		db        = ethdb.NewMemDatabase()
		pool      = newFreeClientPool(db, connLimit, 10000, &clock)
		connected = make([]bool, clientCount)
		connTicks = make([]int, clientCount)
		disconnCh = make(chan int, clientCount)
	)
	peerId := func(i int) string {
		return fmt.Sprintf("test peer #%d", i)
	}
	disconnFn := func(i int) func() {
		return func() {
			disconnCh <- i
		}
	}

	// pool should accept new peers up to its connected limit
	for i := 0; i < connLimit; i++ {
		if pool.connect(peerId(i), disconnFn(i)) {
			connected[i] = true
		} else {
			t.Fatalf("Test peer #%d rejected", i)
		}
	}
	// since all accepted peers are new and should not be kicked out, the next one should be rejected
	if pool.connect(peerId(connLimit), disconnFn(connLimit)) {
		connected[connLimit] = true
		t.Fatalf("Peer accepted over connected limit")
	}

	// randomly connect and disconnect peers, expect to have a similar total connection time at the end
	for tickCounter := 0; tickCounter < testFreeClientPoolTicks; tickCounter++ {
		clock.Run(1 * time.Second)

		i := rand.Intn(clientCount)
		if connected[i] {
			pool.disconnect(peerId(i))
			connected[i] = false
			connTicks[i] += tickCounter
		} else {
			if pool.connect(peerId(i), disconnFn(i)) {
				connected[i] = true
				connTicks[i] -= tickCounter
			}
		}
	pollDisconnects:
		for {
			select {
			case i := <-disconnCh:
				pool.disconnect(peerId(i))
				if connected[i] {
					connTicks[i] += tickCounter
					connected[i] = false
				}
			default:
				break pollDisconnects
			}
		}
	}

	expTicks := testFreeClientPoolTicks * connLimit / clientCount
	expMin := expTicks - expTicks/10
	expMax := expTicks + expTicks/10

	// check if the total connected time of peers are all in the expected range
	for i, c := range connected {
		if c {
			connTicks[i] += testFreeClientPoolTicks
		}
		if connTicks[i] < expMin || connTicks[i] > expMax {
			t.Errorf("Total connected time of test node #%d (%d) outside expected range (%d to %d)", i, connTicks[i], expMin, expMax)
		}
	}

	// a previously unknown peer should be accepted now
	if !pool.connect("newPeer", func() {}) {
		t.Fatalf("Previously unknown peer rejected")
	}

	// close and restart pool
	pool.stop()
	pool = newFreeClientPool(db, connLimit, 10000, &clock)

	// try connecting all known peers (connLimit should be filled up)
	for i := 0; i < clientCount; i++ {
		pool.connect(peerId(i), func() {})
	}
	// expect pool to remember known nodes and kick out one of them to accept a new one
	if !pool.connect("newPeer2", func() {}) {
		t.Errorf("Previously unknown peer rejected after restarting pool")
	}
	pool.stop()
}
