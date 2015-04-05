package whisper

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type Message struct {
	Flags     byte
	Signature []byte
	Payload   []byte
	Sent      int64

	To *ecdsa.PublicKey
}

func NewMessage(payload []byte) *Message {
	return &Message{Flags: 0, Payload: payload, Sent: time.Now().Unix()}
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
	defer func() { recover() }() // in case of invalid sig
	pub, err := crypto.SigToPub(self.hash(), self.Signature)
	if err != nil {
		glog.V(0).Infof("Could not get pubkey from signature: ", err)
		return nil
	}
	return pub
}

func (self *Message) Encrypt(to *ecdsa.PublicKey) (err error) {
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
	if opts.From != nil {
		err := self.sign(opts.From)
		if err != nil {
			return nil, err
		}
	}

	if opts.To != nil {
		err := self.Encrypt(opts.To)
		if err != nil {
			return nil, err
		}
	}

	if opts.Ttl == 0 {
		opts.Ttl = DefaultTtl
	}

	envelope := NewEnvelope(opts.Ttl, opts.Topics, self)
	envelope.Seal(pow)

	return envelope, nil
}
