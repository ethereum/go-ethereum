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
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/log"
	bv "github.com/ethereum/go-ethereum/swarm/network/bitvector"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/opentracing/opentracing-go"
)

var syncBatchTimeout = 30 * time.Second

// Stream defines a unique stream identifier.
type Stream struct {
	// Name is used for Client and Server functions identification.
	Name string
	// Key is the name of specific stream data.
	Key string
	// Live defines whether the stream delivers only new data
	// for the specific stream.
	Live bool
}

func NewStream(name string, key string, live bool) Stream {
	return Stream{
		Name: name,
		Key:  key,
		Live: live,
	}
}

// String return a stream id based on all Stream fields.
func (s Stream) String() string {
	t := "h"
	if s.Live {
		t = "l"
	}
	return fmt.Sprintf("%s|%s|%s", s.Name, s.Key, t)
}

// SubcribeMsg is the protocol msg for requesting a stream(section)
type SubscribeMsg struct {
	Stream   Stream
	History  *Range `rlp:"nil"`
	Priority uint8  // delivered on priority channel
}

// RequestSubscriptionMsg is the protocol msg for a node to request subscription to a
// specific stream
type RequestSubscriptionMsg struct {
	Stream   Stream
	History  *Range `rlp:"nil"`
	Priority uint8  // delivered on priority channel
}

func (p *Peer) handleRequestSubscription(ctx context.Context, req *RequestSubscriptionMsg) (err error) {
	log.Debug(fmt.Sprintf("handleRequestSubscription: streamer %s to subscribe to %s with stream %s", p.streamer.addr, p.ID(), req.Stream))
	if err = p.streamer.Subscribe(p.ID(), req.Stream, req.History, req.Priority); err != nil {
		// The error will be sent as a subscribe error message
		// and will not be returned as it will prevent any new message
		// exchange between peers over p2p. Instead, error will be returned
		// only if there is one from sending subscribe error message.
		err = p.Send(ctx, SubscribeErrorMsg{
			Error: err.Error(),
		})
	}
	return err
}

func (p *Peer) handleSubscribeMsg(ctx context.Context, req *SubscribeMsg) (err error) {
	metrics.GetOrRegisterCounter("peer.handlesubscribemsg", nil).Inc(1)

	defer func() {
		if err != nil {
			// The error will be sent as a subscribe error message
			// and will not be returned as it will prevent any new message
			// exchange between peers over p2p. Instead, error will be returned
			// only if there is one from sending subscribe error message.
			err = p.Send(context.TODO(), SubscribeErrorMsg{
				Error: err.Error(),
			})
		}
	}()

	log.Debug("received subscription", "from", p.streamer.addr, "peer", p.ID(), "stream", req.Stream, "history", req.History)

	f, err := p.streamer.GetServerFunc(req.Stream.Name)
	if err != nil {
		return err
	}

	s, err := f(p, req.Stream.Key, req.Stream.Live)
	if err != nil {
		return err
	}
	os, err := p.setServer(req.Stream, s, req.Priority)
	if err != nil {
		return err
	}

	var from uint64
	var to uint64
	if !req.Stream.Live && req.History != nil {
		from = req.History.From
		to = req.History.To
	}

	go func() {
		if err := p.SendOfferedHashes(os, from, to); err != nil {
			log.Warn("SendOfferedHashes error", "peer", p.ID().TerminalString(), "err", err)
		}
	}()

	if req.Stream.Live && req.History != nil {
		// subscribe to the history stream
		s, err := f(p, req.Stream.Key, false)
		if err != nil {
			return err
		}

		os, err := p.setServer(getHistoryStream(req.Stream), s, getHistoryPriority(req.Priority))
		if err != nil {
			return err
		}
		go func() {
			if err := p.SendOfferedHashes(os, req.History.From, req.History.To); err != nil {
				log.Warn("SendOfferedHashes error", "peer", p.ID().TerminalString(), "err", err)
			}
		}()
	}

	return nil
}

