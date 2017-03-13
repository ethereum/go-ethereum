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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
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

// Stats returns the Whisper statistics for diagnostics.
func (api *PublicWhisperAPI) Info() (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	return api.whisper.Stats(), nil
}

func (api *PublicWhisperAPI) SetMaxMessageLength(val int) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.SetMaxMessageLength(val)
}

func (api *PublicWhisperAPI) SetMinimumPoW(val float64) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	return api.whisper.SetMinimumPoW(val)
}

// AllowP2PMessagesFromPeer marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (api *PublicWhisperAPI) AllowP2PMessagesFromPeer(enode string) error {
	if api.whisper == nil {
		return whisperOffLineErr
	}
	n, err := discover.ParseNode(enode)
	if err != nil {
		info := "Failed to parse enode of trusted peer: " + err.Error()
		log.Error(info)
		return errors.New(info)
	}
	return api.whisper.AllowP2PMessagesFromPeer(n.ID[:])
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
func (api *PublicWhisperAPI) HasKeyPair(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	return api.whisper.HasKeyPair(id), nil
}

// DeleteIdentity deletes the specifies key if it exists.
func (api *PublicWhisperAPI) DeleteKeyPair(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	success := api.whisper.DeleteKeyPair(id)
	return success, nil
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (api *PublicWhisperAPI) NewKeyPair() (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	return api.whisper.NewKeyPair()
}

// GetPublicKey returns the public key for identity id
func (api *PublicWhisperAPI) GetPublicKey(id string) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	key, err := api.whisper.GetPrivateKey(id)
	if err != nil {
		return "", err
	}
	return common.ToHex(crypto.FromECDSAPub(&key.PublicKey)), nil
}

// GetPrivateKey returns the private key for identity id
func (api *PublicWhisperAPI) GetPrivateKey(id string) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	key, err := api.whisper.GetPrivateKey(id)
	if err != nil {
		return "", err
	}
	return common.ToHex(crypto.FromECDSA(key)), nil
}

// GenerateSymKey generates a random symmetric key and stores it under id,
// which is then returned. Will be used in the future for session key exchange.
func (api *PublicWhisperAPI) GenerateSymmetricKey() (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	return api.whisper.GenerateSymKey()
}

// AddSymKeyDirect stores the key, and returns its id.
func (api *PublicWhisperAPI) AddSymmetricKeyDirect(key hexutil.Bytes) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	return api.whisper.AddSymKeyDirect(key)
}

// AddSymKeyFromPassword generates the key from password, stores it, and returns its id.
func (api *PublicWhisperAPI) AddSymmetricKeyFromPassword(password string) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}
	return api.whisper.AddSymKeyFromPassword(password)
}

// HasSymKey returns true if there is a key associated with the given id.
// Otherwise returns false.
func (api *PublicWhisperAPI) HasSymmetricKey(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	res := api.whisper.HasSymKey(id)
	return res, nil
}

func (api *PublicWhisperAPI) GetSymmetricKey(name string) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}

	b, err := api.whisper.GetSymKey(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// DeleteSymKey deletes the key associated with the name string if it exists.
func (api *PublicWhisperAPI) DeleteSymmetricKey(name string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOffLineErr
	}
	res := api.whisper.DeleteSymKey(name)
	return res, nil
}

