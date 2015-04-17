// Contains the external API to the whisper sub-protocol.

package xeth

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/whisper"
)

var qlogger = logger.NewLogger("XSHH")

// Whisper represents the API wrapper around the internal whisper implementation.
type Whisper struct {
	*whisper.Whisper
}

// NewWhisper wraps an internal whisper client into an external API version.
func NewWhisper(w *whisper.Whisper) *Whisper {
	return &Whisper{w}
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (self *Whisper) NewIdentity() string {
	identity := self.Whisper.NewIdentity()
	return common.ToHex(crypto.FromECDSAPub(&identity.PublicKey))
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (self *Whisper) HasIdentity(key string) bool {
	return self.Whisper.HasIdentity(crypto.ToECDSAPub(common.FromHex(key)))
}

// Post injects a message into the whisper network for distribution.
func (self *Whisper) Post(payload string, to, from string, topics []string, priority, ttl uint32) error {
	// Construct the whisper message and transmission options
	message := whisper.NewMessage(common.FromHex(payload))
	options := whisper.Options{
		To:     crypto.ToECDSAPub(common.FromHex(to)),
		TTL:    time.Duration(ttl) * time.Second,
		Topics: whisper.NewTopicsFromStrings(topics...),
	}
	if len(from) != 0 {
		if key := self.Whisper.GetIdentity(crypto.ToECDSAPub(common.FromHex(from))); key != nil {
			options.From = key
		} else {
			return fmt.Errorf("unknown identity to send from: %s", from)
		}
	}
	// Wrap and send the message
	pow := time.Duration(priority) * time.Millisecond
	envelope, err := message.Wrap(pow, options)
	if err != nil {
		return err
	}
	if err := self.Whisper.Send(envelope); err != nil {
		return err
	}
	return nil
}

// Watch installs a new message handler to run in case a matching packet arrives
// from the whisper network.
func (self *Whisper) Watch(to, from string, topics []string, fn func(WhisperMessage)) int {
	filter := whisper.Filter{
		To:     crypto.ToECDSAPub(common.FromHex(to)),
		From:   crypto.ToECDSAPub(common.FromHex(from)),
		Topics: whisper.NewTopicsFromStrings(topics...),
	}
	filter.Fn = func(message *whisper.Message) {
		fn(NewWhisperMessage(message))
	}
	return self.Whisper.Watch(filter)
}

// Messages retrieves all the currently pooled messages matching a filter id.
func (self *Whisper) Messages(id int) []WhisperMessage {
	pool := self.Whisper.Messages(id)

	messages := make([]WhisperMessage, len(pool))
	for i, message := range pool {
		messages[i] = NewWhisperMessage(message)
	}
	return messages
}