type SubscribeErrorMsg struct {
	Error string
}

func (p *Peer) handleSubscribeErrorMsg(req *SubscribeErrorMsg) (err error) {
	//TODO the error should be channeled to whoever calls the subscribe
	return fmt.Errorf("subscribe to peer %s: %v", p.ID(), req.Error)
}

type UnsubscribeMsg struct {
	Stream Stream
}

func (p *Peer) handleUnsubscribeMsg(req *UnsubscribeMsg) error {
	return p.removeServer(req.Stream)
}

type QuitMsg struct {
	Stream Stream
}

func (p *Peer) handleQuitMsg(req *QuitMsg) error {
	return p.removeClient(req.Stream)
}

// OfferedHashesMsg is the protocol msg for offering to hand over a
// stream section
type OfferedHashesMsg struct {
	Stream         Stream // name of Stream
	From, To       uint64 // peer and db-specific entry count
	Hashes         []byte // stream of hashes (128)
	*HandoverProof        // HandoverProof
}

// String pretty prints OfferedHashesMsg
func (m OfferedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v' [%v-%v] (%v)", m.Stream, m.From, m.To, len(m.Hashes)/HashSize)
}

// handleOfferedHashesMsg protocol msg handler calls the incoming streamer interface
// Filter method
func (p *Peer) handleOfferedHashesMsg(ctx context.Context, req *OfferedHashesMsg) error {
	metrics.GetOrRegisterCounter("peer.handleofferedhashes", nil).Inc(1)

	var sp opentracing.Span
	ctx, sp = spancontext.StartSpan(
		ctx,
		"handle.offered.hashes")
	defer sp.Finish()

	c, _, err := p.getOrSetClient(req.Stream, req.From, req.To)
	if err != nil {
		return err
	}

	hashes := req.Hashes
	lenHashes := len(hashes)
	if lenHashes%HashSize != 0 {
		return fmt.Errorf("error invalid hashes length (len: %v)", lenHashes)
	}

	want, err := bv.New(lenHashes / HashSize)
	if err != nil {
		return fmt.Errorf("error initiaising bitvector of length %v: %v", lenHashes/HashSize, err)
	}

	ctr := 0
	errC := make(chan error)
	ctx, cancel := context.WithTimeout(ctx, syncBatchTimeout)

	ctx = context.WithValue(ctx, "source", p.ID().String())
	for i := 0; i < lenHashes; i += HashSize {
		hash := hashes[i : i+HashSize]

		if wait := c.NeedData(ctx, hash); wait != nil {
			ctr++
			want.Set(i/HashSize, true)
			// create request and wait until the chunk data arrives and is stored
			go func(w func(context.Context) error) {
				select {
				case errC <- w(ctx):
				case <-ctx.Done():
				}
			}(wait)
		}
	}

	go func() {
		defer cancel()
		for i := 0; i < ctr; i++ {
			select {
			case err := <-errC:
				if err != nil {
					log.Debug("client.handleOfferedHashesMsg() error waiting for chunk, dropping peer", "peer", p.ID(), "err", err)
					p.Drop(err)
					return
				}
			case <-ctx.Done():
				log.Debug("client.handleOfferedHashesMsg() context done", "ctx.Err()", ctx.Err())
				return
			case <-c.quit:
				log.Debug("client.handleOfferedHashesMsg() quit")
				return
			}
		}
		select {
		case c.next <- c.batchDone(p, req, hashes):
		case <-c.quit:
			log.Debug("client.handleOfferedHashesMsg() quit")
		case <-ctx.Done():
			log.Debug("client.handleOfferedHashesMsg() context done", "ctx.Err()", ctx.Err())
		}
	}()
	// only send wantedKeysMsg if all missing chunks of the previous batch arrived
	// except
	if c.stream.Live {
		c.sessionAt = req.From
	}
	from, to := c.nextBatch(req.To + 1)
	log.Trace("set next batch", "peer", p.ID(), "stream", req.Stream, "from", req.From, "to", req.To, "addr", p.streamer.addr)
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
		log.Trace("sending want batch", "peer", p.ID(), "stream", msg.Stream, "from", msg.From, "to", msg.To)
		select {
		case err := <-c.next:
			if err != nil {
				log.Warn("c.next error dropping peer", "err", err)
				p.Drop(err)
				return
			}
		case <-c.quit:
			log.Debug("client.handleOfferedHashesMsg() quit")
			return
		case <-ctx.Done():
			log.Debug("client.handleOfferedHashesMsg() context done", "ctx.Err()", ctx.Err())
			return
		}
		log.Trace("sending want batch", "peer", p.ID(), "stream", msg.Stream, "from", msg.From, "to", msg.To)
		err := p.SendPriority(ctx, msg, c.priority, "")
		if err != nil {
			log.Warn("SendPriority error", "err", err)
		}
	}()
	return nil
}

