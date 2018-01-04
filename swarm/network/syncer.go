// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	batchSize = 128
)

// wrapper of db-s to provide mockable custom local chunk store access to syncer
type DbAccess struct {
	db  *storage.DbStore
	loc *storage.LocalStore
}

func NewDbAccess(loc *storage.LocalStore) *DbAccess {
	return &DbAccess{loc.DbStore.(*storage.DbStore), loc}
}

// to obtain the chunks from key or request db entry only
func (self *DbAccess) get(key storage.Key) (*storage.Chunk, error) {
	return self.loc.Get(key)
}

// current storage counter of chunk db
func (self *DbAccess) currentBucketStorageIndex(po uint8) uint64 {
	return self.db.CurrentBucketStorageIndex(po)
}

// iteration storage counter and proximity order
func (self *DbAccess) iterator(from uint64, to uint64, po uint8, f func(storage.Key, uint64) bool) error {
	return self.db.SyncIterator(from, to, po, f)
}

// OutgoingSwarmSyncer implements an OutgoingStreamer for history syncing on bins
// offered streams:
// * live request delivery with or without checkback
// * (live/non-live historical) chunk syncing per proximity bin
type OutgoingSwarmSyncer struct {
	po           uint8
	db           *DbAccess
	sessionAt    uint64
	currentBatch []byte
	priority     int
}

// NewOutgoingSwarmSyncer is contructor for OutgoingSwarmSyncer
func NewOutgoingSwarmSyncer(po uint8, db *DbAccess) (*OutgoingSwarmSyncer, error) {
	self := &OutgoingSwarmSyncer{
		po:        po,
		db:        db,
		sessionAt: db.currentBucketStorageIndex(po),
	}
	return self, nil
}

const maxPO = 32

func RegisterOutgoingSyncers(streamer *Streamer, db *DbAccess) {
	for po := uint8(0); po < maxPO; po++ {
		stream := Stream(fmt.Sprintf("SYNC-%02d-live", po))
		streamer.RegisterOutgoingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
			return NewOutgoingSwarmSyncer(po, db)
		})
		stream = Stream(fmt.Sprintf("SYNC-%02d-history", po))
		streamer.RegisterOutgoingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
			return NewOutgoingSwarmSyncer(po, db)
		})
		// stream = Stream(fmt.Sprintf("SYNC-%02d-delete", po))
		// streamer.RegisterOutgoingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
		// 	return NewOutgoingProvableSwarmSyncer(po, db)
		// })
	}
}

// GetSection retrieves the actual chunk from localstore
func (self *OutgoingSwarmSyncer) GetData(key []byte) []byte {
	chunk, err := self.db.get(storage.Key(key))
	if err != nil {
		return nil
	}
	return chunk.SData
}

func (self *OutgoingSwarmSyncer) CurrentBatch() []byte {
	return self.currentBatch
}

func (self *OutgoingSwarmSyncer) Priority() int {
	return self.priority
}

// GetBatch retrieves the next batch of hashes from the dbstore
func (self *OutgoingSwarmSyncer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	var batch []byte
	i := 0
	err := self.db.iterator(from, to, self.po, func(key storage.Key, idx uint64) bool {
		batch = append(batch, key[:]...)
		i++
		to = idx
		return i < batchSize
	})
	if err != nil {
		return nil, 0, 0, nil, err
	}
	self.currentBatch = batch
	log.Debug("Swarm batch", "po", self.po, "len", i, "from", from, "to", to)
	return batch, from, to, nil, nil
}

// IncomingSwarmSyncer
type IncomingSwarmSyncer struct {
	po            uint8
	priority      int
	sessionAt     uint64
	nextC         chan struct{}
	intervals     []uint64
	sessionRoot   storage.Key
	sessionReader storage.LazySectionReader
	retrieveC     chan *storage.Chunk
	storeC        chan *storage.Chunk
	store         storage.ChunkStore
	chunker       storage.Chunker
	currentRoot   storage.Key
	end, start    uint64
}

// NewIncomingSwarmSyncer is a contructor for provable data exchange syncer
func NewIncomingSwarmSyncer(po uint8, priority int, intervals []uint64, p Peer, store storage.ChunkStore, chunker storage.Chunker) (*IncomingSwarmSyncer, error) {
	self := &IncomingSwarmSyncer{
		po:        po,
		priority:  priority,
		intervals: intervals,
		store:     store,
		chunker:   chunker,
	}
	return self, nil
}

func (s *IncomingSwarmSyncer) Priority() int {
	return s.priority
}

