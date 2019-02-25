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
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	Low uint8 = iota
	Mid
	High
	Top
	PriorityQueue    = 4    // number of priority queues - Low, Mid, High, Top
	PriorityQueueCap = 4096 // queue capacity
	HashSize         = 32
)

// Enumerate options for syncing and retrieval
type SyncingOption int
type RetrievalOption int

// Syncing options
const (
	// Syncing disabled
	SyncingDisabled SyncingOption = iota
	// Register the client and the server but not subscribe
	SyncingRegisterOnly
	// Both client and server funcs are registered, subscribe sent automatically
	SyncingAutoSubscribe
)

const (
	// Retrieval disabled. Used mostly for tests to isolate syncing features (i.e. syncing only)
	RetrievalDisabled RetrievalOption = iota
	// Only the client side of the retrieve request is registered.
	// (light nodes do not serve retrieve requests)
	// once the client is registered, subscription to retrieve request stream is always sent
	RetrievalClientOnly
	// Both client and server funcs are registered, subscribe sent automatically
	RetrievalEnabled
)

// subscriptionFunc is used to determine what to do in order to perform subscriptions
// usually we would start to really subscribe to nodes, but for tests other functionality may be needed
// (see TestRequestPeerSubscriptions in streamer_test.go)
var subscriptionFunc = doRequestSubscription

// Registry registry for outgoing and incoming streamer constructors
type Registry struct {
	addr           enode.ID
	api            *API
	skipCheck      bool
	clientMu       sync.RWMutex
	serverMu       sync.RWMutex
	peersMu        sync.RWMutex
	serverFuncs    map[string]func(*Peer, string, bool) (Server, error)
	clientFuncs    map[string]func(*Peer, string, bool) (Client, error)
	peers          map[enode.ID]*Peer
	delivery       *Delivery
	intervalsStore state.Store
	autoRetrieval  bool // automatically subscribe to retrieve request stream
	maxPeerServers int
	spec           *protocols.Spec   //this protocol's spec
	balance        protocols.Balance //implements protocols.Balance, for accounting
	prices         protocols.Prices  //implements protocols.Prices, provides prices to accounting
	quit           chan struct{}     // terminates registry goroutines
}

// RegistryOptions holds optional values for NewRegistry constructor.
type RegistryOptions struct {
	SkipCheck       bool
	Syncing         SyncingOption   // Defines syncing behavior
	Retrieval       RetrievalOption // Defines retrieval behavior
	SyncUpdateDelay time.Duration
	MaxPeerServers  int // The limit of servers for each peer in registry
}

