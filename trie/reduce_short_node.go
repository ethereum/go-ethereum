package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// reduceShortNode returns all possible short nodes that have the same child value as the original node but a shorter prefix key
// indexed by the corresponding original node key prefix.

// e.g for a shortnode(abcd, 0x01), it returns:
// - abc -> shortnode(d, 0x01)
// - ab -> shortnode(cd, 0x01)
// - a -> shortnode(bcd, 0x01)
func ReduceShortNode(n []byte) (nodes map[string]*trienode.Node, ok bool) {
	// The post-state proof is a valid exclusion proof
	// We now assess if the deletion of the key resulted in a trie reduction by checking the last proof node (post-state)
	sn, ok := mustDecodeNode(crypto.Keccak256(n), n).(*shortNode)
	if !ok {
		// The last proof node in the post-state is not a short node, this means that the deletion did not
		// result in any trie reduction, so there is no need to add orphan nodes to the pre-state trie
		return nil, false
	}

	shortNodes := make(map[string]*trienode.Node, 0)

	hasher := newHasher(false)
	defer returnHasherToPool(hasher)

	// If the node is a one-nibble node, it can not be reduced further
	if len(sn.Key) == 1 {
		return shortNodes, true
	}

	// Loop through all possible prefixes of the original node key
	for i := 1; i < len(sn.Key); i++ {
		collapsed, hashed := hasher.proofHash(&shortNode{
			Key: sn.Key[i:],
			Val: sn.Val,
		})
		if hash, ok := hashed.(hashNode); ok {
			// If the node's database encoding is a hash (or is the
			// root node), it becomes a proof element.
			enc := nodeToBytes(collapsed)
			if !ok {
				hash = hasher.hashData(enc)
			}
			shortNodes[string(sn.Key[:i])] = trienode.New(common.Hash(hash), enc)
		}
	}

	return shortNodes, true
}
