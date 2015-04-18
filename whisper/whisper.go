package whisper

import (
	"crypto/ecdsa"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/event/filter"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/fatih/set.v0"
)

const (
	statusCode   = 0x00
	messagesCode = 0x01

	protocolVersion uint64 = 0x02
	protocolName           = "shh"

	signatureFlag   = byte(1 << 7)
	signatureLength = 65

	expirationCycle   = 800 * time.Millisecond
	transmissionCycle = 300 * time.Millisecond
)

const (
	DefaultTTL = 50 * time.Second
	DefaultPoW = 50 * time.Millisecond
)

type MessageEvent struct {
	To      *ecdsa.PrivateKey
	From    *ecdsa.PublicKey
	Message *Message
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol p2p.Protocol
	filters  *filter.Filters

	keys map[string]*ecdsa.PrivateKey

	messages    map[common.Hash]*Envelope // Pool of messages currently tracked by this node
	expirations map[uint32]*set.SetNonTS  // Message expiration pool (TODO: something lighter)
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	quit chan struct{}
}

func New() *Whisper {
	whisper := &Whisper{
		filters:     filter.New(),
		keys:        make(map[string]*ecdsa.PrivateKey),
		messages:    make(map[common.Hash]*Envelope),
		expirations: make(map[uint32]*set.SetNonTS),
		peers:       make(map[*peer]struct{}),
		quit:        make(chan struct{}),
	}
	whisper.filters.Start()

	// p2p whisper sub protocol handler
	whisper.protocol = p2p.Protocol{
		Name:    protocolName,
		Version: uint(protocolVersion),
		Length:  2,
		Run:     whisper.handlePeer,
	}

	return whisper
}

// Protocol returns the whisper sub-protocol handler for this particular client.
func (self *Whisper) Protocol() p2p.Protocol {
	return self.protocol
}

