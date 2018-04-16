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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	set "gopkg.in/fatih/set.v0"
)

// PeerBase represents a whisper protocol peer connection.
type PeerBase struct {
	host *Whisper
	ws   p2p.MsgReadWriter

	trusted        bool
	powRequirement float64
	bloomMu        sync.Mutex
	bloomFilter    []byte
	fullNode       bool

	known *set.Set // Messages already known by the peer to avoid wasting bandwidth

	quit chan struct{}
}

// Peer is an abstract representation of a whisper peer. It could
// represent a devp2p peer or a libp2p peer.
type Peer interface {
	ID() string
	start()
	stop()
	handshake() error
	mark(*Envelope)
	marked(*Envelope) bool
	expire()
	broadcast() error
	notifyAboutPowRequirementChange(pow float64) error
	notifyAboutBloomFilterChange([]byte) error
	bloomMatch(*Envelope) bool
	setBloomFilter([]byte)
	isTrusted() bool
	setTrusted(bool)
	setPoWRequirement(float64)
	stream() p2p.MsgReadWriter
}

func (peer *PeerBase) handshakeBase() error {
	// Send the handshake status message asynchronously
	errc := make(chan error, 1)
	go func() {
		pow := peer.host.MinPow()
		powConverted := math.Float64bits(pow)
		bloom := peer.host.BloomFilter()
		errc <- p2p.SendItems(peer.ws, statusCode, ProtocolVersion, powConverted, bloom)
	}()

	// Fetch the remote status packet and verify protocol match
	packet, err := peer.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer [%s] sent packet %x before status packet", peer.ID(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	_, err = s.List()
	if err != nil {
		return fmt.Errorf("peer [%s] sent bad status message: %v", peer.ID(), err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("peer [%s] sent bad status message (unable to decode version): %v", peer.ID(), err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("peer [%s]: protocol version mismatch %d != %d", peer.ID(), peerVersion, ProtocolVersion)
	}

	// only version is mandatory, subsequent parameters are optional
	powRaw, err := s.Uint()
	if err == nil {
		pow := math.Float64frombits(powRaw)
		if math.IsInf(pow, 0) || math.IsNaN(pow) || pow < 0.0 {
			return fmt.Errorf("peer [%s] sent bad status message: invalid pow", peer.ID())
		}
		peer.powRequirement = pow

		var bloom []byte
		err = s.Decode(&bloom)
		if err == nil {
			sz := len(bloom)
			if sz != BloomFilterSize && sz != 0 {
				return fmt.Errorf("peer [%s] sent bad status message: wrong bloom filter size %d", peer.ID(), sz)
			}
			peer.setBloomFilter(bloom)
		}
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("peer [%s] failed to send status packet: %v", peer.ID(), err)
	}
	return nil
}

// mark marks an envelope known to the peer so that it won't be sent back.
func (peer *PeerBase) mark(envelope *Envelope) {
	peer.known.Add(envelope.Hash())
}

// marked checks if an envelope is already known to the remote peer.
func (peer *PeerBase) marked(envelope *Envelope) bool {
	return peer.known.Has(envelope.Hash())
}

// expire iterates over all the known envelopes in the host and removes all
// expired (unknown) ones from the known list.
func (peer *PeerBase) expire() {
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
func (peer *PeerBase) broadcast() error {
	envelopes := peer.host.Envelopes()
	bundle := make([]*Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if !peer.marked(envelope) && envelope.PoW() >= peer.powRequirement && peer.bloomMatch(envelope) {
			bundle = append(bundle, envelope)
		}
	}

	if len(bundle) > 0 {
		// transmit the batch of envelopes
		if err := p2p.Send(peer.ws, messagesCode, bundle); err != nil {
			return err
		}

		// mark envelopes only if they were successfully sent
		for _, e := range bundle {
			peer.mark(e)
		}

		log.Trace("broadcast", "num. messages", len(bundle))
	}
	return nil
}

// ID returns a peer's id
func (peer *DevP2PPeer) ID() string {
	return peer.peer.ID().String()
}

func (peer *PeerBase) notifyAboutPowRequirementChange(pow float64) error {
	i := math.Float64bits(pow)
	return p2p.Send(peer.ws, powRequirementCode, i)
}

func (peer *PeerBase) notifyAboutBloomFilterChange(bloom []byte) error {
	return p2p.Send(peer.ws, bloomFilterExCode, bloom)
}

func (peer *PeerBase) bloomMatch(env *Envelope) bool {
	peer.bloomMu.Lock()
	defer peer.bloomMu.Unlock()
	return peer.fullNode || BloomFilterMatch(peer.bloomFilter, env.Bloom())
}

func (peer *PeerBase) setBloomFilter(bloom []byte) {
	peer.bloomMu.Lock()
	defer peer.bloomMu.Unlock()
	peer.bloomFilter = bloom
	peer.fullNode = isFullNode(bloom)
	if peer.fullNode && peer.bloomFilter == nil {
		peer.bloomFilter = MakeFullNodeBloom()
	}
}

func (peer *PeerBase) isTrusted() bool {
	return peer.trusted
}

func (peer *PeerBase) setTrusted(t bool) {
	peer.trusted = t
}

func (peer *PeerBase) setPoWRequirement(r float64) {
	peer.powRequirement = r
}

func (peer *PeerBase) stream() p2p.MsgReadWriter {
	return peer.ws
}

func MakeFullNodeBloom() []byte {
	bloom := make([]byte, BloomFilterSize)
	for i := 0; i < BloomFilterSize; i++ {
		bloom[i] = 0xFF
	}
	return bloom
}
