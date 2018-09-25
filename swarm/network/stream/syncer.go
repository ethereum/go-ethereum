// Copyright 2018 The go-ethereum Authors
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

package stream

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	BatchSize = 128
)

// SwarmSyncerServer implements an Server for history syncing on bins
// offered streams:
// * live request delivery with or without checkback
// * (live/non-live historical) chunk syncing per proximity bin
type SwarmSyncerServer struct {
	po        uint8
	store     storage.SyncChunkStore
	sessionAt uint64
	start     uint64
	live      bool
	quit      chan struct{}
}

// NewSwarmSyncerServer is contructor for SwarmSyncerServer
func NewSwarmSyncerServer(live bool, po uint8, syncChunkStore storage.SyncChunkStore) (*SwarmSyncerServer, error) {
	sessionAt := syncChunkStore.BinIndex(po)
	var start uint64
	if live {
		start = sessionAt
	}
	return &SwarmSyncerServer{
		po:        po,
		store:     syncChunkStore,
		sessionAt: sessionAt,
		start:     start,
		live:      live,
		quit:      make(chan struct{}),
	}, nil
}

func RegisterSwarmSyncerServer(streamer *Registry, syncChunkStore storage.SyncChunkStore) {
	streamer.RegisterServerFunc("SYNC", func(p *Peer, t string, live bool) (Server, error) {
		po, err := ParseSyncBinKey(t)
		if err != nil {
			return nil, err
		}
		return NewSwarmSyncerServer(live, po, syncChunkStore)
	})
	// streamer.RegisterServerFunc(stream, func(p *Peer) (Server, error) {
	// 	return NewOutgoingProvableSwarmSyncer(po, db)
	// })
}

// Close needs to be called on a stream server
func (s *SwarmSyncerServer) Close() {
	close(s.quit)
}

// GetData retrieves the actual chunk from netstore
func (s *SwarmSyncerServer) GetData(ctx context.Context, key []byte) ([]byte, error) {
	chunk, err := s.store.Get(ctx, storage.Address(key))
	if err != nil {
		return nil, err
	}
	return chunk.Data(), nil
}

// GetBatch retrieves the next batch of hashes from the dbstore
func (s *SwarmSyncerServer) SetNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	var batch []byte
	i := 0
	if s.live {
		if from == 0 {
			from = s.start
		}
		if to <= from || from >= s.sessionAt {
			to = math.MaxUint64
		}
	} else {
		if (to < from && to != 0) || from > s.sessionAt {
			return nil, 0, 0, nil, nil
		}
		if to == 0 || to > s.sessionAt {
			to = s.sessionAt
		}
	}

	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()
	var wait bool
	for {
		if wait {
			if ticker == nil {
				ticker = time.NewTicker(1000 * time.Millisecond)
			}
			select {
			case <-ticker.C:
			case <-s.quit:
				return nil, 0, 0, nil, nil
			}
		}

		metrics.GetOrRegisterCounter("syncer.setnextbatch.iterator", nil).Inc(1)
		err := s.store.Iterator(from, to, s.po, func(key storage.Address, idx uint64) bool {
			batch = append(batch, key[:]...)
			i++
			to = idx
			return i < BatchSize
		})
		if err != nil {
			return nil, 0, 0, nil, err
		}
		if len(batch) > 0 {
			break
		}
		wait = true
	}

	log.Trace("Swarm syncer offer batch", "po", s.po, "len", i, "from", from, "to", to, "current store count", s.store.BinIndex(s.po))
	return batch, from, to, nil, nil
}

// SwarmSyncerClient
type SwarmSyncerClient struct {
	sessionAt     uint64
	nextC         chan struct{}
	sessionRoot   storage.Address
	sessionReader storage.LazySectionReader
	retrieveC     chan *storage.Chunk
	storeC        chan *storage.Chunk
	store         storage.SyncChunkStore
	// chunker               storage.Chunker
	currentRoot storage.Address
	requestFunc func(chunk *storage.Chunk)
	end, start  uint64
	peer        *Peer
	stream      Stream
}

// NewSwarmSyncerClient is a contructor for provable data exchange syncer
func NewSwarmSyncerClient(p *Peer, store storage.SyncChunkStore, stream Stream) (*SwarmSyncerClient, error) {
	return &SwarmSyncerClient{
		store:  store,
		peer:   p,
		stream: stream,
	}, nil
}

