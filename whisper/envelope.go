package whisper

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/ethutil"
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
		self.hash = H(crypto.Sha3(ethutil.Encode(self)))
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
	copy(d[:32], ethutil.Encode(self.withoutNonce()))

	then := time.Now().Add(dura).UnixNano()
	for n := uint32(0); time.Now().UnixNano() < then; {
		for i := 0; i < 1024; i++ {
			binary.BigEndian.PutUint32(d[60:], n)

			fbs := ethutil.FirstBitSet(ethutil.BigD(crypto.Sha3(d)))
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
	copy(d[:32], ethutil.Encode(self.withoutNonce()))
	binary.BigEndian.PutUint32(d[60:], self.Nonce)
	return ethutil.FirstBitSet(ethutil.BigD(crypto.Sha3(d))) > 0
}

func (self *Envelope) withoutNonce() interface{} {
	return []interface{}{self.Expiry, self.Ttl, ethutil.ByteSliceToInterface(self.Topics), self.Data}
}

func (self *Envelope) RlpData() interface{} {
	return []interface{}{self.Expiry, self.Ttl, ethutil.ByteSliceToInterface(self.Topics), self.Data, self.Nonce}
}

func (self *Envelope) DecodeRLP(s *rlp.Stream) error {
	var extenv struct {
		Expiry uint32
		Ttl    uint32
		Topics [][]byte
		Data   []byte
		Nonce  uint32
	}
	if err := s.Decode(&extenv); err != nil {
		return err
	}

	self.Expiry = extenv.Expiry
	self.Ttl = extenv.Ttl
	self.Topics = extenv.Topics
	self.Data = extenv.Data
	self.Nonce = extenv.Nonce

	// TODO We should use the stream directly here.
	self.hash = H(crypto.Sha3(ethutil.Encode(self)))

	return nil
}