// WantedHashesMsg is the protocol msg data for signaling which hashes
// offered in OfferedHashesMsg downstream peer actually wants sent over
type WantedHashesMsg struct {
	Stream   Stream
	Want     []byte // bitvector indicating which keys of the batch needed
	From, To uint64 // next interval offset - empty if not to be continued
}

// String pretty prints WantedHashesMsg
func (m WantedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v', Want: %x, Next: [%v-%v]", m.Stream, m.Want, m.From, m.To)
}

// handleWantedHashesMsg protocol msg handler
// * sends the next batch of unsynced keys
// * sends the actual data chunks as per WantedHashesMsg
func (p *Peer) handleWantedHashesMsg(ctx context.Context, req *WantedHashesMsg) error {
	metrics.GetOrRegisterCounter("peer.handlewantedhashesmsg", nil).Inc(1)

	log.Trace("received wanted batch", "peer", p.ID(), "stream", req.Stream, "from", req.From, "to", req.To)
	s, err := p.getServer(req.Stream)
	if err != nil {
		return err
	}
	hashes := s.currentBatch
	// launch in go routine since GetBatch blocks until new hashes arrive
	go func() {
		if err := p.SendOfferedHashes(s, req.From, req.To); err != nil {
			log.Warn("SendOfferedHashes error", "peer", p.ID().TerminalString(), "err", err)
		}
	}()
	// go p.SendOfferedHashes(s, req.From, req.To)
	l := len(hashes) / HashSize

	log.Trace("wanted batch length", "peer", p.ID(), "stream", req.Stream, "from", req.From, "to", req.To, "lenhashes", len(hashes), "l", l)
	want, err := bv.NewFromBytes(req.Want, l)
	if err != nil {
		return fmt.Errorf("error initiaising bitvector of length %v: %v", l, err)
	}
	for i := 0; i < l; i++ {
		if want.Get(i) {
			metrics.GetOrRegisterCounter("peer.handlewantedhashesmsg.actualget", nil).Inc(1)

			hash := hashes[i*HashSize : (i+1)*HashSize]
			data, err := s.GetData(ctx, hash)
			if err != nil {
				return fmt.Errorf("handleWantedHashesMsg get data %x: %v", hash, err)
			}
			chunk := storage.NewChunk(hash, data)
			syncing := true
			if err := p.Deliver(ctx, chunk, s.priority, syncing); err != nil {
				return err
			}
		}
	}
	return nil
}

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
func (m TakeoverProofMsg) String() string {
	return fmt.Sprintf("Stream: '%v' [%v-%v], Root: %x, Sig: %x", m.Stream, m.Start, m.End, m.Root, m.Sig)
}

func (p *Peer) handleTakeoverProofMsg(ctx context.Context, req *TakeoverProofMsg) error {
	_, err := p.getServer(req.Stream)
	// store the strongest takeoverproof for the stream in streamer
	return err
}
