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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/params"
)

// Test light syncing which will download all headers from genesis.
func TestLightSyncingLes3(t *testing.T) { testCheckpointSyncing(t, lpv3, 0) }

// Test legacy checkpoint syncing which will download tail headers
// based on a hardcoded checkpoint.
func TestLegacyCheckpointSyncingLes3(t *testing.T) { testCheckpointSyncing(t, lpv3, 1) }

// Test checkpoint syncing which will download tail headers based
// on a verified checkpoint.
func TestCheckpointSyncingLes3(t *testing.T) { testCheckpointSyncing(t, lpv3, 2) }

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
			client.handler.checkpoint = cp
			client.handler.backend.blockchain.AddTrustedCheckpoint(cp)
		} else {
			// Register the assembled checkpoint into oracle.
			header := server.backend.Blockchain().CurrentHeader()

			data := append([]byte{0x19, 0x00}, append(oracleAddr.Bytes(), append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, cp.Hash().Bytes()...)...)...)
			sig, _ := crypto.Sign(crypto.Keccak256(data), signerKey)
			sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
			auth, _ := bind.NewKeyedTransactorWithChainID(signerKey, big.NewInt(1337))
			if _, err := server.handler.server.oracle.Contract().RegisterCheckpoint(auth, cp.SectionIndex, cp.Hash().Bytes(), new(big.Int).Sub(header.Number, big.NewInt(1)), header.ParentHash, [][]byte{sig}); err != nil {
				t.Error("register checkpoint failed", err)
			}
			server.backend.Commit()

			// Wait for the checkpoint registration
			for {
				_, hash, _, err := server.handler.server.oracle.Contract().Contract().GetLatestCheckpoint(nil)
				if err != nil || hash == [32]byte{} {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				break
			}
			expected += 1
		}
	}

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

func TestMissOracleBackendLES3(t *testing.T)             { testMissOracleBackend(t, true, lpv3) }
func TestMissOracleBackendNoCheckpointLES3(t *testing.T) { testMissOracleBackend(t, false, lpv3) }

