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
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	bv "github.com/ethereum/go-ethereum/swarm/network/bitvector"
	pq "github.com/ethereum/go-ethereum/swarm/network/priorityqueue"
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

// Handover represents a statement that the upstream peer hands over the stream section
type Handover struct {
	Stream     string // name of stream
	Start, End uint64 // index of hashes
	Root       []byte // Root hash for indexed segment inclusion proofs
}

// HandoverProof represents a signed statement that the upstream peer handed over the stream section
type HandoverProof struct {
	Sig []byte // Sign(Hash(Serialisation(Handover)))
	*Handover
}

// Takeover represents a statement that downstream peer took over (stored all data)
// handed over
type Takeover Handover

//  TakeoverProof represents a signed statement that the downstream peer took over
// the stream section
type TakeoverProof struct {
	Sig []byte // Sign(Hash(Serialisation(Takeover)))
	*Takeover
}

// TakeoverProofMsg is the protocol msg sent by downstream peer
type TakeoverProofMsg TakeoverProof

// String pretty prints TakeoverProofMsg
func (self TakeoverProofMsg) String() string {
	return fmt.Sprintf("Stream: '%v' [%v-%v], Root: %x, Sig: %x", self.Stream, self.Start, self.End, self.Root, self.Sig)
}

// SubcribeMsg is the protocol msg for requesting a stream(section)
type SubscribeMsg struct {
	Stream   string
	Key      []byte
	From, To uint64
	Priority uint8 // delivered on priority channel
}

// OfferedHashesMsg is the protocol msg for offering to hand over a
// stream section
type OfferedHashesMsg struct {
	Stream         string // name of Stream
	Key            []byte // subtype or key
	From, To       uint64 // peer and db-specific entry count
	Hashes         []byte // stream of hashes (128)
	*HandoverProof        // HandoverProof
}

// String pretty prints OfferedHashesMsg
func (self OfferedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v' [%v-%v] (%v)", self.Stream, self.From, self.To, len(self.Hashes)/HashSize)
}

// WantedHashesMsg is the protocol msg data for signaling which hashes
// offered in OfferedHashesMsg downstream peer actually wants sent over
type WantedHashesMsg struct {
	Stream   string // name of stream
	Key      []byte // subtype or key
	Want     []byte // bitvector indicating which keys of the batch needed
	From, To uint64 // next interval offset - empty if not to be continued
}

// String pretty prints WantedHashesMsg
func (self WantedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v', Want: %x, Next: [%v-%v]", self.Stream, self.Want, self.From, self.To)
}

// Streamer registry for outgoing and incoming streamer constructors
type Streamer struct {
	incomingLock sync.RWMutex
	outgoingLock sync.RWMutex
	peersLock    sync.RWMutex
	outgoing     map[string]func(*StreamerPeer, []byte) (OutgoingStreamer, error)
	incoming     map[string]func(*StreamerPeer, []byte) (IncomingStreamer, error)
	peers        map[discover.NodeID]*StreamerPeer
	delivery     *Delivery
}

// NewStreamer is Streamer constructor
func NewStreamer(delivery *Delivery) *Streamer {
	streamer := &Streamer{
		outgoing: make(map[string]func(*StreamerPeer, []byte) (OutgoingStreamer, error)),
		incoming: make(map[string]func(*StreamerPeer, []byte) (IncomingStreamer, error)),
		peers:    make(map[discover.NodeID]*StreamerPeer),
		delivery: delivery,
	}
	delivery.getPeer = streamer.getPeer
	streamer.RegisterOutgoingStreamer(retrieveRequestStream, func(_ *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		return NewRetrieveRequestStreamer(delivery.dbAccess), nil
	})
	streamer.RegisterIncomingStreamer(retrieveRequestStream, func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return NewIncomingSwarmSyncer(p, delivery.dbAccess, nil)
	})
	return streamer
}

// RegisterIncomingStreamer registers an incoming streamer constructor
func (self *Streamer) RegisterIncomingStreamer(stream string, f func(*StreamerPeer, []byte) (IncomingStreamer, error)) {
	self.incomingLock.Lock()
	defer self.incomingLock.Unlock()
	self.incoming[stream] = f
}

