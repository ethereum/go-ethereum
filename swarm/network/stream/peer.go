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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	pq "github.com/ethereum/go-ethereum/swarm/network/priorityqueue"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var sendTimeout = 5 * time.Second

type notFoundError struct {
	t string
	s Stream
}

func newNotFoundError(t string, s Stream) *notFoundError {
	return &notFoundError{t: t, s: s}
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("%s not found for stream %q", e.t, e.s)
}

// Peer is the Peer extension for the streaming protocol
type Peer struct {
	*protocols.Peer
	streamer *Registry
	pq       *pq.PriorityQueue
	serverMu sync.RWMutex
	clientMu sync.RWMutex // protects both clients and clientParams
	servers  map[string]*server
	clients  map[string]*client
	// clientParams map keeps required client arguments
	// that are set on Registry.Subscribe and used
	// on creating a new client in offered hashes handler.
	clientParams map[string]*clientParams
	quit         chan struct{}
}

// NewPeer is the constructor for Peer
func NewPeer(peer *protocols.Peer, streamer *Registry) *Peer {
	p := &Peer{
		Peer:         peer,
		pq:           pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer:     streamer,
		servers:      make(map[string]*server),
		clients:      make(map[string]*client),
		clientParams: make(map[string]*clientParams),
		quit:         make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go p.pq.Run(ctx, func(i interface{}) { p.Send(i) })
	go func() {
		<-p.quit
		cancel()
	}()
	return p
}

// Deliver sends a storeRequestMsg protocol message to the peer
func (p *Peer) Deliver(chunk *storage.Chunk, priority uint8) error {
	msg := &ChunkDeliveryMsg{
		Key:   chunk.Key,
		SData: chunk.SData,
	}
	return p.SendPriority(msg, priority)
}

// SendPriority sends message to the peer using the outgoing priority queue
func (p *Peer) SendPriority(msg interface{}, priority uint8) error {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	return p.pq.Push(ctx, msg, int(priority))
}

// SendOfferedHashes sends OfferedHashesMsg protocol msg
func (p *Peer) SendOfferedHashes(s *server, f, t uint64) error {
	hashes, from, to, proof, err := s.SetNextBatch(f, t)
	if err != nil {
		return err
	}
	// true only when quiting
	if len(hashes) == 0 {
		return nil
	}
	if proof == nil {
		proof = &HandoverProof{
			Handover: &Handover{},
		}
	}
	s.currentBatch = hashes
	msg := &OfferedHashesMsg{
		HandoverProof: proof,
		Hashes:        hashes,
		From:          from,
		To:            to,
		Stream:        s.stream,
	}
	log.Trace("Swarm syncer offer batch", "peer", p.ID(), "stream", s.stream, "len", len(hashes), "from", from, "to", to)
	return p.SendPriority(msg, s.priority)
}

func (p *Peer) getServer(s Stream) (*server, error) {
	p.serverMu.RLock()
	defer p.serverMu.RUnlock()

	server := p.servers[s.String()]
	if server == nil {
		return nil, fmt.Errorf("server '%v' not provided to peer %v", s, p.ID())
	}
	return server, nil
}

func (p *Peer) setServer(s Stream, o Server, priority uint8) (*server, error) {
	p.serverMu.Lock()
	defer p.serverMu.Unlock()

	sk := s.String()
	if p.servers[sk] != nil {
		return nil, fmt.Errorf("server %v already registered", sk)
	}
	os := &server{
		Server:   o,
		stream:   s,
		priority: priority,
	}
	p.servers[sk] = os
	return os, nil
}

func (p *Peer) removeServer(s Stream) error {
	p.serverMu.Lock()
	defer p.serverMu.Unlock()

	sk := s.String()
	server, ok := p.servers[sk]
	if !ok {
		return newNotFoundError("server", s)
	}
	server.Close()
	delete(p.servers, sk)
	return nil
}

func (p *Peer) getClient(ctx context.Context, s Stream) (c *client, err error) {
	var params *clientParams
	sk := s.String()
	func() {
		p.clientMu.RLock()
		defer p.clientMu.RUnlock()

		c = p.clients[sk]
		if c != nil {
			return
		}
		params = p.clientParams[sk]
	}()
	if c != nil {
		return c, nil
	}

	if params != nil {
		//debug.PrintStack()
		if err := params.waitClient(ctx); err != nil {
			return nil, err
		}
	}

	p.clientMu.RLock()
	defer p.clientMu.RUnlock()

	c = p.clients[sk]
	if c != nil {
		return c, nil
	}
	return nil, newNotFoundError("client", s)
}

func (p *Peer) getOrSetClient(s Stream, from, to uint64) (c *client, created bool, err error) {
	sk := s.String()

	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	c = p.clients[sk]
	if c != nil {
		return c, false, nil
	}

	f, err := p.streamer.GetClientFunc(s.Name)
	if err != nil {
		return nil, false, err
	}

	is, err := f(p, s.Key, s.Live)
	if err != nil {
		return nil, false, err
	}

	cp, err := p.getClientParams(s)
	if err != nil {
		return nil, false, err
	}
	defer func() {
		if err == nil {
			if err := p.removeClientParams(s); err != nil {
				log.Error("stream set client: remove client params", "stream", s, "peer", p, "err", err)
			}
		}
	}()

	intervalsKey := peerStreamIntervalsKey(p, s)
	if s.Live {
		// try to find previous history and live intervals and merge live into history
		historyKey := peerStreamIntervalsKey(p, NewStream(s.Name, s.Key, false))
		historyIntervals := &intervals.Intervals{}
		err := p.streamer.intervalsStore.Get(historyKey, historyIntervals)
		switch err {
		case nil:
			liveIntervals := &intervals.Intervals{}
			err := p.streamer.intervalsStore.Get(intervalsKey, liveIntervals)
			switch err {
			case nil:
				historyIntervals.Merge(liveIntervals)
				if err := p.streamer.intervalsStore.Put(historyKey, historyIntervals); err != nil {
					log.Error("stream set client: put history intervals", "stream", s, "peer", p, "err", err)
				}
			case state.ErrNotFound:
			default:
				log.Error("stream set client: get live intervals", "stream", s, "peer", p, "err", err)
			}
		case state.ErrNotFound:
		default:
			log.Error("stream set client: get history intervals", "stream", s, "peer", p, "err", err)
		}
	}

	if err := p.streamer.intervalsStore.Put(intervalsKey, intervals.NewIntervals(from)); err != nil {
		return nil, false, err
	}

	next := make(chan error, 1)
	c = &client{
		Client:         is,
		stream:         s,
		priority:       cp.priority,
		to:             cp.to,
		next:           next,
		intervalsStore: p.streamer.intervalsStore,
		intervalsKey:   intervalsKey,
	}
	p.clients[sk] = c
	cp.clientCreated() // unblock all possible getClient calls that are waiting
	next <- nil        // this is to allow wantedKeysMsg before first batch arrives
	return c, true, nil
}

func (p *Peer) removeClient(s Stream) error {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	client, ok := p.clients[s.String()]
	if !ok {
		return newNotFoundError("client", s)
	}
	client.close()
	return nil
}

func (p *Peer) setClientParams(s Stream, params *clientParams) error {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	sk := s.String()
	if p.clients[sk] != nil {
		return fmt.Errorf("client %v already exists", sk)
	}
	if p.clientParams[sk] != nil {
		return fmt.Errorf("client params %v already set", sk)
	}
	p.clientParams[sk] = params
	return nil
}

func (p *Peer) getClientParams(s Stream) (*clientParams, error) {
	params := p.clientParams[s.String()]
	if params == nil {
		return nil, fmt.Errorf("client params '%v' not provided to peer %v", s, p.ID())
	}
	return params, nil
}

func (p *Peer) removeClientParams(s Stream) error {
	sk := s.String()
	_, ok := p.clientParams[sk]
	if !ok {
		return newNotFoundError("client params", s)
	}
	delete(p.clientParams, sk)
	return nil
}

func (p *Peer) close() {
	for _, s := range p.servers {
		s.Close()
	}
}
