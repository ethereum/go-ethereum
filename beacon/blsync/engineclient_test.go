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
// With known set, it answers as an execution client with those blocks: it imports a block
// whose parent it has (VALID), and lacks the parent otherwise (SYNCING); a head it has is
// VALID. For the blocks in answers it gives those answers instead ("": an RPC error).
type fakeEngine struct {
	mu            sync.Mutex
	clock         mclock.Clock
	fcu           []fcuCall
	payloads      int
	order         []common.Hash // the blocks sent with newPayload
	fcuStatus     string
	payloadStatus string
	payloadErr    error
	known         map[common.Hash]bool
	answers       map[common.Hash]string
}

type fcuCall struct {
	head, finalized common.Hash
	at              time.Duration // since the test started
}

func (f *fakeEngine) ForkchoiceUpdatedV3(update engine.ForkchoiceStateV1, attr *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fcu = append(f.fcu, fcuCall{update.HeadBlockHash, update.FinalizedBlockHash, time.Duration(f.clock.Now())})
	status := f.fcuStatus
	if f.known != nil {
		status = engine.SYNCING
		if f.known[update.HeadBlockHash] {
			status = engine.VALID
		}
	}
	return engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{Status: status}}, nil
}

func (f *fakeEngine) NewPayloadV4(data engine.ExecutableData, hashes []common.Hash, root *common.Hash, requests []string) (engine.PayloadStatusV1, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads++
	f.order = append(f.order, data.BlockHash)
	if f.known != nil {
		if answer, ok := f.answers[data.BlockHash]; ok {
			if answer == "" {
				return engine.PayloadStatusV1{}, errors.New("connection lost")
			}
			return engine.PayloadStatusV1{Status: answer}, nil
		}
		if f.known[data.BlockHash] || f.known[data.ParentHash] {
			f.known[data.BlockHash] = true
			return engine.PayloadStatusV1{Status: engine.VALID}, nil
		}
		return engine.PayloadStatusV1{Status: engine.SYNCING}, nil
	}
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
	block   *ctypes.Block                        // sent back as a head's fetched block, if set
	chain   map[common.Hash]types.ChainHeadEvent // blocks to send back, by beacon root
	hold    bool                                 // don't send them back: the test does
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
	var fetchBlock blockFetcher
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

func (et *engineTest) fetchBlock(head types.ChainHeadEvent, root, execHash common.Hash) {
	et.mu.Lock()
	et.fetches = append(et.fetches, execHash)
	event, ok := et.chain[root]
	send := ok && event.ExecHash == execHash && !et.hold
	if et.block != nil && root == head.BeaconHead.Hash() {
		event, send = types.ChainHeadEvent{BeaconHead: head.BeaconHead, ExecHash: head.ExecHash, Block: et.block}, true
	}
	et.mu.Unlock()
	if send {
		et.headCh <- event
	}
}

// fetched returns the hashes of the blocks asked for so far.
func (et *engineTest) fetched() []common.Hash {
	et.mu.Lock()
	defer et.mu.Unlock()
	return slices.Clone(et.fetches)
}

// sent returns the hashes of the blocks sent with newPayload so far, in order.
func (et *engineTest) sent() []common.Hash {
	et.fake.mu.Lock()
	defer et.fake.mu.Unlock()
	return slices.Clone(et.fake.order)
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

// expectNoRepeat checks that no repeat is pending, once the engine client has stopped it.
func (et *engineTest) expectNoRepeat() {
	et.t.Helper()
	for deadline := time.Now().Add(time.Second); et.clock.ActiveTimers() > 0; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			et.t.Fatal("a repeat is still pending")
		}
	}
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
	et.fake.mu.Lock()
	if et.fake.known != nil {
		et.fake.known[sentinel.ExecHash] = true
	}
	et.fake.mu.Unlock()
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

// testChain makes n linked blocks, each the parent of the next (execution blocks and their
// beacon headers, one per slot but for an empty slot before the blocks numbered in empty), as
// head events with their blocks; and the same by beacon root. Chains of different forks have
// no block in common.
func testChain(n int, fork byte, empty ...int) ([]types.ChainHeadEvent, map[common.Hash]types.ChainHeadEvent) {
	var (
		events     []types.ChainHeadEvent
		byRoot     = make(map[common.Hash]types.ChainHeadEvent)
		parentHash common.Hash
		parentRoot common.Hash
		slot       = uint64(364032 * 32)
	)
	for i := 0; i < n; i++ {
		if slices.Contains(empty, i) {
			slot++
		}
		block := ctypes.NewBlockWithHeader(&ctypes.Header{ParentHash: parentHash, Number: big.NewInt(int64(i + 1)), BaseFee: big.NewInt(1), Extra: []byte{fork}}).
			WithBody(ctypes.Body{Withdrawals: []*ctypes.Withdrawal{}})
		header := types.Header{Slot: slot, ParentRoot: parentRoot, StateRoot: common.Hash{fork}}
		event := types.ChainHeadEvent{BeaconHead: header, ExecHash: block.Hash(), Block: block}
		events = append(events, event)
		byRoot[header.Hash()] = event
		parentHash, parentRoot, slot = block.Hash(), header.Hash(), slot+1
	}
	return events, byRoot
}

// headOf sends a chain's block as a head without its block, as P2PBlocks does.
func (et *engineTest) headOf(event types.ChainHeadEvent) common.Hash {
	et.headCh <- types.ChainHeadEvent{
		BeaconHead: event.BeaconHead,
		ExecHash:   event.ExecHash,
		Finalized:  common.Hash{0xf, event.ExecHash[0]},
	}
	return event.ExecHash
}

// fallback lets the repeats of a head run up to the one that fetches its block (4 s).
func (et *engineTest) fallback() {
	et.step(time.Second)
	et.step(time.Second)
	et.step(2 * time.Second)
}

// TestEngineClientFallbackAncestors tests that when the fetched head block's parent isn't the
// latest VALID head but a block after it (blsync skipped heads), the blocks back to that head
// are fetched too (an empty slot among them) and sent oldest first, and the head becomes
// VALID; and that the next head's block is fetched again (nothing paused), without its
// parent, the latest VALID head (an empty slot between them).
func TestEngineClientFallbackAncestors(t *testing.T) {
	et := newEngineTest(t, true)
	chain, byRoot := testChain(6, 0, 3, 5)
	et.chain = byRoot
	et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true, chain[1].ExecHash: true}

	et.headOf(chain[1]) // VALID
	et.wait(1, 0, 0)
	head := et.headOf(chain[4]) // 2 and 3 were skipped
	et.wait(2, 0, 0)
	et.fallback()
	et.wait(6, 3, 3)
	et.expectNoRepeat()
	et.expectFcu(head, 0, 1, 2, 4, 4)
	if have, want := et.fetched(), []common.Hash{chain[4].ExecHash, chain[3].ExecHash, chain[2].ExecHash}; !slices.Equal(have, want) {
		t.Fatalf("fetched %x, want %x", have, want)
	}
	if have, want := et.sent(), []common.Hash{chain[2].ExecHash, chain[3].ExecHash, chain[4].ExecHash}; !slices.Equal(have, want) {
		t.Fatalf("newPayload order %x, want %x", have, want)
	}

	head = et.headOf(chain[5])
	et.wait(7, 3, 3)
	et.fallback()
	et.wait(11, 4, 4)
	et.expectFcu(head, 4, 5, 6, 8, 8)
	et.settle(11, 4, 4)
	et.expectTimers(time.Second, time.Second, 2*time.Second, 4*time.Second, time.Second, time.Second, 2*time.Second, 4*time.Second)
}

