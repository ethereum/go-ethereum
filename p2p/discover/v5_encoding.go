// Copyright 2019 The go-ethereum Authors
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

package discover

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/hkdf"
)

// TODO concurrent WHOAREYOU tie-breaker
// TODO deal with WHOAREYOU amplification factor (min packet size?)
// TODO add counter to nonce
// TODO rehandshake after X packets

// Discovery v5 packet types.
const (
	p_pingV5 byte = iota + 1
	p_pongV5
	p_findnodeV5
	p_nodesV5
	p_talkreqV5
	p_talkrespV5
	p_requestTicketV5
	p_ticketV5
	p_regtopicV5
	p_regconfirmationV5
	p_topicqueryV5
	p_unknownV5   = byte(255) // any non-decryptable packet
	p_whoareyouV5 = byte(254) // the WHOAREYOU packet
)

// Discovery v5 packet structures.
type (
	packetHeaderV5 struct {
		ProtocolID [8]byte
		SrcID      enode.ID
		Flags      byte
		AuthSize   uint16
	}

	whoareyouAuthDataV5 struct {
		Nonce     [gcmNonceSize]byte // nonce of request packet
		IDNonce   [32]byte           // ID proof data
		RecordSeq uint64             // highest known ENR sequence of requester
	}

	handshakeAuthDataV5 struct {
		h struct {
			Version    uint8              // protocol version
			Nonce      [gcmNonceSize]byte // AES-GCM nonce of message
			SigSize    byte               // ignature data
			PubkeySize byte               // offset of
		}
		// Trailing variable-size data.
		signature, pubkey, record []byte
	}

	messageAuthDataV5 struct {
		Nonce [gcmNonceSize]byte // AES-GCM nonce of message
	}
)

// Packet header flag values.
const (
	flagMessage   = 0
	flagWhoareyou = 1
	flagHandshake = 2
)

var (
	sizeofPacketHeaderV5      = binary.Size(packetHeaderV5{})
	sizeofWhoareyouAuthDataV5 = binary.Size(whoareyouAuthDataV5{})
	sizeofHandshakeAuthDataV5 = binary.Size(handshakeAuthDataV5{}.h)
	sizeofMessageAuthDataV5   = binary.Size(messageAuthDataV5{})
	protocolIDV5              = [8]byte{'d', 'i', 's', 'c', 'v', '5', ' ', ' '}
)

// Discovery v5 messages.
type (
	// unknownV5 represents any packet that can't be decrypted.
	unknownV5 struct {
		AuthTag []byte
	}

	// WHOAREYOU contains the handshake challenge.
	whoareyouV5 struct {
		AuthTag   []byte
		IDNonce   [32]byte // To be signed by recipient.
		RecordSeq uint64   // ENR sequence number of recipient

		node *enode.Node
		sent mclock.AbsTime
	}

	// PING is sent during liveness checks.
	pingV5 struct {
		ReqID  []byte
		ENRSeq uint64
	}

	// PONG is the reply to PING.
	pongV5 struct {
		ReqID  []byte
		ENRSeq uint64
		ToIP   net.IP // These fields should mirror the UDP envelope address of the ping
		ToPort uint16 // packet, which provides a way to discover the the external address (after NAT).
	}

	// FINDNODE is a query for nodes in the given bucket.
	findnodeV5 struct {
		ReqID     []byte
		Distances []uint
	}

	// NODES is the reply to FINDNODE and TOPICQUERY.
	nodesV5 struct {
		ReqID []byte
		Total uint8
		Nodes []*enr.Record
	}

	// TALKREQ is an application-level request.
	talkreqV5 struct {
		ReqID    []byte
		Protocol string
		Message  []byte
	}

	// TALKRESP is the reply to TALKREQ.
	talkrespV5 struct {
		ReqID   []byte
		Message []byte
	}

	// REQUESTTICKET requests a ticket for a topic queue.
	requestTicketV5 struct {
		ReqID []byte
		Topic []byte
	}

	// TICKET is the response to REQUESTTICKET.
	ticketV5 struct {
		ReqID  []byte
		Ticket []byte
	}

	// REGTOPIC registers the sender in a topic queue using a ticket.
	regtopicV5 struct {
		ReqID  []byte
		Ticket []byte
		ENR    *enr.Record
	}

	// REGCONFIRMATION is the reply to REGTOPIC.
	regconfirmationV5 struct {
		ReqID      []byte
		Registered bool
	}

	// TOPICQUERY asks for nodes with the given topic.
	topicqueryV5 struct {
		ReqID []byte
		Topic []byte
	}
)

