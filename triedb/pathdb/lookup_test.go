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

// addNodes is a helper method for testing that adds nodes to the lookup structure
func (l *lookup) addNodes(stateHash common.Hash, accountNodes map[string]*trienode.Node, storageNodes map[common.Hash]map[string]*trienode.Node) {
	// Add account nodes
	for path := range accountNodes {
		list, exists := l.accountNodes[path]
		if !exists {
			list = make([]common.Hash, 0, 16)
		}
		list = append(list, stateHash)
		l.accountNodes[path] = list
	}

	// Add storage nodes
	for accountHash, slots := range storageNodes {
		for path := range slots {
			key := accountHash.Hex() + path
			list, exists := l.storageNodes[key]
			if !exists {
				list = make([]common.Hash, 0, 16)
			}
			list = append(list, stateHash)
			l.storageNodes[key] = list
		}
	}
}

func BenchmarkAddNodes(b *testing.B) {
	tests := []struct {
		name                string
		accountNodeCount    int
		storageAccountCount int
		nodesPerAccount     int
	}{
		{
			name:                "Small-100-accounts-10-nodes",
			accountNodeCount:    100,
			storageAccountCount: 100,
			nodesPerAccount:     10,
		},
		{
			name:                "Medium-500-accounts-20-nodes",
			accountNodeCount:    500,
			storageAccountCount: 500,
			nodesPerAccount:     20,
		},
		{
			name:                "Large-2000-accounts-40-nodes",
			accountNodeCount:    2000,
			storageAccountCount: 2000,
			nodesPerAccount:     40,
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			accountNodes := generateRandomAccountNodes(tc.accountNodeCount)
			storageNodes := generateRandomStorageNodes(tc.storageAccountCount, tc.nodesPerAccount)

			lookup := &lookup{
				accountNodes: make(map[string][]common.Hash),
				storageNodes: make(map[string][]common.Hash),
			}

			var stateHash common.Hash
			rand.Read(stateHash[:])

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Clear the nodes maps for each iteration
				lookup.accountNodes = make(map[string][]common.Hash)
				lookup.storageNodes = make(map[string][]common.Hash)

				lookup.addNodes(stateHash, accountNodes, storageNodes)
			}
		})
	}
}
