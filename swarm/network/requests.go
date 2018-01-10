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

import "github.com/ethereum/go-ethereum/swarm/storage"

const retrieveRequestStream = "RETRIEVE_REQUEST"

// Intervals is a stream specific history of downloaded intervals
// for historical streams
type Intervals struct {
	streamer *Streamer
	key      string
}

func (s *Intervals) load() error {
	return s.streamer.load(s.key)
}

func (s *Intervals) save() error {
	return s.streamer.save(s.key)
}

func (s *Intervals) get() []uint64 {
	return s.streamer.get(s.key)
}

func (s *Intervals) set(v []uint64) {
	s.streamer.set(s.key, v)
}

func NewIntervals(key string, s *Streamer) *Intervals {
	return &Intervals{
		streamer: s,
		key:      key,
	}
}

// RetrieveRequestStreamer implements OutgoingStreamer
type RetrieveRequestStreamer struct {
	deliveryC  chan *storage.Chunk
	batchC     chan []byte
	db         *DbAccess
	currentLen uint64
}

// RegisterRequestStreamer registers outgoing and incoming streamers for request handling
func RegisterRequestStreamer(streamer *Streamer, db *DbAccess) {
	streamer.RegisterOutgoingStreamer(retrieveRequestStream, func(_ *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		return NewRetrieveRequestStreamer(db), nil
	})
	streamer.RegisterIncomingStreamer(retrieveRequestStream, func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return NewIncomingSwarmSyncer(p, db, nil)
	})
}

// NewRetrieveRequestStreamer is RetrieveRequestStreamer constructor
func NewRetrieveRequestStreamer(db *DbAccess) *RetrieveRequestStreamer {
	s := &RetrieveRequestStreamer{
		deliveryC: make(chan *storage.Chunk),
		batchC:    make(chan []byte),
		db:        db,
	}
	go s.processDeliveries()
	return s
}

// processDeliveries handles delivered chunk hashes
func (s *RetrieveRequestStreamer) processDeliveries() {
	var hashes []byte
	for {
		select {
		case delivery := <-s.deliveryC:
			hashes = append(hashes, delivery.Key[:]...)
		case s.batchC <- hashes:
			hashes = nil
		}
	}
}

// SetNextBatch
func (s *RetrieveRequestStreamer) SetNextBatch(_, _ uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error) {
	hashes = <-s.batchC
	from = s.currentLen
	s.currentLen += uint64(len(hashes))
	to = s.currentLen
	return
}

// GetData retrives chunk data from db store
func (s *RetrieveRequestStreamer) GetData(key []byte) []byte {
	chunk, _ := s.db.get(storage.Key(key))
	return chunk.SData
}
