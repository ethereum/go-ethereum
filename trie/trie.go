package trie

/*
import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
)

func ParanoiaCheck(t1 *Trie) (bool, *Trie) {
	t2 := New(ethutil.Config.Db, "")

	t1.NewIterator().Each(func(key string, v *ethutil.Value) {
		t2.Update(key, v.Str())
	})

	return bytes.Compare(t2.GetRoot(), t1.GetRoot()) == 0, t2
}

func (s *Cache) Len() int {
	return len(s.nodes)
}

// TODO
// A StateObject is an object that has a state root
// This is goig to be the object for the second level caching (the caching of object which have a state such as contracts)
type StateObject interface {
	State() *Trie
	Sync()
	Undo()
}

type Node struct {
	Key   []byte
	Value *ethutil.Value
	Dirty bool
}

func NewNode(key []byte, val *ethutil.Value, dirty bool) *Node {
	return &Node{Key: key, Value: val, Dirty: dirty}
}

func (n *Node) Copy() *Node {
	return NewNode(n.Key, n.Value, n.Dirty)
}

type Cache struct {
	nodes   map[string]*Node
	db      ethutil.Database
	IsDirty bool
}

func NewCache(db ethutil.Database) *Cache {
	return &Cache{db: db, nodes: make(map[string]*Node)}
}

func (cache *Cache) PutValue(v interface{}, force bool) interface{} {
	value := ethutil.NewValue(v)

	enc := value.Encode()
	if len(enc) >= 32 || force {
		sha := crypto.Sha3(enc)

		cache.nodes[string(sha)] = NewNode(sha, value, true)
		cache.IsDirty = true

		return sha
	}

	return v
}

func (cache *Cache) Put(v interface{}) interface{} {
	return cache.PutValue(v, false)
}

func (cache *Cache) Get(key []byte) *ethutil.Value {
	// First check if the key is the cache
	if cache.nodes[string(key)] != nil {
		return cache.nodes[string(key)].Value
	}

	// Get the key of the database instead and cache it
	data, _ := cache.db.Get(key)
	// Create the cached value
	value := ethutil.NewValueFromBytes(data)

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("RECOVER GET", cache, cache.nodes)
			panic("bye")
		}
	}()
	// Create caching node
	cache.nodes[string(key)] = NewNode(key, value, true)

	return value
}

func (cache *Cache) Delete(key []byte) {
	delete(cache.nodes, string(key))

	cache.db.Delete(key)
}

func (cache *Cache) Commit() {
	// Don't try to commit if it isn't dirty
	if !cache.IsDirty {
		return
	}

	for key, node := range cache.nodes {
		if node.Dirty {
			cache.db.Put([]byte(key), node.Value.Encode())
			node.Dirty = false
		}
	}
	cache.IsDirty = false

	// If the nodes grows beyond the 200 entries we simple empty it
	// FIXME come up with something better
	if len(cache.nodes) > 200 {
		cache.nodes = make(map[string]*Node)
	}
}

func (cache *Cache) Undo() {
	for key, node := range cache.nodes {
		if node.Dirty {
			delete(cache.nodes, key)
		}
	}
	cache.IsDirty = false
}

// A (modified) Radix Trie implementation. The Trie implements
// a caching mechanism and will used cached values if they are
// present. If a node is not present in the cache it will try to
// fetch it from the database and store the cached value.
// Please note that the data isn't persisted unless `Sync` is
// explicitly called.
type Trie struct {
	mut      sync.RWMutex
	prevRoot interface{}
	Root     interface{}
	//db   Database
	cache *Cache
}

func copyRoot(root interface{}) interface{} {
	var prevRootCopy interface{}
	if b, ok := root.([]byte); ok {
		prevRootCopy = ethutil.CopyBytes(b)
	} else {
		prevRootCopy = root
	}

	return prevRootCopy
}

func New(db ethutil.Database, Root interface{}) *Trie {
	// Make absolute sure the root is copied
	r := copyRoot(Root)
	p := copyRoot(Root)

	trie := &Trie{cache: NewCache(db), Root: r, prevRoot: p}
	trie.setRoot(Root)

	return trie
}

func (self *Trie) setRoot(root interface{}) {
	switch t := root.(type) {
	case string:
		//if t == "" {
		//	root = crypto.Sha3(ethutil.Encode(""))
		//}
		self.Root = []byte(t)
	case []byte:
		self.Root = root
	default:
		self.Root = self.cache.PutValue(root, true)
	}
}

func (t *Trie) Update(key, value string) {
	t.mut.Lock()
	defer t.mut.Unlock()

	k := CompactHexDecode(key)

	var root interface{}
	if value != "" {
		root = t.UpdateState(t.Root, k, value)
	} else {
		root = t.deleteState(t.Root, k)
	}
	t.setRoot(root)
}

func (t *Trie) Get(key string) string {
	t.mut.Lock()
	defer t.mut.Unlock()

	k := CompactHexDecode(key)
	c := ethutil.NewValue(t.getState(t.Root, k))

	return c.Str()
}

func (t *Trie) Delete(key string) {
	t.mut.Lock()
	defer t.mut.Unlock()

	k := CompactHexDecode(key)

	root := t.deleteState(t.Root, k)
	t.setRoot(root)
}

func (self *Trie) GetRoot() []byte {
	switch t := self.Root.(type) {
	case string:
		if t == "" {
			return crypto.Sha3(ethutil.Encode(""))
		}
		return []byte(t)
	case []byte:
		if len(t) == 0 {
			return crypto.Sha3(ethutil.Encode(""))
		}

		return t
	default:
		panic(fmt.Sprintf("invalid root type %T (%v)", self.Root, self.Root))
	}
}

// Simple compare function which creates a rlp value out of the evaluated objects
func (t *Trie) Cmp(trie *Trie) bool {
	return ethutil.NewValue(t.Root).Cmp(ethutil.NewValue(trie.Root))
}

// Returns a copy of this trie
func (t *Trie) Copy() *Trie {
	trie := New(t.cache.db, t.Root)
	for key, node := range t.cache.nodes {
		trie.cache.nodes[key] = node.Copy()
	}

	return trie
}

// Save the cached value to the database.
func (t *Trie) Sync() {
	t.cache.Commit()
	t.prevRoot = copyRoot(t.Root)
}

func (t *Trie) Undo() {
	t.cache.Undo()
	t.Root = t.prevRoot
}

func (t *Trie) Cache() *Cache {
	return t.cache
}

func (t *Trie) getState(node interface{}, key []byte) interface{} {
	n := ethutil.NewValue(node)
	// Return the node if key is empty (= found)
	if len(key) == 0 || n.IsNil() || n.Len() == 0 {
		return node
	}

	currentNode := t.getNode(node)
	length := currentNode.Len()

	if length == 0 {
		return ""
	} else if length == 2 {
		// Decode the key
		k := CompactDecode(currentNode.Get(0).Str())
		v := currentNode.Get(1).Raw()

		if len(key) >= len(k) && bytes.Equal(k, key[:len(k)]) { //CompareIntSlice(k, key[:len(k)]) {
			return t.getState(v, key[len(k):])
		} else {
			return ""
		}
	} else if length == 17 {
		return t.getState(currentNode.Get(int(key[0])).Raw(), key[1:])
	}

	// It shouldn't come this far
	panic("unexpected return")
}

func (t *Trie) getNode(node interface{}) *ethutil.Value {
	n := ethutil.NewValue(node)

	if !n.Get(0).IsNil() {
		return n
	}

	str := n.Str()
	if len(str) == 0 {
		return n
	} else if len(str) < 32 {
		return ethutil.NewValueFromBytes([]byte(str))
	}

	data := t.cache.Get(n.Bytes())

	return data
}

func (t *Trie) UpdateState(node interface{}, key []byte, value string) interface{} {
	return t.InsertState(node, key, value)
}

func (t *Trie) Put(node interface{}) interface{} {
	return t.cache.Put(node)

}

func EmptyStringSlice(l int) []interface{} {
	slice := make([]interface{}, l)
	for i := 0; i < l; i++ {
		slice[i] = ""
	}
	return slice
}

func (t *Trie) InsertState(node interface{}, key []byte, value interface{}) interface{} {
	if len(key) == 0 {
		return value
	}

	// New node
	n := ethutil.NewValue(node)
	if node == nil || n.Len() == 0 {
		newNode := []interface{}{CompactEncode(key), value}

		return t.Put(newNode)
	}

	currentNode := t.getNode(node)
	// Check for "special" 2 slice type node
	if currentNode.Len() == 2 {
		// Decode the key

		k := CompactDecode(currentNode.Get(0).Str())
		v := currentNode.Get(1).Raw()

		// Matching key pair (ie. there's already an object with this key)
		if bytes.Equal(k, key) { //CompareIntSlice(k, key) {
			newNode := []interface{}{CompactEncode(key), value}
			return t.Put(newNode)
		}

		var newHash interface{}
		matchingLength := MatchingNibbleLength(key, k)
		if matchingLength == len(k) {
			// Insert the hash, creating a new node
			newHash = t.InsertState(v, key[matchingLength:], value)
		} else {
			// Expand the 2 length slice to a 17 length slice
			oldNode := t.InsertState("", k[matchingLength+1:], v)
			newNode := t.InsertState("", key[matchingLength+1:], value)
			// Create an expanded slice
			scaledSlice := EmptyStringSlice(17)
			// Set the copied and new node
			scaledSlice[k[matchingLength]] = oldNode
			scaledSlice[key[matchingLength]] = newNode

			newHash = t.Put(scaledSlice)
		}

		if matchingLength == 0 {
			// End of the chain, return
			return newHash
		} else {
			newNode := []interface{}{CompactEncode(key[:matchingLength]), newHash}
			return t.Put(newNode)
		}
	} else {

		// Copy the current node over to the new node and replace the first nibble in the key
		newNode := EmptyStringSlice(17)

		for i := 0; i < 17; i++ {
			cpy := currentNode.Get(i).Raw()
			if cpy != nil {
				newNode[i] = cpy
			}
		}

		newNode[key[0]] = t.InsertState(currentNode.Get(int(key[0])).Raw(), key[1:], value)

		return t.Put(newNode)
	}

	panic("unexpected end")
}

func (t *Trie) deleteState(node interface{}, key []byte) interface{} {
	if len(key) == 0 {
		return ""
	}

	// New node
	n := ethutil.NewValue(node)
	//if node == nil || (n.Type() == reflect.String && (n.Str() == "" || n.Get(0).IsNil())) || n.Len() == 0 {
	if node == nil || n.Len() == 0 {
		//return nil
		//fmt.Printf("<empty ret> %x %d\n", n, len(n.Bytes()))

		return ""
	}

	currentNode := t.getNode(node)
	// Check for "special" 2 slice type node
	if currentNode.Len() == 2 {
		// Decode the key
		k := CompactDecode(currentNode.Get(0).Str())
		v := currentNode.Get(1).Raw()

		// Matching key pair (ie. there's already an object with this key)
		if bytes.Equal(k, key) { //CompareIntSlice(k, key) {
			//fmt.Printf("<delete ret> %x\n", v)

			return ""
		} else if bytes.Equal(key[:len(k)], k) { //CompareIntSlice(key[:len(k)], k) {
			hash := t.deleteState(v, key[len(k):])
			child := t.getNode(hash)

			var newNode []interface{}
			if child.Len() == 2 {
				newKey := append(k, CompactDecode(child.Get(0).Str())...)
				newNode = []interface{}{CompactEncode(newKey), child.Get(1).Raw()}
			} else {
				newNode = []interface{}{currentNode.Get(0).Str(), hash}
			}

			//fmt.Printf("%x\n", newNode)

			return t.Put(newNode)
		} else {
			return node
		}
	} else {
		// Copy the current node over to the new node and replace the first nibble in the key
		n := EmptyStringSlice(17)
		var newNode []interface{}

		for i := 0; i < 17; i++ {
			cpy := currentNode.Get(i).Raw()
			if cpy != nil {
				n[i] = cpy
			}
		}

		n[key[0]] = t.deleteState(n[key[0]], key[1:])
		amount := -1
		for i := 0; i < 17; i++ {
			if n[i] != "" {
				if amount == -1 {
					amount = i
				} else {
					amount = -2
				}
			}
		}
		if amount == 16 {
			newNode = []interface{}{CompactEncode([]byte{16}), n[amount]}
		} else if amount >= 0 {
			child := t.getNode(n[amount])
			if child.Len() == 17 {
				newNode = []interface{}{CompactEncode([]byte{byte(amount)}), n[amount]}
			} else if child.Len() == 2 {
				key := append([]byte{byte(amount)}, CompactDecode(child.Get(0).Str())...)
				newNode = []interface{}{CompactEncode(key), child.Get(1).Str()}
			}

		} else {
			newNode = n
		}

		//fmt.Printf("%x\n", newNode)
		return t.Put(newNode)
	}

	panic("unexpected return")
}

type TrieIterator struct {
	trie  *Trie
	key   string
	value string

	shas   [][]byte
	values []string

	lastNode []byte
}

func (t *Trie) NewIterator() *TrieIterator {
	return &TrieIterator{trie: t}
}

func (self *Trie) Iterator() *Iterator {
	return NewIterator(self)
}

// Some time in the near future this will need refactoring :-)
// XXX Note to self, IsSlice == inline node. Str == sha3 to node
func (it *TrieIterator) workNode(currentNode *ethutil.Value) {
	if currentNode.Len() == 2 {
		k := CompactDecode(currentNode.Get(0).Str())

		if currentNode.Get(1).Str() == "" {
			it.workNode(currentNode.Get(1))
		} else {
			if k[len(k)-1] == 16 {
				it.values = append(it.values, currentNode.Get(1).Str())
			} else {
				it.shas = append(it.shas, currentNode.Get(1).Bytes())
				it.getNode(currentNode.Get(1).Bytes())
			}
		}
	} else {
		for i := 0; i < currentNode.Len(); i++ {
			if i == 16 && currentNode.Get(i).Len() != 0 {
				it.values = append(it.values, currentNode.Get(i).Str())
			} else {
				if currentNode.Get(i).Str() == "" {
					it.workNode(currentNode.Get(i))
				} else {
					val := currentNode.Get(i).Str()
					if val != "" {
						it.shas = append(it.shas, currentNode.Get(1).Bytes())
						it.getNode([]byte(val))
					}
				}
			}
		}
	}
}

func (it *TrieIterator) getNode(node []byte) {
	currentNode := it.trie.cache.Get(node)
	it.workNode(currentNode)
}

func (it *TrieIterator) Collect() [][]byte {
	if it.trie.Root == "" {
		return nil
	}

	it.getNode(ethutil.NewValue(it.trie.Root).Bytes())

	return it.shas
}

func (it *TrieIterator) Purge() int {
	shas := it.Collect()
	for _, sha := range shas {
		it.trie.cache.Delete(sha)
	}
	return len(it.values)
}

func (it *TrieIterator) Key() string {
	return ""
}

func (it *TrieIterator) Value() string {
	return ""
}

type EachCallback func(key string, node *ethutil.Value)

func (it *TrieIterator) Each(cb EachCallback) {
	it.fetchNode(nil, ethutil.NewValue(it.trie.Root).Bytes(), cb)
}

func (it *TrieIterator) fetchNode(key []byte, node []byte, cb EachCallback) {
	it.iterateNode(key, it.trie.cache.Get(node), cb)
}

func (it *TrieIterator) iterateNode(key []byte, currentNode *ethutil.Value, cb EachCallback) {
	if currentNode.Len() == 2 {
		k := CompactDecode(currentNode.Get(0).Str())

		pk := append(key, k...)
		if currentNode.Get(1).Len() != 0 && currentNode.Get(1).Str() == "" {
			it.iterateNode(pk, currentNode.Get(1), cb)
		} else {
			if k[len(k)-1] == 16 {
				cb(DecodeCompact(pk), currentNode.Get(1))
			} else {
				it.fetchNode(pk, currentNode.Get(1).Bytes(), cb)
			}
		}
	} else {
		for i := 0; i < currentNode.Len(); i++ {
			pk := append(key, byte(i))
			if i == 16 && currentNode.Get(i).Len() != 0 {
				cb(DecodeCompact(pk), currentNode.Get(i))
			} else {
				if currentNode.Get(i).Len() != 0 && currentNode.Get(i).Str() == "" {
					it.iterateNode(pk, currentNode.Get(i), cb)
				} else {
					val := currentNode.Get(i).Str()
					if val != "" {
						it.fetchNode(pk, []byte(val), cb)
					}
				}
			}
		}
	}
}
*/
