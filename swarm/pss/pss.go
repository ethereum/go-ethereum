package pss

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	PssPeerCapacity             = 256 // limit of peers kept in cache. (not implemented)
	PssPeerTopicDefaultCapacity = 8   // limit of topics kept per peer. (not implemented)
	digestLength                = 32  // byte length of digest used for pss cache (currently same as swarm chunk hash)
	digestCapacity              = 256 // cache entry limit (not implement)
)

var (
	errorForwardToSelf = errors.New("forward to self")
	errorWhisper       = errors.New("whisper backend")
)

// abstraction to enable access to p2p.protocols.Peer.Send
type senderPeer interface {
	ID() discover.NodeID
	Address() []byte
	Send(interface{}) error
}

// protocol specification of the pss capsule
var pssSpec = &protocols.Spec{
	Name:       "pss",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		PssMsg{},
	},
}

type pssCacheEntry struct {
	expiresAt    time.Time
	receivedFrom []byte
}

type pssDigest [digestLength]byte

// Toplevel pss object, taking care of message sending and receiving, message handler dispatchers and message forwarding.
//
// Implements node.Service
type Pss struct {
	network.Overlay                                                         // we can get the overlayaddress from this
	peerPool        map[pot.Address]map[whisper.TopicType]p2p.MsgReadWriter // keep track of all virtual p2p.Peers we are currently speaking to
	fwdPool         map[discover.NodeID]*protocols.Peer                     // keep track of all peers sitting on the pssmsg routing layer
	keyPool         map[pot.Address]map[whisper.TopicType]ecdsa.PublicKey   // keep track of all public keys so we can encrypt for our peers
	reverseKeyPool  map[ecdsa.PublicKey]map[whisper.TopicType]pot.Address
	handlers        map[whisper.TopicType]map[*Handler]bool // topic and version based pss payload handlers
	fwdcache        map[pssDigest]pssCacheEntry             // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	cachettl        time.Duration                           // how long to keep messages in fwdcache
	lock            sync.Mutex
	dpa             *storage.DPA
	privatekey      *ecdsa.PrivateKey
}

func (self *Pss) storeMsg(msg *PssMsg) (pssDigest, error) {
	swg := &sync.WaitGroup{}
	wwg := &sync.WaitGroup{}
	buf := bytes.NewReader(msg.serialize())
	key, err := self.dpa.Store(buf, int64(buf.Len()), swg, wwg)
	if err != nil {
		log.Warn("Could not store in swarm", "err", err)
		return pssDigest{}, err
	}
	log.Trace("Stored msg in swarm", "key", key)
	digest := pssDigest{}
	copy(digest[:], key[:digestLength])
	return digest, nil
}

// Creates a new Pss instance.
//
// Needs a swarm network overlay, a DPA storage for message cache storage.
func NewPss(k network.Overlay, dpa *storage.DPA, params *PssParams) *Pss {
	return &Pss{
		Overlay:        k,
		peerPool:       make(map[pot.Address]map[whisper.TopicType]p2p.MsgReadWriter, PssPeerCapacity),
		fwdPool:        make(map[discover.NodeID]*protocols.Peer),
		keyPool:        make(map[pot.Address]map[whisper.TopicType]ecdsa.PublicKey),
		reverseKeyPool: make(map[ecdsa.PublicKey]map[whisper.TopicType]pot.Address),
		handlers:       make(map[whisper.TopicType]map[*Handler]bool),
		fwdcache:       make(map[pssDigest]pssCacheEntry),
		cachettl:       params.Cachettl,
		dpa:            dpa,
		privatekey:     params.privatekey,
	}
}

// Convenience accessor to the swarm overlay address of the pss node
func (self *Pss) BaseAddr() []byte {
	return self.Overlay.BaseAddr()
}

// For node.Service implementation. Does nothing for now, but should be included in the code for backwards compatibility.
func (self *Pss) Start(srv *p2p.Server) error {
	return nil
}

