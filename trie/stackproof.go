package trie

import (
	"fmt"

	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type pathProof struct {
	path  []byte
	proof []byte
}
type shortN struct {
	ext []byte
	val []byte
}
type fullN struct {
	siblings map[int]interface{}
}

func nodeToStacktrie(n node, key []byte) *StackTrie {
	st := stackTrieFromPool(nil)
	switch n := n.(type) {
	case *shortNode:
		st.nodeType = extNode
		st.key = append([]byte{}, n.Key...)
	case *fullNode:
		st.nodeType = branchNode
		idx := int(key[0])
		for i := 0; i < idx; i++ {
			sibling := n.Children[i]
			if sibling == nil {
				continue
			}
			siblingNode := stackTrieFromPool(nil)
			siblingNode.nodeType = hashedNode
			hash, _ := sibling.(hashNode)
			siblingNode.val = []byte(hash)
			st.children[i] = siblingNode
		}
	default:
		panic(fmt.Sprintf("%T", n))
	}
	return st
}

func initLeftside(rootHash common.Hash, root node, key []byte, proofDb ethdb.KeyValueReader, allowNonExistent bool) (*StackTrie, []byte, error) {
	// resolveNode retrieves and resolves trie node from merkle proof stream
	resolveNode := func(hash common.Hash) (node, error) {
		buf, _ := proofDb.Get(hash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
		}
		//fmt.Printf("[s] Looked up node %064x ok\n", hash)
		n, err := decodeNode(hash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %v", err)
		}
		return n, err
	}
	// If the root node is empty, resolve it first.
	// Root node must be included in the proof.
	if root == nil {
		n, err := resolveNode(rootHash)
		if err != nil {
			return nil, nil, err
		}
		root = n
	}
	var (
		err           error
		child, parent node
		keyrest       []byte
		valnode       []byte
	)
	key, parent = keybytesToHex(key), root
	stRoot := nodeToStacktrie(root, key)
	stParent := stRoot
	var stChild *StackTrie
	for {
		keyrest, child = get(parent, key, false)
		switch cld := child.(type) {
		case nil:
			// The trie doesn't contain the key. It's possible
			// the proof is a non-existing proof, but at least
			// we can prove all resolved nodes are correct, it's
			// enough for us to prove range.
			if allowNonExistent {
				return stRoot, nil, nil
			}
			return nil, nil, errors.New("the node is not contained in trie")
		case hashNode:
			child, err = resolveNode(common.BytesToHash(cld))
			if err != nil {
				return nil, nil, err
			}
		case valueNode:
			valnode = cld
			// The value node goes right into the child
			stParent.val = common.CopyBytes(cld)
			stParent.nodeType = leafNode
			// remove the terminator
			stParent.key = stParent.key[:len(stParent.key)-1]
			return stRoot, valnode, nil
		default:
			// we don't expect shortnodes or fullnodes
			panic(fmt.Sprintf("got %T", cld))
		}
		stChild = nodeToStacktrie(child, keyrest)
		// Link the parent and child.
		switch pnode := parent.(type) {
		case *shortNode:
			pnode.Val = child
			stParent.children[0] = stChild
		case *fullNode:
			pnode.Children[key[0]] = child
			stParent.children[key[0]] = stChild
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", pnode, pnode))
		}
		if len(valnode) > 0 {
			return stRoot, valnode, nil // The whole path is resolved
		}
		key, parent = keyrest, child
		stParent = stChild
	}
}

func finalizeRightSide(rootHash common.Hash, root *StackTrie, key []byte, proofDb ethdb.KeyValueReader, allowNonExistent bool) (*StackTrie, []byte, error) {
	panic("not implemented")
}
