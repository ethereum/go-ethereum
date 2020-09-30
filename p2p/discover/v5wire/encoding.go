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

package v5wire

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

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/hkdf"
)

// TODO concurrent WHOAREYOU tie-breaker
// TODO deal with WHOAREYOU amplification factor (min packet size?)
// TODO add counter to nonce
// TODO rehandshake after X packets

// Header represents a packet header.
type Header struct {
	IV [sizeofMaskingIV]byte
	HeaderData
	AuthData []byte

	src enode.ID // used by decoder
}

// HeaderData contains the static fields of a packet header.
type HeaderData struct {
	ProtocolID [6]byte
	Version    uint16
	Flag       byte
	Nonce      Nonce
	AuthSize   uint16
}

// Authdata layouts.
type (
	whoareyouAuthData struct {
		IDNonce   [16]byte // ID proof data
		RecordSeq uint64   // highest known ENR sequence of requester
	}

	handshakeAuthData struct {
		h struct {
			SrcID      enode.ID
			SigSize    byte // ignature data
			PubkeySize byte // offset of
		}
		// Trailing variable-size data.
		signature, pubkey, record []byte
	}

	messageAuthData struct {
		SrcID enode.ID
	}
)

// Packet header flag values.
const (
	flagMessage = iota
	flagWhoareyou
	flagHandshake
)

// Protocol constants.
const (
	version         = 1
	minVersion      = 1
	sizeofMaskingIV = 16

	minMessageSize      = 48 // this refers to data after static headers
	randomPacketMsgSize = 20
)

var protocolID = [6]byte{'d', 'i', 's', 'c', 'v', '5'}

// Errors.
var (
	errTooShort            = errors.New("packet too short")
	errInvalidHeader       = errors.New("invalid packet header")
	errInvalidFlag         = errors.New("invalid flag value in header")
	errMinVersion          = errors.New("version of packet header below minimum")
	errMsgTooShort         = errors.New("message/handshake packet below minimum size")
	errAuthSize            = errors.New("declared auth size is beyond packet length")
	errUnexpectedHandshake = errors.New("unexpected auth response, not in handshake")
	errInvalidAuthKey      = errors.New("invalid ephemeral pubkey")
	errNoRecord            = errors.New("expected ENR in handshake but none sent")
	errInvalidNonceSig     = errors.New("invalid ID nonce signature")
	errMessageTooShort     = errors.New("message contains no data")
	errMessageDecrypt      = errors.New("cannot decrypt message")
)

// Packet sizes.
var (
	sizeofHeaderData        = binary.Size(HeaderData{})
	sizeofWhoareyouAuthData = binary.Size(whoareyouAuthData{})
	sizeofHandshakeAuthData = binary.Size(handshakeAuthData{}.h)
	sizeofMessageAuthData   = binary.Size(messageAuthData{})
	sizeofStaticPacketData  = sizeofMaskingIV + sizeofHeaderData
)

// Codec encodes and decodes discovery v5 packets.
type Codec struct {
	sha256    hash.Hash
	localnode *enode.LocalNode
	privkey   *ecdsa.PrivateKey
	buf       bytes.Buffer // used for encoding of packets
	msgbuf    bytes.Buffer // used for encoding of message content
	reader    bytes.Reader // used for decoding
	sc        *SessionCache
}

// NewCodec creates a wire codec.
func NewCodec(ln *enode.LocalNode, key *ecdsa.PrivateKey, clock mclock.Clock) *Codec {
	c := &Codec{
		sha256:    sha256.New(),
		localnode: ln,
		privkey:   key,
		sc:        NewSessionCache(1024, clock),
	}
	return c
}