// For node.Service implementation. Does nothing for now, but should be included in the code for backwards compatibility.
func (self *Pss) Stop() error {
	return nil
}

// devp2p protocol object for the PssMsg struct.
//
// This represents the PssMsg capsule, and is the entry point for processing, receiving and sending pss messages between directly connected peers.
func (self *Pss) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		p2p.Protocol{
			Name:    pssSpec.Name,
			Version: pssSpec.Version,
			Length:  pssSpec.Length(),
			Run:     self.Run,
		},
	}
}

// Starts the PssMsg protocol
func (self *Pss) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	pp := protocols.NewPeer(p, rw, pssSpec)
	self.fwdPool[p.ID()] = pp
	return pp.Run(self.handlePssMsg)
}

// Exposes the API methods
//
// If the debug-parameter was given to the top Pss object, the TestAPI methods will also be included
func (self *Pss) APIs() []rpc.API {
	apis := []rpc.API{
		rpc.API{
			Namespace: "pss",
			Version:   "0.1",
			Service:   NewAPI(self),
			Public:    true,
		},
	}
	return apis
}

// Links a handler function to a Topic
//
// After calling this, all incoming messages with an envelope Topic matching the Topic specified will be passed to the given Handler function.
//
// Returns a deregister function which needs to be called to deregister the handler,
func (self *Pss) Register(topic *whisper.TopicType, handler Handler) func() {
	self.lock.Lock()
	defer self.lock.Unlock()
	handlers := self.handlers[*topic]
	if handlers == nil {
		handlers = make(map[*Handler]bool)
		self.handlers[*topic] = handlers
	}
	handlers[&handler] = true
	return func() { self.deregister(topic, &handler) }
}

// Add a Public key address mapping
// returns false if identical mapping already exists
func (self *Pss) AddPublicKey(addr pot.Address, topic whisper.TopicType, pubkey ecdsa.PublicKey) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if len(self.keyPool[addr]) == 0 {
		self.keyPool[addr] = make(map[whisper.TopicType]ecdsa.PublicKey)
		self.reverseKeyPool[pubkey] = make(map[whisper.TopicType]pot.Address)
	}
	self.keyPool[addr][topic] = pubkey
	self.reverseKeyPool[pubkey][topic] = addr
	return true
}

func (self *Pss) RemovePublicKey(addr pot.Address, topic whisper.TopicType, pubkey ecdsa.PublicKey) bool {
	if len(self.keyPool[addr]) == 0 {
		return false
	}
	zeroKey := ecdsa.PublicKey{}
	if self.keyPool[addr][topic] == zeroKey {
		return false
	}
	delete(self.reverseKeyPool, pubkey)
	self.keyPool[addr][topic] = zeroKey
	return true
}

func (self *Pss) GetKeys(addr pot.Address) (keys []ecdsa.PublicKey) {
outer:
	for _, key := range self.keyPool[addr] {
		for _, havekey := range keys {
			if havekey == key {
				continue outer
			}
		}
		keys = append(keys, key)
	}
	return
}

func (self *Pss) deregister(topic *whisper.TopicType, h *Handler) {
	self.lock.Lock()
	defer self.lock.Unlock()
	handlers := self.handlers[*topic]
	if len(handlers) == 1 {
		delete(self.handlers, *topic)
		return
	}
	delete(handlers, h)
}

// Adds an address/message pair to the cache
func (self *Pss) AddToCache(addr []byte, msg *PssMsg) error {
	digest, err := self.storeMsg(msg)
	if err != nil {
		return err
	}
	return self.addFwdCacheSender(addr, digest)
}

func (self *Pss) addFwdCacheSender(addr []byte, digest pssDigest) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	var entry pssCacheEntry
	var ok bool
	if entry, ok = self.fwdcache[digest]; !ok {
		entry = pssCacheEntry{}
	}
	entry.receivedFrom = addr
	self.fwdcache[digest] = entry
	return nil
}

