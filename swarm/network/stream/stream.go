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
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	opentracing "github.com/opentracing/opentracing-go"
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
	serverFuncs    map[string]func(*Peer, string, bool) (Server, error)
	clientFuncs    map[string]func(*Peer, string, bool) (Client, error)
	peers          map[discover.NodeID]*Peer
	delivery       *Delivery
	intervalsStore state.Store
	doRetrieve     bool
}

// RegistryOptions holds optional values for NewRegistry constructor.
type RegistryOptions struct {
	SkipCheck       bool
	DoSync          bool
	DoRetrieve      bool
	SyncUpdateDelay time.Duration
}

// NewRegistry is Streamer constructor
func NewRegistry(addr *network.BzzAddr, delivery *Delivery, db *storage.DBAPI, intervalsStore state.Store, options *RegistryOptions) *Registry {
	if options == nil {
		options = &RegistryOptions{}
	}
	if options.SyncUpdateDelay <= 0 {
		options.SyncUpdateDelay = 15 * time.Second
	}
	streamer := &Registry{
		addr:           addr,
		skipCheck:      options.SkipCheck,
		serverFuncs:    make(map[string]func(*Peer, string, bool) (Server, error)),
		clientFuncs:    make(map[string]func(*Peer, string, bool) (Client, error)),
		peers:          make(map[discover.NodeID]*Peer),
		delivery:       delivery,
		intervalsStore: intervalsStore,
		doRetrieve:     options.DoRetrieve,
	}
	streamer.api = NewAPI(streamer)
	delivery.getPeer = streamer.getPeer
	streamer.RegisterServerFunc(swarmChunkServerStreamName, func(_ *Peer, _ string, _ bool) (Server, error) {
		return NewSwarmChunkServer(delivery.db), nil
	})
	streamer.RegisterClientFunc(swarmChunkServerStreamName, func(p *Peer, t string, live bool) (Client, error) {
		return NewSwarmSyncerClient(p, delivery.db, false, NewStream(swarmChunkServerStreamName, t, live))
	})
	RegisterSwarmSyncerServer(streamer, db)
	RegisterSwarmSyncerClient(streamer, db)

	if options.DoSync {
		// latestIntC function ensures that
		//   - receiving from the in chan is not blocked by processing inside the for loop
		// 	 - the latest int value is delivered to the loop after the processing is done
		// In context of NeighbourhoodDepthC:
		// after the syncing is done updating inside the loop, we do not need to update on the intermediate
		// depth changes, only to the latest one
		latestIntC := func(in <-chan int) <-chan int {
			out := make(chan int, 1)

			go func() {
				defer close(out)

				for i := range in {
					select {
					case <-out:
					default:
					}
					out <- i
				}
			}()

			return out
		}

		go func() {
			// wait for kademlia table to be healthy
			time.Sleep(options.SyncUpdateDelay)

			kad := streamer.delivery.overlay.(*network.Kademlia)
			depthC := latestIntC(kad.NeighbourhoodDepthC())
			addressBookSizeC := latestIntC(kad.AddrCountC())

			// initial requests for syncing subscription to peers
			streamer.updateSyncing()

			for depth := range depthC {
				log.Debug("Kademlia neighbourhood depth change", "depth", depth)

				// Prevent too early sync subscriptions by waiting until there are no
				// new peers connecting. Sync streams updating will be done after no
				// peers are connected for at least SyncUpdateDelay period.
				timer := time.NewTimer(options.SyncUpdateDelay)
				// Hard limit to sync update delay, preventing long delays
				// on a very dynamic network
				maxTimer := time.NewTimer(3 * time.Minute)
			loop:
				for {
					select {
					case <-maxTimer.C:
						// force syncing update when a hard timeout is reached
						log.Trace("Sync subscriptions update on hard timeout")
						// request for syncing subscription to new peers
						streamer.updateSyncing()
						break loop
					case <-timer.C:
						// start syncing as no new peers has been added to kademlia
						// for some time
						log.Trace("Sync subscriptions update")
						// request for syncing subscription to new peers
						streamer.updateSyncing()
						break loop
					case size := <-addressBookSizeC:
						log.Trace("Kademlia address book size changed on depth change", "size", size)
						// new peers has been added to kademlia,
						// reset the timer to prevent early sync subscriptions
						if !timer.Stop() {
							<-timer.C
						}
						timer.Reset(options.SyncUpdateDelay)
					}
				}
				timer.Stop()
				maxTimer.Stop()
			}
		}()
	}

	return streamer
}

// RegisterClient registers an incoming streamer constructor
func (r *Registry) RegisterClientFunc(stream string, f func(*Peer, string, bool) (Client, error)) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()

	r.clientFuncs[stream] = f
}

// RegisterServer registers an outgoing streamer constructor
func (r *Registry) RegisterServerFunc(stream string, f func(*Peer, string, bool) (Server, error)) {
	r.serverMu.Lock()
	defer r.serverMu.Unlock()

	r.serverFuncs[stream] = f
}