// Encode encodes a packet to a node. 'id' and 'addr' specify the destination node. The
// 'challenge' parameter should be the most recently received WHOAREYOU packet from that
// node.
func (c *Codec) Encode(id enode.ID, addr string, packet Packet, challenge *Whoareyou) ([]byte, Nonce, error) {
	var (
		header  Header
		msgdata []byte
		err     error
	)
	if p, ok := packet.(*Whoareyou); ok {
		header, err = c.encodeWhoareyou(id, p)
		if err == nil {
			c.sc.storeSentHandshake(id, addr, p)
		}
	} else if challenge != nil {
		// Remote sent a challenge, answer it with a handshake.
		header, msgdata, err = c.encodeHandshakeMessage(id, addr, packet, challenge)
	} else if session := c.sc.session(id, addr); session != nil {
		// There is a session, use it.
		header, msgdata, err = c.encodeMessage(id, session, packet)
	} else {
		// No keys, no handshake: send random data to kick off the handshake.
		header, msgdata, err = c.encodeRandom(id)
	}

	if err != nil {
		return nil, Nonce{}, err
	}
	enc, err := c.EncodeRaw(id, header, msgdata)
	return enc, header.Nonce, err
}

func (c *Codec) EncodeRaw(id enode.ID, head Header, msgdata []byte) ([]byte, error) {
	// Generate masking IV.
	if err := c.sc.maskingIVGen(head.IV[:]); err != nil {
		return nil, fmt.Errorf("can't generate masking IV: %v", err)
	}

	// Encode the packet.
	c.buf.Reset()
	c.buf.Write(head.IV[:])
	binary.Write(&c.buf, binary.BigEndian, &head.HeaderData)
	c.buf.Write(head.AuthData)

	// Apply masking.
	masked := c.buf.Bytes()[sizeofMaskingIV:]
	mask := head.mask(id)
	mask.XORKeyStream(masked[:], masked[:])

	// Write message data.
	c.buf.Write(msgdata)
	return c.buf.Bytes(), nil
}

// makeHeader creates a packet header.
func (c *Codec) makeHeader(toID enode.ID, flag byte, authsizeExtra int) Header {
	var authsize int
	switch flag {
	case flagMessage:
		authsize = sizeofMessageAuthData
	case flagWhoareyou:
		authsize = sizeofWhoareyouAuthData
	case flagHandshake:
		authsize = sizeofHandshakeAuthData
	default:
		panic(fmt.Errorf("BUG: invalid packet header flag %x", flag))
	}
	authsize += authsizeExtra
	if authsize > int(^uint16(0)) {
		panic(fmt.Errorf("BUG: auth size %d overflows uint16", authsize))
	}
	return Header{
		HeaderData: HeaderData{
			ProtocolID: protocolID,
			Version:    version,
			Flag:       flag,
			AuthSize:   uint16(authsize),
		},
	}
}

// encodeRandom encodes a packet with random content.
func (c *Codec) encodeRandom(toID enode.ID) (Header, []byte, error) {
	head := c.makeHeader(toID, flagMessage, 0)
	auth := messageAuthData{SrcID: c.localnode.ID()}
	if _, err := crand.Read(head.Nonce[:]); err != nil {
		return head, nil, fmt.Errorf("can't get random data: %v", err)
	}
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, auth)
	head.AuthData = bytesCopy(&c.buf)

	msgdata := make([]byte, randomPacketMsgSize)
	crand.Read(msgdata)
	return head, msgdata, nil
}

// encodeWhoareyou encodes a WHOAREYOU packet.
func (c *Codec) encodeWhoareyou(toID enode.ID, packet *Whoareyou) (Header, error) {
	// Sanity check node field to catch misbehaving callers.
	if packet.RecordSeq > 0 && packet.Node == nil {
		panic("BUG: missing node in whoareyouV5 with non-zero seq")
	}

	// Encode auth data.
	auth := &whoareyouAuthData{
		IDNonce:   packet.IDNonce,
		RecordSeq: packet.RecordSeq,
	}
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, auth)

	// Create header.
	head := c.makeHeader(toID, flagWhoareyou, 0)
	head.AuthData = bytesCopy(&c.buf)
	head.Nonce = packet.Nonce

	// Update header in packet.
	packet.Header = head
	return head, nil
}