// // NewIncomingProvableSwarmSyncer is a contructor for provable data exchange syncer
// func NewIncomingProvableSwarmSyncer(po int, priority int, index uint64, sessionAt uint64, intervals []uint64, sessionRoot storage.Address, chunker *storage.PyramidChunker, store storage.ChunkStore, p Peer) *SwarmSyncerClient {
// 	retrieveC := make(storage.Chunk, chunksCap)
// 	RunChunkRequestor(p, retrieveC)
// 	storeC := make(storage.Chunk, chunksCap)
// 	RunChunkStorer(store, storeC)
// 	s := &SwarmSyncerClient{
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
// 	return s
// }

// // StartSyncing is called on the Peer to start the syncing process
// // the idea is that it is called only after kademlia is close to healthy
// func StartSyncing(s *Streamer, peerId enode.ID, po uint8, nn bool) {
// 	lastPO := po
// 	if nn {
// 		lastPO = maxPO
// 	}
//
// 	for i := po; i <= lastPO; i++ {
// 		s.Subscribe(peerId, "SYNC", newSyncLabel("LIVE", po), 0, 0, High, true)
// 		s.Subscribe(peerId, "SYNC", newSyncLabel("HISTORY", po), 0, 0, Mid, false)
// 	}
// }

// RegisterSwarmSyncerClient registers the client constructor function for
// to handle incoming sync streams
func RegisterSwarmSyncerClient(streamer *Registry, store storage.SyncChunkStore) {
	streamer.RegisterClientFunc("SYNC", func(p *Peer, t string, live bool) (Client, error) {
		return NewSwarmSyncerClient(p, store, NewStream("SYNC", t, live))
	})
}

// NeedData
func (s *SwarmSyncerClient) NeedData(ctx context.Context, key []byte) (wait func(context.Context) error) {
	return s.store.FetchFunc(ctx, key)
}

// BatchDone
func (s *SwarmSyncerClient) BatchDone(stream Stream, from uint64, hashes []byte, root []byte) func() (*TakeoverProof, error) {
	// TODO: reenable this with putter/getter refactored code
	// if s.chunker != nil {
	// 	return func() (*TakeoverProof, error) { return s.TakeoverProof(stream, from, hashes, root) }
	// }
	return nil
}

func (s *SwarmSyncerClient) TakeoverProof(stream Stream, from uint64, hashes []byte, root storage.Address) (*TakeoverProof, error) {
	// for provable syncer currentRoot is non-zero length
	// TODO: reenable this with putter/getter
	// if s.chunker != nil {
	// 	if from > s.sessionAt { // for live syncing currentRoot is always updated
	// 		//expRoot, err := s.chunker.Append(s.currentRoot, bytes.NewReader(hashes), s.retrieveC, s.storeC)
	// 		expRoot, _, err := s.chunker.Append(s.currentRoot, bytes.NewReader(hashes), s.retrieveC)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if !bytes.Equal(root, expRoot) {
	// 			return nil, fmt.Errorf("HandoverProof mismatch")
	// 		}
	// 		s.currentRoot = root
	// 	} else {
	// 		expHashes := make([]byte, len(hashes))
	// 		_, err := s.sessionReader.ReadAt(expHashes, int64(s.end*HashSize))
	// 		if err != nil && err != io.EOF {
	// 			return nil, err
	// 		}
	// 		if !bytes.Equal(expHashes, hashes) {
	// 			return nil, errors.New("invalid proof")
	// 		}
	// 	}
	// 	return nil, nil
	// }
	s.end += uint64(len(hashes)) / HashSize
	takeover := &Takeover{
		Stream: stream,
		Start:  s.start,
		End:    s.end,
		Root:   root,
	}
	// serialise and sign
	return &TakeoverProof{
		Takeover: takeover,
		Sig:      nil,
	}, nil
}

func (s *SwarmSyncerClient) Close() {}

// base for parsing and formating sync bin key
// it must be 2 <= base <= 36
const syncBinKeyBase = 36

// FormatSyncBinKey returns a string representation of
// Kademlia bin number to be used as key for SYNC stream.
func FormatSyncBinKey(bin uint8) string {
	return strconv.FormatUint(uint64(bin), syncBinKeyBase)
}

// ParseSyncBinKey parses the string representation
// and returns the Kademlia bin number.
func ParseSyncBinKey(s string) (uint8, error) {
	bin, err := strconv.ParseUint(s, syncBinKeyBase, 8)
	if err != nil {
		return 0, err
	}
	return uint8(bin), nil
}