// RegisterOutgoingStreamer registers an outgoing streamer constructor
func (self *Streamer) RegisterOutgoingStreamer(stream string, f func(*StreamerPeer, []byte) (OutgoingStreamer, error)) {
	self.outgoingLock.Lock()
	defer self.outgoingLock.Unlock()
	self.outgoing[stream] = f
}

// GetIncomingStreamer accessor for incoming streamer constructors
func (self *Streamer) GetIncomingStreamer(stream string) (func(*StreamerPeer, []byte) (IncomingStreamer, error), error) {
	self.incomingLock.RLock()
	defer self.incomingLock.RUnlock()
	f := self.incoming[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// GetOutgoingStreamer accessor for incoming streamer constructors
func (self *Streamer) GetOutgoingStreamer(stream string) (func(*StreamerPeer, []byte) (OutgoingStreamer, error), error) {
	self.outgoingLock.RLock()
	defer self.outgoingLock.RUnlock()
	f := self.outgoing[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

func (self *Streamer) NodeInfo() interface{} {
	return nil
}

func (self *Streamer) PeerInfo(id discover.NodeID) interface{} {
	return nil
}

type outgoingStreamer struct {
	OutgoingStreamer
	priority     uint8
	currentBatch []byte
	stream       string
}

// OutgoingStreamer interface for outgoing peer Streamer
type OutgoingStreamer interface {
	SetNextBatch(uint64, uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error)
	GetData([]byte) []byte
}

type incomingStreamer struct {
	IncomingStreamer
	priority  uint8
	sessionAt uint64
	live      bool
	quit      chan struct{}
	next      chan struct{}
}

// IncomingStreamer interface for incoming peer Streamer
type IncomingStreamer interface {
	NeedData([]byte) func()
	BatchDone(string, uint64, []byte, []byte) func() (*TakeoverProof, error)
}

// StreamerPeer is the Peer extention for the streaming protocol
type StreamerPeer struct {
	Peer
	streamer *Streamer
	pq       *pq.PriorityQueue
	//netStore     storage.ChunkStore
	outgoingLock sync.RWMutex
	incomingLock sync.RWMutex
	outgoing     map[string]*outgoingStreamer
	incoming     map[string]*incomingStreamer
	quit         chan struct{}
}

// NewStreamerPeer is the constructor for StreamerPeer
func NewStreamerPeer(p Peer, streamer *Streamer) *StreamerPeer {
	self := &StreamerPeer{
		Peer:     p,
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: streamer,
		outgoing: make(map[string]*outgoingStreamer),
		incoming: make(map[string]*incomingStreamer),
		quit:     make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	go self.pq.Run(ctx, func(i interface{}) { p.Send(i) })
	go func() {
		<-self.quit
		cancel()
	}()
	return self
}

func (self *Streamer) getPeer(peerId discover.NodeID) *StreamerPeer {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	return self.peers[peerId]
}

func (self *Streamer) setPeer(peer *StreamerPeer) {
	self.peersLock.Lock()
	self.peers[peer.ID()] = peer
	self.peersLock.Unlock()
}

func (self *Streamer) deletePeer(peer *StreamerPeer) {
	self.peersLock.Lock()
	delete(self.peers, peer.ID())
	self.peersLock.Unlock()
}

func (self *StreamerPeer) getOutgoingStreamer(s string) (*outgoingStreamer, error) {
	self.outgoingLock.RLock()
	defer self.outgoingLock.RUnlock()
	streamer := self.outgoing[s]
	if streamer == nil {
		return nil, fmt.Errorf("stream '%v' not provided", s)
	}
	return streamer, nil
}

func (self *StreamerPeer) getIncomingStreamer(s string) (*incomingStreamer, error) {
	self.incomingLock.RLock()
	defer self.incomingLock.RUnlock()
	streamer := self.incoming[s]
	if streamer == nil {
		return nil, fmt.Errorf("stream '%v' not provided", s)
	}
	return streamer, nil
}

func (self *StreamerPeer) setOutgoingStreamer(s string, o OutgoingStreamer, priority uint8) (*outgoingStreamer, error) {
	self.outgoingLock.Lock()
	defer self.outgoingLock.Unlock()
	if self.outgoing[s] != nil {
		return nil, fmt.Errorf("stream %v already registered", s)
	}
	os := &outgoingStreamer{
		OutgoingStreamer: o,
		priority:         priority,
		stream:           s,
	}
	self.outgoing[s] = os
	return os, nil
}

func (self *StreamerPeer) setIncomingStreamer(s string, i IncomingStreamer, priority uint8, live bool) error {
	self.incomingLock.Lock()
	defer self.incomingLock.Unlock()
	if self.incoming[s] != nil {
		return fmt.Errorf("stream %v already registered", s)
	}
	next := make(chan struct{}, 1)
	// var intervals *Intervals
	// if !live {
	// key := s + self.ID().String()
	// intervals = NewIntervals(key, self.streamer)
	// }
	self.incoming[s] = &incomingStreamer{
		IncomingStreamer: i,
		// intervals:        intervals,
		live:     live,
		priority: priority,
		next:     next,
	}
	next <- struct{}{} // this is to allow wantedKeysMsg before first batch arrives
	return nil
}

// NextBatch adjusts the indexes by inspecting the intervals
func (self *incomingStreamer) nextBatch(from uint64) (nextFrom uint64, nextTo uint64) {
	var intervals []uint64
	if self.live {
		if len(intervals) == 0 {
			intervals = []uint64{self.sessionAt, from}
		} else {
			intervals[1] = from
		}
		nextFrom = from
	} else if from >= self.sessionAt { // history sync complete
		intervals = nil
	} else if len(intervals) > 2 && from >= intervals[2] { // filled a gap in the intervals
		intervals = append(intervals[:1], intervals[3:]...)
		nextFrom = intervals[1]
		if len(intervals) > 2 {
			nextTo = intervals[2]
		} else {
			nextTo = self.sessionAt
		}
	} else {
		nextFrom = from
		intervals[1] = from
		nextTo = self.sessionAt
	}
	// self.intervals.set(intervals)
	return nextFrom, nextTo
}

// Subscribe initiates the streamer
func (self *Streamer) Subscribe(peerId discover.NodeID, s string, t []byte, from, to uint64, priority uint8, live bool) error {
	f, err := self.GetIncomingStreamer(s)
	if err != nil {
		return err
	}

	peer := self.getPeer(peerId)
	if peer == nil {
		return fmt.Errorf("peer not found %v", peerId)
	}

	is, err := f(peer, t)
	if err != nil {
		return err
	}
	err = peer.setIncomingStreamer(s, is, priority, live)
	if err != nil {
		return err
	}

	msg := &SubscribeMsg{
		Stream:   s,
		Key:      t,
		From:     from,
		To:       to,
		Priority: priority,
	}
	peer.SendPriority(msg, priority)
	return nil
}

func (self *StreamerPeer) handleSubscribeMsg(req *SubscribeMsg) error {
	f, err := self.streamer.GetOutgoingStreamer(req.Stream)
	if err != nil {
		return err
	}
	s, err := f(self, req.Key)
	if err != nil {
		return err
	}
	key := req.Stream + string(req.Key)
	os, err := self.setOutgoingStreamer(key, s, req.Priority)
	if err != nil {
		return nil
	}
	go self.SendOfferedHashes(os, req.From, req.To)
	return nil
}

// handleOfferedHashesMsg protocol msg handler calls the incoming streamer interface
// Filter method
func (self *StreamerPeer) handleOfferedHashesMsg(req *OfferedHashesMsg) error {
	s, err := self.getIncomingStreamer(req.Stream)
	if err != nil {
		return err
	}
	hashes := req.Hashes
	want, err := bv.New(len(hashes) / HashSize)
	if err != nil {
		return fmt.Errorf("error initiaising bitvector of length %v: %v", len(hashes)/HashSize, err)
	}
	wg := sync.WaitGroup{}
	for i := 0; i < len(hashes); i += HashSize {
		hash := hashes[i : i+HashSize]
		if wait := s.NeedData(hash); wait != nil {
			want.Set(i/HashSize, true)
			wg.Add(1)
			// create request and wait until the chunk data arrives and is stored
			go func(w func()) {
				w()
				wg.Done()
			}(wait)
		}
	}
	go func() {
		wg.Wait()
		if tf := s.BatchDone(req.Stream, req.From, hashes, req.Root); tf != nil {
			tp, err := tf()
			if err != nil {
				return
			}
			self.SendPriority(tp, s.priority)
		}
		s.next <- struct{}{}
	}()
	// only send wantedKeysMsg if all missing chunks of the previous batch arrived
	// except
	if s.live {
		s.sessionAt = req.From
	}
	from, to := s.nextBatch(req.To)
	if from == to {
		return nil
	}
	msg := &WantedHashesMsg{
		Stream: req.Stream,
		Want:   want.Bytes(),
		From:   from,
		To:     to,
	}
	go func() {
		select {
		case <-s.next:
		case <-s.quit:
			return
		}
		self.SendPriority(msg, s.priority)
	}()
	return nil
}

// handleWantedHashesMsg protocol msg handler
// * sends the next batch of unsynced keys
// * sends the actual data chunks as per WantedHashesMsg
func (self *StreamerPeer) handleWantedHashesMsg(req *WantedHashesMsg) error {
	s, err := self.getOutgoingStreamer(req.Stream)
	if err != nil {
		return err
	}
	hashes := s.currentBatch
	// launch in go routine since GetBatch blocks until new hashes arrive
	go self.SendOfferedHashes(s, req.From, req.To)
	l := len(hashes) / HashSize
	want, err := bv.NewFromBytes(req.Want, l)
	if err != nil {
		return fmt.Errorf("error initiaising bitvector of length %v: %v", l, err)
	}
	for i := 0; i < l; i++ {
		if want.Get(i) {
			hash := hashes[i*HashSize : (i+1)*HashSize]
			data := s.GetData(hash)
			if data == nil {
				return errors.New("not found")
			}
			chunk := storage.NewChunk(hash, nil)
			chunk.SData = data
			if err := self.Deliver(chunk, s.priority); err != nil {
				return err
			}
		}
	}
	return nil
}

func (self *StreamerPeer) handleTakeoverProofMsg(req *TakeoverProofMsg) error {
	_, err := self.getOutgoingStreamer(req.Stream)
	if err != nil {
		return err
	}
	// store the strongest takeoverproof for the stream in streamer
	return nil
}

// Deliver sends a storeRequestMsg protocol message to the peer
func (self *StreamerPeer) Deliver(chunk *storage.Chunk, priority uint8) error {
	msg := &ChunkDeliveryMsg{
		Key:   chunk.Key,
		SData: chunk.SData,
	}
	return self.pq.Push(nil, msg, int(priority))
}

// Deliver sends a storeRequestMsg protocol message to the peer
func (self *StreamerPeer) SendPriority(msg interface{}, priority uint8) error {
	return self.pq.Push(nil, msg, int(priority))
}

// SendOfferedHashes sends OfferedHashesMsg protocol msg
func (self *StreamerPeer) SendOfferedHashes(s *outgoingStreamer, f, t uint64) error {
	hashes, from, to, proof, err := s.SetNextBatch(f, t)
	if err != nil {
		return err
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
		// TODO: use real key here
		Key: []byte{},
	}
	return self.SendPriority(msg, s.priority)
}

// StreamerSpec is the spec of the streamer protocol.
var StreamerSpec = &protocols.Spec{
	Name:       "stream",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		HandshakeMsg{},
		OfferedHashesMsg{},
		WantedHashesMsg{},
		TakeoverProofMsg{},
		SubscribeMsg{},
		RetrieveRequestMsg{},
		ChunkDeliveryMsg{},
	},
}

// Run protocol run function
func (s *Streamer) Run(p *bzzPeer) error {
	sp := NewStreamerPeer(p, s)
	// load saved intervals

	s.setPeer(sp)

	defer s.deletePeer(sp)
	defer close(sp.quit)
	return sp.Run(sp.HandleMsg)
}

// HandleMsg is the message handler that delegates incoming messages
func (self *StreamerPeer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *SubscribeMsg:
		return self.handleSubscribeMsg(msg)

	case *OfferedHashesMsg:
		return self.handleOfferedHashesMsg(msg)

	case *TakeoverProofMsg:
		return self.handleTakeoverProofMsg(msg)

	case *WantedHashesMsg:
		return self.handleWantedHashesMsg(msg)

	case *ChunkDeliveryMsg:
		return self.streamer.delivery.handleChunkDeliveryMsg(msg)

	case *RetrieveRequestMsg:
		return self.streamer.delivery.handleRetrieveRequestMsg(self, msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}