// encodeHandshakeMessage encodes an encrypted message with a handshake
// response header.
func (c *Codec) encodeHandshakeMessage(toID enode.ID, addr string, packet Packet, challenge *Whoareyou) (Header, []byte, error) {
	// Ensure calling code sets challenge.node.
	if challenge.Node == nil {
		panic("BUG: missing challenge.Node in encode")
	}

	// Generate new secrets.
	auth, session, err := c.makeHandshakeAuth(toID, addr, challenge)
	if err != nil {
		return Header{}, nil, err
	}

	// Generate nonce for message.
	nonce, err := c.sc.nextNonce(session)
	if err != nil {
		return Header{}, nil, fmt.Errorf("can't generate nonce: %v", err)
	}

	// TODO: this should happen when the first authenticated message is received
	c.sc.storeNewSession(toID, addr, session)

	// Encode the auth header.
	var (
		authsizeExtra = len(auth.pubkey) + len(auth.signature) + len(auth.record)
		head          = c.makeHeader(toID, flagHandshake, authsizeExtra)
	)
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, &auth.h)
	c.buf.Write(auth.signature)
	c.buf.Write(auth.pubkey)
	c.buf.Write(auth.record)
	head.AuthData = bytesCopy(&c.buf)
	head.Nonce = nonce

	// Encode message plaintext.
	c.msgbuf.Reset()
	c.msgbuf.WriteByte(packet.Kind())
	if err := rlp.Encode(&c.msgbuf, packet); err != nil {
		return head, nil, err
	}
	messagePT := c.msgbuf.Bytes()

	// Encrypt message data.
	var msgdata []byte
	msgdata, err = encryptGCM(msgdata, session.writeKey, head.Nonce[:], messagePT, nil)
	return head, msgdata, err
}

// encodeAuthHeader creates the auth header on a request packet following WHOAREYOU.
func (c *Codec) makeHandshakeAuth(toID enode.ID, addr string, challenge *Whoareyou) (*handshakeAuthData, *session, error) {
	auth := new(handshakeAuthData)
	auth.h.SrcID = c.localnode.ID()

	// Create the ephemeral key. This needs to be first because the
	// key is part of the ID nonce signature.
	var remotePubkey = new(ecdsa.PublicKey)
	if err := challenge.Node.Load((*enode.Secp256k1)(remotePubkey)); err != nil {
		return nil, nil, fmt.Errorf("can't find secp256k1 key for recipient")
	}
	ephkey, err := c.sc.ephemeralKeyGen()
	if err != nil {
		return nil, nil, fmt.Errorf("can't generate ephemeral key")
	}
	ephpubkey := EncodePubkey(&ephkey.PublicKey)
	auth.pubkey = ephpubkey[:]
	auth.h.PubkeySize = byte(len(auth.pubkey))

	// Add ID nonce signature to response.
	idsig, err := makeIDSignature(c.sha256, c.privkey, toID, ephpubkey[:], &challenge.Header)
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
	sec := c.deriveKeys(c.localnode.ID(), challenge.Node.ID(), ephkey, remotePubkey, challenge)
	if sec == nil {
		return nil, nil, fmt.Errorf("key derivation failed")
	}
	return auth, sec, err
}

// encodeMessage encodes an encrypted message packet.
func (c *Codec) encodeMessage(toID enode.ID, s *session, packet Packet) (Header, []byte, error) {
	head := c.makeHeader(toID, flagMessage, 0)

	// Create the header.
	nonce, err := c.sc.nextNonce(s)
	if err != nil {
		return Header{}, nil, fmt.Errorf("can't generate nonce: %v", err)
	}
	auth := messageAuthData{SrcID: c.localnode.ID()}
	c.buf.Reset()
	binary.Write(&c.buf, binary.BigEndian, &auth)
	head.AuthData = bytesCopy(&c.buf)
	head.Nonce = nonce

	// Encode the message plaintext.
	c.msgbuf.Reset()
	c.msgbuf.WriteByte(packet.Kind())
	if err := rlp.Encode(&c.msgbuf, packet); err != nil {
		return head, nil, err
	}
	messagePT := c.msgbuf.Bytes()

	// Encrypt the message.
	var msgdata []byte
	msgdata, err = encryptGCM(msgdata, s.writeKey, nonce[:], messagePT, nil)
	return head, msgdata, err
}

