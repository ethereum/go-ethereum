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

package pss

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	defaultPaddingByteSize     = 16
	defaultMsgTTL              = time.Second * 120
	defaultDigestCacheTTL      = time.Second * 10
	defaultSymKeyCacheCapacity = 512
	digestLength               = 32 // byte length of digest used for pss cache (currently same as swarm chunk hash)
	defaultWhisperWorkTime     = 3
	defaultWhisperPoW          = 0.0000000001
	defaultMaxMsgSize          = 1024 * 1024
	defaultCleanInterval       = time.Second * 60 * 10
	defaultOutboxCapacity      = 100000
	pssProtocolName            = "pss"
	pssVersion                 = 2
	hasherCount                = 8
)

var (
	addressLength = len(pot.Address{})
)

// cache is used for preventing backwards routing
// will also be instrumental in flood guard mechanism
// and mailbox implementation
type pssCacheEntry struct {
	expiresAt time.Time
}

// abstraction to enable access to p2p.protocols.Peer.Send
type senderPeer interface {
	Info() *p2p.PeerInfo
	ID() discover.NodeID
	Address() []byte
	Send(interface{}) error
}

// per-key peer related information
// member `protected` prevents garbage collection of the instance
type pssPeer struct {
	lastSeen  time.Time
	address   *PssAddress
	protected bool
}

// Pss configuration parameters
type PssParams struct {
	MsgTTL              time.Duration
	CacheTTL            time.Duration
	privateKey          *ecdsa.PrivateKey
	SymKeyCacheCapacity int
	AllowRaw            bool // If true, enables sending and receiving messages without builtin pss encryption
}

// Sane defaults for Pss
func NewPssParams() *PssParams {
	return &PssParams{
		MsgTTL:              defaultMsgTTL,
		CacheTTL:            defaultDigestCacheTTL,
		SymKeyCacheCapacity: defaultSymKeyCacheCapacity,
	}
}

func (params *PssParams) WithPrivateKey(privatekey *ecdsa.PrivateKey) *PssParams {
	params.privateKey = privatekey
	return params
}

// Toplevel pss object, takes care of message sending, receiving, decryption and encryption, message handler dispatchers and message forwarding.
//
// Implements node.Service
type Pss struct {
	network.Overlay                   // we can get the overlayaddress from this
	privateKey      *ecdsa.PrivateKey // pss can have it's own independent key
	w               *whisper.Whisper  // key and encryption backend
	auxAPIs         []rpc.API         // builtins (handshake, test) can add APIs

	// sending and forwarding
	fwdPool         map[string]*protocols.Peer // keep track of all peers sitting on the pssmsg routing layer
	fwdPoolMu       sync.RWMutex
	fwdCache        map[pssDigest]pssCacheEntry // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	fwdCacheMu      sync.RWMutex
	cacheTTL        time.Duration // how long to keep messages in fwdCache (not implemented)
	msgTTL          time.Duration
	paddingByteSize int
	capstring       string
	outbox          chan *PssMsg

	// keys and peers
	pubKeyPool                 map[string]map[Topic]*pssPeer // mapping of hex public keys to peer address by topic.
	pubKeyPoolMu               sync.RWMutex
	symKeyPool                 map[string]map[Topic]*pssPeer // mapping of symkeyids to peer address by topic.
	symKeyPoolMu               sync.RWMutex
	symKeyDecryptCache         []*string // fast lookup of symkeys recently used for decryption; last used is on top of stack
	symKeyDecryptCacheCursor   int       // modular cursor pointing to last used, wraps on symKeyDecryptCache array
	symKeyDecryptCacheCapacity int       // max amount of symkeys to keep.

	// message handling
	handlers   map[Topic]map[*Handler]bool // topic and version based pss payload handlers. See pss.Handle()
	handlersMu sync.RWMutex
	allowRaw   bool
	hashPool   sync.Pool

	// process
	quitC chan struct{}
}

func (p *Pss) String() string {
	return fmt.Sprintf("pss: addr %x, pubkey %v", p.BaseAddr(), common.ToHex(crypto.FromECDSAPub(&p.privateKey.PublicKey)))
}

