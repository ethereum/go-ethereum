package bintrie

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestGroupedSerializationDebug helps understand the grouped serialization format
func TestGroupedSerializationDebug(t *testing.T) {
	leftHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	rightHash := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	node := &InternalNode{
		depth: 0,
		left:  HashedNode(leftHash),
		right: HashedNode(rightHash),
	}

	serialized := SerializeNode(node, MaxGroupDepth)
	t.Logf("Serialized length: %d", len(serialized))
	t.Logf("Type: %d, GroupDepth: %d", serialized[0], serialized[1])

	bitmapSize := BitmapSizeForDepth(MaxGroupDepth)
	bitmap := serialized[2 : 2+bitmapSize]
	t.Logf("Bitmap: %x", bitmap)

	// Count and show set bits
	for i := 0; i < 256; i++ {
		if bitmap[i/8]>>(7-(i%8))&1 == 1 {
			t.Logf("Bit %d is set", i)
		}
	}

	// Deserialize
	deserialized, err := DeserializeNode(serialized, 0)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	t.Logf("Deserialized type: %T", deserialized)

	// Walk the tree and print structure
	printTree(t, deserialized, 0, "root")
}

func printTree(t *testing.T, node BinaryNode, depth int, path string) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	switch n := node.(type) {
	case *InternalNode:
		t.Logf("%s%s: InternalNode (depth=%d)", indent, path, n.depth)
		printTree(t, n.left, depth+1, path+"/L")
		printTree(t, n.right, depth+1, path+"/R")
	case HashedNode:
		t.Logf("%s%s: HashedNode(%x)", indent, path, common.Hash(n))
	case Empty:
		t.Logf("%s%s: Empty", indent, path)
	default:
		t.Logf("%s%s: %T", indent, path, node)
	}
}

// TestFullDepth8Tree tests a full 8-level tree (all 256 bottom positions filled)
func TestFullDepth8Tree(t *testing.T) {
	// Build a full 8-level tree
	root := buildFullTree(0, 8)

	serialized := SerializeNode(root, MaxGroupDepth)
	t.Logf("Full tree serialized length: %d", len(serialized))
	t.Logf("Expected: 1 + 1 + 32 + 256*32 = %d", 1+1+32+256*32)

	// Count set bits in bitmap
	bitmapSize := BitmapSizeForDepth(MaxGroupDepth)
	bitmap := serialized[2 : 2+bitmapSize]
	count := 0
	for i := 0; i < 256; i++ {
		if bitmap[i/8]>>(7-(i%8))&1 == 1 {
			count++
		}
	}
	t.Logf("Set bits in bitmap: %d", count)

	// Deserialize and verify structure
	deserialized, err := DeserializeNode(serialized, 0)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	// Verify it's an InternalNode with depth 0
	in, ok := deserialized.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", deserialized)
	}
	if in.depth != 0 {
		t.Errorf("Expected depth 0, got %d", in.depth)
	}

	// Count leaves at depth 8
	leafCount := countLeavesAtDepth(deserialized, 8, 0)
	t.Logf("Leaves at depth 8: %d", leafCount)
	if leafCount != 256 {
		t.Errorf("Expected 256 leaves, got %d", leafCount)
	}
}

func buildFullTree(depth, maxDepth int) BinaryNode {
	if depth == maxDepth {
		// Create a unique hash for this position
		var h common.Hash
		h[0] = byte(depth)
		h[1] = byte(depth >> 8)
		return HashedNode(h)
	}
	return &InternalNode{
		depth: depth,
		left:  buildFullTree(depth+1, maxDepth),
		right: buildFullTree(depth+1, maxDepth),
	}
}

func countLeavesAtDepth(node BinaryNode, targetDepth, currentDepth int) int {
	if currentDepth == targetDepth {
		if _, ok := node.(Empty); ok {
			return 0
		}
		return 1
	}
	in, ok := node.(*InternalNode)
	if !ok {
		return 0 // Terminated early
	}
	return countLeavesAtDepth(in.left, targetDepth, currentDepth+1) +
		countLeavesAtDepth(in.right, targetDepth, currentDepth+1)
}

// TestRoundTripPreservesHashes tests that round-trip preserves the original hashes
func TestRoundTripPreservesHashes(t *testing.T) {
	// Build a tree with known hashes at specific positions
	hashes := make([]common.Hash, 256)
	for i := range hashes {
		hashes[i] = common.BytesToHash([]byte(fmt.Sprintf("hash-%d", i)))
	}

	root := buildTreeWithHashes(0, 8, 0, hashes)

	serialized := SerializeNode(root, MaxGroupDepth)
	deserialized, err := DeserializeNode(serialized, 0)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	// Verify each hash at depth 8
	for i := 0; i < 256; i++ {
		node := navigateToLeaf(deserialized, i, 8)
		if node == nil {
			t.Errorf("Position %d: node is nil", i)
			continue
		}
		if node.Hash() != hashes[i] {
			t.Errorf("Position %d: hash mismatch, expected %x, got %x", i, hashes[i], node.Hash())
		}
	}
}

