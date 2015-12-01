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

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
	isIterStart := false
	if self.Key == nil {
		isIterStart = true
		self.Key = make([]byte, 32)
	}

	key := remTerm(compactHexDecode(self.Key))
	k := self.next(self.trie.root, key, isIterStart)

	self.Key = []byte(decodeCompact(k))

	return len(k) > 0
}

func (self *Iterator) next(node interface{}, key []byte, isIterStart bool) []byte {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case fullNode:
		if len(key) > 0 {
			k := self.next(node[key[0]], key[1:], isIterStart)
			if k != nil {
				return append([]byte{key[0]}, k...)
			}
		}

		var r byte
		if len(key) > 0 {
			r = key[0] + 1
		}

		for i := r; i < 16; i++ {
			k := self.key(node[i])
			if k != nil {
				return append([]byte{i}, k...)
			}
		}

	case shortNode:
		k := remTerm(node.Key)
		if vnode, ok := node.Val.(valueNode); ok {
			switch bytes.Compare([]byte(k), key) {
			case 0:
				if isIterStart {
					self.Value = vnode
					return k
				}
			case 1:
				self.Value = vnode
				return k
			}
		} else {
			cnode := node.Val

			var ret []byte
			skey := key[len(k):]
			if bytes.HasPrefix(key, k) {
				ret = self.next(cnode, skey, isIterStart)
			} else if bytes.Compare(k, key[:len(k)]) > 0 {
				return self.key(node)
			}

			if ret != nil {
				return append(k, ret...)
			}
		}

	case hashNode:
		rn, err := self.trie.resolveHash(node, nil, nil)
		if err != nil && glog.V(logger.Error) {
			glog.Errorf("Unhandled trie error: %v", err)
		}
		return self.next(rn, key, isIterStart)
	}
	return nil
}

func (self *Iterator) key(node interface{}) []byte {
	switch node := node.(type) {
	case shortNode:
		// Leaf node
		k := remTerm(node.Key)
		if vnode, ok := node.Val.(valueNode); ok {
			self.Value = vnode
			return k
		}
		return append(k, self.key(node.Val)...)
	case fullNode:
		if node[16] != nil {
			self.Value = node[16].(valueNode)
			return []byte{16}
		}
		for i := 0; i < 16; i++ {
			k := self.key(node[i])
			if k != nil {
				return append([]byte{byte(i)}, k...)
			}
		}
	case hashNode:
		rn, err := self.trie.resolveHash(node, nil, nil)
		if err != nil && glog.V(logger.Error) {
			glog.Errorf("Unhandled trie error: %v", err)
		}
		return self.key(rn)
	}

	return nil
}