// Creates a new Pss instance.
//
// In addition to params, it takes a swarm network overlay
// and a FileStore storage for message cache storage.
func NewPss(k network.Overlay, params *PssParams) (*Pss, error) {
	if params.privateKey == nil {
		return nil, errors.New("missing private key for pss")
	}
	cap := p2p.Cap{
		Name:    pssProtocolName,
		Version: pssVersion,
	}
	ps := &Pss{
		Overlay:    k,
		privateKey: params.privateKey,
		w:          whisper.New(&whisper.DefaultConfig),
		quitC:      make(chan struct{}),

		fwdPool:         make(map[string]*protocols.Peer),
		fwdCache:        make(map[pssDigest]pssCacheEntry),
		cacheTTL:        params.CacheTTL,
		msgTTL:          params.MsgTTL,
		paddingByteSize: defaultPaddingByteSize,
		capstring:       cap.String(),
		outbox:          make(chan *PssMsg, defaultOutboxCapacity),

		pubKeyPool:                 make(map[string]map[Topic]*pssPeer),
		symKeyPool:                 make(map[string]map[Topic]*pssPeer),
		symKeyDecryptCache:         make([]*string, params.SymKeyCacheCapacity),
		symKeyDecryptCacheCapacity: params.SymKeyCacheCapacity,

		handlers: make(map[Topic]map[*Handler]bool),
		allowRaw: params.AllowRaw,
		hashPool: sync.Pool{
			New: func() interface{} {
				return storage.MakeHashFunc(storage.DefaultHash)()
			},
		},
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(storage.DefaultHash)()
		ps.hashPool.Put(hashfunc)
	}

	return ps, nil
}

/////////////////////////////////////////////////////////////////////
// SECTION: node.Service interface
/////////////////////////////////////////////////////////////////////

func (p *Pss) Start(srv *p2p.Server) error {
	go func() {
		ticker := time.NewTicker(defaultCleanInterval)
		cacheTicker := time.NewTicker(p.cacheTTL)
		defer ticker.Stop()
		defer cacheTicker.Stop()
		for {
			select {
			case <-cacheTicker.C:
				p.cleanFwdCache()
			case <-ticker.C:
				p.cleanKeys()
			case <-p.quitC:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case msg := <-p.outbox:
				err := p.forward(msg)
				if err != nil {
					log.Error(err.Error())
					metrics.GetOrRegisterCounter("pss.forward.err", nil).Inc(1)
				}
			case <-p.quitC:
				return
			}
		}
	}()
	log.Debug("Started pss", "public key", common.ToHex(crypto.FromECDSAPub(p.PublicKey())))
	return nil
}

func (p *Pss) Stop() error {
	log.Info("pss shutting down")
	close(p.quitC)
	return nil
}

var pssSpec = &protocols.Spec{
	Name:       pssProtocolName,
	Version:    pssVersion,
	MaxMsgSize: defaultMaxMsgSize,
	Messages: []interface{}{
		PssMsg{},
	},
}

func (p *Pss) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    pssSpec.Name,
			Version: pssSpec.Version,
			Length:  pssSpec.Length(),
			Run:     p.Run,
		},
	}
}

func (p *Pss) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	pp := protocols.NewPeer(peer, rw, pssSpec)
	p.fwdPoolMu.Lock()
	p.fwdPool[peer.Info().ID] = pp
	p.fwdPoolMu.Unlock()
	return pp.Run(p.handlePssMsg)
}

func (p *Pss) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "pss",
			Version:   "1.0",
			Service:   NewAPI(p),
			Public:    true,
		},
	}
	apis = append(apis, p.auxAPIs...)
	return apis
}

// add API methods to the pss API
// must be run before node is started
func (p *Pss) addAPI(api rpc.API) {
	p.auxAPIs = append(p.auxAPIs, api)
}

// Returns the swarm overlay address of the pss node
func (p *Pss) BaseAddr() []byte {
	return p.Overlay.BaseAddr()
}

// Returns the pss node's public key
func (p *Pss) PublicKey() *ecdsa.PublicKey {
	return &p.privateKey.PublicKey
}