// TestEngineClientFallbackDepth tests that blocks are fetched back to the latest VALID head
// only if at most p2pFallbackDepth slots lie between it and the head. Otherwise only the
// head's block is, which the execution client can't import (it is syncing): fetching is
// paused.
func TestEngineClientFallbackDepth(t *testing.T) {
	for _, gap := range []int{p2pFallbackDepth, p2pFallbackDepth + 1} {
		t.Run(fmt.Sprint(gap), func(t *testing.T) {
			et := newEngineTest(t, true)
			chain, byRoot := testChain(gap+2, 0)
			et.chain = byRoot
			et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true}

			et.headOf(chain[0]) // VALID
			et.wait(1, 0, 0)
			et.headOf(chain[gap+1]) // gap slots between them
			et.wait(2, 0, 0)
			et.fallback()
			if gap <= p2pFallbackDepth {
				et.wait(6, gap+1, gap+1)
				et.settle(6, gap+1, gap+1)
				return
			}
			et.wait(6, 1, 1)

			// paused: the next head isn't fetched at its 4 s repeat
			et.head(9)
			et.wait(7, 1, 1)
			et.fallback()
			et.wait(10, 1, 1)
		})
	}
}

// TestEngineClientFallbackBatch tests that a block of the batch that the execution client
// doesn't import, after one it did, ends the batch: SYNCING or ACCEPTED (it is syncing, or
// lacks a parent's state) pause fetching, INVALID or an RPC error don't.
func TestEngineClientFallbackBatch(t *testing.T) {
	for _, answer := range []string{engine.SYNCING, engine.ACCEPTED, engine.INVALID, ""} {
		t.Run(answer, func(t *testing.T) {
			et := newEngineTest(t, true)
			chain, byRoot := testChain(5, 0)
			et.chain = byRoot
			et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true, chain[1].ExecHash: true}
			et.fake.answers = map[common.Hash]string{chain[3].ExecHash: answer}

			et.headOf(chain[1]) // VALID
			et.wait(1, 0, 0)
			et.headOf(chain[4])
			et.wait(2, 0, 0)
			et.fallback()
			et.wait(6, 2, 3) // fetched 4, 3 and 2; sent 2 and 3, then the head's forkchoice update

			// the next head is fetched at its 4 s repeat, unless paused
			fetches := 4
			if answer == engine.SYNCING || answer == engine.ACCEPTED {
				fetches = 3
			}
			et.head(9)
			et.wait(7, 2, 3)
			et.fallback()
			et.wait(10, 2, fetches)
		})
	}
}

