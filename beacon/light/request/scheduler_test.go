// Copyright 2023 The go-ethereum Authors
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

package request

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
)

type testRequestServer struct {
	newHead       func(uint64, common.Hash)
	newSignedHead func(types.SignedHeader)
	clock         *mclock.Simulated
	delayUntil    mclock.AbsTime
	failed        bool
}

func (s *testRequestServer) SubscribeHeads(newHead func(uint64, common.Hash), newSignedHead func(types.SignedHeader)) {
	s.newHead, s.newSignedHead = newHead, newSignedHead
}

func (s *testRequestServer) UnsubscribeHeads() {}

func (s *testRequestServer) DelayUntil() mclock.AbsTime {
	return s.delayUntil
}

func (s *testRequestServer) Fail(string) {
	s.failed = true
}

type testModule struct {
	processCh chan testProcess
	triggers  []int
	index     int
}

func (m *testModule) SetupModuleTriggers(trigger func(id string, subscribe bool) *ModuleTrigger) {
	for _, tr := range m.triggers {
		trigger(fmt.Sprintf("trigger %d", tr), true)
	}
}

type testProcess struct {
	m    *testModule
	env  *Environment
	done chan struct{}
}

func (m *testModule) Process(env *Environment) {
	done := make(chan struct{})
	m.processCh <- testProcess{m, env, done}
	<-done
}

type schedulerTest struct {
	t         *testing.T
	clock     *mclock.Simulated
	scheduler *Scheduler
	modules   []*testModule
	servers   []*testRequestServer
	triggers  []*ModuleTrigger
	processCh chan testProcess

	env  *Environment
	done chan struct{}
}

func newSchedulerTest(t *testing.T, clock *mclock.Simulated, serverCount, triggerCount int, moduleTriggers [][]int) *schedulerTest {
	st := &schedulerTest{
		t:         t,
		clock:     clock,
		scheduler: NewScheduler(NewHeadTracker(func(*Server, types.SignedHeader) {}), clock),
		modules:   make([]*testModule, len(moduleTriggers)),
		triggers:  make([]*ModuleTrigger, triggerCount),
		processCh: make(chan testProcess),
	}
	for i, tr := range moduleTriggers {
		st.modules[i] = &testModule{
			processCh: st.processCh,
			triggers:  tr,
			index:     i,
		}
		st.scheduler.RegisterModule(st.modules[i])
	}
	for i := range st.triggers {
		st.triggers[i] = st.scheduler.GetModuleTrigger(fmt.Sprintf("trigger %d", i))
	}
	st.scheduler.Start()
	st.expectProcess(-1) // expect initial waiting state
	for i := 0; i < serverCount; i++ {
		st.addServer()
	}
	return st
}

func (st *schedulerTest) addServer() *testRequestServer {
	server := &testRequestServer{
		clock: st.clock,
	}
	st.scheduler.RegisterServer(server)
	st.servers = append(st.servers, server)
	st.expectServerTrigger()
	return server
}

func (st *schedulerTest) expectServerTrigger() {
	for i := range st.modules {
		st.expectProcess(i) // expect all modules to be triggered
		st.processDone()
	}
	st.expectProcess(-1) // expect waiting state
}

func (st *schedulerTest) expectProcess(expIndex int) {
	var (
		tp    testProcess
		index int
	)
	select {
	case tp = <-st.processCh:
		index = tp.m.index
	case st.scheduler.testWaitCh <- struct{}{}:
		index = -1
	}
	if index != expIndex {
		st.t.Fatalf("Incorrect processed module index (got %d, expected %d)", index, expIndex)
	}
	st.env, st.done = tp.env, tp.done
}

func (st *schedulerTest) processDone() {
	close(st.done)
}

func TestModuleTrigger(t *testing.T) {
	st := newSchedulerTest(t, &mclock.Simulated{}, 0, 6, [][]int{{0, 1, 2, 4}, {1, 3}, {0, 2, 4}, {2, 3, 4}})
	st.triggers[0].Trigger() // triggers modules 0, 2
	// round 1
	st.expectProcess(0)
	st.triggers[1].Trigger() // triggers modules 0, 1
	st.triggers[2].Trigger() // triggers modules 0, 2, 3
	st.triggers[3].Trigger() // triggers modules 1, 3
	st.triggers[4].Trigger() // triggers modules 0, 2, 3
	st.processDone()
	st.expectProcess(2)
	st.processDone()
	// round 2
	st.expectProcess(0)
	st.processDone()
	st.expectProcess(1)
	st.triggers[3].Trigger() // triggers modules 1, 3
	st.processDone()
	st.expectProcess(2)
	st.processDone()
	st.expectProcess(3)
	st.triggers[1].Trigger() // triggers modules 0, 1
	st.processDone()
	// round 3
	st.expectProcess(0)
	st.processDone()
	st.expectProcess(1)
	st.triggers[5].Trigger() // doesn't trigger anything
	st.processDone()
	st.expectProcess(3)
	st.processDone()
	st.expectProcess(-1)
	// waiting for a next trigger
	st.triggers[2].Trigger() // triggers modules 0, 2, 3
	// round 4
	st.expectProcess(0)
	st.processDone()
	st.expectProcess(2)
	st.processDone()
	st.expectProcess(3)
	st.processDone()
	st.expectProcess(-1)
}

