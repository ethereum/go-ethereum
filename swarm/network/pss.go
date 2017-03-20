package network

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	DefaultTTL                  = 6000
	TopicLength                 = 32
	TopicResolverLength         = 8
	PssPeerCapacity             = 256
	PssPeerTopicDefaultCapacity = 8
	digestLength                = 64
	digestCapacity              = 256
	defaultDigestCacheTTL       = time.Second
)

var (
	errorNoForwarder   = errors.New("no available forwarders in routing table")
	errorForwardToSelf = errors.New("forward to self")
	errorBlockByCache  = errors.New("message found in blocking cache")
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
//
// Warning: do not access the To-member directly. Use *PssMsg.GetRecipient() and *PssMsg.SetRecipient() instead.
type PssMsg struct {
	// (we need the To-member exported for type inference)
	To      []byte
	Payload pssEnvelope
}

// Retrieve the remote peer receipient address of the message
func (self *PssMsg) GetRecipient() []byte {
	return self.To
}

// Set the remote peer recipient address of the message
func (self *PssMsg) SetRecipient(to []byte) {
	self.To = to
}

// String representation of PssMsg
func (self *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %x", common.ByteLabel(self.GetRecipient()))
}

// Pre-Whisper placeholder
type pssEnvelope struct {
	Topic       PssTopic
	TTL         uint16
	Payload     []byte
	SenderOAddr []byte
	SenderUAddr []byte
}

// Pre-Whisper placeholder
type pssPayload struct {
	Code       uint64
	Size       uint32
	Data       []byte
	ReceivedAt time.Time
}

// Pre-Whisper placeholder
type pssCacheEntry struct {
	expiresAt    time.Time
	receivedFrom []byte
}

// Topic defines the context of a message being transported over pss
// It is used by pss to determine what action is to be taken on an incoming message
// Typically, one can map protocol handlers for the message payloads by mapping topic to them; see *Pss.Register()
type PssTopic [TopicLength]byte

// Pre-Whisper placeholder
type pssDigest uint32

// pss provides sending messages to nodes without having to be directly connected to them.
//
// The messages are wrapped in a PssMsg structure and routed using the swarm kademlia routing.
// The structure is used by normal incoming message handlers on the nodes to determine which action to take, forward or process.
// Thus it is up to the implementer to write a handler, and link the PssMsg to this appropriate handler.
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
	Overlay // we can get the overlayaddress from this
	//peerPool map[pot.Address]map[PssTopic]*PssReadWriter // keep track of all virtual p2p.Peers we are currently speaking to
	peerPool map[pot.Address]map[PssTopic]p2p.MsgReadWriter     // keep track of all virtual p2p.Peers we are currently speaking to
	handlers map[PssTopic]func([]byte, *p2p.Peer, []byte) error // topic and version based pss payload handlers
	fwdcache map[pssDigest]pssCacheEntry                        // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	cachettl time.Duration                                      // how long to keep messages in fwdcache
	hasher   func(string) storage.Hasher                        // hasher to digest message to cache
	lock     sync.Mutex
}

func (self *Pss) hashMsg(msg *PssMsg) pssDigest {
	hasher := self.hasher("SHA3")()
	hasher.Reset()
	hasher.Write(msg.GetRecipient())
	hasher.Write(msg.Payload.SenderUAddr)
	hasher.Write(msg.Payload.SenderOAddr)
	hasher.Write(msg.Payload.Topic[:])
	hasher.Write(msg.Payload.Payload)
	b := hasher.Sum([]byte{})
	return pssDigest(binary.BigEndian.Uint32(b))
}

// Creates a new Pss instance. A node should only need one of these
//
// TODO error check overlay integrity
func NewPss(k Overlay, params *PssParams) *Pss {
	return &Pss{
		Overlay: k,
		//peerPool: make(map[pot.Address]map[PssTopic]*PssReadWriter, PssPeerCapacity),
		peerPool: make(map[pot.Address]map[PssTopic]p2p.MsgReadWriter, PssPeerCapacity),
		handlers: make(map[PssTopic]func([]byte, *p2p.Peer, []byte) error),
		fwdcache: make(map[pssDigest]pssCacheEntry),
		cachettl: params.Cachettl,
		hasher:   storage.MakeHashFunc,
	}
}