// TestEngineClientFallbackReorg tests that a head on another branch than the latest VALID
// head has its blocks fetched back to that head's slot only, and only the head's block sent
// (its parent is missing: paused).
func TestEngineClientFallbackReorg(t *testing.T) {
	et := newEngineTest(t, true)
	chain, _ := testChain(2, 0)
	branch, byRoot := testChain(4, 1)
	et.chain = byRoot
	et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true, chain[1].ExecHash: true}

	et.headOf(chain[1]) // VALID
	et.wait(1, 0, 0)
	et.headOf(branch[3])
	et.wait(2, 0, 0)
	et.fallback()
	et.wait(6, 1, 2) // fetched 3 and 2 of the branch
	if have, want := et.sent(), []common.Hash{branch[3].ExecHash}; !slices.Equal(have, want) {
		t.Fatalf("newPayload sent %x, want %x", have, want)
	}

	// paused: the next head isn't fetched at its 4 s repeat
	et.head(9)
	et.wait(7, 1, 2)
	et.fallback()
	et.wait(10, 1, 2)
}

// TestEngineClientFallbackOutgrown tests that a walk not finished before the next head starts
// over from that head, and that once the gap outgrows the bound only the head's block is
// sent, which the execution client can't import: paused (slow fetches leave it to peers).
func TestEngineClientFallbackOutgrown(t *testing.T) {
	et := newEngineTest(t, true)
	chain, byRoot := testChain(p2pFallbackDepth+3, 0)
	et.chain, et.hold = byRoot, true
	et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true}

	et.headOf(chain[0]) // VALID
	et.wait(1, 0, 0)

	// p2pFallbackDepth-1, then p2pFallbackDepth slots between: each head's block arrives,
	// and its parent is asked for, which doesn't arrive before the next head
	for i, first := range []int{p2pFallbackDepth, p2pFallbackDepth + 1} {
		et.headOf(chain[first])
		et.wait(2+4*i, 0, 2*i)
		et.fallback()
		et.wait(5+4*i, 0, 1+2*i)
		et.headCh <- chain[first]
		et.wait(5+4*i, 0, 2+2*i)
	}
	if have, want := et.fetched()[3], chain[p2pFallbackDepth].ExecHash; have != want {
		t.Fatalf("fetched %x after the new head's block, want %x", have, want)
	}

	// one more slot: only the head's block is sent
	et.headOf(chain[p2pFallbackDepth+2])
	et.wait(10, 0, 4)
	et.fallback()
	et.wait(13, 0, 5)
	et.headCh <- chain[p2pFallbackDepth+2]
	et.wait(14, 1, 5)
	if have, want := et.sent(), []common.Hash{chain[p2pFallbackDepth+2].ExecHash}; !slices.Equal(have, want) {
		t.Fatalf("newPayload sent %x, want %x", have, want)
	}

	// paused: the next head isn't fetched at its 4 s repeat
	et.head(9)
	et.wait(15, 1, 5)
	et.fallback()
	et.wait(18, 1, 5)
}

// TestEngineClientFallbackLate tests that blocks fetched for a head aren't sent if they
// arrive after a newer head, or after the head became VALID.
func TestEngineClientFallbackLate(t *testing.T) {
	et := newEngineTest(t, true)
	chain, byRoot := testChain(8, 0)
	et.chain, et.hold = byRoot, true
	et.fake.known = map[common.Hash]bool{chain[0].ExecHash: true}
	know := func(blocks ...types.ChainHeadEvent) {
		et.fake.mu.Lock()
		defer et.fake.mu.Unlock()
		for _, b := range blocks {
			et.fake.known[b.ExecHash] = true
		}
	}

	et.headOf(chain[0]) // VALID
	et.wait(1, 0, 0)

	// a newer head arrives while the head's parent is fetched
	et.headOf(chain[3])
	et.wait(2, 0, 0)
	et.fallback()
	et.wait(5, 0, 1)
	et.headCh <- chain[3]
	et.wait(5, 0, 2)
	et.headOf(chain[4])
	et.wait(6, 0, 2)
	et.headCh <- chain[2]

	// the head becomes VALID (its peers delivered) while its block is fetched
	et.fallback()
	et.wait(9, 0, 3)
	know(chain[1], chain[2], chain[3], chain[4])
	et.step(4 * time.Second)
	et.wait(10, 0, 3)
	et.headCh <- chain[4]

	// the head becomes VALID while its parent is fetched
	et.headOf(chain[7]) // 5 and 6 were skipped
	et.wait(11, 0, 3)
	et.fallback()
	et.wait(14, 0, 4)
	et.headCh <- chain[7]
	et.wait(14, 0, 5)
	know(chain[5], chain[6], chain[7])
	et.step(4 * time.Second)
	et.wait(15, 0, 5)
	et.headCh <- chain[6]
	et.settle(15, 0, 5)
}