/////////////////////////////////////////////////////////////////////
// SECTION: Message handling
/////////////////////////////////////////////////////////////////////

// Links a handler function to a Topic
//
// All incoming messages with an envelope Topic matching the
// topic specified will be passed to the given Handler function.
//
// There may be an arbitrary number of handler functions per topic.
//
// Returns a deregister function which needs to be called to
// deregister the handler,
func (p *Pss) Register(topic *Topic, handler Handler) func() {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	handlers := p.handlers[*topic]
	if handlers == nil {
		handlers = make(map[*Handler]bool)
		p.handlers[*topic] = handlers
	}
	handlers[&handler] = true
	return func() { p.deregister(topic, &handler) }
}
func (p *Pss) deregister(topic *Topic, h *Handler) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	handlers := p.handlers[*topic]
	if len(handlers) == 1 {
		delete(p.handlers, *topic)
		return
	}
	delete(handlers, h)
}

// get all registered handlers for respective topics
func (p *Pss) getHandlers(topic Topic) map[*Handler]bool {
	p.handlersMu.RLock()
	defer p.handlersMu.RUnlock()
	return p.handlers[topic]
}

// Filters incoming messages for processing or forwarding.
// Check if address partially matches
// If yes, it CAN be for us, and we process it
// Only passes error to pss protocol handler if payload is not valid pssmsg
func (p *Pss) handlePssMsg(msg interface{}) error {
	metrics.GetOrRegisterCounter("pss.handlepssmsg", nil).Inc(1)

	pssmsg, ok := msg.(*PssMsg)

	if !ok {
		return fmt.Errorf("invalid message type. Expected *PssMsg, got %T ", msg)
	}
	if int64(pssmsg.Expire) < time.Now().Unix() {
		metrics.GetOrRegisterCounter("pss.expire", nil).Inc(1)
		log.Warn("pss filtered expired message", "from", fmt.Sprintf("%x", p.Overlay.BaseAddr()), "to", fmt.Sprintf("%x", common.ToHex(pssmsg.To)))
		return nil
	}
	if p.checkFwdCache(pssmsg) {
		log.Trace(fmt.Sprintf("pss relay block-cache match (process): FROM %x TO %x", p.Overlay.BaseAddr(), common.ToHex(pssmsg.To)))
		return nil
	}
	p.addFwdCache(pssmsg)

	if !p.isSelfPossibleRecipient(pssmsg) {
		log.Trace("pss was for someone else :'( ... forwarding", "pss", common.ToHex(p.BaseAddr()))
		return p.enqueue(pssmsg)
	}

	log.Trace("pss for us, yay! ... let's process!", "pss", common.ToHex(p.BaseAddr()))
	if err := p.process(pssmsg); err != nil {
		qerr := p.enqueue(pssmsg)
		if qerr != nil {
			return fmt.Errorf("process fail: processerr %v, queueerr: %v", err, qerr)
		}
	}
	return nil

}

// Entry point to processing a message for which the current node can be the intended recipient.
// Attempts symmetric and asymmetric decryption with stored keys.
// Dispatches message to all handlers matching the message topic
func (p *Pss) process(pssmsg *PssMsg) error {
	metrics.GetOrRegisterCounter("pss.process", nil).Inc(1)

	var err error
	var recvmsg *whisper.ReceivedMessage
	var payload []byte
	var from *PssAddress
	var asymmetric bool
	var keyid string
	var keyFunc func(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error)

	envelope := pssmsg.Payload
	psstopic := Topic(envelope.Topic)
	if pssmsg.isRaw() {
		if !p.allowRaw {
			return errors.New("raw message support disabled")
		}
		payload = pssmsg.Payload.Data
	} else {
		if pssmsg.isSym() {
			keyFunc = p.processSym
		} else {
			asymmetric = true
			keyFunc = p.processAsym
		}

		recvmsg, keyid, from, err = keyFunc(envelope)
		if err != nil {
			return errors.New("Decryption failed")
		}
		payload = recvmsg.Payload
	}

	if len(pssmsg.To) < addressLength {
		if err := p.enqueue(pssmsg); err != nil {
			return err
		}
	}
	p.executeHandlers(psstopic, payload, from, asymmetric, keyid)

	return nil

}

