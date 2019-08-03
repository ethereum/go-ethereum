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

package les

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestClientPoolL10C100Free(t *testing.T) {
	testClientPool(t, 10, 100, 0, true)
}

func TestClientPoolL40C200Free(t *testing.T) {
	testClientPool(t, 40, 200, 0, true)
}

func TestClientPoolL100C300Free(t *testing.T) {
	testClientPool(t, 100, 300, 0, true)
}

func TestClientPoolL10C100P4(t *testing.T) {
	testClientPool(t, 10, 100, 4, false)
}

func TestClientPoolL40C200P30(t *testing.T) {
	testClientPool(t, 40, 200, 30, false)
}

func TestClientPoolL100C300P20(t *testing.T) {
	testClientPool(t, 100, 300, 20, false)
}

const testClientPoolTicks = 500000

type poolTestPeer int

func (i poolTestPeer) ID() enode.ID {
	return enode.ID{byte(i % 256), byte(i >> 8)}
}

func (i poolTestPeer) freeClientId() string {
	return fmt.Sprintf("addr #%d", i)
}

func (i poolTestPeer) updateCapacity(uint64) {}

func testClientPool(t *testing.T, connLimit, clientCount, paidCount int, randomDisconnect bool) {
	rand.Seed(time.Now().UnixNano())
	var (
		clock     mclock.Simulated
		db        = rawdb.NewMemoryDatabase()
		connected = make([]bool, clientCount)
		connTicks = make([]int, clientCount)
		disconnCh = make(chan int, clientCount)
		disconnFn = func(id enode.ID) {
			disconnCh <- int(id[0]) + int(id[1])<<8
		}
		pool = newClientPool(db, 1, 10000, &clock, disconnFn)
	)
	pool.setLimits(connLimit, uint64(connLimit))
	pool.setPriceFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	// pool should accept new peers up to its connected limit
	for i := 0; i < connLimit; i++ {
		if pool.connect(poolTestPeer(i), 0) != nil {
			connected[i] = true
		} else {
			t.Fatalf("Test peer #%d rejected", i)
		}
	}
	// since all accepted peers are new and should not be kicked out, the next one should be rejected
	if pool.connect(poolTestPeer(connLimit), 0) != nil {
		connected[connLimit] = true
		t.Fatalf("Peer accepted over connected limit")
	}

	// randomly connect and disconnect peers, expect to have a similar total connection time at the end
	for tickCounter := 0; tickCounter < testClientPoolTicks; tickCounter++ {
		clock.Run(1 * time.Second)
		//time.Sleep(time.Microsecond * 100)

		if tickCounter == testClientPoolTicks/4 {
			// give a positive balance to some of the peers
			amount := uint64(testClientPoolTicks / 2 * 1000000000) // enough for half of the simulation period
			for i := 0; i < paidCount; i++ {
				pool.addBalance(poolTestPeer(i).ID(), amount, false)
			}
		}

		i := rand.Intn(clientCount)
		if connected[i] {
			if randomDisconnect {
				pool.disconnect(poolTestPeer(i))
				connected[i] = false
				connTicks[i] += tickCounter
			}
		} else {
			if pool.connect(poolTestPeer(i), 0) != nil {
				connected[i] = true
				connTicks[i] -= tickCounter
			}
		}
	pollDisconnects:
		for {
			select {
			case i := <-disconnCh:
				pool.disconnect(poolTestPeer(i))
				if connected[i] {
					connTicks[i] += tickCounter
					connected[i] = false
				}
			default:
				break pollDisconnects
			}
		}
	}

	expTicks := testClientPoolTicks/2*connLimit/clientCount + testClientPoolTicks/2*(connLimit-paidCount)/(clientCount-paidCount)
	expMin := expTicks - expTicks/10
	expMax := expTicks + expTicks/10
	paidTicks := testClientPoolTicks/2*connLimit/clientCount + testClientPoolTicks/2
	paidMin := paidTicks - paidTicks/10
	paidMax := paidTicks + paidTicks/10

	// check if the total connected time of peers are all in the expected range
	for i, c := range connected {
		if c {
			connTicks[i] += testClientPoolTicks
		}
		min, max := expMin, expMax
		if i < paidCount {
			// expect a higher amount for clients with a positive balance
			min, max = paidMin, paidMax
		}
		if connTicks[i] < min || connTicks[i] > max {
			t.Errorf("Total connected time of test node #%d (%d) outside expected range (%d to %d)", i, connTicks[i], min, max)
		}
	}

	// a previously unknown peer should be accepted now
	if pool.connect(poolTestPeer(54321), 0) == nil {
		t.Fatalf("Previously unknown peer rejected")
	}

	// close and restart pool
	pool.stop()
	pool = newClientPool(db, 1, 10000, &clock, func(id enode.ID) {})
	pool.setLimits(connLimit, uint64(connLimit))

	// try connecting all known peers (connLimit should be filled up)
	for i := 0; i < clientCount; i++ {
		pool.connect(poolTestPeer(i), 0)
	}
	// expect pool to remember known nodes and kick out one of them to accept a new one
	if pool.connect(poolTestPeer(54322), 0) == nil {
		t.Errorf("Previously unknown peer rejected after restarting pool")
	}
	pool.stop()
}