// Decode decodes a discovery packet.
func (c *Codec) Decode(input []byte, addr string) (src enode.ID, n *enode.Node, p Packet, err error) {
	// Unmask the static header.
	if len(input) < sizeofStaticPacketData {
		return enode.ID{}, nil, nil, errTooShort
	}
	var head Header
	copy(head.IV[:], input[:sizeofMaskingIV])
	mask := head.mask(c.localnode.ID())
	input = input[sizeofMaskingIV:]
	headerData := input[:sizeofHeaderData]
	mask.XORKeyStream(headerData, headerData)

	// Decode and verify the header.
	c.reader.Reset(input)
	binary.Read(&c.reader, binary.BigEndian, &head.HeaderData)
	if err := head.checkValid(c.reader.Len()); err != nil {
		return enode.ID{}, nil, nil, err
	}

	// Unmask auth data.
	head.AuthData = make([]byte, head.AuthSize)
	copy(head.AuthData, input[sizeofHeaderData:])
	mask.XORKeyStream(head.AuthData, head.AuthData)

	// Delete timed-out handshakes. This must happen before decoding to avoid
	// processing the same handshake twice.
	c.sc.handshakeGC()

	// Decode auth part and message.
	msgdata := input[sizeofHeaderData+len(head.AuthData):]
	switch head.Flag {
	case flagWhoareyou:
		p, err = c.decodeWhoareyou(&head)
	case flagHandshake:
		n, p, err = c.decodeHandshakeMessage(addr, &head, msgdata)
	case flagMessage:
		p, err = c.decodeMessage(addr, &head, msgdata)
	default:
		err = errInvalidFlag
	}
	return head.src, n, p, err
}

// decodeWhoareyou reads packet data after the header as a WHOAREYOU packet.
func (c *Codec) decodeWhoareyou(head *Header) (Packet, error) {
	if c.reader.Len() < sizeofWhoareyouAuthData {
		return nil, errTooShort
	}
	if int(head.AuthSize) != sizeofWhoareyouAuthData {
		return nil, fmt.Errorf("invalid auth size for whoareyou")
	}
	c.reader.Reset(head.AuthData)
	auth := new(whoareyouAuthData)
	binary.Read(&c.reader, binary.BigEndian, auth)
	p := &Whoareyou{
		Header:    *head,
		IDNonce:   auth.IDNonce,
		RecordSeq: auth.RecordSeq,
	}
	return p, nil
}

func (c *Codec) decodeHandshakeMessage(fromAddr string, head *Header, msgdata []byte) (n *enode.Node, p Packet, err error) {
	node, auth, session, err := c.decodeHandshake(fromAddr, head)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt the message using the new session keys.
	msg, err := c.decryptMessage(msgdata, head.Nonce[:], session.readKey)
	if err != nil {
		return node, msg, err
	}

	// Handshake OK, drop the challenge and store the new session keys.
	c.sc.storeNewSession(auth.h.SrcID, fromAddr, session)
	c.sc.deleteHandshake(auth.h.SrcID, fromAddr)
	return node, msg, nil
}

func (c *Codec) decodeHandshake(fromAddr string, head *Header) (*enode.Node, *handshakeAuthData, *session, error) {
	auth, err := c.decodeHandshakeAuthData(head)
	if err != nil {
		return nil, nil, nil, err
	}

	// Verify against our last WHOAREYOU.
	challenge := c.sc.getHandshake(auth.h.SrcID, fromAddr)
	if challenge == nil {
		return nil, nil, nil, errUnexpectedHandshake
	}
	// Get node record.
	node, err := c.decodeHandshakeRecord(challenge.Node, auth.h.SrcID, auth.record)
	if err != nil {
		return nil, nil, nil, err
	}
	// Verify ephemeral key is on curve.
	ephkey, err := DecodePubkey(c.privkey.Curve, auth.pubkey)
	if err != nil {
		return nil, nil, nil, errInvalidAuthKey
	}
	// Verify ID nonce signature.
	sig := auth.signature
	err = verifyIDSignature(c.sha256, sig, node, c.localnode.ID(), auth.pubkey, &challenge.Header)
	if err != nil {
		return nil, nil, nil, err
	}
	// Derive sesssion keys.
	session := c.deriveKeys(auth.h.SrcID, c.localnode.ID(), c.privkey, ephkey, challenge)
	session = session.keysFlipped()
	return node, auth, session, nil
}

