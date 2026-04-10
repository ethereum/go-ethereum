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

func TestFilterProtectedNoStats(t *testing.T) {
	// When the stats func returns nil/empty, all peers remain droppable.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return nil }

	peers := makePeers(10)
	result := cm.filterProtectedPeers(peers)
	if len(result) != len(peers) {
		t.Fatalf("expected all peers droppable with nil stats, got %d/%d", len(result), len(peers))
	}
}

func TestFilterProtectedEmptyStats(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats {
		return map[string]PeerInclusionStats{}
	}

	peers := makePeers(10)
	result := cm.filterProtectedPeers(peers)
	if len(result) != len(peers) {
		t.Fatalf("expected all peers droppable with empty stats, got %d/%d", len(result), len(peers))
	}
}

func TestFilterProtectedTopPeer(t *testing.T) {
	// 20 peers, maxDialPeers=20, 10% = 2 protected per category.
	// NewPeer creates non-inbound peers, so all go to dialed bucket.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]PeerInclusionStats)
	// Peer 0: top by Finalized
	stats[peers[0].ID().String()] = PeerInclusionStats{Finalized: 100}
	// Peer 1: top by RecentIncluded
	stats[peers[1].ID().String()] = PeerInclusionStats{RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	result := cm.filterProtectedPeers(peers)
	// 2 categories × 2 protected each = up to 4, but peers 0 and 1 are
	// different so both should be removed. Other peers have zero scores.
	protected := len(peers) - len(result)
	if protected != 2 {
		t.Fatalf("expected 2 protected peers, got %d", protected)
	}
	// Verify peers 0 and 1 are not in result.
	for _, p := range result {
		id := p.ID().String()
		if id == peers[0].ID().String() || id == peers[1].ID().String() {
			t.Fatalf("peer %s should be protected", id)
		}
	}
}

func TestFilterProtectedZeroScore(t *testing.T) {
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(10)
	stats := make(map[string]PeerInclusionStats)
	// All peers have zero stats.
	for _, p := range peers {
		stats[p.ID().String()] = PeerInclusionStats{}
	}
	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	result := cm.filterProtectedPeers(peers)
	if len(result) != len(peers) {
		t.Fatalf("expected no protection with zero scores, got %d protected", len(peers)-len(result))
	}
}

func TestFilterProtectedOverlap(t *testing.T) {
	// One peer is top in both categories — should only be removed once.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(20)
	stats := make(map[string]PeerInclusionStats)
	// Peer 0 is top in both.
	stats[peers[0].ID().String()] = PeerInclusionStats{Finalized: 100, RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	result := cm.filterProtectedPeers(peers)
	protected := len(peers) - len(result)
	if protected != 1 {
		t.Fatalf("expected 1 protected peer (overlap), got %d", protected)
	}
}

func TestFilterProtectedAllProtected(t *testing.T) {
	// Only 2 droppable peers, both are top by different categories.
	cm := &dropper{maxDialPeers: 20, maxInboundPeers: 30}

	peers := makePeers(2)
	stats := make(map[string]PeerInclusionStats)
	stats[peers[0].ID().String()] = PeerInclusionStats{Finalized: 100}
	stats[peers[1].ID().String()] = PeerInclusionStats{RecentIncluded: 5.0}

	cm.peerStatsFunc = func() map[string]PeerInclusionStats { return stats }

	result := cm.filterProtectedPeers(peers)
	if len(result) != 0 {
		t.Fatalf("expected all peers protected, got %d droppable", len(result))
	}
}
