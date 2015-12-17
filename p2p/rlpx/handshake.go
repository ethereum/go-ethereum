// Copyright 2015 The go-ethereum Authors
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

package rlpx

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

const (
	maxUint24           = ^uint32(0) >> 8
	kdfSharedDataPrefix = "rlpx handshake\x05"

	// Handshake Sizes
	sskLen         = 16 // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen         = 65 // elliptic S256
	pubLen         = 64 // 512 bit pubkey in uncompressed representation without format byte
	shaLen         = 32 // hash length (for nonce etc)
	nonceLen       = 24
	authMsgLen     = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen    = pubLen + shaLen + 1
	eciesBytes     = 65 + 16 + 32
	encAuthMsgLen  = authMsgLen + eciesBytes  // size of the final ECIES payload sent as initiator's handshake
	encAuthRespLen = authRespLen + eciesBytes // size of the final ECIES payload sent as receiver's handshake
)

var zero [32]byte

// encHandshake contains the state of the encryption handshake.
type handshake struct {
	conn                 io.ReadWriter
	initiator            bool
	localPrivKey         *ecdsa.PrivateKey
	remotePub            *ecies.PublicKey  // remote-pubk
	initNonce, respNonce []byte            // nonce
	randomPrivKey        *ecies.PrivateKey // ecdhe-random
	remoteRandomPub      *ecies.PublicKey  // ecdhe-random-pubk
}

// secrets represents the derived secrets for authenticated encryption.
type secrets struct {
	encKey, encIV, macKey []byte
	mac                   hash.Hash
}

type handshakeRandSource interface {
	generateNonce(b []byte) error
	generateKey() (*ecies.PrivateKey, error)
}

type realRandSource struct{}

func (realRandSource) generateNonce(b []byte) error {
	_, err := io.ReadFull(rand.Reader, b)
	return err
}

func (realRandSource) generateKey() (*ecies.PrivateKey, error) {
	return ecies.GenerateKey(rand.Reader, secp256k1.S256(), nil)
}

func (h *handshake) deriveSecrets(forceV4 bool, auth, authResp []byte) (vsn uint, ingress, egress secrets, err error) {
	remoteNonce := h.initNonce
	if h.initiator {
		remoteNonce = h.respNonce
	}
	remoteVersion := binary.BigEndian.Uint64(remoteNonce[nonceLen:])
	if forceV4 || remoteVersion > 255 {
		return h.deriveSecretsV4(auth, authResp)
	}
	return h.deriveSecretsV5()
}

func (h *handshake) deriveSecretsV4(auth, authResp []byte) (vsn uint, ingress, egress secrets, err error) {
	vsn = 4
	ecdheSecret, err := h.randomPrivKey.GenerateShared(h.remoteRandomPub, sskLen, sskLen)
	if err != nil {
		return vsn, ingress, egress, err
	}
	sharedSecret := crypto.Sha3(ecdheSecret, crypto.Sha3(h.respNonce, h.initNonce))
	aesSecret := crypto.Sha3(ecdheSecret, sharedSecret)
	macSecret := crypto.Sha3(ecdheSecret, aesSecret)

	egress = secrets{encKey: aesSecret, encIV: zero[:16], macKey: macSecret}
	egress.mac = sha3.NewKeccak256()
	egress.mac.Write(xor(h.initNonce, macSecret))
	egress.mac.Write(authResp)
	ingress = secrets{encKey: aesSecret, encIV: zero[:16], macKey: macSecret}
	ingress.mac = sha3.NewKeccak256()
	ingress.mac.Write(xor(h.respNonce, macSecret))
	ingress.mac.Write(auth)
	if h.initiator {
		ingress, egress = egress, ingress
	}
	return vsn, ingress, egress, nil
}

