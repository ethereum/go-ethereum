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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	bv "github.com/ethereum/go-ethereum/swarm/network/bitvector"
	pq "github.com/ethereum/go-ethereum/swarm/network/priorityqueue"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	HashSize = 32

	Low uint8 = iota
	Mid
	High
	Top
	PriorityQueue        // number of queues
	PriorityQueueCap = 3 // queue capacity
)

// Stream is string descriptor of the stream
type Stream string

// Handover represents a statement that the upstream peer hands over the stream section
type Handover struct {
	Stream     Stream // name of stream
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
	Stream   Stream
	Key      []byte
	From, To uint64
	Priority uint8 // delivered on priority channel
}

// UnsyncedKeysMsg is the protocol msg for offering to hand over a
// stream section
type UnsyncedKeysMsg struct {
	Stream         Stream // name of Stream
	Key            []byte // subtype or key
	From, To       uint64 // peer and db-specific entry count
	Hashes         []byte // stream of hashes (128)
	*HandoverProof        // HandoverProof
}

/*
 store requests are put in netstore so they are stored and then
 forwarded to the peers in their kademlia proximity bin by the syncer
*/
type ChunkDeliveryMsg struct {
	Key   storage.Key
	SData []byte // the stored chunk Data (incl size)
	// optional
	Id   uint64 // request ID. if delivery, the ID is retrieve request ID
	from Peer   // [not serialised] protocol registers the requester
}

// String pretty prints UnsyncedKeysMsg
func (self UnsyncedKeysMsg) String() string {
	return fmt.Sprintf("Stream '%v' [%v-%v] (%v)", self.Stream, self.From, self.To, len(self.Hashes)/HashSize)
}

// WantedKeysMsg is the protocol msg data for signaling which hashes
// offered in UnsyncedKeysMsg downstream peer actually wants sent over
type WantedKeysMsg struct {
	Stream   Stream // name of stream
	Key      []byte // subtype or key
	Want     []byte // bitvector indicating which keys of the batch needed
	From, To uint64 // next interval offset - empty if not to be continued
}

// String pretty prints WantedKeysMsg
func (self WantedKeysMsg) String() string {
	return fmt.Sprintf("Stream '%v', Want: %x, Next: [%v-%v]", self.Stream, self.Want, self.From, self.To)
}

// Streamer registry for outgoing and incoming streamer constructors
type Streamer struct {
	incomingLock sync.RWMutex
	outgoingLock sync.RWMutex
	outgoing     map[Stream]func(*StreamerPeer, []byte) (OutgoingStreamer, error)
	incoming     map[Stream]func(*StreamerPeer, []byte) (IncomingStreamer, error)

	dbAccess *DbAccess
	overlay  Overlay
	receiveC chan *ChunkDeliveryMsg
}

// NewStreamer is Streamer constructor
func NewStreamer(overlay Overlay, dbAccess *DbAccess) *Streamer {
	return &Streamer{
		outgoing: make(map[Stream]func(*StreamerPeer, []byte) (OutgoingStreamer, error)),
		incoming: make(map[Stream]func(*StreamerPeer, []byte) (IncomingStreamer, error)),
		dbAccess: dbAccess,
		overlay:  overlay,
		receiveC: make(chan *ChunkDeliveryMsg, 10),
	}
}

// RegisterIncomingStreamer registers an incoming streamer constructor
func (self *Streamer) RegisterIncomingStreamer(stream Stream, f func(*StreamerPeer, []byte) (IncomingStreamer, error)) {
	self.incomingLock.Lock()
	defer self.incomingLock.Unlock()
	self.incoming[stream] = f
}

// RegisterOutgoingStreamer registers an outgoing streamer constructor
func (self *Streamer) RegisterOutgoingStreamer(stream Stream, f func(*StreamerPeer, []byte) (OutgoingStreamer, error)) {
	self.outgoingLock.Lock()
	defer self.outgoingLock.Unlock()
	self.outgoing[stream] = f
}

