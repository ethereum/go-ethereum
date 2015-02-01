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

type ChunkStore interface {
	Put(*Chunk) error
	Get(*Chunk) error
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
func (self *DPA) Put(*Chunk) (err error) {
	return
}

// Get(chunk *Chunk) looks up a chunk in the local stores
// This method is blocking until the chunk is retrieved so additional timeout is needed to wrap this call
func (self *DPA) Get(*Chunk) (err error) {
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
				for _, store := range self.Stores {
					if err := store.Get(chunk); err != nil { // no waiting/blocking here
						dpaLogger.DebugDetailf("%v retrieving chunk %x: %v", store, chunk.Key, err)
					} else {
						if !chunk.update {
							break
						}
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
					if err := store.Put(chunk); err != nil { // no waiting/blocking here
						dpaLogger.DebugDetailf("%v storing chunk %x: %v", store, chunk.Key, err)
					} // no waiting/blocking here
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
