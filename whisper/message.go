// Contains the Whisper protocol Message element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#messages.

package whisper

import (
	"crypto/ecdsa"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Message represents an end-user data packet to trasmit through the Whisper
// protocol. These are wrapped into Envelopes that need not be understood by
// intermediate nodes, just forwarded.
type Message struct {
	Flags     byte // First bit is signature presence, rest reserved and should be random
	Signature []byte
	Payload   []byte
	Sent      int64

	To *ecdsa.PublicKey
}

// Options specifies the exact way a message should be wrapped into an Envelope.
type Options struct {
	From   *ecdsa.PrivateKey
	To     *ecdsa.PublicKey
	TTL    time.Duration
	Topics [][]byte
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewMessage(payload []byte) *Message {
	// Construct an initial flag set: bit #1 = 0 (no signature), rest random
	flags := byte(rand.Intn(128))

	// Assemble and return the message
	return &Message{
		Flags:   flags,
		Payload: payload,
		Sent:    time.Now().Unix(),
	}
}

// Wrap bundles the message into an Envelope to transmit over the network.
//
// Pov (Proof Of Work) controls how much time to spend on hashing the message,
// inherently controlling its priority through the network (smaller hash, bigger
// priority).
//
// The user can control the amount of identity, privacy and encryption through
// the options parameter as follows:
//   - options.From == nil && options.To == nil: anonymous broadcast
//   - options.From != nil && options.To == nil: signed broadcast (known sender)
//   - options.From == nil && options.To != nil: encrypted anonymous message
//   - options.From != nil && options.To != nil: encrypted signed message
func (self *Message) Wrap(pow time.Duration, options Options) (*Envelope, error) {
	// Use the default TTL if non was specified
	if options.TTL == 0 {
		options.TTL = DefaultTimeToLive
	}
	// Sign and encrypt the message if requested
	if options.From != nil {
		if err := self.sign(options.From); err != nil {
			return nil, err
		}
	}
	if options.To != nil {
		if err := self.encrypt(options.To); err != nil {
			return nil, err
		}
	}
	// Wrap the processed message, seal it and return
	envelope := NewEnvelope(options.TTL, options.Topics, self)
	envelope.Seal(pow)

	return envelope, nil
}

// Sign calculates and sets the cryptographic signature for the message , also
// setting the sign flag.
func (self *Message) sign(key *ecdsa.PrivateKey) (err error) {
	self.Flags |= 1 << 7
	self.Signature, err = crypto.Sign(self.hash(), key)
	return
}

// Recover retrieves the public key of the message signer.
func (self *Message) Recover() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	pub, err := crypto.SigToPub(self.hash(), self.Signature)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub
}

// Encrypt encrypts a message payload with a public key.
func (self *Message) encrypt(to *ecdsa.PublicKey) (err error) {
	self.Payload, err = crypto.Encrypt(to, self.Payload)
	return
}

// Hash calculates the SHA3 checksum of the message flags and payload.
func (self *Message) hash() []byte {
	return crypto.Sha3(append([]byte{self.Flags}, self.Payload...))
}

// Bytes flattens the message contents (flags, signature and payload) into a
// single binary blob.
func (self *Message) bytes() []byte {
	return append([]byte{self.Flags}, append(self.Signature, self.Payload...)...)
}
