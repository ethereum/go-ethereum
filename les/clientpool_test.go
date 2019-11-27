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
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"reflect"
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

const testClientPoolTicks = 100000

type poolTestPeer int

func (i poolTestPeer) ID() enode.ID {
	return enode.ID{byte(i % 256), byte(i >> 8)}
}

func (i poolTestPeer) freeClientId() string {
	return fmt.Sprintf("addr #%d", i)
}

func (i poolTestPeer) updateCapacity(uint64) {}

type poolTestPeerWithCap struct {
	poolTestPeer

	cap uint64
}

func (i *poolTestPeerWithCap) updateCapacity(cap uint64) { i.cap = cap }

func (i poolTestPeer) freezeClient() {}

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
		pool = newClientPool(db, 1, &clock, disconnFn)
	)
	pool.disableBias = true
	pool.setLimits(connLimit, uint64(connLimit))
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	// pool should accept new peers up to its connected limit
	for i := 0; i < connLimit; i++ {
		if pool.connect(poolTestPeer(i), 0) {
			connected[i] = true
		} else {
			t.Fatalf("Test peer #%d rejected", i)
		}
	}
	// randomly connect and disconnect peers, expect to have a similar total connection time at the end
	for tickCounter := 0; tickCounter < testClientPoolTicks; tickCounter++ {
		clock.Run(1 * time.Second)

		if tickCounter == testClientPoolTicks/4 {
			// give a positive balance to some of the peers
			amount := testClientPoolTicks / 2 * int64(time.Second) // enough for half of the simulation period
			for i := 0; i < paidCount; i++ {
				pool.addBalance(poolTestPeer(i).ID(), amount, "")
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
			if pool.connect(poolTestPeer(i), 0) {
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
	expMin := expTicks - expTicks/5
	expMax := expTicks + expTicks/5
	paidTicks := testClientPoolTicks/2*connLimit/clientCount + testClientPoolTicks/2
	paidMin := paidTicks - paidTicks/5
	paidMax := paidTicks + paidTicks/5

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
	pool.stop()
}

func TestConnectPaidClient(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(db, 1, &clock, nil)
	defer pool.stop()
	pool.setLimits(10, uint64(10))
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	// Add balance for an external client and mark it as paid client
	pool.addBalance(poolTestPeer(0).ID(), 1000, "")

	if !pool.connect(poolTestPeer(0), 10) {
		t.Fatalf("Failed to connect paid client")
	}
}

func TestConnectPaidClientToSmallPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(db, 1, &clock, nil)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	// Add balance for an external client and mark it as paid client
	pool.addBalance(poolTestPeer(0).ID(), 1000, "")

	// Connect a fat paid client to pool, should reject it.
	if pool.connect(poolTestPeer(0), 100) {
		t.Fatalf("Connected fat paid client, should reject it")
	}
}

func TestConnectPaidClientToFullPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	removeFn := func(enode.ID) {} // Noop
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	for i := 0; i < 10; i++ {
		pool.addBalance(poolTestPeer(i).ID(), 1000000000, "")
		pool.connect(poolTestPeer(i), 1)
	}
	pool.addBalance(poolTestPeer(11).ID(), 1000, "") // Add low balance to new paid client
	if pool.connect(poolTestPeer(11), 1) {
		t.Fatalf("Low balance paid client should be rejected")
	}
	clock.Run(time.Second)
	pool.addBalance(poolTestPeer(12).ID(), 1000000000*60*3, "") // Add high balance to new paid client
	if !pool.connect(poolTestPeer(12), 1) {
		t.Fatalf("High balance paid client should be accpected")
	}
}

func TestPaidClientKickedOut(t *testing.T) {
	var (
		clock    mclock.Simulated
		db       = rawdb.NewMemoryDatabase()
		kickedCh = make(chan int, 1)
	)
	removeFn := func(id enode.ID) { kickedCh <- int(id[0]) }
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	for i := 0; i < 10; i++ {
		pool.addBalance(poolTestPeer(i).ID(), 1000000000, "") // 1 second allowance
		pool.connect(poolTestPeer(i), 1)
		clock.Run(time.Millisecond)
	}
	clock.Run(time.Second)
	clock.Run(connectedBias)
	if !pool.connect(poolTestPeer(11), 0) {
		t.Fatalf("Free client should be accectped")
	}
	select {
	case id := <-kickedCh:
		if id != 0 {
			t.Fatalf("Kicked client mismatch, want %v, got %v", 0, id)
		}
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("timeout")
	}
}

func TestConnectFreeClient(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(db, 1, &clock, nil)
	defer pool.stop()
	pool.setLimits(10, uint64(10))
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})
	if !pool.connect(poolTestPeer(0), 10) {
		t.Fatalf("Failed to connect free client")
	}
}

func TestConnectFreeClientToFullPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	removeFn := func(enode.ID) {} // Noop
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	for i := 0; i < 10; i++ {
		pool.connect(poolTestPeer(i), 1)
	}
	if pool.connect(poolTestPeer(11), 1) {
		t.Fatalf("New free client should be rejected")
	}
	clock.Run(time.Minute)
	if pool.connect(poolTestPeer(12), 1) {
		t.Fatalf("New free client should be rejected")
	}
	clock.Run(time.Millisecond)
	clock.Run(4 * time.Minute)
	if !pool.connect(poolTestPeer(13), 1) {
		t.Fatalf("Old client connects more than 5min should be kicked")
	}
}

func TestFreeClientKickedOut(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 10)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) }
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	for i := 0; i < 10; i++ {
		pool.connect(poolTestPeer(i), 1)
		clock.Run(time.Millisecond)
	}
	if pool.connect(poolTestPeer(10), 1) {
		t.Fatalf("New free client should be rejected")
	}
	clock.Run(5 * time.Minute)
	for i := 0; i < 10; i++ {
		pool.connect(poolTestPeer(i+10), 1)
	}
	for i := 0; i < 10; i++ {
		select {
		case id := <-kicked:
			if id >= 10 {
				t.Fatalf("Old client should be kicked, now got: %d", id)
			}
		case <-time.NewTimer(time.Second).C:
			t.Fatalf("timeout")
		}
	}
}

func TestPositiveBalanceCalculation(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 10)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) } // Noop
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	pool.addBalance(poolTestPeer(0).ID(), int64(time.Minute*3), "")
	pool.connect(poolTestPeer(0), 10)
	clock.Run(time.Minute)

	pool.disconnect(poolTestPeer(0))
	pb := pool.ndb.getOrNewPB(poolTestPeer(0).ID())
	if pb.value != uint64(time.Minute*2) {
		t.Fatalf("Positive balance mismatch, want %v, got %v", uint64(time.Minute*2), pb.value)
	}
}

func TestDowngradePriorityClient(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 10)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) } // Noop
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	p := &poolTestPeerWithCap{
		poolTestPeer: poolTestPeer(0),
	}
	pool.addBalance(p.ID(), int64(time.Minute), "")
	pool.connect(p, 10)
	if p.cap != 10 {
		t.Fatalf("The capcacity of priority peer hasn't been updated, got: %d", p.cap)
	}

	clock.Run(time.Minute)             // All positive balance should be used up.
	time.Sleep(300 * time.Millisecond) // Ensure the callback is called
	if p.cap != 1 {
		t.Fatalf("The capcacity of peer should be downgraded, got: %d", p.cap)
	}
	pb := pool.ndb.getOrNewPB(poolTestPeer(0).ID())
	if pb.value != 0 {
		t.Fatalf("Positive balance mismatch, want %v, got %v", 0, pb.value)
	}

	pool.addBalance(poolTestPeer(0).ID(), int64(time.Minute), "")
	pb = pool.ndb.getOrNewPB(poolTestPeer(0).ID())
	if pb.value != uint64(time.Minute) {
		t.Fatalf("Positive balance mismatch, want %v, got %v", uint64(time.Minute), pb.value)
	}
}

func TestNegativeBalanceCalculation(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 10)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) } // Noop
	pool := newClientPool(db, 1, &clock, removeFn)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(priceFactors{1, 0, 1}, priceFactors{1, 0, 1})

	for i := 0; i < 10; i++ {
		pool.connect(poolTestPeer(i), 1)
	}
	clock.Run(time.Second)

	for i := 0; i < 10; i++ {
		pool.disconnect(poolTestPeer(i))
		nb := pool.ndb.getOrNewNB(poolTestPeer(i).freeClientId())
		if nb.logValue != 0 {
			t.Fatalf("Short connection shouldn't be recorded")
		}
	}

	for i := 0; i < 10; i++ {
		pool.connect(poolTestPeer(i), 1)
	}
	clock.Run(time.Minute)
	for i := 0; i < 10; i++ {
		pool.disconnect(poolTestPeer(i))
		nb := pool.ndb.getOrNewNB(poolTestPeer(i).freeClientId())
		nb.logValue -= pool.logOffset(clock.Now())
		nb.logValue /= fixedPointMultiplier
		if nb.logValue != int64(math.Log(float64(time.Minute/time.Second))) {
			t.Fatalf("Negative balance mismatch, want %v, got %v", int64(math.Log(float64(time.Minute/time.Second))), nb.logValue)
		}
	}
}

