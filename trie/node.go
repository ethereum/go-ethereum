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

import "fmt"

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

type Node interface {
	Value() Node
	Copy(*Trie) Node // All nodes, for now, return them self
	Dirty() bool
	fstring(string) string
	Hash() interface{}
	RlpData() interface{}
	setDirty(dirty bool)
}

// Value node
func (self *ValueNode) String() string            { return self.fstring("") }
func (self *FullNode) String() string             { return self.fstring("") }
func (self *ShortNode) String() string            { return self.fstring("") }
func (self *ValueNode) fstring(ind string) string { return fmt.Sprintf("%x ", self.data) }

//func (self *HashNode) fstring(ind string) string  { return fmt.Sprintf("< %x > ", self.key) }
func (self *HashNode) fstring(ind string) string {
	return fmt.Sprintf("%v", self.trie.trans(self))
}

// Full node
func (self *FullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range self.nodes {
		if node == nil {
			resp += fmt.Sprintf("%s: <nil> ", indices[i])
		} else {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}

	return resp + fmt.Sprintf("\n%s] ", ind)
}

// Short node
func (self *ShortNode) fstring(ind string) string {
	return fmt.Sprintf("[ %x: %v ] ", self.key, self.value.fstring(ind+"  "))
}
