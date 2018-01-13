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

package whisperv2

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// PublicWhisperAPI provides the whisper RPC service.
type PublicWhisperAPI struct {
	w *Whisper

	messagesMu sync.RWMutex
	messages   map[hexutil.Uint]*whisperFilter
}

type whisperOfflineError struct{}

func (e *whisperOfflineError) Error() string {
	return "whisper is offline"
}

// whisperOffLineErr is returned when the node doesn't offer the shh service.
var whisperOffLineErr = new(whisperOfflineError)

// NewPublicWhisperAPI create a new RPC whisper service.
func NewPublicWhisperAPI(w *Whisper) *PublicWhisperAPI {
	return &PublicWhisperAPI{w: w, messages: make(map[hexutil.Uint]*whisperFilter)}
}

// Version returns the Whisper version this node offers.
func (s *PublicWhisperAPI) Version() (hexutil.Uint, error) {
	if s.w == nil {
		return 0, whisperOffLineErr
	}
	return hexutil.Uint(s.w.Version()), nil
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (s *PublicWhisperAPI) HasIdentity(identity string) (bool, error) {
	if s.w == nil {
		return false, whisperOffLineErr
	}
	return s.w.HasIdentity(crypto.ToECDSAPub(common.FromHex(identity))), nil
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (s *PublicWhisperAPI) NewIdentity() (string, error) {
	if s.w == nil {
		return "", whisperOffLineErr
	}

	identity := s.w.NewIdentity()
	return common.ToHex(crypto.FromECDSAPub(&identity.PublicKey)), nil
}

type NewFilterArgs struct {
	To     string
	From   string
	Topics [][][]byte
}

// NewWhisperFilter creates and registers a new message filter to watch for inbound whisper messages.
func (s *PublicWhisperAPI) NewFilter(args NewFilterArgs) (hexutil.Uint, error) {
	if s.w == nil {
		return 0, whisperOffLineErr
	}

	var id hexutil.Uint
	filter := Filter{
		To:     crypto.ToECDSAPub(common.FromHex(args.To)),
		From:   crypto.ToECDSAPub(common.FromHex(args.From)),
		Topics: NewFilterTopics(args.Topics...),
		Fn: func(message *Message) {
			wmsg := NewWhisperMessage(message)
			s.messagesMu.RLock() // Only read lock to the filter pool
			defer s.messagesMu.RUnlock()
			if s.messages[id] != nil {
				s.messages[id].insert(wmsg)
			}
		},
	}
	id = hexutil.Uint(s.w.Watch(filter))

	s.messagesMu.Lock()
	s.messages[id] = newWhisperFilter(id, s.w)
	s.messagesMu.Unlock()

	return id, nil
}

// GetFilterChanges retrieves all the new messages matched by a filter since the last retrieval.
func (s *PublicWhisperAPI) GetFilterChanges(filterId hexutil.Uint) []WhisperMessage {
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	if s.messages[filterId] != nil {
		if changes := s.messages[filterId].retrieve(); changes != nil {
			return changes
		}
	}
	return returnWhisperMessages(nil)
}

// UninstallFilter disables and removes an existing filter.
func (s *PublicWhisperAPI) UninstallFilter(filterId hexutil.Uint) bool {
	s.messagesMu.Lock()
	defer s.messagesMu.Unlock()

	if _, ok := s.messages[filterId]; ok {
		delete(s.messages, filterId)
		return true
	}
	return false
}

// GetMessages retrieves all the known messages that match a specific filter.
func (s *PublicWhisperAPI) GetMessages(filterId hexutil.Uint) []WhisperMessage {
	// Retrieve all the cached messages matching a specific, existing filter
	s.messagesMu.RLock()
	defer s.messagesMu.RUnlock()

	var messages []*Message
	if s.messages[filterId] != nil {
		messages = s.messages[filterId].messages()
	}

	return returnWhisperMessages(messages)
}

// returnWhisperMessages converts aNhisper message to a RPC whisper message.
func returnWhisperMessages(messages []*Message) []WhisperMessage {
	msgs := make([]WhisperMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = NewWhisperMessage(msg)
	}
	return msgs
}

type PostArgs struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Topics   [][]byte `json:"topics"`
	Payload  string   `json:"payload"`
	Priority int64    `json:"priority"`
	TTL      int64    `json:"ttl"`
}

// Post injects a message into the whisper network for distribution.
func (s *PublicWhisperAPI) Post(args PostArgs) (bool, error) {
	if s.w == nil {
		return false, whisperOffLineErr
	}

	// construct whisper message with transmission options
	message := NewMessage(common.FromHex(args.Payload))
	options := Options{
		To:     crypto.ToECDSAPub(common.FromHex(args.To)),
		TTL:    time.Duration(args.TTL) * time.Second,
		Topics: NewTopics(args.Topics...),
	}

	// set sender identity
	if len(args.From) > 0 {
		if key := s.w.GetIdentity(crypto.ToECDSAPub(common.FromHex(args.From))); key != nil {
			options.From = key
		} else {
			return false, fmt.Errorf("unknown identity to send from: %s", args.From)
		}
	}

	// Wrap and send the message
	pow := time.Duration(args.Priority) * time.Millisecond
	envelope, err := message.Wrap(pow, options)
	if err != nil {
		return false, err
	}

	return true, s.w.Send(envelope)
}

// WhisperMessage is the RPC representation of a whisper message.
type WhisperMessage struct {
	ref *Message

	Payload string `json:"payload"`
	To      string `json:"to"`
	From    string `json:"from"`
	Sent    int64  `json:"sent"`
	TTL     int64  `json:"ttl"`
	Hash    string `json:"hash"`
}

func (args *PostArgs) UnmarshalJSON(data []byte) (err error) {
	var obj struct {
		From     string         `json:"from"`
		To       string         `json:"to"`
		Topics   []string       `json:"topics"`
		Payload  string         `json:"payload"`
		Priority hexutil.Uint64 `json:"priority"`
		TTL      hexutil.Uint64 `json:"ttl"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	args.From = obj.From
	args.To = obj.To
	args.Payload = obj.Payload
	args.Priority = int64(obj.Priority) // TODO(gluk256): handle overflow
	args.TTL = int64(obj.TTL)           // ... here too ...

	// decode topic strings
	args.Topics = make([][]byte, len(obj.Topics))
	for i, topic := range obj.Topics {
		args.Topics[i] = common.FromHex(topic)
	}

	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *NewFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		To     interface{} `json:"to"`
		From   interface{} `json:"from"`
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
			return fmt.Errorf("to is not a string")
		}
		args.To = argstr
	}
	if obj.From == nil {
		args.From = ""
	} else {
		argstr, ok := obj.From.(string)
		if !ok {
			return fmt.Errorf("from is not a string")
		}
		args.From = argstr
	}
	// Construct the nested topic array
	if obj.Topics != nil {
		// Make sure we have an actual topic array
		list, ok := obj.Topics.([]interface{})
		if !ok {
			return fmt.Errorf("topics is not an array")
		}
		// Iterate over each topic and handle nil, string or array
		topics := make([][]string, len(list))
		for idx, field := range list {
			switch value := field.(type) {
			case nil:
				topics[idx] = []string{}

			case string:
				topics[idx] = []string{value}

			case []interface{}:
				topics[idx] = make([]string, len(value))
				for i, nested := range value {
					switch value := nested.(type) {
					case nil:
						topics[idx][i] = ""

					case string:
						topics[idx][i] = value

					default:
						return fmt.Errorf("topic[%d][%d] is not a string", idx, i)
					}
				}
			default:
				return fmt.Errorf("topic[%d] not a string or array", idx)
			}
		}

		topicsDecoded := make([][][]byte, len(topics))
		for i, condition := range topics {
			topicsDecoded[i] = make([][]byte, len(condition))
			for j, topic := range condition {
				topicsDecoded[i][j] = common.FromHex(topic)
			}
		}

		args.Topics = topicsDecoded
	}
	return nil
}

// whisperFilter is the message cache matching a specific filter, accumulating
// inbound messages until the are requested by the client.
type whisperFilter struct {
	id  hexutil.Uint // Filter identifier for old message retrieval
	ref *Whisper     // Whisper reference for old message retrieval

	cache  []WhisperMessage         // Cache of messages not yet polled
	skip   map[common.Hash]struct{} // List of retrieved messages to avoid duplication
	update time.Time                // Time of the last message query

	lock sync.RWMutex // Lock protecting the filter internals
}

// messages retrieves all the cached messages from the entire pool matching the
// filter, resetting the filter's change buffer.
func (w *whisperFilter) messages() []*Message {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.cache = nil
	w.update = time.Now()

	w.skip = make(map[common.Hash]struct{})
	messages := w.ref.Messages(int(w.id))
	for _, message := range messages {
		w.skip[message.Hash] = struct{}{}
	}
	return messages
}

// insert injects a new batch of messages into the filter cache.
func (w *whisperFilter) insert(messages ...WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	for _, message := range messages {
		if _, ok := w.skip[message.ref.Hash]; !ok {
			w.cache = append(w.cache, messages...)
		}
	}
}

// retrieve fetches all the cached messages from the filter.
func (w *whisperFilter) retrieve() (messages []WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	messages, w.cache = w.cache, nil
	w.update = time.Now()

	return
}

// newWhisperFilter creates a new serialized, poll based whisper topic filter.
func newWhisperFilter(id hexutil.Uint, ref *Whisper) *whisperFilter {
	return &whisperFilter{
		id:     id,
		ref:    ref,
		update: time.Now(),
		skip:   make(map[common.Hash]struct{}),
	}
}

// NewWhisperMessage converts an internal message into an API version.
func NewWhisperMessage(message *Message) WhisperMessage {
	return WhisperMessage{
		ref: message,

		Payload: common.ToHex(message.Payload),
		From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
		To:      common.ToHex(crypto.FromECDSAPub(message.To)),
		Sent:    message.Sent.Unix(),
		TTL:     int64(message.TTL / time.Second),
		Hash:    common.ToHex(message.Hash.Bytes()),
	}
}
