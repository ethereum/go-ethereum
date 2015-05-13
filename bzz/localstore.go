// localstore.go
package bzz

type localStore struct {
	memStore *memStore
	dbStore  *dbStore
}

// localStore is itself a chunk store , to stores a chunk only
// its integrity is checked ?
func (self *localStore) Put(chunk *Chunk) {
	chunk.dbStored = make(chan bool)
	self.memStore.Put(chunk)
	if chunk.wg != nil {
		chunk.wg.Add(1)
	}
	go func() {
		self.dbStore.Put(chunk)
		if chunk.wg != nil {
			chunk.wg.Done()
		}
	}()
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved so additional timeout is needed to wrap this call
func (self *localStore) Get(key Key) (chunk *Chunk, err error) {
	chunk, err = self.memStore.Get(key)
	if err == nil {
		return
	}
	chunk, err = self.dbStore.Get(key)
	if err != nil {
		return
	}
	self.memStore.Put(chunk)
	return
}
