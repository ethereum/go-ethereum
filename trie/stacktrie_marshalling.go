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

// Compile-time interface checks.
var (
	_ = encoding.BinaryMarshaler((*StackTrie)(nil))
	_ = encoding.BinaryUnmarshaler((*StackTrie)(nil))
)

// NewFromBinaryV2 initialises a serialized stacktrie with the given db.
// OBS! Format was changed along with the name of this constructor.
func NewFromBinaryV2(data []byte) (*StackTrie, error) {
	stack := NewStackTrie(nil)
	if err := stack.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return stack, nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (t *StackTrie) MarshalBinary() (data []byte, err error) {
	var (
		b bytes.Buffer
		w = bufio.NewWriter(&b)
	)
	if err := gob.NewEncoder(w).Encode(t.owner); err != nil {
		return nil, err
	}
	if err := t.root.marshalInto(w); err != nil {
		return nil, err
	}
	w.Flush()
	return b.Bytes(), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (t *StackTrie) UnmarshalBinary(data []byte) error {
	r := bytes.NewReader(data)
	if err := gob.NewDecoder(r).Decode(&t.owner); err != nil {
		return err
	}
	if err := t.root.unmarshalFrom(r); err != nil {
		return err
	}
	return nil
}

type stackNodeMarshaling struct {
	Typ uint8
	Key []byte
	Val []byte
}

func (n *stNode) marshalInto(w *bufio.Writer) (err error) {
	enc := stackNodeMarshaling{
		Typ: n.typ,
		Key: n.key,
		Val: n.val,
	}
	if err := gob.NewEncoder(w).Encode(enc); err != nil {
		return err
	}
	for _, child := range n.children {
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

func (n *stNode) unmarshalFrom(r *bytes.Reader) error {
	var dec stackNodeMarshaling
	if err := gob.NewDecoder(r).Decode(&dec); err != nil {
		return err
	}
	n.typ = dec.Typ
	n.key = dec.Key
	n.val = dec.Val

	for i := range n.children {
		if b, err := r.ReadByte(); err != nil {
			return err
		} else if b == 0 {
			continue
		}
		var child stNode
		if err := child.unmarshalFrom(r); err != nil {
			return err
		}
		n.children[i] = &child
	}
	return nil
}
