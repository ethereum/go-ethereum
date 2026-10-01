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
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// fakeEngine answers forkchoiceUpdated and newPayload with set statuses and counts the calls.
type fakeEngine struct {
	mu            sync.Mutex
	fcu, payloads int
	fcuStatus     string
	payloadStatus string
}

func (f *fakeEngine) ForkchoiceUpdatedV3(update engine.ForkchoiceStateV1, attr *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fcu++
	return engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{Status: f.fcuStatus}}, nil
}

func (f *fakeEngine) NewPayloadV4(data engine.ExecutableData, hashes []common.Hash, root *common.Hash, requests []string) (engine.PayloadStatusV1, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.payloads++
	return engine.PayloadStatusV1{Status: f.payloadStatus}, nil
}

func (f *fakeEngine) set(fcuStatus, payloadStatus string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fcuStatus, f.payloadStatus = fcuStatus, payloadStatus
}

func (f *fakeEngine) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fcu, f.payloads
}

type engineTest struct {
	t       *testing.T
	fake    *fakeEngine
	headCh  chan types.ChainHeadEvent
	mu      sync.Mutex
	fetches int
	block   *ctypes.Block // sent back as the fetched block, if set
}

func newEngineTest(t *testing.T) *engineTest {
	// Repeats in milliseconds instead of seconds.
	saved := p2pRepeats
	p2pRepeats = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 40 * time.Millisecond, 80 * time.Millisecond}
	t.Cleanup(func() { p2pRepeats = saved })

	et := &engineTest{t: t, fake: &fakeEngine{fcuStatus: engine.SYNCING}, headCh: make(chan types.ChainHeadEvent, 4)}
	srv := rpc.NewServer()
	if err := srv.RegisterName("engine", et.fake); err != nil {
		t.Fatal(err)
	}
	config := &params.ClientConfig{ChainConfig: *params.MainnetLightConfig}
	ec := startEngineClient(config, rpc.DialInProc(srv), et.headCh, et.fetchBlock)
	t.Cleanup(ec.stop)
	return et
}

func (et *engineTest) fetchBlock(head types.ChainHeadEvent) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.fetches++
	if et.block != nil {
		head.Block = et.block
		et.headCh <- head
	}
}

// head sends a head without its block (an Electra slot) and waits for its repeats to end.
func (et *engineTest) head(n uint64) {
	et.headCh <- types.ChainHeadEvent{
		BeaconHead: types.Header{Slot: 364032*32 + n},
		ExecHash:   common.Hash{byte(n)},
	}
	time.Sleep(150 * time.Millisecond)
}

func (et *engineTest) expect(fcu, payloads, fetches int) {
	et.t.Helper()
	gotFcu, gotPayloads := et.fake.counts()
	et.mu.Lock()
	gotFetches := et.fetches
	et.mu.Unlock()
	if gotFcu != fcu || gotPayloads != payloads || gotFetches != fetches {
		et.t.Fatalf("forkchoice updates/payloads/fetches: got %d/%d/%d, want %d/%d/%d", gotFcu, gotPayloads, gotFetches, fcu, payloads, fetches)
	}
}

// TestEngineClientRepeats tests that a head sent without its block is repeated at the backed-off
// times until it is VALID, and its block is asked for once, at the p2pFallbackAt-th repeat.
func TestEngineClientRepeats(t *testing.T) {
	et := newEngineTest(t)
	et.head(1) // SYNCING throughout: the update and 4 repeats, one fetch
	et.expect(5, 0, 1)

	et.fake.set(engine.VALID, "")
	et.head(2) // VALID at once: no repeats
	et.expect(6, 0, 1)
}

// TestEngineClientFallbackPause tests that fetching blocks pauses when a fetched block doesn't
// make the head VALID (the execution client is catching up), until a head is VALID again.
func TestEngineClientFallbackPause(t *testing.T) {
	et := newEngineTest(t)
	et.block = ctypes.NewBlockWithHeader(&ctypes.Header{Number: big.NewInt(1), BaseFee: big.NewInt(1)}).
		WithBody(ctypes.Body{Withdrawals: []*ctypes.Withdrawal{}})
	et.fake.set(engine.SYNCING, engine.SYNCING)

	// The fetched block is sent with newPayload, which can't import it: paused. The fetched
	// block's own forkchoice update ends that head's repeats (it carries the block).
	et.head(1)
	et.expect(1+3+1, 1, 1)

	// Paused: repeats, no fetch.
	et.head(2)
	et.expect(5+5, 1, 1)

	// A VALID head resumes it; the next head that stays SYNCING fetches again, and this time
	// the block is imported.
	et.fake.set(engine.VALID, engine.VALID)
	et.head(3)
	et.expect(11, 1, 1)
	et.fake.set(engine.SYNCING, engine.VALID)
	et.head(4)
	et.expect(11+1+3+1, 2, 2)
}
