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
	"bytes"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/pbkdf2"
	set "gopkg.in/fatih/set.v0"
)

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol p2p.Protocol
	filters  *Filters

	privateKeys map[string]*ecdsa.PrivateKey
	topicKeys   map[string][]byte
	keyMu       sync.RWMutex

	envelopes   map[common.Hash]*Envelope        // Pool of messages currently tracked by this node
	messages    map[common.Hash]*ReceivedMessage // Pool of successfully decrypted messages, not expired yet
	expirations map[uint32]*set.SetNonTS         // Message expiration pool
	poolMu      sync.RWMutex                     // Mutex to sync the message and expiration pools

	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	mailServer MailServer

	quit chan struct{}
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
// Param s should be passed if you want to implement mail server, otherwise nil.
func New(server MailServer) *Whisper {
	whisper := &Whisper{
		privateKeys: make(map[string]*ecdsa.PrivateKey),
		topicKeys:   make(map[string][]byte),
		envelopes:   make(map[common.Hash]*Envelope),
		messages:    make(map[common.Hash]*ReceivedMessage),
		expirations: make(map[uint32]*set.SetNonTS),
		peers:       make(map[*Peer]struct{}),
		mailServer:  server,
		quit:        make(chan struct{}),
	}
	whisper.filters = NewFilters(whisper)

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    ProtocolName,
		Version: uint(ProtocolVersion),
		Length:  NumberOfMessageCodes,
		Run:     whisper.HandlePeer,
	}

	return whisper
}

// Protocols returns the whisper sub-protocols ran by this particular client.
func (self *Whisper) Protocols() []p2p.Protocol {
	return []p2p.Protocol{self.protocol}
}

// Version returns the whisper sub-protocols version number.
func (self *Whisper) Version() uint {
	return self.protocol.Version
}

func (self *Whisper) GetFilter(id int) *Filter {
	return self.filters.Get(id)
}

func (self *Whisper) getPeer(peerID []byte) (*Peer, error) {
	self.peerMu.Lock()
	defer self.peerMu.Unlock()
	for p, _ := range self.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Could not find peer with ID: %x", peerID)
}

// MarkPeerTrusted marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (self *Whisper) MarkPeerTrusted(peerID []byte) error {
	p, err := self.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	return nil
}

func (self *Whisper) RequestHistoricMessages(peerID []byte, data []byte) error {
	wp, err := self.getPeer(peerID)
	if err != nil {
		return err
	}
	wp.trusted = true
	return p2p.Send(wp.ws, mailRequestCode, data)
}

func (self *Whisper) SendP2PMessage(peerID []byte, envelope *Envelope) error {
	wp, err := self.getPeer(peerID)
	if err != nil {
		return err
	}
	return p2p.Send(wp.ws, p2pCode, envelope)
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (self *Whisper) NewIdentity() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil || !validatePrivateKey(key) {
		key, err = crypto.GenerateKey() // retry once
	}
	if err != nil {
		panic(err)
	}
	if !validatePrivateKey(key) {
		panic("Failed to generate valid key")
	}
	self.keyMu.Lock()
	defer self.keyMu.Unlock()
	self.privateKeys[string(crypto.FromECDSAPub(&key.PublicKey))] = key
	return key
}

