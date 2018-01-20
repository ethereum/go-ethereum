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
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const swarmChunkServerStreamName = "RETRIEVE_REQUEST"

type Delivery struct {
	db       *storage.DBAPI
	overlay  network.Overlay
	receiveC chan *ChunkDeliveryMsg
	getPeer  func(discover.NodeID) *Peer
	quit     chan struct{}
}

func NewDelivery(overlay network.Overlay, db *storage.DBAPI) *Delivery {
	d := &Delivery{
		db:       db,
		overlay:  overlay,
		receiveC: make(chan *ChunkDeliveryMsg, 10),
	}

	go d.processReceivedChunks()
	return d
}

// SwarmChunkServer implements OutgoingStreamer
type SwarmChunkServer struct {
	deliveryC  chan []byte
	batchC     chan []byte
	db         *storage.DBAPI
	currentLen uint64
}

// NewSwarmChunkServer is SwarmChunkServer constructor
func NewSwarmChunkServer(db *storage.DBAPI) *SwarmChunkServer {
	s := &SwarmChunkServer{
		deliveryC: make(chan []byte),
		batchC:    make(chan []byte),
		db:        db,
	}
	go s.processDeliveries()
	return s
}

// processDeliveries handles delivered chunk hashes
func (s *SwarmChunkServer) processDeliveries() {
	var hashes []byte
	var batchC chan []byte
	for {
		select {
		case hash := <-s.deliveryC:
			hashes = append(hashes, hash...)
			batchC = s.batchC
		case batchC <- hashes:
			hashes = nil
			batchC = nil
		}
	}
}

// SetNextBatch
func (s *SwarmChunkServer) SetNextBatch(_, _ uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error) {
	hashes = <-s.batchC
	from = s.currentLen
	s.currentLen += uint64(len(hashes))
	to = s.currentLen
	return
}

// GetData retrives chunk data from db store
func (s *SwarmChunkServer) GetData(key []byte) []byte {
	chunk, _ := s.db.Get(storage.Key(key))
	return chunk.SData
}

// RetrieveRequestMsg is the protocol msg for chunk retrieve requests
type RetrieveRequestMsg struct {
	Key       storage.Key
	SkipCheck bool
}

func (d *Delivery) handleRetrieveRequestMsg(sp *Peer, req *RetrieveRequestMsg) error {
	s, err := sp.getServer(swarmChunkServerStreamName)
	if err != nil {
		return err
	}
	streamer := s.Server.(*SwarmChunkServer)
	chunk, created := d.db.GetOrCreateRequest(req.Key)
	if chunk.ReqC != nil {
		if created {
			if err := d.RequestFromPeers(chunk.Key[:], false, sp.ID()); err != nil {
				return nil
			}
		}
		go func() {
			t := time.NewTimer(3 * time.Minute)
			defer t.Stop()

			select {
			case <-chunk.ReqC:
			case <-d.quit:
				return
			case <-t.C:
				return
			}

			if req.SkipCheck {
				sp.Deliver(chunk, s.priority)
				return
			}
			streamer.deliveryC <- chunk.Key[:]
		}()
		return nil
	}
	// TODO: call the retrieve function of the outgoing syncer
	if req.SkipCheck {
		return sp.Deliver(chunk, s.priority)
	}
	streamer.deliveryC <- chunk.Key[:]
	return nil
}

type ChunkDeliveryMsg struct {
	Key   storage.Key
	SData []byte // the stored chunk Data (incl size)
}

func (d *Delivery) handleChunkDeliveryMsg(req *ChunkDeliveryMsg) error {
	d.receiveC <- req
	return nil
}

func (d *Delivery) processReceivedChunks() {
	for req := range d.receiveC {
		// this should be has locally
		chunk, err := d.db.Get(req.Key)
		if err == nil && chunk.ReqC == nil {
			continue
		}
		select {
		case <-chunk.ReqC:
		default:
			chunk.SData = req.SData
			d.db.Put(chunk)
			close(chunk.ReqC)
		}
	}
}

// RequestFromPeers sends a chunk retrieve request to
func (d *Delivery) RequestFromPeers(hash []byte, skipCheck bool, peersToSkip ...discover.NodeID) error {
	var success bool
	d.overlay.EachConn(hash, 255, func(p network.OverlayConn, po int, nn bool) bool {
		spId := p.(*network.BzzPeer).ID()
		for _, p := range peersToSkip {
			if p == spId {
				return true
			}
		}
		sp := d.getPeer(spId)
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
