package trie

import (
	"bytes"
	"fmt"

	zktrie "github.com/scroll-tech/zktrie/trie"
	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/ethdb"
)

// Pick Node from its hash directly from database, notice it has different
// interface with the function of same name in `trie`
func (t *ZkTrie) TryGetNode(nodeHash *zkt.Hash) (*zktrie.Node, error) {
	if bytes.Equal(nodeHash[:], zkt.HashZero[:]) {
		return zktrie.NewEmptyNode(), nil
	}
	nBytes, err := t.db.Get(nodeHash[:])
	if err == zktrie.ErrKeyNotFound {
		return nil, zktrie.ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}
	return zktrie.NewNodeFromBytes(nBytes)
}

type ProofTracer struct {
	*ZkTrie
	deletionTracer map[zkt.Hash]struct{}
	rawPaths       map[string][]*zktrie.Node
	emptyTermPaths map[string][]*zktrie.Node
}

// NewProofTracer create a proof tracer object
func (t *ZkTrie) NewProofTracer() *ProofTracer {
	return &ProofTracer{
		ZkTrie: t,
		// always consider 0 is "deleted"
		deletionTracer: map[zkt.Hash]struct{}{zkt.HashZero: {}},
		rawPaths:       make(map[string][]*zktrie.Node),
		emptyTermPaths: make(map[string][]*zktrie.Node),
	}
}

// Merge merge the input tracer into current and return current tracer
func (t *ProofTracer) Merge(another *ProofTracer) *ProofTracer {

	// sanity checking
	if !bytes.Equal(t.Hash().Bytes(), another.Hash().Bytes()) {
		panic("can not merge two proof tracer base on different trie")
	}

	for k := range another.deletionTracer {
		t.deletionTracer[k] = struct{}{}
	}

	for k, v := range another.rawPaths {
		t.rawPaths[k] = v
	}

	for k, v := range another.emptyTermPaths {
		t.emptyTermPaths[k] = v
	}

	return t
}

// GetDeletionProofs generate current deletionTracer and collect deletion proofs
// which is possible to be used from all rawPaths, which enabling witness generator
// to predict the final state root after executing any deletion
// along any of the rawpath, no matter of the deletion occurs in any position of the mpt ops
// Note the collected sibling node has no key along with it since witness generator would
// always decode the node for its purpose
func (t *ProofTracer) GetDeletionProofs() ([][]byte, error) {

	retMap := map[zkt.Hash][]byte{}

	// check each path: reversively, skip the final leaf node
	for _, path := range t.rawPaths {

		checkPath := path[:len(path)-1]
		for i := len(checkPath); i > 0; i-- {
			n := checkPath[i-1]
			_, deletedL := t.deletionTracer[*n.ChildL]
			_, deletedR := t.deletionTracer[*n.ChildR]
			if deletedL && deletedR {
				nodeHash, _ := n.NodeHash()
				t.deletionTracer[*nodeHash] = struct{}{}
			} else {
				var siblingHash *zkt.Hash
				if deletedL {
					siblingHash = n.ChildR
				} else if deletedR {
					siblingHash = n.ChildL
				}
				if siblingHash != nil {
					sibling, err := t.TryGetNode(siblingHash)
					if err != nil {
						return nil, err
					}
					if sibling.Type != zktrie.NodeTypeEmpty_New {
						retMap[*siblingHash] = sibling.Value()
					}
				}
				break
			}
		}
	}

	var ret [][]byte
	for _, bt := range retMap {
		ret = append(ret, bt)
	}

	return ret, nil

}

// MarkDeletion mark a key has been involved into deletion
func (t *ProofTracer) MarkDeletion(key []byte) {
	if path, existed := t.emptyTermPaths[string(key)]; existed {
		// copy empty node terminated path for final scanning
		t.rawPaths[string(key)] = path
	} else if path, existed = t.rawPaths[string(key)]; existed {
		// sanity check
		leafNode := path[len(path)-1]

		if leafNode.Type != zktrie.NodeTypeLeaf_New {
			panic("all path recorded in proofTrace should be ended with leafNode")
		}

		nodeHash, _ := leafNode.NodeHash()
		t.deletionTracer[*nodeHash] = struct{}{}
	}
}

// Prove act the same as zktrie.Prove, while also collect the raw path
// for collecting deletion proofs in a post-work
func (t *ProofTracer) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	var mptPath []*zktrie.Node
	err := t.ZkTrie.ProveWithDeletion(key, fromLevel,
		func(n *zktrie.Node) error {
			nodeHash, err := n.NodeHash()
			if err != nil {
				return err
			}

			switch n.Type {
			case zktrie.NodeTypeLeaf_New:
				preImage := t.GetKey(n.NodeKey.Bytes())
				if len(preImage) > 0 {
					n.KeyPreimage = &zkt.Byte32{}
					copy(n.KeyPreimage[:], preImage)
				}
			case zktrie.NodeTypeBranch_0, zktrie.NodeTypeBranch_1,
				zktrie.NodeTypeBranch_2, zktrie.NodeTypeBranch_3:
				mptPath = append(mptPath, n)
			case zktrie.NodeTypeEmpty_New:
				// empty node is considered as "unhit" but it should be also being added
				// into a temporary slot for possibly being marked as deletion later
				mptPath = append(mptPath, n)
				t.emptyTermPaths[string(key)] = mptPath
			default:
				panic(fmt.Errorf("unexpected node type %d", n.Type))
			}

			return proofDb.Put(nodeHash[:], n.Value())
		},
		func(n *zktrie.Node, _ *zktrie.Node) {
			// only "hit" path (i.e. the leaf node corresponding the input key can be found)
			// would be add into tracer
			mptPath = append(mptPath, n)
			t.rawPaths[string(key)] = mptPath
		},
	)
	if err != nil {
		return err
	}
	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, zktrie.ProofMagicBytes())
}
