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
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/crypto/pbkdf2"
	set "gopkg.in/fatih/set.v0"
)

type Statistics struct {
	messagesCleared      int
	memoryCleared        int
	memoryUsed           int
	cycles               int
	totalMessagesCleared int
}

// Whisper represents a dark communication interface through the Ethereum
// network, using its very own P2P communication layer.
type Whisper struct {
	protocol p2p.Protocol
	filters  *Filters

	privateKeys map[string]*ecdsa.PrivateKey
	//identities map[string]*ecdsa.PrivateKey
	symKeys map[string][]byte
	keyMu   sync.RWMutex

	envelopes   map[common.Hash]*Envelope // Pool of envelopes currently tracked by this node
	expirations map[uint32]*set.SetNonTS  // Message expiration pool
	poolMu      sync.RWMutex              // Mutex to sync the message and expiration pools

	peers  map[*Peer]struct{} // Set of currently active peers
	peerMu sync.RWMutex       // Mutex to sync the active peer set

	mailServer MailServer

	messageQueue chan *Envelope
	p2pMsgQueue  chan *Envelope
	quit         chan struct{}

	stats Statistics

	minPoW       float64
	maxMsgLength int
	overflow     bool
}

// New creates a Whisper client ready to communicate through the Ethereum P2P network.
// Param s should be passed if you want to implement mail server, otherwise nil.
func New() *Whisper {
	whisper := &Whisper{
		privateKeys: make(map[string]*ecdsa.PrivateKey),
		//identities:   make(map[string]*ecdsa.PrivateKey),
		symKeys:      make(map[string][]byte),
		envelopes:    make(map[common.Hash]*Envelope),
		expirations:  make(map[uint32]*set.SetNonTS),
		peers:        make(map[*Peer]struct{}),
		messageQueue: make(chan *Envelope, messageQueueLimit),
		p2pMsgQueue:  make(chan *Envelope, messageQueueLimit),
		quit:         make(chan struct{}),
		minPoW:       DefaultMinimumPoW,
		maxMsgLength: DefaultMaxMessageLength,
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

// APIs returns the RPC descriptors the Whisper implementation offers
func (w *Whisper) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: ProtocolName,
			Version:   ProtocolVersionStr,
			Service:   NewPublicWhisperAPI(w),
			Public:    true,
		},
	}
}

func (w *Whisper) RegisterServer(server MailServer) {
	w.mailServer = server
}

// Protocols returns the whisper sub-protocols ran by this particular client.
func (w *Whisper) Protocols() []p2p.Protocol {
	return []p2p.Protocol{w.protocol}
}

// Version returns the whisper sub-protocols version number.
func (w *Whisper) Version() uint {
	return w.protocol.Version
}

func (w *Whisper) SetMaxMessageLength(val int) error {
	if val <= 0 {
		return fmt.Errorf("Invalid message length: %d", val)
	}
	w.maxMsgLength = val
	return nil
}

func (w *Whisper) SetMinimumPoW(val float64) error {
	if val <= 0.0 {
		return fmt.Errorf("Invalid PoW: %f", val)
	}
	w.minPoW = val
	return nil
}

func (w *Whisper) getPeer(peerID []byte) (*Peer, error) {
	w.peerMu.Lock()
	defer w.peerMu.Unlock()
	for p := range w.peers {
		id := p.peer.ID()
		if bytes.Equal(peerID, id[:]) {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Could not find peer with ID: %x", peerID)
}

// MarkPeerTrusted marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (w *Whisper) MarkPeerTrusted(peerID []byte) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	return nil
}

func (w *Whisper) RequestHistoricMessages(peerID []byte, envelope *Envelope) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	p.trusted = true
	return p2p.Send(p.ws, p2pRequestCode, envelope)
}

func (w *Whisper) SendP2PMessage(peerID []byte, envelope *Envelope) error {
	p, err := w.getPeer(peerID)
	if err != nil {
		return err
	}
	return p2p.Send(p.ws, p2pCode, envelope)
}

