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
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"golang.org/x/crypto/sha3"
)

const (
	defaultPaddingByteSize     = 16
	DefaultMsgTTL              = time.Second * 120
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
	ID() enode.ID
	Address() []byte
	Send(context.Context, interface{}) error
}

// per-key peer related information
// member `protected` prevents garbage collection of the instance
type pssPeer struct {
	lastSeen  time.Time
	address   PssAddress
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
		MsgTTL:              DefaultMsgTTL,
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
	*network.Kademlia // we can get the Kademlia address from this
	*KeyStore

	privateKey *ecdsa.PrivateKey // pss can have it's own independent key
	auxAPIs    []rpc.API         // builtins (handshake, test) can add APIs

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

	// message handling
	handlers           map[Topic]map[*handler]bool // topic and version based pss payload handlers. See pss.Handle()
	handlersMu         sync.RWMutex
	hashPool           sync.Pool
	topicHandlerCaps   map[Topic]*handlerCaps // caches capabilities of each topic's handlers
	topicHandlerCapsMu sync.RWMutex

	// process
	quitC chan struct{}
}

func (p *Pss) String() string {
	return fmt.Sprintf("pss: addr %x, pubkey %v", p.BaseAddr(), common.ToHex(crypto.FromECDSAPub(&p.privateKey.PublicKey)))
}