func TestNodeDB(t *testing.T) {
	ndb := newNodeDB(rawdb.NewMemoryDatabase(), mclock.System{})
	defer ndb.close()

	if !bytes.Equal(ndb.verbuf[:], []byte{0x00, nodeDBVersion}) {
		t.Fatalf("version buffer mismatch, want %v, got %v", []byte{0x00, nodeDBVersion}, ndb.verbuf)
	}
	var cases = []struct {
		id       enode.ID
		ip       string
		balance  interface{}
		positive bool
	}{
		{enode.ID{0x00, 0x01, 0x02}, "", posBalance{value: 100}, true},
		{enode.ID{0x00, 0x01, 0x02}, "", posBalance{value: 200}, true},
		{enode.ID{}, "127.0.0.1", negBalance{logValue: 10}, false},
		{enode.ID{}, "127.0.0.1", negBalance{logValue: 20}, false},
	}
	for _, c := range cases {
		if c.positive {
			ndb.setPB(c.id, c.balance.(posBalance))
			if pb := ndb.getOrNewPB(c.id); !reflect.DeepEqual(pb, c.balance.(posBalance)) {
				t.Fatalf("Positive balance mismatch, want %v, got %v", c.balance.(posBalance), pb)
			}
		} else {
			ndb.setNB(c.ip, c.balance.(negBalance))
			if nb := ndb.getOrNewNB(c.ip); !reflect.DeepEqual(nb, c.balance.(negBalance)) {
				t.Fatalf("Negative balance mismatch, want %v, got %v", c.balance.(negBalance), nb)
			}
		}
	}
	for _, c := range cases {
		if c.positive {
			ndb.delPB(c.id)
			if pb := ndb.getOrNewPB(c.id); !reflect.DeepEqual(pb, posBalance{}) {
				t.Fatalf("Positive balance mismatch, want %v, got %v", posBalance{}, pb)
			}
		} else {
			ndb.delNB(c.ip)
			if nb := ndb.getOrNewNB(c.ip); !reflect.DeepEqual(nb, negBalance{}) {
				t.Fatalf("Negative balance mismatch, want %v, got %v", negBalance{}, nb)
			}
		}
	}
	ndb.setCumulativeTime(100)
	if ndb.getCumulativeTime() != 100 {
		t.Fatalf("Cumulative time mismatch, want %v, got %v", 100, ndb.getCumulativeTime())
	}
}

func TestNodeDBExpiration(t *testing.T) {
	var (
		iterated int
		done     = make(chan struct{}, 1)
	)
	callback := func(now mclock.AbsTime, b negBalance) bool {
		iterated += 1
		return true
	}
	clock := &mclock.Simulated{}
	ndb := newNodeDB(rawdb.NewMemoryDatabase(), clock)
	defer ndb.close()
	ndb.nbEvictCallBack = callback
	ndb.cleanupHook = func() { done <- struct{}{} }

	var cases = []struct {
		ip      string
		balance negBalance
	}{
		{"127.0.0.1", negBalance{logValue: 1}},
		{"127.0.0.2", negBalance{logValue: 1}},
		{"127.0.0.3", negBalance{logValue: 1}},
		{"127.0.0.4", negBalance{logValue: 1}},
	}
	for _, c := range cases {
		ndb.setNB(c.ip, c.balance)
	}
	time.Sleep(100 * time.Millisecond) // Ensure the db expirer is registered.
	clock.Run(time.Hour + time.Minute)
	select {
	case <-done:
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("timeout")
	}
	if iterated != 4 {
		t.Fatalf("Failed to evict useless negative balances, want %v, got %d", 4, iterated)
	}

	for _, c := range cases {
		ndb.setNB(c.ip, c.balance)
	}
	clock.Run(time.Hour + time.Minute)
	select {
	case <-done:
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("timeout")
	}
	if iterated != 8 {
		t.Fatalf("Failed to evict useless negative balances, want %v, got %d", 4, iterated)
	}
}