// GetIncomingStreamer accessor for incoming streamer constructors
func (self *Streamer) GetIncomingStreamer(stream Stream) (func(*StreamerPeer, []byte) (IncomingStreamer, error), error) {
	self.incomingLock.RLock()
	defer self.incomingLock.RUnlock()
	f := self.incoming[stream]
	if f == nil {
		return nil, fmt.Errorf("stream %v not registered", stream)
	}
	return f, nil
}

// GetOutgoingStreamer accessor for incoming streamer constructors
func (self *Streamer) GetOutgoingStreamer(stream Stream) (func(*StreamerPeer, []byte) (OutgoingStreamer, error), error) {
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
}

// OutgoingStreamer interface for outgoing peer Streamer
type OutgoingStreamer interface {
	SetNextBatch(uint64, uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error)
	GetData([]byte) []byte
}

type incomingStreamer struct {
	IncomingStreamer
	priority uint8
	quit     chan struct{}
	next     chan struct{}
}

// IncomingStreamer interface for incoming peer Streamer
type IncomingStreamer interface {
	NextBatch(uint64) (uint64, uint64)
	NeedData([]byte) func()
	BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error)
}

// StreamerPeer is the Peer extention for the streaming protocol
type StreamerPeer struct {
	Peer
	streamer *Streamer
	pq       *pq.PriorityQueue
	//netStore     storage.ChunkStore
	dbAccess     *DbAccess
	outgoingLock sync.RWMutex
	incomingLock sync.RWMutex
	outgoing     map[Stream]*outgoingStreamer
	incoming     map[Stream]*incomingStreamer
	quit         chan struct{}
}

// type IncomingStreamer struct {
// 	priority uint8
// 	peer     *StreamerPeer
// }

// type OutgoingStreamer struct {
// 	priority uint8
// 	peer     *StreamerPeer
// }

