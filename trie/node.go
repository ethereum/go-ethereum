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

package trie

import (
	"fmt"
	"io"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

type Node interface {
	fstring(string) string
	Cache() (HashNode, bool)
}

type (
	FullNode struct {
		Children [17]Node // Actual trie Node data to encode/decode (needs custom encoder)
		flags    NodeFlag
	}
	ShortNode struct {
		Key   []byte
		Val   Node
		flags NodeFlag
	}
	HashNode  []byte
	ValueNode []byte
)

// nilValueNode is used when collapsing internal trie nodes for hashing, since
// unset children need to serialize correctly.
var nilValueNode = ValueNode(nil)

// EncodeRLP encodes a full Node into the consensus RLP format.
func (n *FullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]Node

	for i, child := range &n.Children {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

func (n *FullNode) copy() *FullNode   { copy := *n; return &copy }
func (n *ShortNode) copy() *ShortNode { copy := *n; return &copy }

// NodeFlag contains caching-related metadata about a Node.
type NodeFlag struct {
	hash  HashNode // cached hash of the Node (may be nil)
	dirty bool     // whether the Node has changes that must be written to the database
}

func (n *FullNode) Cache() (HashNode, bool)  { return n.flags.hash, n.flags.dirty }
func (n *ShortNode) Cache() (HashNode, bool) { return n.flags.hash, n.flags.dirty }
func (n HashNode) Cache() (HashNode, bool)   { return nil, true }
func (n ValueNode) Cache() (HashNode, bool)  { return nil, true }

// Pretty printing.
func (n *FullNode) String() string  { return n.fstring("") }
func (n *ShortNode) String() string { return n.fstring("") }
func (n HashNode) String() string   { return n.fstring("") }
func (n ValueNode) String() string  { return n.fstring("") }

func (n *FullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range &n.Children {
		if node == nil {
			resp += fmt.Sprintf("%s: <nil> ", indices[i])
		} else {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}
	return resp + fmt.Sprintf("\n%s] ", ind)
}
func (n *ShortNode) fstring(ind string) string {
	return fmt.Sprintf("{%x: %v} ", n.Key, n.Val.fstring(ind+"  "))
}
func (n HashNode) fstring(ind string) string {
	return fmt.Sprintf("<%x> ", []byte(n))
}
func (n ValueNode) fstring(ind string) string {
	return fmt.Sprintf("%x ", []byte(n))
}

func MustDecodeNode(hash, buf []byte) Node {
	n, err := decodeNode(hash, buf)
	if err != nil {
		panic(fmt.Sprintf("Node %x: %v", hash, err))
	}
	return n
}

// decodeNode parses the RLP encoding of a trie Node.
func decodeNode(hash, buf []byte) (Node, error) {
	if len(buf) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	elems, _, err := rlp.SplitList(buf)
	if err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	switch c, _ := rlp.CountValues(elems); c {
	case 2:
		n, err := decodeShort(hash, elems)
		return n, wrapError(err, "short")
	case 17:
		n, err := decodeFull(hash, elems)
		return n, wrapError(err, "full")
	default:
		return nil, fmt.Errorf("invalid number of list elements: %v", c)
	}
}

func decodeShort(hash, elems []byte) (Node, error) {
	kbuf, rest, err := rlp.SplitString(elems)
	if err != nil {
		return nil, err
	}
	flag := NodeFlag{hash: hash}
	key := compactToHex(kbuf)
	if hasTerm(key) {
		// value Node
		val, _, err := rlp.SplitString(rest)
		if err != nil {
			return nil, fmt.Errorf("invalid value Node: %v", err)
		}
		return &ShortNode{key, append(ValueNode{}, val...), flag}, nil
	}
	r, _, err := decodeRef(rest)
	if err != nil {
		return nil, wrapError(err, "val")
	}
	return &ShortNode{key, r, flag}, nil
}

func decodeFull(hash, elems []byte) (*FullNode, error) {
	n := &FullNode{flags: NodeFlag{hash: hash}}
	for i := 0; i < 16; i++ {
		cld, rest, err := decodeRef(elems)
		if err != nil {
			return n, wrapError(err, fmt.Sprintf("[%d]", i))
		}
		n.Children[i], elems = cld, rest
	}
	val, _, err := rlp.SplitString(elems)
	if err != nil {
		return n, err
	}
	if len(val) > 0 {
		n.Children[16] = append(ValueNode{}, val...)
	}
	return n, nil
}

const hashLen = len(common.Hash{})

func decodeRef(buf []byte) (Node, []byte, error) {
	kind, val, rest, err := rlp.Split(buf)
	if err != nil {
		return nil, buf, err
	}
	switch {
	case kind == rlp.List:
		// 'embedded' Node reference. The encoding must be smaller
		// than a hash in order to be valid.
		if size := len(buf) - len(rest); size > hashLen {
			err := fmt.Errorf("oversized embedded Node (size is %d bytes, want size < %d)", size, hashLen)
			return nil, buf, err
		}
		n, err := decodeNode(nil, buf)
		return n, rest, err
	case kind == rlp.String && len(val) == 0:
		// empty Node
		return nil, rest, nil
	case kind == rlp.String && len(val) == 32:
		return append(HashNode{}, val...), rest, nil
	default:
		return nil, nil, fmt.Errorf("invalid RLP string size %d (want 0 or 32)", len(val))
	}
}

// wraps a decoding error with information about the path to the
// invalid child Node (for debugging encoding issues).
type decodeError struct {
	what  error
	stack []string
}

func wrapError(err error, ctx string) error {
	if err == nil {
		return nil
	}
	if decErr, ok := err.(*decodeError); ok {
		decErr.stack = append(decErr.stack, ctx)
		return decErr
	}
	return &decodeError{err, []string{ctx}}
}

func (err *decodeError) Error() string {
	return fmt.Sprintf("%v (decode path: %s)", err.what, strings.Join(err.stack, "<-"))
}
