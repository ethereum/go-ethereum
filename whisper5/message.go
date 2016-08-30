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
// todo: fix the spec link

package whisper5

import (
	crand "crypto/rand"
	"errors"
	"time"

	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/crypto/pbkdf2"
)

// Options specifies the exact way a message should be wrapped into an Envelope.
type Options struct {
	Topic TopicType
	TTL   time.Duration
	Src   *ecdsa.PrivateKey
	Dst   *ecdsa.PublicKey
	Key   []byte // must be 32 bytes. todo: review
	Salt  []byte
	Pad   []byte
}

// Message represents an end-user data packet to transmit through the Whisper
// protocol. These are wrapped into Envelopes that need not be understood by
// intermediate nodes, just forwarded.
type Message struct {
	//Flags     byte   // first bit: signature presence, second: padding presence
	//Padding   []byte // the first byte contains it's size
	//Payload   []byte // todo: delete all this
	//Signature []byte

	Raw []byte

	// todo: following are the fields, extracted from the Raw field of received msg (not transmitted)
	//Sent time.Time     // Time when the message was posted into the network
	//TTL  time.Duration // Maximum time to live allowed for the message
	//
	Dst *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	//Hash common.Hash      // Message envelope hash to act as a unique id
}

func (self *Message) flags() byte {
	return self.Raw[0]
}

func (self *Message) isSigned() bool {
	return (self.Raw[0] & signatureFlag) != 0
}

func (self *Message) isPadded() bool {
	return (self.Raw[0] & paddingFlag) != 0
}

// Signature returns the signature part of the raw message.
func (self *Message) Signature() []byte {
	sz := len(self.Raw)
	if self.isSigned() && sz >= signatureLength+1 {
		return self.Raw[sz-signatureLength:]
	} else {
		return nil
	}
}

// Payload returns the payload part of the raw message.
func (self *Message) Payload() []byte {
	end := len(self.Raw)
	if self.isSigned() {
		end -= signatureLength
	}
	if self.isPadded() {
		paddingSize := int(self.Raw[end-1])
		end -= paddingSize
	}
	if end <= 1 {
		return nil
	}
	return self.Raw[1:end]
}

// Padding returns the padding part of the raw message
// without the last byte (which only contains the padding size).
func (self *Message) Padding() []byte {
	if !self.isPadded() {
		return nil
	}
	end := len(self.Raw)
	if self.isSigned() {
		end -= signatureLength
	}
	paddingSize := int(self.Raw[end-1])
	beg := end - paddingSize
	if beg <= 1 {
		return nil
	}
	return self.Raw[beg : end-1]
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewMessage(payload []byte) *Message {
	// Construct an initial flag set: no signature, no padding, other bits random
	buf := make([]byte, 1)
	crand.Read(buf)

	flags := buf[0]
	flags &= ^signatureFlag
	flags &= ^paddingFlag

	msg := Message{} //Message{Sent: time.Now()} // todo: review
	msg.Raw = make([]byte, 1, len(payload)+signatureLength+maxPadLength)
	msg.Raw[0] = flags
	msg.Raw = append(msg.Raw, payload...)
	return &msg
}

// appendPadding appends the pseudorandom padding bytes and sets the padding flag.
// The last byte contains the size of padding (thus, its size must not exceed 256).
func (self *Message) appendPadding(options Options) {
	if self.isSigned() {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to pad a message which was already signed")
		return
	} else if self.isPadded() {
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
		if options.Pad != nil {
			copy(buf, options.Pad)
		}
		buf[padSize-1] = byte(padSize)
		self.Raw = append(self.Raw, buf...)
		self.Raw[0] |= paddingFlag
	}
}

// sign calculates and sets the cryptographic signature for the message,
// also setting the sign flag.
func (self *Message) sign(key *ecdsa.PrivateKey) (err error) {
	if self.isSigned() {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to sign a message which was already signed")
		return
	}
	signature, err := crypto.Sign(self.hash(), key)
	if err != nil {
		self.Raw = append(self.Raw, signature...)
		self.Raw[0] |= signatureFlag
	}
	return
}

// Recover retrieves the public key of the message signer.
func (self *Message) Recover() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	signature := self.Signature()
	if signature == nil {
		return nil
	}
	pub, err := crypto.SigToPub(self.hash(), signature)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub
}

// encryptAsymmetric encrypts a message with a public key.
func (self *Message) encryptAsymmetric(key *ecdsa.PublicKey) error {
	encrypted, err := crypto.Encrypt(key, self.Raw)
	if err == nil {
		self.Raw = encrypted
	}
	return err
}

// decryptAsymmetric decrypts an encrypted payload with a private key.
func (self *Message) decryptAsymmetric(key *ecdsa.PrivateKey) error {
	decrypted, err := crypto.Decrypt(key, self.Raw)
	if err == nil {
		self.Raw = decrypted
	}
	return err
}

// encryptSymmetric encrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (self *Message) encryptSymmetric(key []byte) (salt []byte, nonce []byte, err error) {
	// todo: delete this block
	// The key argument should be the AES-256 key, 32 bytes
	//if len(key) != aesKeyLength {
	//	glog.V(logger.Error).Infof("AES key size must be %d bytes", aesKeyLength)
	//	err = errors.New("Wrong size of AES key")
	//	return
	//}

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

// decryptSymmetric decrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (self *Message) decryptSymmetric(key []byte, salt []byte, nonce []byte) error {
	// todo: delete this block
	// The key argument should be the AES-256 key, 32 bytes
	//if len(key) != aesKeyLength {
	//	glog.V(logger.Error).Infof("AES key size must be %d bytes", aesKeyLength)
	//	return errors.New("Wrong size of AES key")
	//}

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

// hash calculates the SHA3 checksum of the message flags and payload.
func (self *Message) hash() []byte {
	if self.isSigned() {
		sz := len(self.Raw) - signatureLength
		return crypto.Keccak256(self.Raw[:sz])
	}
	return crypto.Keccak256(self.Raw)
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
func (self *Message) Wrap(pow time.Duration, options Options) (envelope *Envelope, err error) {
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}
	//self.TTL = options.TTL // todo: review
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
	} else if options.Key != nil {
		salt, nonce, err = self.encryptSymmetric(options.Key)
	} else {
		err = errors.New("Unable to encrypt the message: neither Dst nor Key")
	}

	if err == nil {
		envelope = NewEnvelope(options.TTL, options.Topic, salt, nonce, self)
		envelope.Seal(pow)
	}
	return
}
