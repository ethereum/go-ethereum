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
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const retrieveRequestStream = "RETRIEVE_REQUEST"

type Delivery struct {
	dbAccess *DbAccess
	overlay  Overlay
	receiveC chan *ChunkDeliveryMsg
	getPeer  func(discover.NodeID) *StreamerPeer
	quit     chan struct{}
}

func NewDelivery(overlay Overlay, dbAccess *DbAccess) *Delivery {
	self := &Delivery{
		dbAccess: dbAccess,
		overlay:  overlay,
		receiveC: make(chan *ChunkDeliveryMsg, 10),
	}

	go self.processReceivedChunks()
	return self
}

// RetrieveRequestStreamer implements OutgoingStreamer
type RetrieveRequestStreamer struct {
	deliveryC  chan *storage.Chunk
	batchC     chan []byte
	dbAccess   *DbAccess
	currentLen uint64
}

// NewRetrieveRequestStreamer is RetrieveRequestStreamer constructor
func NewRetrieveRequestStreamer(dbAccess *DbAccess) *RetrieveRequestStreamer {
	s := &RetrieveRequestStreamer{
		deliveryC: make(chan *storage.Chunk),
		batchC:    make(chan []byte),
		dbAccess:  dbAccess,
	}
	go s.processDeliveries()
	return s
}

// processDeliveries handles delivered chunk hashes
func (s *RetrieveRequestStreamer) processDeliveries() {
	var hashes []byte
	var batchC chan []byte
	for {
		select {
		case delivery := <-s.deliveryC:
			hashes = append(hashes, delivery.Key[:]...)
			batchC = s.batchC
		case batchC <- hashes:
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
	chunk, _ := s.dbAccess.get(storage.Key(key))
	return chunk.SData
}

// RetrieveRequestMsg is the protocol msg for chunk retrieve requests
type RetrieveRequestMsg struct {
	Key       storage.Key
	SkipCheck bool
}

func (self *Delivery) handleRetrieveRequestMsg(sp *StreamerPeer, req *RetrieveRequestMsg) error {
	s, err := sp.getOutgoingStreamer(retrieveRequestStream)
	if err != nil {
		return err
	}
	streamer := s.OutgoingStreamer.(*RetrieveRequestStreamer)
	chunk, created := self.dbAccess.getOrCreateRequest(req.Key)
	if chunk.ReqC != nil {
		if created {
			if err := self.RequestFromPeers(chunk.Key[:], false, sp.ID()); err != nil {
				return nil
			}
		}
		go func() {
			t := time.NewTimer(3 * time.Minute)
			defer t.Stop()

			select {
			case <-chunk.ReqC:
			case <-self.quit:
				return
			case <-t.C:
				return
			}

			if req.SkipCheck {
				sp.Deliver(chunk, s.priority)
				return
			}
			streamer.deliveryC <- chunk
		}()
		return nil
	}
	// TODO: call the retrieve function of the outgoing syncer
	if req.SkipCheck {
		return sp.Deliver(chunk, s.priority)
	}
	streamer.deliveryC <- chunk
	return nil
}

type ChunkDeliveryMsg struct {
	Key   storage.Key
	SData []byte // the stored chunk Data (incl size)
}

func (self *Delivery) handleChunkDeliveryMsg(req *ChunkDeliveryMsg) error {
	chunk, err := self.dbAccess.get(req.Key)
	if err != nil {
		return err
	}

	self.receiveC <- req

	log.Trace(fmt.Sprintf("delivery of %v from %v", chunk, self))
	return nil
}

func (self *Delivery) processReceivedChunks() {
	for req := range self.receiveC {
		chunk, err := self.dbAccess.get(req.Key)
		if err != nil {
			continue
		}
		chunk.SData = req.SData
		self.dbAccess.put(chunk)
		close(chunk.ReqC)
	}
}

// RequestFromPeers sends a chunk retrieve request to
func (self *Delivery) RequestFromPeers(hash []byte, skipCheck bool, peersToSkip ...discover.NodeID) error {
	var success bool
	self.overlay.EachConn(hash, 255, func(p OverlayConn, po int, nn bool) bool {
		spId := p.(Peer).ID()
		for _, p := range peersToSkip {
			if p == spId {
				return true
			}
		}
		sp := self.getPeer(spId)
		// TODO: skip light nodes that do not accept retrieve requests
		err := sp.SendPriority(&RetrieveRequestMsg{
			Key:       hash,
			SkipCheck: skipCheck,
		}, Top)
		if err == nil {
			success = true
		}
		return false
	})
	if success {
		return nil
	}
	return errors.New("no peer found")
}
