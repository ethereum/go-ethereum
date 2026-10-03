// Copyright 2026 The go-ethereum Authors
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

package blsync

import (
	"errors"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// fakeEngine answers forkchoiceUpdated and newPayload with set results and records the calls.
type fakeEngine struct {
	mu            sync.Mutex
	clock         mclock.Clock
	fcu           []fcuCall
	payloads      int
	fcuStatus     string
	payloadStatus string
	payloadErr    error
}

type fcuCall struct {
	head, finalized common.Hash
	at              time.Duration // since the test started
}

func (f *fakeEngine) ForkchoiceUpdatedV3(update engine.ForkchoiceStateV1, attr *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fcu = append(f.fcu, fcuCall{update.HeadBlockHash, update.FinalizedBlockHash, time.Duration(f.clock.Now())})
	return engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{Status: f.fcuStatus}}, nil
}

func (f *fakeEngine) NewPayloadV4(data engine.ExecutableData, hashes []common.Hash, root *common.Hash, requests []string) (engine.PayloadStatusV1, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads++
	return engine.PayloadStatusV1{Status: f.payloadStatus}, f.payloadErr
}

func (f *fakeEngine) set(fcuStatus, payloadStatus string, payloadErr error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fcuStatus, f.payloadStatus, f.payloadErr = fcuStatus, payloadStatus, payloadErr
}

// timerClock is a simulated clock that records the durations of the timers it creates.
type timerClock struct {
	*mclock.Simulated
	mu     sync.Mutex
	timers []time.Duration
}

func (c *timerClock) NewTimer(d time.Duration) mclock.ChanTimer {
	c.mu.Lock()
	c.timers = append(c.timers, d)
	c.mu.Unlock()
	return c.Simulated.NewTimer(d)
}

type engineTest struct {
	t         *testing.T
	clock     *timerClock
	fake      *fakeEngine
	headCh    chan types.ChainHeadEvent
	p2pBlocks bool

	mu      sync.Mutex
	fetches []common.Hash
	block   *ctypes.Block // sent back as the fetched block, if set
}

func newEngineTest(t *testing.T, p2pBlocks bool) *engineTest {
	clock := &timerClock{Simulated: new(mclock.Simulated)}
	et := &engineTest{
		t:         t,
		clock:     clock,
		fake:      &fakeEngine{clock: clock, fcuStatus: engine.SYNCING},
		headCh:    make(chan types.ChainHeadEvent, 4),
		p2pBlocks: p2pBlocks,
	}
	srv := rpc.NewServer()
	if err := srv.RegisterName("engine", et.fake); err != nil {
		t.Fatal(err)
	}
	client := rpc.DialInProc(srv)
	var fetchBlock func(types.ChainHeadEvent)
	if p2pBlocks {
		fetchBlock = et.fetchBlock
	}
	config := &params.ClientConfig{ChainConfig: *params.MainnetLightConfig}
	ec := startEngineClientWithClock(config, client, et.headCh, fetchBlock, clock)
	t.Cleanup(func() {
		ec.stop()
		client.Close()
		srv.Stop()
	})
	return et
}

func (et *engineTest) fetchBlock(head types.ChainHeadEvent) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.fetches = append(et.fetches, head.ExecHash)
	if et.block != nil {
		head.Block = et.block
		et.headCh <- head
	}
}

// head sends a head without its block (an Electra slot), as P2PBlocks does.
func (et *engineTest) head(n byte) common.Hash {
	hash := common.Hash{n}
	et.headCh <- types.ChainHeadEvent{
		BeaconHead: types.Header{Slot: 364032*32 + uint64(n)},
		ExecHash:   hash,
		Finalized:  common.Hash{0xf, n},
	}
	return hash
}

