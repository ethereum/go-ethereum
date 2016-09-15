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

package whisper05

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicWhisperAPI provides the whisper RPC service.
type PublicWhisperAPI struct {
	whisper *Whisper
}

// NewPublicWhisperAPI create a new RPC whisper service.
func NewPublicWhisperAPI(w *Whisper) *PublicWhisperAPI {
	return &PublicWhisperAPI{whisper: w}
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

	filter := Filter{
		Src:      crypto.ToECDSAPub(args.From),
		Dst:      crypto.ToECDSAPub(args.To),
		KeySym:   args.KeySym,
		PoW:      args.PoW,
		messages: make(map[common.Hash]*ReceivedMessage),
	}

	if len(args.Topics) > 0 {
		for _, x := range args.Topics {
			filter.Topics = append(filter.Topics, x)
		}
	}

	// todo: add glog on error everywhere

	if len(args.To) == 0 && len(args.KeySym) == 0 {
		info := "NewFilter: filter must contain either symmetric or asymmetric key"
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.KeySym) > 0 && len(args.Topics) == 0 {
		return nil, fmt.Errorf("NewFilter: topic encryption require at least one topic")
	}

	if len(args.KeySym) > 0 {
		filter.TopicKeyHash = crypto.Keccak256Hash(filter.KeySym)
		if len(args.Topics) > 0 {
			// if Topics are not provided, just use the default derivation function
			t := DeriveTopicFromSymmetricKey(args.KeySym)
			filter.Topics = append(filter.Topics, t)
		}
	}

	if len(args.To) > 0 {
		if !validatePublicKey(filter.Dst) {
			return nil, fmt.Errorf("NewFilter: Invalid 'To' address")
		}
		filter.KeyAsym = self.whisper.GetIdentity(filter.Dst)
		if filter.KeyAsym == nil {
			info := "NewFilter: non-existent identity provided"
			glog.V(logger.Error).Infof(info)
			return nil, errors.New(info)
		}
	}

	if len(args.From) > 0 {
		if !validatePublicKey(filter.Src) {
			return nil, fmt.Errorf("NewFilter: Invalid 'From' address")
		}
	}

	id := self.whisper.Watch(&filter)
	return rpc.NewHexNumber(id), nil
}

// UninstallFilter disables and removes an existing filter.
func (self *PublicWhisperAPI) UninstallFilter(filterId rpc.HexNumber) {
	self.whisper.Unwatch(filterId.Int())
}

// GetFilterChanges retrieves all the new messages matched by a filter since the last retrieval.
func (self *PublicWhisperAPI) GetFilterChanges(filterId rpc.HexNumber) []WhisperMessage {
	f := self.whisper.filters.Get(filterId.Int())
	if f != nil {
		newMail := f.retrieve()
		return toWhisperMessages(newMail)
	}
	return toWhisperMessages(nil)
}

// GetMessages retrieves all the known messages that match a specific filter.
func (self *PublicWhisperAPI) GetMessages(filterId rpc.HexNumber) []WhisperMessage {
	all := self.whisper.Messages(filterId.Int())
	return toWhisperMessages(all)
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
func (self *PublicWhisperAPI) Post(args PostArgs) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}

	message := NewSentMessage(args.Payload)
	options := Options{
		TTL:      args.TTL,
		Dst:      crypto.ToECDSAPub(args.To),
		KeySym:   args.KeySym,
		Topic:    args.Topic,
		Pading:   args.Padding,
		WorkTime: args.WorkTime,
		PoW:      args.PoW,
	}

	// todo: add glog on error everywhere

	if len(args.KeySym) > 0 && (args.Topic == TopicType{}) {
		// if Topic is not provided, just use the default derivation function
		options.Topic = DeriveTopicFromSymmetricKey(args.KeySym)
	}

	if len(args.From) > 0 {
		pub := crypto.ToECDSAPub(args.From)
		if !validatePublicKey(pub) {
			return fmt.Errorf("Post: Invalid 'From' address")
		}
		options.Src = self.whisper.GetIdentity(pub)
		if options.Src == nil {
			info := "Post: non-existent identity provided"
			glog.V(logger.Error).Infof(info)
			return errors.New(info)
		}
	}

	if len(args.To) == 0 && len(args.KeySym) == 0 {
		info := "Post: message must be encrypted either symmetrically or asymmetrically"
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if len(args.To) > 0 {
		if !validatePublicKey(options.Dst) {
			return fmt.Errorf("Post: Invalid 'To' address")
		}
	}

	envelope, err := message.Wrap(options)
	if err != nil {
		return err
	}
	return self.whisper.Send(envelope)
}