// enables to set address of node, to avoid backwards forwarding
//
// currently not in use as forwarder address is not known in the handler function hooked to the pss dispatcher.
// it is included as a courtesy to custom transport layers that may want to implement this
func (self *Pss) AddToCache(addr []byte, msg *PssMsg) error {
	digest := self.hashMsg(msg)
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

// Takes the generated PssTopic of a protocol, and links a handler function to it
// This allows the implementer to retrieve the right handler function (invoke the right protocol) for an incoming message by inspecting the topic on it.
func (self *Pss) Register(topic PssTopic, handler func(msg []byte, p *p2p.Peer, from []byte) error) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.handlers[topic] = handler
	return nil
}

// Retrieves the handler function registered by *Pss.Register()
func (self *Pss) GetHandler(topic PssTopic) func([]byte, *p2p.Peer, []byte) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.handlers[topic]
}

// Links a pss peer address and topic to a dedicated p2p.MsgReadWriter in the pss peerpool, and runs the specificed protocol on this p2p.MsgReadWriter and the specified peer
//
// The effect is that now we have a "virtual" protocol running on an artificial p2p.Peer, which can be looked up and piped to through Pss using swarm overlay address and topic
func (self *Pss) AddPeer(p *p2p.Peer, addr pot.Address, protocall adapters.ProtoCall, topic PssTopic, rw p2p.MsgReadWriter) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.addPeerTopic(addr, topic, rw)
	go func() {
		err := protocall(p, rw)
		log.Warn(fmt.Sprintf("pss vprotocol quit on addr %v topic %v: %v", addr, topic, err))
	}()
	return nil
}

// Removes a pss peer from the pss peerpool
func (self *Pss) RemovePeer(id pot.Address) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.peerPool[id] = nil
	return
}

func (self *Pss) addPeerTopic(id pot.Address, topic PssTopic, rw p2p.MsgReadWriter) error {
	if self.peerPool[id][topic] == nil {
		self.peerPool[id] = make(map[PssTopic]p2p.MsgReadWriter, PssPeerTopicDefaultCapacity)
	}
	self.peerPool[id][topic] = rw
	return nil
}

func (self *Pss) removePeerTopic(id pot.Address, topic PssTopic) {
	self.peerPool[id][topic] = nil
	return
}

func (self *Pss) isActive(id pot.Address, topic PssTopic) bool {
	if self.peerPool[id][topic] == nil {
		return false
	}
	return true
}

// Sends a message using pss. The message could be anything at all, and will be handled by whichever handler function is mapped to PssTopic using *Pss.Register()
//
// The to address is a swarm overlay address
func (self *Pss) Send(to []byte, topic PssTopic, msg []byte) error {

	pssenv := pssEnvelope{
		SenderOAddr: self.Overlay.GetAddr().OverlayAddr(),
		SenderUAddr: self.Overlay.GetAddr().UnderlayAddr(),
		Topic:       topic,
		TTL:         DefaultTTL,
		Payload:     msg,
	}

	pssmsg := &PssMsg{
		Payload: pssenv,
	}
	pssmsg.SetRecipient(to)

	return self.Forward(pssmsg)
}

// Forwards a pss message to the peer(s) closest to the to address
//
// Handlers that want to pass on a message should call this directly
func (self *Pss) Forward(msg *PssMsg) error {

	if self.isSelfRecipient(msg) {
		return errorForwardToSelf
	}

	digest := self.hashMsg(msg)

	if self.checkFwdCache(nil, digest) {
		log.Trace(fmt.Sprintf("pss relay block-cache match: FROM %x TO %x", common.ByteLabel(self.Overlay.GetAddr().OverlayAddr()), common.ByteLabel(msg.GetRecipient())))
		//return errorBlockByCache
		return nil
	}

	// TODO:check integrity of message

	sent := 0

	// send with kademlia
	// find the closest peer to the recipient and attempt to send
	self.Overlay.EachLivePeer(msg.GetRecipient(), 256, func(p Peer, po int, isproxbin bool) bool {
		if self.checkFwdCache(p.OverlayAddr(), digest) {
			log.Warn(fmt.Sprintf("BOUNCE DEFER PSS-relay FROM %x TO %x THRU %x:", common.ByteLabel(self.Overlay.GetAddr().OverlayAddr()), common.ByteLabel(msg.GetRecipient()), common.ByteLabel(p.OverlayAddr())))
			return true
		}
		log.Warn(fmt.Sprintf("Attempting PSS-relay FROM %x TO %x THRU %x", common.ByteLabel(self.Overlay.GetAddr().OverlayAddr()), common.ByteLabel(msg.GetRecipient()), common.ByteLabel(p.OverlayAddr())))
		err := p.Send(msg)
		if err != nil {
			log.Warn(fmt.Sprintf("FAILED PSS-relay FROM %x TO %x THRU %x: %v", common.ByteLabel(self.Overlay.GetAddr().OverlayAddr()), common.ByteLabel(msg.GetRecipient()), common.ByteLabel(p.OverlayAddr()), err))
			return true
		}
		sent++
		if bytes.Equal(msg.GetRecipient(), p.OverlayAddr()) || !isproxbin {
			return false
		}
		log.Trace(fmt.Sprintf("%x is in proxbin, so we continue sending", common.ByteLabel(p.OverlayAddr())))
		return true
	})
	if sent == 0 {
		log.Warn("PSS Was not able to send to any peers")
	} else {
		self.addFwdCacheExpire(digest)
	}

	return nil
}

