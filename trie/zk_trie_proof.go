package trie

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
	"github.com/scroll-tech/go-ethereum/ethdb"
)

// TODO: remove this hack
var magicHash []byte
var magicSMTBytes []byte

func init() {
	magicSMTBytes = []byte("THIS IS SOME MAGIC BYTES FOR SMT m1rRXgP2xpDI")
	hasher := newHasher(false)
	defer returnHasherToPool(hasher)
	magicHash = hasher.hashData(magicSMTBytes)
}

// Prove constructs a merkle proof for SMT, it respect the protocol used by the ethereum-trie
// but save the node data with a compact form
func (mt *ZkTrieImpl) prove(kHash *zkt.Hash, fromLevel uint, writeNode func(*Node) error) error {

	path := getPath(mt.maxLevels, kHash[:])
	var nodes []*Node
	tn := mt.rootKey
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(tn)
		if err != nil {
			return err
		}

		finished := true
		switch n.Type {
		case NodeTypeEmpty:
		case NodeTypeLeaf:
			// notice even we found a leaf whose entry didn't match the expected k,
			// we still include it as the proof of absence
		case NodeTypeMiddle:
			finished = false
			if path[i] {
				tn = n.ChildR
			} else {
				tn = n.ChildL
			}
		default:
			return ErrInvalidNodeFound
		}

		nodes = append(nodes, n)
		if finished {
			break
		}
	}

	for _, n := range nodes {
		if fromLevel > 0 {
			fromLevel--
			continue
		}

		// TODO: notice here we may have broken some implicit on the proofDb:
		// the key is not kecca(value) and it even can not be derived from
		// the value by any means without a actually decoding
		if err := writeNode(n); err != nil {
			return err
		}
	}

	return nil
}

func (mt *ZkTrieImpl) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {

	kHash, err := zkt.NewHashFromBytes(common.BytesToHash(key).Bytes())
	if err != nil {
		return err
	}

	err = mt.prove(kHash, fromLevel, func(n *Node) error {
		key, err := n.Key()
		if err != nil {
			return err
		}
		return proofDb.Put(key[:], n.Value())
	})

	if err != nil {
		return err
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, magicSMTBytes)
}

func buildZkTrieProof(rootKey *zkt.Hash, k *big.Int, lvl int, getNode func(key *zkt.Hash) (*Node, error)) (*Proof,
	*Node, error) {

	p := &Proof{}
	var siblingKey *zkt.Hash

	kHash := zkt.NewHashFromBigInt(k)
	path := getPath(lvl, kHash[:])

	nextKey := rootKey
	for p.depth = 0; p.depth < uint(lvl); p.depth++ {
		n, err := getNode(nextKey)
		if err != nil {
			return nil, nil, err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return p, n, nil
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.NodeKey[:]) {
				p.Existence = true
				return p, n, nil
			}
			// We found a leaf whose entry didn't match hIndex
			p.NodeAux = &NodeAux{Key: n.NodeKey, Value: n.valueHash}
			return p, n, nil
		case NodeTypeMiddle:
			if path[p.depth] {
				nextKey = n.ChildR
				siblingKey = n.ChildL
			} else {
				nextKey = n.ChildL
				siblingKey = n.ChildR
			}
		default:
			return nil, nil, ErrInvalidNodeFound
		}
		if !bytes.Equal(siblingKey[:], zkt.HashZero[:]) {
			zkt.SetBitBigEndian(p.notempties[:], p.depth)
			p.Siblings = append(p.Siblings, siblingKey)
		}
	}
	return nil, nil, ErrKeyNotFound

}

// DecodeProof try to decode a node bytes, return can be nil for any non-node data (magic code)
func DecodeSMTProof(data []byte) (*Node, error) {

	if bytes.Equal(magicSMTBytes, data) {
		//skip magic bytes node
		return nil, nil
	}

	return NewNodeFromBytes(data)
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProofSMT(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader) (value []byte, err error) {

	h, err := zkt.NewHashFromBytes(rootHash.Bytes())
	if err != nil {
		return nil, err
	}

	word := zkt.NewByte32FromBytesPaddingZero(key)
	k, err := word.Hash()
	if err != nil {
		return nil, err
	}

	proof, n, err := buildZkTrieProof(h, k, len(key)*8, func(key *zkt.Hash) (*Node, error) {
		buf, _ := proofDb.Get(key[:])
		if buf == nil {
			return nil, ErrKeyNotFound
		}
		n, err := NewNodeFromBytes(buf)
		return n, err
	})

	if err != nil {
		// do not contain the key
		return nil, err
	} else if !proof.Existence {
		return nil, nil
	}

	if VerifyProofZkTrie(h, proof, n) {
		return n.Data(), nil
	} else {
		return nil, fmt.Errorf("bad proof node %v", proof)
	}
}
