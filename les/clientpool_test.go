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
	lps "github.com/ethereum/go-ethereum/les/lespay/server"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
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

type poolTestPeer struct {
	node            *enode.Node
	index           int
	disconnCh       chan int
	cap             uint64
	inactiveAllowed bool
}

func testStateMachine() *nodestate.NodeStateMachine {
	return nodestate.NewNodeStateMachine(nil, nil, mclock.System{}, serverSetup)

}

func newPoolTestPeer(i int, disconnCh chan int) *poolTestPeer {
	return &poolTestPeer{
		index:     i,
		disconnCh: disconnCh,
		node:      enode.SignNull(&enr.Record{}, enode.ID{byte(i % 256), byte(i >> 8)}),
	}
}

func (i *poolTestPeer) Node() *enode.Node {
	return i.node
}

func (i *poolTestPeer) freeClientId() string {
	return fmt.Sprintf("addr #%d", i.index)
}

func (i *poolTestPeer) updateCapacity(cap uint64) {
	i.cap = cap
}

func (i *poolTestPeer) freeze() {}

func (i *poolTestPeer) allowInactive() bool {
	return i.inactiveAllowed
}

func getBalance(pool *clientPool, p *poolTestPeer) (pos, neg uint64) {
	temp := pool.ns.GetField(p.node, clientInfoField) == nil
	if temp {
		pool.ns.SetField(p.node, connAddressField, p.freeClientId())
	}
	n, _ := pool.ns.GetField(p.node, pool.BalanceField).(*lps.NodeBalance)
	pos, neg = n.GetBalance()
	if temp {
		pool.ns.SetField(p.node, connAddressField, nil)
	}
	return
}

func addBalance(pool *clientPool, id enode.ID, amount int64) {
	pool.forClients([]enode.ID{id}, func(c *clientInfo) {
		c.balance.AddBalance(amount)
	})
}

func checkDiff(a, b uint64) bool {
	maxDiff := (a + b) / 2000
	if maxDiff < 1 {
		maxDiff = 1
	}
	return a > b+maxDiff || b > a+maxDiff
}

func testClientPool(t *testing.T, activeLimit, clientCount, paidCount int, randomDisconnect bool) {
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
		pool = newClientPool(testStateMachine(), db, 1, 0, &clock, disconnFn)
	)
	pool.ns.Start()

	pool.setLimits(activeLimit, uint64(activeLimit))
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	// pool should accept new peers up to its connected limit
	for i := 0; i < activeLimit; i++ {
		if cap, _ := pool.connect(newPoolTestPeer(i, disconnCh)); cap != 0 {
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
				addBalance(pool, newPoolTestPeer(i, disconnCh).node.ID(), amount)
			}
		}

		i := rand.Intn(clientCount)
		if connected[i] {
			if randomDisconnect {
				pool.disconnect(newPoolTestPeer(i, disconnCh))
				connected[i] = false
				connTicks[i] += tickCounter
			}
		} else {
			if cap, _ := pool.connect(newPoolTestPeer(i, disconnCh)); cap != 0 {
				connected[i] = true
				connTicks[i] -= tickCounter
			} else {
				pool.disconnect(newPoolTestPeer(i, disconnCh))
			}
		}
	pollDisconnects:
		for {
			select {
			case i := <-disconnCh:
				pool.disconnect(newPoolTestPeer(i, disconnCh))
				if connected[i] {
					connTicks[i] += tickCounter
					connected[i] = false
				}
			default:
				break pollDisconnects
			}
		}
	}

	expTicks := testClientPoolTicks/2*activeLimit/clientCount + testClientPoolTicks/2*(activeLimit-paidCount)/(clientCount-paidCount)
	expMin := expTicks - expTicks/5
	expMax := expTicks + expTicks/5
	paidTicks := testClientPoolTicks/2*activeLimit/clientCount + testClientPoolTicks/2
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

