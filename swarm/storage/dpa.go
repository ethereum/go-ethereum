package storage

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
DPA provides the client API entrypoints Store and Retrieve to store and retrieve
It can store anything that has a byte slice representation, so files or serialised objects etc.

Storage: DPA calls the Chunker to segment the input datastream of any size to a merkle hashed tree of chunks. The key of the root block is returned to the client.

Retrieval: given the key of the root block, the DPA retrieves the block chunks and reconstructs the original data and passes it back as a lazy reader. A lazy reader is a reader with on-demand delayed processing, i.e. the chunks needed to reconstruct a large file are only fetched and processed if that particular part of the document is actually read.

As the chunker produces chunks, DPA dispatches them to its own chunk store
implementation for storage or retrieval.
*/

const (
	storeChanCapacity           = 1000
	retrieveChanCapacity        = 1000
	singletonSwarmDbCapacity    = 50000
	singletonSwarmCacheCapacity = 500
	maxStoreProcesses           = 100
	maxRetrieveProcesses        = 100
)

var (
	notFound = errors.New("not found")
)

type DPA struct {
	ChunkStore
	storeC    chan *Chunk
	retrieveC chan *Chunk
	Chunker   Chunker

	lock    sync.Mutex
	running bool
	wg      *sync.WaitGroup
	quitC   chan bool
}

// for testing locally
func NewLocalDPA(datadir string) (*DPA, error) {

	hash := MakeHashFunc("SHA256")

	dbStore, err := NewDbStore(datadir, hash, singletonSwarmDbCapacity, 0)
	if err != nil {
		return nil, err
	}

	return NewDPA(&LocalStore{
		NewMemStore(dbStore, singletonSwarmCacheCapacity),
		dbStore,
	}, NewChunkerParams()), nil
}

func NewDPA(store ChunkStore, params *ChunkerParams) *DPA {
	chunker := NewTreeChunker(params)
	return &DPA{
		Chunker:    chunker,
		ChunkStore: store,
	}
}

// Public API. Main entry point for document retrieval directly. Used by the
// FS-aware API and httpaccess
// Chunk retrieval blocks on netStore requests with a timeout so reader will
// report error if retrieval of chunks within requested range time out.
func (self *DPA) Retrieve(key Key) LazySectionReader {
	return self.Chunker.Join(key, self.retrieveC)
}

// Public API. Main entry point for document storage directly. Used by the
// FS-aware API and httpaccess
func (self *DPA) Store(data io.Reader, size int64, swg *sync.WaitGroup, wwg *sync.WaitGroup) (key Key, err error) {
	return self.Chunker.Split(data, size, self.storeC, swg, wwg)
}

func (self *DPA) Start() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.running {
		return
	}
	self.running = true
	self.retrieveC = make(chan *Chunk, retrieveChanCapacity)
	self.storeC = make(chan *Chunk, storeChanCapacity)
	self.quitC = make(chan bool)
	self.storeLoop()
	self.retrieveLoop()
}

func (self *DPA) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if !self.running {
		return
	}
	self.running = false
	close(self.quitC)
}

// retrieveLoop dispatches the parallel chunk retrieval requests received on the
// retrieve channel to its ChunkStore  (NetStore or LocalStore)
func (self *DPA) retrieveLoop() {
	for i := 0; i < maxRetrieveProcesses; i++ {
		go self.retrieveWorker()
	}
	glog.V(logger.Detail).Infof("[BZZ] dpa: retrieve loop spawning %v workers", maxRetrieveProcesses)
}

func (self *DPA) retrieveWorker() {
	for chunk := range self.retrieveC {
		glog.V(logger.Detail).Infof("[BZZ] dpa: retrieve loop : chunk %v", chunk.Key.Log())
		storedChunk, err := self.Get(chunk.Key)
		if err == notFound {
			glog.V(logger.Detail).Infof("[BZZ] chunk %v not found", chunk.Key.Log())
		} else if err != nil {
			glog.V(logger.Detail).Infof("[BZZ] error retrieving chunk %v: %v", chunk.Key.Log(), err)
		} else {
			chunk.SData = storedChunk.SData
			chunk.Size = storedChunk.Size
		}
		close(chunk.C)

		select {
		case <-self.quitC:
			return
		default:
		}
	}
}

// storeLoop dispatches the parallel chunk store request processors
// received on the store channel to its ChunkStore (NetStore or LocalStore)
func (self *DPA) storeLoop() {
	for i := 0; i < maxStoreProcesses; i++ {
		go self.storeWorker()
	}
	glog.V(logger.Detail).Infof("[BZZ] dpa: store spawning %v workers", maxStoreProcesses)
}

func (self *DPA) storeWorker() {

	for chunk := range self.storeC {
		self.Put(chunk)
		if chunk.wg != nil {
			glog.V(logger.Detail).Infof("[BZZ] dpa: store processor %v", chunk.Key.Log())
			chunk.wg.Done()

		}
		select {
		case <-self.quitC:
			return
		default:
		}
	}
}

// DpaChunkStore implements the ChunkStore interface,
// this chunk access layer assumed 2 chunk stores
// local storage eg. LocalStore and network storage eg., NetStore
// access by calling network is blocking with a timeout

type dpaChunkStore struct {
	n          int
	localStore ChunkStore
	netStore   ChunkStore
}

func NewDpaChunkStore(localStore, netStore ChunkStore) *dpaChunkStore {
	return &dpaChunkStore{0, localStore, netStore}
}

// Get is the entrypoint for local retrieve requests
// waits for response or times out
func (self *dpaChunkStore) Get(key Key) (chunk *Chunk, err error) {
	chunk, err = self.netStore.Get(key)
	// timeout := time.Now().Add(searchTimeout)
	if chunk.SData != nil {
		glog.V(logger.Detail).Infof("[BZZ] DPA.Get: %v found locally, %d bytes", key.Log(), len(chunk.SData))
		return
	}
	// TODO: use self.timer time.Timer and reset with defer disableTimer
	timer := time.After(searchTimeout)
	select {
	case <-timer:
		glog.V(logger.Detail).Infof("[BZZ] DPA.Get: %v request time out ", key.Log())
		err = notFound
	case <-chunk.Req.C:
		glog.V(logger.Detail).Infof("[BZZ] DPA.Get: %v retrieved, %d bytes (%p)", key.Log(), len(chunk.SData), chunk)
	}
	return
}

// Put is the entrypoint for local store requests coming from storeLoop
func (self *dpaChunkStore) Put(entry *Chunk) {
	chunk, err := self.localStore.Get(entry.Key)
	if err != nil {
		glog.V(logger.Detail).Infof("[BZZ] DPA.Put: %v new chunk. call netStore.Put", entry.Key.Log())
		chunk = entry
	} else if chunk.SData == nil {
		glog.V(logger.Detail).Infof("[BZZ] DPA.Put: %v request entry found", entry.Key.Log())
		chunk.SData = entry.SData
		chunk.Size = entry.Size
	} else {
		glog.V(logger.Detail).Infof("[BZZ] DPA.Put: %v chunk already known", entry.Key.Log())
		return
	}
	// from this point on the storage logic is the same with network storage requests
	glog.V(logger.Detail).Infof("[BZZ] DPA.Put %v: %v", self.n, chunk.Key.Log())
	self.n++
	self.netStore.Put(chunk)
}
