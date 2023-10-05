package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"golang.org/x/exp/slices"
)

// nodeToStNode converts from `node` to `*stNode`.
func nodeToStNode(n node, key []byte) *stNode {
	st := new(stNode)
	switch n := n.(type) {
	case *shortNode:
		st.typ = extNode
		st.key = append([]byte{}, n.Key...)
	case *fullNode:
		st.typ = branchNode
		idx := int(key[0])
		for i := 0; i < idx; i++ {
			sibling := n.Children[i]
			if sibling == nil {
				continue
			}
			siblingNode := new(stNode)
			siblingNode.typ = hashedNode

			if hash, ok := sibling.(hashNode); ok {
				siblingNode.val = []byte(hash)
			} else {
				// This happens is the sibling is small enough (<32B) to be inlined,
				// in which case the rlp-encoded node is embedded instead of the hash
				short := sibling.(*shortNode)
				short.Key = hexToCompact(short.Key)
				siblingNode.val = nodeToBytes(short)
			}
			st.children[i] = siblingNode
		}
	default:
		panic(fmt.Sprintf("%T", n))
	}
	return st
}

func resolveFromProof(proofDb ethdb.KeyValueReader, hash common.Hash) (node, error) {
	data, _ := proofDb.Get(hash[:])
	if data == nil {
		return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
	}
	n, err := decodeNode(data[:], data)
	if err != nil {
		return nil, fmt.Errorf("bad proof node: %v", err)
	}
	return n, err
}

// newStackTrieFromProof creates a new stacktrie, and initialises it from the given
// proof. It does so by starting at the given root, traverses along the given
// key, and, one by one, converts the nodes into stacktrie elements.
//
// OBS: The resulting stacktrie instance is not guaranteed to be structurally
// identical to a stacktrie which is initialized from scratch by feeding the
// corresponding elements!
// A proof-initialized (PI) stack-trie has some implicit prescient knowledge! Therefore,
// a PI can have already expanded a shortnode into shortnode+fullnode, which a non-PI
// will do only later.
//
// However, the two guarantees that PI gives are:
// - Identical hash,
// - Identical commit-sequence of nodes.
//
// OBS 2: The element in proof should _not_ be added again during value-filling.
// OBS 3: Proofs-of-abscence have not been fully tested. TODO @holiman
func newStackTrieFromProof(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader, writeFn NodeWriteFunc) (*StackTrie, error) {
	var (
		err               error
		child, parent     node
		stChild, stParent *stNode
		keyrest           []byte
		stack             = NewStackTrie(writeFn)
	)
	key = keybytesToHex(key)
	// First we need to resolve the root node from the proof.
	if parent, err = resolveFromProof(proofDb, rootHash); err != nil {
		return nil, err
	}
	var lastResolved common.Hash
	stParent = nodeToStNode(parent, key)
	stack.root = stParent
	// Now we pursue the given key downwards, and populate the stacktrie too
	for {
		keyrest, child = get(parent, key, false)
		switch cld := child.(type) {
		case nil:
			/*
				If the parent is a shortnode, it means that the 'next' key is not
				going in here (because then the parent would be a fullnode). We cannot
				leave the parent shortnode dangling without value: either we
				revert it back to hashed form, or we leave it as an empty node.

				There are a few cases to consider, this shortnode does prove the inexistence
				of the 'origin', it does so by proving that the extension does not
				point to 'origin'. However: it may point to
				1. An existing leaf to the left of the 'origin'
				2. An existing leaft to the right of the 'origin'.

				In the former case, we do not want it here, it does not belong. We need
				to replace it with it's hash.

				In the latter case, we cannot replace it with a hash, because as soon
				as we start feeding the leafs, the first one will hit the "trying to insert into hash" case.
				So instead we must make it an emptyNode.
			*/

			if sn, ok := parent.(*shortNode); ok {
				if bytes.Compare(sn.Key, key) < 0 {
					// This is on the lower side of the proof-border. We must replace
					// this with a hash, not an empty node
					stParent.typ = hashedNode
					stParent.val = lastResolved[:]
				} else {
					stParent.typ = emptyNode
				}
			}
			return stack, nil
		case hashNode:
			lastResolved = common.BytesToHash(cld)
			child, err = resolveFromProof(proofDb, lastResolved)
			if err != nil {
				return nil, err
			}
		case valueNode:
			// The value node goes right into the child
			stParent.val = common.CopyBytes(cld)
			stParent.typ = leafNode
			// remove the terminator
			stParent.key = stParent.key[:len(stParent.key)-1]
			return stack, nil
		case *shortNode:
			// In the case of small leaves, we might end up here with a fullnode
			// whose child is an embedded *shortNode.
		default:
			// we don't expect fullnodes
			panic(fmt.Sprintf("got %T", cld))
		}
		stChild = nodeToStNode(child, keyrest) // convert to stacktrie equivalent
		// Link the parent and child.
		switch pnode := parent.(type) {
		case *shortNode:
			stParent.children[0] = stChild
		case *fullNode:
			stParent.children[key[0]] = stChild
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", pnode, pnode))
		}
		key = keyrest
		parent = child
		stParent = stChild
	}
}