// NewStreamerPeer is the constructor for StreamerPeer
func NewStreamerPeer(p Peer, streamer *Streamer) *StreamerPeer {
	self := &StreamerPeer{
		pq:       pq.New(int(PriorityQueue), PriorityQueueCap),
		streamer: streamer,
		outgoing: make(map[Stream]*outgoingStreamer),
		incoming: make(map[Stream]*incomingStreamer),
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

// RetrieveRequestMsg is the protocol msg for chunk retrieve requests
type RetrieveRequestMsg struct {
	Key storage.Key
}

// RetrieveRequestStreamer implements OutgoingStreamer
type RetrieveRequestStreamer struct {
	deliveryC  chan *storage.Chunk
	batchC     chan []byte
	db         *DbAccess
	currentLen uint64
}

// RegisterRequestStreamer registers outgoing and incoming streamers for request handling
func RegisterRequestStreamer(streamer *Streamer, db *DbAccess) {
	streamer.RegisterOutgoingStreamer(retrieveRequestStream, func(_ *StreamerPeer, t []byte) (OutgoingStreamer, error) {
		return NewRetrieveRequestStreamer(db), nil
	})
	streamer.RegisterIncomingStreamer(retrieveRequestStream, func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return NewIncomingSwarmSyncer(nil, p, db, nil)
	})
}

// NewRetrieveRequestStreamer is RetrieveRequestStreamer constructor
func NewRetrieveRequestStreamer(db *DbAccess) *RetrieveRequestStreamer {
	s := &RetrieveRequestStreamer{
		deliveryC: make(chan *storage.Chunk),
		batchC:    make(chan []byte),
		db:        db,
	}
	go s.processDeliveries()
	return s
}

// processDeliveries handles delivered chunk hashes
func (s *RetrieveRequestStreamer) processDeliveries() {
	var hashes []byte
	for {
		select {
		case delivery := <-s.deliveryC:
			hashes = append(hashes, delivery.Key[:]...)
		case s.batchC <- hashes:
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
	chunk, _ := s.db.get(storage.Key(key))
	return chunk.SData
}

const retrieveRequestStream = Stream("RETRIEVE_REQUEST")

func (self *StreamerPeer) handleRetrieveRequestMsg(req *RetrieveRequestMsg) error {
	chunk, created := self.dbAccess.getOrCreateRequest(req.Key)
	s, err := self.getOutgoingStreamer(retrieveRequestStream)
	if err != nil {
		return err
	}
	streamer := s.OutgoingStreamer.(*RetrieveRequestStreamer)
	if chunk.ReqC != nil {
		if created {
			if err := self.streamer.Retrieve(chunk); err != nil {
				return err
			}
		}
		go func() {
			t := time.NewTicker(3 * time.Minute)
			defer t.Stop()

			select {
			case <-chunk.ReqC:
			case <-self.quit:
				return
			case <-t.C:
				return
			}

			streamer.deliveryC <- chunk
		}()
		return nil
	}
	// TODO: call the retrieve function of the outgoing syncer
	streamer.deliveryC <- chunk
	return nil
}

// Retrieve sends a chunk retrieve request to
func (self *Streamer) Retrieve(chunk *storage.Chunk) error {
	self.overlay.EachConn(chunk.Key[:], 255, func(p OverlayConn, po int, nn bool) bool {
		sp := p.(*StreamerPeer)
		// TODO: skip light nodes that do not accept retrieve requests
		sp.SendPriority(&RetrieveRequestMsg{
			Key: chunk.Key[:],
		}, Top)
		return false
	})
	return nil
}

func (self *StreamerPeer) handleChunkDeliveryMsg(req *ChunkDeliveryMsg) error {
	chunk, err := self.dbAccess.get(req.Key)
	if err != nil {
		return err
	}

	self.streamer.receiveC <- req

	log.Trace(fmt.Sprintf("delivery of %v from %v", chunk, self))
	return nil
}

func (self *Streamer) processReceivedChunks() {
	for {
		select {
		case req := <-self.receiveC:
			chunk, err := self.dbAccess.get(req.Key)
			if err != nil {
				continue
			}
			chunk.SData = req.SData
			self.dbAccess.put(chunk)
			close(chunk.ReqC)
		}
	}
}

func (self *StreamerPeer) getOutgoingStreamer(s Stream) (*outgoingStreamer, error) {
	self.outgoingLock.RLock()
	defer self.outgoingLock.RUnlock()
	streamer := self.outgoing[s]
	if streamer == nil {
		return nil, fmt.Errorf("stream '%v' not provided", s)
	}
	return streamer, nil
}

func (self *StreamerPeer) getIncomingStreamer(s Stream) (*incomingStreamer, error) {
	self.incomingLock.RLock()
	defer self.incomingLock.RUnlock()
	streamer := self.incoming[s]
	if streamer == nil {
		return nil, fmt.Errorf("stream '%v' not provided", s)
	}
	return streamer, nil
}

func (self *StreamerPeer) setOutgoingStreamer(s Stream, o OutgoingStreamer, priority uint8) (*outgoingStreamer, error) {
	self.outgoingLock.Lock()
	defer self.outgoingLock.Unlock()
	if self.outgoing[s] != nil {
		return nil, fmt.Errorf("stream %v already registered", s)
	}
	os := &outgoingStreamer{
		OutgoingStreamer: o,
		priority:         priority,
	}
	self.outgoing[s] = os
	return os, nil
}

func (self *StreamerPeer) setIncomingStreamer(s Stream, i IncomingStreamer, priority uint8) error {
	self.incomingLock.Lock()
	defer self.incomingLock.Unlock()
	if self.incoming[s] != nil {
		return fmt.Errorf("stream %v already registered", s)
	}
	next := make(chan struct{}, 1)
	self.incoming[s] = &incomingStreamer{
		IncomingStreamer: i,
		priority:         priority,
		next:             next,
	}
	next <- struct{}{} // this is to allow wantedKeysMsg before first batch arrives
	return nil
}

// Subscribe initiates the streamer
func (self *StreamerPeer) Subscribe(s Stream, t []byte, from, to uint64, priority uint8) error {
	f, err := self.streamer.GetIncomingStreamer(s)
	if err != nil {
		return err
	}
	is, err := f(self, t)
	if err != nil {
		return err
	}
	err = self.setIncomingStreamer(s, is, priority)
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
	self.SendPriority(msg, priority)
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
	key := string(req.Stream) + string(req.Key)
	os, err := self.setOutgoingStreamer(Stream(key), s, req.Priority)
	if err != nil {
		return nil
	}
	go self.SendUnsyncedKeys(os, req.From, req.To)
	return nil
}

// handleUnsyncedKeysMsg protocol msg handler calls the incoming streamer interface
// Filter method
func (self *StreamerPeer) handleUnsyncedKeysMsg(req *UnsyncedKeysMsg) error {
	s, err := self.getIncomingStreamer(req.Stream)
	if err != nil {
		return err
	}
	hashes := req.Hashes
	want, err := bv.New(len(hashes) / HashSize)
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for i := 0; i < len(hashes)/HashSize; i += HashSize {
		hash := hashes[i : i+HashSize]
		if wait := s.NeedData(hash); wait != nil {
			want.Set(i, true)
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
	from, to := s.NextBatch(req.To)
	if from == to {
		return nil
	}
	msg := &WantedKeysMsg{
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

// handleWantedKeysMsg protocol msg handler
// * sends the next batch of unsynced keys
// * sends the actual data chunks as per WantedKeysMsg
func (self *StreamerPeer) handleWantedKeysMsg(req *WantedKeysMsg) error {
	s, err := self.getOutgoingStreamer(req.Stream)
	if err != nil {
		return err
	}
	hashes := s.currentBatch
	// launch in go routine since GetBatch blocks until new hashes arrive
	go self.SendUnsyncedKeys(s, req.From, req.To)
	l := len(hashes) / HashSize
	want, err := bv.NewFromBytes(req.Want, l)
	if err != nil {
		return err
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

// UnsyncedKeys sends UnsyncedKeysMsg protocol msg
func (self *StreamerPeer) SendUnsyncedKeys(s *outgoingStreamer, f, t uint64) error {
	hashes, from, to, proof, err := s.SetNextBatch(f, t)
	if err != nil {
		return err
	}
	s.currentBatch = hashes
	msg := &UnsyncedKeysMsg{
		HandoverProof: proof,
		Hashes:        hashes,
		From:          from,
		To:            to,
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
		UnsyncedKeysMsg{},
		WantedKeysMsg{},
		TakeoverProofMsg{},
		SubscribeMsg{},
	},
}

// Run protocol run function
func (s *Streamer) Run(p *bzzPeer) error {
	sp := NewStreamerPeer(p, s)
	// load saved intervals
	// autosubscribe to request handler to serve request only for non-light nodes
	// sp.handleSubscribeMsg(&SubscribeMsg{
	// 	Stream:   retrieveRequestStream,
	// 	Priority: uint8(Top),
	// })
	// subscribe to request handling ; only with non-light nodes
	sp.Subscribe(retrieveRequestStream, nil, 0, 0, Top)
	defer close(sp.quit)
	return sp.Run(sp.HandleMsg)
}

// HandleMsg is the message handler that delegates incoming messages
func (self *StreamerPeer) HandleMsg(msg interface{}) error {
	switch msg := msg.(type) {

	case *SubscribeMsg:
		return self.handleSubscribeMsg(msg)

	case *UnsyncedKeysMsg:
		return self.handleUnsyncedKeysMsg(msg)

	case *TakeoverProofMsg:
		return self.handleTakeoverProofMsg(msg)

	case *WantedKeysMsg:
		return self.handleWantedKeysMsg(msg)

	case *ChunkDeliveryMsg:
		return self.handleChunkDeliveryMsg(msg)

	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}
