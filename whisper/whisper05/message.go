// Copyright 2014 The go-ethereum Authors
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

// Contains the Whisper protocol Message element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#messages.
// todo: fix the spec link, and move it to doc.go

package whisper05

import (
	crand "crypto/rand"
	"errors"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/crypto/pbkdf2"
)

// Options specifies the exact way a message should be wrapped into an Envelope.
type Options struct {
	TTL      uint32
	Src      *ecdsa.PrivateKey
	Dst      *ecdsa.PublicKey
	KeySym   []byte
	Topic    TopicType
	Pading   []byte
	WorkTime uint32
	PoW      float64
}

// SentMessage represents an end-user data packet to transmit through the
// Whisper protocol. These are wrapped into Envelopes that need not be
// understood by intermediate nodes, just forwarded.
type SentMessage struct {
	Raw []byte
}

// ReceivedMessage represents a data packet to be received through the
// Whisper protocol.
type ReceivedMessage struct {
	Raw []byte

	Payload   []byte
	Padding   []byte
	Signature []byte

	PoW          float64          // Proof of work as described in the Whisper spec
	Sent         uint32           // Time when the message was posted into the network
	TTL          uint32           // Maximum time to live allowed for the message
	Src          *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Dst          *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Topic        TopicType
	TopicKeyHash common.Hash // The Keccak256Hash of the key, associated with the Topic
	EnvelopeHash common.Hash // Message envelope hash to act as a unique id
}

func DeriveTopicFromSymmetricKey(key []byte) TopicType {
	// todo: it is not secure enough, use kdf instead
	hash := crypto.Keccak256Hash(key)
	return HashToTopic(hash)
}

func isMessageSigned(flags byte) bool {
	return (flags & signatureFlag) != 0
}

func isMessagePadded(flags byte) bool {
	return (flags & paddingFlag) != 0
}

func (self *ReceivedMessage) isSymmetricEncryption() bool {
	return self.TopicKeyHash != common.Hash{}
}

func (self *ReceivedMessage) isAsymmetricEncryption() bool {
	return self.Dst != nil
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewSentMessage(payload []byte) *SentMessage {
	// Construct an initial flag set: no signature, no padding, other bits random
	buf := make([]byte, 1)
	crand.Read(buf)
	flags := buf[0]
	flags &= ^signatureFlag
	flags &= ^paddingFlag

	msg := SentMessage{}
	msg.Raw = make([]byte, 1, len(payload)+signatureLength+maxPadLength+1)
	msg.Raw[0] = flags
	msg.Raw = append(msg.Raw, payload...)
	return &msg
}

// appendPadding appends the pseudorandom padding bytes and sets the padding flag.
// The last byte contains the size of padding (thus, its size must not exceed 256).
func (self *SentMessage) appendPadding(options Options) {
	if isMessageSigned(self.Raw[0]) {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to pad a message which was already signed")
		return
	} else if isMessagePadded(self.Raw[0]) {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to pad a message which was already padded")
		return
	}

	total := len(self.Raw)
	if options.Src != nil {
		total += signatureLength
	}
	odd := total % maxPadLength
	if odd > 0 {
		padSize := maxPadLength - odd
		buf := make([]byte, padSize)
		crand.Read(buf)
		if options.Pading != nil {
			copy(buf, options.Pading)
		}
		buf[padSize-1] = byte(padSize)
		self.Raw = append(self.Raw, buf...)
		self.Raw[0] |= paddingFlag
	}
}

// sign calculates and sets the cryptographic signature for the message,
// also setting the sign flag.
func (self *SentMessage) sign(key *ecdsa.PrivateKey) (err error) {
	if isMessageSigned(self.Raw[0]) {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to sign a message which was already signed")
		return
	}
	hash := crypto.Keccak256(self.Raw)
	signature, err := crypto.Sign(hash, key)
	if err != nil {
		self.Raw = append(self.Raw, signature...)
		self.Raw[0] |= signatureFlag
	}
	return
}

// encryptAsymmetric encrypts a message with a public key.
func (self *SentMessage) encryptAsymmetric(key *ecdsa.PublicKey) error {
	encrypted, err := crypto.Encrypt(key, self.Raw)
	if err == nil {
		self.Raw = encrypted
	}
	return err
}

// encryptSymmetric encrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (self *SentMessage) encryptSymmetric(key []byte) (salt []byte, nonce []byte, err error) {
	salt = make([]byte, saltLength)
	_, err = crand.Read(salt)
	if err != nil {
		return
	}

	derivedKey := pbkdf2.Key(key, salt, kdfIterations, aesKeyLength, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}

	// never use more than 2^32 random nonces with a given key
	nonce = make([]byte, aesgcm.NonceSize())
	_, err = crand.Read(nonce)
	if err != nil {
		return
	}
	self.Raw = aesgcm.Seal(nil, nonce, self.Raw, nil)
	return
}