// Creates a new Pss instance.
//
// In addition to params, it takes a swarm network Kademlia
// and a FileStore storage for message cache storage.
func NewPss(k *network.Kademlia, params *PssParams) (*Pss, error) {
	if params.privateKey == nil {
		return nil, errors.New("missing private key for pss")
	}
	cap := p2p.Cap{
		Name:    pssProtocolName,
		Version: pssVersion,
	}
	ps := &Pss{
		Kademlia: k,
		KeyStore: loadKeyStore(),

		privateKey: params.privateKey,
		quitC:      make(chan struct{}),

		fwdPool:         make(map[string]*protocols.Peer),
		fwdCache:        make(map[pssDigest]pssCacheEntry),
		cacheTTL:        params.CacheTTL,
		msgTTL:          params.MsgTTL,
		paddingByteSize: defaultPaddingByteSize,
		capstring:       cap.String(),
		outbox:          make(chan *PssMsg, defaultOutboxCapacity),

		handlers:         make(map[Topic]map[*handler]bool),
		topicHandlerCaps: make(map[Topic]*handlerCaps),

		hashPool: sync.Pool{
			New: func() interface{} {
				return sha3.NewLegacyKeccak256()
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
	log.Info("Started Pss")
	log.Info("Loaded EC keys", "pubkey", common.ToHex(crypto.FromECDSAPub(p.PublicKey())), "secp256", common.ToHex(crypto.CompressPubkey(p.PublicKey())))
	return nil
}

func (p *Pss) Stop() error {
	log.Info("Pss shutting down")
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

// Returns the swarm Kademlia address of the pss node
func (p *Pss) BaseAddr() []byte {
	return p.Kademlia.BaseAddr()
}

// Returns the pss node's public key
func (p *Pss) PublicKey() *ecdsa.PublicKey {
	return &p.privateKey.PublicKey
}

/////////////////////////////////////////////////////////////////////
// SECTION: Message handling
/////////////////////////////////////////////////////////////////////

func (p *Pss) getTopicHandlerCaps(topic Topic) (hc *handlerCaps, found bool) {
	p.topicHandlerCapsMu.RLock()
	defer p.topicHandlerCapsMu.RUnlock()
	hc, found = p.topicHandlerCaps[topic]
	return
}

func (p *Pss) setTopicHandlerCaps(topic Topic, hc *handlerCaps) {
	p.topicHandlerCapsMu.Lock()
	defer p.topicHandlerCapsMu.Unlock()
	p.topicHandlerCaps[topic] = hc
}

// Links a handler function to a Topic
//
// All incoming messages with an envelope Topic matching the
// topic specified will be passed to the given Handler function.
//
// There may be an arbitrary number of handler functions per topic.
//
// Returns a deregister function which needs to be called to
// deregister the handler,
func (p *Pss) Register(topic *Topic, hndlr *handler) func() {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	handlers := p.handlers[*topic]
	if handlers == nil {
		handlers = make(map[*handler]bool)
		p.handlers[*topic] = handlers
		log.Debug("registered handler", "capabilities", hndlr.caps)
	}
	if hndlr.caps == nil {
		hndlr.caps = &handlerCaps{}
	}
	handlers[hndlr] = true

	capabilities, ok := p.getTopicHandlerCaps(*topic)
	if !ok {
		capabilities = &handlerCaps{}
		p.setTopicHandlerCaps(*topic, capabilities)
	}

	if hndlr.caps.raw {
		capabilities.raw = true
	}
	if hndlr.caps.prox {
		capabilities.prox = true
	}
	return func() { p.deregister(topic, hndlr) }
}

func (p *Pss) deregister(topic *Topic, hndlr *handler) {
	p.handlersMu.Lock()
	defer p.handlersMu.Unlock()
	handlers := p.handlers[*topic]
	if len(handlers) > 1 {
		delete(p.handlers, *topic)
		// topic caps might have changed now that a handler is gone
		caps := &handlerCaps{}
		for h := range handlers {
			if h.caps.raw {
				caps.raw = true
			}
			if h.caps.prox {
				caps.prox = true
			}
		}
		p.setTopicHandlerCaps(*topic, caps)
		return
	}
	delete(handlers, hndlr)
}

// Filters incoming messages for processing or forwarding.
// Check if address partially matches
// If yes, it CAN be for us, and we process it
// Only passes error to pss protocol handler if payload is not valid pssmsg
func (p *Pss) handlePssMsg(ctx context.Context, msg interface{}) error {
	metrics.GetOrRegisterCounter("pss.handlepssmsg", nil).Inc(1)
	pssmsg, ok := msg.(*PssMsg)
	if !ok {
		return fmt.Errorf("invalid message type. Expected *PssMsg, got %T ", msg)
	}
	log.Trace("handler", "self", label(p.Kademlia.BaseAddr()), "topic", label(pssmsg.Payload.Topic[:]))
	if int64(pssmsg.Expire) < time.Now().Unix() {
		metrics.GetOrRegisterCounter("pss.expire", nil).Inc(1)
		log.Warn("pss filtered expired message", "from", common.ToHex(p.Kademlia.BaseAddr()), "to", common.ToHex(pssmsg.To))
		return nil
	}
	if p.checkFwdCache(pssmsg) {
		log.Trace("pss relay block-cache match (process)", "from", common.ToHex(p.Kademlia.BaseAddr()), "to", (common.ToHex(pssmsg.To)))
		return nil
	}
	p.addFwdCache(pssmsg)

	psstopic := Topic(pssmsg.Payload.Topic)

	// raw is simplest handler contingency to check, so check that first
	var isRaw bool
	if pssmsg.isRaw() {
		if capabilities, ok := p.getTopicHandlerCaps(psstopic); ok {
			if !capabilities.raw {
				log.Debug("No handler for raw message", "topic", psstopic)
				return nil
			}
		}
		isRaw = true
	}

	// check if we can be recipient:
	// - no prox handler on message and partial address matches
	// - prox handler on message and we are in prox regardless of partial address match
	// store this result so we don't calculate again on every handler
	var isProx bool
	if capabilities, ok := p.getTopicHandlerCaps(psstopic); ok {
		isProx = capabilities.prox
	}
	isRecipient := p.isSelfPossibleRecipient(pssmsg, isProx)
	if !isRecipient {
		log.Trace("pss was for someone else :'( ... forwarding", "pss", common.ToHex(p.BaseAddr()), "prox", isProx)
		return p.enqueue(pssmsg)
	}

	log.Trace("pss for us, yay! ... let's process!", "pss", common.ToHex(p.BaseAddr()), "prox", isProx, "raw", isRaw, "topic", label(pssmsg.Payload.Topic[:]))
	if err := p.process(pssmsg, isRaw, isProx); err != nil {
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
func (p *Pss) process(pssmsg *PssMsg, raw bool, prox bool) error {
	metrics.GetOrRegisterCounter("pss.process", nil).Inc(1)

	var err error
	var recvmsg *whisper.ReceivedMessage
	var payload []byte
	var from PssAddress
	var asymmetric bool
	var keyid string
	var keyFunc func(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, PssAddress, error)

	envelope := pssmsg.Payload
	psstopic := Topic(envelope.Topic)

	if raw {
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
	p.executeHandlers(psstopic, payload, from, raw, prox, asymmetric, keyid)

	return nil
}

// copy all registered handlers for respective topic in order to avoid data race or deadlock
func (p *Pss) getHandlers(topic Topic) (ret []*handler) {
	p.handlersMu.RLock()
	defer p.handlersMu.RUnlock()
	for k := range p.handlers[topic] {
		ret = append(ret, k)
	}
	return ret
}

func (p *Pss) executeHandlers(topic Topic, payload []byte, from PssAddress, raw bool, prox bool, asymmetric bool, keyid string) {
	handlers := p.getHandlers(topic)
	peer := p2p.NewPeer(enode.ID{}, fmt.Sprintf("%x", from), []p2p.Cap{})
	for _, h := range handlers {
		if !h.caps.raw && raw {
			log.Warn("norawhandler")
			continue
		}
		if !h.caps.prox && prox {
			log.Warn("noproxhandler")
			continue
		}
		err := (h.f)(payload, peer, asymmetric, keyid)
		if err != nil {
			log.Warn("Pss handler failed", "err", err)
		}
	}
}

// will return false if using partial address
func (p *Pss) isSelfRecipient(msg *PssMsg) bool {
	return bytes.Equal(msg.To, p.Kademlia.BaseAddr())
}

// test match of leftmost bytes in given message to node's Kademlia address
func (p *Pss) isSelfPossibleRecipient(msg *PssMsg, prox bool) bool {
	local := p.Kademlia.BaseAddr()

	// if a partial address matches we are possible recipient regardless of prox
	// if not and prox is not set, we are surely not
	if bytes.Equal(msg.To, local[:len(msg.To)]) {

		return true
	} else if !prox {
		return false
	}

	depth := p.Kademlia.NeighbourhoodDepth()
	po, _ := network.Pof(p.Kademlia.BaseAddr(), msg.To, 0)
	log.Trace("selfpossible", "po", po, "depth", depth)

	return depth <= po
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
	if err := validateAddress(address); err != nil {
		return err
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
	err := p.enqueue(pssMsg)
	if err != nil {
		return err
	}

	// if we have a proxhandler on this topic
	// also deliver message to ourselves
	if capabilities, ok := p.getTopicHandlerCaps(topic); ok {
		if p.isSelfPossibleRecipient(pssMsg, true) && capabilities.prox {
			return p.process(pssMsg, true, true)
		}
	}
	return nil
}

// Send a message using symmetric encryption
//
// Fails if the key id does not match any of the stored symmetric keys
func (p *Pss) SendSym(symkeyid string, topic Topic, msg []byte) error {
	symkey, err := p.GetSymmetricKey(symkeyid)
	if err != nil {
		return fmt.Errorf("missing valid send symkey %s: %v", symkeyid, err)
	}
	psp, ok := p.getPeerSym(symkeyid, topic)
	if !ok {
		return fmt.Errorf("invalid topic '%s' for symkey '%s'", topic.String(), symkeyid)
	}
	return p.send(psp.address, topic, msg, false, symkey)
}

// Send a message using asymmetric encryption
//
// Fails if the key id does not match any in of the stored public keys
func (p *Pss) SendAsym(pubkeyid string, topic Topic, msg []byte) error {
	if _, err := crypto.UnmarshalPubkey(common.FromHex(pubkeyid)); err != nil {
		return fmt.Errorf("Cannot unmarshal pubkey: %x", pubkeyid)
	}
	psp, ok := p.getPeerPub(pubkeyid, topic)
	if !ok {
		return fmt.Errorf("invalid topic '%s' for pubkey '%s'", topic.String(), pubkeyid)
	}
	return p.send(psp.address, topic, msg, true, common.FromHex(pubkeyid))
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
	err = p.enqueue(pssMsg)
	if err != nil {
		return err
	}
	if capabilities, ok := p.getTopicHandlerCaps(topic); ok {
		if p.isSelfPossibleRecipient(pssMsg, true) && capabilities.prox {
			return p.process(pssMsg, true, true)
		}
	}
	return nil
}

// sendFunc is a helper function that tries to send a message and returns true on success.
// It is set here for usage in production, and optionally overridden in tests.
var sendFunc = sendMsg

// tries to send a message, returns true if successful
func sendMsg(p *Pss, sp *network.Peer, msg *PssMsg) bool {
	var isPssEnabled bool
	info := sp.Info()
	for _, capability := range info.Caps {
		if capability == p.capstring {
			isPssEnabled = true
			break
		}
	}
	if !isPssEnabled {
		log.Error("peer doesn't have matching pss capabilities, skipping", "peer", info.Name, "caps", info.Caps)
		return false
	}

	// get the protocol peer from the forwarding peer cache
	p.fwdPoolMu.RLock()
	pp := p.fwdPool[sp.Info().ID]
	p.fwdPoolMu.RUnlock()

	err := pp.Send(context.TODO(), msg)
	if err != nil {
		metrics.GetOrRegisterCounter("pss.pp.send.error", nil).Inc(1)
		log.Error(err.Error())
	}

	return err == nil
}

// Forwards a pss message to the peer(s) based on recipient address according to the algorithm
// described below. The recipient address can be of any length, and the byte slice will be matched
// to the MSB slice of the peer address of the equivalent length.
//
// If the recipient address (or partial address) is within the neighbourhood depth of the forwarding
// node, then it will be forwarded to all the nearest neighbours of the forwarding node. In case of
// partial address, it should be forwarded to all the peers matching the partial address, if there
// are any; otherwise only to one peer, closest to the recipient address. In any case, if the message
// forwarding fails, the node should try to forward it to the next best peer, until the message is
// successfully forwarded to at least one peer.
func (p *Pss) forward(msg *PssMsg) error {
	metrics.GetOrRegisterCounter("pss.forward", nil).Inc(1)
	sent := 0 // number of successful sends
	to := make([]byte, addressLength)
	copy(to[:len(msg.To)], msg.To)
	neighbourhoodDepth := p.Kademlia.NeighbourhoodDepth()

	// luminosity is the opposite of darkness. the more bytes are removed from the address, the higher is darkness,
	// but the luminosity is less. here luminosity equals the number of bits given in the destination address.
	luminosityRadius := len(msg.To) * 8

	// proximity order function matching up to neighbourhoodDepth bits (po <= neighbourhoodDepth)
	pof := pot.DefaultPof(neighbourhoodDepth)

	// soft threshold for msg broadcast
	broadcastThreshold, _ := pof(to, p.BaseAddr(), 0)
	if broadcastThreshold > luminosityRadius {
		broadcastThreshold = luminosityRadius
	}

	var onlySendOnce bool // indicates if the message should only be sent to one peer with closest address

	// if measured from the recipient address as opposed to the base address (see Kademlia.EachConn
	// call below), then peers that fall in the same proximity bin as recipient address will appear
	// [at least] one bit closer, but only if these additional bits are given in the recipient address.
	if broadcastThreshold < luminosityRadius && broadcastThreshold < neighbourhoodDepth {
		broadcastThreshold++
		onlySendOnce = true
	}

	p.Kademlia.EachConn(to, addressLength*8, func(sp *network.Peer, po int) bool {
		if po < broadcastThreshold && sent > 0 {
			return false // stop iterating
		}
		if sendFunc(p, sp, msg) {
			sent++
			if onlySendOnce {
				return false
			}
			if po == addressLength*8 {
				// stop iterating if successfully sent to the exact recipient (perfect match of full address)
				return false
			}
		}
		return true
	})

	// if we failed to send to anyone, re-insert message in the send-queue
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

func label(b []byte) string {
	return fmt.Sprintf("%04x", b[:2])
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
	return p.digestBytes(msg.serialize())
}

func (p *Pss) digestBytes(msg []byte) pssDigest {
	hasher := p.hashPool.Get().(hash.Hash)
	defer p.hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(msg)
	digest := pssDigest{}
	key := hasher.Sum(nil)
	copy(digest[:], key[:digestLength])
	return digest
}

func validateAddress(addr PssAddress) error {
	if len(addr) > addressLength {
		return errors.New("address too long")
	}
	return nil
}
