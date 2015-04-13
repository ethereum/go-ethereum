// Contains the Whisper protocol Envelope element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#envelopes.

package whisper

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Envelope represents a clear-text data packet to transmit through the Whisper
// network. Its contents may or may not be encrypted and signed.
type Envelope struct {
	Expiry uint32 // Whisper protocol specifies int32, really should be int64
	TTL    uint32 // ^^^^^^
	Topics [][]byte
	Data   []byte
	Nonce  uint32

	hash common.Hash
}

// NewEnvelope wraps a Whisper message with expiration and destination data
// included into an envelope for network forwarding.
func NewEnvelope(ttl time.Duration, topics [][]byte, msg *Message) *Envelope {
	return &Envelope{
		Expiry: uint32(time.Now().Add(ttl).Unix()),
		TTL:    uint32(ttl.Seconds()),
		Topics: topics,
		Data:   msg.bytes(),
		Nonce:  0,
	}
}

// Seal closes the envelope by spending the requested amount of time as a proof
// of work on hashing the data.
func (self *Envelope) Seal(pow time.Duration) {
	d := make([]byte, 64)
	copy(d[:32], self.rlpWithoutNonce())

	finish, bestBit := time.Now().Add(pow).UnixNano(), 0
	for nonce := uint32(0); time.Now().UnixNano() < finish; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], nonce)

			firstBit := common.FirstBitSet(common.BigD(crypto.Sha3(d)))
			if firstBit > bestBit {
				self.Nonce, bestBit = nonce, firstBit
			}
			nonce++
		}
	}
}

// valid checks whether the claimed proof of work was indeed executed.
// TODO: Is this really useful? Isn't this always true?
func (self *Envelope) valid() bool {
	d := make([]byte, 64)
	copy(d[:32], self.rlpWithoutNonce())
	binary.BigEndian.PutUint32(d[60:], self.Nonce)

	return common.FirstBitSet(common.BigD(crypto.Sha3(d))) > 0
}

// rlpWithoutNonce returns the RLP encoded envelope contents, except the nonce.
func (self *Envelope) rlpWithoutNonce() []byte {
	enc, _ := rlp.EncodeToBytes([]interface{}{self.Expiry, self.TTL, self.Topics, self.Data})
	return enc
}

// Open extracts the message contained within a potentially encrypted envelope.
func (self *Envelope) Open(key *ecdsa.PrivateKey) (msg *Message, err error) {
	// Split open the payload into a message construct
	data := self.Data

	message := &Message{
		Flags: data[0],
	}
	data = data[1:]

	if message.Flags&signatureFlag == signatureFlag {
		if len(data) < signatureLength {
			return nil, fmt.Errorf("unable to open envelope. First bit set but len(data) < len(signature)")
		}
		message.Signature, data = data[:signatureLength], data[signatureLength:]
	}
	message.Payload = data

	// Decrypt the message, if requested
	if key == nil {
		return message, nil
	}
	switch message.decrypt(key) {
	case nil:
		return message, nil

	default:
		return nil, fmt.Errorf("unable to open envelope, decrypt failed: %v", err)
	}
}

// Hash returns the SHA3 hash of the envelope, calculating it if not yet done.
func (self *Envelope) Hash() common.Hash {
	if (self.hash == common.Hash{}) {
		enc, _ := rlp.EncodeToBytes(self)
		self.hash = crypto.Sha3Hash(enc)
	}
	return self.hash
}

// rlpenv is an Envelope but is not an rlp.Decoder.
// It is used for decoding because we need to
type rlpenv Envelope

// DecodeRLP decodes an Envelope from an RLP data stream.
func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	if err := rlp.DecodeBytes(raw, (*rlpenv)(self)); err != nil {
		return err
	}
	self.hash = crypto.Sha3Hash(raw)
	return nil
}
