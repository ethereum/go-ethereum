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
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
)

type testLoadPeer struct {
	fcClient *flowcontrol.ClientNode
	fcServer *flowcontrol.ServerNode
	serveCh  chan uint64
}

func newTestLoadPeer(t *testing.T, client *testLoadClient, server *testLoadServer, params *flowcontrol.ServerParams, free bool, quit chan struct{}, clock mclock.Clock) *testLoadPeer {
	cm := server.fcManager
	if free {
		cm = server.fcManagerFree
	}

	peer := &testLoadPeer{
		fcClient: flowcontrol.NewClientNode(cm, params),
		fcServer: flowcontrol.NewServerNode(params, clock),
		serveCh:  make(chan uint64, 1000),
	}
	client.dist.registerTestPeer(peer)

	type reply struct {
		time      mclock.AbsTime
		reqID, bv uint64
	}
	replyCh := make(chan reply, 1000)

	go func() {
		avgServeTime := time.Millisecond
		randMax := avgServeTime * 2

		for {
			select {
			case reqID := <-peer.serveCh:
				ok, bufShort := peer.fcClient.AcceptRequest(testRequestCost)
				if !ok {
					recharge := time.Duration(bufShort * 1000000 / params.MinRecharge)
					t.Errorf("Request came too early (%v)", recharge)
				}
				//testClock.Sleep(time.Microsecond * 500)
				serveTime := time.Duration(rand.Int63n(int64(randMax)))
				randMax += (avgServeTime - serveTime) / 16
				clock.Sleep(serveTime)
				bvAfter, _ := peer.fcClient.RequestProcessed()
				replyCh <- reply{clock.Now(), reqID, bvAfter}
				atomic.AddUint64(&server.count, 1)
			case <-quit:
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case reply := <-replyCh:
				wait := time.Duration(reply.time-clock.Now()) + testMessageDelay
				if wait > 0 {
					clock.Sleep(wait)
				}
				peer.fcServer.ReceivedReply(reply.reqID, reply.bv)
			case <-quit:
				return
			}
		}
	}()

	return peer
}

func (p *testLoadPeer) waitBefore(maxCost uint64) (time.Duration, float64) {
	return p.fcServer.CanSend(maxCost)
}

func (p *testLoadPeer) canQueue() bool {
	return true
}

func (p *testLoadPeer) queueSend(f func()) {
	f()
}

type testLoadTask struct {
	procTime time.Duration
	finished chan struct{}
}

type testLoadServer struct {
	fcManager, fcManagerFree *flowcontrol.ClientManager
	count                    uint64
}

func newTestLoadServer(capacity int, quit chan struct{}, clock mclock.Clock) *testLoadServer {
	f := flowcontrol.NewClientManager(16, float64(capacity)/1000, clock, nil)
	s := &testLoadServer{
		fcManager:     flowcontrol.NewClientManager(16, float64(capacity)/1000, clock, f),
		fcManagerFree: f,
	}
	go func() {
		<-quit
		s.fcManager.Stop()
		s.fcManagerFree.Stop()
	}()
	return s
}

func (s *testLoadServer) requestsProcessed() uint64 {
	return atomic.LoadUint64(&s.count)
}

type testLoadClient struct {
	dist  *requestDistributor
	quit  chan struct{}
	count uint64
}

func newTestLoadClient(quit chan struct{}, clock mclock.Clock) *testLoadClient {
	return &testLoadClient{
		dist: newRequestDistributor(nil, quit, clock),
		quit: quit,
	}
}

func (c *testLoadClient) sendRequests(send bool, sw chan bool) {
	expCh := make(chan struct{}, 100)
	for {
		if send {
			select {
			case send = <-sw:
			case expCh <- struct{}{}:
				reqID := genReqID()
				rq := &distReq{
					getCost: func(dp distPeer) uint64 {
						return testRequestCost
					},
					canSend: func(dp distPeer) bool {
						return true
					},
					request: func(dp distPeer) func() {
						peer := dp.(*testLoadPeer)
						peer.fcServer.QueuedRequest(reqID, testRequestCost)
						return func() {
							peer.serveCh <- reqID
						}
					},
				}

				sentCh := c.dist.queue(rq)
				go func() {
					<-sentCh
					atomic.AddUint64(&c.count, 1)
					<-expCh
				}()
			case <-c.quit:
				return
			}
		} else {
			select {
			case send = <-sw:
			case <-c.quit:
				return
			}
		}
	}
}

func (c *testLoadClient) requestsSent() uint64 {
	return atomic.LoadUint64(&c.count)
}

const (
	testRequestCost = 3000000
	/*testClientCount  = 2
	testServerCount  = 2*/
	testMessageDelay = time.Millisecond * 200
	defaultTolerance = 5 // percent
)

type testServerPeriod struct {
	mode                  int
	measureOff, measureOn int // duration in milliseconds
	expResult, tolerance  uint64
}

type testConnection struct {
	minCapacity int // request per second guaranteed by MRR
	free        bool
}

type testServerParams struct {
	capacity int // processing request per second
	periods  []testServerPeriod
}

type testClientPeriod struct {
	sendOff, measureOff, measureOn int // duration in milliseconds
	expResult, tolerance           uint64
}

type testClientParams struct {
	periods []testClientPeriod
	servers []testConnection
}

