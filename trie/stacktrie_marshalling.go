// Copyright 2023 The go-ethereum Authors
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
	"bufio"
	"bytes"
	"encoding"
	"encoding/gob"
)

var ( //Compile-time interface checks
	_ = encoding.BinaryMarshaler((*StackTrie)(nil))
	_ = encoding.BinaryUnmarshaler((*StackTrie)(nil))
)

// NewFromBinaryV2 initialises a serialized stacktrie with the given db.
// OBS! Format was changed along with the name of this constructor.
func NewFromBinaryV2(data []byte, writeFn NodeWriteFunc) (*StackTrie, error) {
	stack := NewStackTrie(writeFn)
	if err := stack.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return stack, nil
}

// UnmarshalBinary implements encoding.BinaryMarshaler
func (st *StackTrie) MarshalBinary() (data []byte, err error) {
	var (
		b bytes.Buffer
		w = bufio.NewWriter(&b)
	)
	if err := gob.NewEncoder(w).Encode(st.owner); err != nil {
		return nil, err
	}
	if err := st.root.marshalInto(w); err != nil {
		return nil, err
	}
	w.Flush()
	return b.Bytes(), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler
func (stack *StackTrie) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	if err := gob.NewDecoder(r).Decode(&stack.owner); err != nil {
		return err
	}
	if err := stack.root.unmarshalFrom(r); err != nil {
		return err
	}
	return nil
}

type encodedNode struct {
	NodeType uint8
	Val      []byte
	Key      []byte
}

func (st *stNode) marshalInto(w *bufio.Writer) (err error) {
	if err := gob.NewEncoder(w).Encode(encodedNode{st.nodeType, st.val, st.key}); err != nil {
		return err
	}
	for _, child := range st.children {
		if child == nil {
			w.WriteByte(0)
			continue
		}
		w.WriteByte(1)
		if err := child.marshalInto(w); err != nil {
			return err
		}
	}
	return nil
}

func (st *stNode) unmarshalFrom(r *bytes.Reader) error {
	var dec encodedNode
	if err := gob.NewDecoder(r).Decode(&dec); err != nil {
		return err
	}
	st.nodeType = dec.NodeType
	st.val = dec.Val
	st.key = dec.Key

	for i := range st.children {
		if b, err := r.ReadByte(); err != nil {
			return err
		} else if b == 0 {
			continue
		}
		var child stNode
		if err := child.unmarshalFrom(r); err != nil {
			return err
		}
		st.children[i] = &child
	}
	return nil
}
