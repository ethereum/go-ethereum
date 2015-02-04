package bzz

import (
	"errors"
	"sync"
	// "time"

	ethlogger "github.com/ethereum/go-ethereum/logger"
	// "github.com/ethereum/go-ethereum/rlp"
)

/*
DPA provides the client API entrypoints Store and Retrieve to store and retrieve
It can store anything that has a byte slice representation, so files or serialised objects etc.
Storage: DPA calls the Chunker to segment the input datastream of any size to a merkle hashed tree of chunks. The key of the root block is returned to the client.
Retrieval: given the key of the root block, the DPA retrieves the block chunks and reconstructs the original data and passes it back as a lazy reader. A lazy reader is a reader with on-demand delayed processing, i.e. the chunks needed to reconstruct a large file are only fetched and processed if that particular part of the document is actually read.

As the chunker produces chunks, DPA dispatches them to the chunk stores for storage or retrieval. The chunk stores are typically sequenced as memory cache, local disk/db store, cloud/distributed/dht storage. Storage requests will reach to all 3 components while retrieval requests stop after the first successful retrieval.
*/

const (
	storeChanCapacity    = 100
	retrieveChanCapacity = 100
)

var (
	notFound = errors.New("not found")
)

var dpaLogger = ethlogger.NewLogger("BZZ")

type DPA struct {
	Chunker   Chunker
	Stores    []ChunkStore
	storeC    chan *Chunk
	retrieveC chan *Chunk

	lock    sync.Mutex
	running bool
	wg      sync.WaitGroup
	quitC   chan bool
}

// Chunk serves also serves as a request object passed to ChunkStores
// in case it is a retrieval request, Data is nil and Size is 0
// Note that Size is not the size of the data chunk, which is Data.Size() see SectionReader
// but the size of the subtree encoded in the chunk
// 0 if request, to be supplied by the dpa
type Chunk struct {
	Reader SectionReader  // nil if request, to be supplied by dpa
	Data   []byte         // nil if request, to be supplied by dpa
	Size   int64          // size of the data covered by the subtree encoded in this chunk
	Key    Key            // always
	C      chan bool      // to signal data delivery by the dpa
	req    *requestStatus //
}

type ChunkStore interface {
	Put(*Chunk) // effectively there is no error even if there is no error
	Get(Key) (*Chunk, error)
}

func (self *DPA) Retrieve(key Key) (data LazySectionReader, err error) {
	dpaLogger.Debugf("bzz honey retrieve")
	reader, errC := self.Chunker.Join(key, self.retrieveC)
	data = reader
	// we can add subscriptions etc. or timeout here
	go func() {
	LOOP:
		for {
			select {
			case err, ok := <-errC:
				if err != nil {
					dpaLogger.Warnf("%v", err)
				}
				if !ok {
					break LOOP
				}
			case <-self.quitC:
				return
			}
		}
	}()

	return
}

func (self *DPA) Store(data SectionReader) (key Key, err error) {

	dpaLogger.Debugf("bzz honey store")

	errC := self.Chunker.Split(key, data, self.storeC)

	go func() {
	LOOP:
		for {
			select {
			case err, ok := <-errC:
				dpaLogger.Warnf("%v", err)
				if !ok {
					break LOOP
				}

			case <-self.quitC:
				break LOOP
			}
		}
	}()
	return

}

// DPA is itself a chunk store , to stores a chunk only
// its integrity is checked ?
func (self *DPA) Put(chunk *Chunk) {
	// rely on storeC
	return
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved so additional timeout is needed to wrap this call
func (self *DPA) Get(key Key) (chunk *Chunk, err error) {
	// rely on retrieveC
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

func (self *DPA) retrieveLoop() {
	self.retrieveC = make(chan *Chunk, retrieveChanCapacity)

	go func() {
	LOOP:
		for chunk := range self.retrieveC {
			go func() {
				for i, store := range self.Stores {
					storedChunk, err := store.Get(chunk.Key)
					if err == notFound {
						dpaLogger.DebugDetailf("%v retrieving chunk %x: NOT FOUND", store, chunk.Key)
						return
					}
					if err != nil {
						dpaLogger.DebugDetailf("%v retrieving chunk %x: %v", store, chunk.Key, err)
						return
					}
					chunk.Reader = NewChunkReaderFromBytes(storedChunk.Data)
					chunk.Size = storedChunk.Size
					close(chunk.C)
					// if not in cache, cache it in memstore
					if i > 0 {
						self.Stores[0].Put(chunk)
					}
				}
			}()
			select {
			case <-self.quitC:
				break LOOP
			default:
			}
		}
	}()
}

func (self *DPA) storeLoop() {
	self.storeC = make(chan *Chunk)
	go func() {
	LOOP:
		for chunk := range self.storeC {
			go func() {
				for _, store := range self.Stores {
					store.Put(chunk)
					// no waiting/blocking here
				}
			}()
			select {
			case <-self.quitC:
				break LOOP
			default:
			}
		}
	}()
}
