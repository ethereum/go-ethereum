package whisper

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

type Message struct {
	Flags     byte
	Signature []byte
	Payload   []byte
}

func NewMessage(payload []byte) *Message {
	return &Message{Flags: 0, Payload: payload}
}

func (self *Message) hash() []byte {
	return crypto.Sha3(append([]byte{self.Flags}, self.Payload...))
}

func (self *Message) sign(key *ecdsa.PrivateKey) (err error) {
	self.Flags = 1
	self.Signature, err = crypto.Sign(self.hash(), key)
	return
}

func (self *Message) Recover() *ecdsa.PublicKey {
	return crypto.SigToPub(self.hash(), self.Signature)
}

func (self *Message) Encrypt(from *ecdsa.PrivateKey, to *ecdsa.PublicKey) (err error) {
	err = self.sign(from)
	if err != nil {
		return err
	}

	self.Payload, err = crypto.Encrypt(to, self.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (self *Message) Bytes() []byte {
	return append([]byte{self.Flags}, append(self.Signature, self.Payload...)...)
}

type Opts struct {
	From   *ecdsa.PrivateKey
	To     *ecdsa.PublicKey
	Ttl    time.Duration
	Topics [][]byte
}

func (self *Message) Seal(pow time.Duration, opts Opts) (*Envelope, error) {
	if opts.To != nil && opts.From != nil {
		if err := self.Encrypt(opts.From, opts.To); err != nil {
			return nil, err
		}
	}

	envelope := NewEnvelope(DefaultTtl, opts.Topics, self)
	envelope.Seal(pow)

	return envelope, nil
}