// NewRegistry is Streamer constructor
func NewRegistry(localID enode.ID, delivery *Delivery, syncChunkStore storage.SyncChunkStore, intervalsStore state.Store, options *RegistryOptions, balance protocols.Balance) *Registry {
	if options == nil {
		options = &RegistryOptions{}
	}
	if options.SyncUpdateDelay <= 0 {
		options.SyncUpdateDelay = 15 * time.Second
	}
	// check if retrieval has been disabled
	retrieval := options.Retrieval != RetrievalDisabled

	quit := make(chan struct{})

	streamer := &Registry{
		addr:           localID,
		skipCheck:      options.SkipCheck,
		serverFuncs:    make(map[string]func(*Peer, string, bool) (Server, error)),
		clientFuncs:    make(map[string]func(*Peer, string, bool) (Client, error)),
		peers:          make(map[enode.ID]*Peer),
		delivery:       delivery,
		intervalsStore: intervalsStore,
		autoRetrieval:  retrieval,
		maxPeerServers: options.MaxPeerServers,
		balance:        balance,
		quit:           quit,
	}

	streamer.setupSpec()

	streamer.api = NewAPI(streamer)
	delivery.getPeer = streamer.getPeer

	// if retrieval is enabled, register the server func, so that retrieve requests will be served (non-light nodes only)
	if options.Retrieval == RetrievalEnabled {
		streamer.RegisterServerFunc(swarmChunkServerStreamName, func(_ *Peer, _ string, live bool) (Server, error) {
			if !live {
				return nil, errors.New("only live retrieval requests supported")
			}
			return NewSwarmChunkServer(delivery.chunkStore), nil
		})
	}

	// if retrieval is not disabled, register the client func (both light nodes and normal nodes can issue retrieve requests)
	if options.Retrieval != RetrievalDisabled {
		streamer.RegisterClientFunc(swarmChunkServerStreamName, func(p *Peer, t string, live bool) (Client, error) {
			return NewSwarmSyncerClient(p, syncChunkStore, NewStream(swarmChunkServerStreamName, t, live))
		})
	}

	// If syncing is not disabled, the syncing functions are registered (both client and server)
	if options.Syncing != SyncingDisabled {
		RegisterSwarmSyncerServer(streamer, syncChunkStore)
		RegisterSwarmSyncerClient(streamer, syncChunkStore)
	}

	// if syncing is set to automatically subscribe to the syncing stream, start the subscription process
	if options.Syncing == SyncingAutoSubscribe {
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

				for {
					select {
					case i, ok := <-in:
						if !ok {
							return
						}
						select {
						case <-out:
						default:
						}
						out <- i
					case <-quit:
						return
					}
				}
			}()

			return out
		}

		kad := streamer.delivery.kad
		// get notification channels from Kademlia before returning
		// from this function to avoid race with Close method and
		// the goroutine created below
		depthC := latestIntC(kad.NeighbourhoodDepthC())
		addressBookSizeC := latestIntC(kad.AddrCountC())

		go func() {
			// wait for kademlia table to be healthy
			// but return if Registry is closed before
			select {
			case <-time.After(options.SyncUpdateDelay):
			case <-quit:
				return
			}

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
					case <-quit:
						break loop
					}
				}
				timer.Stop()
				maxTimer.Stop()
			}
		}()
	}

	return streamer
}

