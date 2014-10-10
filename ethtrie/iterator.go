package ethtrie

import (
	"bytes"

	"github.com/ethereum/eth-go/ethutil"
)

type NodeType byte

const (
	EmptyNode NodeType = iota
	BranchNode
	LeafNode
	ExtNode
)

func getType(node *ethutil.Value) NodeType {
	if node.Len() == 0 {
		return EmptyNode
	}

	if node.Len() == 2 {
		k := CompactDecode(node.Get(0).Str())
		if HasTerm(k) {
			return LeafNode
		}

		return ExtNode
	}

	return BranchNode
}

type Iterator struct {
	Path [][]byte
	trie *Trie

	Key   []byte
	Value *ethutil.Value
}

func NewIterator(trie *Trie) *Iterator {
	return &Iterator{trie: trie}
}

func (self *Iterator) key(node *ethutil.Value, path [][]byte) []byte {
	switch getType(node) {
	case LeafNode:
		k := RemTerm(CompactDecode(node.Get(0).Str()))

		self.Path = append(path, k)
		self.Value = node.Get(1)

		return k
	case BranchNode:
		if node.Get(16).Len() > 0 {
			return []byte{16}
		}

		for i := byte(0); i < 16; i++ {
			o := self.key(self.trie.getNode(node.Get(int(i)).Raw()), append(path, []byte{i}))
			if o != nil {
				return append([]byte{i}, o...)
			}
		}
	case ExtNode:
		currKey := node.Get(0).Bytes()

		return self.key(self.trie.getNode(node.Get(1).Raw()), append(path, currKey))
	}

	return nil
}

func (self *Iterator) next(node *ethutil.Value, key []byte, path [][]byte) []byte {
	switch typ := getType(node); typ {
	case EmptyNode:
		return nil
	case BranchNode:
		if len(key) > 0 {
			subNode := self.trie.getNode(node.Get(int(key[0])).Raw())

			o := self.next(subNode, key[1:], append(path, key[:1]))
			if o != nil {
				return append([]byte{key[0]}, o...)
			}
		}

		var r byte = 0
		if len(key) > 0 {
			r = key[0] + 1
		}

		for i := r; i < 16; i++ {
			subNode := self.trie.getNode(node.Get(int(i)).Raw())
			o := self.key(subNode, append(path, []byte{i}))
			if o != nil {
				return append([]byte{i}, o...)
			}
		}
	case LeafNode, ExtNode:
		k := RemTerm(CompactDecode(node.Get(0).Str()))
		if typ == LeafNode {
			if bytes.Compare([]byte(k), []byte(key)) > 0 {
				self.Value = node.Get(1)
				self.Path = append(path, k)

				return k
			}
		} else {
			subNode := self.trie.getNode(node.Get(1).Raw())
			subKey := key[len(k):]
			var ret []byte
			if BeginsWith(key, k) {
				ret = self.next(subNode, subKey, append(path, k))
			} else if bytes.Compare(k, key[:len(k)]) > 0 {
				ret = self.key(node, append(path, k))
			} else {
				ret = nil
			}

			if ret != nil {
				return append(k, ret...)
			}
		}
	}

	return nil
}

// Get the next in keys
func (self *Iterator) Next(key string) []byte {
	self.trie.mut.Lock()
	defer self.trie.mut.Unlock()

	k := RemTerm(CompactHexDecode(key))
	n := self.next(self.trie.getNode(self.trie.Root), k, nil)

	self.Key = []byte(DecodeCompact(n))

	return self.Key
}