func testPriorityConnect(t *testing.T, pool *clientPool, p *poolTestPeer, cap uint64, expSuccess bool) {
	if cap, _ := pool.connect(p); cap == 0 {
		if expSuccess {
			t.Fatalf("Failed to connect paid client")
		} else {
			return
		}
	}
	if _, err := pool.setCapacity(p.node, "", cap, defaultConnectedBias, true); err != nil {
		if expSuccess {
			t.Fatalf("Failed to raise capacity of paid client")
		} else {
			return
		}
	}
	if !expSuccess {
		t.Fatalf("Should reject high capacity paid client")
	}
}

func TestConnectPaidClient(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, func(id enode.ID) {})
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10))
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	// Add balance for an external client and mark it as paid client
	addBalance(pool, newPoolTestPeer(0, nil).node.ID(), int64(time.Minute))
	testPriorityConnect(t, pool, newPoolTestPeer(0, nil), 10, true)
}

func TestConnectPaidClientToSmallPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, func(id enode.ID) {})
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	// Add balance for an external client and mark it as paid client
	addBalance(pool, newPoolTestPeer(0, nil).node.ID(), int64(time.Minute))

	// Connect a fat paid client to pool, should reject it.
	testPriorityConnect(t, pool, newPoolTestPeer(0, nil), 100, false)
}

func TestConnectPaidClientToFullPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	removeFn := func(enode.ID) {} // Noop
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	for i := 0; i < 10; i++ {
		addBalance(pool, newPoolTestPeer(i, nil).node.ID(), int64(time.Second*20))
		pool.connect(newPoolTestPeer(i, nil))
	}
	addBalance(pool, newPoolTestPeer(11, nil).node.ID(), int64(time.Second*2)) // Add low balance to new paid client
	if cap, _ := pool.connect(newPoolTestPeer(11, nil)); cap != 0 {
		t.Fatalf("Low balance paid client should be rejected")
	}
	clock.Run(time.Second)
	addBalance(pool, newPoolTestPeer(12, nil).node.ID(), int64(time.Minute*5)) // Add high balance to new paid client
	if cap, _ := pool.connect(newPoolTestPeer(12, nil)); cap == 0 {
		t.Fatalf("High balance paid client should be accepted")
	}
}

func TestPaidClientKickedOut(t *testing.T) {
	var (
		clock    mclock.Simulated
		db       = rawdb.NewMemoryDatabase()
		kickedCh = make(chan int, 100)
	)
	removeFn := func(id enode.ID) {
		kickedCh <- int(id[0])
	}
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	pool.bt.SetExpirationTCs(0, 0)
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	for i := 0; i < 10; i++ {
		addBalance(pool, newPoolTestPeer(i, kickedCh).node.ID(), 10000000000) // 10 second allowance
		pool.connect(newPoolTestPeer(i, kickedCh))
		clock.Run(time.Millisecond)
	}
	clock.Run(defaultConnectedBias + time.Second*11)
	if cap, _ := pool.connect(newPoolTestPeer(11, kickedCh)); cap == 0 {
		t.Fatalf("Free client should be accepted")
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
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, func(id enode.ID) {})
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10))
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})
	if cap, _ := pool.connect(newPoolTestPeer(0, nil)); cap == 0 {
		t.Fatalf("Failed to connect free client")
	}
	testPriorityConnect(t, pool, newPoolTestPeer(0, nil), 2, false)
}

func TestConnectFreeClientToFullPool(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	removeFn := func(enode.ID) {} // Noop
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	for i := 0; i < 10; i++ {
		pool.connect(newPoolTestPeer(i, nil))
	}
	if cap, _ := pool.connect(newPoolTestPeer(11, nil)); cap != 0 {
		t.Fatalf("New free client should be rejected")
	}
	clock.Run(time.Minute)
	if cap, _ := pool.connect(newPoolTestPeer(12, nil)); cap != 0 {
		t.Fatalf("New free client should be rejected")
	}
	clock.Run(time.Millisecond)
	clock.Run(4 * time.Minute)
	if cap, _ := pool.connect(newPoolTestPeer(13, nil)); cap == 0 {
		t.Fatalf("Old client connects more than 5min should be kicked")
	}
}