const (
	// Encryption/authentication parameters.
	aesKeySize       = 16
	gcmNonceSize     = 12
	idNoncePrefix    = "discovery-id-nonce"
	handshakeTimeout = time.Second

	// Protocol constants.
	handshakeVersion = 1
	minVersion       = 1
)

var (
	errTooShort               = errors.New("packet too short")
	errInvalidHeader          = errors.New("invalid packet header")
	errUnexpectedHandshake    = errors.New("unexpected auth response, not in handshake")
	errHandshakeNonceMismatch = errors.New("wrong nonce in auth response")
	errInvalidAuthKey         = errors.New("invalid ephemeral pubkey")
	errUnknownAuthScheme      = errors.New("unknown auth scheme in handshake")
	errNoRecord               = errors.New("expected ENR in handshake but none sent")
	errInvalidNonceSig        = errors.New("invalid ID nonce signature")
	errMessageTooShort        = errors.New("message contains no data")
	errMessageDecrypt         = errors.New("cannot decrypt message")
)

// wireCodec encodes and decodes discovery v5 packets.
type wireCodec struct {
	sha256    hash.Hash
	localnode *enode.LocalNode
	privkey   *ecdsa.PrivateKey
	buf       bytes.Buffer // used for encoding of packets
	msgbuf    bytes.Buffer // used for encoding of message content
	reader    bytes.Reader // used for decoding
	sc        *sessionCache
}

type handshakeSecrets struct {
	writeKey, readKey []byte
}

// newWireCodec creates a wire codec.
func newWireCodec(ln *enode.LocalNode, key *ecdsa.PrivateKey, clock mclock.Clock) *wireCodec {
	c := &wireCodec{
		sha256:    sha256.New(),
		localnode: ln,
		privkey:   key,
		sc:        newSessionCache(1024, clock),
	}
	return c
}

// encode encodes a packet to a node. 'id' and 'addr' specify the destination node. The
// 'challenge' parameter should be the most recently received WHOAREYOU packet from that
// node.
func (c *wireCodec) encode(id enode.ID, addr string, packet packetV5, challenge *whoareyouV5) ([]byte, []byte, error) {
	if packet.kind() == p_whoareyouV5 {
		p := packet.(*whoareyouV5)
		enc, err := c.encodeWhoareyou(id, p)
		if err == nil {
			c.sc.storeSentHandshake(id, addr, p)
		}
		return enc, nil, err
	}

	if challenge != nil {
		return c.encodeHandshakeMessage(id, addr, packet, challenge)
	}
	if key := c.sc.writeKey(id, addr); key != nil {
		return c.encodeMessage(id, addr, packet, key)
	}
	// No keys, no handshake: send random data to kick off the handshake.
	return c.encodeRandom(id)
}

// makeHeader creates a packet header.
func (c *wireCodec) makeHeader(toID enode.ID, flags byte, authsizeExtra int) *packetHeaderV5 {
	var authsize int
	switch flags {
	case flagMessage:
		authsize = sizeofMessageAuthDataV5
	case flagWhoareyou:
		authsize = sizeofWhoareyouAuthDataV5
	case flagHandshake:
		authsize = sizeofHandshakeAuthDataV5
	default:
		panic(fmt.Errorf("BUG: invalid packet header flags %x", flags))
	}
	authsize += authsizeExtra
	if authsize > int(^uint16(0)) {
		panic(fmt.Errorf("BUG: auth size %d overflows uint16", authsize))
	}
	return &packetHeaderV5{
		ProtocolID: protocolIDV5,
		SrcID:      c.localnode.ID(),
		Flags:      flags,
		AuthSize:   uint16(authsize),
	}
}

