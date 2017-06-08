package pss

import (
	"bytes"
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
)

const (
	TopicResolverLength         = 8
	PssPeerCapacity             = 256
	PssPeerTopicDefaultCapacity = 8
	digestLength                = 32
	digestCapacity              = 256
)

var (
	errorForwardToSelf = errors.New("forward to self")
)

type senderPeer interface {
	ID() discover.NodeID
	Address() []byte
	Send(interface{}) error
}

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

// implements node.Service
//
// pss provides sending messages to nodes without having to be directly connected to them.
//
// The messages are wrapped in a PssMsg structure and routed using the swarm kademlia routing.
//
// The top-level Pss object provides:
//
// - access to the swarm overlay and routing (kademlia)
// - a collection of remote overlay addresses mapped to MsgReadWriters, representing the virtually connected peers
// - a collection of remote underlay address, mapped to the overlay addresses above
// - a method to send a message to specific overlayaddr
// - a dispatcher lookup, mapping protocols to topics
// - a message cache to spot messages that previously have been forwarded
type Pss struct {
	network.Overlay                                             // we can get the overlayaddress from this
	peerPool        map[pot.Address]map[Topic]p2p.MsgReadWriter // keep track of all virtual p2p.Peers we are currently speaking to
	//fwdPool         map[pot.Address]*protocols.Peer             // keep track of all peers sitting on the pssmsg routing layer
	fwdPool         map[discover.NodeID]*protocols.Peer             // keep track of all peers sitting on the pssmsg routing layer
	handlers        map[Topic]map[*Handler]bool                 // topic and version based pss payload handlers
	fwdcache        map[pssDigest]pssCacheEntry                 // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	cachettl        time.Duration                               // how long to keep messages in fwdcache
	lock            sync.Mutex
	dpa             *storage.DPA
	debug           bool
}

func (self *Pss) storeMsg(msg *PssMsg) (pssDigest, error) {
	swg := &sync.WaitGroup{}
	wwg := &sync.WaitGroup{}
	buf := bytes.NewReader(msg.Serialize())
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

// Creates a new Pss instance. A node should only need one of these
func NewPss(k network.Overlay, dpa *storage.DPA, params *PssParams) *Pss {
	return &Pss{
		Overlay:  k,
		peerPool: make(map[pot.Address]map[Topic]p2p.MsgReadWriter, PssPeerCapacity),
		//fwdPool:  make(map[pot.Address]*protocols.Peer),
		fwdPool:  make(map[discover.NodeID]*protocols.Peer),
		handlers: make(map[Topic]map[*Handler]bool),
		fwdcache: make(map[pssDigest]pssCacheEntry),
		cachettl: params.Cachettl,
		dpa:      dpa,
		debug:    params.Debug,
	}
}

func (self *Pss) BaseAddr() []byte {
	return self.Overlay.BaseAddr()
}

func (self *Pss) Start(srv *p2p.Server) error {
	return nil
}

func (self *Pss) Stop() error {
	return nil
}

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

func (self *Pss) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	pp := protocols.NewPeer(p, rw, pssSpec)
	//addr := network.NewAddrFromNodeID(id)
	//potaddr := pot.NewHashAddressFromBytes(addr.OAddr)
	//self.fwdPool[potaddr.Address] = pp
	self.fwdPool[p.ID()] = pp

	return pp.Run(self.handlePssMsg)
}

func (self *Pss) APIs() []rpc.API {
	apis := []rpc.API{
		rpc.API{
			Namespace: "pss",
			Version:   "0.1",
			Service:   NewAPI(self),
			Public:    true,
		},
	}
	if self.debug {
		apis = append(apis, rpc.API{
			Namespace: "pss",
			Version:   "0.1",
			Service:   NewAPITest(self),
			Public:    true,
		})
	}
	return apis
}

// Takes the generated Topic of a protocol/chatroom etc, and links a handler function to it
// This allows the implementer to retrieve the right handler functions (invoke the right protocol)
// for an incoming message by inspecting the topic on it.
// a topic allows for multiple handlers
// returns a deregister function which needs to be called to deregister the handler
// (similar to event.Subscription.Unsubscribe())
func (self *Pss) Register(topic *Topic, handler Handler) func() {
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

func (self *Pss) deregister(topic *Topic, h *Handler) {
	self.lock.Lock()
	defer self.lock.Unlock()
	handlers := self.handlers[*topic]
	if len(handlers) == 1 {
		delete(self.handlers, *topic)
		return
	}
	delete(handlers, h)
}

// enables to set address of node, to avoid backwards forwarding
//
// currently not in use as forwarder address is not known in the handler function hooked to the pss dispatcher.
// it is included as a courtesy to custom transport layers that may want to implement this
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

func (self *Pss) getHandlers(topic Topic) map[*Handler]bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.handlers[topic]
}

func (self *Pss) handlePssMsg(msg interface{}) error {
	pssmsg := msg.(*PssMsg)

	if !self.isSelfRecipient(pssmsg) {
		log.Trace("pss was for someone else :'( ... forwarding")
		return self.Forward(pssmsg)
	}
	log.Trace("pss for us, yay! ... let's process!")
	return self.Process(pssmsg)
}