func testLoad(t *testing.T, serverParams []testServerParams, clientParams []testClientParams) {
	quit := make(chan struct{})
	defer close(quit)

	//clock := &mclock.MonotonicClock{}
	clock := mclock.NewSimulatedClock()
	defer clock.Stop()

	var wg sync.WaitGroup

	servers := make([]*testLoadServer, len(serverParams))
	for i, params := range serverParams {
		i, params := i, params
		servers[i] = newTestLoadServer(params.capacity, quit, clock)
		wg.Add(1)
		go func() {
			for k, p := range params.periods {
				servers[i].fcManager.SetMode(p.mode)
				clock.Sleep(time.Millisecond * time.Duration(p.measureOff))
				start := servers[i].requestsProcessed()
				clock.Sleep(time.Millisecond * time.Duration(p.measureOn))
				result := servers[i].requestsProcessed() - start
				relTol := p.tolerance
				if relTol == 0 {
					relTol = defaultTolerance
				}
				tolerance := p.expResult * relTol / 100
				expMin := p.expResult - tolerance
				expMax := p.expResult + tolerance
				if result < expMin || result > expMax {
					t.Errorf("servers[%d].periods[%d] processed count mismatch (processed %d, expected between %d and %d)", i, k, result, expMin, expMax)
				}
			}
			wg.Done()
		}()
	}

	clients := make([]*testLoadClient, len(clientParams))
	for i, params := range clientParams {
		i, params := i, params
		clients[i] = newTestLoadClient(quit, clock)
		for j, conn := range params.servers {
			if conn.minCapacity > 0 {
				params := &flowcontrol.ServerParams{
					BufLimit:    18000000 * uint64(conn.minCapacity),
					MinRecharge: 3000 * uint64(conn.minCapacity),
				}
				newTestLoadPeer(t, clients[i], servers[j], params, conn.free, quit, clock)
			}
		}
		sw := make(chan bool)
		go clients[i].sendRequests(false, sw)
		wg.Add(1)
		go func() {
			for k, p := range params.periods {
				clock.Sleep(time.Millisecond * time.Duration(p.sendOff))
				sw <- true
				clock.Sleep(time.Millisecond * time.Duration(p.measureOff))
				start := clients[i].requestsSent()
				clock.Sleep(time.Millisecond * time.Duration(p.measureOn))
				result := clients[i].requestsSent() - start
				relTol := p.tolerance
				if relTol == 0 {
					relTol = defaultTolerance
				}
				tolerance := p.expResult * relTol / 100
				expMin := p.expResult - tolerance
				expMax := p.expResult + tolerance
				if result < expMin || result > expMax {
					t.Errorf("clients[%d].periods[%d] sent count mismatch (sent %d, expected between %d and %d)", i, k, result, expMin, expMax)
				}
				sw <- false
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestLoadBalance(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 1000, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 5000}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2000}}, []testConnection{{minCapacity: 60}}},
		})
}

func TestLoadBalanceMultiServer1(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 200, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1000}}},
			{capacity: 200, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1000}}},
			{capacity: 200, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1000}}},
			{capacity: 400, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 2000}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}, {minCapacity: 30}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2000}}, []testConnection{{minCapacity: 60}, {minCapacity: 60}, {minCapacity: 60}, {minCapacity: 60}}},
		})
}

func TestLoadBalanceMultiServer2(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 250, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1250}}},
			{capacity: 250, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1250}}},
			{capacity: 250, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1250}}},
			{capacity: 250, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 1250}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 60}, {minCapacity: 60}, {}, {}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{}, {}, {minCapacity: 60}, {minCapacity: 60}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1000}}, []testConnection{{minCapacity: 10}, {minCapacity: 10}, {minCapacity: 10}, {minCapacity: 90}}},
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2000}}, []testConnection{{minCapacity: 80}, {minCapacity: 80}, {minCapacity: 80}, {}}},
		})
}

func TestLoadSingle1(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 2000, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 4500, tolerance: 20}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 4500, tolerance: 20}}, []testConnection{{minCapacity: 30}}},
		})
}

func TestLoadSingle2(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 2000, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 5000}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 5000}}, []testConnection{{minCapacity: 100}}},
		})
}

func TestLoadSingle3(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 500, periods: []testServerPeriod{{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 2500}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2500}}, []testConnection{{minCapacity: 30}}},
		})
}

func TestLoadPriority(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 500, periods: []testServerPeriod{
				{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 2500},
				{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 2500},
				{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 3750},
				{mode: 1, measureOff: 3000, measureOn: 5000, expResult: 2500},
			}},
		},
		[]testClientParams{
			// paying client
			{[]testClientPeriod{
				{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2500},
				{sendOff: 8000, measureOff: 3000, measureOn: 5000, expResult: 2500},
			}, []testConnection{{minCapacity: 30}}},
			// free client
			{[]testClientPeriod{
				{sendOff: 8000, measureOff: 3000, measureOn: 5000, expResult: 2500},
				{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 1250},
				{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 2500},
			}, []testConnection{{minCapacity: 30, free: true}}},
		})
}

func TestLoadMinRecharge(t *testing.T) {
	testLoad(t,
		[]testServerParams{
			{capacity: 500, periods: []testServerPeriod{{mode: 2, measureOff: 3000, measureOn: 5000, expResult: 150}}},
		},
		[]testClientParams{
			{[]testClientPeriod{{sendOff: 0, measureOff: 3000, measureOn: 5000, expResult: 150}}, []testConnection{{minCapacity: 30}}},
		})
}
