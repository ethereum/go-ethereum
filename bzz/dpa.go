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

var dpaLogger = ethlogger.NewLogger("BZZ")

type DPA struct {
	Chunker Chunker
	Stores  []ChunkStore
	wg      sync.WaitGroup
	quitC   chan bool
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

func (self *DPA) Retrieve(key Key) (data LazySectionReader, err error) {
	joinC := make(chan bool)
	reqsC := make(chan bool)
	var requests int

	self.wg.Add(1)
	chunkWg := sync.WaitGroup{}

	chunkWg.Add(1)
	go func() {
		chunkWg.Wait() // wait for all chunk retrieval requests to process
		close(reqsC)   // signal to channel
	}()
	// r is potentially nil pointer, by default chunker allocates
	// a SectionReadWriter based on a byte slice equal to the stored object image's bytes

	dpaLogger.Debugf("bzz honey retrieve")
	reader, chunkC, errC := self.Chunker.Join(key)
	data = reader

	go func() {

		var ok bool

	LOOP:
		for {
			select {

			case chunk, ok := <-chunkC:
				if chunk != nil { // game over
					chunk.wg = chunkWg
					chunkWg.Add(1) // need to call Done by any storage that first retrieves the data
					for _, store := range self.Stores {
						requests++
						if err = store.Get(chunk); err != nil { // no waiting/blocking here
							dpaLogger.DebugDetailf("%v retrieved chunk %x", store, chunk.Key)
							break // the inner loop
						}
					}
				}
				if !ok { // game over but need to continue to see errc still
					chunkC = nil // make it block so no infinite loop
					chunkWg.Done()
				}

			case err, ok = <-errC:
				dpaLogger.Warnf("%v", err)
				if !ok {
					break LOOP
				}
				dpaLogger.DebugDetailf("%v", err)

			case <-reqsC:
				dpaLogger.DebugDetailf("processed all %v chunk retrieval requests for root key %x", requests, key)

			case <-self.quitC:
				break LOOP
			}
		}
		close(joinC)
		self.wg.Done()
	}()

	<-joinC

	return
}

func (self *DPA) Store(data SectionReader) (key Key, err error) {

	dpaLogger.Debugf("bzz honey store")

	chunkC, errC := self.Chunker.Split(key, data)

LOOP:
	for {
		select {

		case chunk, ok := <-chunkC:
			if chunk != nil {
				for _, store := range self.Stores {
					store.Put(chunk) // no waiting/blocking here
				}
			}
			if !ok { // game over but need to continue to see errc still
				chunkC = nil // make it block so no infinite loop
			}

		case err, ok := <-errC:
			dpaLogger.Warnf("%v", err)
			if !ok {
				break LOOP
			}
			dpaLogger.DebugDetailf("%v", err)

		case <-self.quitC:
			break LOOP
		}
	}
	return

}