func buildTreeWithHashes(depth, maxDepth, position int, hashes []common.Hash) BinaryNode {
	if depth == maxDepth {
		return HashedNode(hashes[position])
	}
	return &InternalNode{
		depth: depth,
		left:  buildTreeWithHashes(depth+1, maxDepth, position*2, hashes),
		right: buildTreeWithHashes(depth+1, maxDepth, position*2+1, hashes),
	}
}

// TestCollectNodesGrouping verifies that CollectNodes only flushes at group boundaries
// and that the serialized/deserialized tree matches the original.
func TestCollectNodesGrouping(t *testing.T) {
	// Build a tree that spans multiple groups (16 levels = 2 groups)
	// This creates a tree where:
	// - Group 1: depths 0-7 (root group)
	// - Group 2: depths 8-15 (leaf groups, up to 256 of them)
	// Use unique hashes at leaves so we get unique serialized blobs
	root := buildDeepTreeUnique(0, 16, 0)

	// Compute the root hash before collection
	originalRootHash := root.Hash()

	// Collect and serialize all nodes, storing by hash
	serializedNodes := make(map[common.Hash][]byte)
	var collectedNodes []struct {
		path []byte
		node BinaryNode
	}

	err := root.CollectNodes(nil, func(path []byte, node BinaryNode) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedNodes = append(collectedNodes, struct {
			path []byte
			node BinaryNode
		}{pathCopy, node})

		// Serialize and store by hash
		serialized := SerializeNode(node, MaxGroupDepth)
		serializedNodes[node.Hash()] = serialized
	}, MaxGroupDepth)
	if err != nil {
		t.Fatalf("CollectNodes failed: %v", err)
	}

	// Count nodes by depth
	depthCounts := make(map[int]int)
	for _, cn := range collectedNodes {
		switch n := cn.node.(type) {
		case *InternalNode:
			depthCounts[n.depth]++
		case *StemNode:
			t.Logf("Collected StemNode at path len %d", len(cn.path))
		}
	}

	// With a 16-level tree:
	// - 1 node at depth 0 (the root group)
	// - 256 nodes at depth 8 (the second-level groups)
	// Total: 257 InternalNode groups
	if depthCounts[0] != 1 {
		t.Errorf("Expected 1 node at depth 0, got %d", depthCounts[0])
	}
	if depthCounts[8] != 256 {
		t.Errorf("Expected 256 nodes at depth 8, got %d", depthCounts[8])
	}

	t.Logf("Total collected nodes: %d", len(collectedNodes))
	t.Logf("Total serialized blobs: %d", len(serializedNodes))
	t.Logf("Depth counts: %v", depthCounts)

	// Now deserialize starting from the root hash
	// Create a resolver that looks up serialized data by hash
	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		if data, ok := serializedNodes[hash]; ok {
			return data, nil
		}
		return nil, fmt.Errorf("node not found: %x", hash)
	}

	// Deserialize the root
	rootData, ok := serializedNodes[originalRootHash]
	if !ok {
		t.Fatalf("Root hash not found in serialized nodes: %x", originalRootHash)
	}
	deserializedRoot, err := DeserializeNode(rootData, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize root: %v", err)
	}

	// Verify the deserialized root hash matches
	if deserializedRoot.Hash() != originalRootHash {
		t.Errorf("Deserialized root hash mismatch: expected %x, got %x", originalRootHash, deserializedRoot.Hash())
	}

	// Traverse both trees and compare structure at all 16 levels
	// We need to resolve HashedNodes in the deserialized tree to compare deeper
	err = compareTreesWithResolver(t, root, deserializedRoot, resolver, 0, 16, "root")
	if err != nil {
		t.Errorf("Tree comparison failed: %v", err)
	}

	t.Log("Tree comparison passed - deserialized tree matches original")
}

