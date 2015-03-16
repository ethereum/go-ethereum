package trie

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type SecureTrie struct {
	*Trie
}

func NewSecure(root common.Hash, backend Backend) *SecureTrie {
	return &SecureTrie{New(root, backend)}
}

func (self *SecureTrie) Update(key common.Hash, value []byte) Node {
	return self.Trie.Update(common.BytesToHash(crypto.Sha3(key[:])), value)
}

func (self *SecureTrie) UpdateString(key, value string) Node {
	return self.Update(common.StringToHash(key), []byte(value))
}

func (self *SecureTrie) Get(key common.Hash) []byte {
	return self.Trie.Get(common.BytesToHash(crypto.Sha3(key[:])))
}
func (self *SecureTrie) GetString(key string) []byte {
	return self.Get(common.StringToHash(key))
}

func (self *SecureTrie) Delete(key common.Hash) Node {
	return self.Trie.Delete(common.BytesToHash(crypto.Sha3(key[:])))
}
func (self *SecureTrie) DeleteString(key string) Node {
	return self.Delete(common.StringToHash(key))
}

func (self *SecureTrie) Copy() *SecureTrie {
	return &SecureTrie{self.Trie.Copy()}
}
