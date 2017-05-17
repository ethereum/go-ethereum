package pss

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	DefaultTTL                  = 6000
	TopicLength                 = 32
	TopicResolverLength         = 8
	PssPeerCapacity             = 256
	PssPeerTopicDefaultCapacity = 8
	digestLength                = 32
	digestCapacity              = 256
	defaultDigestCacheTTL       = time.Second
)

var (
	errorForwardToSelf = errors.New("forward to self")
)

// Defines params for Pss
type PssParams struct {
	Cachettl time.Duration
}

// Initializes default params for Pss
func NewPssParams() *PssParams {
	return &PssParams{
		Cachettl: defaultDigestCacheTTL,
	}
}

// Encapsulates the message transported over pss.
type PssMsg struct {
	To      []byte
	Payload *PssEnvelope
}

// String representation of PssMsg
func (self *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %x", common.ByteLabel(self.To))
}

// Topic defines the context of a message being transported over pss
// It is used by pss to determine what action is to be taken on an incoming message
// Typically, one can map protocol handlers for the message payloads by mapping topic to them; see *Pss.Register()
type PssTopic [TopicLength]byte

func (self *PssTopic) String() string {
	return fmt.Sprintf("%x", self)
}

// Pre-Whisper placeholder, payload of PssMsg
type PssEnvelope struct {
	Topic       PssTopic
	TTL         uint16
	Payload     []byte
	From		[]byte
}

// creates Pss envelope from sender address, topic and raw payload
func NewPssEnvelope(addr []byte, topic PssTopic, payload []byte) *PssEnvelope {
	return &PssEnvelope{
		From: 		 addr,
		Topic:       topic,
		TTL:         DefaultTTL,
		Payload:     payload,
	}
}


func (msg *PssMsg) serialize() []byte {
	rlpdata, _ := rlp.EncodeToBytes(msg)
	/*buf := bytes.NewBuffer(nil)
	buf.Write(self.PssEnvelope.Topic[:])
	buf.Write(self.PssEnvelope.Payload)
	buf.Write(self.PssEnvelope.From)
	return buf.Bytes()*/
	return rlpdata
}


var pssTransportProtocol = &protocols.Spec{
	Name:       "pss",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		PssMsg{},
	},
}

// encapsulates a protocol msg as PssEnvelope data
type PssProtocolMsg struct {
	Code       uint64
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

type pssCacheEntry struct {
	expiresAt    time.Time
	receivedFrom []byte
}

type pssDigest [digestLength]byte

// Message handler func for a topic
type pssHandler func(msg []byte, p *p2p.Peer, from []byte) error

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
	network.Overlay                                                 // we can get the overlayaddress from this
	peerPool map[pot.Address]map[PssTopic]p2p.MsgReadWriter // keep track of all virtual p2p.Peers we are currently speaking to
	handlers map[PssTopic]map[*pssHandler]bool              // topic and version based pss payload handlers
	fwdcache map[pssDigest]pssCacheEntry                    // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	cachettl time.Duration                                  // how long to keep messages in fwdcache
	lock     sync.Mutex
	dpa		*storage.DPA
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

// Creates a new Pss instance. A node should only need one of these
//
// TODO: error check overlay integrity
func NewPss(k network.Overlay, dpa *storage.DPA, params *PssParams) *Pss {
	return &Pss{
		Overlay: k,
		peerPool: make(map[pot.Address]map[PssTopic]p2p.MsgReadWriter, PssPeerCapacity),
		handlers: make(map[PssTopic]map[*pssHandler]bool),
		fwdcache: make(map[pssDigest]pssCacheEntry),
		cachettl: params.Cachettl,
		dpa: dpa,
	}
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
			Name:    pssTransportProtocol.Name,
			Version: pssTransportProtocol.Version,
			Length:  pssTransportProtocol.Length(),
			Run:     self.Run,
		},
	}
}

func (self *Pss) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	pp := protocols.NewPeer(p, rw, pssTransportProtocol)
	return pp.Run(self.handlePssMsg)
}

func (self *Pss) APIs() []rpc.API {
	return []rpc.API{
		rpc.API {
			Namespace: "pss",
			Version:   "0.1",
			Service:   NewPssAPI(self),
			Public:    true,
		},
	}
}

// Takes the generated PssTopic of a protocol/chatroom etc, and links a handler function to it
// This allows the implementer to retrieve the right handler functions (invoke the right protocol)
// for an incoming message by inspecting the topic on it.
// a topic allows for multiple handlers
// returns a deregister function which needs to be called to deregister the handler
// (similar to event.Subscription.Unsubscribe())
func (self *Pss) Register(topic *PssTopic, handler pssHandler) func() {
	self.lock.Lock()
	defer self.lock.Unlock()
	handlers := self.handlers[*topic]
	if handlers == nil {
		handlers = make(map[*pssHandler]bool)
		self.handlers[*topic] = handlers
	}
	handlers[&handler] = true
	return func() { self.deregister(topic, &handler) }
}

func (self *Pss) deregister(topic *PssTopic, h *pssHandler) {
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
	//digest := self.hashMsg(msg)
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

func (self *Pss) getHandlers(topic PssTopic) map[*pssHandler]bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.handlers[topic]
}

//
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