// Subscribe creates and registers a new filter to watch for inbound whisper messages.
// Returns the ID of the newly created filter.
func (api *PublicWhisperAPI) Subscribe(args WhisperFilterArgs) (string, error) {
	if api.whisper == nil {
		return "", whisperOffLineErr
	}

	filter := Filter{
		Src:      crypto.ToECDSAPub(common.FromHex(args.SignedWith)),
		PoW:      args.MinPoW,
		Messages: make(map[common.Hash]*ReceivedMessage),
		AllowP2P: args.AllowP2P,
	}

	for _, bt := range args.Topics {
		filter.Topics = append(filter.Topics, bt)
	}

	err := ValidateKeyID(args.Key)
	if err != nil {
		info := "Subscribe: " + err.Error()
		log.Error(info)
		return "", errors.New(info)
	}

	if len(args.SignedWith) > 0 {
		if !ValidatePublicKey(filter.Src) {
			info := "Subscribe: Invalid 'SignedWith' field"
			log.Error(info)
			return "", errors.New(info)
		}
	}

	if args.Symmetric {
		if len(args.Topics) == 0 {
			info := "Subscribe: at least one topic must be specified with symmetric encryption"
			log.Error(info)
			return "", errors.New(info)
		}
		symKey, err := api.whisper.GetSymKey(args.Key)
		if err != nil {
			info := "Subscribe: invalid key ID"
			log.Error(info)
			return "", errors.New(info)
		}
		if !validateSymmetricKey(symKey) {
			info := "Subscribe: retrieved key is invalid"
			log.Error(info)
			return "", errors.New(info)
		}

		filter.KeySym = symKey
		filter.SymKeyHash = crypto.Keccak256Hash(filter.KeySym)
	} else {
		filter.KeyAsym, err = api.whisper.GetPrivateKey(args.Key)
		if err != nil {
			info := "Subscribe: invalid key ID"
			log.Error(info)
			return "", errors.New(info)
		}
		if filter.KeyAsym == nil {
			info := "Subscribe: non-existent identity provided"
			log.Error(info)
			return "", errors.New(info)
		}
	}

	return api.whisper.Watch(&filter)
}

// Unsubscribe disables and removes an existing filter.
func (api *PublicWhisperAPI) Unsubscribe(id string) {
	api.whisper.Unsubscribe(id)
}

// GetFilterChanges retrieves all the new messages matched by a filter since the last retrieval.
func (api *PublicWhisperAPI) GetSubscriptionMessages(filterId string) []*WhisperMessage {
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

	var err error
	params := MessageParams{
		TTL:      args.TTL,
		WorkTime: args.PowTime,
		PoW:      args.PowTarget,
		Payload:  args.Payload,
		Padding:  args.Padding,
	}

	if len(args.Key) == 0 {
		info := "Post: key is missing"
		log.Error(info)
		return errors.New(info)
	}

	if len(args.SignWith) > 0 {
		params.Src, err = api.whisper.GetPrivateKey(args.SignWith)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if params.Src == nil {
			info := "Post: empty identity"
			log.Error(info)
			return errors.New(info)
		}
	}

	if len(args.Topic) == TopicLength {
		params.Topic = BytesToTopic(args.Topic)
	} else if len(args.Topic) != 0 {
		info := fmt.Sprintf("Post: wrong topic size %d", len(args.Topic))
		log.Error(info)
		return errors.New(info)
	}

	if args.Type == "sym" {
		err = ValidateKeyID(args.Key)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		params.KeySym, err = api.whisper.GetSymKey(args.Key)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if !validateSymmetricKey(params.KeySym) {
			info := "Post: key for symmetric encryption is invalid"
			log.Error(info)
			return errors.New(info)
		}
		if len(params.Topic) == 0 {
			info := "Post: topic is missing for symmetric encryption"
			log.Error(info)
			return errors.New(info)
		}
	} else if args.Type == "asym" {
		params.Dst = crypto.ToECDSAPub(common.FromHex(args.Key))
		if !ValidatePublicKey(params.Dst) {
			info := "Post: public key for asymmetric encryption is invalid"
			log.Error(info)
			return errors.New(info)
		}
	} else {
		info := "Post: wrong type (sym/asym)"
		log.Error(info)
		return errors.New(info)
	}

	// encrypt and send
	message := NewSentMessage(&params)
	envelope, err := message.Wrap(&params)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	if envelope.size() > api.whisper.maxMsgLength {
		info := "Post: message is too big"
		log.Error(info)
		return errors.New(info)
	}

	if len(args.TargetPeer) != 0 {
		n, err := discover.ParseNode(args.TargetPeer)
		if err != nil {
			info := "Post: failed to parse enode of target peer: " + err.Error()
			log.Error(info)
			return errors.New(info)
		}
		return api.whisper.SendP2PMessage(n.ID[:], envelope)
	} else if args.PowTarget < api.whisper.minPoW {
		info := "Post: target PoW is less than minimum PoW, the message can not be sent"
		log.Error(info)
		return errors.New(info)
	}

	return api.whisper.Send(envelope)
}

