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

package whisperv5

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	set "gopkg.in/fatih/set.v0"
)

// peer represents a whisper protocol peer connection.
type Peer struct {
	host    *Whisper
	peer    *p2p.Peer
	ws      p2p.MsgReadWriter
	trusted bool

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		host:    host,
		peer:    remote,
		ws:      rw,
		trusted: false,
		known:   set.New(),
		quit:    make(chan struct{}),
	}
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (self *Peer) start() {
	go self.update()
	glog.V(logger.Debug).Infof("%v: whisper started", self.peer)
}

// stop terminates the peer updater, stopping message forwarding to it.
func (self *Peer) stop() {
	close(self.quit)
	glog.V(logger.Debug).Infof("%v: whisper stopped", self.peer)
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (self *Peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(self.ws, statusCode, ProtocolVersion)
	}()
	// Fetch the remote status packet and verify protocol match
	packet, err := self.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer sent %x before status packet", packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	if _, err := s.List(); err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", peerVersion, ProtocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		return fmt.Errorf("failed to send status packet: %v", err)
	}
	return nil
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (self *Peer) update() {
	// Start the tickers for the updates
	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-expire.C:
			self.expire()

		case <-transmit.C:
			if err := self.broadcast(); err != nil {
				glog.V(logger.Info).Infof("%v: broadcast failed: %v", self.peer, err)
				return
			}

		case <-self.quit:
			return
		}
	}
}

// mark marks an envelope known to the peer so that it won't be sent back.
func (self *Peer) mark(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
func (self *Peer) marked(envelope *Envelope) bool {
	return self.known.Has(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
func (self *Peer) expire() {
	// Assemble the list of available envelopes
	available := set.NewNonTS()
	for _, envelope := range self.host.Envelopes() {
		available.Add(envelope.Hash())
	}
	// Cross reference availability with known status
	unmark := make(map[common.Hash]struct{})
	self.known.Each(func(v interface{}) bool {
		if !available.Has(v.(common.Hash)) {
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})
	// Dump all known but unavailable
	for hash, _ := range unmark {
		self.known.Remove(hash)
	}
}

// broadcast iterates over the collection of envelopes and transmits yet unknown
// ones over the network.
func (self *Peer) broadcast() error {
	// Fetch the envelopes and collect the unknown ones
	envelopes := self.host.Envelopes()
	transmit := make([]*Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if !self.marked(envelope) {
			transmit = append(transmit, envelope)
			self.mark(envelope)
		}
	}
	// Transmit the unknown batch (potentially empty)
	if err := p2p.Send(self.ws, messagesCode, transmit); err != nil {
		return err
	}
	glog.V(logger.Detail).Infoln(self.peer, "broadcasted", len(transmit), "message(s)")
	return nil
}
