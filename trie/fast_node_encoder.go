// Copyright 2022 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/rlp"
)

// fastNodeEncoder is the fast node encoder using rlp.EncoderBuffer.
type fastNodeEncoder struct{}

var frlp fastNodeEncoder

// Encode writes the RLP encoding of node to w.
func (fastNodeEncoder) Encode(w io.Writer, node node) error {
	enc := rlp.NewEncoderBuffer(w)
	if err := fastEncodeNode(&enc, node); err != nil {
		return err
	}
	return enc.Flush()
}

// EncodeToBytes returns the RLP encoding of node.
func (fastNodeEncoder) EncodeToBytes(node node) ([]byte, error) {
	enc := rlp.NewEncoderBuffer(nil)
	defer enc.Flush()

	if err := fastEncodeNode(&enc, node); err != nil {
		return nil, err
	}
	return enc.ToBytes(), nil
}

func fastEncodeNode(w *rlp.EncoderBuffer, n node) error {
	switch n := n.(type) {
	case *fullNode:
		offset := w.List()
		for _, c := range n.Children {
			if c != nil {
				if err := fastEncodeNode(w, c); err != nil {
					return err
				}
			} else {
				w.Write(rlp.EmptyString)
			}
		}
		w.ListEnd(offset)
	case *shortNode:
		offset := w.List()
		w.WriteBytes(n.Key)
		if n.Val != nil {
			if err := fastEncodeNode(w, n.Val); err != nil {
				return err
			}
		} else {
			w.Write(rlp.EmptyString)
		}
		w.ListEnd(offset)
	case hashNode:
		w.WriteBytes(n)
	case valueNode:
		w.WriteBytes(n)
	case rawFullNode:
		offset := w.List()
		for _, c := range n {
			if c != nil {
				if err := fastEncodeNode(w, c); err != nil {
					return err
				}
			} else {
				w.Write(rlp.EmptyString)
			}
		}
		w.ListEnd(offset)
	case *rawShortNode:
		offset := w.List()
		w.WriteBytes(n.Key)
		if n.Val != nil {
			if err := fastEncodeNode(w, n.Val); err != nil {
				return err
			}
		} else {
			w.Write(rlp.EmptyString)
		}
		w.ListEnd(offset)
	case rawNode:
		w.Write(n)
	default:
		return fmt.Errorf("unexpected node type: %T", n)
	}
	return nil
}
