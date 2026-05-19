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

type syncingBackend struct {
	Backend
	progress ethereum.SyncProgress
	ready    bool
}

func (b *syncingBackend) SyncProgress(_ context.Context) ethereum.SyncProgress { return b.progress }
func (b *syncingBackend) ConsensusReady() bool                                 { return b.ready }

// Issue #33687: a Done downloader but no CL handshake yet must report syncing.
func TestSyncingBeforeCLContact(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{progress: ethereum.SyncProgress{}, ready: false})
	res, err := api.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing returned error: %v", err)
	}
	if v, ok := res.(bool); ok && !v {
		t.Fatal("expected truthy syncing payload before CL handshake, got false")
	}
}

func TestSyncingAfterCLContact(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{progress: ethereum.SyncProgress{}, ready: true})
	res, err := api.Syncing(context.Background())
	if err != nil {
		t.Fatalf("Syncing returned error: %v", err)
	}
	if v, ok := res.(bool); !ok || v {
		t.Fatalf("expected false after CL handshake when sync is done, got %v", res)
	}
}

// Active sync stays truthy regardless of the CL gate.
func TestSyncingActiveSyncIgnoresCLGate(t *testing.T) {
	api := NewEthereumAPI(&syncingBackend{
		progress: ethereum.SyncProgress{
			StartingBlock: 100,
			CurrentBlock:  150,
			HighestBlock:  200,
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
