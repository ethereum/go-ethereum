// Contains the Whisper protocol Message element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#messages.

package whisper

import (
	"crypto/ecdsa"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Message represents an end-user data packet to transmit through the Whisper
// protocol. These are wrapped into Envelopes that need not be understood by
// intermediate nodes, just forwarded.
type Message struct {
	Flags     byte // First bit is signature presence, rest reserved and should be random
	Signature []byte
	Payload   []byte
	Sent      int64

	To   *ecdsa.PublicKey // Message recipient (identity used to decode the message)
	Hash common.Hash      // Message envelope hash to act as a unique id in de-duplication
}

// Options specifies the exact way a message should be wrapped into an Envelope.
type Options struct {
	From   *ecdsa.PrivateKey
	To     *ecdsa.PublicKey
	TTL    time.Duration
	Topics []Topic
}

// NewMessage creates and initializes a non-signed, non-encrypted Whisper message.
func NewMessage(payload []byte) *Message {
	// Construct an initial flag set: no signature, rest random
	flags := byte(rand.Intn(256))
	flags &= ^signatureFlag

	// Assemble and return the message
	return &Message{
		Flags:   flags,
		Payload: payload,
		Sent:    time.Now().Unix(),
	}
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
func (self *Message) Wrap(pow time.Duration, options Options) (*Envelope, error) {
	// Use the default TTL if non was specified
	if options.TTL == 0 {
		options.TTL = DefaultTTL
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

// sign calculates and sets the cryptographic signature for the message , also
// setting the sign flag.
func (self *Message) sign(key *ecdsa.PrivateKey) (err error) {
	self.Flags |= signatureFlag
	self.Signature, err = crypto.Sign(self.hash(), key)
	return
}

// Recover retrieves the public key of the message signer.
func (self *Message) Recover() *ecdsa.PublicKey {
	defer func() { recover() }() // in case of invalid signature

	// Short circuit if no signature is present
	if self.Signature == nil {
		return nil
	}
	// Otherwise try and recover the signature
	pub, err := crypto.SigToPub(self.hash(), self.Signature)
	if err != nil {
		glog.V(logger.Error).Infof("Could not get public key from signature: %v", err)
		return nil
	}
	return pub
}

// encrypt encrypts a message payload with a public key.
func (self *Message) encrypt(key *ecdsa.PublicKey) (err error) {
	self.Payload, err = crypto.Encrypt(key, self.Payload)
	return
}

// decrypt decrypts an encrypted payload with a private key.
func (self *Message) decrypt(key *ecdsa.PrivateKey) (err error) {
	self.Payload, err = crypto.Decrypt(key, self.Payload)
	return
}

// hash calculates the SHA3 checksum of the message flags and payload.
func (self *Message) hash() []byte {
	return crypto.Sha3(append([]byte{self.Flags}, self.Payload...))
}

// bytes flattens the message contents (flags, signature and payload) into a
// single binary blob.
func (self *Message) bytes() []byte {
	return append([]byte{self.Flags}, append(self.Signature, self.Payload...)...)
}
