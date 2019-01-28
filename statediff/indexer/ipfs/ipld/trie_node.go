// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package ipld

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
)

// TrieNode is the general abstraction for
//ethereum IPLD trie nodes.
type TrieNode struct {
	// leaf, extension or branch
	nodeKind string

	// If leaf or extension: [0] is key, [1] is val.
	// If branch: [0] - [16] are children.
	elements []interface{}

	// IPLD block information
	cid     cid.Cid
	rawdata []byte
}

/*
  OUTPUT
*/

type trieNodeLeafDecoder func([]interface{}) ([]interface{}, error)

// decodeTrieNode returns a TrieNode object from an IPLD block's
// cid and rawdata.
func decodeTrieNode(c cid.Cid, b []byte,
	leafDecoder trieNodeLeafDecoder) (*TrieNode, error) {
	var (
		i, decoded, elements []interface{}
		nodeKind             string
		err                  error
	)

	if err = rlp.DecodeBytes(b, &i); err != nil {
		return nil, err
	}

	codec := c.Type()
	switch len(i) {
	case 2:
		nodeKind, decoded, err = decodeCompactKey(i)
		if err != nil {
			return nil, err
		}

		if nodeKind == "extension" {
			elements, err = parseTrieNodeExtension(decoded, codec)
		}
		if nodeKind == "leaf" {
			elements, err = leafDecoder(decoded)
		}
		if nodeKind != "extension" && nodeKind != "leaf" {
			return nil, fmt.Errorf("unexpected nodeKind returned from decoder")
		}
	case 17:
		nodeKind = "branch"
		elements, err = parseTrieNodeBranch(i, codec)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown trie node type")
	}

	return &TrieNode{
		nodeKind: nodeKind,
		elements: elements,
		rawdata:  b,
		cid:      c,
	}, nil
}

// decodeCompactKey takes a compact key, and returns its nodeKind and value.
func decodeCompactKey(i []interface{}) (string, []interface{}, error) {
	first := i[0].([]byte)
	last := i[1].([]byte)

	switch first[0] / 16 {
	case '\x00':
		return "extension", []interface{}{
			nibbleToByte(first)[2:],
			last,
		}, nil
	case '\x01':
		return "extension", []interface{}{
			nibbleToByte(first)[1:],
			last,
		}, nil
	case '\x02':
		return "leaf", []interface{}{
			nibbleToByte(first)[2:],
			last,
		}, nil
	case '\x03':
		return "leaf", []interface{}{
			nibbleToByte(first)[1:],
			last,
		}, nil
	default:
		return "", nil, fmt.Errorf("unknown hex prefix")
	}
}

// parseTrieNodeExtension helper improves readability
func parseTrieNodeExtension(i []interface{}, codec uint64) ([]interface{}, error) {
	return []interface{}{
		i[0].([]byte),
		keccak256ToCid(codec, i[1].([]byte)),
	}, nil
}

// parseTrieNodeBranch helper improves readability
func parseTrieNodeBranch(i []interface{}, codec uint64) ([]interface{}, error) {
	var out []interface{}

	for i, vi := range i {
		v, ok := vi.([]byte)
		// Sometimes this throws "panic: interface conversion: interface {} is []interface {}, not []uint8"
		// Figure out why, and if it is okay to continue
		if !ok {
			return nil, fmt.Errorf("unable to decode branch node entry into []byte at position: %d value: %+v", i, vi)
		}

		switch len(v) {
		case 0:
			out = append(out, nil)
		case 32:
			out = append(out, keccak256ToCid(codec, v))
		default:
			return nil, fmt.Errorf("unrecognized object: %v", v)
		}
	}

	return out, nil
}

/*
  Node INTERFACE
*/

// Resolve resolves a path through this node, stopping at any link boundary
// and returning the object found as well as the remaining path to traverse
func (t *TrieNode) Resolve(p []string) (interface{}, []string, error) {
	switch t.nodeKind {
	case "extension":
		return t.resolveTrieNodeExtension(p)
	case "leaf":
		return t.resolveTrieNodeLeaf(p)
	case "branch":
		return t.resolveTrieNodeBranch(p)
	default:
		return nil, nil, fmt.Errorf("nodeKind case not implemented")
	}
}

// Tree lists all paths within the object under 'path', and up to the given depth.
// To list the entire object (similar to `find .`) pass "" and -1
func (t *TrieNode) Tree(p string, depth int) []string {
	if p != "" || depth == 0 {
		return nil
	}

	var out []string

	switch t.nodeKind {
	case "extension":
		var val string
		for _, e := range t.elements[0].([]byte) {
			val += fmt.Sprintf("%x", e)
		}
		return []string{val}
	case "branch":
		for i, elem := range t.elements {
			if _, ok := elem.(*cid.Cid); ok {
				out = append(out, fmt.Sprintf("%x", i))
			}
		}
		return out

	default:
		return nil
	}
}

// ResolveLink is a helper function that calls resolve and asserts the
// output is a link
func (t *TrieNode) ResolveLink(p []string) (*node.Link, []string, error) {
	obj, rest, err := t.Resolve(p)
	if err != nil {
		return nil, nil, err
	}

	lnk, ok := obj.(*node.Link)
	if !ok {
		return nil, nil, fmt.Errorf("was not a link")
	}

	return lnk, rest, nil
}

// Copy will go away. It is here to comply with the interface.
func (t *TrieNode) Copy() node.Node {
	panic("dont use this yet")
}