// encodeRandom encodes a packet with random content.
func (c *wireCodec) encodeRandom(toID enode.ID) ([]byte, []byte, error) {
	var auth messageAuthDataV5
	if _, err := crand.Read(auth.Nonce[:]); err != nil {
		return nil, nil, fmt.Errorf("can't get random data: %v", err)
	}

	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, c.makeHeader(toID, flagMessage, 0))
	binary.Write(&c.buf, binary.BigEndian, &auth)
	output := maskOutputPacket(toID, c.buf.Bytes(), c.buf.Len())
	return output, auth.Nonce[:], nil
}

// encodeWhoareyou encodes a WHOAREYOU packet.
func (c *wireCodec) encodeWhoareyou(toID enode.ID, packet *whoareyouV5) ([]byte, error) {
	// Sanity check node field to catch misbehaving callers.
	if packet.RecordSeq > 0 && packet.node == nil {
		panic("BUG: missing node in whoareyouV5 with non-zero seq")
	}
	auth := &whoareyouAuthDataV5{
		IDNonce:   packet.IDNonce,
		RecordSeq: packet.RecordSeq,
	}
	copy(auth.Nonce[:], packet.AuthTag)
	head := c.makeHeader(toID, flagWhoareyou, 0)

	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, head)
	binary.Write(&c.buf, binary.BigEndian, auth)
	output := maskOutputPacket(toID, c.buf.Bytes(), c.buf.Len())
	return output, nil
}

// encodeHandshakeMessage encodes an encrypted message with a handshake
// response header.
func (c *wireCodec) encodeHandshakeMessage(toID enode.ID, addr string, packet packetV5, challenge *whoareyouV5) ([]byte, []byte, error) {
	// Ensure calling code sets challenge.node.
	if challenge.node == nil {
		panic("BUG: missing challenge.node in encode")
	}

	// Generate new secrets.
	auth, sec, err := c.makeHandshakeHeader(toID, addr, challenge)
	if err != nil {
		return nil, nil, err
	}

	// TODO: this should happen when the first authenticated message is received
	c.sc.storeNewSession(toID, addr, sec.readKey, sec.writeKey)

	// Encode header and auth header.
	var (
		authsizeExtra = len(auth.pubkey) + len(auth.signature) + len(auth.record)
		head          = c.makeHeader(toID, flagHandshake, authsizeExtra)
	)
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, head)
	binary.Write(&c.buf, binary.BigEndian, &auth.h)
	c.buf.Write(auth.signature)
	c.buf.Write(auth.pubkey)
	c.buf.Write(auth.record)
	output := c.buf.Bytes()

	// Encrypt packet body.
	c.msgbuf.Reset()
	c.msgbuf.WriteByte(packet.kind())
	if err := rlp.Encode(&c.msgbuf, packet); err != nil {
		return nil, nil, err
	}
	messagePT := c.msgbuf.Bytes()
	headerData := output
	output, err = encryptGCM(output, sec.writeKey, auth.h.Nonce[:], messagePT, headerData)
	if err == nil {
		output = maskOutputPacket(toID, output, len(headerData))
	}
	return output, auth.h.Nonce[:], err
}

// encodeAuthHeader creates the auth header on a call packet following WHOAREYOU.
func (c *wireCodec) makeHandshakeHeader(toID enode.ID, addr string, challenge *whoareyouV5) (*handshakeAuthDataV5, *handshakeSecrets, error) {
	auth := new(handshakeAuthDataV5)
	auth.h.Version = handshakeVersion
	auth.h.Nonce = c.sc.nextNonce(toID, addr)

	// Create the ephemeral key. This needs to be first because the
	// key is part of the ID nonce signature.
	var remotePubkey = new(ecdsa.PublicKey)
	if err := challenge.node.Load((*enode.Secp256k1)(remotePubkey)); err != nil {
		return nil, nil, fmt.Errorf("can't find secp256k1 key for recipient")
	}
	ephkey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("can't generate ephemeral key")
	}
	ephpubkey := encodePubkey(&ephkey.PublicKey)
	auth.pubkey = ephpubkey[:]
	auth.h.PubkeySize = byte(len(auth.pubkey))

	// Add ID nonce signature to response.
	idsig, err := c.signIDNonce(challenge.IDNonce[:], ephpubkey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("can't sign: %v", err)
	}
	auth.signature = idsig
	auth.h.SigSize = byte(len(auth.signature))

	// Add our record to response if it's newer than what remote
	// side has.
	ln := c.localnode.Node()
	if challenge.RecordSeq < ln.Seq() {
		auth.record, _ = rlp.EncodeToBytes(ln.Record())
	}

	// Create session keys.
	sec := c.deriveKeys(c.localnode.ID(), challenge.node.ID(), ephkey, remotePubkey, challenge)
	if sec == nil {
		return nil, nil, fmt.Errorf("key derivation failed")
	}
	return auth, sec, err
}

