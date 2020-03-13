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
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/sha256"
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
		ReqID    []byte
		Distance uint
	}

	// NODES is the reply to FINDNODE and TOPICQUERY.
	nodesV5 struct {
		ReqID []byte
		Total uint8
		Nodes []*enr.Record
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
	authSchemeName   = "gcm"
	aesKeySize       = 16
	gcmNonceSize     = 12
	idNoncePrefix    = "discovery-id-nonce"
	handshakeTimeout = time.Second
)

var (
	errTooShort               = errors.New("packet too short")
	errUnexpectedHandshake    = errors.New("unexpected auth response, not in handshake")
	errHandshakeNonceMismatch = errors.New("wrong nonce in auth response")
	errInvalidAuthKey         = errors.New("invalid ephemeral pubkey")
	errUnknownAuthScheme      = errors.New("unknown auth scheme in handshake")
	errNoRecord               = errors.New("expected ENR in handshake but none sent")
	errInvalidNonceSig        = errors.New("invalid ID nonce signature")
	zeroNonce                 = make([]byte, gcmNonceSize)
)

// wireCodec encodes and decodes discovery v5 packets.
type wireCodec struct {
	sha256           hash.Hash
	localnode        *enode.LocalNode
	privkey          *ecdsa.PrivateKey
	myChtagHash      enode.ID
	myWhoareyouMagic []byte

	sc *sessionCache
}

type handshakeSecrets struct {
	writeKey, readKey, authRespKey []byte
}

type authHeader struct {
	authHeaderList
	isHandshake bool
}

type authHeaderList struct {
	Auth         []byte   // authentication info of packet
	IDNonce      [32]byte // IDNonce of WHOAREYOU
	Scheme       string   // name of encryption/authentication scheme
	EphemeralKey []byte   // ephemeral public key
	Response     []byte   // encrypted authResponse
}

type authResponse struct {
	Version   uint
	Signature []byte
	Record    *enr.Record `rlp:"nil"` // sender's record
}

func (h *authHeader) DecodeRLP(r *rlp.Stream) error {
	k, _, err := r.Kind()
	if err != nil {
		return err
	}
	if k == rlp.Byte || k == rlp.String {
		return r.Decode(&h.Auth)
	}
	h.isHandshake = true
	return r.Decode(&h.authHeaderList)
}

// ephemeralKey decodes the ephemeral public key in the header.
func (h *authHeaderList) ephemeralKey(curve elliptic.Curve) *ecdsa.PublicKey {
	var key encPubkey
	copy(key[:], h.EphemeralKey)
	pubkey, _ := decodePubkey(curve, key)
	return pubkey
}

// newWireCodec creates a wire codec.
func newWireCodec(ln *enode.LocalNode, key *ecdsa.PrivateKey, clock mclock.Clock) *wireCodec {
	c := &wireCodec{
		sha256:    sha256.New(),
		localnode: ln,
		privkey:   key,
		sc:        newSessionCache(1024, clock),
	}
	// Create magic strings for packet matching.
	self := ln.ID()
	c.myWhoareyouMagic = c.sha256sum(self[:], []byte("WHOAREYOU"))
	copy(c.myChtagHash[:], c.sha256sum(self[:]))
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
	// Ensure calling code sets node if needed.
	if challenge != nil && challenge.node == nil {
		panic("BUG: missing challenge.node in encode")
	}
	writeKey := c.sc.writeKey(id, addr)
	if writeKey != nil || challenge != nil {
		return c.encodeEncrypted(id, addr, packet, writeKey, challenge)
	}
	return c.encodeRandom(id)
}

// encodeRandom encodes a random packet.
func (c *wireCodec) encodeRandom(toID enode.ID) ([]byte, []byte, error) {
	tag := xorTag(c.sha256sum(toID[:]), c.localnode.ID())
	r := make([]byte, 44) // TODO randomize size
	if _, err := crand.Read(r); err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcmNonceSize)
	if _, err := crand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("can't get random data: %v", err)
	}
	b := new(bytes.Buffer)
	b.Write(tag[:])
	rlp.Encode(b, nonce)
	b.Write(r)
	return b.Bytes(), nonce, nil
}

// encodeWhoareyou encodes WHOAREYOU.
func (c *wireCodec) encodeWhoareyou(toID enode.ID, packet *whoareyouV5) ([]byte, error) {
	// Sanity check node field to catch misbehaving callers.
	if packet.RecordSeq > 0 && packet.node == nil {
		panic("BUG: missing node in whoareyouV5 with non-zero seq")
	}
	b := new(bytes.Buffer)
	b.Write(c.sha256sum(toID[:], []byte("WHOAREYOU")))
	err := rlp.Encode(b, packet)
	return b.Bytes(), err
}