// Links is a helper function that returns all links within this object
func (t *TrieNode) Links() []*node.Link {
	var out []*node.Link

	for _, i := range t.elements {
		c, ok := i.(cid.Cid)
		if ok {
			out = append(out, &node.Link{Cid: c})
		}
	}

	return out
}

// Stat will go away. It is here to comply with the interface.
func (t *TrieNode) Stat() (*node.NodeStat, error) {
	return &node.NodeStat{}, nil
}

// Size will go away. It is here to comply with the interface.
func (t *TrieNode) Size() (uint64, error) {
	return 0, nil
}

/*
  TrieNode functions
*/

// MarshalJSON processes the transaction trie into readable JSON format.
func (t *TrieNode) MarshalJSON() ([]byte, error) {
	var out map[string]interface{}

	switch t.nodeKind {
	case "extension":
		fallthrough
	case "leaf":
		var hexPrefix string
		for _, e := range t.elements[0].([]byte) {
			hexPrefix += fmt.Sprintf("%x", e)
		}

		// if we got a byte we need to do this casting otherwise
		// it will be marshaled to a base64 encoded value
		if _, ok := t.elements[1].([]byte); ok {
			var hexVal string
			for _, e := range t.elements[1].([]byte) {
				hexVal += fmt.Sprintf("%x", e)
			}

			t.elements[1] = hexVal
		}

		out = map[string]interface{}{
			"type":    t.nodeKind,
			hexPrefix: t.elements[1],
		}

	case "branch":
		out = map[string]interface{}{
			"type": "branch",
			"0":    t.elements[0],
			"1":    t.elements[1],
			"2":    t.elements[2],
			"3":    t.elements[3],
			"4":    t.elements[4],
			"5":    t.elements[5],
			"6":    t.elements[6],
			"7":    t.elements[7],
			"8":    t.elements[8],
			"9":    t.elements[9],
			"a":    t.elements[10],
			"b":    t.elements[11],
			"c":    t.elements[12],
			"d":    t.elements[13],
			"e":    t.elements[14],
			"f":    t.elements[15],
		}
	default:
		return nil, fmt.Errorf("nodeKind %s not supported", t.nodeKind)
	}

	return json.Marshal(out)
}

// nibbleToByte expands the nibbles of a byte slice into their own bytes.
func nibbleToByte(k []byte) []byte {
	var out []byte

	for _, b := range k {
		out = append(out, b/16)
		out = append(out, b%16)
	}

	return out
}

// Resolve reading conveniences
func (t *TrieNode) resolveTrieNodeExtension(p []string) (interface{}, []string, error) {
	nibbles := t.elements[0].([]byte)
	idx, rest := shiftFromPath(p, len(nibbles))
	if len(idx) < len(nibbles) {
		return nil, nil, fmt.Errorf("not enough nibbles to traverse this extension")
	}

	for _, i := range idx {
		if getHexIndex(string(i)) == -1 {
			return nil, nil, fmt.Errorf("invalid path element")
		}
	}

	for i, n := range nibbles {
		if string(idx[i]) != fmt.Sprintf("%x", n) {
			return nil, nil, fmt.Errorf("no such link in this extension")
		}
	}

	return &node.Link{Cid: t.elements[1].(cid.Cid)}, rest, nil
}

func (t *TrieNode) resolveTrieNodeLeaf(p []string) (interface{}, []string, error) {
	nibbles := t.elements[0].([]byte)

	if len(nibbles) != 0 {
		idx, rest := shiftFromPath(p, len(nibbles))
		if len(idx) < len(nibbles) {
			return nil, nil, fmt.Errorf("not enough nibbles to traverse this leaf")
		}

		for _, i := range idx {
			if getHexIndex(string(i)) == -1 {
				return nil, nil, fmt.Errorf("invalid path element")
			}
		}

		for i, n := range nibbles {
			if string(idx[i]) != fmt.Sprintf("%x", n) {
				return nil, nil, fmt.Errorf("no such link in this extension")
			}
		}

		p = rest
	}

	link, ok := t.elements[1].(node.Node)
	if !ok {
		return nil, nil, fmt.Errorf("leaf children is not an IPLD node")
	}

	return link.Resolve(p)
}

func (t *TrieNode) resolveTrieNodeBranch(p []string) (interface{}, []string, error) {
	idx, rest := shiftFromPath(p, 1)
	hidx := getHexIndex(idx)
	if hidx == -1 {
		return nil, nil, fmt.Errorf("incorrect path")
	}

	child := t.elements[hidx]
	if child != nil {
		return &node.Link{Cid: child.(cid.Cid)}, rest, nil
	}
	return nil, nil, fmt.Errorf("no such link in this branch")
}

// shiftFromPath extracts from a given path (as a slice of strings)
// the given number of elements as a single string, returning whatever
// it has not taken.
//
// Examples:
// ["0", "a", "something"] and 1 -> "0" and ["a", "something"]
// ["ab", "c", "d", "1"] and 2 -> "ab" and ["c", "d", "1"]
// ["abc", "d", "1"] and 2 -> "ab" and ["c", "d", "1"]
func shiftFromPath(p []string, i int) (string, []string) {
	var (
		out  string
		rest []string
	)

	for _, pe := range p {
		re := ""
		for _, c := range pe {
			if len(out) < i {
				out += string(c)
			} else {
				re += string(c)
			}
		}

		if len(out) == i && re != "" {
			rest = append(rest, re)
		}
	}

	return out, rest
}

// getHexIndex returns to you the integer 0 - 15 equivalent to your
// string character if applicable, or -1 otherwise.
func getHexIndex(s string) int {
	if len(s) != 1 {
		return -1
	}

	c := byte(s[0])
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	}

	return -1
}
