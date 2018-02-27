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
	"fmt"
	"math"
	"sync"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	Low uint8 = iota
	Mid
	High
	Top
	PriorityQueue         // number of queues
	PriorityQueueCap = 32 // queue capacity
	HashSize         = 32
)

// Registry registry for outgoing and incoming streamer constructors
type Registry struct {
	api            *API
	addr           *network.BzzAddr
	skipCheck      bool
	clientMu       sync.RWMutex
	serverMu       sync.RWMutex
	peersMu        sync.RWMutex
	serverFuncs    map[string]func(*Peer, []byte, bool) (Server, error)
	clientFuncs    map[string]func(*Peer, []byte, bool) (Client, error)
	peers          map[discover.NodeID]*Peer
	delivery       *Delivery
	store          storage.ChunkStore
	intervalsStore intervals.Store
}

// NewRegistry is Streamer constructor
func NewRegistry(addr *network.BzzAddr, delivery *Delivery, store storage.ChunkStore, intervalsStore intervals.Store, skipCheck bool) *Registry {
	streamer := &Registry{
		addr:           addr,
		skipCheck:      skipCheck,
		store:          store,
		serverFuncs:    make(map[string]func(*Peer, []byte, bool) (Server, error)),
		clientFuncs:    make(map[string]func(*Peer, []byte, bool) (Client, error)),
		peers:          make(map[discover.NodeID]*Peer),
		delivery:       delivery,
		intervalsStore: intervalsStore,
	}
	streamer.api = NewAPI(streamer, streamer.store)
	delivery.getPeer = streamer.getPeer
	streamer.RegisterServerFunc(swarmChunkServerStreamName, func(_ *Peer, _ []byte, _ bool) (Server, error) {
		return NewSwarmChunkServer(delivery.db), nil
	})
	streamer.RegisterClientFunc(swarmChunkServerStreamName, func(p *Peer, _ []byte, _ bool) (Client, error) {
		return NewSwarmSyncerClient(p, delivery.db, nil)
	})
	return streamer
}

// RegisterClient registers an incoming streamer constructor
func (r *Registry) RegisterClientFunc(stream string, f func(*Peer, []byte, bool) (Client, error)) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()

	r.clientFuncs[stream] = f
}

// RegisterServer registers an outgoing streamer constructor
func (r *Registry) RegisterServerFunc(stream string, f func(*Peer, []byte, bool) (Server, error)) {
	r.serverMu.Lock()
	defer r.serverMu.Unlock()

	r.serverFuncs[stream] = f
}

