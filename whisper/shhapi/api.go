// Copyright 2016 The go-ethereum Authors
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

package shhapi

import (
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
)

type WhisperOfflineError struct{}

var whisperOffLineErr = new(WhisperOfflineError)

func (e *WhisperOfflineError) Error() string {
	return "whisper is offline"
}

// PublicWhisperAPI provides the whisper RPC service.
type PublicWhisperAPI struct {
	whisper *whisperv5.Whisper
}

// NewPublicWhisperAPI create a new RPC whisper service.
func NewPublicWhisperAPI() *PublicWhisperAPI {
	w := whisperv5.New(nil)
	return &PublicWhisperAPI{whisper: w}
}

// APIs returns the RPC descriptors the Whisper implementation offers
func APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: whisperv5.ProtocolName,
			Version:   whisperv5.ProtocolVersionStr,
			Service:   NewPublicWhisperAPI(),
			Public:    true,
		},
	}
}

// Version returns the Whisper version this node offers.
func (self *PublicWhisperAPI) Version() (*rpc.HexNumber, error) {
	if self.whisper == nil {
		return rpc.NewHexNumber(0), whisperOffLineErr
	}
	return rpc.NewHexNumber(self.whisper.Version()), nil
}

// MarkPeerTrusted marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (self *PublicWhisperAPI) MarkPeerTrusted(peerID *rpc.HexBytes) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	return self.whisper.MarkPeerTrusted(*peerID)
}

// RequestHistoricMessages requests the peer to deliver the old (expired) messages.
// data contains parameters (time frame, payment details, etc.), required
// by the remote email-like server. Whisper is not aware about the data format,
// it will just forward the raw data to the server.
func (self *PublicWhisperAPI) RequestHistoricMessages(peerID *rpc.HexBytes, data *rpc.HexBytes) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	return self.whisper.RequestHistoricMessages(*peerID, *data)
}

// HasIdentity checks if the the whisper node is configured with the private key
// of the specified public pair.
func (self *PublicWhisperAPI) HasIdentity(identity string) (bool, error) {
	if self.whisper == nil {
		return false, whisperOffLineErr
	}
	return self.whisper.HasIdentity(crypto.ToECDSAPub(common.FromHex(identity))), nil
}

// DeleteIdentity deletes the specifies key if it exists.
func (self *PublicWhisperAPI) DeleteIdentity(identity string) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	self.whisper.DeleteIdentity(identity)
	return nil
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

// GenerateTopicKey generates a random key and stores it under the 'name' id.
// Will be used in the future for session key exchange.
func (self *PublicWhisperAPI) GenerateTopicKey(name string) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	return self.whisper.GenerateTopicKey(name)
}

func (self *PublicWhisperAPI) AddTopicKey(name string, key []byte) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	return self.whisper.AddTopicKey(name, key)
}

func (self *PublicWhisperAPI) HasTopicKey(name string) (bool, error) {
	if self.whisper == nil {
		return false, whisperOffLineErr
	}
	res := self.whisper.HasTopicKey(name)
	return res, nil
}

func (self *PublicWhisperAPI) DeleteTopicKey(name string) error {
	if self.whisper == nil {
		return whisperOffLineErr
	}
	self.whisper.DeleteTopicKey(name)
	return nil
}

