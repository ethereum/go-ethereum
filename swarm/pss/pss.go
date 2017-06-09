// pss provides devp2p functionality for swarm nodes without the need for a direct tcp connection between them.
//
// It uses swarm kademlia routing to send and receive messages. Routing is deterministic and will seek the shortest route available on the network.
//
// Messages are encapsulated in a devp2p message structure `PssMsg`. These capsules are forwarded from node to node using ordinary tcp devp2p until it reaches it's destination. The destination address is hinted in `PssMsg.To`
//
// The content of a PssMsg can be anything at all, down to a simple, non-descript byte-slices. But convenience methods are made available to implement devp2p protocol functionality on top of it.
//
// In its final implementation, pss is intended to become "shh over bzz,"  that is; "whisper over swarm." Specifically, this means that the emphemeral encryption envelopes of whisper will be used to obfuscate the correspondance. Ideally, the unencrypted content of the PssMsg will only contain a part of the address of the recipient, where the final recipient is the one who matches this partial address *and* successfully can encrypt the message.
//
// For the current state and roadmap of pss development please see https://github.com/ethersphere/swarm/wiki/swarm-dev-progress.
//
// Please report issues on https://github.com/ethersphere/go-ethereum
//
// Feel free to ask questions in https://gitter.im/ethersphere/pss
//
// TLDR IMPLEMENTATION
//
// Most developers will most probably want to use the protocol-wrapping convenience client in swarm/pss/client. Documentation and a minimal code example for the latter is found in the package documentation. The pss API can of course also be used directly. The client implementation provides a clear illustration of its intended usage.
//
// pss implements the node.Service interface. This means that the API methods will be auto-magically exposed to any RPC layer the node activates. In particular, pss provides subscription to incoming messages using the go-ethereum rpc websocket layer. 	
//
// The important API methods are:
// - Receive() - start a subscription to receive new incoming messages matching specific "topics"
// - Send() - send content over pss to a specified recipient
//
//
// LOWLEVEL IMPLEMENTATION
//
// code speaks louder than words:
// 
//    import (
//    	"io/ioutil"
//    	"os"
//    	"github.com/ethereum/go-ethereum/p2p"
//    	"github.com/ethereum/go-ethereum/log"
//    	"github.com/ethereum/go-ethereum/swarm/pss"
//    	"github.com/ethereum/go-ethereum/swarm/network"
//    	"github.com/ethereum/go-ethereum/swarm/storage"
//    )
//    
//    var (
//    	righttopic = pss.NewTopic("foo", 4)
//    	wrongtopic = pss.NewTopic("bar", 2)
//    )
//    
//    func init() {
//    	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
//    	hf := log.LvlFilterHandler(log.LvlTrace, hs)
//    	h := log.CallerFileHandler(hf)
//    	log.Root().SetHandler(h)
//    }
//    
//    
//    // Pss.Handler type
//    func handler(msg []byte, p *p2p.Peer, from []byte) error {
//    	log.Debug("received", "msg", msg, "from", from, "forwarder", p.ID())
//    	return nil
//    }
//    
//    func implementation() {
//    
//    	// bogus addresses for illustration purposes
//    	meaddr := network.RandomAddr()
//    	toaddr := network.RandomAddr()
//    	fwdaddr := network.RandomAddr()
//    
//    	// new kademlia for routing
//    	kp := network.NewKadParams()
//    	to := network.NewKademlia(meaddr.Over(), kp)
//    
//    	// new (local) storage for cache
//    	cachedir, err := ioutil.TempDir("", "pss-cache")
//    	if err != nil {
//    		panic("overlay")
//    	}
//    	dpa, err := storage.NewLocalDPA(cachedir)
//    	if err != nil {
//    		panic("storage")
//    	}
//    
//    	// setup pss
//    	psp := pss.NewPssParams(false)
//    	ps := pss.NewPss(to, dpa, psp)
//    
//    	// does nothing but please include it
//    	ps.Start(nil)
//    
//    	dereg := ps.Register(&righttopic, handler)
//    
//    	// in its simplest form a message is just a byteslice
//    	payload := []byte("foobar")
//    
//    	// send a raw message
//    	err = ps.SendRaw(toaddr.Over(), righttopic, payload)
//    	log.Error("Fails. Not connect, so nothing in kademlia. But it illustrates the point.", "err", err)
//    
//    	// forward a full message
//    	envfwd := pss.NewEnvelope(fwdaddr.Over(), righttopic, payload)
//    	msgfwd := &pss.PssMsg{
//    		To: toaddr.Over(),
//    		Payload: envfwd,
//    	}
//    	err = ps.Forward(msgfwd)
//    	log.Error("Also fails, same reason. I wish, I wish, I wish there was somebody out there.", "err", err)
//    
//    	// process an incoming message
//    	// (this is the first step after the devp2p PssMsg message handler)
//    	envme := pss.NewEnvelope(toaddr.Over(), righttopic, payload)
//    	msgme := &pss.PssMsg{
//    		To: meaddr.Over(),
//    		Payload: envme,
//    	}
//    	err = ps.Process(msgme)
//    	if err == nil {
//    		log.Info("this works :)")
//    	}
//    
//    	// if we don't have a registered topic it fails
//    	dereg() // remove the previously registered topic-handler link
//    	ps.Process(msgme)
//    	log.Error("It fails as we expected", "err", err)
//    
//    	// does nothing but please include it
//    	ps.Stop()
//    }
//
// MESSAGE STRUCTURE
//
// NOTE! This part is subject to change. In particular the envelope structure will be re-implemented using whisper.
//
// A pss message has the following layers:
//
//     PssMsg
// Contains (eventually only part of) recipient address, and (eventually) encrypted Envelope. 
//
//     Envelope
// Currently rlp-encoded. Contains the Payload, along with sender address, topic and expiry information.
//
//     Payload
// Byte-slice of arbitrary data
//
//     ProtocolMsg
// An optional convenience structure for implementation of devp2p protocols. Contains Code, Size and Payload analogous to the p2p.Msg structure, where the payload is a rlp-encoded byteslice. For transport, this struct is serialized and used as the "payload" above.
//
// TOPICS AND PROTOCOLS
//
// Pure pss is protocol agnostic. Instead it uses the notion of Topic. This is NOT the "subject" of a message. Instead this type is used to internally register handlers for messages matching respective Topics.
//
// Topic in this context virtually mean anything; protocols, chatrooms, or social media groups.
//
// When implementing devp2p protocols, topics are direct mappings to protocols name and version. The pss package provides the PssProtocol convenience structure, and a generic Handler that can be passed to Pss.Register. This makes it possible to use the same message handler code  for pss that are used for direct connected peers.
//
// CONNECTIONS 
//
// A "connection" in pss is a purely virtual construct. There is no mechanisms in place to ensure that the remote peer actually is there. In fact, "adding" a peer involves merely the node's opinion that the peer is there. It may issue messages to that remote peer to a directly connected peer, which in turn passes it on. But if it is not present on the network - or if there is no route to it - the message will never reach its destination through mere forwarding.
//
// When implementing the devp2p protocol stack, the "adding" of a remote peer is a prerequisite for the side actually initiating the protocol communication. Adding a peer in effect "runs" the protocol on that peer, and adds an internal mapping between a topic and that peer. It also enables sending and receiving messages using the main io-construct in devp2p - the p2p.MsgReadWriter.
//
// Under the hood, pss implements its own MsgReadWriter, which bridges MsgReadWriter.WriteMsg with Pss.SendRaw, and deftly adds an InjectMsg method which pipes incoming messages to appear on the MsgReadWriter.ReadMsg channel.
//
// An incoming connection is nothing more than an actual PssMsg appearing with a certain Topic. If a Handler har been registered to that Topic, the message will be passed to it. This constitutes a "new" connection if:
//
// - The pss node never called AddPeer with this combination of remote peer address and topic, and
//
// - The pss node never received a PssMsg from this remote peer with this specific Topic before.
//
// If it is a "new" connection, the protocol will be "run" on the remote peer, in the same manner as if it was pre-emptively added.
//
// ROUTING AND CACHING
//
// (please refer to swarm kademlia routing for an explanation of the routing algorithm used for pss)
//
// pss implements a simple caching mechanism, using the swarm DPA for storage of the messages and generation of the digest keys used in the cache table. The caching is intended to alleviate the following:
//
// - save messages so that they can be delivered later if the recipient was not online at the time of sending.
//
// - drop an identical message to the same recipient if received within a given time interval
//
// - prevent backwards routing of messages
//
// the latter may occur if only one entry is in the receiving node's kademlia. In this case the forwarder will be provided as the "nearest node" to the final recipient. The cache keeps the address of who the message was forwarded from, and if the cache lookup matches, the message will be dropped.
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
	PssPeerCapacity             = 256 // limit of peers kept in cache. (not implemented)
	PssPeerTopicDefaultCapacity = 8 // limit of topics kept per peer. (not implemented)
	digestLength                = 32 // byte length of digest used for pss cache (currently same as swarm chunk hash)
	digestCapacity              = 256 // cache entry limit (not implement)
)

