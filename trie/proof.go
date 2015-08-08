package trie

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	//	"github.com/ethereum/go-ethereum/rlp"
)

/*
A merkle proof for the trie consists of the RLP data for all nodes
on the path from the root to the node of interest
*/

type TrieProof struct {
	Key        []byte
	Value      []byte
	InnerNodes []ProofNode // ShortNode or FullNode
	RootHash   []byte
}

func (proof *TrieProof) RlpData() interface{} {
	return []interface{}{proof.Key, proof.Value, proof.InnerNodes, proof.RootHash}
}

// Prove a byte array is in the trie
func (trie *Trie) Prove(key []byte) *TrieProof {
	if trie == nil {
		return nil
	}
	k := CompactHexDecode(key) // nibbles

	rootHash := trie.Hash()
	proof := &TrieProof{
		Key:      key,
		RootHash: rootHash,
	}
	// recursively appends nodes on the path from root to k to proof.InnerNodes
	if exists := trie.constructProof(trie.root, k, proof); !exists {
		return nil
	}
	return proof
}

// Verify the proof actually proves the given key byte-array is in the trie
func (proof *TrieProof) Verify(key, value, rootHash []byte) bool {
	if !bytes.Equal(key, proof.Key) {
		return false
	}
	if !bytes.Equal(value, proof.Value) {
		return false
	}
	if !bytes.Equal(rootHash, proof.RootHash) {
		return false
	}
	// build up proof nodes into proper nodes in a dumy trie
	trie := New(nil, nil)
	nextNode := Node(NewValueNode(trie, value))
	for _, proofNode := range proof.InnerNodes {
		nextNode = linkProofNodes(trie, proofNode, nextNode)
		if nextNode == nil {
			return false
		}
	}
	trie.root = nextNode
	finalHash := trie.Hash()
	return bytes.Equal(proof.RootHash, finalHash)
}

//--------------------------------------------------------------------------------

// Proof node contains some rlp encoded data that can be decoded into a Node
type ProofNode struct {
	Key   []byte   // bytes
	Nodes [][]byte `rlp:"nil"` // empty (ShortNode) or FullNode
}

func (proof ProofNode) RlpData() interface{} {
	var t []interface{}
	if proof.Nodes != nil {
		t = make([]interface{}, 17)
		for i, n := range proof.Nodes {
			t[i] = n
		}
	}
	return []interface{}{proof.Key, t}
}

// Create ShortNodefrom or FullNode. FullNode's should only have 17 entries in byte array
func (proof ProofNode) TrieNode(trie *Trie) Node {
	if len(proof.Nodes) == 0 {
		return &ShortNode{trie: trie, key: proof.Key}
	} else {
		fullNode := NewFullNode(trie)
		if len(proof.Nodes) != len(fullNode.nodes) {
			return nil
		}
		for i, hash := range proof.Nodes {
			fullNode.nodes[i] = trie.mknode(common.NewValueFromBytes(hash))
		}
		return fullNode
	}
}

// Create proof node from full node by rlp encoding all children
func FullNodeToProof(node *FullNode, key byte) ProofNode {
	proofNode := ProofNode{Key: []byte{key}, Nodes: make([][]byte, 17)}
	for i, n := range node.nodes {
		if n == nil || i == int(key) { // don't store the node for the branch we're proving
			proofNode.Nodes[i] = common.Encode("")
		} else {
			proofNode.Nodes[i] = common.Encode(n)
		}
	}
	return proofNode
}

func ShortNodeToProof(node *ShortNode) ProofNode {
	return ProofNode{Key: node.key, Nodes: nil}
}

//--------------------------------------------------------------------------------

// key should be nibbles
func (trie *Trie) constructProof(node Node, key []byte, proof *TrieProof) (exists bool) {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *ValueNode:
		// NOTE: we should have made sure the key matches by now
		// getting here == much success, very proof
		proof.Value = n.Val()
	case *HashNode:
		// resolve the hash node and call constructProof
		if exists := trie.constructProof(trie.trans(n), key, proof); !exists {
			return false
		}
	case *ShortNode:
		// chew off some key, constructProof on the value
		if !bytes.HasPrefix(key, n.Key()) {
			return false
		}
		k := key[len(n.Key()):]
		if exists := trie.constructProof(n.Value(), k, proof); !exists {
			return false
		}
		proof.InnerNodes = append(proof.InnerNodes, ShortNodeToProof(n))
	case *FullNode:
		// pick the right branch and carry on
		if exists := trie.constructProof(n.branch(key[0]), key[1:], proof); !exists {
			return false
		}
		proof.InnerNodes = append(proof.InnerNodes, FullNodeToProof(n, key[0]))
	default:
		panic("unknown node type")
	}
	return true
}

// fill in the nodes as we climb up the tree.
//returns nil if full node has an empty key or too many branches
func linkProofNodes(trie *Trie, proofNode ProofNode, node Node) Node {
	nextNode := proofNode.TrieNode(trie)
	if nextNode == nil {
		return nil
	}
	switch nextNode := nextNode.(type) {
	case *ShortNode:
		nextNode.value = node
		return nextNode
	case *FullNode:
		if len(proofNode.Key) == 0 {
			return nil
		}
		nextNode.nodes[proofNode.Key[0]] = node
		return nextNode
	default:
		panic("invalid proof node")
	}
	return nil
}