type PostArgs struct {
	TTL      uint32       `json:"ttl"`
	From     rpc.HexBytes `json:"from"`
	To       rpc.HexBytes `json:"to"`
	KeySym   rpc.HexBytes `json:"key"`
	Topic    TopicType    `json:"topic"`
	Padding  rpc.HexBytes `json:"padding"`
	Payload  rpc.HexBytes `json:"payload"`
	WorkTime uint32       `json:"worktime"`
	PoW      float64      `json:"pow"`
}

func (args *PostArgs) UnmarshalJSON(data []byte) (err error) {
	var obj struct {
		TTL      uint32       `json:"ttl"`
		From     rpc.HexBytes `json:"from"`
		To       rpc.HexBytes `json:"to"`
		Key      rpc.HexBytes `json:"key"`
		Topic    TopicType    `json:"topic"`
		Payload  rpc.HexBytes `json:"payload"`
		Padding  rpc.HexBytes `json:"padding"`
		WorkTime uint32       `json:"worktime"`
		PoW      float64      `json:"pow"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	args.TTL = obj.TTL
	args.From = obj.From
	args.To = obj.To
	args.KeySym = obj.Key
	args.Topic = obj.Topic
	args.Payload = obj.Payload
	args.Padding = obj.Padding
	args.WorkTime = obj.WorkTime
	args.PoW = obj.PoW

	return nil
}

type WhisperFilterArgs struct {
	To     []byte
	From   []byte
	KeySym []byte
	PoW    float64
	Topics []TopicType
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		To     rpc.HexBytes  `json:"to"`
		From   rpc.HexBytes  `json:"from"`
		Key    rpc.HexBytes  `json:"key"`
		PoW    float64       `json:"pow"`
		Topics []interface{} `json:"topics"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	args.To = obj.To
	args.From = obj.From
	args.KeySym = obj.Key
	args.PoW = obj.PoW

	// Construct the topic array
	if obj.Topics != nil {
		topics := make([]string, len(obj.Topics))
		for i, field := range obj.Topics {
			switch value := field.(type) {
			case string:
				topics[i] = value
			case nil:
				return fmt.Errorf("topic[%d] is empty", i)
			default:
				return fmt.Errorf("topic[%d] is not a string", i)
			}
		}
		topicsDecoded := make([]TopicType, len(topics))
		for j, s := range topics {
			x := common.FromHex(s)
			if x == nil || len(x) != topicLength {
				return fmt.Errorf("topic[%d] is invalid", j)
			}
			topicsDecoded[j] = BytesToTopic(x)
		}
		args.Topics = topicsDecoded
	}

	return nil
}

// WhisperMessage is the RPC representation of a whisper message.
type WhisperMessage struct {
	Payload string  `json:"payload"`
	Padding string  `json:"padding"`
	From    string  `json:"from"`
	To      string  `json:"to"`
	Sent    uint32  `json:"sent"`
	TTL     uint32  `json:"ttl"`
	PoW     float64 `json:"pow"`
	Hash    string  `json:"hash"`
}

// NewWhisperMessage converts an internal message into an API version.
func NewWhisperMessage(message *ReceivedMessage) WhisperMessage {
	return WhisperMessage{
		Payload: common.ToHex(message.Payload),
		Padding: common.ToHex(message.Padding),
		From:    common.ToHex(crypto.FromECDSAPub(message.Recover())),
		To:      common.ToHex(crypto.FromECDSAPub(message.Dst)),
		Sent:    message.Sent,
		TTL:     message.TTL,
		PoW:     message.PoW,
		Hash:    common.ToHex(message.EnvelopeHash.Bytes()),
	}
}