// GetClient accessor for incoming streamer constructors
func (r *Registry) GetClientFunc(stream string) (func(*Peer, string, bool) (Client, error), error) {
	r.clientMu.RLock()
	defer r.clientMu.RUnlock()

	f := r.clientFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// GetServer accessor for incoming streamer constructors
func (r *Registry) GetServerFunc(stream string) (func(*Peer, string, bool) (Server, error), error) {
	r.serverMu.RLock()
	defer r.serverMu.RUnlock()

	f := r.serverFuncs[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

func (r *Registry) RequestSubscription(peerId discover.NodeID, s Stream, h *Range, prio uint8) error {
	// check if the stream is registered
	if _, err := r.GetServerFunc(s.Name); err != nil {
		return err
	}

	peer := r.getPeer(peerId)
	if peer == nil {
		return fmt.Errorf("peer not found %v", peerId)
	}

	if _, err := peer.getServer(s); err != nil {
		if e, ok := err.(*notFoundError); ok && e.t == "server" {
			// request subscription only if the server for this stream is not created
			log.Debug("RequestSubscription ", "peer", peerId, "stream", s, "history", h)
			return peer.Send(context.TODO(), &RequestSubscriptionMsg{
				Stream:   s,
				History:  h,
				Priority: prio,
			})
		}
		return err
	}
	log.Trace("RequestSubscription: already subscribed", "peer", peerId, "stream", s, "history", h)
	return nil
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

	return peer.SendPriority(context.TODO(), msg, priority)
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

	if err := peer.Send(context.TODO(), msg); err != nil {
		return err
	}
	return peer.removeClient(s)
}

// Quit sends the QuitMsg to the peer to remove the
// stream peer client and terminate the streaming.
func (r *Registry) Quit(peerId discover.NodeID, s Stream) error {
	peer := r.getPeer(peerId)
	if peer == nil {
		log.Debug("stream quit: peer not found", "peer", peerId, "stream", s)
		// if the peer is not found, abort the request
		return nil
	}

	msg := &QuitMsg{
		Stream: s,
	}
	log.Debug("Quit ", "peer", peerId, "stream", s)

	return peer.Send(context.TODO(), msg)
}

func (r *Registry) Retrieve(ctx context.Context, chunk *storage.Chunk) error {
	var sp opentracing.Span
	ctx, sp = spancontext.StartSpan(
		ctx,
		"registry.retrieve")
	defer sp.Finish()

	return r.delivery.RequestFromPeers(ctx, chunk.Addr[:], r.skipCheck)
}

func (r *Registry) NodeInfo() interface{} {
	return nil
}

func (r *Registry) PeerInfo(id discover.NodeID) interface{} {
	return nil
}

func (r *Registry) Close() error {
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
	metrics.GetOrRegisterGauge("registry.peers", nil).Update(int64(len(r.peers)))
	r.peersMu.Unlock()
}

func (r *Registry) deletePeer(peer *Peer) {
	r.peersMu.Lock()
	delete(r.peers, peer.ID())
	metrics.GetOrRegisterGauge("registry.peers", nil).Update(int64(len(r.peers)))
	r.peersMu.Unlock()
}

func (r *Registry) peersCount() (c int) {
	r.peersMu.Lock()
	c = len(r.peers)
	r.peersMu.Unlock()
	return
}

// Run protocol run function
func (r *Registry) Run(p *network.BzzPeer) error {
	sp := NewPeer(p.Peer, r)
	r.setPeer(sp)
	defer r.deletePeer(sp)
	defer close(sp.quit)
	defer sp.close()

	if r.doRetrieve {
		err := r.Subscribe(p.ID(), NewStream(swarmChunkServerStreamName, "", false), nil, Top)
		if err != nil {
			return err
		}
	}

	return sp.Run(sp.HandleMsg)
}

// updateSyncing subscribes to SYNC streams by iterating over the
// kademlia connections and bins. If there are existing SYNC streams
// and they are no longer required after iteration, request to Quit
// them will be send to appropriate peers.
func (r *Registry) updateSyncing() {
	// if overlay in not Kademlia, panic
	kad := r.delivery.overlay.(*network.Kademlia)

	// map of all SYNC streams for all peers
	// used at the and of the function to remove servers
	// that are not needed anymore
	subs := make(map[discover.NodeID]map[Stream]struct{})
	r.peersMu.RLock()
	for id, peer := range r.peers {
		peer.serverMu.RLock()
		for stream := range peer.servers {
			if stream.Name == "SYNC" {
				if _, ok := subs[id]; !ok {
					subs[id] = make(map[Stream]struct{})
				}
				subs[id][stream] = struct{}{}
			}
		}
		peer.serverMu.RUnlock()
	}
	r.peersMu.RUnlock()

	// request subscriptions for all nodes and bins
	kad.EachBin(r.addr.Over(), pot.DefaultPof(256), 0, func(conn network.OverlayConn, bin int) bool {
		p := conn.(network.Peer)
		log.Debug(fmt.Sprintf("Requesting subscription by: registry %s from peer %s for bin: %d", r.addr.ID(), p.ID(), bin))

		// bin is always less then 256 and it is safe to convert it to type uint8
		stream := NewStream("SYNC", FormatSyncBinKey(uint8(bin)), true)
		if streams, ok := subs[p.ID()]; ok {
			// delete live and history streams from the map, so that it won't be removed with a Quit request
			delete(streams, stream)
			delete(streams, getHistoryStream(stream))
		}
		err := r.RequestSubscription(p.ID(), stream, NewRange(0, 0), High)
		if err != nil {
			log.Debug("Request subscription", "err", err, "peer", p.ID(), "stream", stream)
			return false
		}
		return true
	})

	// remove SYNC servers that do not need to be subscribed
	for id, streams := range subs {
		if len(streams) == 0 {
			continue
		}
		peer := r.getPeer(id)
		if peer == nil {
			continue
		}
		for stream := range streams {
			log.Debug("Remove sync server", "peer", id, "stream", stream)
			err := r.Quit(peer.ID(), stream)
			if err != nil && err != p2p.ErrShuttingDown {
				log.Error("quit", "err", err, "peer", peer.ID(), "stream", stream)
			}
		}
	}
}

func (r *Registry) runProtocol(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := protocols.NewPeer(p, rw, Spec)
	bzzPeer := network.NewBzzTestPeer(peer, r.addr)
	r.delivery.overlay.On(bzzPeer)
	defer r.delivery.overlay.Off(bzzPeer)
	return r.Run(bzzPeer)
}

// HandleMsg is the message handler that delegates incoming messages
func (p *Peer) HandleMsg(ctx context.Context, msg interface{}) error {
	switch msg := msg.(type) {

	case *SubscribeMsg:
		return p.handleSubscribeMsg(ctx, msg)

	case *SubscribeErrorMsg:
		return p.handleSubscribeErrorMsg(msg)

	case *UnsubscribeMsg:
		return p.handleUnsubscribeMsg(msg)

	case *OfferedHashesMsg:
		return p.handleOfferedHashesMsg(ctx, msg)

	case *TakeoverProofMsg:
		return p.handleTakeoverProofMsg(ctx, msg)

	case *WantedHashesMsg:
		return p.handleWantedHashesMsg(ctx, msg)

	case *ChunkDeliveryMsg:
		return p.streamer.delivery.handleChunkDeliveryMsg(ctx, p, msg)

	case *RetrieveRequestMsg:
		return p.streamer.delivery.handleRetrieveRequestMsg(ctx, p, msg)

	case *RequestSubscriptionMsg:
		return p.handleRequestSubscription(ctx, msg)

	case *QuitMsg:
		return p.handleQuitMsg(msg)

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
	GetData(context.Context, []byte) ([]byte, error)
	Close()
}

type client struct {
	Client
	stream    Stream
	priority  uint8
	sessionAt uint64
	to        uint64
	next      chan error
	quit      chan struct{}

	intervalsKey   string
	intervalsStore state.Store
}

func peerStreamIntervalsKey(p *Peer, s Stream) string {
	return p.ID().String() + s.String()
}

func (c client) AddInterval(start, end uint64) (err error) {
	i := &intervals.Intervals{}
	err = c.intervalsStore.Get(c.intervalsKey, i)
	if err != nil {
		return err
	}
	i.Add(start, end)
	return c.intervalsStore.Put(c.intervalsKey, i)
}

func (c client) NextInterval() (start, end uint64, err error) {
	i := &intervals.Intervals{}
	err = c.intervalsStore.Get(c.intervalsKey, i)
	if err != nil {
		return 0, 0, err
	}
	start, end = i.Next()
	return start, end, nil
}

// Client interface for incoming peer Streamer
type Client interface {
	NeedData(context.Context, []byte) func()
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
		if err := p.SendPriority(context.TODO(), tp, c.priority); err != nil {
			return err
		}
		if c.to > 0 && tp.Takeover.End >= c.to {
			return p.streamer.Unsubscribe(p.Peer.ID(), req.Stream)
		}
		return nil
	}
	// TODO: make a test case for testing if the interval is added when the batch is done
	if err := c.AddInterval(req.From, req.To); err != nil {
		return err
	}
	return nil
}

func (c *client) close() {
	select {
	case <-c.quit:
	default:
		close(c.quit)
	}
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
	Version:    4,
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
		RequestSubscriptionMsg{},
		QuitMsg{},
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
			Version:   "3.0",
			Service:   r.api,
			Public:    true,
		},
	}
}

func (r *Registry) Start(server *p2p.Server) error {
	log.Info("Streamer started")
	return nil
}

func (r *Registry) Stop() error {
	return nil
}

type Range struct {
	From, To uint64
}

func NewRange(from, to uint64) *Range {
	return &Range{
		From: from,
		To:   to,
	}
}

func (r *Range) String() string {
	return fmt.Sprintf("%v-%v", r.From, r.To)
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
}

func NewAPI(r *Registry) *API {
	return &API{
		streamer: r,
	}
}

func (api *API) SubscribeStream(peerId discover.NodeID, s Stream, history *Range, priority uint8) error {
	return api.streamer.Subscribe(peerId, s, history, priority)
}

func (api *API) UnsubscribeStream(peerId discover.NodeID, s Stream) error {
	return api.streamer.Unsubscribe(peerId, s)
}
