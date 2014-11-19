package ptrie

import "github.com/ethereum/go-ethereum/trie"

type ShortNode struct {
	trie  *Trie
	key   []byte
	value Node
}

func NewShortNode(t *Trie, key []byte, value Node) *ShortNode {
	return &ShortNode{t, []byte(trie.CompactEncode(key)), value}
}
func (self *ShortNode) Value() Node {
	self.value = self.trie.trans(self.value)

	return self.value
}
func (self *ShortNode) Dirty() bool { return true }
func (self *ShortNode) Copy() Node  { return NewShortNode(self.trie, self.key, self.value) }

func (self *ShortNode) RlpData() interface{} {
	return []interface{}{self.key, self.value.Hash()}
}
func (self *ShortNode) Hash() interface{} {
	return self.trie.store(self)
}

func (self *ShortNode) Key() []byte {
	return trie.CompactDecode(string(self.key))
}