// encodeMessage encodes an encrypted message packet.
func (c *wireCodec) encodeMessage(toID enode.ID, addr string, packet packetV5, writeKey []byte) (enc []byte, authTag []byte, err error) {
	var (
		auth messageAuthDataV5
		head = c.makeHeader(toID, flagMessage, 0)
	)
	auth.Nonce = c.sc.nextNonce(toID, addr)
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, head)
	binary.Write(&c.buf, binary.BigEndian, &auth)
	output := c.buf.Bytes()

	// Encode the message plaintext.
	c.msgbuf.Reset()
	c.msgbuf.WriteByte(packet.kind())
	if err := rlp.Encode(&c.msgbuf, packet); err != nil {
		return nil, nil, err
	}
	messagePT := c.msgbuf.Bytes()

	// Encrypt message data.
	headerData := output
	output, err = encryptGCM(output, writeKey, auth.Nonce[:], messagePT, headerData)
	if err == nil {
		output = maskOutputPacket(toID, output, len(headerData))
	}
	return output, auth.Nonce[:], err
}

// decode decodes a discovery packet.
func (c *wireCodec) decode(input []byte, addr string) (src enode.ID, n *enode.Node, p packetV5, err error) {
	// Delete timed-out handshakes. This must happen before decoding to avoid
	// processing the same handshake twice.
	c.sc.handshakeGC()

	// Unmask the header.
	if len(input) < sizeofPacketHeaderV5+maskIVSize {
		return enode.ID{}, nil, nil, errTooShort
	}
	mask := headerMask(c.localnode.ID(), input)
	input = input[maskIVSize:]
	headerData := input[:sizeofPacketHeaderV5]
	mask.XORKeyStream(headerData, headerData)

	// Decode and verify the header.
	var head packetHeaderV5
	c.reader.Reset(input)
	binary.Read(&c.reader, binary.BigEndian, &head)
	if head.ProtocolID != protocolIDV5 {
		return enode.ID{}, nil, nil, errInvalidHeader
	}
	if int(head.AuthSize) > c.reader.Len() {
		return enode.ID{}, nil, nil, errInvalidHeader
	}
	src = head.SrcID

	// Unmask auth data.
	authData := input[sizeofPacketHeaderV5 : sizeofPacketHeaderV5+int(head.AuthSize)]
	mask.XORKeyStream(authData, authData)

	// Decode auth part and message.
	switch {
	case head.Flags&flagWhoareyou != 0:
		p, err = c.decodeWhoareyou(&head)
	case head.Flags&flagHandshake != 0:
		n, p, err = c.decodeHandshakeMessage(addr, &head, input)
	default:
		p, err = c.decodeMessage(addr, &head, input)
	}
	return src, n, p, err
}

// decodeWhoareyou reads packet data after the header as a WHOAREYOU packet.
func (c *wireCodec) decodeWhoareyou(head *packetHeaderV5) (packetV5, error) {
	if c.reader.Len() < sizeofWhoareyouAuthDataV5 {
		return nil, errTooShort
	}
	if int(head.AuthSize) != sizeofWhoareyouAuthDataV5 {
		return nil, fmt.Errorf("invalid auth size for whoareyou")
	}
	auth := new(whoareyouAuthDataV5)
	binary.Read(&c.reader, binary.BigEndian, auth)
	p := &whoareyouV5{
		AuthTag:   auth.Nonce[:],
		IDNonce:   auth.IDNonce,
		RecordSeq: auth.RecordSeq,
	}
	return p, nil
}

