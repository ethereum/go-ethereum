// Copyright 2014 The go-ethereum Authors && Copyright 2015 go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package trie

import "github.com/expanse-project/go-expanse/common"

type HashNode struct {
	key   []byte
	trie  *Trie
	dirty bool
}

func NewHash(key []byte, trie *Trie) *HashNode {
	return &HashNode{key, trie, false}
}

func (self *HashNode) RlpData() interface{} {
	return self.key
}

func (self *HashNode) Hash() interface{} {
	return self.key
}

func (self *HashNode) setDirty(dirty bool) {
	self.dirty = dirty
}

// These methods will never be called but we have to satisfy Node interface
func (self *HashNode) Value() Node       { return nil }
func (self *HashNode) Dirty() bool       { return true }
func (self *HashNode) Copy(t *Trie) Node { return NewHash(common.CopyBytes(self.key), t) }