var (
	errorForwardToSelf = errors.New("forward to self")
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
	network.Overlay                                             // we can get the overlayaddress from this
	peerPool        map[pot.Address]map[Topic]p2p.MsgReadWriter // keep track of all virtual p2p.Peers we are currently speaking to
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
		Overlay:  k,
		peerPool: make(map[pot.Address]map[Topic]p2p.MsgReadWriter, PssPeerCapacity),
		fwdPool:  make(map[discover.NodeID]*protocols.Peer),
		handlers: make(map[Topic]map[*Handler]bool),
		fwdcache: make(map[pssDigest]pssCacheEntry),
		cachettl: params.Cachettl,
		dpa:      dpa,
		debug:    params.Debug,
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
	//addr := network.NewAddrFromNodeID(id)
	//potaddr := pot.NewHashAddressFromBytes(addr.OAddr)
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

// Links a handler function to a Topic
//
// After calling this, all incoming messages with an envelope Topic matching the Topic specified will be passed to the given Handler function.
//
// Returns a deregister function which needs to be called to deregister the handler,
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

// Entry point to processing a message for which the current node is the intended recipient.
func (self *Pss) Process(pssmsg *PssMsg) error {
	env := pssmsg.Payload
	payload := env.Payload
	handlers := self.getHandlers(env.Topic)
	if len(handlers) == 0 {
		return fmt.Errorf("No registered handler for topic '%x'", env.Topic)
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

// Sends a message using Pss.
//
// This method is payload agnostic, and will accept any arbitrary byte slice as the payload for a message.
//
// It generates an envelope for the specified recipient and topic, and wraps the message payload in it.
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
// Handlers that are merely passing on the PssMsg to its final recipient should call this directly
func (self *Pss) Forward(msg *PssMsg) error {

	if self.isSelfRecipient(msg) {
		return errorForwardToSelf
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
		sendMsg := fmt.Sprintf("TO %x FROM %x VIA %x", common.ByteLabel(msg.To), common.ByteLabel(self.BaseAddr()), common.ByteLabel(op.Address()))
		pp := self.fwdPool[sp.ID()]
		if self.checkFwdCache(op.Address(), digest) {
			log.Info("%v: peer already forwarded to", sendMsg)
			return true
		}
		err := pp.Send(msg)
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


// For devp2p protocol integration only.
//
// Convenience object for passing messages in and out of the p2p layer
type PssProtocol struct {
	*Pss
	proto *p2p.Protocol
	topic *Topic
	spec  *protocols.Spec
}

// For devp2p protocol integration only.
// 
// Maps a Topic to a devp2p protocol.
func RegisterPssProtocol(ps *Pss, topic *Topic, spec *protocols.Spec, targetprotocol *p2p.Protocol) *PssProtocol {
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