func (p *Pss) executeHandlers(topic Topic, payload []byte, from *PssAddress, asymmetric bool, keyid string) {
	handlers := p.getHandlers(topic)
	nid, _ := discover.HexID("0x00") // this hack is needed to satisfy the p2p method
	peer := p2p.NewPeer(nid, fmt.Sprintf("%x", from), []p2p.Cap{})
	for f := range handlers {
		err := (*f)(payload, peer, asymmetric, keyid)
		if err != nil {
			log.Warn("Pss handler %p failed: %v", f, err)
		}
	}
}

// will return false if using partial address
func (p *Pss) isSelfRecipient(msg *PssMsg) bool {
	return bytes.Equal(msg.To, p.Overlay.BaseAddr())
}

// test match of leftmost bytes in given message to node's overlay address
func (p *Pss) isSelfPossibleRecipient(msg *PssMsg) bool {
	local := p.Overlay.BaseAddr()
	return bytes.Equal(msg.To[:], local[:len(msg.To)])
}

/////////////////////////////////////////////////////////////////////
// SECTION: Encryption
/////////////////////////////////////////////////////////////////////

// Links a peer ECDSA public key to a topic
//
// This is required for asymmetric message exchange
// on the given topic
//
// The value in `address` will be used as a routing hint for the
// public key / topic association
func (p *Pss) SetPeerPublicKey(pubkey *ecdsa.PublicKey, topic Topic, address *PssAddress) error {
	pubkeybytes := crypto.FromECDSAPub(pubkey)
	if len(pubkeybytes) == 0 {
		return fmt.Errorf("invalid public key: %v", pubkey)
	}
	pubkeyid := common.ToHex(pubkeybytes)
	psp := &pssPeer{
		address: address,
	}
	p.pubKeyPoolMu.Lock()
	if _, ok := p.pubKeyPool[pubkeyid]; !ok {
		p.pubKeyPool[pubkeyid] = make(map[Topic]*pssPeer)
	}
	p.pubKeyPool[pubkeyid][topic] = psp
	p.pubKeyPoolMu.Unlock()
	log.Trace("added pubkey", "pubkeyid", pubkeyid, "topic", topic, "address", common.ToHex(*address))
	return nil
}

// Automatically generate a new symkey for a topic and address hint
func (p *Pss) generateSymmetricKey(topic Topic, address *PssAddress, addToCache bool) (string, error) {
	keyid, err := p.w.GenerateSymKey()
	if err != nil {
		return "", err
	}
	p.addSymmetricKeyToPool(keyid, topic, address, addToCache, false)
	return keyid, nil
}

// Links a peer symmetric key (arbitrary byte sequence) to a topic
//
// This is required for symmetrically encrypted message exchange
// on the given topic
//
// The key is stored in the whisper backend.
//
// If addtocache is set to true, the key will be added to the cache of keys
// used to attempt symmetric decryption of incoming messages.
//
// Returns a string id that can be used to retrieve the key bytes
// from the whisper backend (see pss.GetSymmetricKey())
func (p *Pss) SetSymmetricKey(key []byte, topic Topic, address *PssAddress, addtocache bool) (string, error) {
	return p.setSymmetricKey(key, topic, address, addtocache, true)
}

func (p *Pss) setSymmetricKey(key []byte, topic Topic, address *PssAddress, addtocache bool, protected bool) (string, error) {
	keyid, err := p.w.AddSymKeyDirect(key)
	if err != nil {
		return "", err
	}
	p.addSymmetricKeyToPool(keyid, topic, address, addtocache, protected)
	return keyid, nil
}

