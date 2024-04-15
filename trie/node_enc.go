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
	"github.com/ethereum/go-ethereum/rlp"
)

func (n *fullNode) encode(buf []byte) []byte {
	if buf == nil {
		buf = make([]byte, 0, 550)
	}
	var offset int
	buf, offset = rlp.StartList(buf)
	for _, c := range n.Children {
		if c != nil {
			buf = c.encode(buf)
		} else {
			buf = rlp.AppendUint64(buf, 0)
		}
	}
	return rlp.EndList(buf, offset)
}

func (n *shortNode) encode(buf []byte) []byte {
	if buf == nil {
		buf = make([]byte, 0, len(n.Key)+40)
	}
	var offset int
	buf, offset = rlp.StartList(buf)
	buf = rlp.AppendString(buf, n.Key)
	if n.Val != nil {
		buf = n.Val.encode(buf)
	} else {
		buf = rlp.AppendUint64(buf, 0)
	}
	return rlp.EndList(buf, offset)
}

func (n hashNode) encode(buf []byte) []byte {
	return rlp.AppendString(buf, n)
}

func (n valueNode) encode(buf []byte) []byte {
	return rlp.AppendString(buf, n)
}

func (n rawNode) encode(buf []byte) []byte {
	return append(buf, n...)
}
