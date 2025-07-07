package pathdb

import (
	"crypto/rand"
	"fmt"
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

	// Add storage nodes using single-level sharded structure
	for accountHash, slots := range storageNodes {
		accountHex := accountHash.Hex()

		for path := range slots {
			// Construct the combined key but use only path for shard calculation
			key := accountHex + path
			shardIndex := getStorageShardIndex(path) // Use only path for sharding
			shardMap := l.storageNodes[shardIndex]

			list, exists := shardMap[key]
			if !exists {
				list = make([]common.Hash, 0, 16)
			}
			list = append(list, stateHash)
			shardMap[key] = list
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
			}
			// Initialize all 16 storage node shards
			for i := 0; i < 16; i++ {
				lookup.storageNodes[i] = make(map[string][]common.Hash)
			}

			var stateHash common.Hash
			rand.Read(stateHash[:])

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Clear the nodes maps for each iteration
				lookup.accountNodes = make(map[string][]common.Hash)
				// Reinitialize all 16 storage node shards
				for j := 0; j < 16; j++ {
					lookup.storageNodes[j] = make(map[string][]common.Hash)
				}

				lookup.addNodes(stateHash, accountNodes, storageNodes)
			}
		})
	}
}

func TestConcurrentStorageNodesUpdate(b *testing.T) {
	// Create a lookup instance
	lookup := &lookup{
		accountNodes: make(map[string][]common.Hash),
	}
	// Initialize all 16 storage node shards
	for i := 0; i < 16; i++ {
		lookup.storageNodes[i] = make(map[string][]common.Hash)
	}

	// Create test data with known shard distribution
	testData := map[common.Hash]map[string]*trienode.Node{}

	// Create accounts that will distribute across different shards
	for i := 0; i < 100; i++ {
		var accountHash common.Hash
		accountHash[0] = byte(i)

		testData[accountHash] = make(map[string]*trienode.Node)

		// Create paths that will hash to different shards
		for j := 0; j < 10; j++ {
			path := fmt.Sprintf("path_%d_%d", i, j)
			var nodeHash common.Hash
			nodeHash[0] = byte(j)

			testData[accountHash][path] = &trienode.Node{Hash: nodeHash}
		}
	}

	// Add nodes using the concurrent method
	var stateHash common.Hash
	stateHash[0] = 0x42
	lookup.addNodes(stateHash, nil, testData)

	// Verify that all nodes were added correctly
	totalNodes := 0
	for accountHash, slots := range testData {
		accountHex := accountHash.Hex()
		for path := range slots {
			key := accountHex + path
			shardIndex := getStorageShardIndex(path)

			list, exists := lookup.storageNodes[shardIndex][key]
			if !exists {
				b.Errorf("Node not found: account=%x, path=%s, shard=%d", accountHash, path, shardIndex)
				continue
			}

			if len(list) != 1 {
				b.Errorf("Expected 1 state hash, got %d: account=%x, path=%s", len(list), accountHash, path)
				continue
			}

			if list[0] != stateHash {
				b.Errorf("Expected state hash %x, got %x: account=%x, path=%s", stateHash, list[0], accountHash, path)
				continue
			}

			totalNodes++
		}
	}

	expectedTotal := 100 * 10 // 100 accounts * 10 nodes each
	if totalNodes != expectedTotal {
		b.Errorf("Expected %d total nodes, got %d", expectedTotal, totalNodes)
	}

	// Verify shard distribution
	for i := 0; i < 16; i++ {
		shardSize := len(lookup.storageNodes[i])
		if shardSize == 0 {
			b.Logf("Shard %d is empty", i)
		} else {
			b.Logf("Shard %d has %d entries", i, shardSize)
		}
	}
}

func TestShardDistribution(b *testing.T) {
	// Test shard distribution with different path patterns
	paths := []string{
		"path_0_0", "path_0_1", "path_0_2", "path_0_3",
		"path_1_0", "path_1_1", "path_1_2", "path_1_3",
		"path_2_0", "path_2_1", "path_2_2", "path_2_3",
		"path_3_0", "path_3_1", "path_3_2", "path_3_3",
		"path_4_0", "path_4_1", "path_4_2", "path_4_3",
		"path_5_0", "path_5_1", "path_5_2", "path_5_3",
		"path_6_0", "path_6_1", "path_6_2", "path_6_3",
		"path_7_0", "path_7_1", "path_7_2", "path_7_3",
		"path_8_0", "path_8_1", "path_8_2", "path_8_3",
		"path_9_0", "path_9_1", "path_9_2", "path_9_3",
	}

	shardCounts := make(map[int]int)
	for _, path := range paths {
		shardIndex := getStorageShardIndex(path)
		shardCounts[shardIndex]++
		b.Logf("Path: %s -> Shard: %d", path, shardIndex)
	}

	b.Logf("Shard distribution:")
	for i := 0; i < 16; i++ {
		count := shardCounts[i]
		b.Logf("  Shard %d: %d paths", i, count)
	}

	// Check if we have a reasonable distribution
	usedShards := 0
	for _, count := range shardCounts {
		if count > 0 {
			usedShards++
		}
	}

	if usedShards < 4 {
		b.Logf("Warning: Only %d shards are being used out of 16", usedShards)
	}
}
