// Copyright 2015 The go-ethereum Authors
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

package trie

import (
	"fmt"

	zktrie "github.com/scroll-tech/zktrie/trie"
	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/poseidon"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
)

var magicHash []byte = []byte("THIS IS THE MAGIC INDEX FOR ZKTRIE")

// wrap zktrie for trie interface
type ZkTrie struct {
	*zktrie.ZkTrie
	db *ZktrieDatabase
}

func init() {
	zkt.InitHashScheme(poseidon.HashFixedWithDomain)
}

func sanityCheckByte32Key(b []byte) {
	if len(b) != 32 && len(b) != 20 {
		panic(fmt.Errorf("do not support length except for 120bit and 256bit now. data: %v len: %v", b, len(b)))
	}
}

// NewZkTrie creates a trie
// NewZkTrie bypasses all the buffer mechanism in *Database, it directly uses the
// underlying diskdb
func NewZkTrie(root common.Hash, db *ZktrieDatabase) (*ZkTrie, error) {
	tr, err := zktrie.NewZkTrie(*zkt.NewByte32FromBytes(root.Bytes()), db)
	if err != nil {
		return nil, err
	}
	return &ZkTrie{tr, db}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *ZkTrie) Get(key []byte) []byte {
	sanityCheckByte32Key(key)
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryUpdateAccount will abstract the write of an account to the
// secure trie.
func (t *ZkTrie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	sanityCheckByte32Key(key)
	value, flag := acc.MarshalFields()
	return t.ZkTrie.TryUpdate(key, flag, value)
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *ZkTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// NOTE: value is restricted to length of bytes32.
// we override the underlying zktrie's TryUpdate method
func (t *ZkTrie) TryUpdate(key, value []byte) error {
	sanityCheckByte32Key(key)
	return t.ZkTrie.TryUpdate(key, 1, []zkt.Byte32{*zkt.NewByte32FromBytes(value)})
}

// Delete removes any existing value for key from the trie.
func (t *ZkTrie) Delete(key []byte) {
	sanityCheckByte32Key(key)
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// GetKey returns the preimage of a hashed key that was
// previously used to store a value.
func (t *ZkTrie) GetKey(kHashBytes []byte) []byte {
	// TODO: use a kv cache in memory
	k, err := zkt.NewBigIntFromHashBytes(kHashBytes)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	if t.db.db.preimages != nil {
		return t.db.db.preimages.preimage(common.BytesToHash(k.Bytes()))
	}
	return nil
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *ZkTrie) Commit(LeafCallback) (common.Hash, int, error) {
	// in current implmentation, every update of trie already writes into database
	// so Commmit does nothing
	return t.Hash(), 0, nil
}

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *ZkTrie) Hash() common.Hash {
	var hash common.Hash
	hash.SetBytes(t.ZkTrie.Hash())
	return hash
}

// Copy returns a copy of SecureBinaryTrie.
func (t *ZkTrie) Copy() *ZkTrie {
	return &ZkTrie{t.ZkTrie.Copy(), t.db}
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *ZkTrie) NodeIterator(start []byte) NodeIterator {
	/// FIXME
	panic("not implemented")
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
/*func (t *ZkTrie) hashKey(key []byte) []byte {
	if len(key) != 32 {
		panic("non byte32 input to hashKey")
	}
	low16 := new(big.Int).SetBytes(key[:16])
	high16 := new(big.Int).SetBytes(key[16:])
	hash, err := poseidon.Hash([]*big.Int{low16, high16})
	if err != nil {
		panic(err)
	}
	return hash.Bytes()
}
*/

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *ZkTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	err := t.ZkTrie.Prove(key, fromLevel, func(n *zktrie.Node) error {
		nodeHash, err := n.NodeHash()
		if err != nil {
			return err
		}

		if n.Type == zktrie.NodeTypeLeaf_New {
			preImage := t.GetKey(n.NodeKey.Bytes())
			if len(preImage) > 0 {
				n.KeyPreimage = &zkt.Byte32{}
				copy(n.KeyPreimage[:], preImage)
				//return fmt.Errorf("key preimage not found for [%x] ref %x", n.NodeKey.Bytes(), k.Bytes())
			}
		}
		return proofDb.Put(nodeHash[:], n.Value())
	})
	if err != nil {
		return err
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, zktrie.ProofMagicBytes())
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProofSMT(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader) (value []byte, err error) {

	h := zkt.NewHashFromBytes(rootHash.Bytes())
	k, err := zkt.ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	proof, n, err := zktrie.BuildZkTrieProof(h, k, len(key)*8, func(key *zkt.Hash) (*zktrie.Node, error) {
		buf, _ := proofDb.Get(key[:])
		if buf == nil {
			return nil, zktrie.ErrKeyNotFound
		}
		n, err := zktrie.NewNodeFromBytes(buf)
		return n, err
	})

	if err != nil {
		// do not contain the key
		return nil, err
	} else if !proof.Existence {
		return nil, nil
	}

	if zktrie.VerifyProofZkTrie(h, proof, n) {
		return n.Data(), nil
	} else {
		return nil, fmt.Errorf("bad proof node %v", proof)
	}
}
