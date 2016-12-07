// Copyright 2016 The go-ethereum Authors
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

// Contains the Whisper protocol Message element.

package whisperv5

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	mrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/crypto/pbkdf2"
)

// Options specifies the exact way a message should be wrapped into an Envelope.
type MessageParams struct {
	TTL      uint32
	Src      *ecdsa.PrivateKey
	Dst      *ecdsa.PublicKey
	KeySym   []byte
	Topic    TopicType
	WorkTime uint32
	PoW      float64
	Payload  []byte
	Padding  []byte
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

	PoW   float64          // Proof of work as described in the Whisper spec
	Sent  uint32           // Time when the message was posted into the network
	TTL   uint32           // Maximum time to live allowed for the message
	Src   *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Dst   *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Topic TopicType

	SymKeyHash      common.Hash // The Keccak256Hash of the key, associated with the Topic
	EnvelopeHash    common.Hash // Message envelope hash to act as a unique id
	EnvelopeVersion uint64
}

func isMessageSigned(flags byte) bool {
	return (flags & signatureFlag) != 0
}

func (msg *ReceivedMessage) isSymmetricEncryption() bool {
	return msg.SymKeyHash != common.Hash{}
}

func (msg *ReceivedMessage) isAsymmetricEncryption() bool {
	return msg.Dst != nil
}

func DeriveOneTimeKey(key []byte, salt []byte, version uint64) ([]byte, error) {
	if version == 0 {
		derivedKey := pbkdf2.Key(key, salt, 8, aesKeyLength, sha256.New)
		return derivedKey, nil
	} else {
		return nil, unknownVersionError(version)
	}
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewSentMessage(params *MessageParams) *SentMessage {
	msg := SentMessage{}
	msg.Raw = make([]byte, 1, len(params.Payload)+len(params.Payload)+signatureLength+padSizeLimitUpper)
	msg.Raw[0] = 0 // set all the flags to zero
	msg.appendPadding(params)
	msg.Raw = append(msg.Raw, params.Payload...)
	return &msg
}

// appendPadding appends the pseudorandom padding bytes and sets the padding flag.
// The last byte contains the size of padding (thus, its size must not exceed 256).
func (msg *SentMessage) appendPadding(params *MessageParams) {
	total := len(params.Payload) + 1
	if params.Src != nil {
		total += signatureLength
	}
	padChunk := padSizeLimitUpper
	if total <= padSizeLimitLower {
		padChunk = padSizeLimitLower
	}
	odd := total % padChunk
	if odd > 0 {
		padSize := padChunk - odd
		if padSize > 255 {
			// this algorithm is only valid if padSizeLimitUpper <= 256.
			// if padSizeLimitUpper will ever change, please fix the algorithm
			// (for more information see ReceivedMessage.extractPadding() function).
			panic("please fix the padding algorithm before releasing new version")
		}
		buf := make([]byte, padSize)
		randomize(buf[1:])
		buf[0] = byte(padSize)
		if params.Padding != nil {
			copy(buf[1:], params.Padding)
		}
		msg.Raw = append(msg.Raw, buf...)
		msg.Raw[0] |= byte(0x1) // number of bytes indicating the padding size
	}
}

// sign calculates and sets the cryptographic signature for the message,
// also setting the sign flag.
func (msg *SentMessage) sign(key *ecdsa.PrivateKey) error {
	if isMessageSigned(msg.Raw[0]) {
		// this should not happen, but no reason to panic
		glog.V(logger.Error).Infof("Trying to sign a message which was already signed")
		return nil
	}

	msg.Raw[0] |= signatureFlag
	hash := crypto.Keccak256(msg.Raw)
	signature, err := crypto.Sign(hash, key)
	if err != nil {
		msg.Raw[0] &= ^signatureFlag // clear the flag
		return err
	}
	msg.Raw = append(msg.Raw, signature...)
	return nil
}

// encryptAsymmetric encrypts a message with a public key.
func (msg *SentMessage) encryptAsymmetric(key *ecdsa.PublicKey) error {
	if !ValidatePublicKey(key) {
		return fmt.Errorf("Invalid public key provided for asymmetric encryption")
	}
	encrypted, err := crypto.Encrypt(key, msg.Raw)
	if err == nil {
		msg.Raw = encrypted
	}
	return err
}

// encryptSymmetric encrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (msg *SentMessage) encryptSymmetric(key []byte) (salt []byte, nonce []byte, err error) {
	if !validateSymmetricKey(key) {
		return nil, nil, errors.New("invalid key provided for symmetric encryption")
	}

	salt = make([]byte, saltLength)
	_, err = crand.Read(salt)
	if err != nil {
		return nil, nil, err
	} else if !validateSymmetricKey(salt) {
		return nil, nil, errors.New("crypto/rand failed to generate salt")
	}

	derivedKey, err := DeriveOneTimeKey(key, salt, EnvelopeVersion)
	if err != nil {
		return nil, nil, err
	}
	if !validateSymmetricKey(derivedKey) {
		return nil, nil, errors.New("failed to derive one-time key")
	}
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	// never use more than 2^32 random nonces with a given key
	nonce = make([]byte, aesgcm.NonceSize())
	_, err = crand.Read(nonce)
	if err != nil {
		return nil, nil, err
	} else if !validateSymmetricKey(nonce) {
		return nil, nil, errors.New("crypto/rand failed to generate nonce")
	}

	msg.Raw = aesgcm.Seal(nil, nonce, msg.Raw, nil)
	return salt, nonce, nil
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
func (msg *SentMessage) Wrap(options *MessageParams) (envelope *Envelope, err error) {
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}
	if options.Src != nil {
		err = msg.sign(options.Src)
		if err != nil {
			return nil, err
		}
	}
	if len(msg.Raw) > MaxMessageLength {
		glog.V(logger.Error).Infof("Message size must not exceed %d bytes", MaxMessageLength)
		return nil, errors.New("Oversized message")
	}
	var salt, nonce []byte
	if options.Dst != nil {
		err = msg.encryptAsymmetric(options.Dst)
	} else if options.KeySym != nil {
		salt, nonce, err = msg.encryptSymmetric(options.KeySym)
	} else {
		err = errors.New("Unable to encrypt the message: neither Dst nor Key")
	}

	if err != nil {
		return nil, err
	}

	envelope = NewEnvelope(options.TTL, options.Topic, salt, nonce, msg)
	err = envelope.Seal(options)
	if err != nil {
		return nil, err
	}

	return envelope, nil
}

