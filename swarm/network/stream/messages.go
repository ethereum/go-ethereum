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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	bv "github.com/ethereum/go-ethereum/swarm/network/bitvector"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// SubcribeMsg is the protocol msg for requesting a stream(section)
type SubscribeMsg struct {
	Stream   string
	Key      []byte
	From, To uint64
	Priority uint8 // delivered on priority channel
}

func (p *Peer) handleSubscribeMsg(req *SubscribeMsg) (err error) {
	defer func() {
		if err != nil {
			if e := p.Send(SubscribeErrorMsg{
				Error: err.Error(),
			}); e != nil {
				log.Error("send stream subscribe error message", "err", err)
			}
		}
	}()

	f, err := p.streamer.GetServerFunc(req.Stream)
	if err != nil {
		return err
	}
	s, err := f(p, req.Key)
	if err != nil {
		return err
	}
	os, err := p.setServer(req.Stream, req.Key, s, req.Priority)
	if err != nil {
		return err
	}
	log.Debug("received subscription", "peer", p.ID(), "stream", req.Stream, "Key", req.Key, "from", req.From, "to", req.To)
	go func() {
		if err := p.SendOfferedHashes(os, req.From, req.To); err != nil {
			p.Drop(err)
		}
	}()
	return nil
}

type SubscribeErrorMsg struct {
	Error string
}

func (p *Peer) handleSubscribeErrorMsg(req *SubscribeErrorMsg) (err error) {
	return fmt.Errorf("subscribe to peer %s: %v", p.ID(), req.Error)
}

type UnsubscribeMsg struct {
	Stream string
	Key    []byte
}

func (p *Peer) handleUnsubscribeMsg(req *UnsubscribeMsg) error {
	p.removeServer(req.Stream, req.Key)
	return nil
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
func (m OfferedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v' [%v-%v] (%v)", m.Stream, m.From, m.To, len(m.Hashes)/HashSize)
}

// handleOfferedHashesMsg protocol msg handler calls the incoming streamer interface
// Filter method
func (p *Peer) handleOfferedHashesMsg(req *OfferedHashesMsg) error {
	sk := req.Stream
	sk += keyToString(req.Key)
	s, err := p.getClient(sk)
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
	// done := make(chan bool)
	// go func() {
	// 	wg.Wait()
	// 	close(done)
	// }()
	// go func() {
	// 	select {
	// 	case <-done:
	// 		s.next <- s.batchDone(p, req, hashes)
	// 	case <-time.After(1 * time.Second):
	// 		p.Drop(errors.New("timeout waiting for batch to be delivered"))
	// 	}
	// }()
	go func() {
		wg.Wait()
		s.next <- s.batchDone(p, req, hashes)
	}()
	// only send wantedKeysMsg if all missing chunks of the previous batch arrived
	// except
	if s.live {
		s.sessionAt = req.From
	}
	from, to := s.nextBatch(req.To)
	log.Trace("received offered batch", "peer", p.ID(), "stream", req.Stream, "Key", req.Key, "from", req.From, "to", req.To)
	if from == to {
		return nil
	}

	msg := &WantedHashesMsg{
		Stream: req.Stream,
		Key:    req.Key,
		Want:   want.Bytes(),
		From:   from,
		To:     to,
	}
	go func() {
		select {
		case <-time.After(30 * time.Second):
			p.Drop(err)
			return
		case err := <-s.next:
			if err != nil {
				p.Drop(err)
				return
			}
		}
		log.Trace("sending want batch", "peer", p.ID(), "stream", msg.Stream, "Key", msg.Key, "from", msg.From, "to", msg.To)
		err := p.SendPriority(msg, s.priority)
		if err != nil {
			p.Drop(err)
		}
	}()
	return nil
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
func (m WantedHashesMsg) String() string {
	return fmt.Sprintf("Stream '%v', Want: %x, Next: [%v-%v]", m.Stream, m.Want, m.From, m.To)
}

// handleWantedHashesMsg protocol msg handler
// * sends the next batch of unsynced keys
// * sends the actual data chunks as per WantedHashesMsg
func (p *Peer) handleWantedHashesMsg(req *WantedHashesMsg) error {
	log.Trace("received wanted batch", "peer", p.ID(), "stream", req.Stream, "Key", req.Key, "from", req.From, "to", req.To)
	s, err := p.getServer(req.Stream + keyToString(req.Key))
	if err != nil {
		return err
	}
	hashes := s.currentBatch
	// launch in go routine since GetBatch blocks until new hashes arrive
	go func() {
		if err := p.SendOfferedHashes(s, req.From, req.To); err != nil {
			p.Drop(err)
		}
	}()
	// go p.SendOfferedHashes(s, req.From, req.To)
	l := len(hashes) / HashSize
	want, err := bv.NewFromBytes(req.Want, l)
	if err != nil {
		return fmt.Errorf("error initiaising bitvector of length %v: %v", l, err)
	}
	for i := 0; i < l; i++ {
		if want.Get(i) {
			hash := hashes[i*HashSize : (i+1)*HashSize]
			data, err := s.GetData(hash)
			if err != nil {
				return fmt.Errorf("handleWantedHashesMsg get data %x: %v", hash, err)
			}
			chunk := storage.NewChunk(hash, nil)
			chunk.SData = data
			if err := p.Deliver(chunk, s.priority); err != nil {
				return err
			}
		}
	}
	return nil
}

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
func (m TakeoverProofMsg) String() string {
	return fmt.Sprintf("Stream: '%v' [%v-%v], Root: %x, Sig: %x", m.Stream, m.Start, m.End, m.Root, m.Sig)
}

func (p *Peer) handleTakeoverProofMsg(req *TakeoverProofMsg) error {
	_, err := p.getServer(req.Stream)
	if err != nil {
		return err
	}
	// store the strongest takeoverproof for the stream in streamer
	return nil
}
