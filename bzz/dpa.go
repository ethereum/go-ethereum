package bzz

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
DPA provides the client API entrypoints Store and Retrieve to store and retrieve
It can store anything that has a byte slice representation, so files or serialised objects etc.

Storage: DPA calls the Chunker to segment the input datastream of any size to a merkle hashed tree of chunks. The key of the root block is returned to the client.

Retrieval: given the key of the root block, the DPA retrieves the block chunks and reconstructs the original data and passes it back as a lazy reader. A lazy reader is a reader with on-demand delayed processing, i.e. the chunks needed to reconstruct a large file are only fetched and processed if that particular part of the document is actually read.

As the chunker produces chunks, DPA dispatches them to the chunk store for storage or retrieval.

The ChunkStore interface is implemented by :

- memStore: a memory cache
- dbStore: local disk/db store
- localStore: a combination (sequence of) memStore and dbStore
- netStore: dht storage
*/

const (
	storeChanCapacity    = 100
	retrieveChanCapacity = 100
)

var (
	notFound = errors.New("not found")
)

var dpaLogger = logger.NewLogger("BZZ")

type DPA struct {
	Chunker    Chunker
	ChunkStore ChunkStore
	storeC     chan *Chunk
	retrieveC  chan *Chunk

	lock    sync.Mutex
	running bool
	wg      *sync.WaitGroup
	quitC   chan bool
}

// Chunk also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size()
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	SData    []byte          // nil if request, to be supplied by dpa
	Size     int64           // size of the data covered by the subtree encoded in this chunk
	Key      Key             // always
	C        chan bool       // to signal data delivery by the dpa
	req      *requestStatus  // request Status needed by netStore
	wg       *sync.WaitGroup // wg to synchronize
	dbStored chan bool       // never remove a chunk from memStore before it is written to dbStore
	source   *peer           // source peer of the chunk
}

type ChunkStore interface {
	Put(*Chunk) // effectively there is no error even if there is an error
	Get(Key) (*Chunk, error)
}

// Public API. Main entry point for document retrieval directly. Used by the
// FS-aware API and httpaccess
// Chunk retrieval blocks on netStore requests with a timeout so reader will
// report error if retrieval of chunks within requested range time out.
func (self *DPA) Retrieve(key Key) SectionReader {
	return self.Chunker.Join(key, self.retrieveC)
}

// Public API. Main entry point for document storage directly. Used by the
// FS-aware API and httpaccess
func (self *DPA) Store(data SectionReader, wg *sync.WaitGroup) (key Key, err error) {
	key = make([]byte, self.Chunker.KeySize())
	errC := self.Chunker.Split(key, data, self.storeC, wg)

SPLIT:
	for {
		select {
		case err, ok := <-errC:
			if err != nil {
				glog.V(logger.Error).Infof("[BZZ] chunker split error: %v", err)
			}
			if !ok {
				break SPLIT
			}

		case <-self.quitC:
			break SPLIT
		}
	}
	return

}

func (self *DPA) Start() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.running {
		return
	}
	self.running = true
	self.quitC = make(chan bool)
	self.storeLoop()
	self.retrieveLoop()
	glog.V(logger.Info).Infof("[BZZ] Swarm DPA started.")
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
// retrieve channel to its ChunkStore  (netStore or localStore)
func (self *DPA) retrieveLoop() {
	self.retrieveC = make(chan *Chunk, retrieveChanCapacity)

	go func() {
	RETRIEVE:
		for ch := range self.retrieveC {

			go func(chunk *Chunk) {
				glog.V(logger.Detail).Infof("[BZZ] dpa: retrieve loop : chunk '%x'", chunk.Key)
				storedChunk, err := self.ChunkStore.Get(chunk.Key)
				if err == notFound {
					glog.V(logger.Detail).Infof("[BZZ] chunk '%x' not found", chunk.Key)
				} else if err != nil {
					glog.V(logger.Detail).Infof("[BZZ] error retrieving chunk %x: %v", chunk.Key, err)
				} else {
					chunk.SData = storedChunk.SData
					chunk.Size = storedChunk.Size
				}
				close(chunk.C)
			}(ch)
			select {
			case <-self.quitC:
				break RETRIEVE
			default:
			}
		}
	}()
}

// storeLoop dispatches the parallel chunk store requests received on the
// store channel to its ChunkStore  (netStore or localStore)
func (self *DPA) storeLoop() {
	self.storeC = make(chan *Chunk)
	go func() {
	STORE:
		for ch := range self.storeC {
			go func(chunk *Chunk) {
				self.ChunkStore.Put(chunk)
				if chunk.wg != nil {
					glog.V(logger.Detail).Infof("[BZZ] DPA.storeLoop %064x", chunk.Key)
					chunk.wg.Done()
				}
			}(ch)
			select {
			case <-self.quitC:
				break STORE
			default:
			}
		}
	}()
}
