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

package ethapi

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum"
)

// syncingBackend is a minimal Backend embedding that only implements the two
// methods Syncing calls. Embedding the interface avoids pulling in the full
// testBackend setup just to flip a single bool.
type syncingBackend struct {
	Backend
	progress ethereum.SyncProgress
	ready    bool
}

func (b *syncingBackend) SyncProgress(_ context.Context) ethereum.SyncProgress { return b.progress }
func (b *syncingBackend) ConsensusReady() bool                                 { return b.ready }

// TestSyncingReportsBeforeConsensusContact verifies that on a CL-paired node
// (ConsensusReady false), eth_syncing returns a truthy progress object even
// when the local downloader believes itself to be done. This is the bug fix
// for issue #33687: a freshly started node must not advertise itself as
// "synced" before the consensus client has actually driven it.
func TestSyncingReportsBeforeConsensusContact(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{
		// progress.Done() returns true on a zero-valued struct because all
		// remaining counters are zero and CurrentBlock >= HighestBlock.
		progress: ethereum.SyncProgress{},
		ready:    false,
	})
	res, err := api.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing returned error: %v", err)
	}
	if v, ok := res.(bool); ok && !v {
		t.Fatal("expected truthy syncing payload before CL handshake, got false")
	}
}

// TestSyncingReportsFalseAfterConsensusContact verifies that once the
// consensus layer has handshaken at least once (or the backend does not
// expect one) and progress.Done() is true, eth_syncing reports false.
func TestSyncingReportsFalseAfterConsensusContact(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{
		progress: ethereum.SyncProgress{},
		ready:    true,
	})
	res, err := api.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing returned error: %v", err)
	}
	v, ok := res.(bool)
	if !ok || v {
		t.Fatalf("expected false after CL handshake when sync is done, got %v", res)
	}
}

// TestSyncingReportsActiveSyncEvenWithoutConsensusContact verifies that when
// the downloader is actively syncing, eth_syncing returns the progress map
// regardless of the CL gate. This preserves the legacy semantics for the case
// the issue thread did not affect.
func TestSyncingReportsActiveSyncEvenWithoutConsensusContact(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{
		progress: ethereum.SyncProgress{
			StartingBlock: 100,
			CurrentBlock:  150,
			HighestBlock:  200, // CurrentBlock < HighestBlock => Done()=false
		},
		ready: false,
	})
	res, err := api.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing returned error: %v", err)
	}
	if _, ok := res.(map[string]interface{}); !ok {
		t.Fatalf("expected progress map during active sync, got %T", res)
	}
}