// decryptSymmetric decrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (msg *ReceivedMessage) decryptSymmetric(key []byte, salt []byte, nonce []byte) error {
	derivedKey, err := DeriveOneTimeKey(key, salt, msg.EnvelopeVersion)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if len(nonce) != aesgcm.NonceSize() {
		info := fmt.Sprintf("Wrong AES nonce size - want: %d, got: %d", len(nonce), aesgcm.NonceSize())
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}
	decrypted, err := aesgcm.Open(nil, nonce, msg.Raw, nil)
	if err != nil {
		return err
	}
	msg.Raw = decrypted
	return nil
}

// decryptAsymmetric decrypts an encrypted payload with a private key.
func (msg *ReceivedMessage) decryptAsymmetric(key *ecdsa.PrivateKey) error {
	decrypted, err := crypto.Decrypt(key, msg.Raw)
	if err == nil {
		msg.Raw = decrypted
	}
	return err
}

// Validate checks the validity and extracts the fields in case of success
func (msg *ReceivedMessage) Validate() bool {
	end := len(msg.Raw)
	if end < 1 {
		return false
	}

	if isMessageSigned(msg.Raw[0]) {
		end -= signatureLength
		if end <= 1 {
			return false
		}
		msg.Signature = msg.Raw[end:]
		msg.Src = msg.SigToPubKey()
		if msg.Src == nil {
			return false
		}
	}

	padSize, ok := msg.extractPadding(end)
	if !ok {
		return false
	}

	msg.Payload = msg.Raw[1+padSize : end]
	return true
}

// extractPadding extracts the padding from raw message.
// although we don't support sending messages with padding size
// exceeding 255 bytes, such messages are perfectly valid, and
// can be successfully decrypted.
func (msg *ReceivedMessage) extractPadding(end int) (int, bool) {
	paddingSize := 0
	sz := int(msg.Raw[0] & paddingMask) // number of bytes containing the entire size of padding, could be zero
	if sz != 0 {
		paddingSize = int(bytesToIntLittleEndian(msg.Raw[1 : 1+sz]))
		if paddingSize < sz || paddingSize+1 > end {
			return 0, false
		}
		msg.Padding = msg.Raw[1+sz : 1+paddingSize]
	}
	return paddingSize, true
}

// Recover retrieves the public key of the message signer.
func (msg *ReceivedMessage) SigToPubKey() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	pub, err := crypto.SigToPub(msg.hash(), msg.Signature)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub
}

// hash calculates the SHA3 checksum of the message flags, payload and padding.
func (msg *ReceivedMessage) hash() []byte {
	if isMessageSigned(msg.Raw[0]) {
		sz := len(msg.Raw) - signatureLength
		return crypto.Keccak256(msg.Raw[:sz])
	}
	return crypto.Keccak256(msg.Raw)
}

// rand.Rand provides a Read method in Go 1.7 and later,
// but we can't use it yet.
func randomize(b []byte) {
	cnt := 0
	val := mrand.Int63()
	for n := 0; n < len(b); n++ {
		b[n] = byte(val)
		val >>= 8
		cnt++
		if cnt >= 7 {
			cnt = 0
			val = mrand.Int63()
		}
	}
}