// // NewIncomingProvableSwarmSyncer is a contructor for provable data exchange syncer
// func NewIncomingProvableSwarmSyncer(po int, priority int, index uint64, sessionAt uint64, intervals []uint64, sessionRoot storage.Key, chunker *storage.PyramidChunker, store storage.ChunkStore, p Peer) *IncomingSwarmSyncer {
// 	retrieveC := make(storage.Chunk, chunksCap)
// 	RunChunkRequestor(p, retrieveC)
// 	storeC := make(storage.Chunk, chunksCap)
// 	RunChunkStorer(store, storeC)
// 	self := &IncomingSwarmSyncer{
// 		po:            po,
// 		priority:      priority,
// 		sessionAt:     sessionAt,
// 		start:         index,
// 		end:           index,
// 		nextC:         make(chan struct{}, 1),
// 		intervals:     intervals,
// 		sessionRoot:   sessionRoot,
// 		sessionReader: chunker.Join(sessionRoot, retrieveC),
// 		retrieveC:     retrieveC,
// 		storeC:        storeC,
// 	}
// 	return self
// }

func RegisterIncomingSyncers(streamer *Streamer, db *DbAccess) {
	for po := uint8(0); po < maxPO; po++ {
		stream := Stream(fmt.Sprintf("SYNC-%02d-live", po))
		streamer.RegisterIncomingStreamer(stream, func(p *StreamerPeer) (IncomingStreamer, error) {
			return NewIncomingSwarmSyncer(po, High, nil, p, nil, nil)
		})
		stream = Stream(fmt.Sprintf("SYNC-%02d-history", po))
		streamer.RegisterIncomingStreamer(stream, func(p *StreamerPeer) (IncomingStreamer, error) {
			//intervals := loadIntervals(p, po, false)
			return NewIncomingSwarmSyncer(po, Mid, nil, p, nil, nil)
		})
		// stream = fmt.Sprintf("SYNC-%02d-delete", po)
		// streamer.RegisterIncomingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
		// 	intervals := loadIntervals(p, po, true)
		// 	return NewIncomingSwarmSyncer(po, Mid, sessionAt, intervals, p)
		// })
	}
}

// NeedData
func (self *IncomingSwarmSyncer) NeedData(key []byte) func() {
	chunk, err := self.store.Get(key)
	if err == nil {
		if chunk.SData == nil {
			// send a request instead
			return nil
		}
	}
	// create request and wait until the chunk data arrives and is stored
	return chunk.WaitToStore
}

// NextBatch adjusts the indexes by inspecting the intervals
func (self *IncomingSwarmSyncer) NextBatch(from uint64) (nextFrom uint64, nextTo uint64) {
	if self.intervals[0] >= self.sessionAt { // live syncing
		nextFrom = from
		self.intervals[1] = from
	} else if from >= self.sessionAt { // history sync complete
		self.intervals = nil
	} else if len(self.intervals) > 2 && from >= self.intervals[2] { // filled a gap in the intervals
		self.intervals = append(self.intervals[:1], self.intervals[3:]...)
		nextFrom = self.intervals[1]
		if len(self.intervals) > 2 {
			nextTo = self.intervals[2]
		} else {
			nextTo = self.sessionAt
		}
	} else {
		nextFrom = from
		self.intervals[1] = from
		nextTo = self.sessionAt
	}
	return nextFrom, nextTo
}

//
func (self *IncomingSwarmSyncer) TakeoverProof(s Stream, from uint64, hashes []byte, root storage.Key) (*TakeoverProof, error) {
	// for provable syncer currentRoot is non-zero length
	if self.chunker != nil {
		if from > self.sessionAt { // for live syncing currentRoot is always updated
			//expRoot, err := self.chunker.Append(self.currentRoot, bytes.NewReader(hashes), self.retrieveC, self.storeC)
			expRoot, _, err := self.chunker.Append(self.currentRoot, bytes.NewReader(hashes), self.retrieveC)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(root, expRoot) {
				return nil, fmt.Errorf("HandoverProof mismatch")
			}
			self.currentRoot = root
		} else {
			expHashes := make([]byte, len(hashes))
			_, err := self.sessionReader.ReadAt(expHashes, int64(self.end*HashSize))
			if err != nil && err != io.EOF {
				return nil, err
			}
			if !bytes.Equal(expHashes, hashes) {
				return nil, errors.New("invalid proof")
			}
		}
		return nil, nil
	}
	self.end += uint64(len(hashes)) / HashSize
	takeover := &Takeover{
		Stream: s,
		Start:  self.start,
		End:    self.end,
		Root:   root,
	}
	// serialise and sign
	return &TakeoverProof{
		Takeover: takeover,
		Sig:      nil,
	}, nil
}
