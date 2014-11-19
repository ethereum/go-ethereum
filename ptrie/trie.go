package ptrie

import (
	"bytes"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/trie"
)

type Backend interface {
	Get([]byte) []byte
	Set([]byte, []byte)
}

type Cache map[string][]byte

func (self Cache) Get(key []byte) []byte {
	return self[string(key)]
}
func (self Cache) Set(key []byte, data []byte) {
	self[string(key)] = data
}

type Trie struct {
	mu       sync.Mutex
	root     Node
	roothash []byte
	backend  Backend
}

func NewEmpty() *Trie {
	return &Trie{sync.Mutex{}, nil, nil, make(Cache)}
}

func New(root []byte, backend Backend) *Trie {
	trie := &Trie{}
	trie.roothash = root
	trie.backend = backend

	value := ethutil.NewValueFromBytes(trie.backend.Get(root))
	trie.root = trie.mknode(value)

	return trie
}

func (self *Trie) Iterator() *Iterator {
	return NewIterator(self)
}

// Legacy support
func (self *Trie) Root() []byte { return self.Hash() }
func (self *Trie) Hash() []byte {
	var hash []byte
	if self.root != nil {
		t := self.root.Hash()
		if byts, ok := t.([]byte); ok {
			hash = byts
		} else {
			hash = crypto.Sha3(ethutil.Encode(self.root.RlpData()))
		}
	} else {
		hash = crypto.Sha3(ethutil.Encode(self.root))
	}

	self.roothash = hash

	return hash
}

func (self *Trie) UpdateString(key, value string) Node { return self.Update([]byte(key), []byte(value)) }
func (self *Trie) Update(key, value []byte) Node {
	self.mu.Lock()
	defer self.mu.Unlock()

	k := trie.CompactHexDecode(string(key))

	if len(value) != 0 {
		self.root = self.insert(self.root, k, &ValueNode{self, value})
	} else {
		self.root = self.delete(self.root, k)
	}

	return self.root
}

func (self *Trie) GetString(key string) []byte { return self.Get([]byte(key)) }
func (self *Trie) Get(key []byte) []byte {
	self.mu.Lock()
	defer self.mu.Unlock()

	k := trie.CompactHexDecode(string(key))

	n := self.get(self.root, k)
	if n != nil {
		return n.(*ValueNode).Val()
	}

	return nil
}

func (self *Trie) DeleteString(key string) Node { return self.Delete([]byte(key)) }
func (self *Trie) Delete(key []byte) Node {
	self.mu.Lock()
	defer self.mu.Unlock()

	k := trie.CompactHexDecode(string(key))
	self.root = self.delete(self.root, k)

	return self.root
}

func (self *Trie) insert(node Node, key []byte, value Node) Node {
	if len(key) == 0 {
		return value
	}

	if node == nil {
		return NewShortNode(self, key, value)
	}

	switch node := node.(type) {
	case *ShortNode:
		k := node.Key()
		cnode := node.Value()
		if bytes.Equal(k, key) {
			return NewShortNode(self, key, value)
		}

		var n Node
		matchlength := trie.MatchingNibbleLength(key, k)
		if matchlength == len(k) {
			n = self.insert(cnode, key[matchlength:], value)
		} else {
			pnode := self.insert(nil, k[matchlength+1:], cnode)
			nnode := self.insert(nil, key[matchlength+1:], value)
			fulln := NewFullNode(self)
			fulln.set(k[matchlength], pnode)
			fulln.set(key[matchlength], nnode)
			n = fulln
		}
		if matchlength == 0 {
			return n
		}

		return NewShortNode(self, key[:matchlength], n)

	case *FullNode:
		cpy := node.Copy().(*FullNode)
		cpy.set(key[0], self.insert(node.branch(key[0]), key[1:], value))

		return cpy

	default:
		panic("Invalid node")
	}
}

func (self *Trie) get(node Node, key []byte) Node {
	if len(key) == 0 {
		return node
	}

	if node == nil {
		return nil
	}

	switch node := node.(type) {
	case *ShortNode:
		k := node.Key()
		cnode := node.Value()

		if len(key) >= len(k) && bytes.Equal(k, key[:len(k)]) {
			return self.get(cnode, key[len(k):])
		}

		return nil
	case *FullNode:
		return self.get(node.branch(key[0]), key[1:])
	default:
		panic("Invalid node")
	}
}

func (self *Trie) delete(node Node, key []byte) Node {
	if len(key) == 0 {
		return nil
	}

	switch node := node.(type) {
	case *ShortNode:
		k := node.Key()
		cnode := node.Value()
		if bytes.Equal(key, k) {
			return nil
		} else if bytes.Equal(key[:len(k)], k) {
			child := self.delete(cnode, key[len(k):])

			var n Node
			switch child := child.(type) {
			case *ShortNode:
				nkey := append(k, child.Key()...)
				n = NewShortNode(self, nkey, child.Value())
			case *FullNode:
				n = NewShortNode(self, node.key, child)
			}

			return n
		} else {
			return node
		}

	case *FullNode:
		n := node.Copy().(*FullNode)
		n.set(key[0], self.delete(n.branch(key[0]), key[1:]))

		pos := -1
		for i := 0; i < 17; i++ {
			if n.branch(byte(i)) != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
				}
			}
		}

		var nnode Node
		if pos == 16 {
			nnode = NewShortNode(self, []byte{16}, n.branch(byte(pos)))
		} else if pos >= 0 {
			cnode := n.branch(byte(pos))
			switch cnode := cnode.(type) {
			case *ShortNode:
				// Stitch keys
				k := append([]byte{byte(pos)}, cnode.Key()...)
				nnode = NewShortNode(self, k, cnode.Value())
			case *FullNode:
				nnode = NewShortNode(self, []byte{byte(pos)}, n.branch(byte(pos)))
			}
		} else {
			nnode = n
		}

		return nnode

	default:
		panic("Invalid node")
	}
}

// casting functions and cache storing
func (self *Trie) mknode(value *ethutil.Value) Node {
	l := value.Len()
	switch l {
	case 2:
		return NewShortNode(self, trie.CompactDecode(string(value.Get(0).Bytes())), self.mknode(value.Get(1)))
	case 17:
		fnode := NewFullNode(self)
		for i := 0; i < l; i++ {
			fnode.set(byte(i), self.mknode(value.Get(i)))
		}
		return fnode
	case 32:
		return &HashNode{value.Bytes()}
	default:
		return &ValueNode{self, value.Bytes()}
	}
}

func (self *Trie) trans(node Node) Node {
	switch node := node.(type) {
	case *HashNode:
		value := ethutil.NewValueFromBytes(self.backend.Get(node.key))
		return self.mknode(value)
	default:
		return node
	}
}

func (self *Trie) store(node Node) interface{} {
	data := ethutil.Encode(node)
	if len(data) >= 32 {
		key := crypto.Sha3(data)
		self.backend.Set(key, data)

		return key
	}

	return node.RlpData()
}