func (st *stNode) dumpTrie(lvl int) {
	var indent []byte
	for i := 0; i < lvl; i++ {
		indent = append(indent, ' ')
	}
	switch st.typ {
	case branchNode:
		fmt.Printf("\n%s FN (key='%#x')", string(indent), st.key)

		for i := 0; i < 16; i++ {
			if st.children[i] == nil {
				continue
			}
			fmt.Printf("\n%s %#x. ", string(indent), i)
			st.children[i].dumpTrie(lvl + 1)
		}
		fmt.Println("")
	case extNode:
		fmt.Printf("%s: sn('%#x')", string(indent), st.key)
		st.children[0].dumpTrie(lvl + 1)
	case leafNode:
		fmt.Printf("%s: leaf('%#x'): %x ", string(indent), st.key, st.val)
	case hashedNode:
		fmt.Printf("hash: %#x %x", st.val, st.key)
	default:
		fmt.Printf("Foo: %d ? ", st.typ)
	}
}

type hashPath struct {
	path []byte
	hash []byte
}

// iterateProof iterates through a proof, starting at the root given by rootHash, and follows the path.
// Along the way, the hashes/paths are collected and delivered.
// If 'ascending' is true, the paths will be on the left side of the proof going down,
// If 'ascending' is false, the paths will be on the right side of the proof going up.
func iterateProof(rootHash common.Hash, path []byte, ascending bool, proof ethdb.KeyValueReader) ([]*hashPath, error) {
	path = keybytesToHex(path)
	var (
		position = 0
		n, _     = resolveFromProof(proof, rootHash)
		paths    []*hashPath
	)
	if n == nil {
		return nil, fmt.Errorf("proof node (hash %064x) missing", rootHash)
	}
	for {
		//fmt.Printf("At position %x\n", path[:position])
		switch typ := n.(type) {
		case *shortNode:
			n = typ.Val
			position += len(typ.Key)
		case *fullNode:
			i, delta := 0, 1 // Start at zero, iterate upwards
			if !ascending {
				i, delta = len(typ.Children)-1, -1 // Start at max, iterate down
			}
			for ; byte(i) != path[position]; i += delta {
				if typ.Children[i] == nil {
					continue
				}
				currentPath := append([]byte{}, path[:position]...)
				currentPath = append(currentPath, byte(i))
				if hn, ok := typ.Children[i].(hashNode); ok {
					paths = append(paths, &hashPath{currentPath, []byte(hn)})
				} else {
					// This happens is the sibling is small enough (<32B) to be inlined,
					// in which case the rlp-encoded node is embedded instead of the hash
					short := typ.Children[i].(*shortNode)
					short.Key = hexToCompact(short.Key)
					data := nodeToBytes(short)
					paths = append(paths, &hashPath{currentPath, data})
				}
				//fmt.Printf("%d. node at (typ %T) : %x\n", i, typ.Children[i], currentPath)
			}
			n = typ.Children[path[position]]
			position++
		default:
			break
		}
		if position == len(path) {
			break
		}
		if hn, ok := n.(hashNode); ok {
			n, _ = resolveFromProof(proof, common.Hash(hn))
		}
		if n == nil {
			// n is nil!
			// At this point, we are following a path of nonexistence.
			// nothing more to iterate here
			// However, while iterating towards a non-present end leaf,
			// we may have encountered an actual leaf here. Must we
			// restore the hash for that one?
			break
		}
	}
	return paths, nil
}

// RootFromLeafs calculates the trie root for the trie built up with the key/values
// given as input.
// This method errors if
// 1. The keys/values are not of equal length
// 2. The keys are not monotonically increasing
func RootFromLeafs(keys [][]byte, values [][]byte) (common.Hash, error) {
	var (
		tr   = NewStackTrie(nil)
		pKey []byte
	)
	for i, key := range keys {
		// Ensure the received batch is monotonic increasing and contains no deletions
		if bytes.Compare(pKey, key) >= 0 {
			return common.Hash{}, errors.New("range is not monotonically increasing")
		}
		if len(values[i]) == 0 {
			return common.Hash{}, errors.New("range contains deletion")
		}
		tr.Update(key, values[i])
		pKey = key
	}
	return tr.Hash(), nil
}

func VerifyRootFromLeafs(root common.Hash, keys [][]byte, values [][]byte) error {
	have, err := RootFromLeafs(keys, values)
	if err != nil {
		return err
	}
	if have != root {
		return fmt.Errorf("want root %x, have %x", root, have)
	}
	return nil
}

