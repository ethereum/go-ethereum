package trie

import (
	"bytes"
	"container/list"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ParanoiaCheck(t1 *Trie, backend Backend) (bool, *Trie) {
	t2 := New(nil, backend)

	it := t1.Iterator()
	for it.Next() {
		t2.Update(it.Key, it.Value)
	}

	return bytes.Equal(t2.Hash(), t1.Hash()), t2
}

type Trie struct {
	mu       sync.Mutex
	root     Node
	roothash []byte
	cache    *Cache

	revisions *list.List
}

func New(root []byte, backend Backend) *Trie {
	trie := &Trie{}
	trie.revisions = list.New()
	trie.roothash = root
	if backend != nil {
		trie.cache = NewCache(backend)
	}

	if root != nil {
		value := common.NewValueFromBytes(trie.cache.Get(root))
		trie.root = trie.mknode(value)
	}

	return trie
}

func (self *Trie) Iterator() *Iterator {
	return NewIterator(self)
}

func (self *Trie) Copy() *Trie {
	cpy := make([]byte, 32)
	copy(cpy, self.roothash)
	trie := New(nil, nil)
	trie.cache = self.cache.Copy()
	if self.root != nil {
		trie.root = self.root.Copy(trie)
	}

	return trie
}

// Legacy support
func (self *Trie) Root() []byte { return self.Hash() }
func (self *Trie) Hash() []byte {
	var hash []byte
	if self.root != nil {
		t := self.root.Hash()
		if byts, ok := t.([]byte); ok && len(byts) > 0 {
			hash = byts
		} else {
			hash = crypto.Sha3(common.Encode(self.root.RlpData()))
		}
	} else {
		hash = crypto.Sha3(common.Encode(""))
	}

	if !bytes.Equal(hash, self.roothash) {
		self.revisions.PushBack(self.roothash)
		self.roothash = hash
	}

	return hash
}
func (self *Trie) Commit() {
	self.mu.Lock()
	defer self.mu.Unlock()

	// Hash first
	self.Hash()

	self.cache.Flush()
}

// Reset should only be called if the trie has been hashed
func (self *Trie) Reset() {
	self.mu.Lock()
	defer self.mu.Unlock()

	self.cache.Reset()

	if self.revisions.Len() > 0 {
		revision := self.revisions.Remove(self.revisions.Back()).([]byte)
		self.roothash = revision
	}
	value := common.NewValueFromBytes(self.cache.Get(self.roothash))
	self.root = self.mknode(value)
}

func (self *Trie) UpdateString(key, value string) Node { return self.Update([]byte(key), []byte(value)) }
func (self *Trie) Update(key, value []byte) Node {
	self.mu.Lock()
	defer self.mu.Unlock()

	k := CompactHexDecode(string(key))

	if len(value) != 0 {
		node := NewValueNode(self, value)
		node.dirty = true
		self.root = self.insert(self.root, k, node)
	} else {
		self.root = self.delete(self.root, k)
	}

	return self.root
}

func (self *Trie) GetString(key string) []byte { return self.Get([]byte(key)) }
func (self *Trie) Get(key []byte) []byte {
	self.mu.Lock()
	defer self.mu.Unlock()

	k := CompactHexDecode(string(key))

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

	k := CompactHexDecode(string(key))
	self.root = self.delete(self.root, k)

	return self.root
}

func (self *Trie) insert(node Node, key []byte, value Node) Node {
	if len(key) == 0 {
		return value
	}

	if node == nil {
		node := NewShortNode(self, key, value)
		node.dirty = true
		return node
	}

	switch node := node.(type) {
	case *ShortNode:
		k := node.Key()
		cnode := node.Value()
		if bytes.Equal(k, key) {
			node := NewShortNode(self, key, value)
			node.dirty = true
			return node

		}

		var n Node
		matchlength := MatchingNibbleLength(key, k)
		if matchlength == len(k) {
			n = self.insert(cnode, key[matchlength:], value)
		} else {
			pnode := self.insert(nil, k[matchlength+1:], cnode)
			nnode := self.insert(nil, key[matchlength+1:], value)
			fulln := NewFullNode(self)
			fulln.dirty = true
			fulln.set(k[matchlength], pnode)
			fulln.set(key[matchlength], nnode)
			n = fulln
		}
		if matchlength == 0 {
			return n
		}

		snode := NewShortNode(self, key[:matchlength], n)
		snode.dirty = true
		return snode

	case *FullNode:
		cpy := node.Copy(self).(*FullNode)
		cpy.set(key[0], self.insert(node.branch(key[0]), key[1:], value))
		cpy.dirty = true

		return cpy

	default:
		panic(fmt.Sprintf("%T: invalid node: %v", node, node))
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
		panic(fmt.Sprintf("%T: invalid node: %v", node, node))
	}
}

func (self *Trie) delete(node Node, key []byte) Node {
	if len(key) == 0 && node == nil {
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
				n.(*ShortNode).dirty = true
			case *FullNode:
				sn := NewShortNode(self, node.Key(), child)
				sn.dirty = true
				sn.key = node.key
				n = sn
			}

			return n
		} else {
			return node
		}

	case *FullNode:
		n := node.Copy(self).(*FullNode)
		n.set(key[0], self.delete(n.branch(key[0]), key[1:]))
		n.dirty = true

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
			nnode.(*ShortNode).dirty = true
		} else if pos >= 0 {
			cnode := n.branch(byte(pos))
			switch cnode := cnode.(type) {
			case *ShortNode:
				// Stitch keys
				k := append([]byte{byte(pos)}, cnode.Key()...)
				nnode = NewShortNode(self, k, cnode.Value())
				nnode.(*ShortNode).dirty = true
			case *FullNode:
				nnode = NewShortNode(self, []byte{byte(pos)}, n.branch(byte(pos)))
				nnode.(*ShortNode).dirty = true
			}
		} else {
			nnode = n
		}

		return nnode
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", node, node, key))
	}
}

// casting functions and cache storing
func (self *Trie) mknode(value *common.Value) Node {
	l := value.Len()
	switch l {
	case 0:
		return nil
	case 2:
		// A value node may consists of 2 bytes.
		if value.Get(0).Len() != 0 {
			key := CompactDecode(string(value.Get(0).Bytes()))
			if key[len(key)-1] == 16 {
				return NewShortNode(self, key, NewValueNode(self, value.Get(1).Bytes()))
			} else {
				return NewShortNode(self, key, self.mknode(value.Get(1)))
			}
		}
	case 17:
		if len(value.Bytes()) != 17 {
			fnode := NewFullNode(self)
			for i := 0; i < 16; i++ {
				fnode.set(byte(i), self.mknode(value.Get(i)))
			}
			return fnode
		}
	case 32:
		return NewHash(value.Bytes(), self)
	}

	return NewValueNode(self, value.Bytes())
}

func (self *Trie) trans(node Node) Node {
	switch node := node.(type) {
	case *HashNode:
		value := common.NewValueFromBytes(self.cache.Get(node.key))
		return self.mknode(value)
	default:
		return node
	}
}

func (self *Trie) store(node Node) interface{} {
	data := common.Encode(node)
	if len(data) >= 32 {
		key := crypto.Sha3(data)
		if node.Dirty() {
			//fmt.Println("save", node)
			//fmt.Println()
			self.cache.Put(key, data)
		}

		return key
	}

	return node.RlpData()
}

func (self *Trie) PrintRoot() {
	fmt.Println(self.root)
	fmt.Printf("root=%x\n", self.Root())
}
