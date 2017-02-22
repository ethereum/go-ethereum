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

package whisperv5

import (
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

var whisperOffLineErr = errors.New("whisper is offline")

// PublicWhisperAPI provides the whisper RPC service.
type PublicWhisperAPI struct {
	whisper *Whisper
}

// NewPublicWhisperAPI create a new RPC whisper service.
func NewPublicWhisperAPI(w *Whisper) *PublicWhisperAPI {
	return &PublicWhisperAPI{whisper: w}
}

// Start starts the Whisper worker threads.
func (api *PublicWhisperAPI) Start() error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.Start(nil)
}

// Stop stops the Whisper worker threads.
func (api *PublicWhisperAPI) Stop() error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.Stop()
}

// Version returns the Whisper version this node offers.
func (api *PublicWhisperAPI) Version() (hexutil.Uint, error) {
	if api.whisper == nil {
		return 0, whisperOffLineErr
	}
	return hexutil.Uint(api.whisper.Version()), nil
}

// MarkPeerTrusted marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (api *PublicWhisperAPI) MarkPeerTrusted(peerID hexutil.Bytes) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.MarkPeerTrusted(peerID)
}

// RequestHistoricMessages requests the peer to deliver the old (expired) messages.
// data contains parameters (time frame, payment details, etc.), required
// by the remote email-like server. Whisper is not aware about the data format,
// it will just forward the raw data to the server.
//func (api *PublicWhisperAPI) RequestHistoricMessages(peerID hexutil.Bytes, data hexutil.Bytes) error {
//	if api.whisper == nil {
//		return whisperOffLineErr
//	}
//	return api.whisper.RequestHistoricMessages(peerID, data)
//}

// HasIdentity checks if the whisper node is configured with the private key
// of the specified public pair.
func (api *PublicWhisperAPI) HasIdentity(identity string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	return api.whisper.HasIdentity(identity), nil
}

// DeleteIdentity deletes the specifies key if it exists.
func (api *PublicWhisperAPI) DeleteIdentity(identity string) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	api.whisper.DeleteIdentity(identity)
	return nil
}

// NewIdentity generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (api *PublicWhisperAPI) NewIdentity() (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	identity := api.whisper.NewIdentity()
	return common.ToHex(crypto.FromECDSAPub(&identity.PublicKey)), nil
}

// GenerateSymKey generates a random symmetric key and stores it under
// the 'name' id. Will be used in the future for session key exchange.
func (api *PublicWhisperAPI) GenerateSymKey(name string) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.GenerateSymKey(name)
}

// AddSymKey stores the key under the 'name' id.
func (api *PublicWhisperAPI) AddSymKey(name string, key hexutil.Bytes) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.AddSymKey(name, key)
}

// HasSymKey returns true if there is a key associated with the name string.
// Otherwise returns false.
func (api *PublicWhisperAPI) HasSymKey(name string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	res := api.whisper.HasSymKey(name)
	return res, nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (api *PublicWhisperAPI) DeleteSymKey(name string) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	api.whisper.DeleteSymKey(name)
	return nil
}