// adds a symmetric key to the pss key pool, and optionally adds the key
// to the collection of keys used to attempt symmetric decryption of
// incoming messages
func (p *Pss) addSymmetricKeyToPool(keyid string, topic Topic, address *PssAddress, addtocache bool, protected bool) {
	psp := &pssPeer{
		address:   address,
		protected: protected,
	}
	p.symKeyPoolMu.Lock()
	if _, ok := p.symKeyPool[keyid]; !ok {
		p.symKeyPool[keyid] = make(map[Topic]*pssPeer)
	}
	p.symKeyPool[keyid][topic] = psp
	p.symKeyPoolMu.Unlock()
	if addtocache {
		p.symKeyDecryptCacheCursor++
		p.symKeyDecryptCache[p.symKeyDecryptCacheCursor%cap(p.symKeyDecryptCache)] = &keyid
	}
	key, _ := p.GetSymmetricKey(keyid)
	log.Trace("added symkey", "symkeyid", keyid, "symkey", common.ToHex(key), "topic", topic, "address", fmt.Sprintf("%p", address), "cache", addtocache)
}

// Returns a symmetric key byte seqyence stored in the whisper backend
// by its unique id
//
// Passes on the error value from the whisper backend
func (p *Pss) GetSymmetricKey(symkeyid string) ([]byte, error) {
	symkey, err := p.w.GetSymKey(symkeyid)
	if err != nil {
		return nil, err
	}
	return symkey, nil
}

// Returns all recorded topic and address combination for a specific public key
func (p *Pss) GetPublickeyPeers(keyid string) (topic []Topic, address []PssAddress, err error) {
	p.pubKeyPoolMu.RLock()
	defer p.pubKeyPoolMu.RUnlock()
	for t, peer := range p.pubKeyPool[keyid] {
		topic = append(topic, t)
		address = append(address, *peer.address)
	}

	return topic, address, nil
}

func (p *Pss) getPeerAddress(keyid string, topic Topic) (PssAddress, error) {
	p.pubKeyPoolMu.RLock()
	defer p.pubKeyPoolMu.RUnlock()
	if peers, ok := p.pubKeyPool[keyid]; ok {
		if t, ok := peers[topic]; ok {
			return *t.address, nil
		}
	}
	return nil, fmt.Errorf("peer with pubkey %s, topic %x not found", keyid, topic)
}

// Attempt to decrypt, validate and unpack a
// symmetrically encrypted message
// If successful, returns the unpacked whisper ReceivedMessage struct
// encapsulating the decrypted message, and the whisper backend id
// of the symmetric key used to decrypt the message.
// It fails if decryption of the message fails or if the message is corrupted
func (p *Pss) processSym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error) {
	metrics.GetOrRegisterCounter("pss.process.sym", nil).Inc(1)

	for i := p.symKeyDecryptCacheCursor; i > p.symKeyDecryptCacheCursor-cap(p.symKeyDecryptCache) && i > 0; i-- {
		symkeyid := p.symKeyDecryptCache[i%cap(p.symKeyDecryptCache)]
		symkey, err := p.w.GetSymKey(*symkeyid)
		if err != nil {
			continue
		}
		recvmsg, err := envelope.OpenSymmetric(symkey)
		if err != nil {
			continue
		}
		if !recvmsg.Validate() {
			return nil, "", nil, fmt.Errorf("symmetrically encrypted message has invalid signature or is corrupt")
		}
		p.symKeyPoolMu.Lock()
		from := p.symKeyPool[*symkeyid][Topic(envelope.Topic)].address
		p.symKeyPoolMu.Unlock()
		p.symKeyDecryptCacheCursor++
		p.symKeyDecryptCache[p.symKeyDecryptCacheCursor%cap(p.symKeyDecryptCache)] = symkeyid
		return recvmsg, *symkeyid, from, nil
	}
	return nil, "", nil, fmt.Errorf("could not decrypt message")
}

