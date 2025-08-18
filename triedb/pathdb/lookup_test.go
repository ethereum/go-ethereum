package pathdb

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// generateRandomAccountNodes creates a map of random trie nodes
func generateRandomAccountNodes(count int) map[string]*trienode.Node {
	nodes := make(map[string]*trienode.Node, count)

	for i := 0; i < count; i++ {
		path := make([]byte, 32)
		rand.Read(path)

		blob := make([]byte, 64)
		rand.Read(blob)

		var hash common.Hash
		rand.Read(hash[:])

		nodes[common.Bytes2Hex(path)] = &trienode.Node{Hash: hash, Blob: blob}
	}

	return nodes
}

// generateRandomStorageNodes creates a map of storage nodes organized by account
func generateRandomStorageNodes(accountCount, nodesPerAccount int) map[common.Hash]map[string]*trienode.Node {
	storageNodes := make(map[common.Hash]map[string]*trienode.Node, accountCount)

	for i := 0; i < accountCount; i++ {
		var hash common.Hash
		rand.Read(hash[:])

		storageNodes[hash] = generateRandomAccountNodes(nodesPerAccount)
	}

	return storageNodes
}

func BenchmarkAddNodes(b *testing.B) {
	tests := []struct {
		name             string
		accountNodeCount int
		nodesPerAccount  int
	}{
		{
			name:             "Small-100-accounts-10-nodes",
			accountNodeCount: 100,
			nodesPerAccount:  10,
		},
		{
			name:             "Medium-500-accounts-20-nodes",
			accountNodeCount: 500,
			nodesPerAccount:  20,
		},
		{
			name:             "Large-2000-accounts-40-nodes",
			accountNodeCount: 2000,
			nodesPerAccount:  40,
		},
		{
			name:             "XLarge-5000-accounts-50-nodes",
			accountNodeCount: 5000,
			nodesPerAccount:  50,
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			storageNodes := generateRandomStorageNodes(tc.accountNodeCount, tc.nodesPerAccount)

			lookup := &lookup{
				accountNodes: make(map[string][]common.Hash),
			}

			// Initialize all 16 storage node shards
			for i := 0; i < storageNodesShardCount; i++ {
				lookup.storageNodes[i] = make(map[trienodeKey][]common.Hash)
			}

			var state common.Hash
			rand.Read(state[:])

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Reset the lookup instance for each benchmark iteration
				for j := 0; j < storageNodesShardCount; j++ {
					lookup.storageNodes[j] = make(map[trienodeKey][]common.Hash)
				}

				lookup.addStorageNodes(state, storageNodes)
			}
		})
	}
}
