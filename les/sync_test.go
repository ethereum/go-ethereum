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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
)

// Test light syncing which will download all headers from genesis.
func TestLightSyncingLes3(t *testing.T) { testSyncing(t, lpv3) }

func testSyncing(t *testing.T, protocol int) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= 1 && bts >= 1 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Generate 128+1 blocks (totally 1 CHT section)
	netconfig := testnetConfig{
		blocks:    int(config.ChtSize + config.ChtConfirms),
		protocol:  protocol,
		indexFn:   waitIndexers,
		nopruning: true,
	}
	server, client, tearDown := newClientServerEnv(t, netconfig)
	defer tearDown()

	expected := config.ChtSize + config.ChtConfirms

	done := make(chan error)
	client.handler.syncEnd = func(header *types.Header) {
		if header.Number.Uint64() == expected {
			done <- nil
		} else {
			done <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expected, header.Number)
		}
	}

	// Create connected peer pair.
	peer1, peer2, err := newTestPeerPair("peer", protocol, server.handler, client.handler, false)
	if err != nil {
		t.Fatalf("Failed to connect testing peers %v", err)
	}
	defer peer1.close()
	defer peer2.close()

	select {
	case err := <-done:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}
}
