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

import "github.com/ethereum/go-ethereum/common"

type ShortNode struct {
	trie  *Trie
	key   []byte
	value Node
	dirty bool
}

func NewShortNode(t *Trie, key []byte, value Node) *ShortNode {
	return &ShortNode{t, []byte(CompactEncode(key)), value, false}
}
func (self *ShortNode) Value() Node {
	self.value = self.trie.trans(self.value)

	return self.value
}
func (self *ShortNode) Dirty() bool { return self.dirty }
func (self *ShortNode) Copy(t *Trie) Node {
	node := &ShortNode{t, nil, self.value.Copy(t), self.dirty}
	node.key = common.CopyBytes(self.key)
	node.dirty = true
	return node
}

func (self *ShortNode) RlpData() interface{} {
	return []interface{}{self.key, self.value.Hash()}
}
func (self *ShortNode) Hash() interface{} {
	return self.trie.store(self)
}

func (self *ShortNode) Key() []byte {
	return CompactDecode(string(self.key))
}

func (self *ShortNode) setDirty(dirty bool) {
	self.dirty = dirty
}