func (h *handshake) deriveSecretsV5() (vsn uint, ingress, egress secrets, err error) {
	vsn = 5
	ecdheSecret, err := h.randomPrivKey.GenerateShared(h.remoteRandomPub, sskLen, sskLen)
	if err != nil {
		return vsn, ingress, egress, err
	}
	initPub := exportPubkey(h.remotePub)
	respPub := elliptic.Marshal(h.localPrivKey.Curve, h.localPrivKey.X, h.localPrivKey.Y)[1:]
	if h.initiator {
		initPub, respPub = respPub, initPub
	}
	sharedData := make([]byte, len(kdfSharedDataPrefix)+nonceLen*2+pubLen*2)
	n := copy(sharedData, kdfSharedDataPrefix)
	n += copy(sharedData[n:], h.initNonce[:nonceLen])
	n += copy(sharedData[n:], h.respNonce[:nonceLen])
	n += copy(sharedData[n:], initPub)
	n += copy(sharedData[n:], respPub)
	derived, err := ecies.ConcatKDF(sha3.NewKeccak256(), ecdheSecret, sharedData, 160)
	if err != nil {
		return vsn, ingress, egress, err
	}

	ingress = secrets{encKey: derived[0:32], encIV: derived[64:80], macKey: derived[96:128]}
	ingress.mac = sha3.NewKeccak256()
	ingress.mac.Write(ingress.macKey)
	egress = secrets{encKey: derived[32:64], encIV: derived[80:96], macKey: derived[128:160]}
	egress.mac = sha3.NewKeccak256()
	egress.mac.Write(egress.macKey)
	if h.initiator {
		ingress, egress = egress, ingress
	}
	return vsn, ingress, egress, nil
}

func (h *handshake) ecdhShared(prv *ecdsa.PrivateKey) ([]byte, error) {
	return ecies.ImportECDSA(prv).GenerateShared(h.remotePub, sskLen, sskLen)
}

func (c *Conn) fillHandshake(nonce *[]byte, key **ecies.PrivateKey) (err error) {
	*nonce = make([]byte, shaLen)
	if c.cfg.ForceV4 {
		err = c.handshakeRand.generateNonce(*nonce)
	} else {
		binary.BigEndian.PutUint64((*nonce)[nonceLen:], 5)
		err = c.handshakeRand.generateNonce((*nonce)[:nonceLen])
	}
	if err != nil {
		return err
	}
	*key, err = c.handshakeRand.generateKey()
	return err
}

// initiatorHandshake negotiates connection secrets on conn.
// it should be called on the dialing end of the connection.
// prv is the local client's private key.
func (c *Conn) initiatorHandshake() (vsn uint, ingress, egress secrets, err error) {
	h := &handshake{initiator: true, localPrivKey: c.cfg.Key, remotePub: ecies.ImportECDSAPublic(c.remoteID)}
	if err := c.fillHandshake(&h.initNonce, &h.randomPrivKey); err != nil {
		return 0, ingress, egress, err
	}
	auth, err := h.authMsg()
	if err != nil {
		return 0, ingress, egress, err
	}
	if _, err := c.fd.Write(auth); err != nil {
		return 0, ingress, egress, err
	}

	response := make([]byte, encAuthRespLen)
	if _, err := io.ReadFull(c.fd, response); err != nil {
		return 0, ingress, egress, err
	}
	if err := h.decodeAuthResp(response); err != nil {
		return 0, ingress, egress, err
	}
	return h.deriveSecrets(c.cfg.ForceV4, auth, response)
}

// authMsg creates an encrypted initiator handshake message.
func (h *handshake) authMsg() ([]byte, error) {
	staticSharedSecret, err := h.ecdhShared(h.localPrivKey)
	if err != nil {
		return nil, err
	}
	// sign static-shared-secret^nonce
	signed := xor(staticSharedSecret, h.initNonce)
	signature, err := crypto.Sign(signed, h.randomPrivKey.ExportECDSA())
	if err != nil {
		return nil, err
	}
	// encode auth message: sig || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	msg := make([]byte, authMsgLen)
	n := copy(msg, signature)
	n += copy(msg[n:], crypto.Sha3(exportPubkey(&h.randomPrivKey.PublicKey)))
	n += copy(msg[n:], crypto.FromECDSAPub(&h.localPrivKey.PublicKey)[1:])
	n += copy(msg[n:], h.initNonce)
	msg[n] = 0
	// encrypt auth message using remote-pubk
	return ecies.Encrypt(rand.Reader, h.remotePub, msg, nil, nil)
}