// wait waits until the engine client has made the given numbers of forkchoice updates,
// newPayload calls and block fetches; it fails on more or on a timeout.
func (et *engineTest) wait(fcu, payloads, fetches int) {
	et.t.Helper()
	var have string
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		et.fake.mu.Lock()
		gotFcu, gotPayloads := len(et.fake.fcu), et.fake.payloads
		et.fake.mu.Unlock()
		et.mu.Lock()
		gotFetches := len(et.fetches)
		et.mu.Unlock()
		if gotFcu == fcu && gotPayloads == payloads && gotFetches == fetches {
			return
		}
		have = fmt.Sprintf("%d/%d/%d", gotFcu, gotPayloads, gotFetches)
		if gotFcu > fcu || gotPayloads > payloads || gotFetches > fetches {
			break
		}
	}
	et.t.Fatalf("forkchoice updates/payloads/fetches: have %s, want %d/%d/%d", have, fcu, payloads, fetches)
}

// step lets the next repeat's timer expire; it fails if no repeat is pending.
func (et *engineTest) step(d time.Duration) {
	et.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); et.clock.ActiveTimers() == 0; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			et.t.Fatal("no repeat pending")
		}
	}
	et.clock.Run(d)
}

// expectTimers checks the durations of the repeat timers created so far.
func (et *engineTest) expectTimers(want ...time.Duration) {
	et.t.Helper()
	et.clock.mu.Lock()
	defer et.clock.mu.Unlock()
	if !slices.Equal(et.clock.timers, want) {
		et.t.Fatalf("repeat timers %v, want %v", et.clock.timers, want)
	}
}

// settle waits until the engine client has handled the events sent so far: it handles them
// in order, so it sends one more head, VALID at once, and waits for its forkchoice update.
// fcu, payloads and fetches are the calls made before it. Check the timers after this.
func (et *engineTest) settle(fcu, payloads, fetches int) {
	et.t.Helper()
	et.fake.set(engine.VALID, engine.VALID, nil)
	sentinel := types.ChainHeadEvent{BeaconHead: types.Header{Slot: 364032 * 32}, ExecHash: common.Hash{0xff}}
	if !et.p2pBlocks {
		sentinel.Block = testBlockForFetch()
		payloads++
	}
	et.headCh <- sentinel
	et.wait(fcu+1, payloads, fetches)
}

// expectFcu checks the times (seconds since the test started) of the forkchoice updates
// naming a head, and that they carried its finalized hash.
func (et *engineTest) expectFcu(head common.Hash, at ...int) {
	et.t.Helper()
	et.fake.mu.Lock()
	defer et.fake.mu.Unlock()
	var have []int
	for _, call := range et.fake.fcu {
		if call.head != head {
			continue
		}
		if call.finalized != (common.Hash{0xf, head[0]}) {
			et.t.Errorf("forkchoice update for %x carried finalized %x", head, call.finalized)
		}
		have = append(have, int(call.at/time.Second))
	}
	if !slices.Equal(have, at) {
		et.t.Fatalf("forkchoice updates for %x at %v s, want %v s", head, have, at)
	}
}

func testBlockForFetch() *ctypes.Block {
	return ctypes.NewBlockWithHeader(&ctypes.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1)}).
		WithBody(ctypes.Body{Withdrawals: []*ctypes.Withdrawal{}})
}

// TestEngineClientRepeats tests that a head sent without its block is repeated 1, 2, 4 and
// 8 s after it and then every 8 s until it is VALID, and its block is asked for once, at
// the 4 s repeat.
func TestEngineClientRepeats(t *testing.T) {
	et := newEngineTest(t, true)
	head := et.head(1)
	et.wait(1, 0, 0)
	for i, d := range []time.Duration{time.Second, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
		et.step(d)
		fetches := 0
		if i >= 2 {
			fetches = 1
		}
		et.wait(2+i, 0, fetches)
	}
	et.expectFcu(head, 0, 1, 2, 4, 8, 16, 24)

	// VALID at the next repeat: no more repeats
	et.fake.set(engine.VALID, "", nil)
	et.step(8 * time.Second)
	et.wait(8, 0, 1)
	et.settle(8, 0, 1)
	et.expectTimers(time.Second, time.Second, 2*time.Second, 4*time.Second, 8*time.Second, 8*time.Second, 8*time.Second)
}