// Attempt to decrypt, validate and unpack an
// asymmetrically encrypted message
// If successful, returns the unpacked whisper ReceivedMessage struct
// encapsulating the decrypted message, and the byte representation of
// the public key used to decrypt the message.
// It fails if decryption of message fails, or if the message is corrupted
func (p *Pss) processAsym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error) {
	metrics.GetOrRegisterCounter("pss.process.asym", nil).Inc(1)

	recvmsg, err := envelope.OpenAsymmetric(p.privateKey)
	if err != nil {
		return nil, "", nil, fmt.Errorf("could not decrypt message: %s", err)
	}
	// check signature (if signed), strip padding
	if !recvmsg.Validate() {
		return nil, "", nil, fmt.Errorf("invalid message")
	}
	pubkeyid := common.ToHex(crypto.FromECDSAPub(recvmsg.Src))
	var from *PssAddress
	p.pubKeyPoolMu.Lock()
	if p.pubKeyPool[pubkeyid][Topic(envelope.Topic)] != nil {
		from = p.pubKeyPool[pubkeyid][Topic(envelope.Topic)].address
	}
	p.pubKeyPoolMu.Unlock()
	return recvmsg, pubkeyid, from, nil
}

// Symkey garbage collection
// a key is removed if:
// - it is not marked as protected
// - it is not in the incoming decryption cache
func (p *Pss) cleanKeys() (count int) {
	for keyid, peertopics := range p.symKeyPool {
		var expiredtopics []Topic
		for topic, psp := range peertopics {
			if psp.protected {
				continue
			}

			var match bool
			for i := p.symKeyDecryptCacheCursor; i > p.symKeyDecryptCacheCursor-cap(p.symKeyDecryptCache) && i > 0; i-- {
				cacheid := p.symKeyDecryptCache[i%cap(p.symKeyDecryptCache)]
				if *cacheid == keyid {
					match = true
				}
			}
			if !match {
				expiredtopics = append(expiredtopics, topic)
			}
		}
		for _, topic := range expiredtopics {
			p.symKeyPoolMu.Lock()
			delete(p.symKeyPool[keyid], topic)
			log.Trace("symkey cleanup deletion", "symkeyid", keyid, "topic", topic, "val", p.symKeyPool[keyid])
			p.symKeyPoolMu.Unlock()
			count++
		}
	}
	return
}

/////////////////////////////////////////////////////////////////////
// SECTION: Message sending
/////////////////////////////////////////////////////////////////////

func (p *Pss) enqueue(msg *PssMsg) error {
	select {
	case p.outbox <- msg:
		return nil
	default:
	}

	metrics.GetOrRegisterCounter("pss.enqueue.outbox.full", nil).Inc(1)
	return errors.New("outbox full")
}

// Send a raw message (any encryption is responsibility of calling client)
//
// Will fail if raw messages are disallowed
func (p *Pss) SendRaw(address PssAddress, topic Topic, msg []byte) error {
	if !p.allowRaw {
		return errors.New("Raw messages not enabled")
	}
	pssMsgParams := &msgParams{
		raw: true,
	}
	payload := &whisper.Envelope{
		Data:  msg,
		Topic: whisper.TopicType(topic),
	}
	pssMsg := newPssMsg(pssMsgParams)
	pssMsg.To = address
	pssMsg.Expire = uint32(time.Now().Add(p.msgTTL).Unix())
	pssMsg.Payload = payload
	p.addFwdCache(pssMsg)
	return p.enqueue(pssMsg)
}

// Send a message using symmetric encryption
//
// Fails if the key id does not match any of the stored symmetric keys
func (p *Pss) SendSym(symkeyid string, topic Topic, msg []byte) error {
	symkey, err := p.GetSymmetricKey(symkeyid)
	if err != nil {
		return fmt.Errorf("missing valid send symkey %s: %v", symkeyid, err)
	}
	p.symKeyPoolMu.Lock()
	psp, ok := p.symKeyPool[symkeyid][topic]
	p.symKeyPoolMu.Unlock()
	if !ok {
		return fmt.Errorf("invalid topic '%s' for symkey '%s'", topic.String(), symkeyid)
	} else if psp.address == nil {
		return fmt.Errorf("no address hint for topic '%s' symkey '%s'", topic.String(), symkeyid)
	}
	err = p.send(*psp.address, topic, msg, false, symkey)
	return err
}