type testRequest struct {
	reqLock   SingleLock
	returnFns []func()
}

func (r *testRequest) CanSendTo(server *Server, moduleData *interface{}) (canSend bool, priority uint64) {
	return r.reqLock.CanRequest(), 0
}

func (r *testRequest) SendTo(server *Server, moduleData *interface{}) {
	reqId := r.reqLock.Send(server)
	r.returnFns = append(r.returnFns, func() {
		r.reqLock.Returned(server, reqId)
	})
}

func (r *testRequest) returned() {
	r.returnFns[0]()
	r.returnFns = r.returnFns[1:]
}

func TestServerTrigger(t *testing.T) {
	st := newSchedulerTest(t, &mclock.Simulated{}, 3, 2, [][]int{{0}, {}})
	st.scheduler.testTimerResults = []bool{}
	req := &testRequest{}
	req.reqLock.Trigger = st.triggers[0]

	tryRequest := func(expCanRequest, expSuccess, serverTrigger bool) {
		st.expectProcess(0)
		c := st.env.CanRequestNow()
		if c != expCanRequest {
			t.Fatalf("Environment.CanRequestNow() returned wrong result (got: %v expected: %v)", c, expCanRequest)
		}
		if c {
			if s := st.env.TryRequest(req); s != expSuccess {
				t.Fatalf("Environment.TryRequest(req) returned wrong result (got: %v expected: %v)", s, expSuccess)
			}
		}
		st.processDone()
		if serverTrigger {
			for i := 1; i < len(st.modules); i++ {
				st.expectProcess(i)
				st.processDone()
			}
		}
		st.expectProcess(-1)
	}

	expectActiveTimers := func(exp int) {
		if timers := st.clock.ActiveTimers(); timers != exp {
			t.Fatalf("Invalid number of simulated clock timers (got %v, expected %v)", timers, exp)
		}
	}

	expectTimerFinished := func(exp bool) {
		if len(st.scheduler.testTimerResults) == 0 {
			t.Fatalf("No timer results found (expected %v)", exp)
		}
		if st.scheduler.testTimerResults[0] != exp {
			t.Fatalf("Invalid simulated timer result (got finished == %v, expected %v)", st.scheduler.testTimerResults[0], exp)
		}
		st.scheduler.testTimerResults = st.scheduler.testTimerResults[1:]
	}

	st.servers[2].delayUntil = mclock.AbsTime(time.Second * 5)
	st.triggers[0].Trigger()
	tryRequest(true, true, false)
	expectActiveTimers(2) // delay, timeout
	st.triggers[0].Trigger()
	// there are available servers but first request not timed out yet
	tryRequest(true, false, false)
	expectActiveTimers(2) // delay, timeout
	st.clock.Run(softRequestTimeout)
	expectActiveTimers(1)     // delay
	expectTimerFinished(true) // timeout timer processed
	// expect module trigger by Server (timeout), should be able to send again
	tryRequest(true, true, false)
	expectActiveTimers(2) // delay, timeout
	st.clock.Run(softRequestTimeout)
	expectActiveTimers(1)     // delay
	expectTimerFinished(true) // timeout timer processed
	// two servers timed out and one delayed; no one to request from
	tryRequest(false, false, false)
	st.clock.Run(time.Second*5 - softRequestTimeout*2)
	expectActiveTimers(0)
	expectTimerFinished(true) // timeout timer processed
	// expect server trigger by Server (expired delay), should be able to send again
	tryRequest(true, true, true)
	expectActiveTimers(1) // timeout
	st.clock.Run(softRequestTimeout)
	expectActiveTimers(0)
	expectTimerFinished(true) // timeout timer processed
	// expect module trigger by Server; all servers timed out now
	tryRequest(false, false, false)
	req.returned()
	st.expectServerTrigger() // server triggered because not blocked by timeout anymore
	req.returned()
	st.expectServerTrigger()
	req.returned()
	st.expectServerTrigger()
	// now send a request that does not time out
	st.triggers[0].Trigger()
	tryRequest(true, true, false)
	expectActiveTimers(1) // timeout
	st.clock.Run(softRequestTimeout / 2)
	expectActiveTimers(1) // timeout
	req.returned()
	expectTimerFinished(false) // timeout timer stopped
	expectActiveTimers(0)
	st.expectProcess(0) // module triggered by request lock
	st.processDone()
	st.expectProcess(-1) // no server trigger expected because server was not blocked by timeout
}
