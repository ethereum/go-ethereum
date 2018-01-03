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

package whisperv6

import (
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	set "gopkg.in/fatih/set.v0"
)

// peer represents a whisper protocol peer connection.
type Peer struct {
	host *Whisper
	peer *p2p.Peer
	ws   p2p.MsgReadWriter

	trusted        bool
	powRequirement float64
	bloomFilter    []byte // may contain nil in case of full node

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		host:           host,
		peer:           remote,
		ws:             rw,
		trusted:        false,
		powRequirement: 0.0,
		known:          set.New(),
		quit:           make(chan struct{}),
	}
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (p *Peer) start() {
	go p.update()
	log.Trace("start", "peer", p.ID())
}

// stop terminates the peer updater, stopping message forwarding to it.
func (p *Peer) stop() {
	close(p.quit)
	log.Trace("stop", "peer", p.ID())
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (p *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		pow := p.host.MinPow()
		powConverted := math.Float64bits(pow)
		bloom := p.host.BloomFilter()
		errc <- p2p.SendItems(p.ws, statusCode, ProtocolVersion, powConverted, bloom)
	}()

	// Fetch the remote status packet and verify protocol match
	packet, err := p.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer [%x] sent packet %x before status packet", p.ID(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	_, err = s.List()
	if err != nil {
		return fmt.Errorf("peer [%x] sent bad status message: %v", p.ID(), err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("peer [%x] sent bad status message (unable to decode version): %v", p.ID(), err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("peer [%x]: protocol version mismatch %d != %d", p.ID(), peerVersion, ProtocolVersion)
	}

	// only version is mandatory, subsequent parameters are optional
	powRaw, err := s.Uint()
	if err == nil {
		pow := math.Float64frombits(powRaw)
		if math.IsInf(pow, 0) || math.IsNaN(pow) || pow < 0.0 {
			return fmt.Errorf("peer [%x] sent bad status message: invalid pow", p.ID())
		}
		p.powRequirement = pow

		var bloom []byte
		err = s.Decode(&bloom)
		if err == nil {
			sz := len(bloom)
			if sz != bloomFilterSize && sz != 0 {
				return fmt.Errorf("peer [%x] sent bad status message: wrong bloom filter size %d", p.ID(), sz)
			}
			if isFullNode(bloom) {
				p.bloomFilter = nil
			} else {
				p.bloomFilter = bloom
			}
		}
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("peer [%x] failed to send status packet: %v", p.ID(), err)
	}
	return nil
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (p *Peer) update() {
	// Start the tickers for the updates
	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-expire.C:
			p.expire()

		case <-transmit.C:
			if err := p.broadcast(); err != nil {
				log.Trace("broadcast failed", "reason", err, "peer", p.ID())
				return
			}

		case <-p.quit:
			return
		}
	}
}

// mark marks an envelope known to the peer so that it won't be sent back.
func (peer *Peer) mark(envelope *Envelope) {
	peer.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
func (peer *Peer) marked(envelope *Envelope) bool {
	return peer.known.Has(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
func (peer *Peer) expire() {
	unmark := make(map[common.Hash]struct{})
	peer.known.Each(func(v interface{}) bool {
		if !peer.host.isEnvelopeCached(v.(common.Hash)) {
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})
	// Dump all known but no longer cached
	for hash := range unmark {
		peer.known.Remove(hash)
	}
}

// broadcast iterates over the collection of envelopes and transmits yet unknown
// ones over the network.
func (p *Peer) broadcast() error {
	envelopes := p.host.Envelopes()
	bundle := make([]*Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if !p.marked(envelope) && envelope.PoW() >= p.powRequirement && p.bloomMatch(envelope) {
			bundle = append(bundle, envelope)
		}
	}

	if len(bundle) > 0 {
		// transmit the batch of envelopes
		if err := p2p.Send(p.ws, messagesCode, bundle); err != nil {
			return err
		}

		// mark envelopes only if they were successfully sent
		for _, e := range bundle {
			p.mark(e)
		}

		log.Trace("broadcast", "num. messages", len(bundle))
	}
	return nil
}

func (p *Peer) ID() []byte {
	id := p.peer.ID()
	return id[:]
}

func (p *Peer) notifyAboutPowRequirementChange(pow float64) error {
	i := math.Float64bits(pow)
	return p2p.Send(p.ws, powRequirementCode, i)
}

func (p *Peer) notifyAboutBloomFilterChange(bloom []byte) error {
	return p2p.Send(p.ws, bloomFilterExCode, bloom)
}

func (p *Peer) bloomMatch(env *Envelope) bool {
	if p.bloomFilter == nil {
		// no filter - full node, accepts all envelops
		return true
	}

	return bloomFilterMatch(p.bloomFilter, env.Bloom())
}
