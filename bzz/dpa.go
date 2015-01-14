package bzz

import (
	"sync"
	// "time"

	ethlogger "github.com/ethereum/go-ethereum/logger"
	// "github.com/ethereum/go-ethereum/rlp"
)

/*
DPA provides the client API entrypoints Store and Retrieve to store and retrieve
It can store anything that has a byte slice representation, so files or serialised objects etc.
Storage: DPA calls the Chunker to segment the input datastream of any size to a merkle hashed tree of blocks. The key of the root block is returned to the client.
Retrieval: given the key of the root block, the DPA retrieves the block chunks and reconstructs the original data.

As the chunker produces chunks, DPA dispatches them to the chunk stores for storage or retrieval. The chunk stores are typically sequenced as memory cache, local disk/db store, cloud/distributed/dht storage. Storage requests will reach to all 3 components while retrieval requests stop after the first successful retrieval.
*/

const (
	storeChanCapacity    = 100
	retrieveChanCapacity = 100
)

var dpaLogger = ethlogger.NewLogger("BZZ")

type DPA struct {
	Chunker   Chunker
	Stores    []ChunkStore
	wg        sync.WaitGroup
	quitC     chan bool
	storeC    chan *Chunk
	retrieveC chan *Chunk
}

type ChunkStore interface {
	Put(*Chunk) error
	Get(*Chunk) error
}

/*
convenience methods to help convert various typical data inputs to the canonical input to DPA storage: SectionReader
BytesToReader(data []byte) (SectionReader, error)
*/

// func BytesToReader(data []byte) (SectionReader, error) {
// 	return NewChunkReaderFromBytes(data), nil
// }

// func AnythingToReader(data interface{}) (SectionReader, error) {
// 	return NewChunkReaderFromBytes(rlp.Encode(data)), nil
// }

func (self *DPA) retrieveLoop() {
	self.retrieveC = make(chan *Chunk, retrieveChanCapacity)

	go func() {
	LOOP:
		for chunk := range self.retrieveC {
			for _, store := range self.Stores {
				if err := store.Get(chunk); err != nil { // no waiting/blocking here
					dpaLogger.DebugDetailf("%v retrieving chunk %x: %v", store, chunk.Key, err)
				}
			}
			select {
			case <-self.quitC:
				break LOOP
			default:
			}
		}
	}()
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

func (self *DPA) storeLoop() {
	self.storeC = make(chan *Chunk)
	go func() {
	LOOP:
		for chunk := range self.storeC {
			for _, store := range self.Stores {
				if err := store.Put(chunk); err != nil { // no waiting/blocking here
					dpaLogger.DebugDetailf("%v storing chunk %x: %v", store, chunk.Key, err)
				} // no waiting/blocking here
			}
			select {
			case <-self.quitC:
				break LOOP
			default:
			}
		}
	}()
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