// Send a message using asymmetric encryption
//
// Fails if the key id does not match any in of the stored public keys
func (p *Pss) SendAsym(pubkeyid string, topic Topic, msg []byte) error {
	if _, err := crypto.UnmarshalPubkey(common.FromHex(pubkeyid)); err != nil {
		return fmt.Errorf("Cannot unmarshal pubkey: %x", pubkeyid)
	}
	p.pubKeyPoolMu.Lock()
	psp, ok := p.pubKeyPool[pubkeyid][topic]
	p.pubKeyPoolMu.Unlock()
	if !ok {
		return fmt.Errorf("invalid topic '%s' for pubkey '%s'", topic.String(), pubkeyid)
	} else if psp.address == nil {
		return fmt.Errorf("no address hint for topic '%s' pubkey '%s'", topic.String(), pubkeyid)
	}
	go func() {
		p.send(*psp.address, topic, msg, true, common.FromHex(pubkeyid))
	}()
	return nil
}

// Send is payload agnostic, and will accept any byte slice as payload
// It generates an whisper envelope for the specified recipient and topic,
// and wraps the message payload in it.
// TODO: Implement proper message padding
func (p *Pss) send(to []byte, topic Topic, msg []byte, asymmetric bool, key []byte) error {
	metrics.GetOrRegisterCounter("pss.send", nil).Inc(1)

	if key == nil || bytes.Equal(key, []byte{}) {
		return fmt.Errorf("Zero length key passed to pss send")
	}
	padding := make([]byte, p.paddingByteSize)
	c, err := rand.Read(padding)
	if err != nil {
		return err
	} else if c < p.paddingByteSize {
		return fmt.Errorf("invalid padding length: %d", c)
	}
	wparams := &whisper.MessageParams{
		TTL:      defaultWhisperTTL,
		Src:      p.privateKey,
		Topic:    whisper.TopicType(topic),
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  msg,
		Padding:  padding,
	}
	if asymmetric {
		pk, err := crypto.UnmarshalPubkey(key)
		if err != nil {
			return fmt.Errorf("Cannot unmarshal pubkey: %x", key)
		}
		wparams.Dst = pk
	} else {
		wparams.KeySym = key
	}
	// set up outgoing message container, which does encryption and envelope wrapping
	woutmsg, err := whisper.NewSentMessage(wparams)
	if err != nil {
		return fmt.Errorf("failed to generate whisper message encapsulation: %v", err)
	}
	// performs encryption.
	// Does NOT perform / performs negligible PoW due to very low difficulty setting
	// after this the message is ready for sending
	envelope, err := woutmsg.Wrap(wparams)
	if err != nil {
		return fmt.Errorf("failed to perform whisper encryption: %v", err)
	}
	log.Trace("pssmsg whisper done", "env", envelope, "wparams payload", common.ToHex(wparams.Payload), "to", common.ToHex(to), "asym", asymmetric, "key", common.ToHex(key))

	// prepare for devp2p transport
	pssMsgParams := &msgParams{
		sym: !asymmetric,
	}
	pssMsg := newPssMsg(pssMsgParams)
	pssMsg.To = to
	pssMsg.Expire = uint32(time.Now().Add(p.msgTTL).Unix())
	pssMsg.Payload = envelope
	return p.enqueue(pssMsg)
}

