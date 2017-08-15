package pss

import (
	"bytes"
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

// TODO: proper padding generation for messages
// TODO: continue forwarding even through node is recipient (subvert traffic analysis attack)
const (
	PssPeerCapacity             = 256 // limit of peers kept in cache. (not implemented)
	PssPeerTopicDefaultCapacity = 8   // limit of topics kept per peer. (not implemented)
	digestLength                = 32  // byte length of digest used for pss cache (currently same as swarm chunk hash)
	digestCapacity              = 256 // cache entry limit (not implement)
	DefaultTTL                  = 6000
	defaultWhisperWorkTime      = 3
	defaultWhisperPoW           = 0.0000000001
	defaultSymKeyLength         = 32
)

// abstraction to enable access to p2p.protocols.Peer.Send
type senderPeer interface {
	ID() discover.NodeID
	Address() []byte
	Send(interface{}) error
}

// used to encapsulate symkey in asymmetric key exchange
type pssKeyMsg struct {
	From []byte
	Key  []byte
}

type pssPeer struct {
	rw            p2p.MsgReadWriter
	pubkey        *ecdsa.PublicKey
	recvsymkey    string
	sendsymkey    string
	symkeyexpires time.Time // symkeys should be renewed at this time
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
// Implements node.Service
type Pss struct {
	network.Overlay                                                              // we can get the overlayaddress from this
	peerPool                      map[pot.Address]map[whisper.TopicType]*pssPeer // keep track of all virtual p2p.Peers we are currently speaking to
	fwdPool                       map[discover.NodeID]*protocols.Peer            // keep track of all peers sitting on the pssmsg routing layer
	reverseSymKeyPool             map[string]pot.Address                         // reverse mapping of sentkeysymkeyids to peeraddr
	reversePubKeyPool             map[string]pot.Address                         // reverse mappling of sendkeysymkey to peeraddr
	handlers                      map[whisper.TopicType]map[*Handler]bool        // topic and version based pss payload handlers
	fwdcache                      map[pssDigest]pssCacheEntry                    // checksum of unique fields from pssmsg mapped to expiry, cache to determine whether to drop msg
	cachettl                      time.Duration                                  // how long to keep messages in fwdcache
	lock                          sync.Mutex
	dpa                           *storage.DPA
	privatekey                    *ecdsa.PrivateKey
	w                             *whisper.Whisper
	symkeycache                   []*pssPeer // fast lookup of recently used symkeys; last used is on top of stack
	symkeycachecursor             int        // modular cursor pointing to last used, wraps on symkeycache array
	recipientAddressLength        int        // this value will be used to truncate recipient addresses
	defaultRecipientAddressLength int        // the original value, use for revert after temporary change
}

func (self *Pss) String() string {
	return fmt.Sprintf("pss: addr %x, pubkey %v", self.BaseAddr(), common.ToHex(crypto.FromECDSAPub(&self.privatekey.PublicKey)))
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
		Overlay:                       k,
		peerPool:                      make(map[pot.Address]map[whisper.TopicType]*pssPeer, PssPeerCapacity),
		fwdPool:                       make(map[discover.NodeID]*protocols.Peer),
		reverseSymKeyPool:             make(map[string]pot.Address),
		reversePubKeyPool:             make(map[string]pot.Address),
		handlers:                      make(map[whisper.TopicType]map[*Handler]bool),
		fwdcache:                      make(map[pssDigest]pssCacheEntry),
		cachettl:                      params.Cachettl,
		dpa:                           dpa,
		privatekey:                    params.privatekey,
		w:                             whisper.New(),
		symkeycache:                   make([]*pssPeer, params.SymKeyCacheCapacity),
		recipientAddressLength:        params.RecipientAddressLength,
		defaultRecipientAddressLength: params.RecipientAddressLength,
	}
}

// Convenience accessor to the swarm overlay address of the pss node
func (self *Pss) BaseAddr() []byte {
	return self.Overlay.BaseAddr()
}

// Debug accessor for own public key
func (self *Pss) PublicKey() ecdsa.PublicKey {
	return self.privatekey.PublicKey
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
		rpc.API{
			Namespace: "psstest",
			Version:   "0.1",
			Service:   NewAPITest(self),
			Public:    false,
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

// Set the amount of bytes that will be disclosed encrypted of recipient address
//
// 0 will equal whisper routing (forward to all, no unencrypted partial address)
// -1 will return to default value
func (self *Pss) SetRecipientAddressLength(l int) {
	if l > 32 {
		l = 32
	} else if l < 0 {
		l = self.defaultRecipientAddressLength
	}
	self.recipientAddressLength = l
}

// Add a Public key address mapping
// this is needed to initiate handshakes
func (self *Pss) SetPeerPublicKey(addr pot.Address, topic whisper.TopicType, pubkey *ecdsa.PublicKey) {
	self.preparePeerTopic(addr, topic)
	self.lock.Lock()
	defer self.lock.Unlock()
	psp := self.peerPool[addr][topic]
	psp.pubkey = pubkey
	self.reversePubKeyPool[common.ToHex(crypto.FromECDSAPub(pubkey))] = addr
}

// Set the symmetric key for incoming communications
// this is either:
// - key sent when initiating a pss handshake to the other side
// - key sent as response to incoming handshake
func (self *Pss) GenerateIncomingSymmetricKey(addr pot.Address, topic whisper.TopicType) (string, error) {
	keyid, err := self.w.GenerateSymKey()
	if err != nil {
		return "", err
	}
	self.preparePeerTopic(addr, topic)
	self.lock.Lock()
	defer self.lock.Unlock()
	if _, ok := self.peerPool[addr]; !ok {
		return "", fmt.Errorf("no address entry %x in peerpool", addr)
	}
	psp := self.peerPool[addr][topic]
	psp.recvsymkey = keyid
	psp.symkeyexpires = time.Now().Add(time.Hour * 24 * 365)
	self.reverseSymKeyPool[keyid] = addr
	self.symkeycachecursor++
	self.symkeycache[self.symkeycachecursor%cap(self.symkeycache)] = psp
	return keyid, nil
}

// Set the symmetric key for outgoing communications
// this is the key received when receiving an incoming handshake
func (self *Pss) SetOutgoingSymmetricKey(addr pot.Address, topic whisper.TopicType, key []byte) (string, error) {
	keyid, err := self.w.AddSymKeyDirect(key)
	if err != nil {
		return "", err
	}
	self.preparePeerTopic(addr, topic)
	self.lock.Lock()
	defer self.lock.Unlock()
	psp := self.peerPool[addr][topic]
	psp.sendsymkey = keyid
	psp.symkeyexpires = time.Now().Add(time.Hour * 24 * 365)
	return keyid, nil
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

func (self *Pss) addFwdCache(digest pssDigest) error {
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

// protocol handler for incoming pss msg
// check if address partially matches = CAN be for us
func (self *Pss) handlePssMsg(msg interface{}) error {
	pssmsg, ok := msg.(*PssMsg)
	if ok {
		if !self.isSelfPossibleRecipient(pssmsg) {
			log.Trace("pss was for someone else :'( ... forwarding")
			return self.Forward(pssmsg)
		}
		log.Trace("pss for us, yay! ... let's process!")

		return self.Process(pssmsg)
	}

	return fmt.Errorf("invalid message type. Expected *PssMsg, got %T ", msg)
}

// Entry point to processing a message for which the current node can be the intended recipient.
func (self *Pss) Process(pssmsg *PssMsg) error {
	var recvmsg *whisper.ReceivedMessage
	var from []byte
	var err error
	envelope := pssmsg.Payload

	// save process cycles if the messages have no internal handler
	handlers := self.getHandlers(envelope.Topic)
	if len(handlers) == 0 {
		return fmt.Errorf("No registered handler for topic '%x'", envelope.Topic)
	}

	if len(envelope.AESNonce) > 0 { // see whisper.envelope.go.OpenSymmetric
		recvmsg, from, err = self.processSym(envelope)
		if err != nil {
			return err
		}
	} else {
		recvmsg, from, err = self.processAsym(envelope)
		if err != nil {
			return err
		}
		keymsgraw := recvmsg.Payload
		keymsg := &pssKeyMsg{}
		err = rlp.DecodeBytes(keymsgraw, keymsg)
		if err == nil {
			var potfrom pot.Address
			log.Trace("have symkeymsg", "from", keymsg.From)
			//copy(from[:], keymsg.From)
			copy(potfrom[:], keymsg.From)
			// TODO: need to handle / check for expired keys also here
			_, err = self.SetOutgoingSymmetricKey(potfrom, envelope.Topic, keymsg.Key)
			if err != nil {
				return fmt.Errorf("received invalid symkey in pss handshake for peer %x topic %x", keymsg.From, envelope.Topic)
			}
			// if we by now don't have keys for both in- and outgoing msgs, we need to make one for incoming and send it to the other party
			// along with an encrypted secret so that it can tell that we received its key
			// the encrypted secret will be our key encrypted with its key
			if !self.isSecured(potfrom, envelope.Topic) {
				_, err := self.sendKey(keymsg.From, &envelope.Topic)
				return err
			}
			// if it's a keymsg we don't want to pass it on to the handler
			return nil
		}
	}

	// this condition checks if we either have a successfully decrypted asym msg that's not a pssKeyMsg
	// OR a successfully decrypted sym msg
	// if so we know for sure it's for this pss node
	if recvmsg != nil {
		handlers := self.getHandlers(envelope.Topic)
		nid, _ := discover.HexID("0x00")
		p := p2p.NewPeer(nid, fmt.Sprintf("%x", from), []p2p.Cap{})
		for f := range handlers {
			err := (*f)(recvmsg.Payload, p, from)
			if err != nil {
				log.Warn("Pss handler %p failed: %v", err)
			}
		}
	}

	return nil
}

// attempt to decrypt, validate and unpack sym msg
func (self *Pss) processSym(envelope *whisper.Envelope) (recvmsg *whisper.ReceivedMessage, frombytes []byte, err error) {
	for i := self.symkeycachecursor; i > self.symkeycachecursor-cap(self.symkeycache) && i > 0; i-- {
		symkeyid := self.symkeycache[i%cap(self.symkeycache)].recvsymkey
		log.Trace("attempting symmetric decrypt", "symkey", symkeyid)
		symkey, err := self.w.GetSymKey(symkeyid)
		if err != nil {
			log.Debug("could not retrieve whisper symkey id %v: %v", symkeyid, err)
			continue
		}
		recvmsg, err = envelope.OpenSymmetric(symkey)
		if err != nil {
			log.Trace("sym decrypt failed", "symkey", symkeyid, "err", err)
			continue
		}
		if !recvmsg.Validate() {
			return nil, nil, fmt.Errorf("symmetrically encrypted message has invalid signature or is corrupt")
		}
		from := self.reverseSymKeyPool[symkeyid]
		self.symkeycachecursor++
		self.symkeycache[self.symkeycachecursor%cap(self.symkeycache)] = self.peerPool[from][envelope.Topic]
		log.Debug("successfully decrypted symmetrically encrypted pss message", "symkeys tried", i, "from", from, "symkey cache insert", self.symkeycachecursor%cap(self.symkeycache))
		return recvmsg, from.Bytes(), nil
	}
	return nil, nil, nil
}

// attempt to decrypt, validate and unpack asym msg
func (self *Pss) processAsym(envelope *whisper.Envelope) (recvmsg *whisper.ReceivedMessage, from []byte, err error) {
	recvmsg, err = envelope.OpenAsymmetric(self.privatekey)
	if err != nil {
		return nil, nil, fmt.Errorf("asym default decrypt of pss msg failed: %v", "err", err)
	}
	// check signature (if signed), strip padding
	if !recvmsg.Validate() {
		return nil, nil, fmt.Errorf("invalid message")
	}
	pubkeyhex := common.ToHex(crypto.FromECDSAPub(recvmsg.Src))
	log.Debug("successfully decrypted asymmetrically encrypted pss message", "from", from, "pubkey", pubkeyhex)
	from = self.reversePubKeyPool[pubkeyhex].Bytes()
	return recvmsg, from, nil
}

// generate and send symkey to peer using asym send (handshake)
func (self *Pss) sendKey(to []byte, topic *whisper.TopicType) (string, error) {
	log.Trace("sending our symkey", "to", to)
	var potaddr pot.Address
	copy(potaddr[:], to)
	recvkeyid, err := self.GenerateIncomingSymmetricKey(potaddr, *topic)
	if err != nil {
		return "", fmt.Errorf("set receive symkey fail (peer %x topic %x): %v", to, topic, err)
	}
	recvkey, err := self.w.GetSymKey(recvkeyid)
	if err != nil {
		return "", fmt.Errorf("get generated outgoing symkey fail (peer %x topic %x): %v", to, topic, err)
	}
	recvkeymsg := &pssKeyMsg{
		From: self.BaseAddr(),
		Key:  recvkey,
	}
	recvkeybytes, err := rlp.EncodeToBytes(recvkeymsg)
	if err != nil {
		return "", fmt.Errorf("rlp keymsg encode fail: %v", err)
	}
	// if the send fails it means this public key is not registered for this particular address AND topic
	err = self.SendAsym(to, *topic, recvkeybytes)
	log.Debug("recvkeybytes", "bytes", recvkeybytes, "recvkey", recvkey)
	if err != nil {
		return "", fmt.Errorf("Send symkey failed: %v", err)
	}
	return recvkeyid, nil
}

// Prepares a msg for sending with symmetric encryption
//
// this can only succeed if there exist unexpired symmetric keys both for incoming and outgoing traffic. This will be the state after a asymmetric exchange of symmetric keys (handshake)
func (self *Pss) SendSym(to []byte, topic whisper.TopicType, msg []byte) error {
	var potaddr pot.Address
	copy(potaddr[:], to)
	// isSecured also checks whether the first dimension of the map is populated
	if !self.isSecured(potaddr, topic) {
		return fmt.Errorf("missing complete handshake")
	}
	psp := self.peerPool[potaddr][topic]
	symkey, err := self.w.GetSymKey(psp.sendsymkey)
	if err != nil {
		return fmt.Errorf("missing valid symkey %s: %v", psp.sendsymkey, err)
	}
	return self.send(to, topic, msg, nil, symkey)
}

// Prepares a msg for sending with asymmetric encryption
//
// Asymmetric send can be used to exchange symmetric keys (handshake)
func (self *Pss) SendAsym(to []byte, topic whisper.TopicType, msg []byte) error {
	var potaddr pot.Address
	copy(potaddr[:], to)
	topicmap := self.peerPool[potaddr]
	if topicmap == nil {
		return fmt.Errorf("No public key for address %x", to)
	}
	psp := self.peerPool[potaddr][topic]
	topubkey := psp.pubkey
	return self.send(to, topic, msg, topubkey, nil)
}

// Sends a pss message
//
// The method itself is payload agnostic, and will accept any arbitrary byte slice as the payload for a message.//
// It generates an whisper envelope for the specified recipient and topic, and wraps the message payload in it.
func (self *Pss) send(to []byte, topic whisper.TopicType, msg []byte, pubkey *ecdsa.PublicKey, symkey []byte) error {
	if pubkey != nil && symkey != nil {
		return fmt.Errorf("Can only specify one of symkey and pubkey for send")
	} else if pubkey == nil && symkey == nil {
		return fmt.Errorf("No keys set, refusing to send unencrypted pss")
	}
	wparams := &whisper.MessageParams{
		TTL:      DefaultTTL,
		Src:      self.privatekey,
		Dst:      pubkey, // if this is set, the message is considered asymmetric, if not it is considered symmetric
		KeySym:   symkey,
		Topic:    topic,
		WorkTime: defaultWhisperWorkTime,
		PoW:      defaultWhisperPoW,
		Payload:  msg,
		Padding:  []byte("1234567890abcdef"),
	}
	// set up outgoing message container, which does encryption and envelope wrapping
	woutmsg, err := whisper.NewSentMessage(wparams)
	if err != nil {
		return fmt.Errorf("failed to generate whisper message encapsulation: %v", err)
	}
	// performs encryption and PoW
	// after this the message is ready for sending
	envelope, err := woutmsg.Wrap(wparams)
	if err != nil {
		return fmt.Errorf("failed to perform whisper encryption: %v", err)
	}
	log.Trace("pssmsg whisper done", "env", envelope, "wparams payloda", wparams.Payload)
	// prepare for devp2p transport
	pssmsg := &PssMsg{
		To:      to,
		Payload: envelope,
	}
	return self.Forward(pssmsg)
}

// Forwards a pss message to the peer(s) closest to the to recipient address in the PssMsg struct
//
// The To-field in the PssMsg struct will be plaintext, unencrypted, and will be truncated to the magnitude set by SetRecipientAddressLength (PssParams.RecipientAddressLength by default).
// If the recipientAddressLength value is 0, no bytes will be disclosed, and message will be forwarded to ALL devp2p peers
//
// Handlers that are merely passing on the PssMsg to its final recipient might call this directly
func (self *Pss) Forward(msg *PssMsg) error {
	to := msg.To
	zeros := make([]byte, len(to)-int(self.recipientAddressLength))
	// truncate part of the pssmsg recipient address
	// this way noone can know for certain exactly who it's for
	// the kademlia address will be same length
	// but equivalent to truncated part will be overwritten by zeros
	copy(to[self.recipientAddressLength:len(to)], zeros)
	log.Trace("truncated recipient", "original", msg.To, "truncated", to)
	msg.To = msg.To[:self.recipientAddressLength]

	// cache the message
	digest, err := self.storeMsg(msg)
	if err != nil {
		log.Warn(fmt.Sprintf("could not store message %v to cache: %v", msg, err))
	}

	// flood guard:
	// don't allow identical messages we saw shortly before
	if self.checkFwdCache(nil, digest) {
		log.Trace(fmt.Sprintf("pss relay block-cache match: FROM %x TO %x", common.ByteLabel(self.Overlay.BaseAddr()), common.ByteLabel(msg.To)))
		return nil
	}

	// send with kademlia
	// find the closest peer to the recipient and attempt to send
	sent := 0

	self.Overlay.EachConn(to, 256, func(op network.OverlayConn, po int, isproxbin bool) bool {
		sp, ok := op.(senderPeer)
		if !ok {
			log.Crit("Pss cannot use kademlia peer type")
			return false
		}
		sendMsg := fmt.Sprintf("MSG %x TO %x FROM %x VIA %x", digest, common.ByteLabel(to), common.ByteLabel(self.BaseAddr()), common.ByteLabel(op.Address()))
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
		return fmt.Errorf("unable to forward to any peers")
	}

	self.addFwdCache(digest)
	return nil
}

// For devp2p protocol integration only. Analogous to an outgoing devp2p connection.
//
// Links a remote peer and Topic to a dedicated p2p.MsgReadWriter in the pss peerpool, and runs the specificed protocol using these resources.
//
// The effect is that now we have a "virtual" protocol running on an artificial p2p.Peer, which can be looked up and piped to through Pss using swarm overlay address and topic
//
// The peer's encryption keys must be added separately.
func (self *Pss) AddPeer(p *p2p.Peer, addr pot.Address, run func(*p2p.Peer, p2p.MsgReadWriter) error, topic whisper.TopicType, rw p2p.MsgReadWriter) error {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.preparePeerTopic(addr, topic)
	psp := self.peerPool[addr][topic]
	psp.rw = rw
	go func() {
		err := run(p, rw)
		log.Warn(fmt.Sprintf("pss vprotocol quit on addr %v topic %v: %v", addr, topic, err))
		self.removePeerTopic(rw, topic)
	}()
	return nil
}

func (self *Pss) preparePeerTopic(id pot.Address, topic whisper.TopicType) bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.peerPool[id] == nil {
		self.peerPool[id] = make(map[whisper.TopicType]*pssPeer, PssPeerTopicDefaultCapacity)
	}
	if self.peerPool[id][topic] != nil {
		return false
	}
	self.peerPool[id][topic] = &pssPeer{}
	return true
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

func (self *Pss) isSelfPossibleRecipient(msg *PssMsg) bool {
	local := self.Overlay.BaseAddr()
	return bytes.Equal(msg.To[:], local[:len(msg.To)])
}

func (self *Pss) isActive(id pot.Address, topic whisper.TopicType) bool {
	if self.peerPool[id] == nil {
		return false
	}
	return self.peerPool[id][topic].rw != nil
}

// todo: maybe not enough to check that the symkey id strings are empty
func (self *Pss) isSecured(id pot.Address, topic whisper.TopicType) bool {
	if _, ok := self.peerPool[id]; !ok {
		return false
	}
	if _, ok := self.peerPool[id][topic]; !ok {
		return false
	}
	if self.peerPool[id][topic].symkeyexpires.Before(time.Now()) {
		return false
	}
	if self.peerPool[id][topic].recvsymkey == "" || self.peerPool[id][topic].sendsymkey == "" {
		return false
	}
	return true
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
	return prw.SendSym(prw.To.Bytes(), *prw.topic, pmsg)
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
// BROKEN! Implementation for pubkey lookup must be implemented
//
// Generic handler for initiating devp2p-like protocol connections
//
// This handler should be passed to Pss.Register with the associated topic.
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

	vrw := self.Pss.peerPool[hashoaddr][*self.topic].rw.(*PssReadWriter)
	vrw.injectMsg(pmsg)

	return nil
}
