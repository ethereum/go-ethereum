// Copyright 2018 The go-ethereum Authors
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
	"math/big"
	"testing"
	"time"

	"github.com/maticnetwork/bor/accounts/abi/bind"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/light"
	"github.com/maticnetwork/bor/params"
)

// Test light syncing which will download all headers from genesis.
func TestLightSyncingLes2(t *testing.T) { testCheckpointSyncing(t, 2, 0) }
func TestLightSyncingLes3(t *testing.T) { testCheckpointSyncing(t, 3, 0) }

// Test legacy checkpoint syncing which will download tail headers
// based on a hardcoded checkpoint.
func TestLegacyCheckpointSyncingLes2(t *testing.T) { testCheckpointSyncing(t, 2, 1) }
func TestLegacyCheckpointSyncingLes3(t *testing.T) { testCheckpointSyncing(t, 3, 1) }

// Test checkpoint syncing which will download tail headers based
// on a verified checkpoint.
func TestCheckpointSyncingLes2(t *testing.T) { testCheckpointSyncing(t, 2, 2) }
func TestCheckpointSyncingLes3(t *testing.T) { testCheckpointSyncing(t, 3, 2) }

func testCheckpointSyncing(t *testing.T, protocol int, syncMode int) {
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
	// Generate 512+4 blocks (totally 1 CHT sections)
	server, client, tearDown := newClientServerEnv(t, int(config.ChtSize+config.ChtConfirms), protocol, waitIndexers, false)
	defer tearDown()

	expected := config.ChtSize + config.ChtConfirms

	// Checkpoint syncing or legacy checkpoint syncing.
	if syncMode == 1 || syncMode == 2 {
		// Assemble checkpoint 0
		s, _, head := server.chtIndexer.Sections()
		cp := &params.TrustedCheckpoint{
			SectionIndex: 0,
			SectionHead:  head,
			CHTRoot:      light.GetChtRoot(server.db, s-1, head),
			BloomRoot:    light.GetBloomTrieRoot(server.db, s-1, head),
		}
		if syncMode == 1 {
			// Register the assembled checkpoint as hardcoded one.
			client.pm.checkpoint = cp
			client.pm.blockchain.(*light.LightChain).AddTrustedCheckpoint(cp)
		} else {
			// Register the assembled checkpoint into oracle.
			header := server.backend.Blockchain().CurrentHeader()

			data := append([]byte{0x19, 0x00}, append(registrarAddr.Bytes(), append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, cp.Hash().Bytes()...)...)...)
			sig, _ := crypto.Sign(crypto.Keccak256(data), signerKey)
			sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
			if _, err := server.pm.reg.contract.RegisterCheckpoint(bind.NewKeyedTransactor(signerKey), cp.SectionIndex, cp.Hash().Bytes(), new(big.Int).Sub(header.Number, big.NewInt(1)), header.ParentHash, [][]byte{sig}); err != nil {
				t.Error("register checkpoint failed", err)
			}
			server.backend.Commit()

			// Wait for the checkpoint registration
			for {
				_, hash, _, err := server.pm.reg.contract.Contract().GetLatestCheckpoint(nil)
				if err != nil || hash == [32]byte{} {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				break
			}
			expected += 1
		}
	}

	done := make(chan error)
	client.pm.reg.syncDoneHook = func() {
		header := client.pm.blockchain.CurrentHeader()
		if header.Number.Uint64() == expected {
			done <- nil
		} else {
			done <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expected, header.Number)
		}
	}

	// Create connected peer pair.
	peer, err1, lPeer, err2 := newTestPeerPair("peer", protocol, server.pm, client.pm)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 2 handshake error: %v", err)
	}
	server.rPeer, client.rPeer = peer, lPeer

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
