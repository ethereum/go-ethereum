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
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// verifyImportEvent verifies that one single event arrive on an import channel.
func verifyImportEvent(t *testing.T, imported chan interface{}, arrive bool) {
	if arrive {
		select {
		case <-imported:
		case <-time.After(time.Second):
			t.Fatalf("import timeout")
		}
	} else {
		select {
		case <-imported:
			t.Fatalf("import invoked")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// verifyImportDone verifies that no more events are arriving on an import channel.
func verifyImportDone(t *testing.T, imported chan interface{}) {
	select {
	case <-imported:
		t.Fatalf("extra block imported")
	case <-time.After(50 * time.Millisecond):
	}
}

// verifyChainHeight verifies the chain height is as expected.
func verifyChainHeight(t *testing.T, fetcher *lightFetcher, height uint64) {
	local := fetcher.chain.CurrentHeader().Number.Uint64()
	if local != height {
		t.Fatalf("chain height mismatch, got %d, want %d", local, height)
	}
}

func TestSequentialAnnouncementsLes2(t *testing.T) { testSequentialAnnouncements(t, 2) }
func TestSequentialAnnouncementsLes3(t *testing.T) { testSequentialAnnouncements(t, 3) }

func testSequentialAnnouncements(t *testing.T, protocol int) {
	netconfig := testnetConfig{
		blocks:    4,
		protocol:  protocol,
		nopruning: true,
	}
	s, c, teardown := newClientServerEnv(t, netconfig)
	defer teardown()

	// Create connected peer pair, the initial signal from LES server
	// is discarded to prevent syncing.
	p1, _, err := newTestPeerPair("peer", protocol, s.handler, c.handler, true)
	if err != nil {
		t.Fatalf("Failed to create peer pair %v", err)
	}
	importCh := make(chan interface{})
	c.handler.fetcher.newHeadHook = func(header *types.Header) {
		importCh <- header
	}
	for i := uint64(1); i <= s.backend.Blockchain().CurrentHeader().Number.Uint64(); i++ {
		header := s.backend.Blockchain().GetHeaderByNumber(i)
		hash, number := header.Hash(), header.Number.Uint64()
		td := rawdb.ReadTd(s.db, hash, number)

		announce := announceData{hash, number, td, 0, nil}
		if p1.cpeer.announceType == announceTypeSigned {
			announce.sign(s.handler.server.privateKey)
		}
		p1.cpeer.sendAnnounce(announce)
		verifyImportEvent(t, importCh, true)
	}
	verifyImportDone(t, importCh)
	verifyChainHeight(t, c.handler.fetcher, 4)
}

func TestGappedAnnouncementsLes2(t *testing.T) { testGappedAnnouncements(t, 2) }
func TestGappedAnnouncementsLes3(t *testing.T) { testGappedAnnouncements(t, 3) }

func testGappedAnnouncements(t *testing.T, protocol int) {
	netconfig := testnetConfig{
		blocks:    4,
		protocol:  protocol,
		nopruning: true,
	}
	s, c, teardown := newClientServerEnv(t, netconfig)
	defer teardown()

	// Create connected peer pair, the initial signal from LES server
	// is discarded to prevent syncing.
	peer, _, err := newTestPeerPair("peer", protocol, s.handler, c.handler, true)
	if err != nil {
		t.Fatalf("Failed to create peer pair %v", err)
	}
	done := make(chan *types.Header, 1)
	c.handler.fetcher.newHeadHook = func(header *types.Header) { done <- header }

	// Prepare announcement by latest header.
	latest := s.backend.Blockchain().CurrentHeader()
	hash, number := latest.Hash(), latest.Number.Uint64()
	td := rawdb.ReadTd(s.db, hash, number)

	// Sign the announcement if necessary.
	announce := announceData{hash, number, td, 0, nil}
	if peer.cpeer.announceType == announceTypeSigned {
		announce.sign(s.handler.server.privateKey)
	}
	peer.cpeer.sendAnnounce(announce)

	<-done // Wait syncing
	verifyChainHeight(t, c.handler.fetcher, 4)

	// Send a reorged announcement
	blocks, _ := core.GenerateChain(rawdb.ReadChainConfig(s.db, s.backend.Blockchain().Genesis().Hash()), s.backend.Blockchain().GetBlockByNumber(3),
		ethash.NewFaker(), s.db, 2, func(i int, gen *core.BlockGen) {
			gen.OffsetTime(-9) // higher block difficulty
		})
	s.backend.Blockchain().InsertChain(blocks)

	<-done // Wait syncing
	verifyChainHeight(t, c.handler.fetcher, 5)
}

func TestInvalidAnnouncesLES2(t *testing.T) { testInvalidAnnounces(t, lpv2) }
func TestInvalidAnnouncesLES3(t *testing.T) { testInvalidAnnounces(t, lpv3) }
func TestInvalidAnnouncesLES4(t *testing.T) { testInvalidAnnounces(t, lpv4) }

func testInvalidAnnounces(t *testing.T, protocol int) {
	netconfig := testnetConfig{
		blocks:    4,
		protocol:  protocol,
		nopruning: true,
	}
	s, c, teardown := newClientServerEnv(t, netconfig)
	defer teardown()

	// Create connected peer pair, the initial signal from LES server
	// is discarded to prevent syncing.
	peer, _, err := newTestPeerPair("peer", lpv3, s.handler, c.handler, true)
	if err != nil {
		t.Fatalf("Failed to create peer pair %v", err)
	}
	done := make(chan *types.Header, 1)
	c.handler.fetcher.newHeadHook = func(header *types.Header) { done <- header }

	// Prepare announcement by latest header.
	headerOne := s.backend.Blockchain().GetHeaderByNumber(1)
	hash, number := headerOne.Hash(), headerOne.Number.Uint64()
	td := big.NewInt(params.GenesisDifficulty.Int64() + 200) // bad td

	// Sign the announcement if necessary.
	announce := announceData{hash, number, td, 0, nil}
	if peer.cpeer.announceType == announceTypeSigned {
		announce.sign(s.handler.server.privateKey)
	}
	peer.cpeer.sendAnnounce(announce)
	<-done // Wait syncing

	// Ensure the bad peer is evicted
	if c.handler.backend.peers.len() != 0 {
		t.Fatalf("Failed to evict invalid peer")
	}
}
