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

type ValueNode struct {
	trie  *Trie
	data  []byte
	dirty bool
}

func NewValueNode(trie *Trie, data []byte) *ValueNode {
	return &ValueNode{trie, data, false}
}

func (self *ValueNode) Value() Node { return self } // Best not to call :-)
func (self *ValueNode) Val() []byte { return self.data }
func (self *ValueNode) Dirty() bool { return self.dirty }
func (self *ValueNode) Copy(t *Trie) Node {
	return &ValueNode{t, common.CopyBytes(self.data), self.dirty}
}
func (self *ValueNode) RlpData() interface{} { return self.data }
func (self *ValueNode) Hash() interface{}    { return self.data }

func (self *ValueNode) setDirty(dirty bool) {
	self.dirty = dirty
}
