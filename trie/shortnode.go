package trie

import "github.com/ethereum/go-ethereum/common"

type ShortNode struct {
	trie  *Trie
	key   []byte
	value Node
}

func NewShortNode(t *Trie, key []byte, value Node) *ShortNode {
	return &ShortNode{t, []byte(CompactEncode(key)), value}
}
func (self *ShortNode) Value() Node {
	self.value = self.trie.trans(self.value)

	return self.value
}
func (self *ShortNode) Dirty() bool { return true }
func (self *ShortNode) Copy(t *Trie) Node {
	node := &ShortNode{t, nil, self.value.Copy(t)}
	node.key = common.CopyBytes(self.key)
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