// Wrap bundles the message into an Envelope to transmit over the network.
//
// pow (Proof Of Work) controls how much time to spend on hashing the message,
// inherently controlling its priority through the network (smaller hash, bigger
// priority).
//
// The user can control the amount of identity, privacy and encryption through
// the options parameter as follows:
//   - options.From == nil && options.To == nil: anonymous broadcast
//   - options.From != nil && options.To == nil: signed broadcast (known sender)
//   - options.From == nil && options.To != nil: encrypted anonymous message
//   - options.From != nil && options.To != nil: encrypted signed message
func (self *SentMessage) Wrap(options Options) (envelope *Envelope, err error) {
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}
	self.appendPadding(options)
	if options.Src != nil {
		if err = self.sign(options.Src); err != nil {
			return
		}
	}
	if len(self.Raw) > msgMaxLength {
		glog.V(logger.Error).Infof("Message size must not exceed %d bytes", msgMaxLength)
		err = errors.New("Oversized message")
		return
	}
	var salt, nonce []byte
	if options.Dst != nil {
		err = self.encryptAsymmetric(options.Dst)
	} else if options.KeySym != nil {
		salt, nonce, err = self.encryptSymmetric(options.KeySym)
	} else {
		err = errors.New("Unable to encrypt the message: neither Dst nor Key")
	}

	if err == nil {
		if (options.Topic == TopicType{}) {
			options.Topic = DeriveTopicFromSymmetricKey(options.KeySym)
		}

		envelope = NewEnvelope(options.TTL, options.Topic, salt, nonce, self)
		envelope.Seal(options)
	}
	return
}

// decryptSymmetric decrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (self *ReceivedMessage) decryptSymmetric(key []byte, salt []byte, nonce []byte) error {
	derivedKey := pbkdf2.Key(key, salt, kdfIterations, aesKeyLength, sha256.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if len(nonce) != aesgcm.NonceSize() {
		glog.V(logger.Error).Infof("AES nonce size must be %d bytes", aesgcm.NonceSize())
		return errors.New("Wrong AES nonce size")
	}
	decrypted, err := aesgcm.Open(nil, nonce, self.Raw, nil)
	if err != nil {
		return err
	}
	self.Raw = decrypted
	return nil
}

// decryptAsymmetric decrypts an encrypted payload with a private key.
func (self *ReceivedMessage) decryptAsymmetric(key *ecdsa.PrivateKey) error {
	decrypted, err := crypto.Decrypt(key, self.Raw)
	if err == nil {
		self.Raw = decrypted
	}
	return err
}

// Validate checks the validity and extracts the fields in case of success
func (self *ReceivedMessage) Validate() bool {
	sz := len(self.Raw)
	cur := sz
	if sz < 1 {
		return false
	}

	if isMessageSigned(self.Raw[0]) {
		cur -= signatureLength
		if cur <= 1 {
			return false
		}
		self.Signature = self.Raw[cur:]
		self.Src = self.Recover()
		if self.Src == nil {
			return false
		}
	}

	if isMessagePadded(self.Raw[0]) {
		paddingSize := int(self.Raw[cur-1])
		beg := cur - paddingSize
		if beg <= 1 {
			return false
		}
		self.Padding = self.Raw[beg : cur-1]
		cur = beg
	}

	self.Payload = self.Raw[1:cur]
	if self.isSymmetricEncryption() == self.isAsymmetricEncryption() {
		return false
	}
	return true
}

// Recover retrieves the public key of the message signer.
func (self *ReceivedMessage) Recover() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	pub, err := crypto.SigToPub(self.hash(), self.Signature)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub
}

// hash calculates the SHA3 checksum of the message flags, payload and padding.
func (self *ReceivedMessage) hash() []byte {
	if isMessageSigned(self.Raw[0]) {
		sz := len(self.Raw) - signatureLength
		return crypto.Keccak256(self.Raw[:sz])
	}
	return crypto.Keccak256(self.Raw)
}
