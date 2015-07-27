// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

import "github.com/ethereum/go-ethereum/crypto"

var keyPrefix = []byte("secure-key-")

type SecureTrie struct {
	*Trie
}

func NewSecure(root []byte, backend Backend) *SecureTrie {
	return &SecureTrie{New(root, backend)}
}

func (self *SecureTrie) Update(key, value []byte) Node {
	shaKey := crypto.Sha3(key)
	self.Trie.cache.Put(append(keyPrefix, shaKey...), key)

	return self.Trie.Update(shaKey, value)
}
func (self *SecureTrie) UpdateString(key, value string) Node {
	return self.Update([]byte(key), []byte(value))
}

func (self *SecureTrie) Get(key []byte) []byte {
	return self.Trie.Get(crypto.Sha3(key))
}
func (self *SecureTrie) GetString(key string) []byte {
	return self.Get([]byte(key))
}

func (self *SecureTrie) Delete(key []byte) Node {
	return self.Trie.Delete(crypto.Sha3(key))
}
func (self *SecureTrie) DeleteString(key string) Node {
	return self.Delete([]byte(key))
}

func (self *SecureTrie) Copy() *SecureTrie {
	return &SecureTrie{self.Trie.Copy()}
}

func (self *SecureTrie) GetKey(shaKey []byte) []byte {
	return self.Trie.cache.Get(append(keyPrefix, shaKey...))
}