// Convenience object that:
//
// - allows passing of the unwrapped PssMsg payload to the p2p level message handlers
// - interprets outgoing p2p.Msg from the p2p level to pass in to *Pss.Send()
//
// Implements p2p.MsgReadWriter
type PssReadWriter struct {
	*Pss
	RecipientOAddr pot.Address
	LastActive     time.Time
	rw             chan p2p.Msg
	ct             *protocols.CodeMap
	topic          *PssTopic
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
	ifc, found := prw.ct.GetInterface(msg.Code)
	if !found {
		return fmt.Errorf("Writemsg couldn't find matching interface for code %d", msg.Code)
	}
	msg.Decode(ifc)

	to := prw.RecipientOAddr.Bytes()

	pmsg, _ := makeMsg(msg.Code, ifc)

	return prw.Pss.Send(to, *prw.topic, pmsg)
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
	virtualProtocol *p2p.Protocol
	topic           *PssTopic
	ct              *protocols.CodeMap
}

// Constructor
func NewPssProtocol(pss *Pss, topic *PssTopic, ct *protocols.CodeMap, targetprotocol *p2p.Protocol) *PssProtocol {
	pp := &PssProtocol{
		Pss:             pss,
		virtualProtocol: targetprotocol,
		topic:           topic,
		ct:              ct,
	}
	return pp
}

// Retrieves a convenience method for passing an incoming message into the p2p layer
//
// If the implementer wishes to use the p2p.Protocol (or p2p/protocols) message handling, this handler can be directly registered as a handler for the PssMsg structure
func (self *PssProtocol) GetHandler() func([]byte, *p2p.Peer, []byte) error {
	return self.handle
}

func (self *PssProtocol) handle(msg []byte, p *p2p.Peer, senderAddr []byte) error {
	hashoaddr := pot.NewHashAddressFromBytes(senderAddr).Address
	if !self.isActive(hashoaddr, *self.topic) {
		rw := &PssReadWriter{
			Pss:            self.Pss,
			RecipientOAddr: hashoaddr,
			rw:             make(chan p2p.Msg),
			ct:             self.ct,
			topic:          self.topic,
		}
		self.Pss.AddPeer(p, hashoaddr, self.virtualProtocol.Run, *self.topic, rw)
	}

	payload := &pssPayload{}
	rlp.DecodeBytes(msg, payload)

	pmsg := p2p.Msg{
		Code:       payload.Code,
		Size:       uint32(len(payload.Data)),
		ReceivedAt: time.Now(),
		Payload:    bytes.NewBuffer(payload.Data),
	}

	vrw := self.Pss.peerPool[hashoaddr][*self.topic].(*PssReadWriter)
	vrw.injectMsg(pmsg)

	return nil
}

func (ps *Pss) isSelfRecipient(msg *PssMsg) bool {
	if bytes.Equal(msg.GetRecipient(), ps.Overlay.GetAddr().OverlayAddr()) {
		return true
	}
	return false
}

// Pre-Whisper placeholder
func makeMsg(code uint64, msg interface{}) ([]byte, error) {

	rlpdata, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return nil, err
	}

	// previous attempts corrupted nested structs in the payload iself upon deserializing
	// therefore we use two separate []byte fields instead of peerAddr
	// TODO verify that nested structs cannot be used in rlp
	smsg := &pssPayload{
		Code: code,
		Size: uint32(len(rlpdata)),
		Data: rlpdata,
	}

	rlpbundle, err := rlp.EncodeToBytes(smsg)
	if err != nil {
		return nil, err
	}

	return rlpbundle, nil
}

// Compiles a new PssTopic from a given name and version.
//
// Analogous to the name and version members of p2p.Protocol
func MakeTopic(s string, v int) (PssTopic, error) {
	t := [TopicLength]byte{}
	if len(s)+4 <= TopicLength {
		copy(t[4:len(s)+4], s)
	} else {
		return t, fmt.Errorf("topic '%t' too long", s)
	}
	binary.PutVarint(t[:4], int64(v))
	return t, nil
}
