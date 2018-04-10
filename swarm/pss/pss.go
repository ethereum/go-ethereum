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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	defaultPaddingByteSize     = 16
	defaultMsgTTL              = time.Second * 8
	defaultDigestCacheTTL      = time.Second
	defaultSymKeyCacheCapacity = 512
	digestLength               = 32 // byte length of digest used for pss cache (currently same as swarm chunk hash)
	defaultWhisperWorkTime     = 3
	defaultWhisperPoW          = 0.0000000001
	defaultMaxMsgSize          = 1024 * 1024
	defaultCleanInterval       = time.Second * 60 * 10
	defaultDequeueInterval     = time.Millisecond * 10
	defaultOutboxCapacity      = 10000
	pssProtocolName            = "pss"
	pssVersion                 = 1
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
func NewPssParams(privatekey *ecdsa.PrivateKey) *PssParams {
	return &PssParams{
		MsgTTL:              defaultMsgTTL,
		CacheTTL:            defaultDigestCacheTTL,
		privateKey:          privatekey,
		SymKeyCacheCapacity: defaultSymKeyCacheCapacity,
	}
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
	outboxCapacity  int
	outboxCounter   int

	// keys and peers
	pubKeyPool                 map[string]map[Topic]*pssPeer // mapping of hex public keys to peer address by topic.
	pubKeyPoolMu               sync.Mutex
	symKeyPool                 map[string]map[Topic]*pssPeer // mapping of symkeyids to peer address by topic.
	symKeyPoolMu               sync.Mutex
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

func (self *Pss) String() string {
	return fmt.Sprintf("pss: addr %x, pubkey %v", self.BaseAddr(), common.ToHex(crypto.FromECDSAPub(&self.privateKey.PublicKey)))
}

// Creates a new Pss instance.
//
// In addition to params, it takes a swarm network overlay
// and a DPA storage for message cache storage.
func NewPss(k network.Overlay, params *PssParams) *Pss {
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
		outboxCapacity:  defaultOutboxCapacity,

		pubKeyPool:                 make(map[string]map[Topic]*pssPeer),
		symKeyPool:                 make(map[string]map[Topic]*pssPeer),
		symKeyDecryptCache:         make([]*string, params.SymKeyCacheCapacity),
		symKeyDecryptCacheCapacity: params.SymKeyCacheCapacity,

		handlers: make(map[Topic]map[*Handler]bool),
		allowRaw: params.AllowRaw,
		hashPool: sync.Pool{
			New: func() interface{} {
				return storage.MakeHashFunc(storage.SHA3Hash)()
			},
		},
	}

	for i := 0; i < hasherCount; i++ {
		hashfunc := storage.MakeHashFunc(storage.SHA3Hash)()
		ps.hashPool.Put(hashfunc)
	}

	return ps
}

/////////////////////////////////////////////////////////////////////
// SECTION: node.Service interface
/////////////////////////////////////////////////////////////////////

func (self *Pss) Start(srv *p2p.Server) error {
	go func() {
		for {
			tickC := time.Tick(defaultCleanInterval)
			cacheTickC := time.Tick(self.cacheTTL)
			select {
			case <-cacheTickC:
				self.cleanFwdCache()
			case <-tickC:
				self.cleanKeys()
			case <-self.quitC:
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case msg := <-self.outbox:
				self.forward(msg)
			case <-self.quitC:
				return
			}
		}
	}()
	log.Debug("Started pss", "public key", common.ToHex(crypto.FromECDSAPub(self.PublicKey())))
	return nil
}

func (self *Pss) Stop() error {
	log.Info("pss shutting down")
	close(self.quitC)
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

func (self *Pss) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    pssSpec.Name,
			Version: pssSpec.Version,
			Length:  pssSpec.Length(),
			Run:     self.Run,
		},
	}
}

func (self *Pss) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	pp := protocols.NewPeer(p, rw, pssSpec)
	self.fwdPoolMu.Lock()
	self.fwdPool[p.Info().ID] = pp
	self.fwdPoolMu.Unlock()
	return pp.Run(self.handlePssMsg)
}

func (self *Pss) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "pss",
			Version:   "1.0",
			Service:   NewAPI(self),
			Public:    true,
		},
	}
	apis = append(apis, self.auxAPIs...)
	return apis
}

// add API methods to the pss API
// must be run before node is started
func (self *Pss) addAPI(api rpc.API) {
	self.auxAPIs = append(self.auxAPIs, api)
}

// Returns the swarm overlay address of the pss node
func (self *Pss) BaseAddr() []byte {
	return self.Overlay.BaseAddr()
}