func (c *wireCodec) decodeHandshakeMessage(fromAddr string, head *packetHeaderV5, input []byte) (n *enode.Node, p packetV5, err error) {
	node, nonce, sec, err := c.decodeHandshake(fromAddr, head)
	if err != nil {
		return nil, nil, err
	}

	// Handshake OK, drop the challenge and store the new session keys.
	sec.readKey, sec.writeKey = sec.writeKey, sec.readKey
	c.sc.storeNewSession(head.SrcID, fromAddr, sec.readKey, sec.writeKey)
	c.sc.deleteHandshake(head.SrcID, fromAddr)

	// Decrypt the message using the new session keys.
	msg, err := c.decryptMessage(input, nonce, sec.readKey)
	return node, msg, err
}

func (c *wireCodec) decodeHandshake(fromAddr string, head *packetHeaderV5) (*enode.Node, []byte, *handshakeSecrets, error) {
	auth, err := c.decodeHandshakeAuthData(head)
	if err != nil {
		return nil, nil, nil, err
	}

	// Verify against our last WHOAREYOU.
	challenge := c.sc.getHandshake(head.SrcID, fromAddr)
	if challenge == nil {
		return nil, nil, nil, errUnexpectedHandshake
	}
	// Get node record.
	node, err := c.decodeHandshakeRecord(challenge.node, head.SrcID, auth.record)
	if err != nil {
		return nil, nil, nil, err
	}
	// Verify ephemeral key is on curve.
	ephkey, err := decodePubkey(c.privkey.Curve, auth.pubkey)
	if err != nil {
		return nil, nil, nil, errInvalidAuthKey
	}
	// Verify ID nonce signature.
	err = c.verifyIDSignature(challenge.IDNonce[:], auth.pubkey, auth.signature, node)
	if err != nil {
		return nil, nil, nil, err
	}
	// Derive sesssion keys.
	sec := c.deriveKeys(head.SrcID, c.localnode.ID(), c.privkey, ephkey, challenge)
	return node, auth.h.Nonce[:], sec, nil
}

// decodeHandshakeAuthData reads the authdata section of a handshake packet.
func (c *wireCodec) decodeHandshakeAuthData(head *packetHeaderV5) (*handshakeAuthDataV5, error) {
	if int(head.AuthSize) < sizeofHandshakeAuthDataV5 {
		return nil, fmt.Errorf("header authsize %d too low for handshake", head.AuthSize)
	}
	if c.reader.Len() < int(head.AuthSize) {
		return nil, errTooShort
	}

	// Decode fixed size part.
	var auth handshakeAuthDataV5
	binary.Read(&c.reader, binary.BigEndian, &auth.h)
	if auth.h.Version > handshakeVersion || auth.h.Version < minVersion {
		return nil, fmt.Errorf("invalid handshake version %d", auth.h.Version)
	}
	// Decode variable-size part.
	varspace := int(head.AuthSize) - sizeofHandshakeAuthDataV5
	if int(auth.h.SigSize)+int(auth.h.PubkeySize) > varspace {
		return nil, fmt.Errorf("invalid handshake data sizes (%d+%d > %d)", auth.h.SigSize, auth.h.PubkeySize, varspace)
	}
	if !readNew(&auth.signature, int(auth.h.SigSize), &c.reader) {
		return nil, fmt.Errorf("can't read auth signature")
	}
	if !readNew(&auth.pubkey, int(auth.h.PubkeySize), &c.reader) {
		return nil, fmt.Errorf("can't read auth pubkey")
	}
	recordsize := varspace - int(auth.h.SigSize) - int(auth.h.PubkeySize)
	if !readNew(&auth.record, recordsize, &c.reader) {
		return nil, fmt.Errorf("can't read auth node record")
	}
	return &auth, nil
}

