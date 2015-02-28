package trie

import "github.com/ethereum/go-ethereum/crypto"

type SecureTrie struct {
	*Trie
}

func NewSecure(root []byte, backend Backend) *SecureTrie {
	return &SecureTrie{New(root, backend)}
}

func (self *SecureTrie) Update(key, value []byte) Node {
	return self.Trie.Update(crypto.Sha3(key), value)
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
func (self *SecureTrie) DeletString(key string) Node {
	return self.Delete([]byte(key))
}