func (w *Whisper) SendP2PDirect(peer *Peer, envelope *Envelope) error {
	return p2p.Send(peer.ws, p2pCode, envelope)
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption. Returns ID of the new key pair.
func (w *Whisper) NewKeyPair() (string, error) {
	key, err := crypto.GenerateKey()
	if err != nil || !validatePrivateKey(key) {
		key, err = crypto.GenerateKey() // retry once
	}
	if err != nil {
		return "", err
	}
	if !validatePrivateKey(key) {
		return "", fmt.Errorf("Failed to generate valid key")
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("Failed to generate ID: %s", err)
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.privateKeys[id] != nil {
		return "", fmt.Errorf("Failed to generate unique ID")
	}
	w.privateKeys[id] = key
	return id, nil
}

// DeleteIdentity deletes the specified key if it exists.
func (w *Whisper) DeleteKeyPair(key string) bool {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.privateKeys[key] != nil {
		delete(w.privateKeys, key)
		return true
	}
	return false
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (w *Whisper) HasKeyPair(id string) bool {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.privateKeys[id] != nil
}

// GetIdentity retrieves the private key of the specified public identity.
func (w *Whisper) GetPrivateKey(pubKey string) (*ecdsa.PrivateKey, error) {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	key := w.privateKeys[pubKey]
	if key == nil {
		return nil, fmt.Errorf("invalid id")
	}
	return key, nil
}

func (w *Whisper) GenerateSymKey() (string, error) {
	const size = aesKeyLength * 2
	buf := make([]byte, size)
	_, err := crand.Read(buf)
	if err != nil {
		return "", err
	} else if !validateSymmetricKey(buf) {
		return "", fmt.Errorf("error in GenerateSymKey: crypto/rand failed to generate random data")
	}

	key := buf[:aesKeyLength]
	salt := buf[aesKeyLength:]
	derived, err := DeriveOneTimeKey(key, salt, EnvelopeVersion)
	if err != nil {
		return "", err
	} else if !validateSymmetricKey(derived) {
		return "", fmt.Errorf("failed to derive valid key")
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("Failed to generate ID: %s", err)
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.symKeys[id] != nil {
		return "", fmt.Errorf("Failed to generate unique ID")
	}
	w.symKeys[id] = derived
	return id, nil
}

func (w *Whisper) AddSymKeyDirect(key []byte) (string, error) {
	if len(key) != aesKeyLength {
		return "", fmt.Errorf("Wrong key size: %d", len(key))
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("Failed to generate ID: %s", err)
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	if w.symKeys[id] != nil {
		return "", fmt.Errorf("Failed to generate unique ID")
	}
	w.symKeys[id] = key
	return id, nil
}

func (w *Whisper) AddSymKeyFromPassword(password string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", fmt.Errorf("Failed to generate ID: %s", err)
	}
	if w.HasSymKey(id) {
		return "", fmt.Errorf("Failed to generate unique ID")
	}

	derived, err := deriveKeyMaterial([]byte(password), EnvelopeVersion)
	if err != nil {
		return "", err
	}

	w.keyMu.Lock()
	defer w.keyMu.Unlock()

	// double check is necessary, because deriveKeyMaterial() is very slow
	if w.symKeys[id] != nil {
		return "", fmt.Errorf("Severe error: failed to generate unique ID")
	}
	w.symKeys[id] = derived
	return id, nil
}

func (w *Whisper) HasSymKey(id string) bool {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	return w.symKeys[id] != nil
}

func (w *Whisper) DeleteSymKey(id string) bool {
	w.keyMu.Lock()
	defer w.keyMu.Unlock()
	if w.symKeys[id] != nil {
		delete(w.symKeys, id)
		return true
	}
	return false
}

func (w *Whisper) GetSymKey(id string) ([]byte, error) {
	w.keyMu.RLock()
	defer w.keyMu.RUnlock()
	if w.symKeys[id] != nil {
		return w.symKeys[id], nil
	}
	return nil, fmt.Errorf("non-existent ID")
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
func (w *Whisper) Watch(f *Filter) (string, error) {
	return w.filters.Install(f)
}

func (w *Whisper) GetFilter(id string) *Filter {
	return w.filters.Get(id)
}

// Unsubscribe removes an installed message handler.
func (w *Whisper) Unsubscribe(id string) error {
	ok := w.filters.Uninstall(id)
	if !ok {
		return fmt.Errorf("Invalid ID")
	}
	return nil
}

// Send injects a message into the whisper send queue, to be distributed in the
// network in the coming cycles.
func (w *Whisper) Send(envelope *Envelope) error {
	ok, err := w.add(envelope)
	if !ok {
		return fmt.Errorf("failed to add envelope")
	}
	return err
}

// Start implements node.Service, starting the background data propagation thread
// of the Whisper protocol.
func (w *Whisper) Start(*p2p.Server) error {
	log.Info(fmt.Sprint("Whisper started"))
	go w.update()

	numCPU := runtime.NumCPU()
	for i := 0; i < numCPU; i++ {
		go w.processQueue()
	}

	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the Whisper protocol.
func (w *Whisper) Stop() error {
	close(w.quit)
	log.Info(fmt.Sprint("Whisper stopped"))
	return nil
}

// handlePeer is called by the underlying P2P layer when the whisper sub-protocol
// connection is negotiated.
func (wh *Whisper) HandlePeer(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	// Create the new peer and start tracking it
	whisperPeer := newPeer(wh, peer, rw)

	wh.peerMu.Lock()
	wh.peers[whisperPeer] = struct{}{}
	wh.peerMu.Unlock()

	defer func() {
		wh.peerMu.Lock()
		delete(wh.peers, whisperPeer)
		wh.peerMu.Unlock()
	}()

	// Run the peer handshake and state updates
	if err := whisperPeer.handshake(); err != nil {
		return err
	}
	whisperPeer.start()
	defer whisperPeer.stop()

	return wh.runMessageLoop(whisperPeer, rw)
}

// runMessageLoop reads and processes inbound messages directly to merge into client-global state.
func (wh *Whisper) runMessageLoop(p *Peer, rw p2p.MsgReadWriter) error {
	for {
		// fetch the next packet
		packet, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		if packet.Size > uint32(wh.maxMsgLength) {
			return fmt.Errorf("oversized message received")
		}

		switch packet.Code {
		case statusCode:
			// this should not happen, but no need to panic; just ignore this message.
			log.Warn(fmt.Sprintf("%v: unxepected status message received", p.peer))
		case messagesCode:
			// decode the contained envelopes
			var envelopes []*Envelope
			if err := packet.Decode(&envelopes); err != nil {
				log.Warn(fmt.Sprintf("%v: failed to decode envelope: [%v], peer will be disconnected", p.peer, err))
				return fmt.Errorf("garbage received")
			}
			// inject all envelopes into the internal pool
			for _, envelope := range envelopes {
				cached, err := wh.add(envelope)
				if err != nil {
					log.Warn(fmt.Sprintf("%v: bad envelope received: [%v], peer will be disconnected", p.peer, err))
					return fmt.Errorf("invalid envelope")
				}
				if cached {
					p.mark(envelope)
				}
			}
		case p2pCode:
			// peer-to-peer message, sent directly to peer bypassing PoW checks, etc.
			// this message is not supposed to be forwarded to other peers, and
			// therefore might not satisfy the PoW, expiry and other requirements.
			// these messages are only accepted from the trusted peer.
			if p.trusted {
				var envelope Envelope
				if err := packet.Decode(&envelope); err != nil {
					log.Warn(fmt.Sprintf("%v: failed to decode direct message: [%v], peer will be disconnected", p.peer, err))
					return fmt.Errorf("garbage received (directMessage)")
				}
				wh.postEvent(&envelope, true)
			}
		case p2pRequestCode:
			// Must be processed if mail server is implemented. Otherwise ignore.
			if wh.mailServer != nil {
				var request Envelope
				if err := packet.Decode(&request); err != nil {
					log.Warn(fmt.Sprintf("%v: failed to decode p2p request message: [%v], peer will be disconnected", p.peer, err))
					return fmt.Errorf("garbage received (p2p request)")
				}
				wh.mailServer.DeliverMail(p, &request)
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
func (wh *Whisper) add(envelope *Envelope) (bool, error) {
	now := uint32(time.Now().Unix())
	sent := envelope.Expiry - envelope.TTL

	if sent > now {
		if sent-SynchAllowance > now {
			return false, fmt.Errorf("envelope created in the future [%x]", envelope.Hash())
		} else {
			// recalculate PoW, adjusted for the time difference, plus one second for latency
			envelope.calculatePoW(sent - now + 1)
		}
	}

	if envelope.Expiry < now {
		if envelope.Expiry+SynchAllowance*2 < now {
			return false, fmt.Errorf("very old message")
		} else {
			log.Debug(fmt.Sprintf("expired envelope dropped [%x]", envelope.Hash()))
			return false, nil // drop envelope without error
		}
	}

	if envelope.size() > wh.maxMsgLength {
		return false, fmt.Errorf("huge messages are not allowed [%x]", envelope.Hash())
	}

	if len(envelope.Version) > 4 {
		return false, fmt.Errorf("oversized version [%x]", envelope.Hash())
	}

	if len(envelope.AESNonce) > AESNonceMaxLength {
		// the standard AES GSM nonce size is 12,
		// but const gcmStandardNonceSize cannot be accessed directly
		return false, fmt.Errorf("oversized AESNonce [%x]", envelope.Hash())
	}

	if len(envelope.Salt) > saltLength {
		return false, fmt.Errorf("oversized salt [%x]", envelope.Hash())
	}

	if envelope.PoW() < wh.minPoW {
		log.Debug(fmt.Sprintf("envelope with low PoW dropped: %f [%x]", envelope.PoW(), envelope.Hash()))
		return false, nil // drop envelope without error
	}

	hash := envelope.Hash()

	wh.poolMu.Lock()
	_, alreadyCached := wh.envelopes[hash]
	if !alreadyCached {
		wh.envelopes[hash] = envelope
		if wh.expirations[envelope.Expiry] == nil {
			wh.expirations[envelope.Expiry] = set.NewNonTS()
		}
		if !wh.expirations[envelope.Expiry].Has(hash) {
			wh.expirations[envelope.Expiry].Add(hash)
		}
	}
	wh.poolMu.Unlock()

	if alreadyCached {
		log.Trace(fmt.Sprintf("whisper envelope already cached [%x]\n", envelope.Hash()))
	} else {
		log.Trace(fmt.Sprintf("cached whisper envelope [%x]: %v\n", envelope.Hash(), envelope))
		wh.stats.memoryUsed += envelope.size()
		wh.postEvent(envelope, false) // notify the local node about the new message
		if wh.mailServer != nil {
			wh.mailServer.Archive(envelope)
		}
	}
	return true, nil
}

// postEvent queues the message for further processing.
func (w *Whisper) postEvent(envelope *Envelope, isP2P bool) {
	// if the version of incoming message is higher than
	// currently supported version, we can not decrypt it,
	// and therefore just ignore this message
	if envelope.Ver() <= EnvelopeVersion {
		if isP2P {
			w.p2pMsgQueue <- envelope
		} else {
			w.checkOverflow()
			w.messageQueue <- envelope
		}
	}
}

// checkOverflow checks if message queue overflow occurs and reports it if necessary.
func (w *Whisper) checkOverflow() {
	queueSize := len(w.messageQueue)

	if queueSize == messageQueueLimit {
		if !w.overflow {
			w.overflow = true
			log.Warn(fmt.Sprint("message queue overflow"))
		}
	} else if queueSize <= messageQueueLimit/2 {
		if w.overflow {
			w.overflow = false
			log.Warn(fmt.Sprint("message queue overflow fixed (back to normal)"))
		}
	}
}

// processQueue delivers the messages to the watchers during the lifetime of the whisper node.
func (w *Whisper) processQueue() {
	var e *Envelope
	for {
		select {
		case <-w.quit:
			return

		case e = <-w.messageQueue:
			w.filters.NotifyWatchers(e, false)

		case e = <-w.p2pMsgQueue:
			w.filters.NotifyWatchers(e, true)
		}
	}
}

// update loops until the lifetime of the whisper node, updating its internal
// state by expiring stale messages from the pool.
func (w *Whisper) update() {
	// Start a ticker to check for expirations
	expire := time.NewTicker(expirationCycle)

	// Repeat updates until termination is requested
	for {
		select {
		case <-expire.C:
			w.expire()

		case <-w.quit:
			return
		}
	}
}

// expire iterates over all the expiration timestamps, removing all stale
// messages from the pools.
func (w *Whisper) expire() {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	w.stats.reset()
	now := uint32(time.Now().Unix())
	for expiry, hashSet := range w.expirations {
		if expiry < now {
			w.stats.messagesCleared++

			// Dump all expired messages and remove timestamp
			hashSet.Each(func(v interface{}) bool {
				sz := w.envelopes[v.(common.Hash)].size()
				w.stats.memoryCleared += sz
				w.stats.memoryUsed -= sz
				delete(w.envelopes, v.(common.Hash))
				return true
			})
			w.expirations[expiry].Clear()
			delete(w.expirations, expiry)
		}
	}
}

func (w *Whisper) Stats() string {
	result := fmt.Sprintf("Memory usage: %d bytes.\nAverage messages cleared per expiry cycle: %d.",
		w.stats.memoryUsed, w.stats.totalMessagesCleared/w.stats.cycles)
	if w.stats.messagesCleared > 0 {
		result += fmt.Sprintf("\nLatest expiry cycle cleared %d messages (%d bytes).",
			w.stats.messagesCleared, w.stats.memoryCleared)
	}
	return result
}

// envelopes retrieves all the messages currently pooled by the node.
func (w *Whisper) Envelopes() []*Envelope {
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	all := make([]*Envelope, 0, len(w.envelopes))
	for _, envelope := range w.envelopes {
		all = append(all, envelope)
	}
	return all
}

// Messages iterates through all currently floating envelopes
// and retrieves all the messages, that this filter could decrypt.
func (w *Whisper) Messages(id string) []*ReceivedMessage {
	result := make([]*ReceivedMessage, 0)
	w.poolMu.RLock()
	defer w.poolMu.RUnlock()

	if filter := w.filters.Get(id); filter != nil {
		for _, env := range w.envelopes {
			msg := filter.processEnvelope(env)
			if msg != nil {
				result = append(result, msg)
			}
		}
	}
	return result
}

func (w *Whisper) isEnvelopeCached(hash common.Hash) bool {
	w.poolMu.Lock()
	defer w.poolMu.Unlock()

	_, exist := w.envelopes[hash]
	return exist
}

func (s *Statistics) reset() {
	s.cycles++
	s.totalMessagesCleared += s.messagesCleared

	s.memoryCleared = 0
	s.messagesCleared = 0
}

func ValidateKeyID(id string) error {
	if len(id) != keyIdSize*2 {
		return fmt.Errorf("Wrong size of key ID")
	}
	return nil
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
			return false
		}
	}
	return true
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
		// kdf should run no less than 0.1 seconds on average compute,
		// because it's a once in a session experience
		derivedKey := pbkdf2.Key(key, nil, 65356, aesKeyLength, sha256.New)
		return derivedKey, nil
	} else {
		return nil, unknownVersionError(version)
	}
}

func GenerateRandomID() (id string, err error) {
	buf := make([]byte, keyIdSize)
	_, err = crand.Read(buf)
	if err != nil {
		return "", err
	}
	if !validateSymmetricKey(buf) {
		return "", fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
	}
	id = common.Bytes2Hex(buf)
	return id, err
}