// Version returns the whisper sub-protocols version number.
func (self *Whisper) Version() uint {
	return self.protocol.Version
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (self *Whisper) NewIdentity() *ecdsa.PrivateKey {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}
	self.keys[string(crypto.FromECDSAPub(&key.PublicKey))] = key

	return key
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (self *Whisper) HasIdentity(key *ecdsa.PublicKey) bool {
	return self.keys[string(crypto.FromECDSAPub(key))] != nil
}

// GetIdentity retrieves the private key of the specified public identity.
func (self *Whisper) GetIdentity(key *ecdsa.PublicKey) *ecdsa.PrivateKey {
	return self.keys[string(crypto.FromECDSAPub(key))]
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
func (self *Whisper) Watch(options Filter) int {
	filter := filter.Generic{
		Str1: string(crypto.FromECDSAPub(options.To)),
		Str2: string(crypto.FromECDSAPub(options.From)),
		Data: newTopicSet(options.Topics),
		Fn: func(data interface{}) {
			options.Fn(data.(*Message))
		},
	}
	return self.filters.Install(filter)
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

func (self *Whisper) Start() {
	glog.V(logger.Info).Infoln("Whisper started")
	go self.update()
}

func (self *Whisper) Stop() {
	close(self.quit)
	glog.V(logger.Info).Infoln("Whisper stopped")
}

// Messages retrieves the currently pooled messages matching a filter id.
func (self *Whisper) Messages(id int) []*Message {
	messages := make([]*Message, 0)
	if filter := self.filters.Get(id); filter != nil {
		for _, envelope := range self.messages {
			if message := self.open(envelope); message != nil {
				if self.filters.Match(filter, createFilter(message, envelope.Topics)) {
					messages = append(messages, message)
				}
			}
		}
	}
	return messages
}

// func (self *Whisper) RemoveIdentity(key *ecdsa.PublicKey) bool {
// 	k := string(crypto.FromECDSAPub(key))
// 	if _, ok := self.keys[k]; ok {
// 		delete(self.keys, k)
// 		return true
// 	}
// 	return false
// }

// handlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (self *Whisper) handlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create, initialize and start the whisper peer
	whisperPeer, err := newPeer(self, peer, rw)
	if err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	// Start tracking the active peer
	self.peerMu.Lock()
	self.peers[whisperPeer] = struct{}{}
	self.peerMu.Unlock()

	defer func() {
		self.peerMu.Lock()
		delete(self.peers, whisperPeer)
		self.peerMu.Unlock()
	}()
	// Read and process inbound messages directly to merge into client-global state
	for {
		// Fetch the next packet and decode the contained envelopes
		packet, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		var envelopes []*Envelope
		if err := packet.Decode(&envelopes); err != nil {
			peer.Infof("failed to decode enveloped: %v", err)
			continue
		}
		// Inject all envelopes into the internal pool
		for _, envelope := range envelopes {
			if err := self.add(envelope); err != nil {
				// TODO Punish peer here. Invalid envelope.
				peer.Debugf("failed to pool envelope: %f", err)
			}
			whisperPeer.mark(envelope)
		}
	}
}

// add inserts a new envelope into the message pool to be distributed within the
// whisper network. It also inserts the envelope into the expiration pool at the
// appropriate time-stamp.
func (self *Whisper) add(envelope *Envelope) error {
	self.poolMu.Lock()
	defer self.poolMu.Unlock()

	// Insert the message into the tracked pool
	hash := envelope.Hash()
	if _, ok := self.messages[hash]; ok {
		glog.V(logger.Detail).Infof("whisper envelope already cached: %x\n", envelope)
		return nil
	}
	self.messages[hash] = envelope

	// Insert the message into the expiration pool for later removal
	if self.expirations[envelope.Expiry] == nil {
		self.expirations[envelope.Expiry] = set.NewNonTS()
	}
	if !self.expirations[envelope.Expiry].Has(hash) {
		self.expirations[envelope.Expiry].Add(hash)

		// Notify the local node of a message arrival
		go self.postEvent(envelope)
	}
	glog.V(logger.Detail).Infof("cached whisper envelope %x\n", envelope)

	return nil
}

// postEvent opens an envelope with the configured identities and delivers the
// message upstream from application processing.
func (self *Whisper) postEvent(envelope *Envelope) {
	if message := self.open(envelope); message != nil {
		self.filters.Notify(createFilter(message, envelope.Topics), message)
	}
}

// open tries to decrypt a whisper envelope with all the configured identities,
// returning the decrypted message and the key used to achieve it. If not keys
// are configured, open will return the payload as if non encrypted.
func (self *Whisper) open(envelope *Envelope) *Message {
	// Short circuit if no identity is set, and assume clear-text
	if len(self.keys) == 0 {
		if message, err := envelope.Open(nil); err == nil {
			return message
		}
	}
	// Iterate over the keys and try to decrypt the message
	for _, key := range self.keys {
		message, err := envelope.Open(key)
		if err == nil || err == ecies.ErrInvalidPublicKey {
			message.To = &key.PublicKey
			return message
		}
	}
	// Failed to decrypt, don't return anything
	return nil
}

// createFilter creates a message filter to check against installed handlers.
func createFilter(message *Message, topics []Topic) filter.Filter {
	return filter.Generic{
		Str1: string(crypto.FromECDSAPub(message.To)),
		Str2: string(crypto.FromECDSAPub(message.Recover())),
		Data: newTopicSet(topics),
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
			delete(self.messages, v.(common.Hash))
			return true
		})
		self.expirations[then].Clear()
	}
}

// envelopes retrieves all the messages currently pooled by the node.
func (self *Whisper) envelopes() []*Envelope {
	self.poolMu.RLock()
	defer self.poolMu.RUnlock()

	envelopes := make([]*Envelope, 0, len(self.messages))
	for _, envelope := range self.messages {
		envelopes = append(envelopes, envelope)
	}
	return envelopes
}
