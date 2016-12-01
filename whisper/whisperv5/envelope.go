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

// Contains the Whisper protocol Envelope element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#envelopes.

package whisperv5

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/rlp"
)

// Envelope represents a clear-text data packet to transmit through the Whisper
// network. Its contents may or may not be encrypted and signed.
type Envelope struct {
	Version  []byte
	Expiry   uint32
	TTL      uint32
	Topic    TopicType
	Salt     []byte
	AESNonce []byte
	Data     []byte
	EnvNonce uint64

	pow  float64     // Message-specific PoW as described in the Whisper specification.
	hash common.Hash // Cached hash of the envelope to avoid rehashing every time.
	// Don't access hash directly, use Hash() function instead.
}

// NewEnvelope wraps a Whisper message with expiration and destination data
// included into an envelope for network forwarding.
func NewEnvelope(ttl uint32, topic TopicType, salt []byte, aesNonce []byte, msg *SentMessage) *Envelope {
	env := Envelope{
		Version:  make([]byte, 1),
		Expiry:   uint32(time.Now().Add(time.Second * time.Duration(ttl)).Unix()),
		TTL:      ttl,
		Topic:    topic,
		Salt:     salt,
		AESNonce: aesNonce,
		Data:     msg.Raw,
		EnvNonce: 0,
	}

	if EnvelopeVersion < 256 {
		env.Version[0] = byte(EnvelopeVersion)
	} else {
		panic("please increase the size of Envelope.Version before releasing this version")
	}

	return &env
}

func (e *Envelope) IsSymmetric() bool {
	return len(e.AESNonce) > 0
}

func (e *Envelope) isAsymmetric() bool {
	return !e.IsSymmetric()
}

func (e *Envelope) Ver() uint64 {
	return bytesToIntLittleEndian(e.Version)
}

// Seal closes the envelope by spending the requested amount of time as a proof
// of work on hashing the data.
func (e *Envelope) Seal(options *MessageParams) {
	var target int
	if options.PoW == 0 {
		// adjust for the duration of Seal() execution only if execution time is predefined unconditionally
		e.Expiry += options.WorkTime
	} else {
		target = e.powToFirstBit(options.PoW)
	}

	buf := make([]byte, 64)
	h := crypto.Keccak256(e.rlpWithoutNonce())
	copy(buf[:32], h)

	finish, bestBit := time.Now().Add(time.Duration(options.WorkTime)*time.Second).UnixNano(), 0
	for nonce := uint64(0); time.Now().UnixNano() < finish; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint64(buf[56:], nonce)
			h = crypto.Keccak256(buf)
			firstBit := common.FirstBitSet(common.BigD(h))
			if firstBit > bestBit {
				e.EnvNonce, bestBit = nonce, firstBit
				if target > 0 && bestBit >= target {
					return
				}
			}
			nonce++
		}
	}
}

func (e *Envelope) PoW() float64 {
	if e.pow == 0 {
		e.calculatePoW(0)
	}
	return e.pow
}

func (e *Envelope) calculatePoW(diff uint32) {
	buf := make([]byte, 64)
	h := crypto.Keccak256(e.rlpWithoutNonce())
	copy(buf[:32], h)
	binary.BigEndian.PutUint64(buf[56:], e.EnvNonce)
	h = crypto.Keccak256(buf)
	firstBit := common.FirstBitSet(common.BigD(h))
	x := math.Pow(2, float64(firstBit))
	x /= float64(len(e.Data)) // we only count e.Data, other variable-sized members are checked in Whisper.add()
	x /= float64(e.TTL + diff)
	e.pow = x
}

func (e *Envelope) powToFirstBit(pow float64) int {
	x := pow
	x *= float64(len(e.Data))
	x *= float64(e.TTL)
	bits := math.Log2(x)
	bits = math.Ceil(bits)
	return int(bits)
}

// rlpWithoutNonce returns the RLP encoded envelope contents, except the nonce.
func (e *Envelope) rlpWithoutNonce() []byte {
	res, _ := rlp.EncodeToBytes([]interface{}{e.Expiry, e.TTL, e.Topic, e.Salt, e.AESNonce, e.Data})
	return res
}

// Hash returns the SHA3 hash of the envelope, calculating it if not yet done.
func (e *Envelope) Hash() common.Hash {
	if (e.hash == common.Hash{}) {
		encoded, _ := rlp.EncodeToBytes(e)
		e.hash = crypto.Keccak256Hash(encoded)
	}
	return e.hash
}

// DecodeRLP decodes an Envelope from an RLP data stream.
func (e *Envelope) DecodeRLP(s *rlp.Stream) error {
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
	if err := rlp.DecodeBytes(raw, (*rlpenv)(e)); err != nil {
		return err
	}
	e.hash = crypto.Keccak256Hash(raw)
	return nil
}

// OpenAsymmetric tries to decrypt an envelope, potentially encrypted with a particular key.
func (e *Envelope) OpenAsymmetric(key *ecdsa.PrivateKey) (*ReceivedMessage, error) {
	message := &ReceivedMessage{Raw: e.Data}
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
func (e *Envelope) OpenSymmetric(key []byte) (msg *ReceivedMessage, err error) {
	msg = &ReceivedMessage{Raw: e.Data}
	err = msg.decryptSymmetric(key, e.Salt, e.AESNonce)
	if err != nil {
		msg = nil
	}
	return msg, err
}

// Open tries to decrypt an envelope, and populates the message fields in case of success.
func (e *Envelope) Open(watcher *Filter) (msg *ReceivedMessage) {
	if e.isAsymmetric() {
		msg, _ = e.OpenAsymmetric(watcher.KeyAsym)
		if msg != nil {
			msg.Dst = &watcher.KeyAsym.PublicKey
		}
	} else if e.IsSymmetric() {
		msg, _ = e.OpenSymmetric(watcher.KeySym)
		if msg != nil {
			msg.SymKeyHash = crypto.Keccak256Hash(watcher.KeySym)
		}
	}

	if msg != nil {
		ok := msg.Validate()
		if !ok {
			return nil
		}
		msg.Topic = e.Topic
		msg.PoW = e.PoW()
		msg.TTL = e.TTL
		msg.Sent = e.Expiry - e.TTL
		msg.EnvelopeHash = e.Hash()
		msg.EnvelopeVersion = e.Ver()
	}
	return msg
}