// TODO @holiman make this handle proofs-of-nonexistence
func VerifyRangeProofWithStack(rootHash common.Hash, firstKey []byte, keys [][]byte, values [][]byte, proof ethdb.KeyValueReader) (bool, error) {
	if len(keys) != len(values) {
		return false, fmt.Errorf("inconsistent proof data, keys: %d, values: %d", len(keys), len(values))
	}
	// Special case, there is no edge proof at all. The given range is expected
	// to be the whole leaf-set in the trie.
	if proof == nil {
		return false, VerifyRootFromLeafs(rootHash, keys, values)
	}
	// Special case, there is a provided edge proof but zero key/value
	// pairs, ensure there are no more accounts / slots in the trie.
	if len(keys) == 0 {
		root, val, err := proofToPath(rootHash, nil, firstKey, proof, true)
		if err != nil {
			return false, err
		}
		if val != nil || hasRightElement(root, firstKey) {
			return false, errors.New("more entries available")
		}
		return false, nil
	}
	lastKey := keys[len(keys)-1]
	// Special case, there is only one element and two edge keys are same.
	// In this case, we can't construct two edge paths. So handle it here.
	if len(keys) == 1 && bytes.Equal(firstKey, lastKey) {
		root, val, err := proofToPath(rootHash, nil, firstKey, proof, false)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(firstKey, keys[0]) {
			return false, errors.New("correct proof but invalid key")
		}
		if !bytes.Equal(val, values[0]) {
			return false, errors.New("correct proof but invalid data")
		}
		return hasRightElement(root, firstKey), nil
	}
	// Ok, in all other cases, we require two edge paths available.
	// First check the validity of edge keys.
	if bytes.Compare(firstKey, lastKey) >= 0 {
		return false, errors.New("invalid edge keys")
	}
	// todo(rjl493456442) different length edge keys should be supported
	if len(firstKey) != len(lastKey) {
		return false, errors.New("inconsistent edge keys")
	}
	// Use the proof to initiate a stacktrie along the first path/value
	stTrie, err := newStackTrieFromProof(rootHash, firstKey, proof, nil)
	if err != nil {
		return false, fmt.Errorf("could not initate stacktrie: %v", err)
	}
	// Feed in the values
	if bytes.Compare(firstKey, keys[0]) > 0 {
		return false, errors.New("range is not monotonically increasing")
	}
	var pKey []byte
	for i := 0; i < len(keys); i++ {
		if bytes.Compare(pKey, keys[i]) >= 0 {
			return false, errors.New("range is not monotonically increasing")
		}
		pKey = keys[i]
		if len(values[i]) == 0 {
			return false, errors.New("range contains deletion")
		}
		if err := stTrie.Update(keys[i], values[i]); err != nil {
			return false, err
		}
	}
	if !bytes.Equal(lastKey, keys[len(keys)-1]) {
		// The method we have of inserting right-hand hashes only works
		// if the proof indeed concerns the last element: otherwise it forces
		// restructurings on the trie which result in insertion-into-hash.
		// However, for snap-sync, we do not expect "right side proof-of-inexistence":
		// the rhs proof should prove the last element that was sent along.
		return false, fmt.Errorf("proofs must prove the last item")
	}
	// For the right-hand-side, we need a list of hashes ot inject
	hps, err := iterateProof(rootHash, lastKey, false, proof)
	if err != nil {
		return false, fmt.Errorf("proof iteration failed: %v", err)
	}
	slices.Reverse(hps)
	// Insert into stacktrie
	for _, hp := range hps {
		//fmt.Printf("Inserting at %x: hash %x \n", hp.path, hp.hash)
		if err := stTrie.insert(stTrie.root, hp.path, hp.hash[:], nil, newHashed); err != nil {
			return false, err
		}
	}
	if have := stTrie.Hash(); have != rootHash {
		return false, fmt.Errorf("invalid proof, want hash %x, got %x", rootHash, have)
	}
	// hasRightElement is true if the hashes we inserted are non-0
	return len(hps) > 0, nil
}

// wrarpWriteFunction returns a NodeWriteFunc which filters away writes that are
// on the boundary: parents of first/last.
func wrapWriteFunction(origin, first, last []byte, w NodeWriteFunc) NodeWriteFunc {
	if w == nil {
		return nil
	}
	var originBorder = keybytesToHex(origin)
	var leftBorder = keybytesToHex(first)
	var rightBorder = keybytesToHex(last)
	return func(origin common.Hash, path []byte, hash common.Hash, blob []byte) {
		if bytes.HasPrefix(originBorder, path) {
			//fmt.Printf("path %x  tainted left (parent to %x)\n", path, leftBorder)
			return
		}
		if bytes.HasPrefix(leftBorder, path) {
			//fmt.Printf("path %x  tainted left (parent to %x)\n", path, leftBorder)
			return
		}
		if bytes.HasPrefix(rightBorder, path) {
			//fmt.Printf("path %x  tainted right (parent to %x)\n", path, rightBorder)
			return
		}
		w(origin, path, hash, blob)
	}
}