// This is an accounted protocol, therefore we need to provide a pricing Hook to the spec
// For simulations to be able to run multiple nodes and not override the hook's balance,
// we need to construct a spec instance per node instance
func (r *Registry) setupSpec() {
	// first create the "bare" spec
	r.createSpec()
	// now create the pricing object
	r.createPriceOracle()
	// if balance is nil, this node has been started without swap support (swapEnabled flag is false)
	if r.balance != nil && !reflect.ValueOf(r.balance).IsNil() {
		// swap is enabled, so setup the hook
		r.spec.Hook = protocols.NewAccounting(r.balance, r.prices)
	}
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

func (r *Registry) RequestSubscription(peerId enode.ID, s Stream, h *Range, prio uint8) error {
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
func (r *Registry) Subscribe(peerId enode.ID, s Stream, h *Range, priority uint8) error {
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

func (r *Registry) Unsubscribe(peerId enode.ID, s Stream) error {
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
func (r *Registry) Quit(peerId enode.ID, s Stream) error {
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

func (r *Registry) Close() error {
	// Stop sending neighborhood depth change and address count
	// change from Kademlia that were initiated in NewRegistry constructor.
	r.delivery.kad.CloseNeighbourhoodDepthC()
	r.delivery.kad.CloseAddrCountC()
	close(r.quit)
	return r.intervalsStore.Close()
}

func (r *Registry) getPeer(peerId enode.ID) *Peer {
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

	if r.autoRetrieval && !p.LightNode {
		err := r.Subscribe(p.ID(), NewStream(swarmChunkServerStreamName, "", true), nil, Top)
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
	kad := r.delivery.kad
	// map of all SYNC streams for all peers
	// used at the and of the function to remove servers
	// that are not needed anymore
	subs := make(map[enode.ID]map[Stream]struct{})
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

	// start requesting subscriptions from peers
	r.requestPeerSubscriptions(kad, subs)

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

// requestPeerSubscriptions calls on each live peer in the kademlia table
// and sends a `RequestSubscription` to peers according to their bin
// and their relationship with kademlia's depth.
// Also check `TestRequestPeerSubscriptions` in order to understand the
// expected behavior.
// The function expects:
//   * the kademlia
//   * a map of subscriptions
//   * the actual function to subscribe
//     (in case of the test, it doesn't do real subscriptions)
func (r *Registry) requestPeerSubscriptions(kad *network.Kademlia, subs map[enode.ID]map[Stream]struct{}) {

	var startPo int
	var endPo int
	var ok bool

	// kademlia's depth
	kadDepth := kad.NeighbourhoodDepth()
	// request subscriptions for all nodes and bins
	// nil as base takes the node's base; we need to pass 255 as `EachConn` runs
	// from deepest bins backwards
	kad.EachConn(nil, 255, func(p *network.Peer, po int) bool {
		// nodes that do not provide stream protocol
		// should not be subscribed, e.g. bootnodes
		if !p.HasCap("stream") {
			return true
		}
		//if the peer's bin is shallower than the kademlia depth,
		//only the peer's bin should be subscribed
		if po < kadDepth {
			startPo = po
			endPo = po
		} else {
			//if the peer's bin is equal or deeper than the kademlia depth,
			//each bin from the depth up to k.MaxProxDisplay should be subscribed
			startPo = kadDepth
			endPo = kad.MaxProxDisplay
		}

		for bin := startPo; bin <= endPo; bin++ {
			//do the actual subscription
			ok = subscriptionFunc(r, p, uint8(bin), subs)
		}
		return ok
	})
}

// doRequestSubscription sends the actual RequestSubscription to the peer
func doRequestSubscription(r *Registry, p *network.Peer, bin uint8, subs map[enode.ID]map[Stream]struct{}) bool {
	log.Debug("Requesting subscription by registry:", "registry", r.addr, "peer", p.ID(), "bin", bin)
	// bin is always less then 256 and it is safe to convert it to type uint8
	stream := NewStream("SYNC", FormatSyncBinKey(bin), true)
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
}

func (r *Registry) runProtocol(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := protocols.NewPeer(p, rw, r.spec)
	bp := network.NewBzzPeer(peer)
	np := network.NewPeer(bp, r.delivery.kad)
	r.delivery.kad.On(np)
	defer r.delivery.kad.Off(np)
	return r.Run(bp)
}

// HandleMsg is the message handler that delegates incoming messages
func (p *Peer) HandleMsg(ctx context.Context, msg interface{}) error {
	select {
	case <-p.streamer.quit:
		log.Trace("message received after the streamer is closed", "peer", p.ID())
		// return without an error since streamer is closed and
		// no messages should be handled as other subcomponents like
		// storage leveldb may be closed
		return nil
	default:
	}

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

	case *ChunkDeliveryMsgRetrieval:
		// handling chunk delivery is the same for retrieval and syncing, so let's cast the msg
		return p.streamer.delivery.handleChunkDeliveryMsg(ctx, p, ((*ChunkDeliveryMsg)(msg)))

	case *ChunkDeliveryMsgSyncing:
		// handling chunk delivery is the same for retrieval and syncing, so let's cast the msg
		return p.streamer.delivery.handleChunkDeliveryMsg(ctx, p, ((*ChunkDeliveryMsg)(msg)))

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
	sessionIndex uint64
}

// setNextBatch adjusts passed interval based on session index and whether
// stream is live or history. It calls Server SetNextBatch with adjusted
// interval and returns batch hashes and their interval.
func (s *server) setNextBatch(from, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	if s.stream.Live {
		if from == 0 {
			from = s.sessionIndex
		}
		if to <= from || from >= s.sessionIndex {
			to = math.MaxUint64
		}
	} else {
		if (to < from && to != 0) || from > s.sessionIndex {
			return nil, 0, 0, nil, nil
		}
		if to == 0 || to > s.sessionIndex {
			to = s.sessionIndex
		}
	}
	return s.SetNextBatch(from, to)
}

// Server interface for outgoing peer Streamer
type Server interface {
	// SessionIndex is called when a server is initialized
	// to get the current cursor state of the stream data.
	// Based on this index, live and history stream intervals
	// will be adjusted before calling SetNextBatch.
	SessionIndex() (uint64, error)
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

func (c *client) AddInterval(start, end uint64) (err error) {
	i := &intervals.Intervals{}
	if err = c.intervalsStore.Get(c.intervalsKey, i); err != nil {
		return err
	}
	i.Add(start, end)
	return c.intervalsStore.Put(c.intervalsKey, i)
}

func (c *client) NextInterval() (start, end uint64, err error) {
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
	NeedData(context.Context, []byte) func(context.Context) error
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
	return c.AddInterval(req.From, req.To)
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

// GetSpec returns the streamer spec to callers
// This used to be a global variable but for simulations with
// multiple nodes its fields (notably the Hook) would be overwritten
func (r *Registry) GetSpec() *protocols.Spec {
	return r.spec
}

func (r *Registry) createSpec() {
	// Spec is the spec of the streamer protocol
	var spec = &protocols.Spec{
		Name:       "stream",
		Version:    8,
		MaxMsgSize: 10 * 1024 * 1024,
		Messages: []interface{}{
			UnsubscribeMsg{},
			OfferedHashesMsg{},
			WantedHashesMsg{},
			TakeoverProofMsg{},
			SubscribeMsg{},
			RetrieveRequestMsg{},
			ChunkDeliveryMsgRetrieval{},
			SubscribeErrorMsg{},
			RequestSubscriptionMsg{},
			QuitMsg{},
			ChunkDeliveryMsgSyncing{},
		},
	}
	r.spec = spec
}

// An accountable message needs some meta information attached to it
// in order to evaluate the correct price
type StreamerPrices struct {
	priceMatrix map[reflect.Type]*protocols.Price
	registry    *Registry
}

// Price implements the accounting interface and returns the price for a specific message
func (sp *StreamerPrices) Price(msg interface{}) *protocols.Price {
	t := reflect.TypeOf(msg).Elem()
	return sp.priceMatrix[t]
}

// Instead of hardcoding the price, get it
// through a function - it could be quite complex in the future
func (sp *StreamerPrices) getRetrieveRequestMsgPrice() uint64 {
	return uint64(1)
}

// Instead of hardcoding the price, get it
// through a function - it could be quite complex in the future
func (sp *StreamerPrices) getChunkDeliveryMsgRetrievalPrice() uint64 {
	return uint64(1)
}

// createPriceOracle sets up a matrix which can be queried to get
// the price for a message via the Price method
func (r *Registry) createPriceOracle() {
	sp := &StreamerPrices{
		registry: r,
	}
	sp.priceMatrix = map[reflect.Type]*protocols.Price{
		reflect.TypeOf(ChunkDeliveryMsgRetrieval{}): {
			Value:   sp.getChunkDeliveryMsgRetrievalPrice(), // arbitrary price for now
			PerByte: true,
			Payer:   protocols.Receiver,
		},
		reflect.TypeOf(RetrieveRequestMsg{}): {
			Value:   sp.getRetrieveRequestMsgPrice(), // arbitrary price for now
			PerByte: false,
			Payer:   protocols.Sender,
		},
	}
	r.prices = sp
}

func (r *Registry) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    r.spec.Name,
			Version: r.spec.Version,
			Length:  r.spec.Length(),
			Run:     r.runProtocol,
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

func (api *API) SubscribeStream(peerId enode.ID, s Stream, history *Range, priority uint8) error {
	return api.streamer.Subscribe(peerId, s, history, priority)
}

func (api *API) UnsubscribeStream(peerId enode.ID, s Stream) error {
	return api.streamer.Unsubscribe(peerId, s)
}

/*
GetPeerSubscriptions is a API function which allows to query a peer for stream subscriptions it has.
It can be called via RPC.
It returns a map of node IDs with an array of string representations of Stream objects.
*/
func (api *API) GetPeerSubscriptions() map[string][]string {
	//create the empty map
	pstreams := make(map[string][]string)

	//iterate all streamer peers
	api.streamer.peersMu.RLock()
	defer api.streamer.peersMu.RUnlock()

	for id, p := range api.streamer.peers {
		var streams []string
		//every peer has a map of stream servers
		//every stream server represents a subscription
		p.serverMu.RLock()
		for s := range p.servers {
			//append the string representation of the stream
			//to the list for this peer
			streams = append(streams, s.String())
		}
		p.serverMu.RUnlock()
		//set the array of stream servers to the map
		pstreams[id.String()] = streams
	}
	return pstreams
}
