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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/swarm/log"
	pq "github.com/ethereum/go-ethereum/swarm/network/priorityqueue"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/tracing"
	opentracing "github.com/opentracing/opentracing-go"
)

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

// ErrMaxPeerServers will be returned if peer server limit is reached.
// It will be sent in the SubscribeErrorMsg.
var ErrMaxPeerServers = errors.New("max peer servers")

// Peer is the Peer extension for the streaming protocol
type Peer struct {
	*protocols.Peer
	streamer *Registry
	pq       *pq.PriorityQueue
	serverMu sync.RWMutex
	clientMu sync.RWMutex // protects both clients and clientParams
	servers  map[Stream]*server
	clients  map[Stream]*client
	// clientParams map keeps required client arguments
	// that are set on Registry.Subscribe and used
	// on creating a new client in offered hashes handler.
	clientParams map[Stream]*clientParams
	quit         chan struct{}
}

type WrappedPriorityMsg struct {
	Context context.Context
	Msg     interface{}
}

// NewPeer is the constructor for Peer
func NewPeer(peer *protocols.Peer, streamer *Registry) *Peer {
	p := &Peer{
		Peer:         peer,
		pq:           pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer:     streamer,
		servers:      make(map[Stream]*server),
		clients:      make(map[Stream]*client),
		clientParams: make(map[Stream]*clientParams),
		quit:         make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go p.pq.Run(ctx, func(i interface{}) {
		wmsg := i.(WrappedPriorityMsg)
		err := p.Send(wmsg.Context, wmsg.Msg)
		if err != nil {
			log.Error("Message send error, dropping peer", "peer", p.ID(), "err", err)
			p.Drop(err)
		}
	})

	// basic monitoring for pq contention
	go func(pq *pq.PriorityQueue) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				var lenMaxi int
				var capMaxi int
				for k := range pq.Queues {
					if lenMaxi < len(pq.Queues[k]) {
						lenMaxi = len(pq.Queues[k])
					}

					if capMaxi < cap(pq.Queues[k]) {
						capMaxi = cap(pq.Queues[k])
					}
				}

				metrics.GetOrRegisterGauge(fmt.Sprintf("pq_len_%s", p.ID().TerminalString()), nil).Update(int64(lenMaxi))
				metrics.GetOrRegisterGauge(fmt.Sprintf("pq_cap_%s", p.ID().TerminalString()), nil).Update(int64(capMaxi))
			case <-p.quit:
				return
			}
		}
	}(p.pq)

	go func() {
		<-p.quit

		cancel()
	}()
	return p
}

// Deliver sends a storeRequestMsg protocol message to the peer
// Depending on the `syncing` parameter we send different message types
func (p *Peer) Deliver(ctx context.Context, chunk storage.Chunk, priority uint8, syncing bool) error {
	var msg interface{}

	spanName := "send.chunk.delivery"

	//we send different types of messages if delivery is for syncing or retrievals,
	//even if handling and content of the message are the same,
	//because swap accounting decides which messages need accounting based on the message type
	if syncing {
		msg = &ChunkDeliveryMsgSyncing{
			Addr:  chunk.Address(),
			SData: chunk.Data(),
		}
		spanName += ".syncing"
	} else {
		msg = &ChunkDeliveryMsgRetrieval{
			Addr:  chunk.Address(),
			SData: chunk.Data(),
		}
		spanName += ".retrieval"
	}

	ctx = context.WithValue(ctx, "stream_send_tag", nil)
	return p.SendPriority(ctx, msg, priority)
}

// SendPriority sends message to the peer using the outgoing priority queue
func (p *Peer) SendPriority(ctx context.Context, msg interface{}, priority uint8) error {
	defer metrics.GetOrRegisterResettingTimer(fmt.Sprintf("peer.sendpriority_t.%d", priority), nil).UpdateSince(time.Now())
	tracing.StartSaveSpan(ctx)
	metrics.GetOrRegisterCounter(fmt.Sprintf("peer.sendpriority.%d", priority), nil).Inc(1)
	wmsg := WrappedPriorityMsg{
		Context: ctx,
		Msg:     msg,
	}
	err := p.pq.Push(wmsg, int(priority))
	if err == pq.ErrContention {
		log.Warn("dropping peer on priority queue contention", "peer", p.ID())
		p.Drop(err)
	}
	return err
}

