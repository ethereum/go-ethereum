package trie

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rlp"
)

// Prove constructs a merkle proof for key. The result contains all
// encoded nodes on the path to the value at key. The value itself is
// also included in the last node and can be retrieved by verifying
// the proof.
//
// The returned proof is nil if the trie does not contain a value for key.
// For existing keys, the proof will have at least one element.
func (t *Trie) Prove(key []byte) []rlp.RawValue {
	// Collect all nodes on the path to key.
	key = compactHexDecode(key)
	nodes := []node{}
	tn := t.root
	for len(key) > 0 {
		switch n := tn.(type) {
		case shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				// The trie doesn't contain the key.
				return nil
			}
			tn = n.Val
			key = key[len(n.Key):]
			nodes = append(nodes, n)
		case fullNode:
			tn = n[key[0]]
			key = key[1:]
			nodes = append(nodes, n)
		case nil:
			return nil
		case hashNode:
			var err error
			tn, err = t.resolveHash(n, nil, nil)
			if err != nil {
				if glog.V(logger.Error) {
					glog.Errorf("Unhandled trie error: %v", err)
				}
				return nil
			}
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	if t.hasher == nil {
		t.hasher = newHasher()
	}
	proof := make([]rlp.RawValue, 0, len(nodes))
	for i, n := range nodes {
		// Don't bother checking for errors here since hasher panics
		// if encoding doesn't work and we're not writing to any database.
		n, _ = t.hasher.replaceChildren(n, nil)
		hn, _ := t.hasher.store(n, nil, false)
		if _, ok := hn.(hashNode); ok || i == 0 {
			// If the node's database encoding is a hash (or is the
			// root node), it becomes a proof element.
			enc, _ := rlp.EncodeToBytes(n)
			proof = append(proof, enc)
		}
	}
	return proof
}

// VerifyProof checks merkle proofs. The given proof must contain the
// value for key in a trie with the given root hash. VerifyProof
// returns an error if the proof contains invalid trie nodes or the
// wrong value.
func VerifyProof(rootHash common.Hash, key []byte, proof []rlp.RawValue) (value []byte, err error) {
	key = compactHexDecode(key)
	sha := sha3.NewKeccak256()
	wantHash := rootHash.Bytes()
	for i, buf := range proof {
		sha.Reset()
		sha.Write(buf)
		if !bytes.Equal(sha.Sum(nil), wantHash) {
			return nil, fmt.Errorf("bad proof node %d: hash mismatch", i)
		}
		n, err := decodeNode(buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err)
		}
		keyrest, cld := get(n, key)
		switch cld := cld.(type) {
		case nil:
			return nil, fmt.Errorf("key mismatch at proof node %d", i)
		case hashNode:
			key = keyrest
			wantHash = cld
		case valueNode:
			if i != len(proof)-1 {
				return nil, errors.New("additional nodes at end of proof")
			}
			return cld, nil
		}
	}
	return nil, errors.New("unexpected end of proof")
}

func get(tn node, key []byte) ([]byte, node) {
	for len(key) > 0 {
		switch n := tn.(type) {
		case shortNode:
			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
				return nil, nil
			}
			tn = n.Val
			key = key[len(n.Key):]
		case fullNode:
			tn = n[key[0]]
			key = key[1:]
		case hashNode:
			return key, n
		case nil:
			return key, nil
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
		}
	}
	return nil, tn.(valueNode)
}
