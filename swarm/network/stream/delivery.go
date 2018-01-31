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
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	swarmChunkServerStreamName = "RETRIEVE_REQUEST"
	deliveryCap                = 32
)

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
		receiveC: make(chan *ChunkDeliveryMsg, deliveryCap),
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
	quit       chan struct{}
}

// NewSwarmChunkServer is SwarmChunkServer constructor
func NewSwarmChunkServer(db *storage.DBAPI) *SwarmChunkServer {
	s := &SwarmChunkServer{
		deliveryC: make(chan []byte, deliveryCap),
		batchC:    make(chan []byte),
		db:        db,
		quit:      make(chan struct{}),
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
		case <-s.quit:
			return
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
	select {
	case hashes = <-s.batchC:
	case <-s.quit:
		return
	}

	from = s.currentLen
	s.currentLen += uint64(len(hashes))
	to = s.currentLen
	return
}

// Close needs to be called on a stream server
func (s *SwarmChunkServer) Close() {
	close(s.quit)
}

// GetData retrives chunk data from db store
func (s *SwarmChunkServer) GetData(key []byte) ([]byte, error) {
	chunk, err := s.db.Get(storage.Key(key))
	if err == storage.ErrFetching {
		<-chunk.ReqC
	} else if err != nil {
		return nil, err
	}
	return chunk.SData, nil
}

// RetrieveRequestMsg is the protocol msg for chunk retrieve requests
type RetrieveRequestMsg struct {
	Key       storage.Key
	SkipCheck bool
}

func (d *Delivery) handleRetrieveRequestMsg(sp *Peer, req *RetrieveRequestMsg) error {
	log.Debug("received request", "peer", sp.ID(), "hash", req.Key)
	s, err := sp.getServer(swarmChunkServerStreamName)
	if err != nil {
		return err
	}
	streamer := s.Server.(*SwarmChunkServer)
	chunk, created := d.db.GetOrCreateRequest(req.Key)
	if chunk.ReqC != nil {
		if created {
			if err := d.RequestFromPeers(chunk.Key[:], false, sp.ID()); err != nil {
				log.Warn("unable to forward chunk request", "peer", sp.ID(), "key", chunk.Key, "err", err)
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
				err := sp.Deliver(chunk, s.priority)
				if err != nil {
					sp.Drop(err)
				}
			}
			streamer.deliveryC <- chunk.Key[:]
		}()
		return nil
	}
	// TODO: call the retrieve function of the outgoing syncer
	if req.SkipCheck {
		log.Trace("deliver", "peer", sp.ID(), "hash", chunk.Key)
		return sp.Deliver(chunk, s.priority)
	}
	streamer.deliveryC <- chunk.Key[:]
	return nil
}

type ChunkDeliveryMsg struct {
	Key   storage.Key
	SData []byte // the stored chunk Data (incl size)
	peer  *Peer  // set in handleChunkDeliveryMsg
}

func (d *Delivery) handleChunkDeliveryMsg(sp *Peer, req *ChunkDeliveryMsg) error {
	req.peer = sp
	d.receiveC <- req
	return nil
}

func (d *Delivery) processReceivedChunks() {
R:
	for req := range d.receiveC {
		// this should be has locally
		chunk, err := d.db.Get(req.Key)
		if !bytes.Equal(chunk.Key, req.Key) {
			panic(fmt.Errorf("processReceivedChunks: chunk key %s != req key %s (peer %s)", chunk.Key.Hex(), storage.Key(req.Key).Hex(), req.peer.ID()))
		}
		if err == nil {
			continue R
		}
		if err != storage.ErrFetching {
			panic(fmt.Sprintf("not in db? key %v chunk %v", req.Key, chunk))
		}
		select {
		case <-chunk.ReqC:
			log.Error("someone else delivered?", "hash", chunk.Key.Hex())
			continue R
		default:
		}
		chunk.SData = req.SData
		d.db.Put(chunk)
		chunk.WaitToStore()
		close(chunk.ReqC)
	}
}

// RequestFromPeers sends a chunk retrieve request to
func (d *Delivery) RequestFromPeers(hash []byte, skipCheck bool, peersToSkip ...discover.NodeID) error {
	var success bool
	var err error
	d.overlay.EachConn(hash, 255, func(p network.OverlayConn, po int, nn bool) bool {
		spId := p.(*network.BzzPeer).ID()
		for _, p := range peersToSkip {
			if p == spId {
				log.Trace("Delivery.RequestFromPeers: skip peer", "peer", spId)
				return true
			}
		}
		sp := d.getPeer(spId)
		if sp == nil {
			log.Warn("Delivery.RequestFromPeers: peer not found", "id", spId)
			return true
		}
		// TODO: skip light nodes that do not accept retrieve requests
		err = sp.SendPriority(&RetrieveRequestMsg{
			Key:       hash,
			SkipCheck: skipCheck,
		}, Top)
		success = true
		return false
	})
	if success {
		return err
	}
	return errors.New("no peer found")
}
