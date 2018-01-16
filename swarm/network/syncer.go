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
	"math"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
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

// to obtain the chunks from key or request db entry only
func (self *DbAccess) getOrCreateRequest(key storage.Key) (*storage.Chunk, bool) {
	log.Warn("getOrCreateRequest", "self", self)
	return self.loc.GetOrCreateRequest(key)
}

// to obtain the chunks from key or request db entry only
func (self *DbAccess) put(chunk *storage.Chunk) {
	self.loc.Put(chunk)
}

// OutgoingSwarmSyncer implements an OutgoingStreamer for history syncing on bins
// offered streams:
// * live request delivery with or without checkback
// * (live/non-live historical) chunk syncing per proximity bin
type OutgoingSwarmSyncer struct {
	po        uint8
	db        *DbAccess
	sessionAt uint64
	start     uint64
}

// NewOutgoingSwarmSyncer is contructor for OutgoingSwarmSyncer
func NewOutgoingSwarmSyncer(live bool, po uint8, db *DbAccess) (*OutgoingSwarmSyncer, error) {
	sessionAt := db.currentBucketStorageIndex(po)
	var start uint64
	if live {
		start = sessionAt
	}
	self := &OutgoingSwarmSyncer{
		po:        po,
		db:        db,
		sessionAt: sessionAt,
		start:     start,
	}
	return self, nil
}

const maxPO = 32

func RegisterOutgoingSyncer(streamer *Streamer, db *DbAccess) {
	streamer.RegisterOutgoingStreamer("SYNC", func(p *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		syncType, po := parseSyncLabel(t)
		// TODO: make this work for HISTORY too
		syncType = "LIVE"
		switch syncType {
		case "LIVE":
			return NewOutgoingSwarmSyncer(true, po, db)
		case "HISTORY":
			return NewOutgoingSwarmSyncer(false, po, db)
		default:
			return nil, errors.New("invalid sync type")
		}
	})
	// streamer.RegisterOutgoingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
	// 	return NewOutgoingProvableSwarmSyncer(po, db)
	// })
}

// GetSection retrieves the actual chunk from localstore
func (self *OutgoingSwarmSyncer) GetData(key []byte) []byte {
	chunk, err := self.db.get(storage.Key(key))
	if err != nil {
		return nil
	}
	return chunk.SData
}

// GetBatch retrieves the next batch of hashes from the dbstore
func (self *OutgoingSwarmSyncer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	var batch []byte
	i := 0
	if from == 0 {
		from = self.start
	}
	if to <= from {
		to = math.MaxUint64
	}
	log.Warn("!!!!!!!!!!!!! setNextBatch", "from", from, "to", to, "currentStoreCount", self.db.currentBucketStorageIndex(1))
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		err := self.db.iterator(from, to, self.po, func(key storage.Key, idx uint64) bool {
			batch = append(batch, key[:]...)
			i++
			to = idx
			return i < batchSize
		})
		if err != nil {
			return nil, 0, 0, nil, err
		}
		if len(batch) > 0 {
			break
		}
	}

	log.Debug("Swarm batch", "po", self.po, "len", i, "from", from, "to", to)
	return batch, from, to + 1, nil, nil
}

// IncomingSwarmSyncer
type IncomingSwarmSyncer struct {
	sessionAt     uint64
	nextC         chan struct{}
	sessionRoot   storage.Key
	sessionReader storage.LazySectionReader
	retrieveC     chan *storage.Chunk
	storeC        chan *storage.Chunk
	dbAccess      *DbAccess
	chunker       storage.Chunker
	currentRoot   storage.Key
	requestFunc   func(chunk *storage.Chunk)
	end, start    uint64
}

// NewIncomingSwarmSyncer is a contructor for provable data exchange syncer
func NewIncomingSwarmSyncer(p Peer, dbAccess *DbAccess, chunker storage.Chunker) (*IncomingSwarmSyncer, error) {
	self := &IncomingSwarmSyncer{
		dbAccess: dbAccess,
		chunker:  chunker,
	}
	return self, nil
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

func newSyncLabel(typ string, po uint8) []byte {
	t := []byte(typ)
	t = append(t, byte(po))
	return t
}

func parseSyncLabel(t []byte) (string, uint8) {
	l := len(t) - 1
	return string(t[:l]), uint8(t[l])
}

// StartSyncing is called on the StreamerPeer to start the syncing process
// the idea is that it is called only after kademlia is close to healthy
func StartSyncing(s *Streamer, peerId discover.NodeID, po uint8, nn bool) {
	lastPO := po
	if nn {
		lastPO = maxPO
	}

	for i := po; i <= lastPO; i++ {
		s.Subscribe(peerId, "SYNC", newSyncLabel("LIVE", po), 0, 0, High, true)
		s.Subscribe(peerId, "SYNC", newSyncLabel("HISTORY", po), 0, 0, Mid, false)
	}
}

func RegisterIncomingSyncer(streamer *Streamer, db *DbAccess) {
	streamer.RegisterIncomingStreamer("SYNC", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return NewIncomingSwarmSyncer(p, db, nil)
	})
	// stream = fmt.Sprintf("SYNC-%02d-delete", po)
	// streamer.RegisterIncomingStreamer(stream, func(p *StreamerPeer) (OutgoingStreamer, error) {
	// 	intervals := loadIntervals(p, po, true)
	// 	return NewIncomingSwarmSyncer(po, Mid, sessionAt, intervals, p)
	// })
}

// NeedData
func (self *IncomingSwarmSyncer) NeedData(key []byte) (wait func()) {
	chunk, _ := self.dbAccess.getOrCreateRequest(key)
	// TODO: we may want to request from this peer anyway even if the request exists
	if chunk.ReqC == nil {
		return nil
	}
	// create request and wait until the chunk data arrives and is stored
	return chunk.WaitToStore
}

// BatchDone
func (self *IncomingSwarmSyncer) BatchDone(s string, from uint64, hashes []byte, root []byte) func() (*TakeoverProof, error) {
	if self.chunker != nil {
		return func() (*TakeoverProof, error) { return self.TakeoverProof(s, from, hashes, root) }
	}
	return nil
}

func (self *IncomingSwarmSyncer) TakeoverProof(s string, from uint64, hashes []byte, root storage.Key) (*TakeoverProof, error) {
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