// NewWhisperFilter creates and registers a new message filter to watch for inbound whisper messages.
func (self *PublicWhisperAPI) NewFilter(args WhisperFilterArgs) (*rpc.HexNumber, error) {
	if self.whisper == nil {
		return nil, whisperOffLineErr
	}

	filter := whisperv5.Filter{
		Src:       crypto.ToECDSAPub(args.From),
		Dst:       crypto.ToECDSAPub(args.To),
		KeySym:    self.whisper.GetTopicKey(args.KeyName),
		PoW:       args.PoW,
		Messages:  make(map[common.Hash]*whisperv5.ReceivedMessage),
		AcceptP2P: args.AcceptP2P,
	}

	if len(filter.KeySym) > 0 {
		filter.TopicKeyHash = crypto.Keccak256Hash(filter.KeySym)
	}

	for _, t := range args.Topics {
		filter.Topics = append(filter.Topics, t)
	}

	if len(args.Topics) == 0 {
		info := "NewFilter: at least one topic must be specified"
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.KeyName) != 0 && len(filter.KeySym) == 0 {
		info := "NewFilter: key was not found by name: " + args.KeyName
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.To) == 0 && len(filter.KeySym) == 0 {
		info := "NewFilter: filter must contain either symmetric or asymmetric key"
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.To) != 0 && len(filter.KeySym) != 0 {
		info := "NewFilter: filter must not contain both symmetric and asymmetric key"
		glog.V(logger.Error).Infof(info)
		return nil, errors.New(info)
	}

	if len(args.To) > 0 {
		if !whisperv5.ValidatePublicKey(filter.Dst) {
			info := "NewFilter: Invalid 'To' address"
			glog.V(logger.Error).Infof(info)
			return nil, errors.New(info)
		}
		filter.KeyAsym = self.whisper.GetIdentity(filter.Dst)
		if filter.KeyAsym == nil {
			info := "NewFilter: non-existent identity provided"
			glog.V(logger.Error).Infof(info)
			return nil, errors.New(info)
		}
	}

	if len(args.From) > 0 {
		if !whisperv5.ValidatePublicKey(filter.Src) {
			info := "NewFilter: Invalid 'From' address"
			glog.V(logger.Error).Infof(info)
			return nil, errors.New(info)
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
	f := self.whisper.GetFilter(filterId.Int())
	if f != nil {
		newMail := f.Retrieve()
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
func toWhisperMessages(messages []*whisperv5.ReceivedMessage) []WhisperMessage {
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

	params := whisperv5.MessageParams{
		TTL:      args.TTL,
		Dst:      crypto.ToECDSAPub(args.To),
		KeySym:   self.whisper.GetTopicKey(args.KeyName),
		Topic:    args.Topic,
		Payload:  args.Payload,
		Padding:  args.Padding,
		WorkTime: args.WorkTime,
		PoW:      args.PoW,
	}

	if len(args.From) > 0 {
		pub := crypto.ToECDSAPub(args.From)
		if !whisperv5.ValidatePublicKey(pub) {
			info := "Post: Invalid 'From' address"
			glog.V(logger.Error).Infof(info)
			return errors.New(info)
		}
		params.Src = self.whisper.GetIdentity(pub)
		if params.Src == nil {
			info := "Post: non-existent identity provided"
			glog.V(logger.Error).Infof(info)
			return errors.New(info)
		}
	}

	filter := self.whisper.GetFilter(args.FilterID)
	if filter == nil && args.FilterID > -1 {
		info := fmt.Sprintf("Post: wrong filter id %d", args.FilterID)
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if filter != nil {
		// get the missing fields from the filter
		if params.KeySym == nil && filter.KeySym != nil {
			params.KeySym = filter.KeySym
		}
		if params.Dst == nil && filter.Dst != nil {
			params.Dst = filter.Dst
		}
		if params.Src == nil && filter.Src != nil {
			params.Src = filter.KeyAsym
		}
		if (params.Topic == whisperv5.TopicType{}) {
			sz := len(filter.Topics)
			if sz < 1 {
				info := fmt.Sprintf("Post: no topics in filter # %d", args.FilterID)
				glog.V(logger.Error).Infof(info)
				return errors.New(info)
			} else if sz == 1 {
				params.Topic = filter.Topics[0]
			} else {
				// choose randomly
				rnd := mathrand.Intn(sz)
				params.Topic = filter.Topics[rnd]
			}
		}
	}

	// validate
	if len(args.KeyName) != 0 && len(params.KeySym) == 0 {
		info := "Post: key was not found by name: " + args.KeyName
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if len(args.To) == 0 && len(args.KeyName) == 0 {
		info := "Post: message must be encrypted either symmetrically or asymmetrically"
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if len(args.To) != 0 && len(args.KeyName) != 0 {
		info := "Post: ambigous encryption method requested"
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if len(args.To) > 0 {
		if !whisperv5.ValidatePublicKey(params.Dst) {
			info := "Post: Invalid 'To' address"
			glog.V(logger.Error).Infof(info)
			return errors.New(info)
		}
	}

	// encrypt and send
	message := whisperv5.NewSentMessage(&params)
	envelope, err := message.Wrap(params)
	if err != nil {
		glog.V(logger.Error).Infof(err.Error())
		return err
	}
	if len(envelope.Data) > whisperv5.MaxMessageLength {
		info := "Post: message is too big"
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}
	if (envelope.Topic == whisperv5.TopicType{} && envelope.IsSymmetric()) {
		info := "Post: topic is missing for symmetric encryption"
		glog.V(logger.Error).Infof(info)
		return errors.New(info)
	}

	if args.PeerID != nil {
		return self.whisper.SendP2PMessage(args.PeerID, envelope)
	}

	return self.whisper.Send(envelope)
}

type PostArgs struct {
	TTL      uint32              `json:"ttl"`
	From     rpc.HexBytes        `json:"from"`
	To       rpc.HexBytes        `json:"to"`
	KeyName  string              `json:"keyname"`
	Topic    whisperv5.TopicType `json:"topic"`
	Padding  rpc.HexBytes        `json:"padding"`
	Payload  rpc.HexBytes        `json:"payload"`
	WorkTime uint32              `json:"worktime"`
	PoW      float64             `json:"pow"`
	FilterID int                 `json:"filter"`
	PeerID   rpc.HexBytes        `json:"directP2P"`
}

func (args *PostArgs) UnmarshalJSON(data []byte) (err error) {
	var obj struct {
		TTL      uint32              `json:"ttl"`
		From     rpc.HexBytes        `json:"from"`
		To       rpc.HexBytes        `json:"to"`
		KeyName  string              `json:"keyname"`
		Topic    whisperv5.TopicType `json:"topic"`
		Payload  rpc.HexBytes        `json:"payload"`
		Padding  rpc.HexBytes        `json:"padding"`
		WorkTime uint32              `json:"worktime"`
		PoW      float64             `json:"pow"`
		FilterID rpc.HexBytes        `json:"filter"`
		PeerID   rpc.HexBytes        `json:"directP2P"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	args.TTL = obj.TTL
	args.From = obj.From
	args.To = obj.To
	args.KeyName = obj.KeyName
	args.Topic = obj.Topic
	args.Payload = obj.Payload
	args.Padding = obj.Padding
	args.WorkTime = obj.WorkTime
	args.PoW = obj.PoW
	args.FilterID = -1
	args.PeerID = obj.PeerID

	if obj.FilterID != nil {
		x := whisperv5.BytesToIntBigEndian(obj.FilterID)
		args.FilterID = int(x)
	}

	return nil
}

type WhisperFilterArgs struct {
	To        []byte
	From      []byte
	KeyName   string
	PoW       float64
	Topics    []whisperv5.TopicType
	AcceptP2P bool
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		To        rpc.HexBytes  `json:"to"`
		From      rpc.HexBytes  `json:"from"`
		KeyName   string        `json:"keyname"`
		PoW       float64       `json:"pow"`
		Topics    []interface{} `json:"topics"`
		AcceptP2P bool          `json:"acceptP2P"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	args.To = obj.To
	args.From = obj.From
	args.KeyName = obj.KeyName
	args.PoW = obj.PoW
	args.AcceptP2P = obj.AcceptP2P

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
		topicsDecoded := make([]whisperv5.TopicType, len(topics))
		for j, s := range topics {
			x := common.FromHex(s)
			if x == nil || len(x) != whisperv5.TopicLength {
				return fmt.Errorf("topic[%d] is invalid", j)
			}
			topicsDecoded[j] = whisperv5.BytesToTopic(x)
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
func NewWhisperMessage(message *whisperv5.ReceivedMessage) WhisperMessage {
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