func testMissOracleBackend(t *testing.T, hasCheckpoint bool, protocol int) {
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

	s, _, head := server.chtIndexer.Sections()
	cp := &params.TrustedCheckpoint{
		SectionIndex: 0,
		SectionHead:  head,
		CHTRoot:      light.GetChtRoot(server.db, s-1, head),
		BloomRoot:    light.GetBloomTrieRoot(server.db, s-1, head),
	}
	// Register the assembled checkpoint into oracle.
	header := server.backend.Blockchain().CurrentHeader()

	data := append([]byte{0x19, 0x00}, append(oracleAddr.Bytes(), append([]byte{0, 0, 0, 0, 0, 0, 0, 0}, cp.Hash().Bytes()...)...)...)
	sig, _ := crypto.Sign(crypto.Keccak256(data), signerKey)
	sig[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	auth, _ := bind.NewKeyedTransactorWithChainID(signerKey, big.NewInt(1337))
	if _, err := server.handler.server.oracle.Contract().RegisterCheckpoint(auth, cp.SectionIndex, cp.Hash().Bytes(), new(big.Int).Sub(header.Number, big.NewInt(1)), header.ParentHash, [][]byte{sig}); err != nil {
		t.Error("register checkpoint failed", err)
	}
	server.backend.Commit()

	// Wait for the checkpoint registration
	for {
		_, hash, _, err := server.handler.server.oracle.Contract().Contract().GetLatestCheckpoint(nil)
		if err != nil || hash == [32]byte{} {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		break
	}
	expected += 1

	// Explicitly set the oracle as nil. In normal use case it can happen
	// that user wants to unlock something which blocks the oracle backend
	// initialisation. But at the same time syncing starts.
	//
	// See https://github.com/ethereum/go-ethereum/issues/20097 for more detail.
	//
	// In this case, client should run light sync or legacy checkpoint sync
	// if hardcoded checkpoint is configured.
	client.handler.backend.oracle = nil

	// For some private networks it can happen checkpoint syncing is enabled
	// but there is no hardcoded checkpoint configured.
	if hasCheckpoint {
		client.handler.checkpoint = cp
		client.handler.backend.blockchain.AddTrustedCheckpoint(cp)
	}

	done := make(chan error)
	client.handler.syncEnd = func(header *types.Header) {
		if header.Number.Uint64() == expected {
			done <- nil
		} else {
			done <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expected, header.Number)
		}
	}
	// Create connected peer pair.
	if _, _, err := newTestPeerPair("peer", 2, server.handler, client.handler, false); err != nil {
		t.Fatalf("Failed to connect testing peers %v", err)
	}
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

func TestSyncFromConfiguredCheckpointLES3(t *testing.T) { testSyncFromConfiguredCheckpoint(t, lpv3) }

func testSyncFromConfiguredCheckpoint(t *testing.T, protocol int) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= 2 && bts >= 2 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Generate 256+1 blocks (totally 2 CHT sections)
	netconfig := testnetConfig{
		blocks:    int(2*config.ChtSize + config.ChtConfirms),
		protocol:  protocol,
		indexFn:   waitIndexers,
		nopruning: true,
	}
	server, client, tearDown := newClientServerEnv(t, netconfig)
	defer tearDown()

	// Configure the local checkpoint(the first section)
	head := server.handler.blockchain.GetHeaderByNumber(config.ChtSize - 1).Hash()
	cp := &params.TrustedCheckpoint{
		SectionIndex: 0,
		SectionHead:  head,
		CHTRoot:      light.GetChtRoot(server.db, 0, head),
		BloomRoot:    light.GetBloomTrieRoot(server.db, 0, head),
	}
	client.handler.backend.config.SyncFromCheckpoint = true
	client.handler.backend.config.Checkpoint = cp
	client.handler.checkpoint = cp
	client.handler.backend.blockchain.AddTrustedCheckpoint(cp)

	var (
		start       = make(chan error, 1)
		end         = make(chan error, 1)
		expectStart = config.ChtSize - 1
		expectEnd   = 2*config.ChtSize + config.ChtConfirms
	)
	client.handler.syncStart = func(header *types.Header) {
		if header.Number.Uint64() == expectStart {
			start <- nil
		} else {
			start <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expectStart, header.Number)
		}
	}
	client.handler.syncEnd = func(header *types.Header) {
		if header.Number.Uint64() == expectEnd {
			end <- nil
		} else {
			end <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expectEnd, header.Number)
		}
	}
	// Create connected peer pair.
	if _, _, err := newTestPeerPair("peer", 2, server.handler, client.handler, false); err != nil {
		t.Fatalf("Failed to connect testing peers %v", err)
	}

	select {
	case err := <-start:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}

	select {
	case err := <-end:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}
}

func TestSyncAll(t *testing.T) { testSyncAll(t, lpv3) }

func testSyncAll(t *testing.T, protocol int) {
	config := light.TestServerIndexerConfig

	waitIndexers := func(cIndexer, bIndexer, btIndexer *core.ChainIndexer) {
		for {
			cs, _, _ := cIndexer.Sections()
			bts, _, _ := btIndexer.Sections()
			if cs >= 2 && bts >= 2 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	// Generate 256+1 blocks (totally 2 CHT sections)
	netconfig := testnetConfig{
		blocks:    int(2*config.ChtSize + config.ChtConfirms),
		protocol:  protocol,
		indexFn:   waitIndexers,
		nopruning: true,
	}
	server, client, tearDown := newClientServerEnv(t, netconfig)
	defer tearDown()

	client.handler.backend.config.SyncFromCheckpoint = true

	var (
		start       = make(chan error, 1)
		end         = make(chan error, 1)
		expectStart = uint64(0)
		expectEnd   = 2*config.ChtSize + config.ChtConfirms
	)
	client.handler.syncStart = func(header *types.Header) {
		if header.Number.Uint64() == expectStart {
			start <- nil
		} else {
			start <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expectStart, header.Number)
		}
	}
	client.handler.syncEnd = func(header *types.Header) {
		if header.Number.Uint64() == expectEnd {
			end <- nil
		} else {
			end <- fmt.Errorf("blockchain length mismatch, want %d, got %d", expectEnd, header.Number)
		}
	}
	// Create connected peer pair.
	if _, _, err := newTestPeerPair("peer", 2, server.handler, client.handler, false); err != nil {
		t.Fatalf("Failed to connect testing peers %v", err)
	}

	select {
	case err := <-start:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}

	select {
	case err := <-end:
		if err != nil {
			t.Error("sync failed", err)
		}
		return
	case <-time.NewTimer(10 * time.Second).C:
		t.Error("checkpoint syncing timeout")
	}
}
