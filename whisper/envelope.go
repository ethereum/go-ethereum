package whisper

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

const (
	DefaultPow = 50 * time.Millisecond
)

type Envelope struct {
	Expiry uint32 // Whisper protocol specifies int32, really should be int64
	Ttl    uint32 // ^^^^^^
	Topics [][]byte
	Data   []byte
	Nonce  uint32

	hash Hash
}

func (self *Envelope) Hash() Hash {
	if self.hash == EmptyHash {
		enc, _ := rlp.EncodeToBytes(self)
		self.hash = H(crypto.Sha3(enc))
	}
	return self.hash
}

func NewEnvelope(ttl time.Duration, topics [][]byte, data *Message) *Envelope {
	exp := time.Now().Add(ttl)

	return &Envelope{uint32(exp.Unix()), uint32(ttl.Seconds()), topics, data.Bytes(), 0, Hash{}}
}

func (self *Envelope) Seal(pow time.Duration) {
	self.proveWork(pow)
}

func (self *Envelope) Open(prv *ecdsa.PrivateKey) (msg *Message, err error) {
	data := self.Data
	var message Message
	dataStart := 1
	if data[0] > 0 {
		if len(data) < 66 {
			return nil, fmt.Errorf("unable to open envelope. First bit set but len(data) < 66")
		}
		dataStart = 66
		message.Flags = data[0]
		message.Signature = data[1:66]
	}

	payload := data[dataStart:]
	if prv != nil {
		message.Payload, err = crypto.Decrypt(prv, payload)
		switch err {
		case nil: // OK
		case ecies.ErrInvalidPublicKey: // Payload isn't encrypted
			message.Payload = payload
			return &message, err
		default:
			return nil, fmt.Errorf("unable to open envelope. Decrypt failed: %v", err)
		}
	}

	return &message, nil
}

func (self *Envelope) proveWork(dura time.Duration) {
	var bestBit int
	d := make([]byte, 64)
	enc, _ := rlp.EncodeToBytes(self.withoutNonce())
	copy(d[:32], enc)

	then := time.Now().Add(dura).UnixNano()
	for n := uint32(0); time.Now().UnixNano() < then; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], n)

			fbs := common.FirstBitSet(common.BigD(crypto.Sha3(d)))
			if fbs > bestBit {
				bestBit = fbs
				self.Nonce = n
			}

			n++
		}
	}
}

func (self *Envelope) valid() bool {
	d := make([]byte, 64)
	enc, _ := rlp.EncodeToBytes(self.withoutNonce())
	copy(d[:32], enc)
	binary.BigEndian.PutUint32(d[60:], self.Nonce)
	return common.FirstBitSet(common.BigD(crypto.Sha3(d))) > 0
}

func (self *Envelope) withoutNonce() interface{} {
	return []interface{}{self.Expiry, self.Ttl, self.Topics, self.Data}
}

// rlpenv is an Envelope but is not an rlp.Decoder.
// It is used for decoding because we need to
type rlpenv Envelope

func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	if err := rlp.DecodeBytes(raw, (*rlpenv)(self)); err != nil {
		return err
	}
	self.hash = H(crypto.Sha3(raw))
	return nil
}