// readNew reads 'length' bytes from 'r' and stores them into 'data'.
func readNew(data *[]byte, length int, r *bytes.Reader) bool {
	if length == 0 {
		return true
	}
	*data = make([]byte, length)
	n, _ := r.Read(*data)
	return n == length
}

// decodeHandshakeRecord verifies the node record contained in a handshake packet. The
// remote node should include the record if we don't have one or if ours is older than the
// latest sequence number.
func (c *wireCodec) decodeHandshakeRecord(local *enode.Node, wantID enode.ID, remote []byte) (node *enode.Node, err error) {
	node = local
	if len(remote) > 0 {
		var record enr.Record
		if err := rlp.DecodeBytes(remote, &record); err != nil {
			return nil, err
		}
		if local == nil || local.Seq() < record.Seq() {
			n, err := enode.New(enode.ValidSchemes, &record)
			if err != nil {
				return nil, fmt.Errorf("invalid node record: %v", err)
			}
			if n.ID() != wantID {
				return nil, fmt.Errorf("record in handshake has wrong ID: %v", n.ID())
			}
			node = n
		}
	}
	if node == nil {
		err = errNoRecord
	}
	return node, err
}

// decodeMessage reads packet data following the header as an ordinary message packet.
func (c *wireCodec) decodeMessage(fromAddr string, head *packetHeaderV5, input []byte) (packetV5, error) {
	if c.reader.Len() < sizeofMessageAuthDataV5 {
		return nil, errTooShort
	}
	key := c.sc.readKey(head.SrcID, fromAddr)
	auth := new(messageAuthDataV5)
	binary.Read(&c.reader, binary.BigEndian, auth)

	// Try decrypting the message.
	msg, err := c.decryptMessage(input, auth.Nonce[:], key)
	if err == errMessageDecrypt {
		// It didn't work. Start the handshake since this is an ordinary message packet.
		return &unknownV5{AuthTag: auth.Nonce[:]}, nil
	}
	return msg, err
}

func (c *wireCodec) decryptMessage(input, nonce, key []byte) (packetV5, error) {
	headerData := input[:len(input)-c.reader.Len()]
	messageCT := input[len(headerData):]
	message, err := decryptGCM(key, nonce, messageCT, headerData)
	if err != nil {
		return nil, errMessageDecrypt
	}
	if len(message) == 0 {
		return nil, errMessageTooShort
	}
	return decodePacketBodyV5(message[0], message[1:])
}

// decodePacketBody decodes the body of an encrypted discovery packet.
func decodePacketBodyV5(ptype byte, body []byte) (packetV5, error) {
	var dec packetV5
	switch ptype {
	case p_pingV5:
		dec = new(pingV5)
	case p_pongV5:
		dec = new(pongV5)
	case p_findnodeV5:
		dec = new(findnodeV5)
	case p_nodesV5:
		dec = new(nodesV5)
	case p_talkreqV5:
		dec = new(talkreqV5)
	case p_talkrespV5:
		dec = new(talkrespV5)
	case p_requestTicketV5:
		dec = new(requestTicketV5)
	case p_ticketV5:
		dec = new(ticketV5)
	case p_regtopicV5:
		dec = new(regtopicV5)
	case p_regconfirmationV5:
		dec = new(regconfirmationV5)
	case p_topicqueryV5:
		dec = new(topicqueryV5)
	default:
		return nil, fmt.Errorf("unknown packet type %d", ptype)
	}
	if err := rlp.DecodeBytes(body, dec); err != nil {
		return nil, err
	}
	return dec, nil
}

// signIDNonce creates the ID nonce signature.
func (c *wireCodec) signIDNonce(nonce, ephkey []byte) ([]byte, error) {
	idsig, err := crypto.Sign(c.idNonceHash(nonce, ephkey), c.privkey)
	if err != nil {
		return nil, fmt.Errorf("can't sign: %v", err)
	}
	return idsig[:len(idsig)-1], nil // remove recovery ID
}

// idNonceHash computes the hash of id nonce with prefix.
func (c *wireCodec) idNonceHash(nonce, ephkey []byte) []byte {
	h := c.sha256reset()
	h.Write([]byte(idNoncePrefix))
	h.Write(nonce)
	h.Write(ephkey)
	return h.Sum(nil)
}

