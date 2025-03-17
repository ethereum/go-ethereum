// --- Start fork code ---
package trie

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type Diff struct {
	Key       []byte
	PreValue  []byte
	PostValue []byte
}

var branchIndices = []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}

func CompareTrie(left, right common.Hash, proofDB ethdb.KeyValueReader) ([]*Diff, error) {
	resolveNode := func(hash common.Hash) (node, error) {
		buf, _ := proofDB.Get(hash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
		}
		n, err := decodeNode(hash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %v", err)
		}
		return n, err
	}

	leftNode, err := resolveNode(left)
	if err != nil {
		return nil, err
	}
	rightNode, err := resolveNode(right)
	if err != nil {
		return nil, err
	}

	return compareTrie(leftNode, rightNode, proofDB)
}

func compareTrie(left, right node, proofDB ethdb.KeyValueReader) ([]*Diff, error) {
	// resolveNode retrieves and resolves trie node from merkle proof stream
	resolveNode := func(hash common.Hash) (node, error) {
		buf, _ := proofDB.Get(hash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
		}
		n, err := decodeNode(hash[:], buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %v", err)
		}
		return n, err
	}

	// resolve hash nodes
	if l, ok := left.(hashNode); ok {
		n, err := resolveNode(common.BytesToHash(l))
		if err != nil {
			return nil, err
		}
		return compareTrie(n, right, proofDB)
	}

	// resolve hash nodes
	if r, ok := right.(hashNode); ok {
		n, err := resolveNode(common.BytesToHash(r))
		if err != nil {
			return nil, err
		}
		return compareTrie(left, n, proofDB)
	}

	switch l := left.(type) {
	case nil:
		switch r := right.(type) {
		case nil:
			// both nodes are nil => no diff
			return nil, nil
		case valueNode:
			// left is nil, right is value => right value is a diff
			return []*Diff{{PostValue: r}}, nil
		case *shortNode:
			// left is nil, right is short => look for diffs in the right child
			diffs, err := compareTrie(nil, r.Val, proofDB)
			if err != nil {
				return nil, err
			}
			return extendDiffKeys(r.Key, diffs), nil
		case *fullNode:
			// left is nil, right is full => look for diffs in the right children
			diffs := make([]*Diff, 0)
			for i := 0; i < 15; i++ {
				cldDiffs, err := compareTrie(nil, r.Children[i], proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
			}
			return diffs, nil

		}
	case valueNode:
		switch r := right.(type) {
		case nil:
			// left is value, right is nil => this is a diff
			return []*Diff{{Key: []byte{}, PreValue: l}}, nil
		case valueNode:
			// left is value, right is value => compare the values
			if bytes.Equal(l, r) {
				return nil, nil
			}
			return []*Diff{{Key: []byte{}, PreValue: l, PostValue: r}}, nil
		case *shortNode:
			// left is value, right is short => everything is a diff
			diffs, err := compareTrie(nil, r.Val, proofDB)
			if err != nil {
				return nil, err
			}
			diffs = extendDiffKeys(r.Key, diffs)
			diffs = append(diffs, &Diff{Key: []byte{}, PreValue: l})
			return diffs, nil
		case *fullNode:
			// left is value, right is full => left value is a diff + look for diffs in the right children
			diffs := []*Diff{{PreValue: l}}
			for i := range 15 {
				cldDiffs, err := compareTrie(nil, r.Children[i], proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
			}
			return diffs, nil
		}
	case *shortNode:
		switch r := right.(type) {
		case nil:
			// left is short, right is nil => look for diffs in the left child
			diffs, err := compareTrie(l.Val, nil, proofDB)
			if err != nil {
				return nil, err
			}
			return extendDiffKeys(l.Key, diffs), nil
		case *shortNode:
			// both nodes are short
			if bytes.Equal(l.Key, r.Key) {
				// same key => compare the children
				diffs, err := compareTrie(l.Val, r.Val, proofDB)
				if err != nil {
					return nil, err
				}
				return extendDiffKeys(l.Key, diffs), nil
			}

			if bytes.HasPrefix(l.Key, r.Key) {
				// right is prefix of left
				// compare the right child with the left shortened by the right key
				// Note: the right child could be full and have common values with the left shortened
				diffs, err := compareTrie(&shortNode{Key: l.Key[len(r.Key):], Val: l.Val}, r.Val, proofDB)
				if err != nil {
					return nil, err
				}
				return extendDiffKeys(r.Key, diffs), nil
			}

			if bytes.HasPrefix(r.Key, l.Key) {
				// left is prefix of right
				// same as above
				diffs, err := compareTrie(l.Val, &shortNode{Key: r.Key[len(l.Key):], Val: r.Val}, proofDB)
				if err != nil {
					return nil, err
				}
				return extendDiffKeys(l.Key, diffs), nil
			}

			// left and right are different
			// look for diffs in the left child and the right child
			lDiffs, err := compareTrie(l.Val, nil, proofDB)
			if err != nil {
				return nil, err
			}
			lDiffs = extendDiffKeys(l.Key, lDiffs)

			rDiffs, err := compareTrie(nil, r.Val, proofDB)
			if err != nil {
				return nil, err
			}
			rDiffs = extendDiffKeys(r.Key, rDiffs)

			return append(lDiffs, rDiffs...), nil
		case *fullNode:
			// left is short, right is full
			diffs := make([]*Diff, 0)
			if len(l.Key) == 0 { // this should never happen when comparing two valid trie roots
				// left is empty key => look for diffs in the left child and the right node
				lDiffs, err := compareTrie(l.Val, nil, proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, lDiffs...)

				rDiffs, err := compareTrie(nil, r, proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, rDiffs...)

				return diffs, nil
			}

			for i := range 15 {
				if l.Key[0] == branchIndices[i] {
					// left key matches the branch index => compare the left node shortened by the index with the right child
					cldDiffs, err := compareTrie(&shortNode{Key: l.Key[1:], Val: l.Val}, r.Children[i], proofDB)
					if err != nil {
						return nil, err
					}
					diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
				} else {
					// left key does not match the branch index => look for diffs in the right child
					cldDiffs, err := compareTrie(nil, r.Children[i], proofDB)
					if err != nil {
						return nil, err
					}
					diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
				}
			}

			return diffs, nil
		}
	case *fullNode:
		switch r := right.(type) {
		case nil:
			// left is full, right is nil => look for diffs in the left children
			diffs := make([]*Diff, 0)
			for i := range 15 {
				cldDiffs, err := compareTrie(l.Children[i], nil, proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
			}
			return diffs, nil
		case *shortNode:
			// left is full, right is short
			diffs := make([]*Diff, 0)
			if len(r.Key) == 0 { // this should never happen when comparing two valid trie roots
				// right is empty key => look for diffs in the left children and the right node
				lDiffs, err := compareTrie(l, nil, proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, lDiffs...)

				rDiffs, err := compareTrie(nil, r.Val, proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, rDiffs...)

				return diffs, nil
			}

			for i := range 15 {
				if r.Key[0] == branchIndices[i] {
					// right key matches the branch index => compare the left child with the right child shortened by the index
					cldDiffs, err := compareTrie(l.Children[i], &shortNode{Key: r.Key[1:], Val: r.Val}, proofDB)
					if err != nil {
						return nil, err
					}
					diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
				} else {
					// right key does not match the branch index => look for diffs in the left child
					cldDiffs, err := compareTrie(l.Children[i], nil, proofDB)
					if err != nil {
						return nil, err
					}
					diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
				}
			}

			return diffs, nil
		case *fullNode:
			// both nodes are full
			diffs := make([]*Diff, 0)
			for i := range 15 {
				cldDiffs, err := compareTrie(l.Children[i], r.Children[i], proofDB)
				if err != nil {
					return nil, err
				}
				diffs = append(diffs, extendDiffKeys([]byte{branchIndices[i]}, cldDiffs)...)
			}
			return diffs, nil
		}
	}

	return nil, nil
}

func extendDiffKeys(prefix []byte, diffs []*Diff) []*Diff {
	for _, d := range diffs {
		d.Key = append(prefix, d.Key...)
	}
	return diffs
}

// --- End fork code ---
