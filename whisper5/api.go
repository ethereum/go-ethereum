// Copyright 2015 The go-ethereum Authors
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

package whisper5

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicWhisperAPI provides the whisper RPC service.
type PublicWhisperAPI struct {
	whisper    *Whisper
	messagesMu sync.RWMutex
	messages   map[int]*whisperFilter
}

type whisperOfflineError struct{}

func (e *whisperOfflineError) Error() string {
	return "whisper is offline"
}

// whisperOffLineErr is returned when the node doesn't offer the shh service.
var whisperOffLineErr = new(whisperOfflineError)

// NewPublicWhisperAPI create a new RPC whisper service.
func NewPublicWhisperAPI(w *Whisper) *PublicWhisperAPI {
	return &PublicWhisperAPI{whisper: w, messages: make(map[int]*whisperFilter)}
}

// Version returns the Whisper version this node offers.
func (self *PublicWhisperAPI) Version() (*rpc.HexNumber, error) {
	if self.whisper == nil {
		return rpc.NewHexNumber(0), whisperOffLineErr
	}
	return rpc.NewHexNumber(self.whisper.Version()), nil
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (self *PublicWhisperAPI) HasIdentity(identity string) (bool, error) {
	if self.whisper == nil {
		return false, whisperOffLineErr
	}
	return self.whisper.HasIdentity(crypto.ToECDSAPub(common.FromHex(identity))), nil
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (self *PublicWhisperAPI) NewIdentity() (string, error) {
	if self.whisper == nil {
		return "", whisperOffLineErr
	}

	identity := self.whisper.NewIdentity()
	return common.ToHex(crypto.FromECDSAPub(&identity.PublicKey)), nil
}

// NewWhisperFilter creates and registers a new message filter to watch for inbound whisper messages.
func (self *PublicWhisperAPI) NewFilter(args WhisperFilterArgs) (*rpc.HexNumber, error) {
	if self.whisper == nil {
		return nil, whisperOffLineErr
	}

	var id int
	filter := Filter{
		Src:    crypto.ToECDSAPub(common.FromHex(args.From)),
		Dst:    crypto.ToECDSAPub(common.FromHex(args.To)),
		KeySym: common.FromHex(args.KeySym),
		PoW:    args.PoW,
		Fn: func(message *ReceivedMessage) {
			wmsg := NewWhisperMessage(message)
			self.messagesMu.RLock() // Only read lock to the filter pool
			defer self.messagesMu.RUnlock()
			if self.messages[id] != nil {
				self.messages[id].insert(wmsg)
			}
			// todo: review after api is ready
		},
	}

	if len(args.To) == 0 && len(args.KeySym) == 0 {
		info := "Filter must contain at least one key"
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.Topics) > 0 {
		for _, s := range args.Topics {
			t := common.FromHex(s)
			filter.Topics = append(filter.Topics, BytesToTopic(t))
		}
	} else {
		// if Topics are not provided, just use the default derivation function
		t := DeriveTopicFromSymmetricKey(common.FromHex(args.KeySym))
		filter.Topics = append(filter.Topics, t)
	}

	id = self.whisper.Watch(&filter)

	self.messagesMu.Lock()
	self.messages[id] = newWhisperFilter(id, self.whisper)
	self.messagesMu.Unlock()

	return rpc.NewHexNumber(id), nil
}

// UninstallFilter disables and removes an existing filter.
func (self *PublicWhisperAPI) UninstallFilter(filterId rpc.HexNumber) bool {
	self.messagesMu.Lock()
	defer self.messagesMu.Unlock()

	if _, ok := self.messages[filterId.Int()]; ok {
		delete(self.messages, filterId.Int())
		return true
	}
	return false
}

// GetFilterChanges retrieves all the new messages matched by a filter since the last retrieval.
func (self *PublicWhisperAPI) GetFilterChanges(filterId rpc.HexNumber) []WhisperMessage {
	self.messagesMu.RLock()
	defer self.messagesMu.RUnlock()

	if self.messages[filterId.Int()] != nil {
		if changes := self.messages[filterId.Int()].retrieve(); changes != nil {
			return changes
		}
	}
	return toWhisperMessages(nil)
}

// GetMessages retrieves all the known messages that match a specific filter.
func (self *PublicWhisperAPI) GetMessages(filterId rpc.HexNumber) []WhisperMessage {
	// Retrieve all the cached messages matching a specific, existing filter
	self.messagesMu.RLock()
	defer self.messagesMu.RUnlock()

	var messages []*ReceivedMessage
	if self.messages[filterId.Int()] != nil {
		messages = self.messages[filterId.Int()].messages()
	}

	return toWhisperMessages(messages)
}

// toWhisperMessages converts a Whisper message to a RPC whisper message.
func toWhisperMessages(messages []*ReceivedMessage) []WhisperMessage {
	msgs := make([]WhisperMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = NewWhisperMessage(msg)
	}
	return msgs
}

// Post injects a message into the whisper network for distribution.
func (self *PublicWhisperAPI) Post(args PostArgs) (bool, error) {
	if self.whisper == nil {
		return false, whisperOffLineErr
	}

	// construct whisper message with transmission options
	message := NewSentMessage(common.FromHex(args.Payload))
	options := Options{
		TTL:    time.Duration(args.TTL) * time.Second,
		Dst:    crypto.ToECDSAPub(common.FromHex(args.To)),
		KeySym: common.FromHex(args.Key),
		Topic:  BytesToTopic(common.FromHex(args.Topic)),
		Pad:    common.FromHex(args.Padding),
	}

	// set sender identity
	if len(args.From) > 0 {
		if privateKey := self.whisper.GetIdentity(crypto.ToECDSAPub(common.FromHex(args.From))); privateKey != nil {
			options.Src = privateKey
		} else {
			return false, fmt.Errorf("unknown identity to send from: %s", args.From)
		}
	}

	// Wrap and send the message
	options.Work = time.Duration(options.Work) * time.Millisecond
	envelope, err := message.Wrap(options)
	if err != nil {
		return false, err
	}

	return true, self.whisper.Send(envelope)
}

type PostArgs struct {
	TTL     int64  `json:"ttl"`
	From    string `json:"from"`
	To      string `json:"to"`
	Key     string `json:"key"`
	Topic   string `json:"topic"`
	Padding string `json:"padding"`
	Payload string `json:"payload"`
	Work    int64  `json:"work"` // todo: review this field usage
	PoW     int64  `json:"pow"`  // todo: review this field usage
}

func (args *PostArgs) UnmarshalJSON(data []byte) (err error) {
	var obj struct {
		TTL     int64  `json:"ttl"`
		From    string `json:"from"`
		To      string `json:"to"`
		Key     string `json:"key"`
		Topic   string `json:"topic"`
		Payload string `json:"payload"`
		Padding string `json:"padding"`
		Work    int64  `json:"work"`
		PoW     int64  `json:"pow"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	args.TTL = obj.TTL
	args.From = obj.From
	args.To = obj.To
	args.Key = obj.Key
	args.Topic = obj.Topic
	args.Payload = obj.Payload
	args.Padding = obj.Padding
	args.Work = obj.Work
	args.PoW = obj.PoW

	return nil
}

// WhisperMessage is the RPC representation of a whisper message.
type WhisperMessage struct {
	ref *ReceivedMessage

	Payload string `json:"payload"`
	Padding string `json:"padding"`
	From    string `json:"from"`
	To      string `json:"to"`
	Sent    int64  `json:"sent"`
	TTL     int64  `json:"ttl"`
	PoW     int64  `json:"pow"`
	Hash    string `json:"hash"`
}

// NewWhisperMessage converts an internal message into an API version.
func NewWhisperMessage(message *ReceivedMessage) WhisperMessage {
	return WhisperMessage{
		ref: message,

		Payload: common.ToHex(message.Payload),
		Padding: common.ToHex(message.Padding),
		From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
		To:      common.ToHex(crypto.FromECDSAPub(message.Dst)),
		Sent:    int64(message.Sent), // todo: review format
		TTL:     int64(message.TTL),  // todo: review format
		PoW:     int64(message.PoW),
		Hash:    common.ToHex(message.EnvelopeHash.Bytes()),
	}
}

type WhisperFilterArgs struct {
	// todo: review types
	To     string
	From   string
	KeySym string
	PoW    int
	Topics []string
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		To     interface{} `json:"to"`
		From   interface{} `json:"from"`
		Key    interface{} `json:"key"`
		PoW    interface{} `json:"pow"`
		Topics interface{} `json:"topics"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	// Retrieve the simple data contents of the filter arguments
	if obj.To == nil {
		args.To = ""
	} else {
		argstr, ok := obj.To.(string)
		if !ok {
			return fmt.Errorf("'to' is not a string")
		}
		args.To = argstr
	}
	if obj.From == nil {
		args.From = ""
	} else {
		argstr, ok := obj.From.(string)
		if !ok {
			return fmt.Errorf("'from' is not a string")
		}
		args.From = argstr
	}
	if obj.Key == nil {
		args.KeySym = ""
	} else {
		argstr, ok := obj.Key.(string)
		if !ok {
			return fmt.Errorf("'key' is not a string")
		}
		args.KeySym = argstr
	}
	if obj.PoW == nil {
		args.PoW = 0
	} else {
		argstr, ok := obj.PoW.(string)
		if !ok {
			return fmt.Errorf("'pow' is not a string")
		}
		x, err := strconv.Atoi(argstr)
		if err != nil {
			return fmt.Errorf("'pow' is invalid")
		}
		args.PoW = x
	}
	// Construct the topic array
	if obj.Topics != nil {
		// Make sure we have an actual topic array
		list, ok := obj.Topics.([]interface{})
		if !ok {
			return fmt.Errorf("topics is not an array")
		}
		// Iterate over each topic and handle nil, string or array
		topics := make([]string, len(list))
		for i, field := range list {
			switch value := field.(type) {
			case nil:
				topics[i] = ""
			case string:
				topics[i] = value
			default:
				return fmt.Errorf("topic[%d] is not a string", i)
			}
		}

		// todo: delete this block
		//topicsDecoded := make([][]byte, len(topics))
		//for j, t := range topics {
		//	topicsDecoded[j] = common.FromHex(t)
		//}
		//args.Topics = topicsDecoded
		args.Topics = topics
	}
	return nil
}

// whisperFilter is the message cache matching a specific filter, accumulating
// inbound messages until the are requested by the client.
type whisperFilter struct {
	id      int      // Filter identifier for old message retrieval
	whisper *Whisper // Whisper reference for old message retrieval

	cache  []WhisperMessage         // Cache of messages not yet polled
	skip   map[common.Hash]struct{} // List of retrieved messages to avoid duplication
	update time.Time                // Time of the last message query

	lock sync.RWMutex // Lock protecting the filter internals
}

// newWhisperFilter creates a new serialized, poll based whisper topic filter.
func newWhisperFilter(id int, w *Whisper) *whisperFilter {
	return &whisperFilter{
		id:      id,
		whisper: w,
		update:  time.Now(),
		skip:    make(map[common.Hash]struct{}),
	}
}

// messages retrieves all the cached messages from the entire pool matching the
// filter, resetting the filter's change buffer.
func (filter *whisperFilter) messages() []*ReceivedMessage {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	filter.cache = nil
	filter.update = time.Now()

	filter.skip = make(map[common.Hash]struct{})
	messages := filter.whisper.Messages(filter.id)
	for _, message := range messages {
		filter.skip[message.EnvelopeHash] = struct{}{}
	}
	return messages
}

// insert injects a new batch of messages into the filter cache.
func (filter *whisperFilter) insert(message WhisperMessage) {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	if _, ok := filter.skip[message.ref.EnvelopeHash]; !ok {
		filter.cache = append(filter.cache, message)
	}
}

// retrieve fetches all the cached messages from the filter.
func (filter *whisperFilter) retrieve() (messages []WhisperMessage) {
	filter.lock.Lock()
	defer filter.lock.Unlock()

	messages, filter.cache = filter.cache, nil
	filter.update = time.Now()
	return
}