func TestFreeClientKickedOut(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 100)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) }
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	for i := 0; i < 10; i++ {
		pool.connect(newPoolTestPeer(i, kicked))
		clock.Run(time.Millisecond)
	}
	if cap, _ := pool.connect(newPoolTestPeer(10, kicked)); cap != 0 {
		t.Fatalf("New free client should be rejected")
	}
	select {
	case <-kicked:
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("timeout")
	}
	pool.disconnect(newPoolTestPeer(10, kicked))
	clock.Run(5 * time.Minute)
	for i := 0; i < 10; i++ {
		pool.connect(newPoolTestPeer(i+10, kicked))
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
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	addBalance(pool, newPoolTestPeer(0, kicked).node.ID(), int64(time.Minute*3))
	testPriorityConnect(t, pool, newPoolTestPeer(0, kicked), 10, true)
	clock.Run(time.Minute)

	pool.disconnect(newPoolTestPeer(0, kicked))
	pb, _ := getBalance(pool, newPoolTestPeer(0, kicked))
	if checkDiff(pb, uint64(time.Minute*2)) {
		t.Fatalf("Positive balance mismatch, want %v, got %v", uint64(time.Minute*2), pb)
	}
}

func TestDowngradePriorityClient(t *testing.T) {
	var (
		clock  mclock.Simulated
		db     = rawdb.NewMemoryDatabase()
		kicked = make(chan int, 10)
	)
	removeFn := func(id enode.ID) { kicked <- int(id[0]) } // Noop
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, removeFn)
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 1})

	p := newPoolTestPeer(0, kicked)
	addBalance(pool, p.node.ID(), int64(time.Minute))
	testPriorityConnect(t, pool, p, 10, true)
	if p.cap != 10 {
		t.Fatalf("The capacity of priority peer hasn't been updated, got: %d", p.cap)
	}

	clock.Run(time.Minute)             // All positive balance should be used up.
	time.Sleep(300 * time.Millisecond) // Ensure the callback is called
	if p.cap != 1 {
		t.Fatalf("The capcacity of peer should be downgraded, got: %d", p.cap)
	}
	pb, _ := getBalance(pool, newPoolTestPeer(0, kicked))
	if pb != 0 {
		t.Fatalf("Positive balance mismatch, want %v, got %v", 0, pb)
	}

	addBalance(pool, newPoolTestPeer(0, kicked).node.ID(), int64(time.Minute))
	pb, _ = getBalance(pool, newPoolTestPeer(0, kicked))
	if checkDiff(pb, uint64(time.Minute)) {
		t.Fatalf("Positive balance mismatch, want %v, got %v", uint64(time.Minute), pb)
	}
}

func TestNegativeBalanceCalculation(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, func(id enode.ID) {})
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(10, uint64(10)) // Total capacity limit is 10
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1e-3, CapacityFactor: 0, RequestFactor: 1}, lps.PriceFactors{TimeFactor: 1e-3, CapacityFactor: 0, RequestFactor: 1})

	for i := 0; i < 10; i++ {
		pool.connect(newPoolTestPeer(i, nil))
	}
	clock.Run(time.Second)

	for i := 0; i < 10; i++ {
		pool.disconnect(newPoolTestPeer(i, nil))
		_, nb := getBalance(pool, newPoolTestPeer(i, nil))
		if nb != 0 {
			t.Fatalf("Short connection shouldn't be recorded")
		}
	}
	for i := 0; i < 10; i++ {
		pool.connect(newPoolTestPeer(i, nil))
	}
	clock.Run(time.Minute)
	for i := 0; i < 10; i++ {
		pool.disconnect(newPoolTestPeer(i, nil))
		_, nb := getBalance(pool, newPoolTestPeer(i, nil))
		if checkDiff(nb, uint64(time.Minute)/1000) {
			t.Fatalf("Negative balance mismatch, want %v, got %v", uint64(time.Minute)/1000, nb)
		}
	}
}