// SendOfferedHashes sends OfferedHashesMsg protocol msg
func (p *Peer) SendOfferedHashes(s *server, f, t uint64) error {
	var sp opentracing.Span
	ctx, sp := spancontext.StartSpan(
		context.TODO(),
		"send.offered.hashes",
	)
	defer sp.Finish()

	hashes, from, to, proof, err := s.setNextBatch(f, t)
	if err != nil {
		return err
	}
	// true only when quitting
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
	ctx = context.WithValue(ctx, "stream_send_tag", "send.offered.hashes")
	return p.SendPriority(ctx, msg, s.priority)
}

func (p *Peer) getServer(s Stream) (*server, error) {
	p.serverMu.RLock()
	defer p.serverMu.RUnlock()

	server := p.servers[s]
	if server == nil {
		return nil, newNotFoundError("server", s)
	}
	return server, nil
}

func (p *Peer) setServer(s Stream, o Server, priority uint8) (*server, error) {
	p.serverMu.Lock()
	defer p.serverMu.Unlock()

	if p.servers[s] != nil {
		return nil, fmt.Errorf("server %s already registered", s)
	}

	if p.streamer.maxPeerServers > 0 && len(p.servers) >= p.streamer.maxPeerServers {
		return nil, ErrMaxPeerServers
	}

	sessionIndex, err := o.SessionIndex()
	if err != nil {
		return nil, err
	}
	os := &server{
		Server:       o,
		stream:       s,
		priority:     priority,
		sessionIndex: sessionIndex,
	}
	p.servers[s] = os
	return os, nil
}

func (p *Peer) removeServer(s Stream) error {
	p.serverMu.Lock()
	defer p.serverMu.Unlock()

	server, ok := p.servers[s]
	if !ok {
		return newNotFoundError("server", s)
	}
	server.Close()
	delete(p.servers, s)
	return nil
}

func (p *Peer) getClient(ctx context.Context, s Stream) (c *client, err error) {
	var params *clientParams
	func() {
		p.clientMu.RLock()
		defer p.clientMu.RUnlock()

		c = p.clients[s]
		if c != nil {
			return
		}
		params = p.clientParams[s]
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

	c = p.clients[s]
	if c != nil {
		return c, nil
	}
	return nil, newNotFoundError("client", s)
}

func (p *Peer) getOrSetClient(s Stream, from, to uint64) (c *client, created bool, err error) {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	c = p.clients[s]
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
		quit:           make(chan struct{}),
		intervalsStore: p.streamer.intervalsStore,
		intervalsKey:   intervalsKey,
	}
	p.clients[s] = c
	cp.clientCreated() // unblock all possible getClient calls that are waiting
	next <- nil        // this is to allow wantedKeysMsg before first batch arrives
	return c, true, nil
}

func (p *Peer) removeClient(s Stream) error {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	client, ok := p.clients[s]
	if !ok {
		return newNotFoundError("client", s)
	}
	client.close()
	delete(p.clients, s)
	return nil
}

func (p *Peer) setClientParams(s Stream, params *clientParams) error {
	p.clientMu.Lock()
	defer p.clientMu.Unlock()

	if p.clients[s] != nil {
		return fmt.Errorf("client %s already exists", s)
	}
	if p.clientParams[s] != nil {
		return fmt.Errorf("client params %s already set", s)
	}
	p.clientParams[s] = params
	return nil
}

func (p *Peer) getClientParams(s Stream) (*clientParams, error) {
	params := p.clientParams[s]
	if params == nil {
		return nil, fmt.Errorf("client params '%v' not provided to peer %v", s, p.ID())
	}
	return params, nil
}

func (p *Peer) removeClientParams(s Stream) error {
	_, ok := p.clientParams[s]
	if !ok {
		return newNotFoundError("client params", s)
	}
	delete(p.clientParams, s)
	return nil
}

func (p *Peer) close() {
	for _, s := range p.servers {
		s.Close()
	}
}
