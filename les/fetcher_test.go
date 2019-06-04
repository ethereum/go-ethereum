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

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/p2p/enode"
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
	s, c, teardown := newClientServerEnv(t, 4, protocol, nil, nil, false, false)
	defer teardown()

	// Create connected peer pair.
	c.handler.fetcher.ignoreAnnounce = true // Ignore the first announce from peer which can trigger a resync.
	p1, err1, _, err2 := newTestPeerPair("peer", protocol, s.handler, c.handler)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 2 handshake error: %v", err)
	}
	c.handler.fetcher.ignoreAnnounce = false

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
	s, c, teardown := newClientServerEnv(t, 4, protocol, nil, nil, false, false)
	defer teardown()

	// Create connected peer pair.
	c.handler.fetcher.ignoreAnnounce = true // Ignore the first announce from peer which can trigger a resync.
	p1, err1, _, err2 := newTestPeerPair("peer", protocol, s.handler, c.handler)
	select {
	case <-time.After(time.Millisecond * 100):
	case err := <-err1:
		t.Fatalf("peer 1 handshake error: %v", err)
	case err := <-err2:
		t.Fatalf("peer 2 handshake error: %v", err)
	}
	c.handler.fetcher.ignoreAnnounce = false

	// Prepare announcement by latest header.
	latest := s.backend.Blockchain().CurrentHeader()
	hash, number := latest.Hash(), latest.Number.Uint64()
	td := rawdb.ReadTd(s.db, hash, number)

	// Sign the announcement if necessary.
	announce := announceData{hash, number, td, 0, nil}
	if p1.cpeer.announceType == announceTypeSigned {
		announce.sign(s.handler.server.privateKey)
	}
	p1.cpeer.sendAnnounce(announce)
	time.Sleep(10 * time.Millisecond)
	verifyChainHeight(t, c.handler.fetcher, 4)
}

func TestTrustedAnnouncementsLes2(t *testing.T) { testTrustedAnnouncement(t, 2) }
func TestTrustedAnnouncementsLes3(t *testing.T) { testTrustedAnnouncement(t, 3) }

func testTrustedAnnouncement(t *testing.T, protocol int) {
	var (
		servers   []*testServer
		teardowns []func()
		nodes     []*enode.Node
		ids       []string
		cpeers    []*clientPeer
		speers    []*serverPeer
	)
	for i := 0; i < 10; i++ {
		s, n, teardown := newTestServerPeer(t, 10, protocol)

		servers = append(servers, s)
		nodes = append(nodes, n)
		teardowns = append(teardowns, teardown)

		// A half of them are trusted servers.
		if i < 5 {
			ids = append(ids, n.String())
		}
	}
	config := &eth.ULCConfig{
		MinTrustedFraction: 60, // At least 3 approvals
		TrustedServers:     ids,
	}
	_, c, teardown := newClientServerEnv(t, 0, protocol, nil, config, false, false)
	defer teardown()
	defer func() {
		for i := 0; i < len(teardowns); i++ {
			teardowns[i]()
		}
	}()

	c.handler.fetcher.ignoreAnnounce = true // Ignore the first announce from peer which can trigger a resync.

	// Connect all server instances.
	for i := 0; i < len(servers); i++ {
		sp, cp, err := connect(servers[i].handler, nodes[i].ID(), c.handler, protocol)
		if err != nil {
			t.Fatalf("connect server and client failed, err %s", err)
		}
		cpeers = append(cpeers, cp)
		speers = append(speers, sp)
	}
	c.handler.fetcher.ignoreAnnounce = false

	check := func(height []uint64, expected uint64, callback func()) {
		for i := 0; i < len(height); i++ {
			for j := 0; j < len(servers); j++ {
				h := servers[j].backend.Blockchain().GetHeaderByNumber(height[i])
				hash, number := h.Hash(), h.Number.Uint64()
				td := rawdb.ReadTd(servers[j].db, hash, number)

				// Sign the announcement if necessary.
				announce := announceData{hash, number, td, 0, nil}
				p := cpeers[j]
				if p.announceType == announceTypeSigned {
					announce.sign(servers[j].handler.server.privateKey)
				}
				p.sendAnnounce(announce)

				if j < 2 {
					time.Sleep(10 * time.Millisecond) // Ensure the announcement has been processed.
					if !c.handler.fetcher.queryAnnounced(speers[j], hash) {
						t.Fatalf("the announcement from server peer %d should be kept", j+1)
					}
				} else if j == 2 {
					// The block should be imported
				}
			}
		}
		if callback != nil {
			callback()
		}
		time.Sleep(10 * time.Millisecond) // Ensure the announcement has been processed.
		verifyChainHeight(t, c.handler.fetcher, expected)
	}
	check([]uint64{1}, 1, nil) // Sequential announcements
	check([]uint64{4}, 4, nil) // ULC-style light syncing, rollback untrusted headers

	done := make(chan struct{})
	c.handler.fetcher.syncingHook = func() { <-done }
	check([]uint64{6, 8}, 8, func() { done <- struct{}{} }) // ULC-style light syncing, keep the later trusted announces.

	c.handler.fetcher.syncingHook = nil
	check([]uint64{10}, 10, nil) // Sync the whole chain.
}

func TestAnnounceDelayLes2(t *testing.T) { testAnnounceDelay(t, 2) }
func TestAnnounceDelayLes3(t *testing.T) { testAnnounceDelay(t, 3) }

func testAnnounceDelay(t *testing.T, protocol int) {
	var (
		servers   []*testServer
		teardowns []func()
		nodes     []*enode.Node
		cpeers    []*clientPeer
		speers    []*serverPeer
	)
	for i := 0; i < 5; i++ {
		s, n, teardown := newTestServerPeer(t, 10, protocol)

		servers = append(servers, s)
		nodes = append(nodes, n)
		teardowns = append(teardowns, teardown)
	}
	_, c, teardown := newClientServerEnv(t, 0, protocol, nil, nil, false, false)
	defer teardown()
	defer func() {
		for i := 0; i < len(teardowns); i++ {
			teardowns[i]()
		}
	}()

	c.handler.fetcher.ignoreAnnounce = true // Ignore the first announce from peer which can trigger a resync.
	// Connect all server instances.
	for i := 0; i < len(servers); i++ {
		sp, cp, err := connect(servers[i].handler, nodes[i].ID(), c.handler, protocol)
		if err != nil {
			t.Fatalf("connect server and client failed, err %s", err)
		}
		cpeers = append(cpeers, cp)
		speers = append(speers, sp)
	}
	c.handler.fetcher.ignoreAnnounce = false

	delays := make(map[*serverPeer]time.Duration)
	c.handler.fetcher.addDelayHook = func(p *serverPeer, delay time.Duration) { delays[p] = delay }

	for i := 0; i < 2; i++ {
		h := servers[i].backend.Blockchain().GetHeaderByNumber(1)
		hash, number := h.Hash(), h.Number.Uint64()
		td := rawdb.ReadTd(servers[i].db, hash, number)

		announce := announceData{hash, number, td, 0, nil}
		p := cpeers[i]
		if p.announceType == announceTypeSigned {
			announce.sign(servers[i].handler.server.privateKey)
		}
		p.sendAnnounce(announce)
	}
	time.Sleep(10 * time.Millisecond)
	if d, exist := delays[speers[1]]; !exist || d > time.Millisecond {
		t.Fatalf("the second announcement should be confirmed soon")
	}
}
