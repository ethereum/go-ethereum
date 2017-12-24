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

package whisperv6

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/log"
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
type sentMessage struct {
	Raw []byte
}

// ReceivedMessage represents a data packet to be received through the
// Whisper protocol.
type ReceivedMessage struct {
	Raw []byte

	Payload   []byte
	Padding   []byte
	Signature []byte
	Salt      []byte

	PoW   float64          // Proof of work as described in the Whisper spec
	Sent  uint32           // Time when the message was posted into the network
	TTL   uint32           // Maximum time to live allowed for the message
	Src   *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Dst   *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Topic TopicType

	SymKeyHash   common.Hash // The Keccak256Hash of the key, associated with the Topic
	EnvelopeHash common.Hash // Message envelope hash to act as a unique id
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

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewSentMessage(params *MessageParams) (*sentMessage, error) {
	msg := sentMessage{}
	msg.Raw = make([]byte, 1, len(params.Payload)+len(params.Padding)+signatureLength+padSizeLimit)
	msg.Raw[0] = 0 // set all the flags to zero
	err := msg.appendPadding(params)
	if err != nil {
		return nil, err
	}
	msg.Raw = append(msg.Raw, params.Payload...)
	return &msg, nil
}

// getSizeOfLength returns the number of bytes necessary to encode the entire size padding (including these bytes)
func getSizeOfLength(b []byte) (sz int, err error) {
	sz = intSize(len(b))      // first iteration
	sz = intSize(len(b) + sz) // second iteration
	if sz > 3 {
		err = errors.New("oversized padding parameter")
	}
	return sz, err
}

// sizeOfIntSize returns minimal number of bytes necessary to encode an integer value
func intSize(i int) (s int) {
	for s = 1; i >= 256; s++ {
		i /= 256
	}
	return s
}

// appendPadding appends the pseudorandom padding bytes and sets the padding flag.
// The last byte contains the size of padding (thus, its size must not exceed 256).
func (msg *sentMessage) appendPadding(params *MessageParams) error {
	rawSize := len(params.Payload) + 1
	if params.Src != nil {
		rawSize += signatureLength
	}

	if params.KeySym != nil {
		rawSize += AESNonceLength
	}
	odd := rawSize % padSizeLimit

	if len(params.Padding) != 0 {
		padSize := len(params.Padding)
		padLengthSize, err := getSizeOfLength(params.Padding)
		if err != nil {
			return err
		}
		totalPadSize := padSize + padLengthSize
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint32(buf, uint32(totalPadSize))
		buf = buf[:padLengthSize]
		msg.Raw = append(msg.Raw, buf...)
		msg.Raw = append(msg.Raw, params.Padding...)
		msg.Raw[0] |= byte(padLengthSize) // number of bytes indicating the padding size
	} else if odd != 0 {
		totalPadSize := padSizeLimit - odd
		if totalPadSize > 255 {
			// this algorithm is only valid if padSizeLimit < 256.
			// if padSizeLimit will ever change, please fix the algorithm
			// (please see also ReceivedMessage.extractPadding() function).
			panic("please fix the padding algorithm before releasing new version")
		}
		buf := make([]byte, totalPadSize)
		_, err := crand.Read(buf[1:])
		if err != nil {
			return err
		}
		if totalPadSize > 6 && !validateSymmetricKey(buf) {
			return errors.New("failed to generate random padding of size " + strconv.Itoa(totalPadSize))
		}
		buf[0] = byte(totalPadSize)
		msg.Raw = append(msg.Raw, buf...)
		msg.Raw[0] |= byte(0x1) // number of bytes indicating the padding size
	}
	return nil
}

// sign calculates and sets the cryptographic signature for the message,
// also setting the sign flag.
func (msg *sentMessage) sign(key *ecdsa.PrivateKey) error {
	if isMessageSigned(msg.Raw[0]) {
		// this should not happen, but no reason to panic
		log.Error("failed to sign the message: already signed")
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
func (msg *sentMessage) encryptAsymmetric(key *ecdsa.PublicKey) error {
	if !ValidatePublicKey(key) {
		return errors.New("invalid public key provided for asymmetric encryption")
	}
	encrypted, err := ecies.Encrypt(crand.Reader, ecies.ImportECDSAPublic(key), msg.Raw, nil, nil)
	if err == nil {
		msg.Raw = encrypted
	}
	return err
}

// encryptSymmetric encrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (msg *sentMessage) encryptSymmetric(key []byte) (err error) {
	if !validateSymmetricKey(key) {
		return errors.New("invalid key provided for symmetric encryption")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	// never use more than 2^32 random nonces with a given key
	salt := make([]byte, aesgcm.NonceSize())
	_, err = crand.Read(salt)
	if err != nil {
		return err
	} else if !validateSymmetricKey(salt) {
		return errors.New("crypto/rand failed to generate salt")
	}

	msg.Raw = append(aesgcm.Seal(nil, salt, msg.Raw, nil), salt...)
	return nil
}

// Wrap bundles the message into an Envelope to transmit over the network.
func (msg *sentMessage) Wrap(options *MessageParams) (envelope *Envelope, err error) {
	if options.TTL == 0 {
		options.TTL = DefaultTTL
	}
	if options.Src != nil {
		if err = msg.sign(options.Src); err != nil {
			return nil, err
		}
	}
	if options.Dst != nil {
		err = msg.encryptAsymmetric(options.Dst)
	} else if options.KeySym != nil {
		err = msg.encryptSymmetric(options.KeySym)
	} else {
		err = errors.New("unable to encrypt the message: neither symmetric nor assymmetric key provided")
	}
	if err != nil {
		return nil, err
	}

	envelope = NewEnvelope(options.TTL, options.Topic, msg)
	if err = envelope.Seal(options); err != nil {
		return nil, err
	}
	return envelope, nil
}

// decryptSymmetric decrypts a message with a topic key, using AES-GCM-256.
// nonce size should be 12 bytes (see cipher.gcmStandardNonceSize).
func (msg *ReceivedMessage) decryptSymmetric(key []byte) error {
	// In v6, symmetric messages are expected to contain the 12-byte
	// "salt" at the end of the payload.
	if len(msg.Raw) < AESNonceLength {
		return errors.New("missing salt or invalid payload in symmetric message")
	}
	salt := msg.Raw[len(msg.Raw)-AESNonceLength:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if len(salt) != aesgcm.NonceSize() {
		log.Error("decrypting the message", "AES salt size", len(salt))
		return errors.New("wrong AES salt size")
	}
	decrypted, err := aesgcm.Open(nil, salt, msg.Raw[:len(msg.Raw)-AESNonceLength], nil)
	if err != nil {
		return err
	}
	msg.Raw = decrypted
	msg.Salt = salt
	return nil
}

// decryptAsymmetric decrypts an encrypted payload with a private key.
func (msg *ReceivedMessage) decryptAsymmetric(key *ecdsa.PrivateKey) error {
	decrypted, err := ecies.ImportECDSA(key).Decrypt(crand.Reader, msg.Raw, nil, nil)
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
	sz := int(msg.Raw[0] & paddingMask) // number of bytes indicating the entire size of padding (including these bytes)
	// could be zero -- it means no padding
	if sz != 0 {
		paddingSize = int(bytesToUintLittleEndian(msg.Raw[1 : 1+sz]))
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
		log.Error("failed to recover public key from signature", "err", err)
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