// Forwards a pss message to the peer(s) closest to the to recipient address in the PssMsg struct
// The recipient address can be of any length, and the byte slice will be matched to the MSB slice
// of the peer address of the equivalent length.
func (p *Pss) forward(msg *PssMsg) error {
	metrics.GetOrRegisterCounter("pss.forward", nil).Inc(1)

	to := make([]byte, addressLength)
	copy(to[:len(msg.To)], msg.To)

	// send with kademlia
	// find the closest peer to the recipient and attempt to send
	sent := 0
	p.Overlay.EachConn(to, 256, func(op network.OverlayConn, po int, isproxbin bool) bool {
		// we need p2p.protocols.Peer.Send
		// cast and resolve
		sp, ok := op.(senderPeer)
		if !ok {
			log.Crit("Pss cannot use kademlia peer type")
			return false
		}
		info := sp.Info()

		// check if the peer is running pss
		var ispss bool
		for _, cap := range info.Caps {
			if cap == p.capstring {
				ispss = true
				break
			}
		}
		if !ispss {
			log.Trace("peer doesn't have matching pss capabilities, skipping", "peer", info.Name, "caps", info.Caps)
			return true
		}

		// get the protocol peer from the forwarding peer cache
		sendMsg := fmt.Sprintf("MSG TO %x FROM %x VIA %x", to, p.BaseAddr(), op.Address())
		p.fwdPoolMu.RLock()
		pp := p.fwdPool[sp.Info().ID]
		p.fwdPoolMu.RUnlock()

		// attempt to send the message
		err := pp.Send(msg)
		if err != nil {
			metrics.GetOrRegisterCounter("pss.pp.send.error", nil).Inc(1)
			log.Error(err.Error())
			return true
		}
		sent++
		log.Trace(fmt.Sprintf("%v: successfully forwarded", sendMsg))

		// continue forwarding if:
		// - if the peer is end recipient but the full address has not been disclosed
		// - if the peer address matches the partial address fully
		// - if the peer is in proxbin
		if len(msg.To) < addressLength && bytes.Equal(msg.To, op.Address()[:len(msg.To)]) {
			log.Trace(fmt.Sprintf("Pss keep forwarding: Partial address + full partial match"))
			return true
		} else if isproxbin {
			log.Trace(fmt.Sprintf("%x is in proxbin, keep forwarding", common.ToHex(op.Address())))
			return true
		}
		// at this point we stop forwarding, and the state is as follows:
		// - the peer is end recipient and we have full address
		// - we are not in proxbin (directed routing)
		// - partial addresses don't fully match
		return false
	})

	if sent == 0 {
		log.Debug("unable to forward to any peers")
		if err := p.enqueue(msg); err != nil {
			metrics.GetOrRegisterCounter("pss.forward.enqueue.error", nil).Inc(1)
			log.Error(err.Error())
			return err
		}
	}

	// cache the message
	p.addFwdCache(msg)
	return nil
}

/////////////////////////////////////////////////////////////////////
// SECTION: Caching
/////////////////////////////////////////////////////////////////////

// cleanFwdCache is used to periodically remove expired entries from the forward cache
func (p *Pss) cleanFwdCache() {
	metrics.GetOrRegisterCounter("pss.cleanfwdcache", nil).Inc(1)
	p.fwdCacheMu.Lock()
	defer p.fwdCacheMu.Unlock()
	for k, v := range p.fwdCache {
		if v.expiresAt.Before(time.Now()) {
			delete(p.fwdCache, k)
		}
	}
}

// add a message to the cache
func (p *Pss) addFwdCache(msg *PssMsg) error {
	metrics.GetOrRegisterCounter("pss.addfwdcache", nil).Inc(1)

	var entry pssCacheEntry
	var ok bool

	p.fwdCacheMu.Lock()
	defer p.fwdCacheMu.Unlock()

	digest := p.digest(msg)
	if entry, ok = p.fwdCache[digest]; !ok {
		entry = pssCacheEntry{}
	}
	entry.expiresAt = time.Now().Add(p.cacheTTL)
	p.fwdCache[digest] = entry
	return nil
}

// check if message is in the cache
func (p *Pss) checkFwdCache(msg *PssMsg) bool {
	p.fwdCacheMu.Lock()
	defer p.fwdCacheMu.Unlock()

	digest := p.digest(msg)
	entry, ok := p.fwdCache[digest]
	if ok {
		if entry.expiresAt.After(time.Now()) {
			log.Trace("unexpired cache", "digest", fmt.Sprintf("%x", digest))
			metrics.GetOrRegisterCounter("pss.checkfwdcache.unexpired", nil).Inc(1)
			return true
		}
		metrics.GetOrRegisterCounter("pss.checkfwdcache.expired", nil).Inc(1)
	}
	return false
}

// Digest of message
func (p *Pss) digest(msg *PssMsg) pssDigest {
	hasher := p.hashPool.Get().(storage.SwarmHash)
	defer p.hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(msg.serialize())
	digest := pssDigest{}
	key := hasher.Sum(nil)
	copy(digest[:], key[:digestLength])
	return digest
}