// DeleteIdentity deletes the specifies key if it exists.
func (self *Whisper) DeleteIdentity(key string) {
	self.keyMu.Lock()
	defer self.keyMu.Unlock()
	delete(self.privateKeys, key)
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (self *Whisper) HasIdentity(key *ecdsa.PublicKey) bool {
	self.keyMu.RLock()
	defer self.keyMu.RUnlock()
	return self.privateKeys[string(crypto.FromECDSAPub(key))] != nil
}

// GetIdentity retrieves the private key of the specified public identity.
func (self *Whisper) GetIdentity(key *ecdsa.PublicKey) *ecdsa.PrivateKey {
	self.keyMu.RLock()
	defer self.keyMu.RUnlock()
	return self.privateKeys[string(crypto.FromECDSAPub(key))]
}

func (self *Whisper) GenerateTopicKey(name string) error {
	if self.HasTopicKey(name) {
		return fmt.Errorf("Key with name [%s] already exists", name)
	}

	key := make([]byte, aesKeyLength)
	_, err := crand.Read(key) // todo: check how safe is this function
	if err != nil {
		return err
	} else if !validateSymmetricKey(key) {
		return fmt.Errorf("crypto/rand failed to generate valid key")
	}

	self.keyMu.Lock()
	defer self.keyMu.Unlock()
	self.topicKeys[name] = key
	return nil
}

func (self *Whisper) AddTopicKey(name string, key []byte) error {
	if self.HasTopicKey(name) {
		return fmt.Errorf("Key with name [%s] already exists", name)
	}

	derived, err := deriveKeyMaterial(key, EnvelopeVersion)
	if err != nil {
		return err
	}

	self.keyMu.Lock()
	defer self.keyMu.Unlock()
	self.topicKeys[name] = derived
	return nil
}

func (self *Whisper) HasTopicKey(name string) bool {
	self.keyMu.RLock()
	defer self.keyMu.RUnlock()
	return self.topicKeys[name] != nil
}

func (self *Whisper) DeleteTopicKey(name string) {
	self.keyMu.Lock()
	defer self.keyMu.Unlock()
	delete(self.topicKeys, name)
}

func (self *Whisper) GetTopicKey(name string) []byte {
	self.keyMu.RLock()
	defer self.keyMu.RUnlock()
	return self.topicKeys[name]
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
func (self *Whisper) Watch(f *Filter) int {
	return self.filters.Install(f)
}

// Unwatch removes an installed message handler.
func (self *Whisper) Unwatch(id int) {
	self.filters.Uninstall(id)
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
func (self *Whisper) Send(envelope *Envelope) error {
	return self.add(envelope)
}

// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (self *Whisper) Start(*p2p.Server) error {
	glog.V(logger.Info).Infoln("Whisper started")
	go self.update()
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (self *Whisper) Stop() error {
	close(self.quit)
	glog.V(logger.Info).Infoln("Whisper stopped")
	return nil
}

// handlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (self *Whisper) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(self, peer, rw)

	self.peerMu.Lock()
	self.peers[whisperPeer] = struct{}{}
	self.peerMu.Unlock()

	defer func() {
		self.peerMu.Lock()
		delete(self.peers, whisperPeer)
		self.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	return self.runMessageLoop(whisperPeer, rw)
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (self *Whisper) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			return err
		}

		switch packet.Code {
		case statusCode:
			// this should not happen, but no need to panic; just ignore this message.
			glog.V(logger.Warn).Infof("%v: unxepected status message received", p.peer)
		case messagesCode:
			// decode the contained envelopes
			var envelopes []*Envelope
			if err := packet.Decode(&envelopes); err != nil {
				glog.V(logger.Warn).Infof("%v: failed to decode envelope: [%v], peer will be disconnected", p.peer, err)
				return fmt.Errorf("garbage received")
			}
			// inject all envelopes into the internal pool
			for _, envelope := range envelopes {
				if err := self.add(envelope); err != nil {
					glog.V(logger.Warn).Infof("%v: bad envelope received: [%v], peer will be disconnected", p.peer, err)
					return fmt.Errorf("invalid envelope")
				}
				p.mark(envelope)
				if self.mailServer != nil {
					self.mailServer.Archive(envelope)
				}
			}
		case p2pCode:
			// peer-to-peer message, sent directly to peer bypassing PoW checks, etc.
			// this message is not supposed to be forwarded to other peers, and
			// therefore might not satisfy the PoW, expiry and other requirements.
			// these messages are only accepted from the trusted peer.
			if p.trusted {
				var envelopes []*Envelope
				if err := packet.Decode(&envelopes); err != nil {
					glog.V(logger.Warn).Infof("%v: failed to decode direct message: [%v], peer will be disconnected", p.peer, err)
					return fmt.Errorf("garbage received (directMessage)")
				}
				for _, envelope := range envelopes {
					self.postEvent(envelope, p2pCode)
				}
			}
		case mailRequestCode:
			// Must be processed if mail server is implemented. Otherwise ignore.
			if self.mailServer != nil {
				s := rlp.NewStream(packet.Payload, uint64(packet.Size))
				data, err := s.Bytes()
				if err == nil {
					self.mailServer.DeliverMail(p, data)
				} else {
					glog.V(logger.Error).Infof("%v: bad requestHistoricMessages received: [%v]", p.peer, err)
				}
			}
		default:
			// New message types might be implemented in the future versions of Whisper.
			// For forward compatibility, just ignore.
		}

		packet.Discard()
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// whisper network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp. In case of error, connection should be dropped.
func (self *Whisper) add(envelope *Envelope) error {
	now := uint32(time.Now().Unix())
	sent := envelope.Expiry - envelope.TTL

	if sent > now {
		if sent+SynchAllowance > now {
			return fmt.Errorf("message created in the future")
		} else {
			// recalculate PoW, adjusted for the time difference, plus one second for latency
			envelope.calculatePoW(sent - now + 1)
		}
	}

	if envelope.Expiry < now {
		if envelope.Expiry+SynchAllowance*2 < now {
			return fmt.Errorf("very old message")
		} else {
			return nil // drop envelope without error
		}
	}

	if len(envelope.Data) > MaxMessageLength {
		return fmt.Errorf("huge messages are not allowed")
	}

	if envelope.PoW() < MinimumPoW {
		glog.V(logger.Debug).Infof("envelope with low PoW dropped: %f", envelope.PoW())
		return nil // drop envelope without error
	}

	hash := envelope.Hash()

	self.poolMu.Lock()
	_, alreadyCached := self.envelopes[hash]
	if !alreadyCached {
		self.envelopes[hash] = envelope
		if self.expirations[envelope.Expiry] == nil {
			self.expirations[envelope.Expiry] = set.NewNonTS()
		}
		if !self.expirations[envelope.Expiry].Has(hash) {
			self.expirations[envelope.Expiry].Add(hash)
		}
	}
	self.poolMu.Unlock()

	if alreadyCached {
		glog.V(logger.Detail).Infof("whisper envelope already cached: %x\n", envelope)
	} else {
		self.postEvent(envelope, messagesCode) // notify the local node about the new message
		glog.V(logger.Detail).Infof("cached whisper envelope %x\n", envelope)
	}
	return nil
}

// postEvent delivers the message to the watchers.
func (self *Whisper) postEvent(envelope *Envelope, messageCode uint64) {
	// if the version of incoming message is higher than
	// currently supported version, we can not decrypt it,
	// and therefore just ignore this message
	if envelope.Ver() <= EnvelopeVersion {
		// todo: review if you need an additional thread here
		go self.filters.NotifyWatchers(envelope, messageCode)
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (self *Whisper) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			self.expire()

		case <-self.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (self *Whisper) expire() {
	self.poolMu.Lock()
	defer self.poolMu.Unlock()

	now := uint32(time.Now().Unix())
	for then, hashSet := range self.expirations {
		// Short circuit if a future time
		if then > now {
			continue
		}
		// Dump all expired messages and remove timestamp
		hashSet.Each(func(v interface{}) bool {
			delete(self.envelopes, v.(common.Hash))
			delete(self.messages, v.(common.Hash))
			return true
		})
		self.expirations[then].Clear()
	}
}

// envelopes retrieves all the messages currently pooled by the node.
func (self *Whisper) Envelopes() []*Envelope {
	self.poolMu.RLock()
	defer self.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(self.envelopes))
	for _, envelope := range self.envelopes {
		all = append(all, envelope)
	}
	return all
}

// Messages retrieves all the currently pooled messages matching a filter id.
func (self *Whisper) Messages(id int) []*ReceivedMessage {
	self.poolMu.RLock()
	defer self.poolMu.RUnlock()

	result := make([]*ReceivedMessage, 0)
	if filter := self.filters.Get(id); filter != nil {
		for _, msg := range self.messages {
			if filter.MatchMessage(msg) {
				result = append(result, msg)
			}
		}
	}
	return result
}

func (self *Whisper) addDecryptedMessage(msg *ReceivedMessage) {
	self.poolMu.Lock()
	defer self.poolMu.Unlock()

	self.messages[msg.EnvelopeHash] = msg
}

func ValidatePublicKey(k *ecdsa.PublicKey) bool {
	return k != nil && k.X != nil && k.Y != nil && k.X.Sign() != 0 && k.Y.Sign() != 0
}

func validatePrivateKey(k *ecdsa.PrivateKey) bool {
	if k == nil || k.D == nil || k.D.Sign() == 0 {
		return false
	}
	return ValidatePublicKey(&k.PublicKey)
}

// validateSymmetricKey returns false if the key contains all zeros
func validateSymmetricKey(k []byte) bool {
	return len(k) > 0 && !containsOnlyZeros(k)
}

func containsOnlyZeros(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return true
		}
	}
	return false
}

func bytesToIntLittleEndian(b []byte) (res uint64) {
	mul := uint64(1)
	for i := 0; i < len(b); i++ {
		res += uint64(b[i]) * mul
		mul *= 256
	}
	return res
}

func BytesToIntBigEndian(b []byte) (res uint64) {
	for i := 0; i < len(b); i++ {
		res *= 256
		res += uint64(b[i])
	}
	return res
}

// DeriveSymmetricKey derives symmetric key material from the key or password.
// pbkdf2 is used for security, in case people use password instead of randomly generated keys.
func deriveKeyMaterial(key []byte, version uint64) (derivedKey []byte, err error) {
	if version == 0 {
		// todo: review: kdf should run no less than 1 sec, because it's a once in a session experience
		derivedKey := pbkdf2.Key(key, nil, 65356, aesKeyLength, sha256.New)
		return derivedKey, nil
	} else {
		return nil, unknownVersionError(version)
	}
}
