package ethutil

import (
	"fmt"
	"reflect"
)

type Node struct {
	Key   []byte
	Value *Value
	Dirty bool
}

func NewNode(key []byte, val *Value, dirty bool) *Node {
	return &Node{Key: key, Value: val, Dirty: dirty}
}

func (n *Node) Copy() *Node {
	return NewNode(n.Key, n.Value, n.Dirty)
}

type Cache struct {
	nodes   map[string]*Node
	db      Database
	IsDirty bool
}

func NewCache(db Database) *Cache {
	return &Cache{db: db, nodes: make(map[string]*Node)}
}

func (cache *Cache) Put(v interface{}) interface{} {
	value := NewValue(v)

	enc := value.Encode()
	if len(enc) >= 32 {
		sha := Sha3Bin(enc)

		cache.nodes[string(sha)] = NewNode(sha, value, true)
		cache.IsDirty = true

		return sha
	}

	return v
}

func (cache *Cache) Get(key []byte) *Value {
	// First check if the key is the cache
	if cache.nodes[string(key)] != nil {
		return cache.nodes[string(key)].Value
	}

	// Get the key of the database instead and cache it
	data, _ := cache.db.Get(key)
	// Create the cached value
	value := NewValueFromBytes(data)
	// Create caching node
	cache.nodes[string(key)] = NewNode(key, value, false)

	return value
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
	Root interface{}
	//db   Database
	cache *Cache
}

func NewTrie(db Database, Root interface{}) *Trie {
	return &Trie{cache: NewCache(db), Root: Root}
}

// Save the cached value to the database.
func (t *Trie) Sync() {
	t.cache.Commit()
}

func (t *Trie) Undo() {
	t.cache.Undo()
}

/*
 * Public (query) interface functions
 */
func (t *Trie) Update(key string, value string) {
	k := CompactHexDecode(key)

	t.Root = t.UpdateState(t.Root, k, value)
}

func (t *Trie) Get(key string) string {
	k := CompactHexDecode(key)
	c := NewValue(t.GetState(t.Root, k))

	return c.Str()
}

func (t *Trie) GetState(node interface{}, key []int) interface{} {
	n := NewValue(node)
	// Return the node if key is empty (= found)
	if len(key) == 0 || n.IsNil() || n.Len() == 0 {
		return node
	}

	currentNode := t.GetNode(node)
	length := currentNode.Len()

	if length == 0 {
		return ""
	} else if length == 2 {
		// Decode the key
		k := CompactDecode(currentNode.Get(0).Str())
		v := currentNode.Get(1).Raw()

		if len(key) >= len(k) && CompareIntSlice(k, key[:len(k)]) {
			return t.GetState(v, key[len(k):])
		} else {
			return ""
		}
	} else if length == 17 {
		return t.GetState(currentNode.Get(key[0]).Raw(), key[1:])
	}

	// It shouldn't come this far
	fmt.Println("GetState unexpected return")
	return ""
}

func (t *Trie) GetNode(node interface{}) *Value {
	n := NewValue(node)

	if !n.Get(0).IsNil() {
		return n
	}

	str := n.Str()
	if len(str) == 0 {
		return n
	} else if len(str) < 32 {
		return NewValueFromBytes([]byte(str))
	}

	return t.cache.Get(n.Bytes())
}

func (t *Trie) UpdateState(node interface{}, key []int, value string) interface{} {
	if value != "" {
		return t.InsertState(node, key, value)
	} else {
		// delete it
	}

	return ""
}

func (t *Trie) Put(node interface{}) interface{} {
	/*
		enc := Encode(node)
		if len(enc) >= 32 {
			var sha []byte
			sha = Sha3Bin(enc)
			//t.db.Put([]byte(sha), enc)

			return sha
		}
		return node
	*/

	/*
		TODO?
			c := Conv(t.Root)
			fmt.Println(c.Type(), c.Length())
			if c.Type() == reflect.String && c.AsString() == "" {
				return enc
			}
	*/

	return t.cache.Put(node)

}

func EmptyStringSlice(l int) []interface{} {
	slice := make([]interface{}, l)
	for i := 0; i < l; i++ {
		slice[i] = ""
	}
	return slice
}

func (t *Trie) InsertState(node interface{}, key []int, value interface{}) interface{} {
	if len(key) == 0 {
		return value
	}

	// New node
	n := NewValue(node)
	if node == nil || (n.Type() == reflect.String && (n.Str() == "" || n.Get(0).IsNil())) || n.Len() == 0 {
		newNode := []interface{}{CompactEncode(key), value}

		return t.Put(newNode)
	}

	currentNode := t.GetNode(node)
	// Check for "special" 2 slice type node
	if currentNode.Len() == 2 {
		// Decode the key
		k := CompactDecode(currentNode.Get(0).Str())
		v := currentNode.Get(1).Raw()

		// Matching key pair (ie. there's already an object with this key)
		if CompareIntSlice(k, key) {
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

		newNode[key[0]] = t.InsertState(currentNode.Get(key[0]).Raw(), key[1:], value)

		return t.Put(newNode)
	}

	return ""
}

// Simple compare function which creates a rlp value out of the evaluated objects
func (t *Trie) Cmp(trie *Trie) bool {
	return NewValue(t.Root).Cmp(NewValue(trie.Root))
}

// Returns a copy of this trie
func (t *Trie) Copy() *Trie {
	trie := NewTrie(t.cache.db, t.Root)
	for key, node := range t.cache.nodes {
		trie.cache.nodes[key] = node.Copy()
	}

	return trie
}
