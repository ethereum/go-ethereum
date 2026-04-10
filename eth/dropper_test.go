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

package eth

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func makePeers(n int) []*p2p.Peer {
	peers := make([]*p2p.Peer, n)
	for i := range peers {
		id := enode.ID{byte(i)}
		peers[i] = p2p.NewPeer(id, fmt.Sprintf("peer%d", i), nil)
	}
	return peers
}

func TestProtectedPeersNoStats(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return nil }

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protected peers with nil stats, got %d", len(protected))
	}
}

func TestProtectedPeersEmptyStats(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats {
		return map[string]PeerInclusionStats{}
	}

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protected peers with empty stats, got %d", len(protected))
	}
}

func TestProtectedPeersTopPeer(t *testing.T) {
	// 20 peers, 10% of 20 = 2 protected per category.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]PeerInclusionStats)
	stats[peers[0].ID().String()] = PeerInclusionStats{Finalized: 100}
	stats[peers[1].ID().String()] = PeerInclusionStats{RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 2 {
		t.Fatalf("expected 2 protected peers, got %d", len(protected))
	}
	if !protected[peers[0]] {
		t.Fatal("peer 0 should be protected (top Finalized)")
	}
	if !protected[peers[1]] {
		t.Fatal("peer 1 should be protected (top RecentIncluded)")
	}
}

func TestProtectedPeersZeroScore(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(10)
	stats := make(map[string]PeerInclusionStats)
	for _, p := range peers {
		stats[p.ID().String()] = PeerInclusionStats{}
	}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 0 {
		t.Fatalf("expected no protection with zero scores, got %d", len(protected))
	}
}

func TestProtectedPeersOverlap(t *testing.T) {
	// One peer is top in both categories — counted once.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]PeerInclusionStats)
	stats[peers[0].ID().String()] = PeerInclusionStats{Finalized: 100, RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	protected := cm.protectedPeers(peers)
	if len(protected) != 1 {
		t.Fatalf("expected 1 protected peer (overlap), got %d", len(protected))
	}
}

func TestProtectedPeersNilFunc(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	// peerStatsFunc is nil (default).

	peers := makePeers(10)
	protected := cm.protectedPeers(peers)
	if protected != nil {
		t.Fatalf("expected nil with nil stats func, got %v", protected)
	}
}
