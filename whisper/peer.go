package whisper

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"gopkg.in/fatih/set.v0"
)

// peer represents a whisper protocol peer connection.
type peer struct {
	host *Whisper
	peer *p2p.Peer
	ws   p2p.MsgReadWriter

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		host:  host,
		peer:  remote,
		ws:    rw,
		known: set.New(),
		quit:  make(chan struct{}),
	}
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (self *peer) start() {
	go self.update()
	glog.V(logger.Debug).Infof("%v: whisper started", self.peer)
}

// stop terminates the peer updater, stopping message forwarding to it.
func (self *peer) stop() {
	close(self.quit)
	glog.V(logger.Debug).Infof("%v: whisper stopped", self.peer)
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (self *peer) handshake() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.SendItems(self.ws, statusCode, protocolVersion)
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
	if peerVersion != protocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", peerVersion, protocolVersion)
	}
	// Wait until out own status is consumed too
	if err := <-errc; err != nil {
		return fmt.Errorf("failed to send status packet: %v", err)
	}
	return nil
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (self *peer) update() {
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
func (self *peer) mark(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
func (self *peer) marked(envelope *Envelope) bool {
	return self.known.Has(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
func (self *peer) expire() {
	// Assemble the list of available envelopes
	available := set.NewNonTS()
	for _, envelope := range self.host.envelopes() {
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
func (self *peer) broadcast() error {
	// Fetch the envelopes and collect the unknown ones
	envelopes := self.host.envelopes()
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
