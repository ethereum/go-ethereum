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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

func TestSetBalance(t *testing.T) {
	var clock = &mclock.Simulated{}
	var inputs = []struct {
		pos uint64
		neg uint64
	}{
		{1000, 0},
		{0, 1000},
		{1000, 1000},
	}

	tracker := balanceTracker{}
	tracker.init(clock, 1000)
	defer tracker.stop(clock.Now())

	for _, i := range inputs {
		tracker.setBalance(i.pos, i.neg)
		pos, neg := tracker.getBalance(clock.Now())
		if pos != i.pos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.pos, pos)
		}
		if neg != i.neg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.neg, neg)
		}
	}
}

func TestBalanceTimeCost(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000)
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	tracker.setBalance(uint64(time.Minute), 0) // 1 minute time allowance

	var inputs = []struct {
		runTime time.Duration
		expPos  uint64
		expNeg  uint64
	}{
		{time.Second, uint64(time.Second * 59), 0},
		{0, uint64(time.Second * 59), 0},
		{time.Second * 59, 0, 0},
		{time.Second, 0, uint64(time.Second)},
	}
	for _, i := range inputs {
		clock.Run(i.runTime)
		if pos, _ := tracker.getBalance(clock.Now()); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := tracker.getBalance(clock.Now()); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}

	tracker.setBalance(uint64(time.Minute), 0) // Refill 1 minute time allowance
	for _, i := range inputs {
		clock.Run(i.runTime)
		if pos, _ := tracker.getBalance(clock.Now()); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := tracker.getBalance(clock.Now()); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}
}

func TestBalanceReqCost(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000)
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	tracker.setBalance(uint64(time.Minute), 0) // 1 minute time serving time allowance
	var inputs = []struct {
		reqCost uint64
		expPos  uint64
		expNeg  uint64
	}{
		{uint64(time.Second), uint64(time.Second * 59), 0},
		{0, uint64(time.Second * 59), 0},
		{uint64(time.Second * 59), 0, 0},
		{uint64(time.Second), 0, uint64(time.Second)},
	}
	for _, i := range inputs {
		tracker.requestCost(i.reqCost)
		if pos, _ := tracker.getBalance(clock.Now()); pos != i.expPos {
			t.Fatalf("Positive balance mismatch, want %v, got %v", i.expPos, pos)
		}
		if _, neg := tracker.getBalance(clock.Now()); neg != i.expNeg {
			t.Fatalf("Negative balance mismatch, want %v, got %v", i.expNeg, neg)
		}
	}
}

func TestBalanceToPriority(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000) // cap = 1000
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	var inputs = []struct {
		pos      uint64
		neg      uint64
		priority int64
	}{
		{1000, 0, ^int64(1)},
		{2000, 0, ^int64(2)}, // Higher balance, lower priority value
		{0, 0, 0},
		{0, 1000, 1000},
	}
	for _, i := range inputs {
		tracker.setBalance(i.pos, i.neg)
		priority := tracker.getPriority(clock.Now())
		if priority != i.priority {
			t.Fatalf("Priority mismatch, want %v, got %v", i.priority, priority)
		}
	}
}

func TestEstimatedPriority(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000000000) // cap = 1000,000,000
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	tracker.setBalance(uint64(time.Minute), 0)
	var inputs = []struct {
		runTime    time.Duration // time cost
		futureTime time.Duration // diff of future time
		reqCost    uint64        // single request cost
		priority   int64         // expected estimated priority
	}{
		{time.Second, time.Second, 0, ^int64(58)},
		{0, time.Second, 0, ^int64(58)},

		// 2 seconds time cost, 1 second estimated time cost, 10^9 request cost,
		// 10^9 estimated request cost per second.
		{time.Second, time.Second, 1000000000, ^int64(55)},

		// 3 seconds time cost, 3 second estimated time cost, 10^9*2 request cost,
		// 4*10^9 estimated request cost.
		{time.Second, 3 * time.Second, 1000000000, ^int64(48)},

		// All positive balance is used up
		{time.Second * 55, 0, 0, 0},

		// 1 minute estimated time cost, 4/58 * 10^9 estimated request cost per sec.
		{0, time.Minute, 0, int64(time.Minute) + int64(time.Second)*120/29},
	}
	for _, i := range inputs {
		clock.Run(i.runTime)
		tracker.requestCost(i.reqCost)
		priority := tracker.estimatedPriority(clock.Now()+mclock.AbsTime(i.futureTime), true)
		if priority != i.priority {
			t.Fatalf("Estimated priority mismatch, want %v, got %v", i.priority, priority)
		}
	}
}

func TestCallbackChecking(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000000) // cap = 1000,000
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	var inputs = []struct {
		priority int64
		expDiff  time.Duration
	}{
		{^int64(500), time.Millisecond * 500},
		{0, time.Second},
		{int64(time.Second), 2 * time.Second},
	}
	tracker.setBalance(uint64(time.Second), 0)
	for _, i := range inputs {
		diff, _ := tracker.timeUntil(i.priority)
		if diff != i.expDiff {
			t.Fatalf("Time difference mismatch, want %v, got %v", i.expDiff, diff)
		}
	}
}

func TestCallback(t *testing.T) {
	var (
		clock   = &mclock.Simulated{}
		tracker = balanceTracker{}
	)
	tracker.init(clock, 1000) // cap = 1000
	defer tracker.stop(clock.Now())
	tracker.setFactors(false, 1, 1)
	tracker.setFactors(true, 1, 1)

	callCh := make(chan struct{}, 1)
	tracker.setBalance(uint64(time.Minute), 0)
	tracker.addCallback(balanceCallbackZero, 0, func() { callCh <- struct{}{} })

	clock.Run(time.Minute)
	select {
	case <-callCh:
	case <-time.NewTimer(time.Second).C:
		t.Fatalf("Callback hasn't been called yet")
	}

	tracker.setBalance(uint64(time.Minute), 0)
	tracker.addCallback(balanceCallbackZero, 0, func() { callCh <- struct{}{} })
	tracker.removeCallback(balanceCallbackZero)

	clock.Run(time.Minute)
	select {
	case <-callCh:
		t.Fatalf("Callback shouldn't be called")
	case <-time.NewTimer(time.Millisecond * 100).C:
	}
}