// NewWhisperFilter creates and registers a new message filter to watch for inbound whisper messages.
// Returns the ID of the newly created Filter.
func (api *PublicWhisperAPI) NewFilter(args WhisperFilterArgs) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}

	filter := Filter{
		Src:       crypto.ToECDSAPub(common.FromHex(args.From)),
		KeySym:    api.whisper.GetSymKey(args.KeyName),
		PoW:       args.PoW,
		Messages:  make(map[common.Hash]*ReceivedMessage),
		AcceptP2P: args.AcceptP2P,
	}
	if len(filter.KeySym) > 0 {
		filter.SymKeyHash = crypto.Keccak256Hash(filter.KeySym)
	}
	filter.Topics = append(filter.Topics, args.Topics...)

	if len(args.Topics) == 0 && len(args.KeyName) != 0 {
		info := "NewFilter: at least one topic must be specified"
		log.Error(fmt.Sprintf(info))
		return "", errors.New(info)
	}

	if len(args.KeyName) != 0 && len(filter.KeySym) == 0 {
		info := "NewFilter: key was not found by name: " + args.KeyName
		log.Error(fmt.Sprintf(info))
		return "", errors.New(info)
	}

	if len(args.To) == 0 && len(filter.KeySym) == 0 {
		info := "NewFilter: filter must contain either symmetric or asymmetric key"
		log.Error(fmt.Sprintf(info))
		return "", errors.New(info)
	}

	if len(args.To) != 0 && len(filter.KeySym) != 0 {
		info := "NewFilter: filter must not contain both symmetric and asymmetric key"
		log.Error(fmt.Sprintf(info))
		return "", errors.New(info)
	}

	if len(args.To) > 0 {
		dst := crypto.ToECDSAPub(common.FromHex(args.To))
		if !ValidatePublicKey(dst) {
			info := "NewFilter: Invalid 'To' address"
			log.Error(fmt.Sprintf(info))
			return "", errors.New(info)
		}
		filter.KeyAsym = api.whisper.GetIdentity(string(args.To))
		if filter.KeyAsym == nil {
			info := "NewFilter: non-existent identity provided"
			log.Error(fmt.Sprintf(info))
			return "", errors.New(info)
		}
	}

	if len(args.From) > 0 {
		if !ValidatePublicKey(filter.Src) {
			info := "NewFilter: Invalid 'From' address"
			log.Error(fmt.Sprintf(info))
			return "", errors.New(info)
		}
	}

	return api.whisper.Watch(&filter)
}

// UninstallFilter disables and removes an existing filter.
func (api *PublicWhisperAPI) UninstallFilter(filterId string) {
	api.whisper.Unwatch(filterId)
}

// GetFilterChanges retrieves all the new messages matched by a filter since the last retrieval.
func (api *PublicWhisperAPI) GetFilterChanges(filterId string) []*WhisperMessage {
	f := api.whisper.GetFilter(filterId)
	if f != nil {
		newMail := f.Retrieve()
		return toWhisperMessages(newMail)
	}
	return toWhisperMessages(nil)
}

// GetMessages retrieves all the known messages that match a specific filter.
func (api *PublicWhisperAPI) GetMessages(filterId string) []*WhisperMessage {
	all := api.whisper.Messages(filterId)
	return toWhisperMessages(all)
}

// toWhisperMessages converts a Whisper message to a RPC whisper message.
func toWhisperMessages(messages []*ReceivedMessage) []*WhisperMessage {
	msgs := make([]*WhisperMessage, len(messages))
	for i, msg := range messages {
		msgs[i] = NewWhisperMessage(msg)
	}
	return msgs
}