// processes a message with self as recipient
func (self *Pss) Process(pssmsg *PssMsg) error {
	env := pssmsg.Payload
	payload := env.Payload
	handlers := self.getHandlers(env.Topic)
	if len(handlers) == 0 {
		return fmt.Errorf("No registered handler for topic '%s'", env.Topic)
	}
	nid, _ := discover.HexID("0x00")
	p := p2p.NewPeer(nid, fmt.Sprintf("%x", env.From), []p2p.Cap{})
	for f := range handlers {
		err := (*f)(payload, p, env.From)
		if err != nil {
			return err
		}
	}
	return nil
}

// Sends a message using  The message could be anything at all, and will be handled by whichever handler function is mapped to Topic using *Pss.Register()
//
// The to address is a swarm overlay address
func (self *Pss) SendRaw(to []byte, topic Topic, msg []byte) error {
	sender := self.Overlay.BaseAddr()
	pssenv := NewEnvelope(sender, topic, msg)
	pssmsg := &PssMsg{
		To:      to,
		Payload: pssenv,
	}
	return self.Forward(pssmsg)
}

// Forwards a pss message to the peer(s) closest to the to address
//
// Handlers that want to pass on a message should call this directly
func (self *Pss) Forward(msg *PssMsg) error {

	if self.isSelfRecipient(msg) {
		return errorForwardToSelf
	}

	digest, err := self.storeMsg(msg)
	if err != nil {
		log.Warn(fmt.Sprintf("could not store message %v to cache: %v", msg, err))
	}

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
		sendMsg := fmt.Sprintf("TO %x FROM %x VIA %x", common.ByteLabel(msg.To), common.ByteLabel(self.BaseAddr()), common.ByteLabel(op.Address()))
		//h := pot.NewHashAddressFromBytes(op.Address())
		//pp := self.fwdPool[h.Address]
		pp := self.fwdPool[sp.ID()]
		if self.checkFwdCache(op.Address(), digest) {
			log.Info("%v: peer already forwarded to", sendMsg)
			return true
		}
		err := pp.Send(msg)
		//err := sp.Send(msg)
		if err != nil {
			log.Warn(fmt.Sprintf("%v: failed forwarding: %v", sendMsg, err))
			return true
		}
		log.Trace(fmt.Sprintf("%v: successfully forwarded", sendMsg))
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
		return nil
	}

	self.addFwdCacheExpire(digest)
	return nil
}

// Links a pss peer address and topic to a dedicated p2p.MsgReadWriter in the pss peerpool, and runs the specificed protocol on this p2p.MsgReadWriter and the specified peer
//
// The effect is that now we have a "virtual" protocol running on an artificial p2p.Peer, which can be looked up and piped to through Pss using swarm overlay address and topic
func (self *Pss) AddPeer(p *p2p.Peer, addr pot.Address, run func(*p2p.Peer, p2p.MsgReadWriter) error, topic Topic, rw p2p.MsgReadWriter) error {
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

func (self *Pss) addPeerTopic(id pot.Address, topic Topic, rw p2p.MsgReadWriter) error {
	if self.peerPool[id] == nil {
		self.peerPool[id] = make(map[Topic]p2p.MsgReadWriter, PssPeerTopicDefaultCapacity)
	}
	self.peerPool[id][topic] = rw
	return nil
}

func (self *Pss) removePeerTopic(rw p2p.MsgReadWriter, topic Topic) {
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

func (self *Pss) isActive(id pot.Address, topic Topic) bool {
	if self.peerPool[id] == nil {
		return false
	}
	return self.peerPool[id][topic] != nil
}

// Convenience object that:
//
// - allows passing of the unwrapped PssMsg payload to the p2p level message handlers
// - interprets outgoing p2p.Msg from the p2p level to pass in to *Pss.Send()
//
// Implements p2p.MsgReadWriter
type PssReadWriter struct {
	*Pss
	To         pot.Address
	LastActive time.Time
	rw         chan p2p.Msg
	spec       *protocols.Spec
	topic      *Topic
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
	return prw.SendRaw(prw.To.Bytes(), *prw.topic, pmsg)
}

// Injects a p2p.Msg into the MsgReadWriter, so that it appears on the associated p2p.MsgReader
func (prw PssReadWriter) injectMsg(msg p2p.Msg) error {
	log.Trace(fmt.Sprintf("pssrw injectmsg: %v", msg))
	prw.rw <- msg
	return nil
}

// Convenience object for passing messages in and out of the p2p layer
type PssProtocol struct {
	*Pss
	proto *p2p.Protocol
	topic *Topic
	spec  *protocols.Spec
}

// Constructor
func RegisterPssProtocol(ps *Pss, topic *Topic, spec *protocols.Spec, targetprotocol *p2p.Protocol) *PssProtocol {
	pp := &PssProtocol{
		Pss:   ps,
		proto: targetprotocol,
		topic: topic,
		spec:  spec,
	}
	return pp
}

func (self *PssProtocol) Handle(msg []byte, p *p2p.Peer, senderAddr []byte) error {
	hashoaddr := pot.NewHashAddressFromBytes(senderAddr).Address
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

