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
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/p2p"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	Low uint8 = iota
	Mid
	High
	Top
	PriorityQueue        // number of queues
	PriorityQueueCap = 3 // queue capacity
	HashSize         = 32
)

// Registry registry for outgoing and incoming streamer constructors
type Registry struct {
	clientMu    sync.RWMutex
	serverMu    sync.RWMutex
	peersMu     sync.RWMutex
	serverFuncs map[string]func(*Peer, []byte) (Server, error)
	clientFuncs map[string]func(*Peer, []byte) (Client, error)
	peers       map[discover.NodeID]*Peer
	delivery    *Delivery
}

// NewRegistry is Streamer constructor
func NewRegistry(delivery *Delivery) *Registry {
	streamer := &Registry{
		serverFuncs: make(map[string]func(*Peer, []byte) (Server, error)),
		clientFuncs: make(map[string]func(*Peer, []byte) (Client, error)),
		peers:       make(map[discover.NodeID]*Peer),
		delivery:    delivery,
	}
	delivery.getPeer = streamer.getPeer
	streamer.RegisterServerFunc(swarmChunkServerStreamName, func(_ *Peer, t []byte) (Server, error) {
		return NewSwarmChunkServer(delivery.db), nil
	})
	streamer.RegisterClientFunc(swarmChunkServerStreamName, func(p *Peer, t []byte) (Client, error) {
		return NewSwarmSyncerClient(p, delivery.db, nil)
	})
	return streamer
}

// RegisterClient registers an incoming streamer constructor
func (r *Registry) RegisterClientFunc(stream string, f func(*Peer, []byte) (Client, error)) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()

	r.clientFuncs[stream] = f
}

// RegisterServer registers an outgoing streamer constructor
func (r *Registry) RegisterServerFunc(stream string, f func(*Peer, []byte) (Server, error)) {
	r.serverMu.Lock()
	defer r.serverMu.Unlock()

	r.serverFuncs[stream] = f
}

// GetClient accessor for incoming streamer constructors
func (r *Registry) GetClientFunc(stream string) (func(*Peer, []byte) (Client, error), error) {
	r.clientMu.RLock()
	defer r.clientMu.RUnlock()

	f := r.clientFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// GetServer accessor for incoming streamer constructors
func (r *Registry) GetServerFunc(stream string) (func(*Peer, []byte) (Server, error), error) {
	r.serverMu.RLock()
	defer r.serverMu.RUnlock()

	f := r.serverFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// Subscribe initiates the streamer
func (r *Registry) Subscribe(peerId discover.NodeID, s string, t []byte, from, to uint64, priority uint8, live bool) error {
	f, err := r.GetClientFunc(s)
	if err != nil {
		return err
	}

	peer := r.getPeer(peerId)
	if peer == nil {
		return fmt.Errorf("peer not found %v", peerId)
	}

	is, err := f(peer, t)
	if err != nil {
		return err
	}
	err = peer.setClient(s, t, is, priority, live)
	if err != nil {
		return err
	}

	msg := &SubscribeMsg{
		Stream: s,
		Key:    t,
		// Live:     live,
		From:     from,
		To:       to,
		Priority: priority,
	}
	log.Debug("Subscribe ", "peer", peerId, "stream", s, "key", t, "from", from, "to", to)

	peer.SendPriority(msg, priority)
	return nil
}

func (r *Registry) Retrieve(chunk *storage.Chunk) error {
	return r.delivery.RequestFromPeers(chunk.Key[:], false)
}

func (r *Registry) NodeInfo() interface{} {
	return nil
}

func (r *Registry) PeerInfo(id discover.NodeID) interface{} {
	return nil
}

func (r *Registry) getPeer(peerId discover.NodeID) *Peer {
	r.peersMu.RLock()
	defer r.peersMu.RUnlock()

	return r.peers[peerId]
}

func (r *Registry) setPeer(peer *Peer) {
	r.peersMu.Lock()
	r.peers[peer.ID()] = peer
	r.peersMu.Unlock()
}

func (r *Registry) deletePeer(peer *Peer) {
	r.peersMu.Lock()
	delete(r.peers, peer.ID())
	r.peersMu.Unlock()
}

// Run protocol run function
func (r *Registry) Run(p *protocols.Peer) error {
	sp := NewPeer(p, r)
	// load saved intervals

	r.setPeer(sp)

	defer r.deletePeer(sp)
	defer close(sp.quit)
	return sp.Run(sp.HandleMsg)
}

// HandleMsg is the message handler that delegates incoming messages
func (p *Peer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *SubscribeMsg:
		return p.handleSubscribeMsg(msg)

	case *OfferedHashesMsg:
		return p.handleOfferedHashesMsg(msg)

	case *TakeoverProofMsg:
		return p.handleTakeoverProofMsg(msg)

	case *WantedHashesMsg:
		return p.handleWantedHashesMsg(msg)

	case *ChunkDeliveryMsg:
		return p.streamer.delivery.handleChunkDeliveryMsg(msg)

	case *RetrieveRequestMsg:
		return p.streamer.delivery.handleRetrieveRequestMsg(p, msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

func keyToString(key []byte) string {
	l := len(key)
	if l == 0 {
		return ""
	}
	return fmt.Sprintf("%s-%d", string(key[:l-1]), uint8(key[l-1]))
}

type server struct {
	Server
	priority     uint8
	currentBatch []byte
	stream       string
	key          []byte
}

// Server interface for outgoing peer Streamer
type Server interface {
	SetNextBatch(uint64, uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error)
	GetData([]byte) []byte
}

type client struct {
	Client
	priority  uint8
	sessionAt uint64
	live      bool
	stream    string
	key       []byte
	quit      chan struct{}
	next      chan struct{}
}

// Client interface for incoming peer Streamer
type Client interface {
	NeedData([]byte) func()
	BatchDone(string, uint64, []byte, []byte) func() (*TakeoverProof, error)
}

// NextBatch adjusts the indexes by inspecting the intervals
func (c *client) nextBatch(from uint64) (nextFrom uint64, nextTo uint64) {
	var intervals []uint64
	if c.live {
		if len(intervals) == 0 {
			intervals = []uint64{c.sessionAt, from}
		} else {
			intervals[1] = from
		}
		nextFrom = from
	} else if from >= c.sessionAt { // history sync complete
		intervals = nil
		nextFrom = from
		nextTo = math.MaxUint64
	} else if len(intervals) > 2 && from >= intervals[2] { // filled a gap in the intervals
		intervals = append(intervals[:1], intervals[3:]...)
		nextFrom = intervals[1]
		if len(intervals) > 2 {
			nextTo = intervals[2]
		} else {
			nextTo = c.sessionAt
		}
	} else {
		nextFrom = from
		intervals[1] = from
		nextTo = c.sessionAt
	}
	// b.intervals.set(intervals)
	return nextFrom, nextTo
}

// Spec is the spec of the streamer protocol.
var Spec = &protocols.Spec{
	Name:       "stream",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		UnsubscribeMsg{},
		OfferedHashesMsg{},
		WantedHashesMsg{},
		TakeoverProofMsg{},
		SubscribeMsg{},
		RetrieveRequestMsg{},
		ChunkDeliveryMsg{},
	},
}

func (r *Registry) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    Spec.Name,
			Version: Spec.Version,
			Length:  Spec.Length(),
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := protocols.NewPeer(p, rw, Spec)
				return r.Run(peer)
			},
			NodeInfo: r.NodeInfo,
			PeerInfo: r.PeerInfo,
		},
	}
}
