// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package trie implements Merkle Patricia Tries.
package trie

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)
)

// LeafCallback is a callback type invoked when a trie operation reaches a leaf
// node.
//
// The paths is a path tuple identifying a particular trie node either in a single
// trie (account) or a layered trie (account -> storage). Each path in the tuple
// is in the raw format(32 bytes).
//
// The hexpath is a composite hexary path identifying the trie node. All the key
// bytes are converted to the hexary nibbles and composited with the parent path
// if the trie node is in a layered trie.
//
// It's used by state sync and commit to allow handling external references
// between account and storage tries. And also it's used in the state healing
// for extracting the raw states(leaf nodes) with corresponding paths.
type LeafCallback func(paths [][]byte, hexpath []byte, leaf []byte, parent common.Hash) error

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use New to create a trie that sits on top of a database.
//
// Trie is not safe for concurrent use.
type Trie struct {
	db   *Database
	root node
	// Keep track of the number leafs which have been inserted since the last
	// hashing operation. This number will not directly map to the number of
	// actually unhashed nodes
	unhashed int
}

// newFlag returns the cache flag value for a newly created node.
func (t *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true}
}

// New creates a trie with an existing root node from db.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty and does not require a database. Otherwise,
// New will panic if db is nil and returns a MissingNodeError if root does
// not exist in the database. Accessing the trie loads nodes from db on demand.
func New(root common.Hash, db *Database) (*Trie, error) {
	if db == nil {
		panic("trie.New called without a database")
	}
	trie := &Trie{
		db: db,
	}
	if root != (common.Hash{}) && root != emptyRoot {
		rootnode, err := trie.resolveHash(root[:], nil)
		if err != nil {
			return nil, err
		}
		trie.root = rootnode
	}
	return trie, nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration starts at
// the key after the given start key.
func (t *Trie) NodeIterator(start []byte) NodeIterator {
	return newNodeIterator(t, start)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Trie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGet(key []byte) ([]byte, error) {
	value, newroot, didResolve, err := t.tryGet(t.root, keybytesToHex(key), 0)
	if err == nil && didResolve {
		t.root = newroot
	}
	return value, err
}

func (t *Trie) tryGet(origNode node, key []byte, pos int) (value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key)-pos < len(n.Key) || !bytes.Equal(n.Key, key[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n, false, nil
		}
		value, newnode, didResolve, err = t.tryGet(n.Val, key, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newnode, didResolve, err = t.tryGet(n.Children[key[pos]], key, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.Children[key[pos]] = newnode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := t.resolveHash(n, key[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newnode, _, err := t.tryGet(child, key, pos)
		return value, newnode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *Trie) TryGetNode(path []byte) ([]byte, int, error) {
	item, newroot, resolved, err := t.tryGetNode(t.root, compactToHex(path), 0)
	if err != nil {
		return nil, resolved, err
	}
	if resolved > 0 {
		t.root = newroot
	}
	if item == nil {
		return nil, resolved, nil
	}
	return item, resolved, err
}

func (t *Trie) tryGetNode(origNode node, path []byte, pos int) (item []byte, newnode node, resolved int, err error) {
	// If we reached the requested path, return the current node
	if pos >= len(path) {
		// Although we most probably have the original node expanded, encoding
		// that into consensus form can be nasty (needs to cascade down) and
		// time consuming. Instead, just pull the hash up from disk directly.
		var hash hashNode
		if node, ok := origNode.(hashNode); ok {
			hash = node
		} else {
			hash, _ = origNode.cache()
		}
		if hash == nil {
			return nil, origNode, 0, errors.New("non-consensus node")
		}
		blob, err := t.db.Node(common.BytesToHash(hash))
		return blob, origNode, 1, err
	}
	// Path still needs to be traversed, descend into children
	switch n := (origNode).(type) {
	case nil:
		// Non-existent path requested, abort
		return nil, nil, 0, nil

	case valueNode:
		// Path prematurely ended, abort
		return nil, nil, 0, nil

	case *shortNode:
		if len(path)-pos < len(n.Key) || !bytes.Equal(n.Key, path[pos:pos+len(n.Key)]) {
			// Path branches off from short node
			return nil, n, 0, nil
		}
		item, newnode, resolved, err = t.tryGetNode(n.Val, path, pos+len(n.Key))
		if err == nil && resolved > 0 {
			n = n.copy()
			n.Val = newnode
		}
		return item, n, resolved, err

	case *fullNode:
		item, newnode, resolved, err = t.tryGetNode(n.Children[path[pos]], path, pos+1)
		if err == nil && resolved > 0 {
			n = n.copy()
			n.Children[path[pos]] = newnode
		}
		return item, n, resolved, err

	case hashNode:
		child, err := t.resolveHash(n, path[:pos])
		if err != nil {
			return nil, n, 1, err
		}
		item, newnode, resolved, err := t.tryGetNode(child, path, pos)
		return item, newnode, resolved + 1, err

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *Trie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryUpdate(key, value []byte) error {
	t.unhashed++
	k := keybytesToHex(key)
	if len(value) != 0 {
		_, n, err := t.insert(t.root, nil, k, valueNode(value))
		if err != nil {
			return err
		}
		t.root = n
	} else {
		_, n, err := t.delete(t.root, nil, k)
		if err != nil {
			return err
		}
		t.root = n
	}
	return nil
}

func (t *Trie) insert(n node, prefix, key []byte, value node) (bool, node, error) {
	if len(key) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchlen == len(n.Key) {
			dirty, nn, err := t.insert(n.Val, append(prefix, key[:matchlen]...), key[matchlen:], value)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, t.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: t.newFlag()}
		var err error
		_, branch.Children[n.Key[matchlen]], err = t.insert(nil, append(prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val)
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key[matchlen]], err = t.insert(nil, append(prefix, key[:matchlen+1]...), key[matchlen+1:], value)
		if err != nil {
			return false, nil, err
		}
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return true, branch, nil
		}
		// Otherwise, replace it with a short node leading up to the branch.
		return true, &shortNode{key[:matchlen], branch, t.newFlag()}, nil

	case *fullNode:
		dirty, nn, err := t.insert(n.Children[key[0]], append(prefix, key[0]), key[1:], value)
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn
		return true, n, nil

	case nil:
		return true, &shortNode{key, value, t.newFlag()}, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.insert(rn, prefix, key, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryDelete(key []byte) error {
	t.unhashed++
	k := keybytesToHex(key)
	_, n, err := t.delete(t.root, nil, k)
	if err != nil {
		return err
	}
	t.root = n
	return nil
}

// delete returns the new root of the trie with key deleted.
// It reduces the trie to minimal form by simplifying
// nodes on the way up after deleting recursively.
func (t *Trie) delete(n node, prefix, key []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen < len(n.Key) {
			return false, n, nil // don't replace n on mismatch
		}
		if matchlen == len(key) {
			return true, nil, nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		dirty, child, err := t.delete(n.Val, append(prefix, key[:len(n.Key)]...), key[len(n.Key):])
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, t.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, t.newFlag()}, nil
		}

	case *fullNode:
		dirty, nn, err := t.delete(n.Children[key[0]], append(prefix, key[0]), key[1:])
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = t.newFlag()
		n.Children[key[0]] = nn

		// Check how many non-nil entries are left after deleting and
		// reduce the full node to a short node if only one entry is
		// left. Since n must've contained at least two children
		// before deletion (otherwise it would not be a full node) n
		// can never be reduced to nil.
		//
		// When the loop is done, pos contains the index of the single
		// value that is left in n or -2 if n contains at least two
		// values.
		pos := -1
		for i, cld := range &n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				// If the remaining entry is a short node, it replaces
				// n and its key gets the missing nibble tacked to the
				// front. This avoids creating an invalid
				// shortNode{..., shortNode{...}}.  Since the entry
				// might not be loaded yet, resolve it just for this
				// check.
				cnode, err := t.resolve(n.Children[pos], prefix)
				if err != nil {
					return false, nil, err
				}
				if cnode, ok := cnode.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return true, &shortNode{k, cnode.Val, t.newFlag()}, nil
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], t.newFlag()}, nil
		}
		// n still contains at least two values and cannot be reduced.
		return true, n, nil

	case valueNode:
		return true, nil, nil

	case nil:
		return false, nil, nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := t.resolveHash(n, prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := t.delete(rn, prefix, key)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil

	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key))
	}
}

func concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

func (t *Trie) resolve(n node, prefix []byte) (node, error) {
	if n, ok := n.(hashNode); ok {
		return t.resolveHash(n, prefix)
	}
	return n, nil
}

func (t *Trie) resolveHash(n hashNode, prefix []byte) (node, error) {
	hash := common.BytesToHash(n)
	if node := t.db.node(hash); node != nil {
		return node, nil
	}
	return nil, &MissingNodeError{NodeHash: hash, Path: prefix}
}

// Hash returns the root hash of the trie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *Trie) Hash() common.Hash {
	hash, cached, _ := t.hashRoot()
	t.root = cached
	return common.BytesToHash(hash.(hashNode))
}

// Commit writes all nodes to the trie's memory database, tracking the internal
// and external (for account tries) references.
func (t *Trie) Commit(onleaf LeafCallback) (root common.Hash, err error) {
	if t.db == nil {
		panic("commit called on trie with nil database")
	}
	if t.root == nil {
		return emptyRoot, nil
	}
	// Derive the hash for all dirty nodes first. We hold the assumption
	// in the following procedure that all nodes are hashed.
	rootHash := t.Hash()
	h := newCommitter()
	defer returnCommitterToPool(h)

	// Do a quick check if we really need to commit, before we spin
	// up goroutines. This can happen e.g. if we load a trie for reading storage
	// values, but don't write to it.
	if _, dirty := t.root.cache(); !dirty {
		return rootHash, nil
	}
	var wg sync.WaitGroup
	if onleaf != nil {
		h.onleaf = onleaf
		h.leafCh = make(chan *leaf, leafChanSize)
		wg.Add(1)
		go func() {
			defer wg.Done()
			h.commitLoop(t.db)
		}()
	}
	var newRoot hashNode
	newRoot, err = h.Commit(t.root, t.db)
	if onleaf != nil {
		// The leafch is created in newCommitter if there was an onleaf callback
		// provided. The commitLoop only _reads_ from it, and the commit
		// operation was the sole writer. Therefore, it's safe to close this
		// channel here.
		close(h.leafCh)
		wg.Wait()
	}
	if err != nil {
		return common.Hash{}, err
	}
	t.root = newRoot
	return rootHash, nil
}

// hashRoot calculates the root hash of the given trie
func (t *Trie) hashRoot() (node, node, error) {
	if t.root == nil {
		return hashNode(emptyRoot.Bytes()), nil, nil
	}
	// If the number of changes is below 100, we let one thread handle it
	h := newHasher(t.unhashed >= 100)
	defer returnHasherToPool(h)
	hashed, cached := h.hash(t.root, true)
	t.unhashed = 0
	return hashed, cached, nil
}

// Reset drops the referenced root node and cleans all internal state.
func (t *Trie) Reset() {
	t.root = nil
	t.unhashed = 0
}

// print trie nodes details in human readable form (jmlee)
func (t *Trie) Print() {
	fmt.Println(t.root.toString("", t.db))
}

// get trie's db size (jmlee)
func (t *Trie) Size() common.StorageSize {
	size, _ := t.db.Size()
	return size
}

// make empty trie (jmlee)
func NewEmpty() *Trie {
	trie, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	return trie
}

func (t *Trie) MyCommit() {
	// triedb.Commit(root, false, nil)
	t.db.Commit(t.Hash(), false, nil)
}

// get last key among leaf nodes (i.e., right-most key value) (jmlee)
func (t *Trie) GetLastKey() *big.Int {
	lastKey := t.getLastKey(t.root, nil)
	// fmt.Println("lastKey:", lastKey)
	return lastKey
}

// get last key among leaf nodes (i.e., right-most key value) (jmlee)
func (t *Trie) getLastKey(origNode node, lastKey []byte) *big.Int {
	switch n := (origNode).(type) {
	case nil:
		return big.NewInt(0)
	case valueNode:
		hexToInt := new(big.Int)
		hexToInt.SetString(common.BytesToHash(hexToKeybytes(lastKey)).Hex()[2:], 16)
		return hexToInt
	case *shortNode:
		lastKey = append(lastKey, n.Key...)
		// fmt.Println("at getLastKey -> lastKey: ", lastKey, "/ appended key:", n.Key, " (short node)")
		return t.getLastKey(n.Val, lastKey)
	case *fullNode:
		last := 0
		for i, node := range &n.Children {
			if node != nil {
				last = i
			}
		}
		lastByte := common.HexToHash("0x" + indices[last])
		lastKey = append(lastKey, lastByte[len(lastByte)-1])
		// fmt.Println("at getLastKey -> lastKey: ", indices[last], "/ appended key:", indices[last], " (full node)")
		return t.getLastKey(n.Children[last], lastKey)
	case hashNode:
		child, err := t.resolveHash(n, nil)
		if err != nil {
			lastKey = nil
			return big.NewInt(0)
		}
		return t.getLastKey(child, lastKey)
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

// Trie size inspection from nakamoto.snu.ac.kr(jhkim)

// trie inspecting results (jhkim)
type TrieInspectResult struct {
	Count                   int // number of calling function TriInspectNode
	TrieSize                int // bytes
	LeafNodeNum             int // # of leaf nodes in the trie (if this trie is state trie, then = EOANum + CANum)
	LeafNodeSize            int
	EOANum                  int // # of Externally Owned Accounts in the trie
	CANum                   int // # of Contract Accounts in the trie
	FullNodeNum             int // # of full node in the trie
	FullNodeSize            int
	ShortNodeNum            int // # of short node in the trie
	ShortNodeSize           int
	IntermediateNodeNum     int // # of short/full node in the trie
	StorageTrieNum          int // # of non-empty storage tries (for state trie inspection)
	StorageTrieSizeSum      int // total size of storage tries (for state trie inspection)
	StorageTrieLeafNodeNum  int // # of nodes in all storage trie
	StorageTrieFullNodeNum  int
	StorageTrieShortNodeNum int
	ErrorNum                int // # of error occurence while inspecting the trie
	StateTrieFullNodeDepth  [20]int
	StateTrieShortNodeDepth [20]int
	StateTrieLeafNodeDepth  [20]int
}

func (tir *TrieInspectResult) PrintTrieInspectResult(blockNumber uint64, elapsedTime int) {
	f1, err := os.Create("/home/jhkim/go/src/github.com/ethereum/result_" + strconv.FormatUint(blockNumber, 10) + ".txt")
	if err != nil {
		fmt.Printf("Cannot create result file.\n")
		os.Exit(1)
	}
	defer f1.Close()
	fmt.Fprintln(f1, "trie inspect result at block", blockNumber, "with", maxGoroutine, "goroutines (it took", elapsedTime, "seconds)")
	fmt.Fprintln(f1, "  total trie size:", tir.TrieSize, "bytes (about", tir.TrieSize/1000000, "MB)")
	fmt.Fprintln(f1, "  # of full nodes:", tir.FullNodeNum)
	fmt.Fprintln(f1, "  total size of full nodes:", tir.FullNodeSize)
	fmt.Fprintln(f1, "  # of short nodes:", tir.ShortNodeNum)
	fmt.Fprintln(f1, "  total size of short nodes:", tir.ShortNodeSize)
	fmt.Fprintln(f1, "  # of intermediate nodes:", tir.IntermediateNodeNum)
	fmt.Fprintln(f1, "  # of leaf nodes:", tir.LeafNodeNum, "( EOA:", tir.EOANum, "/ CA:", tir.CANum, ")")
	fmt.Fprintln(f1, "  total size of leaf nodes:", tir.LeafNodeSize)
	fmt.Fprintln(f1, "  depth distribution of Full nodes:", tir.StateTrieFullNodeDepth)
	fmt.Fprintln(f1, "  depth distribution of Short nodes:", tir.StateTrieShortNodeDepth)
	fmt.Fprintln(f1, "")
	fmt.Fprintln(f1, "  # of non-empty storage tries:", tir.StorageTrieNum, "(", tir.StorageTrieSizeSum, "bytes =", tir.StorageTrieSizeSum/1000000, "MB )")
	fmt.Fprintln(f1, "  # of Full nodes of storage tries:", tir.StorageTrieFullNodeNum)
	fmt.Fprintln(f1, "  # of short nodes of storage tries:", tir.StorageTrieShortNodeNum)
	fmt.Fprintln(f1, "  # of leaf nodes of storage tries:", tir.StorageTrieLeafNodeNum)
	fmt.Fprintln(f1, "  # of errors:", tir.ErrorNum)

	fmt.Println("\n\n\ntrie inspect result at block", blockNumber, "with", maxGoroutine, "goroutines(it took", elapsedTime, "seconds)")
	fmt.Println("  total trie size:", tir.TrieSize, "bytes (about", tir.TrieSize/1000000, "MB)")
	fmt.Println("  # of full nodes:", tir.FullNodeNum)
	fmt.Println("  total size of full nodes:", tir.FullNodeSize)
	fmt.Println("  # of short nodes:", tir.ShortNodeNum)
	fmt.Println("  total size of short nodes:", tir.ShortNodeSize)
	fmt.Println("  # of intermediate nodes:", tir.IntermediateNodeNum)
	fmt.Println("  # of leaf nodes:", tir.LeafNodeNum, "( EOA:", tir.EOANum, "/ CA:", tir.CANum, ")")
	fmt.Println("  total size of leaf nodes:", tir.LeafNodeSize)
	fmt.Println("  depth distribution of Full nodes:", tir.StateTrieFullNodeDepth)
	fmt.Println("  depth distribution of Short nodes:", tir.StateTrieShortNodeDepth)
	fmt.Println("")
	fmt.Println("  # of non-empty storage tries:", tir.StorageTrieNum, "(", tir.StorageTrieSizeSum, "bytes =", tir.StorageTrieSizeSum/1000000, "MB )")
	fmt.Println("  # of Full nodes of storage tries:", tir.StorageTrieFullNodeNum)
	fmt.Println("  # of short nodes of storage tries:", tir.StorageTrieShortNodeNum)
	fmt.Println("  # of leaf nodes of storage tries:", tir.StorageTrieLeafNodeNum)
	fmt.Println("  # of errors:", tir.ErrorNum) // this should be 0, of course

}

// get shortnode's size (for debugging)
func getShortnodeSize(n shortNode) int {
	h := newHasher(false)
	defer returnHasherToPool(h)
	collapsed, _ := h.hashShortNodeChildren(&n)
	h.tmp.Reset()
	if err := rlp.Encode(&h.tmp, collapsed); err != nil {
		panic("encode error: " + err.Error())
	}
	return len(h.tmp)
}

// get fullnode's size (for debugging)
func getFullnodeSize(n fullNode) int {
	h := newHasher(false)
	defer returnHasherToPool(h)
	collapsed, _ := h.hashFullNodeChildren(&n)
	h.tmp.Reset()
	if err := rlp.Encode(&h.tmp, collapsed); err != nil {
		panic("encode error: " + err.Error())
	}
	return len(h.tmp)
}

var wg sync.WaitGroup
var isFirst = true

// inspect the trie
func (t *Trie) InspectTrie() TrieInspectResult {
	if isFirst {
		// fmt.Println("First call: InspectTrie function. This should be printed only once")
		isFirst = false
		debug.SetMaxThreads(15000) // default MaxThread is 10000

		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	debug.FreeOSMemory()
	var tir TrieInspectResult
	t.inspectTrieNodes(t.root, &tir, &wg, 0, "state") // Inspect Start
	wg.Wait()
	return tir
}

func (t *Trie) InspectStorageTrie() TrieInspectResult {

	var tir TrieInspectResult
	t.inspectTrieNodes(t.root, &tir, &wg, 0, "storage")
	return tir
}

var cnt = 0
var rwMutex = new(sync.RWMutex)
var maxGoroutine = 10000

func (t *Trie) inspectTrieNodes(n node, tir *TrieInspectResult, wg *sync.WaitGroup, depth int, trie string) {

	cnt += 1
	if cnt%100000 == 0 && trie == "state" {
		cnt = 0
		fmt.Println("  intermediate result -> trie size:", tir.TrieSize/1000000, "MB / goroutines", runtime.NumGoroutine(), "/ EOA:", tir.EOANum, "/ CA:", tir.CANum, "/ inter nodes:", tir.IntermediateNodeNum, "/ err:", tir.ErrorNum)

	}

	switch n := n.(type) {
	case *shortNode:
		hn, _ := n.cache()
		hash := common.BytesToHash(hn)
		if hn == nil { // storage trie case
			// this node is smaller than 32 bytes, cause there is no cached hash
			// "Nodes smaller than 32 bytes are stored inside their parent"
			if getShortnodeSize(*n) >= 32 {
				// ERROR: this must not be printed, this is big error
				fmt.Println("ERROR: this shortnode is larger than 32 bytes, but has no cache()")
				os.Exit(1)
			}
			rwMutex.Lock()
			tir.LeafNodeNum++
			tir.ShortNodeNum++
			rwMutex.Unlock()
			return
		}
		nodeBytes, err := t.db.Node(hash) // DB acceess
		if err != nil {
			// in normal case (ex. archive node), it will not come in here
			fmt.Println("ERROR: short node not found -> node hash:", hash.Hex())
			os.Exit(1)
		}
		increaseSize(len(nodeBytes), "short", tir, depth) // increase tir
		t.inspectTrieNodes(n.Val, tir, wg, depth+1, trie) // go child node

	case *fullNode:
		hn, _ := n.cache()
		nodeBytes, err := t.db.Node(common.BytesToHash(hn))
		if err != nil {
			// in normal case (ex. archive node), it will not come in here
			fmt.Println("ERROR: full node not found -> node hash:", common.BytesToHash(hn).Hex())
			os.Exit(1)
		}
		increaseSize(len(nodeBytes), "full", tir, depth) // increase tir
		gortn := runtime.NumGoroutine()

		// // vanilla version
		// for _, child := range &n.Children {
		// 	if child != nil {
		// 		t.inspectTrieNodes(child, tir, wg, depth+1, trie)
		// 	}
		// }

		// goroutine version
		if gortn < maxGoroutine && depth < 6 { // if current number of goroutines exceed max goroutine number
			for _, child := range &n.Children {
				if child != nil {
					wg.Add(1)
					go func(child node, tir *TrieInspectResult, wg *sync.WaitGroup, depth int, trie string) {
						defer wg.Done()
						t.inspectTrieNodes(child, tir, wg, depth+1, trie)
					}(child, tir, wg, depth, trie)
				}
			}
		} else {
			for _, child := range &n.Children {
				if child != nil {
					t.inspectTrieNodes(child, tir, wg, depth+1, trie)
				}
			}
		}

	case hashNode:
		hash := common.BytesToHash([]byte(n))
		resolvedNode := t.db.node(hash) // error
		if resolvedNode != nil {
			t.inspectTrieNodes(resolvedNode, tir, wg, depth, trie)
		} else {
			// in normal case (ex. archive node), it will not come in here
			fmt.Println("ERROR: cannot resolve hash node -> node hash:", hash.Hex())
			os.Exit(1)
		}

	case valueNode:
		// Value nodes don't have children so they're left as were
		// fmt.Println("this node is value node (size:", len(n), "bytes)")
		increaseSize(len(n), "value", tir, depth)
		// value node has account info, decode it
		var acc Account
		if err := rlp.DecodeBytes(n, &acc); err != nil {
			// if this leaf node is from state trie, this decoding will not fail
			// but if this leaf node is from storage trie, this decoding will fail, but not error
			// so I just not add error count

			// log.Error("Failed to decode state object", "err", err)
			// tir.ErrorNum += 1
		} else {
			// check if account has empty codeHash value or not
			codeHash := common.Bytes2Hex(acc.CodeHash)
			if codeHash == "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470" { // empty code hash
				rwMutex.Lock()
				tir.EOANum += 1
				rwMutex.Unlock()
			} else {
				rwMutex.Lock()
				tir.CANum += 1
				rwMutex.Unlock()

				// inspect CA's storage trie (if it is not empty trie)
				if acc.Root.Hex() != "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421" { // empty root hash
					storageTrie, err := NewSecure(acc.Root, t.db) // storage trie is secure trie
					if err != nil {
						fmt.Println("ERROR: cannot find the storage trie")
						rwMutex.Lock()
						tir.ErrorNum += 1
						rwMutex.Unlock()
					} else {
						stateTrieRootHash := storageTrie.Hash()
						if acc.Root.Hex() != stateTrieRootHash.Hex() {
							fmt.Println("maybe this is problem")
							fmt.Println("saved storage root:", acc.Root.Hex(), "/ rehashed storage root:", stateTrieRootHash.Hex())
						}

						// storage trie inspect
						storageTir := storageTrie.InspectStorageTrie()
						// storageTir.PrintTrieInspectResult()
						rwMutex.Lock()
						tir.StorageTrieNum += 1
						tir.StorageTrieSizeSum += storageTir.TrieSize
						tir.ErrorNum += storageTir.ErrorNum

						tir.StorageTrieFullNodeNum += storageTir.FullNodeNum
						tir.StorageTrieShortNodeNum += storageTir.ShortNodeNum
						tir.StorageTrieLeafNodeNum += storageTir.LeafNodeNum
						// if you want to see storage trie's node distribution, add fields of storageTrie
						rwMutex.Unlock()

						if storageTir.ErrorNum != 0 {
							fmt.Print("!!! ERROR: something is wrong while inspecting storage trie ->", storageTir.ErrorNum, "errors\n\n")
							// os.Exit(1)
						}
					}
				}
			}
		}
	default:
		// should not reach here! maybe there is something wrong
		fmt.Println("ERROR: unknown trie node type? node:", n)
		os.Exit(1)
	}
}

func increaseSize(nodeSize int, node string, tir *TrieInspectResult, depth int) {
	rwMutex.Lock()
	tir.TrieSize += nodeSize
	if node == "short" {
		tir.IntermediateNodeNum++
		tir.ShortNodeNum++
		tir.ShortNodeSize += nodeSize
		tir.StateTrieShortNodeDepth[depth]++

	} else if node == "full" {
		tir.IntermediateNodeNum++
		tir.FullNodeNum++
		tir.FullNodeSize += nodeSize
		tir.StateTrieFullNodeDepth[depth]++

	} else if node == "value" {
		tir.LeafNodeNum++
		tir.LeafNodeSize += nodeSize
		tir.StateTrieLeafNodeDepth[depth]++
	} else {
		fmt.Println("wrong node format in increaseSize")
		os.Exit(1)
	}
	rwMutex.Unlock()
}