// verifyIDSignature checks that signature over idnonce was made by the node with given record.
func (c *wireCodec) verifyIDSignature(nonce, ephkey, sig []byte, n *enode.Node) error {
	switch idscheme := n.Record().IdentityScheme(); idscheme {
	case "v4":
		var pk ecdsa.PublicKey
		n.Load((*enode.Secp256k1)(&pk)) // cannot fail because record is valid
		if !crypto.VerifySignature(crypto.FromECDSAPub(&pk), c.idNonceHash(nonce, ephkey), sig) {
			return errInvalidNonceSig
		}
		return nil
	default:
		return fmt.Errorf("can't verify ID nonce signature against scheme %q", idscheme)
	}
}

// deriveKeys generates session keys using elliptic-curve Diffie-Hellman key agreement.
func (c *wireCodec) deriveKeys(n1, n2 enode.ID, priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey, challenge *whoareyouV5) *handshakeSecrets {
	eph := ecdh(priv, pub)
	if eph == nil {
		return nil
	}

	info := []byte("discovery v5 key agreement")
	info = append(info, n1[:]...)
	info = append(info, n2[:]...)
	kdf := hkdf.New(c.sha256reset, eph, challenge.IDNonce[:], info)
	sec := handshakeSecrets{
		writeKey: make([]byte, aesKeySize),
		readKey:  make([]byte, aesKeySize),
	}
	kdf.Read(sec.writeKey)
	kdf.Read(sec.readKey)
	for i := range eph {
		eph[i] = 0
	}
	return &sec
}

// sha256reset returns the shared hash instance.
func (c *wireCodec) sha256reset() hash.Hash {
	c.sha256.Reset()
	return c.sha256
}

// sha256sum computes sha256 on the concatenation of inputs.
func (c *wireCodec) sha256sum(inputs ...[]byte) []byte {
	c.sha256.Reset()
	for _, b := range inputs {
		c.sha256.Write(b)
	}
	return c.sha256.Sum(nil)
}

// ecdh creates a shared secret.
func ecdh(privkey *ecdsa.PrivateKey, pubkey *ecdsa.PublicKey) []byte {
	secX, secY := pubkey.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	if secX == nil {
		return nil
	}
	sec := make([]byte, 33)
	sec[0] = 0x02 | byte(secY.Bit(0))
	math.ReadBits(secX, sec[1:])
	return sec
}

// encryptGCM encrypts pt using AES-GCM with the given key and nonce.
func encryptGCM(dest, key, nonce, pt, authData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(fmt.Errorf("can't create block cipher: %v", err))
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		panic(fmt.Errorf("can't create GCM: %v", err))
	}
	return aesgcm.Seal(dest, nonce, pt, authData), nil
}

// decryptGCM decrypts ct using AES-GCM with the given key and nonce.
func decryptGCM(key, nonce, ct, authData []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("can't create block cipher: %v", err)
	}
	if len(nonce) != gcmNonceSize {
		return nil, fmt.Errorf("invalid GCM nonce size: %d", len(nonce))
	}
	aesgcm, err := cipher.NewGCMWithNonceSize(block, gcmNonceSize)
	if err != nil {
		return nil, fmt.Errorf("can't create GCM: %v", err)
	}
	pt := make([]byte, 0, len(ct))
	return aesgcm.Open(pt, nonce, ct, authData)
}

// header masking

const maskIVSize = 16

func headerMask(destID enode.ID, input []byte) cipher.Stream {
	block, err := aes.NewCipher(destID[:16])
	if err != nil {
		panic("can't create cipher")
	}
	return cipher.NewCTR(block, input[:maskIVSize])
}

func maskOutputPacket(destID enode.ID, output []byte, headerDataLen int) []byte {
	masked := make([]byte, maskIVSize+len(output))
	crand.Read(masked[:maskIVSize])
	mask := headerMask(destID, masked)
	copy(masked[maskIVSize:], output)
	mask.XORKeyStream(masked[maskIVSize:], output[:headerDataLen])
	return masked
}