func TestInactiveClient(t *testing.T) {
	var (
		clock mclock.Simulated
		db    = rawdb.NewMemoryDatabase()
	)
	pool := newClientPool(testStateMachine(), db, 1, defaultConnectedBias, &clock, func(id enode.ID) {})
	pool.ns.Start()
	defer pool.stop()
	pool.setLimits(2, uint64(2))

	p1 := newPoolTestPeer(1, nil)
	p1.inactiveAllowed = true
	p2 := newPoolTestPeer(2, nil)
	p2.inactiveAllowed = true
	p3 := newPoolTestPeer(3, nil)
	p3.inactiveAllowed = true
	addBalance(pool, p1.node.ID(), 1000*int64(time.Second))
	addBalance(pool, p3.node.ID(), 2000*int64(time.Second))
	// p1: 1000  p2: 0  p3: 2000
	p1.cap, _ = pool.connect(p1)
	if p1.cap != 1 {
		t.Fatalf("Failed to connect peer #1")
	}
	p2.cap, _ = pool.connect(p2)
	if p2.cap != 1 {
		t.Fatalf("Failed to connect peer #2")
	}
	p3.cap, _ = pool.connect(p3)
	if p3.cap != 1 {
		t.Fatalf("Failed to connect peer #3")
	}
	if p2.cap != 0 {
		t.Fatalf("Failed to deactivate peer #2")
	}
	addBalance(pool, p2.node.ID(), 3000*int64(time.Second))
	// p1: 1000  p2: 3000  p3: 2000
	if p2.cap != 1 {
		t.Fatalf("Failed to activate peer #2")
	}
	if p1.cap != 0 {
		t.Fatalf("Failed to deactivate peer #1")
	}
	addBalance(pool, p2.node.ID(), -2500*int64(time.Second))
	// p1: 1000  p2: 500  p3: 2000
	if p1.cap != 1 {
		t.Fatalf("Failed to activate peer #1")
	}
	if p2.cap != 0 {
		t.Fatalf("Failed to deactivate peer #2")
	}
	pool.setDefaultFactors(lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 0}, lps.PriceFactors{TimeFactor: 1, CapacityFactor: 0, RequestFactor: 0})
	p4 := newPoolTestPeer(4, nil)
	addBalance(pool, p4.node.ID(), 1500*int64(time.Second))
	// p1: 1000  p2: 500  p3: 2000  p4: 1500
	p4.cap, _ = pool.connect(p4)
	if p4.cap != 1 {
		t.Fatalf("Failed to activate peer #4")
	}
	if p1.cap != 0 {
		t.Fatalf("Failed to deactivate peer #1")
	}
	clock.Run(time.Second * 600)
	// manually trigger a check to avoid a long real-time wait
	pool.ns.SetState(p1.node, pool.UpdateFlag, nodestate.Flags{}, 0)
	pool.ns.SetState(p1.node, nodestate.Flags{}, pool.UpdateFlag, 0)
	// p1: 1000  p2: 500  p3: 2000  p4: 900
	if p1.cap != 1 {
		t.Fatalf("Failed to activate peer #1")
	}
	if p4.cap != 0 {
		t.Fatalf("Failed to deactivate peer #4")
	}
	pool.disconnect(p2)
	pool.disconnect(p4)
	addBalance(pool, p1.node.ID(), -1000*int64(time.Second))
	if p1.cap != 1 {
		t.Fatalf("Should not deactivate peer #1")
	}
	if p2.cap != 0 {
		t.Fatalf("Should not activate peer #2")
	}
}
