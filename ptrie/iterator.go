package ptrie

import (
	"bytes"

	"github.com/ethereum/go-ethereum/trie"
)

type Iterator struct {
	trie *Trie

	Key   []byte
	Value []byte
}

func NewIterator(trie *Trie) *Iterator {
	return &Iterator{trie: trie, Key: []byte{0}}
}

func (self *Iterator) Next() bool {
	self.trie.mu.Lock()
	defer self.trie.mu.Unlock()

	key := trie.RemTerm(trie.CompactHexDecode(string(self.Key)))
	k := self.next(self.trie.root, key)

	self.Key = []byte(trie.DecodeCompact(k))

	return len(k) > 0

}

func (self *Iterator) next(node Node, key []byte) []byte {
	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case *FullNode:
		if len(key) > 0 {
			k := self.next(node.branch(key[0]), key[1:])
			if k != nil {
				return append([]byte{key[0]}, k...)
			}
		}

		var r byte
		if len(key) > 0 {
			r = key[0] + 1
		}

		for i := r; i < 16; i++ {
			k := self.key(node.branch(byte(i)))
			if k != nil {
				return append([]byte{i}, k...)
			}
		}

	case *ShortNode:
		k := trie.RemTerm(node.Key())
		if vnode, ok := node.Value().(*ValueNode); ok {
			if bytes.Compare([]byte(k), key) > 0 {
				self.Value = vnode.Val()
				return k
			}
		} else {
			cnode := node.Value()
			skey := key[len(k):]

			var ret []byte
			if trie.BeginsWith(key, k) {
				ret = self.next(cnode, skey)
			} else if bytes.Compare(k, key[:len(k)]) > 0 {
				ret = self.key(node)
			}

			if ret != nil {
				return append(k, ret...)
			}
		}
	}

	return nil
}

func (self *Iterator) key(node Node) []byte {
	switch node := node.(type) {
	case *ShortNode:
		// Leaf node
		if vnode, ok := node.Value().(*ValueNode); ok {
			k := trie.RemTerm(node.Key())
			self.Value = vnode.Val()

			return k
		} else {
			return self.key(node.Value())
		}
	case *FullNode:
		if node.Value() != nil {
			self.Value = node.Value().(*ValueNode).Val()

			return []byte{16}
		}

		for i := 0; i < 16; i++ {
			k := self.key(node.branch(byte(i)))
			if k != nil {
				return append([]byte{byte(i)}, k...)
			}
		}
	}

	return nil
}