// Post creates a whisper message and injects it into the network for distribution.
func (api *PublicWhisperAPI) Post(args PostArgs) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}

	params := MessageParams{
		TTL:      args.TTL,
		Dst:      crypto.ToECDSAPub(common.FromHex(args.To)),
		KeySym:   api.whisper.GetSymKey(args.KeyName),
		Topic:    args.Topic,
		Payload:  args.Payload,
		Padding:  args.Padding,
		WorkTime: args.WorkTime,
		PoW:      args.PoW,
	}

	if len(args.From) > 0 {
		pub := crypto.ToECDSAPub(common.FromHex(args.From))
		if !ValidatePublicKey(pub) {
			info := "Post: Invalid 'From' address"
			log.Error(fmt.Sprintf(info))
			return errors.New(info)
		}
		params.Src = api.whisper.GetIdentity(string(args.From))
		if params.Src == nil {
			info := "Post: non-existent identity provided"
			log.Error(fmt.Sprintf(info))
			return errors.New(info)
		}
	}

	filter := api.whisper.GetFilter(args.FilterID)
	if filter == nil && len(args.FilterID) > 0 {
		info := fmt.Sprintf("Post: wrong filter id %s", args.FilterID)
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}

	if filter != nil {
		// get the missing fields from the filter
		if params.KeySym == nil && filter.KeySym != nil {
			params.KeySym = filter.KeySym
		}
		if params.Src == nil && filter.Src != nil {
			params.Src = filter.KeyAsym
		}
		if (params.Topic == TopicType{}) {
			sz := len(filter.Topics)
			if sz < 1 {
				info := fmt.Sprintf("Post: no topics in filter # %s", args.FilterID)
				log.Error(fmt.Sprintf(info))
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
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}

	if len(args.To) == 0 && len(params.KeySym) == 0 {
		info := "Post: message must be encrypted either symmetrically or asymmetrically"
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}

	if len(args.To) != 0 && len(params.KeySym) != 0 {
		info := "Post: ambigous encryption method requested"
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}

	if len(args.To) > 0 {
		if !ValidatePublicKey(params.Dst) {
			info := "Post: Invalid 'To' address"
			log.Error(fmt.Sprintf(info))
			return errors.New(info)
		}
	}

	// encrypt and send
	message := NewSentMessage(&params)
	envelope, err := message.Wrap(&params)
	if err != nil {
		log.Error(fmt.Sprintf(err.Error()))
		return err
	}
	if len(envelope.Data) > MaxMessageLength {
		info := "Post: message is too big"
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}
	if (envelope.Topic == TopicType{} && envelope.IsSymmetric()) {
		info := "Post: topic is missing for symmetric encryption"
		log.Error(fmt.Sprintf(info))
		return errors.New(info)
	}

	if args.PeerID != nil {
		return api.whisper.SendP2PMessage(args.PeerID, envelope)
	}

	return api.whisper.Send(envelope)
}

type PostArgs struct {
	TTL      uint32        `json:"ttl"`
	From     string        `json:"from"`
	To       string        `json:"to"`
	KeyName  string        `json:"keyname"`
	Topic    TopicType     `json:"topic"`
	Padding  hexutil.Bytes `json:"padding"`
	Payload  hexutil.Bytes `json:"payload"`
	WorkTime uint32        `json:"worktime"`
	PoW      float64       `json:"pow"`
	FilterID string        `json:"filterID"`
	PeerID   hexutil.Bytes `json:"peerID"`
}

type WhisperFilterArgs struct {
	To        string      `json:"to"`
	From      string      `json:"from"`
	KeyName   string      `json:"keyname"`
	PoW       float64     `json:"pow"`
	Topics    []TopicType `json:"topics"`
	AcceptP2P bool        `json:"p2p"`
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		To        string        `json:"to"`
		From      string        `json:"from"`
		KeyName   string        `json:"keyname"`
		PoW       float64       `json:"pow"`
		Topics    []interface{} `json:"topics"`
		AcceptP2P bool          `json:"p2p"`
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
		topicsDecoded := make([]TopicType, len(topics))
		for j, s := range topics {
			x := common.FromHex(s)
			if x == nil || len(x) != TopicLength {
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
	Topic   string  `json:"topic"`
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
func NewWhisperMessage(message *ReceivedMessage) *WhisperMessage {
	msg := WhisperMessage{
		Topic:   common.ToHex(message.Topic[:]),
		Payload: common.ToHex(message.Payload),
		Padding: common.ToHex(message.Padding),
		Sent:    message.Sent,
		TTL:     message.TTL,
		PoW:     message.PoW,
		Hash:    common.ToHex(message.EnvelopeHash.Bytes()),
	}

	if message.Dst != nil {
		msg.To = common.ToHex(crypto.FromECDSAPub(message.Dst))
	}
	if isMessageSigned(message.Raw[0]) {
		msg.From = common.ToHex(crypto.FromECDSAPub(message.SigToPubKey()))
	}
	return &msg
}
