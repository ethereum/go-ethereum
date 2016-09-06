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

// Contains the Whisper protocol Envelope element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#envelopes.

package whisper5

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/rlp"
)

// Envelope represents a clear-text data packet to transmit through the Whisper
// network. Its contents may or may not be encrypted and signed.
type Envelope struct {
	Expiry   uint32
	TTL      uint32
	Topic    TopicType
	Salt     []byte
	AESNonce []byte
	Data     []byte
	EnvNonce uint64

	hash common.Hash // Cached hash of the envelope to avoid rehashing every time
	pow  int         // Message-specific PoW as described in the Whisper specification
}

// NewEnvelope wraps a Whisper message with expiration and destination data
// included into an envelope for network forwarding.
func NewEnvelope(ttl time.Duration, topic TopicType, salt []byte, aesNonce []byte, msg *SentMessage) *Envelope {
	return &Envelope{
		Expiry:   uint32(time.Now().Add(ttl).Unix()),
		TTL:      uint32(ttl.Seconds()),
		Topic:    topic,
		Salt:     salt,
		AESNonce: aesNonce,
		Data:     msg.Raw,
		EnvNonce: 0,
	}
}

func (self *Envelope) isSymmetric() bool {
	return self.AESNonce != nil
}

func (self *Envelope) isAsymmetric() bool {
	return !self.isSymmetric()
}

// Seal closes the envelope by spending the requested amount of time as a proof
// of work on hashing the data.
func (self *Envelope) Seal(work time.Duration) {
	self.Expiry += uint32(work.Seconds()) // adjust for the duration of Seal() execution

	buf := make([]byte, 64)
	h := crypto.Keccak256(self.rlpWithoutNonce())
	copy(buf[:32], h)

	finish, bestBit := time.Now().Add(work).UnixNano(), 0
	for nonce := uint64(0); time.Now().UnixNano() < finish; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint64(buf[56:], nonce)
			h = crypto.Keccak256(buf)
			firstBit := common.FirstBitSet(common.BigD(h))
			if firstBit > bestBit {
				self.EnvNonce, bestBit = nonce, firstBit
			}
			nonce++
		}
	}
	//return bestBit // todo: uncomment?
}

// rlpWithoutNonce returns the RLP encoded envelope contents, except the nonce.
func (self *Envelope) rlpWithoutNonce() []byte {
	enc, _ := rlp.EncodeToBytes([]interface{}{self.Expiry, self.TTL, self.Topic, self.Salt, self.AESNonce, self.Data})
	return enc
}

// Hash returns the SHA3 hash of the envelope, calculating it if not yet done.
func (self *Envelope) Hash() common.Hash {
	if (self.hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(self)
		self.hash = crypto.Keccak256Hash(enc)
	}
	return self.hash
}

// DecodeRLP decodes an Envelope from an RLP data stream.
func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	// The decoding of Envelope uses the struct fields but also needs
	// to compute the hash of the whole RLP-encoded envelope. This
	// type has the same structure as Envelope but is not an
	// rlp.Decoder (does not implement DecodeRLP function).
	// Only public members will be encoded.
	type rlpenv Envelope
	if err := rlp.DecodeBytes(raw, (*rlpenv)(self)); err != nil {
		return err
	}
	self.hash = crypto.Keccak256Hash(raw)
	return nil
}

// OpenAsymmetric tries to decrypt an envelope, potentially encrypted with a particular key.
func (self *Envelope) OpenAsymmetric(key *ecdsa.PrivateKey) (*ReceivedMessage, error) {
	message := &ReceivedMessage{Raw: self.Data}
	err := message.decryptAsymmetric(key)
	switch err {
	case nil:
		return message, nil
	case ecies.ErrInvalidPublicKey: // addressed to somebody else
		return nil, err
	default:
		return nil, fmt.Errorf("unable to open envelope, decrypt failed: %v", err)
	}
}

// OpenSymmetric tries to decrypt an envelope, potentially encrypted with a particular key.
func (self *Envelope) OpenSymmetric(key []byte) (msg *ReceivedMessage, err error) {
	msg = &ReceivedMessage{Raw: self.Data}
	err = msg.decryptSymmetric(key, self.Salt, self.AESNonce)
	if err != nil {
		msg = nil
	}
	return
}

// Open tries to decrypt an envelope, and populates the message fields in case of success.
func (self *Envelope) Open(watcher *Filter) (msg *ReceivedMessage) {
	if self.isAsymmetric() {
		msg, _ = self.OpenAsymmetric(watcher.KeyAsym)
		if msg != nil {
			msg.Dst = watcher.Dst
		}
	} else if self.isSymmetric() {
		msg, _ = self.OpenSymmetric(watcher.KeySym)
		if msg != nil {
			msg.TopicKeyHash = crypto.Keccak256Hash(watcher.KeySym)
		}
	}

	if msg != nil {
		ok := msg.Validate()
		if !ok {
			return nil
		}
		msg.Topic = self.Topic
		msg.PoW = self.pow
		msg.TTL = self.TTL
		msg.Sent = self.Expiry - self.TTL // todo: review
		msg.EnvelopeHash = self.hash
	}
	return msg
}
