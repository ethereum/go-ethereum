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
	"hash"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

const defaultCacheCapacity = 800

var (
	// The global cache stores decoded trie nodes by hash as they get loaded.
	globalCache = newARC(defaultCacheCapacity)
	// This is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

var ErrMissingRoot = errors.New("missing root node")

// Database must be implemented by backing stores for the trie.
type Database interface {
	DatabaseWriter
	// Get returns the value for key from the database.
	Get(key []byte) (value []byte, err error)
}

// DatabaseWriter wraps the Put method of a backing store for the trie.
type DatabaseWriter interface {
	// Put stores the mapping key->value in the database.
	// Implementations must not hold onto the value bytes, the trie
	// will reuse the slice across calls to Put.
	Put(key, value []byte) error
}

// Trie is a Merkle Patricia Trie.
// The zero value is an empty trie with no database.
// Use New to create a trie that sits on top of a database.
//
// Trie is not safe for concurrent use.
type Trie struct {
	root node
	db   Database
	*hasher
}

// New creates a trie with an existing root node from db.
//
// If root is the zero hash or the sha3 hash of an empty string, the
// trie is initially empty and does not require a database. Otherwise,
// New will panics if db is nil or root does not exist in the
// database. Accessing the trie loads nodes from db on demand.
func New(root common.Hash, db Database) (*Trie, error) {
	trie := &Trie{db: db}
	if (root != common.Hash{}) && root != emptyRoot {
		if db == nil {
			panic("trie.New: cannot use existing root without a database")
		}
		if v, _ := trie.db.Get(root[:]); len(v) == 0 {
			return nil, ErrMissingRoot
		}
		trie.root = hashNode(root.Bytes())
	}
	return trie, nil
}

// Iterator returns an iterator over all mappings in the trie.
func (t *Trie) Iterator() *Iterator {
	return NewIterator(t)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Trie) Get(key []byte) []byte {
	key = compactHexDecode(key)
	tn := t.root
	for len(key) > 0 {
		switch n := tn.(type) {
		case shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil
			}
			tn = n.Val
			key = key[len(n.Key):]
		case fullNode:
			tn = n[key[0]]
			key = key[1:]
		case nil:
			return nil
		case hashNode:
			tn = t.resolveHash(n)
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	return tn.(valueNode)
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *Trie) Update(key, value []byte) {
	k := compactHexDecode(key)
	if len(value) != 0 {
		t.root = t.insert(t.root, k, valueNode(value))
	} else {
		t.root = t.delete(t.root, k)
	}
}

func (t *Trie) insert(n node, key []byte, value node) node {
	if len(key) == 0 {
		return value
	}
	switch n := n.(type) {
	case shortNode:
		matchlen := prefixLen(key, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchlen == len(n.Key) {
			return shortNode{n.Key, t.insert(n.Val, key[matchlen:], value)}
		}
		// Otherwise branch out at the index where they differ.
		var branch fullNode
		branch[n.Key[matchlen]] = t.insert(nil, n.Key[matchlen+1:], n.Val)
		branch[key[matchlen]] = t.insert(nil, key[matchlen+1:], value)
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return branch
		}
		// Otherwise, replace it with a short node leading up to the branch.
		return shortNode{key[:matchlen], branch}

	case fullNode:
		n[key[0]] = t.insert(n[key[0]], key[1:], value)
		return n

	case nil:
		return shortNode{key, value}

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		//
		// TODO: track whether insertion changed the value and keep
		// n as a hash node if it didn't.
		return t.insert(t.resolveHash(n), key, value)

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) {
	k := compactHexDecode(key)
	t.root = t.delete(t.root, k)
}

// delete returns the new root of the trie with key deleted.
// It reduces the trie to minimal form by simplifying
// nodes on the way up after deleting recursively.
func (t *Trie) delete(n node, key []byte) node {
	switch n := n.(type) {
	case shortNode:
		matchlen := prefixLen(key, n.Key)
		if matchlen < len(n.Key) {
			return n // don't replace n on mismatch
		}
		if matchlen == len(key) {
			return nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		child := t.delete(n.Val, key[len(n.Key):])
		switch child := child.(type) {
		case shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return shortNode{concat(n.Key, child.Key...), child.Val}
		default:
			return shortNode{n.Key, child}
		}

	case fullNode:
		n[key[0]] = t.delete(n[key[0]], key[1:])
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
		for i, cld := range n {
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
				cnode := t.resolve(n[pos])
				if cnode, ok := cnode.(shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return shortNode{k, cnode.Val}
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return shortNode{[]byte{byte(pos)}, n[pos]}
		}
		// n still contains at least two values and cannot be reduced.
		return n

	case nil:
		return nil

	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		//
		// TODO: track whether deletion actually hit a key and keep
		// n as a hash node if it didn't.
		return t.delete(t.resolveHash(n), key)

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

func (t *Trie) resolve(n node) node {
	if n, ok := n.(hashNode); ok {
		return t.resolveHash(n)
	}
	return n
}

func (t *Trie) resolveHash(n hashNode) node {
	if v, ok := globalCache.Get(n); ok {
		return v
	}
	enc, err := t.db.Get(n)
	if err != nil || enc == nil {
		// TODO: This needs to be improved to properly distinguish errors.
		// Disk I/O errors shouldn't produce nil (and cause a
		// consensus failure or weird crash), but it is unclear how
		// they could be handled because the entire stack above the trie isn't
		// prepared to cope with missing state nodes.
		if glog.V(logger.Error) {
			glog.Errorf("Dangling hash node ref %x: %v", n, err)
		}
		return nil
	}
	dec := mustDecodeNode(n, enc)
	if dec != nil {
		globalCache.Put(n, dec)
	}
	return dec
}

// Root returns the root hash of the trie.
// Deprecated: use Hash instead.
func (t *Trie) Root() []byte { return t.Hash().Bytes() }

// Hash returns the root hash of the trie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *Trie) Hash() common.Hash {
	root, _ := t.hashRoot(nil)
	return common.BytesToHash(root.(hashNode))
}

// Commit writes all nodes to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory.
// Subsequent Get calls will load nodes from the database.
func (t *Trie) Commit() (root common.Hash, err error) {
	if t.db == nil {
		panic("Commit called on trie with nil database")
	}
	return t.CommitTo(t.db)
}

// CommitTo writes all nodes to the given database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will
// load nodes from the trie's database. Calling code must ensure that
// the changes made to db are written back to the trie's attached
// database before using the trie.
func (t *Trie) CommitTo(db DatabaseWriter) (root common.Hash, err error) {
	n, err := t.hashRoot(db)
	if err != nil {
		return (common.Hash{}), err
	}
	t.root = n
	return common.BytesToHash(n.(hashNode)), nil
}

func (t *Trie) hashRoot(db DatabaseWriter) (node, error) {
	if t.root == nil {
		return hashNode(emptyRoot.Bytes()), nil
	}
	if t.hasher == nil {
		t.hasher = newHasher()
	}
	return t.hasher.hash(t.root, db, true)
}

type hasher struct {
	tmp *bytes.Buffer
	sha hash.Hash
}

func newHasher() *hasher {
	return &hasher{tmp: new(bytes.Buffer), sha: sha3.NewKeccak256()}
}

func (h *hasher) hash(n node, db DatabaseWriter, force bool) (node, error) {
	hashed, err := h.replaceChildren(n, db)
	if err != nil {
		return hashNode{}, err
	}
	if n, err = h.store(hashed, db, force); err != nil {
		return hashNode{}, err
	}
	return n, nil
}

// hashChildren replaces child nodes of n with their hashes if the encoded
// size of the child is larger than a hash.
func (h *hasher) replaceChildren(n node, db DatabaseWriter) (node, error) {
	var err error
	switch n := n.(type) {
	case shortNode:
		n.Key = compactEncode(n.Key)
		if _, ok := n.Val.(valueNode); !ok {
			if n.Val, err = h.hash(n.Val, db, false); err != nil {
				return n, err
			}
		}
		if n.Val == nil {
			// Ensure that nil children are encoded as empty strings.
			n.Val = valueNode(nil)
		}
		return n, nil
	case fullNode:
		for i := 0; i < 16; i++ {
			if n[i] != nil {
				if n[i], err = h.hash(n[i], db, false); err != nil {
					return n, err
				}
			} else {
				// Ensure that nil children are encoded as empty strings.
				n[i] = valueNode(nil)
			}
		}
		if n[16] == nil {
			n[16] = valueNode(nil)
		}
		return n, nil
	default:
		return n, nil
	}
}

func (h *hasher) store(n node, db DatabaseWriter, force bool) (node, error) {
	// Don't store hashes or empty nodes.
	if _, isHash := n.(hashNode); n == nil || isHash {
		return n, nil
	}
	h.tmp.Reset()
	if err := rlp.Encode(h.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}
	if h.tmp.Len() < 32 && !force {
		// Nodes smaller than 32 bytes are stored inside their parent.
		return n, nil
	}
	// Larger nodes are replaced by their hash and stored in the database.
	h.sha.Reset()
	h.sha.Write(h.tmp.Bytes())
	key := hashNode(h.sha.Sum(nil))
	if db != nil {
		err := db.Put(key, h.tmp.Bytes())
		return key, err
	}
	return key, nil
}