// Returns the pss node's public key
func (self *Pss) PublicKey() *ecdsa.PublicKey {
	return &self.privateKey.PublicKey
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
func (self *Pss) Register(topic *Topic, handler Handler) func() {
	self.handlersMu.Lock()
	defer self.handlersMu.Unlock()
	handlers := self.handlers[*topic]
	if handlers == nil {
		handlers = make(map[*Handler]bool)
		self.handlers[*topic] = handlers
	}
	handlers[&handler] = true
	return func() { self.deregister(topic, &handler) }
}
func (self *Pss) deregister(topic *Topic, h *Handler) {
	self.handlersMu.Lock()
	defer self.handlersMu.Unlock()
	handlers := self.handlers[*topic]
	if len(handlers) == 1 {
		delete(self.handlers, *topic)
		return
	}
	delete(handlers, h)
}

// get all registered handlers for respective topics
func (self *Pss) getHandlers(topic Topic) map[*Handler]bool {
	self.handlersMu.RLock()
	defer self.handlersMu.RUnlock()
	return self.handlers[topic]
}

// Filters incoming messages for processing or forwarding.
// Check if address partially matches
// If yes, it CAN be for us, and we process it
// Passes error to pss protocol handler if payload is not valid pssmsg
func (self *Pss) handlePssMsg(msg interface{}) error {
	pssmsg, ok := msg.(*PssMsg)

	if ok {
		if self.checkFwdCache(pssmsg) {
			log.Trace(fmt.Sprintf("pss relay block-cache match (process): FROM %x TO %x", self.Overlay.BaseAddr(), common.ToHex(pssmsg.To)))
			return nil
		}
		self.addFwdCache(pssmsg)

		var err error
		if !self.isSelfPossibleRecipient(pssmsg) {
			log.Trace("pss was for someone else :'( ... forwarding", "pss", common.ToHex(self.BaseAddr()))
			if !self.enqueue(pssmsg) {
				return errors.New("outbox full!")
			}
		}
		log.Trace("pss for us, yay! ... let's process!", "pss", common.ToHex(self.BaseAddr()))

		if !self.process(pssmsg) {
			if !self.enqueue(pssmsg) {
				return errors.New("outbox full!")
			}
		}
		return err
	}

	return fmt.Errorf("invalid message type. Expected *PssMsg, got %T ", msg)
}

// Entry point to processing a message for which the current node can be the intended recipient.
// Attempts symmetric and asymmetric decryption with stored keys.
// Dispatches message to all handlers matching the message topic
func (self *Pss) process(pssmsg *PssMsg) bool {
	var err error
	var recvmsg *whisper.ReceivedMessage
	var from *PssAddress
	var asymmetric bool
	var keyid string
	var keyFunc func(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error)

	envelope := pssmsg.Payload
	psstopic := Topic(envelope.Topic)
	if self.allowRaw && psstopic == rawTopic {
		self.executeHandlers(rawTopic, envelope.Data, nil, false, "")
		return true
	}

	if len(envelope.AESNonce) > 0 { // detect symkey msg according to whisperv5/envelope.go:OpenSymmetric
		keyFunc = self.processSym
	} else {
		asymmetric = true
		keyFunc = self.processAsym
	}
	recvmsg, keyid, from, err = keyFunc(envelope)
	if err != nil {
		return false
	}

	if len(pssmsg.To) < addressLength {
		if !self.enqueue(pssmsg) {
			log.Error("outbox full!")
			return false
		}
	}
	self.executeHandlers(psstopic, recvmsg.Payload, from, asymmetric, keyid)

	return true

}

func (self *Pss) executeHandlers(topic Topic, payload []byte, from *PssAddress, asymmetric bool, keyid string) {
	handlers := self.getHandlers(topic)
	nid, _ := discover.HexID("0x00") // this hack is needed to satisfy the p2p method
	p := p2p.NewPeer(nid, fmt.Sprintf("%x", from), []p2p.Cap{})
	for f := range handlers {
		err := (*f)(payload, p, asymmetric, keyid)
		if err != nil {
			log.Warn("Pss handler %p failed: %v", f, err)
		}
	}
}

// will return false if using partial address
func (self *Pss) isSelfRecipient(msg *PssMsg) bool {
	return bytes.Equal(msg.To, self.Overlay.BaseAddr())
}

// test match of leftmost bytes in given message to node's overlay address
func (self *Pss) isSelfPossibleRecipient(msg *PssMsg) bool {
	local := self.Overlay.BaseAddr()
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
func (self *Pss) SetPeerPublicKey(pubkey *ecdsa.PublicKey, topic Topic, address *PssAddress) error {
	pubkeybytes := crypto.FromECDSAPub(pubkey)
	if len(pubkeybytes) == 0 {
		return fmt.Errorf("invalid public key: %v", pubkey)
	}
	pubkeyid := common.ToHex(pubkeybytes)
	psp := &pssPeer{
		address: address,
	}
	self.pubKeyPoolMu.Lock()
	if _, ok := self.pubKeyPool[pubkeyid]; !ok {
		self.pubKeyPool[pubkeyid] = make(map[Topic]*pssPeer)
	}
	self.pubKeyPool[pubkeyid][topic] = psp
	self.pubKeyPoolMu.Unlock()
	log.Trace("added pubkey", "pubkeyid", pubkeyid, "topic", topic, "address", common.ToHex(*address))
	return nil
}

// Automatically generate a new symkey for a topic and address hint
func (self *Pss) generateSymmetricKey(topic Topic, address *PssAddress, addToCache bool) (string, error) {
	keyid, err := self.w.GenerateSymKey()
	if err != nil {
		return "", err
	}
	self.addSymmetricKeyToPool(keyid, topic, address, addToCache, false)
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
func (self *Pss) SetSymmetricKey(key []byte, topic Topic, address *PssAddress, addtocache bool) (string, error) {
	return self.setSymmetricKey(key, topic, address, addtocache, true)
}

func (self *Pss) setSymmetricKey(key []byte, topic Topic, address *PssAddress, addtocache bool, protected bool) (string, error) {
	keyid, err := self.w.AddSymKeyDirect(key)
	if err != nil {
		return "", err
	}
	self.addSymmetricKeyToPool(keyid, topic, address, addtocache, protected)
	return keyid, nil
}

// adds a symmetric key to the pss key pool, and optionally adds the key
// to the collection of keys used to attempt symmetric decryption of
// incoming messages
func (self *Pss) addSymmetricKeyToPool(keyid string, topic Topic, address *PssAddress, addtocache bool, protected bool) {
	psp := &pssPeer{
		address:   address,
		protected: protected,
	}
	self.symKeyPoolMu.Lock()
	if _, ok := self.symKeyPool[keyid]; !ok {
		self.symKeyPool[keyid] = make(map[Topic]*pssPeer)
	}
	self.symKeyPool[keyid][topic] = psp
	self.symKeyPoolMu.Unlock()
	if addtocache {
		self.symKeyDecryptCacheCursor++
		self.symKeyDecryptCache[self.symKeyDecryptCacheCursor%cap(self.symKeyDecryptCache)] = &keyid
	}
	key, _ := self.GetSymmetricKey(keyid)
	log.Trace("added symkey", "symkeyid", keyid, "symkey", common.ToHex(key), "topic", topic, "address", fmt.Sprintf("%p", address), "cache", addtocache)
}

// Returns a symmetric key byte seqyence stored in the whisper backend
// by its unique id
//
// Passes on the error value from the whisper backend
func (self *Pss) GetSymmetricKey(symkeyid string) ([]byte, error) {
	symkey, err := self.w.GetSymKey(symkeyid)
	if err != nil {
		return nil, err
	}
	return symkey, nil
}

// Attempt to decrypt, validate and unpack a
// symmetrically encrypted message
// If successful, returns the unpacked whisper ReceivedMessage struct
// encapsulating the decrypted message, and the whisper backend id
// of the symmetric key used to decrypt the message.
// It fails if decryption of the message fails or if the message is corrupted
func (self *Pss) processSym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error) {
	for i := self.symKeyDecryptCacheCursor; i > self.symKeyDecryptCacheCursor-cap(self.symKeyDecryptCache) && i > 0; i-- {
		symkeyid := self.symKeyDecryptCache[i%cap(self.symKeyDecryptCache)]
		symkey, err := self.w.GetSymKey(*symkeyid)
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
		self.symKeyPoolMu.Lock()
		from := self.symKeyPool[*symkeyid][Topic(envelope.Topic)].address
		self.symKeyPoolMu.Unlock()
		self.symKeyDecryptCacheCursor++
		self.symKeyDecryptCache[self.symKeyDecryptCacheCursor%cap(self.symKeyDecryptCache)] = symkeyid
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
func (self *Pss) processAsym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, *PssAddress, error) {
	recvmsg, err := envelope.OpenAsymmetric(self.privateKey)
	if err != nil {
		return nil, "", nil, fmt.Errorf("could not decrypt message: %s", err)
	}
	// check signature (if signed), strip padding
	if !recvmsg.Validate() {
		return nil, "", nil, fmt.Errorf("invalid message")
	}
	pubkeyid := common.ToHex(crypto.FromECDSAPub(recvmsg.Src))
	var from *PssAddress
	self.pubKeyPoolMu.Lock()
	if self.pubKeyPool[pubkeyid][Topic(envelope.Topic)] != nil {
		from = self.pubKeyPool[pubkeyid][Topic(envelope.Topic)].address
	}
	self.pubKeyPoolMu.Unlock()
	return recvmsg, pubkeyid, from, nil
}

// Symkey garbage collection
// a key is removed if:
// - it is not marked as protected
// - it is not in the incoming decryption cache
func (self *Pss) cleanKeys() (count int) {
	for keyid, peertopics := range self.symKeyPool {
		var expiredtopics []Topic
		for topic, psp := range peertopics {
			//log.Trace("check topic", "topic", topic, "id", keyid, "protect", psp.protected, "p", fmt.Sprintf("%p", self.symKeyPool[keyid][topic]))
			if psp.protected {
				continue
			}

			var match bool
			for i := self.symKeyDecryptCacheCursor; i > self.symKeyDecryptCacheCursor-cap(self.symKeyDecryptCache) && i > 0; i-- {
				cacheid := self.symKeyDecryptCache[i%cap(self.symKeyDecryptCache)]
				//log.Trace("check cache", "idx", i, "id", *cacheid)
				if *cacheid == keyid {
					match = true
				}
			}
			if !match {
				expiredtopics = append(expiredtopics, topic)
			}
		}
		for _, topic := range expiredtopics {
			self.symKeyPoolMu.Lock()
			delete(self.symKeyPool[keyid], topic)
			log.Trace("symkey cleanup deletion", "symkeyid", keyid, "topic", topic, "val", self.symKeyPool[keyid])
			self.symKeyPoolMu.Unlock()
			count++
		}
	}
	return
}

/////////////////////////////////////////////////////////////////////
// SECTION: Message sending
/////////////////////////////////////////////////////////////////////

func (self *Pss) enqueue(msg *PssMsg) bool {
	select {
	case self.outbox <- msg:
		return true
	default:
	}
	return true
}

// Send a raw message (any encryption is responsibility of calling client)
//
// Will fail if raw messages are disallowed
func (self *Pss) SendRaw(msg []byte, address PssAddress) error {
	if !self.allowRaw {
		return errors.New("Raw messages not enabled")
	}
	pssmsg := &PssMsg{
		To:     address,
		Expire: uint32(time.Now().Add(self.msgTTL).Unix()),
		Payload: &whisper.Envelope{
			Data:  msg,
			Topic: whisper.TopicType(rawTopic),
		},
	}
	self.addFwdCache(pssmsg)
	if !self.enqueue(pssmsg) {
		return errors.New("outbox full!")
	}
	return nil
}

// Send a message using symmetric encryption
//
// Fails if the key id does not match any of the stored symmetric keys
func (self *Pss) SendSym(symkeyid string, topic Topic, msg []byte) error {
	symkey, err := self.GetSymmetricKey(symkeyid)
	if err != nil {
		return fmt.Errorf("missing valid send symkey %s: %v", symkeyid, err)
	}
	self.symKeyPoolMu.Lock()
	psp, ok := self.symKeyPool[symkeyid][topic]
	self.symKeyPoolMu.Unlock()
	if !ok {
		return fmt.Errorf("invalid topic '%s' for symkey '%s'", topic, symkeyid)
	} else if psp.address == nil {
		return fmt.Errorf("no address hint for topic '%s' symkey '%s'", topic, symkeyid)
	}
	err = self.send(*psp.address, topic, msg, false, symkey)
	return err
}

// Send a message using asymmetric encryption
//
// Fails if the key id does not match any in of the stored public keys
func (self *Pss) SendAsym(pubkeyid string, topic Topic, msg []byte) error {
	pubkey := crypto.ToECDSAPub(common.FromHex(pubkeyid))
	if pubkey == nil {
		return fmt.Errorf("Invalid public key id %x", pubkey)
	}
	self.pubKeyPoolMu.Lock()
	psp, ok := self.pubKeyPool[pubkeyid][topic]
	self.pubKeyPoolMu.Unlock()
	if !ok {
		return fmt.Errorf("invalid topic '%s' for pubkey '%s'", topic, pubkeyid)
	} else if psp.address == nil {
		return fmt.Errorf("no address hint for topic '%s' pubkey '%s'", topic, pubkeyid)
	}
	go func() {
		self.send(*psp.address, topic, msg, true, common.FromHex(pubkeyid))
	}()
	return nil
}

// Send is payload agnostic, and will accept any byte slice as payload
// It generates an whisper envelope for the specified recipient and topic,
// and wraps the message payload in it.
// TODO: Implement proper message padding
func (self *Pss) send(to []byte, topic Topic, msg []byte, asymmetric bool, key []byte) error {
	if key == nil || bytes.Equal(key, []byte{}) {
		return fmt.Errorf("Zero length key passed to pss send")
	}
	padding := make([]byte, self.paddingByteSize)
	c, err := rand.Read(padding)
	if err != nil {
		return err
	} else if c < self.paddingByteSize {
		return fmt.Errorf("invalid padding length: %d", c)
	}
	wparams := &whisper.MessageParams{
		TTL:      defaultWhisperTTL,
		Src:      self.privateKey,
		Topic:    whisper.TopicType(topic),
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  msg,
		Padding:  padding,
	}
	if asymmetric {
		wparams.Dst = crypto.ToECDSAPub(key)
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
	pssmsg := &PssMsg{
		To:      to,
		Expire:  uint32(time.Now().Add(self.msgTTL).Unix()),
		Payload: envelope,
	}
	if !self.enqueue(pssmsg) {
		return errors.New("outbox full!")
	}
	return nil
}

// Forwards a pss message to the peer(s) closest to the to recipient address in the PssMsg struct
// The recipient address can be of any length, and the byte slice will be matched to the MSB slice
// of the peer address of the equivalent length.
func (self *Pss) forward(msg *PssMsg) error {
	to := make([]byte, addressLength)
	copy(to[:len(msg.To)], msg.To)

	// send with kademlia
	// find the closest peer to the recipient and attempt to send
	sent := 0
	self.Overlay.EachConn(to, 256, func(op network.OverlayConn, po int, isproxbin bool) bool {
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
			if cap == self.capstring {
				ispss = true
				break
			}
		}
		if !ispss {
			log.Trace("peer doesn't have matching pss capabilities, skipping", "peer", info.Name, "caps", info.Caps)
			return true
		}

		// get the protocol peer from the forwarding peer cache
		sendMsg := fmt.Sprintf("MSG TO %x FROM %x VIA %x", to, self.BaseAddr(), op.Address())
		self.fwdPoolMu.RLock()
		pp := self.fwdPool[sp.Info().ID]
		self.fwdPoolMu.RUnlock()

		// attempt to send the message
		err := pp.Send(msg)
		if err != nil {
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
		time.Sleep(time.Millisecond)
		if !self.enqueue(msg) {
			return errors.New("outbox full!")
		}
	}

	// cache the message
	self.addFwdCache(msg)
	return nil
}

/////////////////////////////////////////////////////////////////////
// SECTION: Caching
/////////////////////////////////////////////////////////////////////

// cleanFwdCache is used to periodically remove expired entries from the forward cache
func (self *Pss) cleanFwdCache() {
	self.fwdCacheMu.Lock()
	defer self.fwdCacheMu.Unlock()
	for k, v := range self.fwdCache {
		if v.expiresAt.Before(time.Now()) {
			delete(self.fwdCache, k)
		}
	}
}

// add a message to the cache
func (self *Pss) addFwdCache(msg *PssMsg) error {
	var entry pssCacheEntry
	var ok bool

	self.fwdCacheMu.Lock()
	defer self.fwdCacheMu.Unlock()

	digest := self.digest(msg)
	if entry, ok = self.fwdCache[digest]; !ok {
		entry = pssCacheEntry{}
	}
	entry.expiresAt = time.Now().Add(self.cacheTTL)
	self.fwdCache[digest] = entry
	return nil
}

// check if message is in the cache
func (self *Pss) checkFwdCache(msg *PssMsg) bool {
	self.fwdCacheMu.Lock()
	defer self.fwdCacheMu.Unlock()

	digest := self.digest(msg)
	entry, ok := self.fwdCache[digest]
	if ok {
		if entry.expiresAt.After(time.Now()) {
			log.Trace(fmt.Sprintf("unexpired cache for digest %x", digest))
			return true
		}
	}
	return false
}

// Digest of message
func (self *Pss) digest(msg *PssMsg) pssDigest {
	hasher := self.hashPool.Get().(storage.SwarmHash)
	defer self.hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(msg.serialize())
	digest := pssDigest{}
	key := hasher.Sum(nil)
	copy(digest[:], key[:digestLength])
	return digest
}

func (self *Pss) isMsgExpired(msg *PssMsg) bool {
	msgexp := time.Unix(int64(msg.Expire), 0)
	if msgexp.Before(time.Now()) || msgexp.After(time.Now().Add(self.msgTTL)) {
		return true
	}
	return false
}