// encodeEncrypted encodes an encrypted packet.
func (c *wireCodec) encodeEncrypted(toID enode.ID, toAddr string, packet packetV5, writeKey []byte, challenge *whoareyouV5) (enc []byte, authTag []byte, err error) {
	nonce := make([]byte, gcmNonceSize)
	if _, err := crand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("can't get random data: %v", err)
	}

	var headEnc []byte
	if challenge == nil {
		// Regular packet, use existing key and simply encode nonce.
		headEnc, _ = rlp.EncodeToBytes(nonce)
	} else {
		// We're answering WHOAREYOU, generate new keys and encrypt with those.
		header, sec, err := c.makeAuthHeader(nonce, challenge)
		if err != nil {
			return nil, nil, err
		}
		if headEnc, err = rlp.EncodeToBytes(header); err != nil {
			return nil, nil, err
		}
		c.sc.storeNewSession(toID, toAddr, sec.readKey, sec.writeKey)
		writeKey = sec.writeKey
	}

	// Encode the packet.
	body := new(bytes.Buffer)
	body.WriteByte(packet.kind())
	if err := rlp.Encode(body, packet); err != nil {
		return nil, nil, err
	}
	tag := xorTag(c.sha256sum(toID[:]), c.localnode.ID())
	headsize := len(tag) + len(headEnc)
	headbuf := make([]byte, headsize)
	copy(headbuf[:], tag[:])
	copy(headbuf[len(tag):], headEnc)

	// Encrypt the body.
	enc, err = encryptGCM(headbuf, writeKey, nonce, body.Bytes(), tag[:])
	return enc, nonce, err
}

// encodeAuthHeader creates the auth header on a call packet following WHOAREYOU.
func (c *wireCodec) makeAuthHeader(nonce []byte, challenge *whoareyouV5) (*authHeaderList, *handshakeSecrets, error) {
	resp := &authResponse{Version: 5}

	// Add our record to response if it's newer than what remote
	// side has.
	ln := c.localnode.Node()
	if challenge.RecordSeq < ln.Seq() {
		resp.Record = ln.Record()
	}

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

	// Add ID nonce signature to response.
	idsig, err := c.signIDNonce(challenge.IDNonce[:], ephpubkey[:])
	if err != nil {
		return nil, nil, fmt.Errorf("can't sign: %v", err)
	}
	resp.Signature = idsig

	// Create session keys.
	sec := c.deriveKeys(c.localnode.ID(), challenge.node.ID(), ephkey, remotePubkey, challenge)
	if sec == nil {
		return nil, nil, fmt.Errorf("key derivation failed")
	}

	// Encrypt the authentication response and assemble the auth header.
	respRLP, err := rlp.EncodeToBytes(resp)
	if err != nil {
		return nil, nil, fmt.Errorf("can't encode auth response: %v", err)
	}
	respEnc, err := encryptGCM(nil, sec.authRespKey, zeroNonce, respRLP, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("can't encrypt auth response: %v", err)
	}
	head := &authHeaderList{
		Auth:         nonce,
		Scheme:       authSchemeName,
		IDNonce:      challenge.IDNonce,
		EphemeralKey: ephpubkey[:],
		Response:     respEnc,
	}
	return head, sec, err
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
		writeKey:    make([]byte, aesKeySize),
		readKey:     make([]byte, aesKeySize),
		authRespKey: make([]byte, aesKeySize),
	}
	kdf.Read(sec.writeKey)
	kdf.Read(sec.readKey)
	kdf.Read(sec.authRespKey)
	for i := range eph {
		eph[i] = 0
	}
	return &sec
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

// decode decodes a discovery packet.
func (c *wireCodec) decode(input []byte, addr string) (enode.ID, *enode.Node, packetV5, error) {
	// Delete timed-out handshakes. This must happen before decoding to avoid
	// processing the same handshake twice.
	c.sc.handshakeGC()

	if len(input) < 32 {
		return enode.ID{}, nil, nil, errTooShort
	}
	if bytes.HasPrefix(input, c.myWhoareyouMagic) {
		p, err := c.decodeWhoareyou(input)
		return enode.ID{}, nil, p, err
	}
	sender := xorTag(input[:32], c.myChtagHash)
	p, n, err := c.decodeEncrypted(sender, addr, input)
	return sender, n, p, err
}