// decodeHandshakeAuthData reads the authdata section of a handshake packet.
func (c *Codec) decodeHandshakeAuthData(head *Header) (*handshakeAuthData, error) {
	// Decode fixed size part.
	if int(head.AuthSize) < sizeofHandshakeAuthData {
		return nil, fmt.Errorf("header authsize %d too low for handshake", head.AuthSize)
	}
	c.reader.Reset(head.AuthData)
	var auth handshakeAuthData
	binary.Read(&c.reader, binary.BigEndian, &auth.h)
	head.src = auth.h.SrcID

	// Decode variable-size part.
	varspace := int(head.AuthSize) - sizeofHandshakeAuthData
	if c.reader.Len() < varspace {
		return nil, errTooShort
	}
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

// decodeHandshakeRecord verifies the node record contained in a handshake packet. The
// remote node should include the record if we don't have one or if ours is older than the
// latest sequence number.
func (c *Codec) decodeHandshakeRecord(local *enode.Node, wantID enode.ID, remote []byte) (node *enode.Node, err error) {
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
func (c *Codec) decodeMessage(fromAddr string, head *Header, msgdata []byte) (Packet, error) {
	if len(head.AuthData) < sizeofMessageAuthData {
		return nil, errTooShort
	}
	var auth messageAuthData
	c.reader.Reset(head.AuthData)
	binary.Read(&c.reader, binary.BigEndian, &auth)
	head.src = auth.SrcID

	// Try decrypting the message.
	key := c.sc.readKey(auth.SrcID, fromAddr)
	msg, err := c.decryptMessage(msgdata, head.Nonce[:], key)
	if err == errMessageDecrypt {
		// It didn't work. Start the handshake since this is an ordinary message packet.
		return &Unknown{Nonce: head.Nonce}, nil
	}
	return msg, err
}

func (c *Codec) decryptMessage(input, nonce, readKey []byte) (Packet, error) {
	msgdata, err := decryptGCM(readKey, nonce, input, nil)
	if err != nil {
		return nil, errMessageDecrypt
	}
	if len(msgdata) == 0 {
		return nil, errMessageTooShort
	}
	return DecodeMessage(msgdata[0], msgdata[1:])
}

// deriveKeys generates session keys using elliptic-curve Diffie-Hellman key agreement.
func (c *Codec) deriveKeys(n1, n2 enode.ID, priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey, challenge *Whoareyou) *session {
	var (
		info = []byte("discovery v5 key agreement")
		salt = make([]byte, 32)
	)
	info = append(info, n1[:]...)
	info = append(info, n2[:]...)
	copy(salt, challenge.Header.IV[:])
	copy(salt[len(challenge.Header.IV):], challenge.IDNonce[:])

	eph := ecdh(priv, pub)
	if eph == nil {
		return nil
	}
	kdf := hkdf.New(c.sha256reset, eph, salt, info)
	sec := session{writeKey: make([]byte, aesKeySize), readKey: make([]byte, aesKeySize)}
	kdf.Read(sec.writeKey)
	kdf.Read(sec.readKey)
	for i := range eph {
		eph[i] = 0
	}
	return &sec
}

// sha256 returns the shared hash instance.
func (c *Codec) sha256reset() hash.Hash {
	c.sha256.Reset()
	return c.sha256
}

// checkValid performs some basic validity checks on the header.
// The packetLen here is the length remaining after the static header.
func (h *HeaderData) checkValid(packetLen int) error {
	if h.ProtocolID != protocolID {
		return errInvalidHeader
	}
	if h.Version < minVersion {
		return errMinVersion
	}
	if h.Flag != flagWhoareyou && packetLen < minMessageSize {
		return errMsgTooShort
	}
	if int(h.AuthSize) > packetLen {
		return errAuthSize
	}
	return nil
}

// headerMask returns a cipher for 'masking' / 'unmasking' packet headers.
func (h *Header) mask(destID enode.ID) cipher.Stream {
	block, err := aes.NewCipher(destID[:16])
	if err != nil {
		panic("can't create cipher")
	}
	return cipher.NewCTR(block, h.IV[:])
}

func bytesCopy(r *bytes.Buffer) []byte {
	b := make([]byte, r.Len())
	copy(b, r.Bytes())
	return b
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