func (self *Pss) addFwdCacheExpire(digest pssDigest) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	var entry pssCacheEntry
	var ok bool
	if entry, ok = self.fwdcache[digest]; !ok {
		entry = pssCacheEntry{}
	}
	entry.expiresAt = time.Now().Add(self.cachettl)
	self.fwdcache[digest] = entry
	return nil
}

func (self *Pss) checkFwdCache(addr []byte, digest pssDigest) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	entry, ok := self.fwdcache[digest]
	if ok {
		if entry.expiresAt.After(time.Now()) {
			log.Debug(fmt.Sprintf("unexpired cache for digest %x", digest))
			return true
		} else if entry.expiresAt.IsZero() && bytes.Equal(addr, entry.receivedFrom) {
			log.Debug(fmt.Sprintf("sendermatch %x for digest %x", common.ByteLabel(addr), digest))
			return true
		}
	}
	return false
}

func (self *Pss) getHandlers(topic whisper.TopicType) map[*Handler]bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.handlers[topic]
}

func (self *Pss) handlePssMsg(msg interface{}) error {
	pssmsg, ok := msg.(*PssMsg)
	if ok {
		if !self.isSelfRecipient(pssmsg) {
			log.Trace("pss was for someone else :'( ... forwarding")
			return self.Forward(pssmsg)
		}
		log.Trace("pss for us, yay! ... let's process!")

		return self.Process(pssmsg)
	}

	return fmt.Errorf("invalid message")
}