// decodeWhoareyou decode a WHOAREYOU packet.
func (c *wireCodec) decodeWhoareyou(input []byte) (packetV5, error) {
	packet := new(whoareyouV5)
	err := rlp.DecodeBytes(input[32:], packet)
	return packet, err
}

// decodeEncrypted decodes an encrypted discovery packet.
func (c *wireCodec) decodeEncrypted(fromID enode.ID, fromAddr string, input []byte) (packetV5, *enode.Node, error) {
	// Decode packet header.
	var head authHeader
	r := bytes.NewReader(input[32:])
	err := rlp.Decode(r, &head)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt and process auth response.
	readKey, node, err := c.decodeAuth(fromID, fromAddr, &head)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt and decode the packet body.
	headsize := len(input) - r.Len()
	bodyEnc := input[headsize:]
	body, err := decryptGCM(readKey, head.Auth, bodyEnc, input[:32])
	if err != nil {
		if !head.isHandshake {
			// Can't decrypt, start handshake.
			return &unknownV5{AuthTag: head.Auth}, nil, nil
		}
		return nil, nil, fmt.Errorf("handshake failed: %v", err)
	}
	if len(body) == 0 {
		return nil, nil, errTooShort
	}
	p, err := decodePacketBodyV5(body[0], body[1:])
	return p, node, err
}

// decodeAuth processes an auth header.
func (c *wireCodec) decodeAuth(fromID enode.ID, fromAddr string, head *authHeader) ([]byte, *enode.Node, error) {
	if !head.isHandshake {
		return c.sc.readKey(fromID, fromAddr), nil, nil
	}

	// Remote is attempting handshake. Verify against our last WHOAREYOU.
	challenge := c.sc.getHandshake(fromID, fromAddr)
	if challenge == nil {
		return nil, nil, errUnexpectedHandshake
	}
	if head.IDNonce != challenge.IDNonce {
		return nil, nil, errHandshakeNonceMismatch
	}
	sec, n, err := c.decodeAuthResp(fromID, fromAddr, &head.authHeaderList, challenge)
	if err != nil {
		return nil, n, err
	}
	// Swap keys to match remote.
	sec.readKey, sec.writeKey = sec.writeKey, sec.readKey
	c.sc.storeNewSession(fromID, fromAddr, sec.readKey, sec.writeKey)
	c.sc.deleteHandshake(fromID, fromAddr)
	return sec.readKey, n, err
}

// decodeAuthResp decodes and verifies an authentication response.
func (c *wireCodec) decodeAuthResp(fromID enode.ID, fromAddr string, head *authHeaderList, challenge *whoareyouV5) (*handshakeSecrets, *enode.Node, error) {
	// Decrypt / decode the response.
	if head.Scheme != authSchemeName {
		return nil, nil, errUnknownAuthScheme
	}
	ephkey := head.ephemeralKey(c.privkey.Curve)
	if ephkey == nil {
		return nil, nil, errInvalidAuthKey
	}
	sec := c.deriveKeys(fromID, c.localnode.ID(), c.privkey, ephkey, challenge)
	respPT, err := decryptGCM(sec.authRespKey, zeroNonce, head.Response, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("can't decrypt auth response header: %v", err)
	}
	var resp authResponse
	if err := rlp.DecodeBytes(respPT, &resp); err != nil {
		return nil, nil, fmt.Errorf("invalid auth response: %v", err)
	}

	// Verify response node record. The remote node should include the record
	// if we don't have one or if ours is older than the latest version.
	node := challenge.node
	if resp.Record != nil {
		if node == nil || node.Seq() < resp.Record.Seq() {
			n, err := enode.New(enode.ValidSchemes, resp.Record)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid node record: %v", err)
			}
			if n.ID() != fromID {
				return nil, nil, fmt.Errorf("record in auth respose has wrong ID: %v", n.ID())
			}
			node = n
		}
	}
	if node == nil {
		return nil, nil, errNoRecord
	}

	// Verify ID nonce signature.
	err = c.verifyIDSignature(challenge.IDNonce[:], head.EphemeralKey, resp.Signature, node)
	if err != nil {
		return nil, nil, err
	}
	return sec, node, nil
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

func xorTag(a []byte, b enode.ID) enode.ID {
	var r enode.ID
	for i := range r {
		r[i] = a[i] ^ b[i]
	}
	return r
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