type PostArgs struct {
	Type       string        `json:"type"`
	TTL        uint32        `json:"ttl"`
	SignWith   string        `json:"signWith"`
	Key        string        `json:"key"`
	Topic      hexutil.Bytes `json:"topic"`
	Padding    hexutil.Bytes `json:"padding"`
	Payload    hexutil.Bytes `json:"payload"`
	PowTime    uint32        `json:"powTime"`
	PowTarget  float64       `json:"powTarget"`
	TargetPeer string        `json:"targetPeer"`
}

type WhisperFilterArgs struct {
	Symmetric  bool
	Key        string
	SignedWith string
	MinPoW     float64
	Topics     [][]byte
	AllowP2P   bool
}

// UnmarshalJSON implements the json.Unmarshaler interface, invoked to convert a
// JSON message blob into a WhisperFilterArgs structure.
func (args *WhisperFilterArgs) UnmarshalJSON(b []byte) (err error) {
	// Unmarshal the JSON message and sanity check
	var obj struct {
		Type       string        `json:"type"`
		Key        string        `json:"key"`
		SignedWith string        `json:"signedWith"`
		MinPoW     float64       `json:"minPoW"`
		Topics     []interface{} `json:"topics"`
		AllowP2P   bool          `json:"allowP2P"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}

	if obj.Type == "sym" {
		args.Symmetric = true
	} else if obj.Type == "asym" {
		args.Symmetric = false
	} else {
		return fmt.Errorf("Wrong type (sym/asym")
	}

	args.Key = obj.Key
	args.SignedWith = obj.SignedWith
	args.MinPoW = obj.MinPoW
	args.AllowP2P = obj.AllowP2P

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
		topicsDecoded := make([][]byte, len(topics))
		for j, s := range topics {
			x := common.FromHex(s)
			if x == nil || len(x) > TopicLength {
				return fmt.Errorf("topic[%d] is invalid", j)
			}
			topicsDecoded[j] = x
		}
		args.Topics = topicsDecoded
	}

	return nil
}

// WhisperMessage is the RPC representation of a whisper message.
type WhisperMessage struct {
	Topic     string  `json:"topic"`
	Payload   string  `json:"payload"`
	Padding   string  `json:"padding"`
	Src       string  `json:"signedWith"`
	Dst       string  `json:"receipientPublicKey"`
	Timestamp uint32  `json:"timestamp"`
	TTL       uint32  `json:"ttl"`
	PoW       float64 `json:"pow"`
	Hash      string  `json:"hash"`
}

// NewWhisperMessage converts an internal message into an API version.
func NewWhisperMessage(message *ReceivedMessage) *WhisperMessage {
	msg := WhisperMessage{
		Topic:     common.ToHex(message.Topic[:]),
		Payload:   common.ToHex(message.Payload),
		Padding:   common.ToHex(message.Padding),
		Timestamp: message.Sent,
		TTL:       message.TTL,
		PoW:       message.PoW,
		Hash:      common.ToHex(message.EnvelopeHash.Bytes()),
	}

	if message.Dst != nil {
		msg.Dst = common.ToHex(crypto.FromECDSAPub(message.Dst))
	}
	if isMessageSigned(message.Raw[0]) {
		msg.Src = common.ToHex(crypto.FromECDSAPub(message.SigToPubKey()))
	}
	return &msg
}