// Sends a message using pss. The message could be anything at all, and will be handled by whichever handler function is mapped to PssTopic using *Pss.Register()
//
// The to address is a swarm overlay address
func (self *Pss) Send(to []byte, topic PssTopic, msg []byte) error {
	sender := self.Overlay.BaseAddr()
	pssenv := NewPssEnvelope(sender, topic, msg)
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
		p, ok := op.(network.Peer)
		if !ok {
			return true
		}
		addr := self.Overlay.BaseAddr()
		sendMsg := fmt.Sprintf("%x: msg to %x via %x", common.ByteLabel(addr), common.ByteLabel(msg.To), common.ByteLabel(p.Over()))
		if self.checkFwdCache(p.Over(), digest) {
			log.Info(fmt.Sprintf("%v: peer already forwarded to", sendMsg))
			return true
		}
		err := p.Send(msg)
		if err != nil {
			log.Warn(fmt.Sprintf("%v: failed forwarding: %v", sendMsg, err))
			return true
		}
		log.Trace(fmt.Sprintf("%v: successfully forwarded", sendMsg))
		sent++
		// if equality holds, p is always the first peer given in the iterator
		if bytes.Equal(msg.To, p.Over()) || !isproxbin {
			return false
		}
		log.Trace(fmt.Sprintf("%x is in proxbin, keep forwarding", common.ByteLabel(p.Over())))
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
func (self *Pss) AddPeer(p *p2p.Peer, addr pot.Address, run adapters.RunProtocol, topic PssTopic, rw p2p.MsgReadWriter) error {
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

func (self *Pss) addPeerTopic(id pot.Address, topic PssTopic, rw p2p.MsgReadWriter) error {
	if self.peerPool[id] == nil {
		self.peerPool[id] = make(map[PssTopic]p2p.MsgReadWriter, PssPeerTopicDefaultCapacity)
	}
	self.peerPool[id][topic] = rw
	return nil
}

func (self *Pss) removePeerTopic(rw p2p.MsgReadWriter, topic PssTopic) {
	prw, ok := rw.(*PssReadWriter)
	if !ok {
		return
	}
	delete(self.peerPool[prw.To], topic)
	if len(self.peerPool[prw.To]) == 0 {
		delete(self.peerPool, prw.To)
	}
}

func (self *Pss) isActive(id pot.Address, topic PssTopic) bool {
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
	spec		*protocols.Spec
	topic      *PssTopic
}

// Implements p2p.MsgReader
func (prw PssReadWriter) ReadMsg() (p2p.Msg, error) {
	msg := <-prw.rw
	log.Trace(fmt.Sprintf("pssrw readmsg: %v", msg))
	return msg, nil
}

// Implements p2p.MsgWriter
func (prw PssReadWriter) WriteMsg(msg p2p.Msg) error {
	log.Trace(fmt.Sprintf("pssrw writemsg: %v", msg))
	ifc, found := prw.spec.NewMsg(msg.Code)
	if !found {
		return fmt.Errorf("Writemsg couldn't find matching interface for code %d", msg.Code)
	}
	msg.Decode(ifc)

	pmsg, err := newProtocolMsg(msg.Code, ifc)
	if err != nil {
		return err
	}
	return prw.Pss.Send(prw.To.Bytes(), *prw.topic, pmsg)
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
	topic           *PssTopic
	spec			*protocols.Spec
}

// Constructor
//func RegisterPssProtocol(pss *Pss, topic *PssTopic, spec *protocols.Spec, targetprotocol *p2p.Protocol) *PssProtocol {
func RegisterPssProtocol(pss *Pss, topic *PssTopic, spec *protocols.Spec, targetprotocol *p2p.Protocol) error {
	pp := &PssProtocol{
		Pss:             pss,
		proto: targetprotocol,
		topic:           topic,
		spec:			 spec,
	}
	pss.Register(topic, pp.handle)
	//return pp
	return nil
}

func (self *PssProtocol) handle(msg []byte, p *p2p.Peer, senderAddr []byte) error {
	hashoaddr := pot.NewHashAddressFromBytes(senderAddr).Address
	if !self.isActive(hashoaddr, *self.topic) {
		rw := &PssReadWriter{
			Pss:   self.Pss,
			To:    hashoaddr,
			rw:    make(chan p2p.Msg),
			spec:	self.spec,
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

func (self *Pss) isSelfRecipient(msg *PssMsg) bool {
	return bytes.Equal(msg.To, self.Overlay.BaseAddr())
}

func newProtocolMsg(code uint64, msg interface{}) ([]byte, error) {

	rlpdata, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return nil, err
	}

	// previous attempts corrupted nested structs in the payload iself upon deserializing
	// therefore we use two separate []byte fields instead of peerAddr
	// TODO verify that nested structs cannot be used in rlp
	smsg := &PssProtocolMsg{
		Code: code,
		Size: uint32(len(rlpdata)),
		Payload: rlpdata,
	}

	return rlp.EncodeToBytes(smsg)
}

// constructs a new PssTopic from a given name and version.
//
// Analogous to the name and version members of p2p.Protocol
func NewTopic(s string, v int) (topic PssTopic) {
	h := sha3.NewKeccak256()
	h.Write([]byte(s))
	buf := make([]byte, TopicLength / 8)
	binary.PutUvarint(buf, uint64(v))
	h.Write(buf)
	copy(topic[:], h.Sum(buf)[:])
	return topic
}


func ToP2pMsg(msg []byte) (p2p.Msg, error) {
	payload := &PssProtocolMsg{}
	if err := rlp.DecodeBytes(msg, payload); err != nil {
		return p2p.Msg{}, fmt.Errorf("pss protocol handler unable to decode payload as p2p message: %v", err)
	}
	
	return p2p.Msg{
		Code:       payload.Code,
		Size:       uint32(len(payload.Payload)),
		ReceivedAt: time.Now(),
		Payload:    bytes.NewBuffer(payload.Payload),
	}, nil
}
