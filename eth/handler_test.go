package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/p2p"
)

// Tests that hashes can be retrieved from a remote chain by hashes in reverse
// order.
func TestGetBlockHashes(t *testing.T) {
	pm := newTestProtocolManager(downloader.MaxHashFetch+15, nil)
	peer, _ := newTestPeer("peer", pm, true)
	defer peer.close()

	// Create a batch of tests for various scenarios
	limit := downloader.MaxHashFetch
	tests := []struct {
		origin common.Hash
		number int
		result int
	}{
		{common.Hash{}, 1, 0},                                 // Make sure non existent hashes don't return results
		{pm.chainman.Genesis().Hash(), 1, 0},                  // There are no hashes to retrieve up from the genesis
		{pm.chainman.GetBlockByNumber(5).Hash(), 5, 5},        // All the hashes including the genesis requested
		{pm.chainman.GetBlockByNumber(5).Hash(), 10, 5},       // More hashes than available till the genesis requested
		{pm.chainman.GetBlockByNumber(100).Hash(), 10, 10},    // All hashes available from the middle of the chain
		{pm.chainman.CurrentBlock().Hash(), 10, 10},           // All hashes available from the head of the chain
		{pm.chainman.CurrentBlock().Hash(), limit, limit},     // Request the maximum allowed hash count
		{pm.chainman.CurrentBlock().Hash(), limit + 1, limit}, // Request more than the maximum allowed hash count
	}
	// Run each of the tests and verify the results against the chain
	for i, tt := range tests {
		// Assemble the hash response we would like to receive
		resp := make([]common.Hash, tt.result)
		if len(resp) > 0 {
			from := pm.chainman.GetBlock(tt.origin).NumberU64() - 1
			for j := 0; j < len(resp); j++ {
				resp[j] = pm.chainman.GetBlockByNumber(uint64(int(from) - j)).Hash()
			}
		}
		// Send the hash request and verify the response
		p2p.Send(peer.app, 0x03, getBlockHashesData{tt.origin, uint64(tt.number)})
		if err := p2p.ExpectMsg(peer.app, 0x04, resp); err != nil {
			t.Errorf("test %d: block hashes mismatch: %v", i, err)
		}
	}
}

// Tests that hashes can be retrieved from a remote chain by numbers in forward
// order.
func TestGetBlockHashesFromNumber(t *testing.T) {
	pm := newTestProtocolManager(downloader.MaxHashFetch+15, nil)
	peer, _ := newTestPeer("peer", pm, true)
	defer peer.close()

	// Create a batch of tests for various scenarios
	limit := downloader.MaxHashFetch
	tests := []struct {
		origin uint64
		number int
		result int
	}{
		{pm.chainman.CurrentBlock().NumberU64() + 1, 1, 0},     // Out of bounds requests should return empty
		{pm.chainman.CurrentBlock().NumberU64(), 1, 1},         // Make sure the head hash can be retrieved
		{pm.chainman.CurrentBlock().NumberU64() - 4, 5, 5},     // All hashes, including the head hash requested
		{pm.chainman.CurrentBlock().NumberU64() - 4, 10, 5},    // More hashes requested than available till the head
		{pm.chainman.CurrentBlock().NumberU64() - 100, 10, 10}, // All hashes available from the middle of the chain
		{0, 10, 10},           // All hashes available from the root of the chain
		{0, limit, limit},     // Request the maximum allowed hash count
		{0, limit + 1, limit}, // Request more than the maximum allowed hash count
		{0, 1, 1},             // Make sure the genesis hash can be retrieved
	}
	// Run each of the tests and verify the results against the chain
	for i, tt := range tests {
		// Assemble the hash response we would like to receive
		resp := make([]common.Hash, tt.result)
		for j := 0; j < len(resp); j++ {
			resp[j] = pm.chainman.GetBlockByNumber(tt.origin + uint64(j)).Hash()
		}
		// Send the hash request and verify the response
		p2p.Send(peer.app, 0x08, getBlockHashesFromNumberData{tt.origin, uint64(tt.number)})
		if err := p2p.ExpectMsg(peer.app, 0x04, resp); err != nil {
			t.Errorf("test %d: block hashes mismatch: %v", i, err)
		}
	}
}