// TestEngineClientNewHead tests that a new head ends the repeats of the previous one and
// starts its own.
func TestEngineClientNewHead(t *testing.T) {
	et := newEngineTest(t, true)
	head1 := et.head(1)
	et.wait(1, 0, 0)
	et.step(time.Second)
	et.wait(2, 0, 0)

	head2 := et.head(2)
	et.wait(3, 0, 0)
	et.step(time.Second)
	et.wait(4, 0, 0)
	et.step(time.Second)
	et.wait(5, 0, 0)
	et.step(2 * time.Second)
	et.wait(6, 0, 1)

	et.expectFcu(head1, 0, 1)
	et.expectFcu(head2, 1, 2, 3, 5)
	et.settle(6, 0, 1)
	et.expectTimers(time.Second, time.Second, time.Second, time.Second, 2*time.Second, 4*time.Second)
	if et.fetches[0] != head2 {
		t.Fatalf("fetched %x, want %x", et.fetches[0], head2)
	}
}

// TestEngineClientFallbackPause tests that a fetched block the execution client can't import
// (SYNCING: it is still catching up) pauses fetching blocks while the head's repeats go on,
// and that a VALID head resumes it.
func TestEngineClientFallbackPause(t *testing.T) {
	et := newEngineTest(t, true)
	et.block = testBlockForFetch()
	et.fake.set(engine.SYNCING, engine.SYNCING, nil)

	// At the 4 s repeat the block is fetched and sent with newPayload, then its forkchoice
	// update: paused, and the head's repeats go on (8 s).
	head1 := et.head(1)
	et.wait(1, 0, 0)
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
	et.wait(5, 1, 1)
	et.step(4 * time.Second)
	et.wait(6, 1, 1)
	et.expectFcu(head1, 0, 1, 2, 4, 4, 8)

	// Paused: the next head isn't fetched at its 4 s repeat.
	et.head(2)
	et.wait(7, 1, 1)
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
	et.wait(10, 1, 1)

	// It becomes VALID, which resumes fetching: the next head that stays SYNCING is
	// fetched, and this time the block is imported.
	et.fake.set(engine.VALID, engine.VALID, nil)
	et.step(4 * time.Second)
	et.wait(11, 1, 1)
	et.fake.set(engine.SYNCING, engine.VALID, nil)
	head3 := et.head(3)
	et.wait(12, 1, 1)
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
	et.wait(16, 2, 2)
	et.expectFcu(head3, 16, 17, 18, 20, 20)
}

// TestEngineClientFallbackError tests that a fetched block whose newPayload fails (an RPC
// error, not an answer that the execution client is catching up) doesn't pause fetching.
func TestEngineClientFallbackError(t *testing.T) {
	et := newEngineTest(t, true)
	et.block = testBlockForFetch()
	et.fake.set(engine.SYNCING, "", errors.New("connection lost"))

	et.head(1)
	et.wait(1, 0, 0)
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
	et.wait(5, 1, 1)

	et.head(2)
	et.wait(6, 1, 1)
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
	et.wait(10, 2, 2)
}

// TestEngineClientDefaultMode tests that without P2PBlocks a head is sent with newPayload
// and a single forkchoice update, without repeats or fetches.
func TestEngineClientDefaultMode(t *testing.T) {
	et := newEngineTest(t, false)
	et.headCh <- types.ChainHeadEvent{
		BeaconHead: types.Header{Slot: 364032 * 32},
		ExecHash:   testBlockForFetch().Hash(),
		Block:      testBlockForFetch(),
	}
	et.wait(1, 1, 0)
	et.settle(1, 1, 0)
	et.expectTimers()
}