// Entry point to processing a message for which the current node is the intended recipient.
func (self *Pss) Process(pssmsg *PssMsg) error {
	env := pssmsg.Payload
	recvmsg, err := env.OpenAsymmetric(self.privatekey)
	if err != nil {
		// todo: add check on if key is full length and identical, then fail
		//return self.Forward(pssmsg)
		return fmt.Errorf("not for us", "err", err)
	}
	if !recvmsg.Validate() {
		return fmt.Errorf("invalid signature")
	}

	payload := recvmsg.Payload
	handlers := self.getHandlers(env.Topic)
	if len(handlers) == 0 {
		return fmt.Errorf("No registered handler for topic '%x'", env.Topic)
	}

	nid, _ := discover.HexID("0x00")
	p := p2p.NewPeer(nid, fmt.Sprintf("%x", recvmsg.Src), []p2p.Cap{})
	//addr := self.reverseKeyPool[common.ToHex(crypto.FromECDSAPub(recvmsg.Src))]
	addr := self.reverseKeyPool[*recvmsg.Src][recvmsg.Topic]
	log.Warn("recvkey", "key", *recvmsg.Src, "addr", addr)
	if addr.IsZero() {
		return fmt.Errorf("unknown key", "addr", addr)
	}

	for f := range handlers {
		err := (*f)(payload, p, addr.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

// Sends a message using Pss.
//
// This method is payload agnostic, and will accept any arbitrary byte slice as the payload for a message.
//
// It generates an envelope for the specified recipient and topic, and wraps the message payload in it.
func (self *Pss) SendAsym(to []byte, topic whisper.TopicType, msg []byte) error {
	var potaddr pot.Address
	copy(potaddr[:], to)
	topubkey := self.keyPool[potaddr][topic]
	log.Debug("using pubkey", "pubkey", topubkey)
	wparams := &whisper.MessageParams{
		TTL:      DefaultTTL,
		Src:      self.privatekey,
		Dst:      &topubkey,
		Topic:    topic,
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  msg,
	}

	// set up outgoing message container, which does encryption and envelope wrapping
	woutmsg, err := whisper.NewSentMessage(wparams)
	if err != nil {
		return fmt.Errorf("%v: %s", errorWhisper, err)
	}

	// performs encryption and PoW
	// after this the message is ready for sending
	env, err := woutmsg.Wrap(wparams)
	if err != nil {
		return fmt.Errorf("%v: %s", errorWhisper, err)
	}
	log.Trace("pssmsg whisper done", "env", env)

	pssmsg := &PssMsg{
		To:      to,
		Payload: env,
	}
	return self.Forward(pssmsg)
}

// Forwards a pss message to the peer(s) closest to the to address
//
// Handlers that are merely passing on the PssMsg to its final recipient should call this directly
func (self *Pss) Forward(msg *PssMsg) error {

	if self.isSelfRecipient(msg) {
		//return errorForwardToSelf
		return self.Process(msg)
	}

	// cache it
	digest, err := self.storeMsg(msg)
	if err != nil {
		log.Warn(fmt.Sprintf("could not store message %v to cache: %v", msg, err))
	}

	// flood guard
	if self.checkFwdCache(nil, digest) {
		log.Trace(fmt.Sprintf("pss relay block-cache match: FROM %x TO %x", common.ByteLabel(self.Overlay.BaseAddr()), common.ByteLabel(msg.To)))
		return nil
	}

	// TODO:check integrity of message
	sent := 0

	// send with kademlia
	// find the closest peer to the recipient and attempt to send
	self.Overlay.EachConn(msg.To, 256, func(op network.OverlayConn, po int, isproxbin bool) bool {
		sp, ok := op.(senderPeer)
		if !ok {
			log.Crit("Pss cannot use kademlia peer type")
			return false
		}
		sendMsg := fmt.Sprintf("MSG %x TO %x FROM %x VIA %x", digest, common.ByteLabel(msg.To), common.ByteLabel(self.BaseAddr()), common.ByteLabel(op.Address()))
		//sendMsg := fmt.Sprintf("TO %x FROM %x VIA %x", common.ByteLabel(msg.To), common.ByteLabel(self.BaseAddr()), common.ByteLabel(op.Address()))
		pp := self.fwdPool[sp.ID()]
		if self.checkFwdCache(op.Address(), digest) {
			log.Info(fmt.Sprintf("%v: peer already forwarded to", sendMsg))
			return true
		}
		err := pp.Send(msg)
		if err != nil {
			log.Warn(fmt.Sprintf("%v: failed forwarding: %v", sendMsg, err))
			return true
		}
		log.Debug(fmt.Sprintf("%v: successfully forwarded", sendMsg))
		sent++
		// if equality holds, p is always the first peer given in the iterator
		if bytes.Equal(msg.To, op.Address()) || !isproxbin {
			return false
		}
		log.Trace(fmt.Sprintf("%x is in proxbin, keep forwarding", common.ByteLabel(op.Address())))
		return true
	})

	if sent == 0 {
		log.Error("PSS: unable to forward to any peers")
		return fmt.Errorf("unable to forward to any peers")
	}

	self.addFwdCacheExpire(digest)
	return nil
}

// For devp2p protocol integration only. Analogous to an outgoing devp2p connection.
//
// Links a remote peer and Topic to a dedicated p2p.MsgReadWriter in the pss peerpool, and runs the specificed protocol using these resources.
//
// The effect is that now we have a "virtual" protocol running on an artificial p2p.Peer, which can be looked up and piped to through Pss using swarm overlay address and topic
func (self *Pss) AddPeer(p *p2p.Peer, addr pot.Address, run func(*p2p.Peer, p2p.MsgReadWriter) error, topic whisper.TopicType, rw p2p.MsgReadWriter) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.addPeerTopic(addr, topic, rw)
	go func() {
		err := run(p, rw)
		log.Warn(fmt.Sprintf("pss vprotocol quit on addr %v topic %v: %v", addr, topic, err))
		self.removePeerTopic(rw, topic)
	}()
	return nil
}

func (self *Pss) addPeerTopic(id pot.Address, topic whisper.TopicType, rw p2p.MsgReadWriter) error {
	if self.peerPool[id] == nil {
		self.peerPool[id] = make(map[whisper.TopicType]p2p.MsgReadWriter, PssPeerTopicDefaultCapacity)
	}
	self.peerPool[id][topic] = rw
	return nil
}

func (self *Pss) removePeerTopic(rw p2p.MsgReadWriter, topic whisper.TopicType) {
	prw, ok := rw.(*PssReadWriter)
	if !ok {
		return
	}
	delete(self.peerPool[prw.To], topic)
	if len(self.peerPool[prw.To]) == 0 {
		delete(self.peerPool, prw.To)
	}
}

func (self *Pss) isSelfRecipient(msg *PssMsg) bool {
	return bytes.Equal(msg.To, self.Overlay.BaseAddr())
}

func (self *Pss) isActive(id pot.Address, topic whisper.TopicType) bool {
	if self.peerPool[id] == nil {
		return false
	}
	return self.peerPool[id][topic] != nil
}

// For devp2p protocol integration only.
//
// Bridges pss send/receive with devp2p protocol send/receive
//
// Implements p2p.MsgReadWriter
type PssReadWriter struct {
	*Pss
	To         pot.Address
	LastActive time.Time
	rw         chan p2p.Msg
	spec       *protocols.Spec
	topic      *whisper.TopicType
}

// Implements p2p.MsgReader
func (prw PssReadWriter) ReadMsg() (p2p.Msg, error) {
	msg := <-prw.rw
	log.Trace(fmt.Sprintf("pssrw readmsg: %v", msg))
	return msg, nil
}

// Implements p2p.MsgWriter
func (prw PssReadWriter) WriteMsg(msg p2p.Msg) error {
	log.Trace("pssrw writemsg", "msg", msg)
	rlpdata := make([]byte, msg.Size)
	msg.Payload.Read(rlpdata)
	pmsg, err := rlp.EncodeToBytes(ProtocolMsg{
		Code:    msg.Code,
		Size:    msg.Size,
		Payload: rlpdata,
	})
	if err != nil {
		return err
	}
	return prw.SendAsym(prw.To.Bytes(), *prw.topic, pmsg)
}

// Injects a p2p.Msg into the MsgReadWriter, so that it appears on the associated p2p.MsgReader
func (prw PssReadWriter) injectMsg(msg p2p.Msg) error {
	log.Trace(fmt.Sprintf("pssrw injectmsg: %v", msg))
	prw.rw <- msg
	return nil
}

// For devp2p protocol integration only.
//
// Convenience object for passing messages in and out of the p2p layer
type PssProtocol struct {
	*Pss
	proto *p2p.Protocol
	topic *whisper.TopicType
	spec  *protocols.Spec
}

// For devp2p protocol integration only.
//
// Maps a Topic to a devp2p protocol.
func RegisterPssProtocol(ps *Pss, topic *whisper.TopicType, spec *protocols.Spec, targetprotocol *p2p.Protocol) *PssProtocol {
	pp := &PssProtocol{
		Pss:   ps,
		proto: targetprotocol,
		topic: topic,
		spec:  spec,
	}
	return pp
}

// For devp2p protocol integration only.
//
// Generic handler for initiating devp2p-like protocol connections
//
// This handler should be passed to Pss.Register with the associated ropic.
func (self *PssProtocol) Handle(msg []byte, p *p2p.Peer, senderAddr []byte) error {
	hashoaddr := pot.NewAddressFromBytes(senderAddr)
	if !self.isActive(hashoaddr, *self.topic) {
		rw := &PssReadWriter{
			Pss:   self.Pss,
			To:    hashoaddr,
			rw:    make(chan p2p.Msg),
			spec:  self.spec,
			topic: self.topic,
		}
		self.Pss.AddPeer(p, hashoaddr, self.proto.Run, *self.topic, rw)
	}

	pmsg, err := ToP2pMsg(msg)
	if err != nil {
		return fmt.Errorf("could not decode pssmsg")
	}

	vrw := self.Pss.peerPool[hashoaddr][*self.topic].(*PssReadWriter)
	vrw.injectMsg(pmsg)

	return nil
}

func getPadding() []byte {
	return []byte{0x64, 0x6f, 0x6f, 0x62, 0x61, 0x72}
}
