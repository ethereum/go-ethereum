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
	"bytes"
)

type Iterator struct {
	trie *Trie

	Key   []byte
	Value []byte
}

func NewIterator(trie *Trie) *Iterator {
	return &Iterator{trie: trie, Key: nil}
}

func (self *Iterator) Next() bool {
	self.trie.mu.Lock()
	defer self.trie.mu.Unlock()

	isIterStart := false
	if self.Key == nil {
		isIterStart = true
		self.Key = make([]byte, 32)
	}

	key := RemTerm(CompactHexDecode(self.Key))
	k := self.next(self.trie.root, key, isIterStart)

	self.Key = []byte(DecodeCompact(k))

	return len(k) > 0
}

func (self *Iterator) next(node Node, key []byte, isIterStart bool) []byte {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case *FullNode:
		if len(key) > 0 {
			k := self.next(node.branch(key[0]), key[1:], isIterStart)
			if k != nil {
				return append([]byte{key[0]}, k...)
			}
		}

		var r byte
		if len(key) > 0 {
			r = key[0] + 1
		}

		for i := r; i < 16; i++ {
			k := self.key(node.branch(byte(i)))
			if k != nil {
				return append([]byte{i}, k...)
			}
		}

	case *ShortNode:
		k := RemTerm(node.Key())
		if vnode, ok := node.Value().(*ValueNode); ok {
			switch bytes.Compare([]byte(k), key) {
			case 0:
				if isIterStart {
					self.Value = vnode.Val()
					return k
				}
			case 1:
				self.Value = vnode.Val()
				return k
			}
		} else {
			cnode := node.Value()

			var ret []byte
			skey := key[len(k):]
			if BeginsWith(key, k) {
				ret = self.next(cnode, skey, isIterStart)
			} else if bytes.Compare(k, key[:len(k)]) > 0 {
				return self.key(node)
			}

			if ret != nil {
				return append(k, ret...)
			}
		}
	}

	return nil
}

func (self *Iterator) key(node Node) []byte {
	switch node := node.(type) {
	case *ShortNode:
		// Leaf node
		if vnode, ok := node.Value().(*ValueNode); ok {
			k := RemTerm(node.Key())
			self.Value = vnode.Val()

			return k
		} else {
			k := RemTerm(node.Key())
			return append(k, self.key(node.Value())...)
		}
	case *FullNode:
		if node.Value() != nil {
			self.Value = node.Value().(*ValueNode).Val()

			return []byte{16}
		}

		for i := 0; i < 16; i++ {
			k := self.key(node.branch(byte(i)))
			if k != nil {
				return append([]byte{byte(i)}, k...)
			}
		}
	}

	return nil
}