// GetClient accessor for incoming streamer constructors
func (r *Registry) GetClientFunc(stream string) (func(*Peer, []byte, bool) (Client, error), error) {
	r.clientMu.RLock()
	defer r.clientMu.RUnlock()

	f := r.clientFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// GetServer accessor for incoming streamer constructors
func (r *Registry) GetServerFunc(stream string) (func(*Peer, []byte, bool) (Server, error), error) {
	r.serverMu.RLock()
	defer r.serverMu.RUnlock()

	f := r.serverFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// Subscribe initiates the streamer
func (r *Registry) Subscribe(peerId discover.NodeID, s Stream, h *Range, priority uint8) error {
	// check if the stream is registered
	if _, err := r.GetClientFunc(s.Name); err != nil {
		return err
	}

	peer := r.getPeer(peerId)
	if peer == nil {
		return fmt.Errorf("peer not found %v", peerId)
	}

	var to uint64
	if !s.Live && h != nil {
		to = h.To
	}

	err := peer.setClientParams(s, newClientParams(priority, to))
	if err != nil {
		return err
	}

	if s.Live && h != nil {
		if err := peer.setClientParams(
			getHistoryStream(s),
			newClientParams(getHistoryPriority(priority), h.To),
		); err != nil {
			return err
		}
	}

	msg := &SubscribeMsg{
		Stream:   s,
		History:  h,
		Priority: priority,
	}
	log.Debug("Subscribe ", "peer", peerId, "stream", s, "history", h)

	return peer.SendPriority(msg, priority)
}

func (r *Registry) Unsubscribe(peerId discover.NodeID, s Stream) error {
	peer := r.getPeer(peerId)
	if peer == nil {
		return fmt.Errorf("peer not found %v", peerId)
	}

	msg := &UnsubscribeMsg{
		Stream: s,
	}
	log.Debug("Unsubscribe ", "peer", peerId, "stream", s)

	if err := peer.Send(msg); err != nil {
		return err
	}
	return peer.removeClient(s)
}

func (r *Registry) Retrieve(chunk *storage.Chunk) error {
	return r.delivery.RequestFromPeers(chunk.Key[:], r.skipCheck)
}

func (r *Registry) NodeInfo() interface{} {
	return nil
}

func (r *Registry) PeerInfo(id discover.NodeID) interface{} {
	return nil
}

func (r *Registry) Close() error {
	r.store.Close()
	return r.intervalsStore.Close()
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

func (r *Registry) peersCount() (c int) {
	r.peersMu.Lock()
	c = len(r.peers)
	r.peersMu.Unlock()
	return
}

// Run protocol run function
func (r *Registry) run(p *protocols.Peer) error {
	sp := NewPeer(p, r)
	r.setPeer(sp)
	defer r.deletePeer(sp)
	defer close(sp.quit)
	defer sp.close()
	return sp.Run(sp.HandleMsg)
}

func (r *Registry) runProtocol(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := protocols.NewPeer(p, rw, Spec)
	bzzPeer := network.NewBzzTestPeer(peer, r.addr)
	r.delivery.overlay.On(bzzPeer)
	defer r.delivery.overlay.Off(bzzPeer)
	return r.run(peer)
}

// HandleMsg is the message handler that delegates incoming messages
func (p *Peer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *SubscribeMsg:
		return p.handleSubscribeMsg(msg)

	case *SubscribeErrorMsg:
		return p.handleSubscribeErrorMsg(msg)

	case *UnsubscribeMsg:
		return p.handleUnsubscribeMsg(msg)

	case *OfferedHashesMsg:
		return p.handleOfferedHashesMsg(msg)

	case *TakeoverProofMsg:
		return p.handleTakeoverProofMsg(msg)

	case *WantedHashesMsg:
		return p.handleWantedHashesMsg(msg)

	case *ChunkDeliveryMsg:
		return p.streamer.delivery.handleChunkDeliveryMsg(p, msg)

	case *RetrieveRequestMsg:
		return p.streamer.delivery.handleRetrieveRequestMsg(p, msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

type server struct {
	Server
	stream       Stream
	priority     uint8
	currentBatch []byte
}

// Server interface for outgoing peer Streamer
type Server interface {
	SetNextBatch(uint64, uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error)
	GetData([]byte) ([]byte, error)
	Close()
}

type client struct {
	Client
	stream    Stream
	priority  uint8
	sessionAt uint64
	to        uint64
	next      chan error

	intervalsKey   string
	intervalsStore intervals.Store
}

func peerStreamIntervalsKey(p *Peer, s Stream) string {
	return p.ID().String() + s.String()
}

func (c client) AddInterval(start, end uint64) (err error) {
	i, err := c.intervalsStore.Get(c.intervalsKey)
	if err != nil {
		return err
	}
	i.Add(start, end)
	return c.intervalsStore.Put(c.intervalsKey, i)
}

func (c client) NextInterval() (start, end uint64, err error) {
	i, err := c.intervalsStore.Get(c.intervalsKey)
	if err != nil {
		return 0, 0, err
	}
	start, end = i.Next()
	return start, end, nil
}

// Client interface for incoming peer Streamer
type Client interface {
	NeedData([]byte) func()
	BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error)
	Close()
}

func (c *client) nextBatch(from uint64) (nextFrom uint64, nextTo uint64) {
	if c.to > 0 && from >= c.to {
		return 0, 0
	}
	if c.stream.Live {
		return from, 0
	} else if from >= c.sessionAt {
		if c.to > 0 {
			return from, c.to
		}
		return from, math.MaxUint64
	}
	nextFrom, nextTo, err := c.NextInterval()
	if err != nil {
		log.Error("next intervals", "stream", c.stream)
		return
	}
	if nextTo > c.to {
		nextTo = c.to
	}
	if nextTo == 0 {
		nextTo = c.sessionAt
	}
	return
}

func (c *client) batchDone(p *Peer, req *OfferedHashesMsg, hashes []byte) error {
	if tf := c.BatchDone(req.Stream, req.From, hashes, req.Root); tf != nil {
		tp, err := tf()
		if err != nil {
			return err
		}
		// TODO: make a test case for testing if the interval is added when the batch is done
		if err := c.AddInterval(tp.Takeover.Start, tp.Takeover.End); err != nil {
			return err
		}
		if err := p.SendPriority(tp, c.priority); err != nil {
			return err
		}
		if c.to > 0 && tp.Takeover.End >= c.to {
			return p.streamer.Unsubscribe(p.Peer.ID(), req.Stream)
		}
		return nil
	}
	return nil
}

func (c *client) close() {
	close(c.next)
	c.Close()
}

// clientParams store parameters for the new client
// between a subscription and initial offered hashes request handling.
type clientParams struct {
	priority uint8
	to       uint64
	// signal when the client is created
	clientCreatedC chan struct{}
}

func newClientParams(priority uint8, to uint64) *clientParams {
	return &clientParams{
		priority:       priority,
		to:             to,
		clientCreatedC: make(chan struct{}),
	}
}

func (c *clientParams) waitClient(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.clientCreatedC:
		return nil
	}
}

func (c *clientParams) clientCreated() {
	close(c.clientCreatedC)
}

// Spec is the spec of the streamer protocol
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
		SubscribeErrorMsg{},
	},
}

func (r *Registry) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    Spec.Name,
			Version: Spec.Version,
			Length:  Spec.Length(),
			Run:     r.runProtocol,
			// NodeInfo: ,
			// PeerInfo: ,
		},
	}
}

func (r *Registry) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "stream",
			Version:   "0.1",
			Service:   r.api,
			Public:    true,
		},
	}
}

func (r *Registry) Start(server *p2p.Server) error {
	r.api.dpa.Start()
	return nil
}

func (r *Registry) Stop() error {
	r.api.dpa.Stop()
	return nil
}

type Range struct {
	From, To uint64
}

func getHistoryPriority(priority uint8) uint8 {
	if priority == 0 {
		return 0
	}
	return priority - 1
}

func getHistoryStream(s Stream) Stream {
	return NewStream(s.Name, s.Key, false)
}

type API struct {
	streamer *Registry
	dpa      *storage.DPA
}

func NewAPI(r *Registry, store storage.ChunkStore) *API {
	dpa := storage.NewDPA(store, storage.NewChunkerParams())
	return &API{
		streamer: r,
		dpa:      dpa,
	}
}

func (api *API) SubscribeStream(peerId discover.NodeID, s Stream, history *Range, priority uint8) error {
	return api.streamer.Subscribe(peerId, s, history, priority)
}

func (api *API) UnsubscribeStream(peerId discover.NodeID, s Stream) error {
	return api.streamer.Unsubscribe(peerId, s)
}
