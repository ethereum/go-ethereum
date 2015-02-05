package trie

type Backend interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte)
}

type Cache struct {
	store   map[string][]byte
	backend Backend
}

func NewCache(backend Backend) *Cache {
	return &Cache{make(map[string][]byte), backend}
}

func (self *Cache) Get(key []byte) []byte {
	data := self.store[string(key)]
	if data == nil {
		data, _ = self.backend.Get(key)
	}

	return data
}

func (self *Cache) Put(key []byte, data []byte) {
	self.store[string(key)] = data
}

func (self *Cache) Flush() {
	for k, v := range self.store {
		self.backend.Put([]byte(k), v)
	}

	// This will eventually grow too large. We'd could
	// do a make limit on storage and push out not-so-popular nodes.
	//self.Reset()
}

func (self *Cache) Copy() *Cache {
	cache := NewCache(self.backend)
	for k, v := range self.store {
		cache.store[k] = v
	}
	return cache
}

func (self *Cache) Reset() {
	//self.store = make(map[string][]byte)
}