// compareTreesWithResolver compares two trees, resolving HashedNodes as needed
func compareTreesWithResolver(t *testing.T, original, deserialized BinaryNode, resolver NodeResolverFn, depth, maxDepth int, path string) error {
	if depth >= maxDepth {
		// At leaf level, just compare hashes
		if original.Hash() != deserialized.Hash() {
			return fmt.Errorf("hash mismatch at %s: original=%x, deserialized=%x", path, original.Hash(), deserialized.Hash())
		}
		return nil
	}

	// Get the actual nodes (resolve HashedNodes if needed)
	origNode := original
	deserNode := deserialized

	// Resolve deserialized HashedNode if needed
	if h, ok := deserNode.(HashedNode); ok {
		data, err := resolver(nil, common.Hash(h))
		if err != nil {
			return fmt.Errorf("failed to resolve deserialized node at %s: %v", path, err)
		}
		deserNode, err = DeserializeNode(data, depth)
		if err != nil {
			return fmt.Errorf("failed to deserialize node at %s: %v", path, err)
		}
	}

	// Both should be InternalNodes at this point
	origInternal, origOk := origNode.(*InternalNode)
	deserInternal, deserOk := deserNode.(*InternalNode)

	if !origOk || !deserOk {
		// Check if both are the same type
		if fmt.Sprintf("%T", origNode) != fmt.Sprintf("%T", deserNode) {
			return fmt.Errorf("type mismatch at %s: original=%T, deserialized=%T", path, origNode, deserNode)
		}
		// Both are non-InternalNode, compare hashes
		if origNode.Hash() != deserNode.Hash() {
			return fmt.Errorf("hash mismatch at %s: original=%x, deserialized=%x", path, origNode.Hash(), deserNode.Hash())
		}
		return nil
	}

	// Compare depths
	if origInternal.depth != deserInternal.depth {
		return fmt.Errorf("depth mismatch at %s: original=%d, deserialized=%d", path, origInternal.depth, deserInternal.depth)
	}

	// Recursively compare children
	if err := compareTreesWithResolver(t, origInternal.left, deserInternal.left, resolver, depth+1, maxDepth, path+"/L"); err != nil {
		return err
	}
	if err := compareTreesWithResolver(t, origInternal.right, deserInternal.right, resolver, depth+1, maxDepth, path+"/R"); err != nil {
		return err
	}

	return nil
}

func buildDeepTree(depth, maxDepth int) BinaryNode {
	if depth == maxDepth {
		// Create a unique hash for this leaf position
		var h common.Hash
		h[0] = byte(depth)
		h[1] = byte(depth >> 8)
		return HashedNode(h)
	}
	return &InternalNode{
		depth: depth,
		left:  buildDeepTree(depth+1, maxDepth),
		right: buildDeepTree(depth+1, maxDepth),
	}
}

// buildDeepTreeUnique builds a tree where each leaf has a unique hash based on its position
func buildDeepTreeUnique(depth, maxDepth, position int) BinaryNode {
	if depth == maxDepth {
		// Create a unique hash based on position in the tree
		var h common.Hash
		h[0] = byte(position)
		h[1] = byte(position >> 8)
		h[2] = byte(position >> 16)
		h[3] = byte(position >> 24)
		return HashedNode(h)
	}
	return &InternalNode{
		depth: depth,
		left:  buildDeepTreeUnique(depth+1, maxDepth, position*2),
		right: buildDeepTreeUnique(depth+1, maxDepth, position*2+1),
	}
}

// TestVariableGroupDepth tests serialization with different group depths (1-8)
func TestVariableGroupDepth(t *testing.T) {
	for groupDepth := 1; groupDepth <= MaxGroupDepth; groupDepth++ {
		t.Run(fmt.Sprintf("groupDepth=%d", groupDepth), func(t *testing.T) {
			// Build a tree with depth equal to groupDepth * 2 (two full groups)
			treeDepth := groupDepth * 2
			root := buildDeepTreeUnique(0, treeDepth, 0)
			originalHash := root.Hash()

			// Serialize with this group depth
			serialized := SerializeNode(root, groupDepth)

			// Verify header
			if serialized[0] != nodeTypeInternal {
				t.Errorf("Expected type byte %d, got %d", nodeTypeInternal, serialized[0])
			}
			if int(serialized[1]) != groupDepth {
				t.Errorf("Expected group depth %d, got %d", groupDepth, serialized[1])
			}

			// Verify bitmap size
			expectedBitmapSize := BitmapSizeForDepth(groupDepth)
			expectedMinLen := 1 + 1 + expectedBitmapSize // type + depth + bitmap
			if len(serialized) < expectedMinLen {
				t.Errorf("Serialized data too short: got %d, expected at least %d", len(serialized), expectedMinLen)
			}

			// Deserialize and verify hash matches
			deserialized, err := DeserializeNode(serialized, 0)
			if err != nil {
				t.Fatalf("DeserializeNode failed: %v", err)
			}

			if deserialized.Hash() != originalHash {
				t.Errorf("Hash mismatch after round-trip: expected %x, got %x", originalHash, deserialized.Hash())
			}

			// Collect nodes and verify correct grouping
			var collectedDepths []int
			err = root.CollectNodes(nil, func(path []byte, node BinaryNode) {
				if in, ok := node.(*InternalNode); ok {
					collectedDepths = append(collectedDepths, in.depth)
				}
			}, groupDepth)
			if err != nil {
				t.Fatalf("CollectNodes failed: %v", err)
			}

			// Verify all collected nodes are at group boundaries
			for _, depth := range collectedDepths {
				if depth%groupDepth != 0 {
					t.Errorf("Collected node at depth %d, but groupDepth is %d (not a boundary)", depth, groupDepth)
				}
			}

			t.Logf("groupDepth=%d: serialized=%d bytes, collected=%d nodes at depths %v",
				groupDepth, len(serialized), len(collectedDepths), collectedDepths)
		})
	}
}