// decodeAuthResp decode an encrypted authentication response message.
func (h *handshake) decodeAuthResp(auth []byte) error {
	msg, err := crypto.Decrypt(h.localPrivKey, auth)
	if err != nil {
		return fmt.Errorf("could not decrypt auth response (%v)", err)
	}
	h.respNonce = msg[pubLen : pubLen+shaLen]
	h.remoteRandomPub, err = importPublicKey(msg[:pubLen])
	if err != nil {
		return err
	}
	return nil
}

// recipientHandshake negotiates connection secrets on conn.
// it should be called on the listening side of the connection.
// prv is the local client's private key.
func (c *Conn) recipientHandshake() (vsn uint, remoteID *ecdsa.PublicKey, ingress, egress secrets, err error) {
	auth := make([]byte, encAuthMsgLen)
	if _, err := io.ReadFull(c.fd, auth); err != nil {
		return 0, nil, ingress, egress, err
	}
	h := &handshake{localPrivKey: c.cfg.Key}
	if err := h.decodeAuthMsg(auth); err != nil {
		return 0, nil, ingress, egress, fmt.Errorf("invalid auth: %v", err)
	}
	if err := c.fillHandshake(&h.respNonce, &h.randomPrivKey); err != nil {
		return 0, nil, ingress, egress, err
	}

	resp, err := h.authResp()
	if err != nil {
		return 0, nil, ingress, egress, fmt.Errorf("can't create auth resp: %v", err)
	}
	if _, err := c.fd.Write(resp); err != nil {
		return 0, nil, ingress, egress, err
	}
	vsn, ingress, egress, err = h.deriveSecrets(c.cfg.ForceV4, auth, resp)
	if h.remotePub != nil {
		remoteID = h.remotePub.ExportECDSA()
	}
	return vsn, remoteID, ingress, egress, err
}

func (h *handshake) decodeAuthMsg(auth []byte) error {
	msg, err := crypto.Decrypt(h.localPrivKey, auth)
	if err != nil {
		return err
	}
	// signature || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	h.initNonce = msg[authMsgLen-shaLen-1 : authMsgLen-1]
	h.remotePub, err = importPublicKey(msg[sigLen+shaLen : sigLen+shaLen+pubLen])
	if err != nil {
		return fmt.Errorf("invalid remote identity: %v", err)
	}
	// recover remote random pubkey from signed message.
	staticSharedSecret, err := h.ecdhShared(h.localPrivKey)
	if err != nil {
		return err
	}
	signed := xor(staticSharedSecret, h.initNonce)
	remoteRandomPub, err := secp256k1.RecoverPubkey(signed, msg[:sigLen])
	if err != nil {
		return err
	}
	// validate the sha3 of recovered pubkey
	remoteRandomPubMAC := msg[sigLen : sigLen+shaLen]
	shaRemoteRandomPub := crypto.Sha3(remoteRandomPub[1:])
	if !bytes.Equal(remoteRandomPubMAC, shaRemoteRandomPub) {
		return fmt.Errorf("recovered pubkey hash mismatch")
	}
	h.remoteRandomPub, _ = importPublicKey(remoteRandomPub)
	return nil
}

// authResp generates the encrypted authentication response message.
func (h *handshake) authResp() ([]byte, error) {
	// E(remote-pubk, ecdhe-random-pubk || nonce || token-flag)
	resp := make([]byte, authRespLen)
	n := copy(resp, exportPubkey(&h.randomPrivKey.PublicKey))
	n += copy(resp[n:], h.respNonce)
	resp[n] = 0
	return ecies.Encrypt(rand.Reader, h.remotePub, resp, nil, nil)
}

// importPublicKey unmarshals 512 bit public keys.
func importPublicKey(pubKey []byte) (*ecies.PublicKey, error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		// add 'uncompressed key' flag
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	// TODO: fewer pointless conversions
	return ecies.ImportECDSAPublic(crypto.ToECDSAPub(pubKey65)), nil
}

func exportPubkey(pub *ecies.PublicKey) []byte {
	if pub == nil {
		panic("nil pubkey")
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)[1:]
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
