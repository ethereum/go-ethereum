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
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var whisperOfflineErr = errors.New("whisper is offline")

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
		return whisperOfflineErr
	}
	return api.whisper.Start(nil)
}

// Stop stops the Whisper worker threads.
func (api *PublicWhisperAPI) Stop() error {
	if api.whisper == nil {
		return whisperOfflineErr
	}
	return api.whisper.Stop()
}

// Version returns the Whisper version this node offers.
func (api *PublicWhisperAPI) Version() (hexutil.Uint, error) {
	if api.whisper == nil {
		return 0, whisperOfflineErr
	}
	return hexutil.Uint(api.whisper.Version()), nil
}

// Info returns the Whisper statistics for diagnostics.
func (api *PublicWhisperAPI) Info() (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	return api.whisper.Stats(), nil
}

// SetMaxMessageLength sets the maximal message length allowed by this node
func (api *PublicWhisperAPI) SetMaxMessageLength(val int) error {
	if api.whisper == nil {
		return whisperOfflineErr
	}
	return api.whisper.SetMaxMessageLength(val)
}

// SetMinimumPoW sets the minimal PoW required by this node
func (api *PublicWhisperAPI) SetMinimumPoW(val float64) error {
	if api.whisper == nil {
		return whisperOfflineErr
	}
	return api.whisper.SetMinimumPoW(val)
}

// AllowP2PMessagesFromPeer marks specific peer trusted, which will allow it
// to send historic (expired) messages.
func (api *PublicWhisperAPI) AllowP2PMessagesFromPeer(enode string) error {
	if api.whisper == nil {
		return whisperOfflineErr
	}
	n, err := discover.ParseNode(enode)
	if err != nil {
		return errors.New("failed to parse enode of trusted peer: " + err.Error())
	}
	return api.whisper.AllowP2PMessagesFromPeer(n.ID[:])
}

// HasKeyPair checks if the whisper node is configured with the private key
// of the specified public pair.
func (api *PublicWhisperAPI) HasKeyPair(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOfflineErr
	}
	return api.whisper.HasKeyPair(id), nil
}

// DeleteKeyPair deletes the specifies key if it exists.
func (api *PublicWhisperAPI) DeleteKeyPair(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOfflineErr
	}
	return api.whisper.DeleteKeyPair(id), nil
}

// NewKeyPair generates a new cryptographic identity for the client, and injects
// it into the known identities for message decryption.
func (api *PublicWhisperAPI) NewKeyPair() (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	return api.whisper.NewKeyPair()
}

// GetPublicKey returns the public key for identity id
func (api *PublicWhisperAPI) GetPublicKey(id string) (hexutil.Bytes, error) {
	if api.whisper == nil {
		return nil, whisperOfflineErr
	}
	key, err := api.whisper.GetPrivateKey(id)
	if err != nil {
		return nil, err
	}
	return crypto.FromECDSAPub(&key.PublicKey), nil
}

// GetPrivateKey returns the private key for identity id
func (api *PublicWhisperAPI) GetPrivateKey(id string) (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	key, err := api.whisper.GetPrivateKey(id)
	if err != nil {
		return "", err
	}
	return common.ToHex(crypto.FromECDSA(key)), nil
}

// GenerateSymmetricKey generates a random symmetric key and stores it under id,
// which is then returned. Will be used in the future for session key exchange.
func (api *PublicWhisperAPI) GenerateSymmetricKey() (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	return api.whisper.GenerateSymKey()
}

// AddSymmetricKeyDirect stores the key, and returns its id.
func (api *PublicWhisperAPI) AddSymmetricKeyDirect(key hexutil.Bytes) (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	return api.whisper.AddSymKeyDirect(key)
}

// AddSymmetricKeyFromPassword generates the key from password, stores it, and returns its id.
func (api *PublicWhisperAPI) AddSymmetricKeyFromPassword(password string) (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}
	return api.whisper.AddSymKeyFromPassword(password)
}

// HasSymmetricKey returns true if there is a key associated with the given id.
// Otherwise returns false.
func (api *PublicWhisperAPI) HasSymmetricKey(id string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOfflineErr
	}
	res := api.whisper.HasSymKey(id)
	return res, nil
}

// GetSymmetricKey returns the symmetric key associated with the given id.
func (api *PublicWhisperAPI) GetSymmetricKey(name string) (hexutil.Bytes, error) {
	if api.whisper == nil {
		return nil, whisperOfflineErr
	}
	b, err := api.whisper.GetSymKey(name)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// DeleteSymmetricKey deletes the key associated with the name string if it exists.
func (api *PublicWhisperAPI) DeleteSymmetricKey(name string) (bool, error) {
	if api.whisper == nil {
		return false, whisperOfflineErr
	}
	res := api.whisper.DeleteSymKey(name)
	return res, nil
}

// Subscribe creates and registers a new filter to watch for inbound whisper messages.
// Returns the ID of the newly created filter.
func (api *PublicWhisperAPI) Subscribe(args WhisperFilterArgs) (string, error) {
	if api.whisper == nil {
		return "", whisperOfflineErr
	}

	filter := Filter{
		Src:      crypto.ToECDSAPub(common.FromHex(args.SignedWith)),
		PoW:      args.MinPoW,
		Messages: make(map[common.Hash]*ReceivedMessage),
		AllowP2P: args.AllowP2P,
	}

	var err error
	for i, bt := range args.Topics {
		if len(bt) == 0 || len(bt) > 4 {
			return "", errors.New(fmt.Sprintf("subscribe: topic %d has wrong size: %d", i, len(bt)))
		}
		filter.Topics = append(filter.Topics, bt)
	}

	if err = ValidateKeyID(args.Key); err != nil {
		return "", errors.New("subscribe: " + err.Error())
	}

	if len(args.SignedWith) > 0 {
		if !ValidatePublicKey(filter.Src) {
			return "", errors.New("subscribe: invalid 'SignedWith' field")
		}
	}

	if args.Symmetric {
		if len(args.Topics) == 0 {
			return "", errors.New("subscribe: at least one topic must be specified with symmetric encryption")
		}
		symKey, err := api.whisper.GetSymKey(args.Key)
		if err != nil {
			return "", errors.New("subscribe: invalid key ID")
		}
		if !validateSymmetricKey(symKey) {
			return "", errors.New("subscribe: retrieved key is invalid")
		}
		filter.KeySym = symKey
		filter.SymKeyHash = crypto.Keccak256Hash(filter.KeySym)
	} else {
		filter.KeyAsym, err = api.whisper.GetPrivateKey(args.Key)
		if err != nil {
			return "", errors.New("subscribe: invalid key ID")
		}
		if filter.KeyAsym == nil {
			return "", errors.New("subscribe: non-existent identity provided")
		}
	}

	return api.whisper.Subscribe(&filter)
}

// Unsubscribe disables and removes an existing filter.
func (api *PublicWhisperAPI) Unsubscribe(id string) {
	api.whisper.Unsubscribe(id)
}

// GetSubscriptionMessages retrieves all the new messages matched by a filter since the last retrieval.
func (api *PublicWhisperAPI) GetSubscriptionMessages(filterId string) []*WhisperMessage {
	f := api.whisper.GetFilter(filterId)
	if f != nil {
		newMail := f.Retrieve()
		return toWhisperMessages(newMail)
	}
	return toWhisperMessages(nil)
}

// GetMessages retrieves all the floating messages that match a specific filter.
// It is likely to be called once per session, right after Subscribe call.
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
		return whisperOfflineErr
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
		return errors.New("post: key is missing")
	}

	if len(args.SignWith) > 0 {
		params.Src, err = api.whisper.GetPrivateKey(args.SignWith)
		if err != nil {
			return err
		}
		if params.Src == nil {
			return errors.New("post: empty identity")
		}
	}

	if len(args.Topic) == TopicLength {
		params.Topic = BytesToTopic(args.Topic)
	} else if len(args.Topic) != 0 {
		return errors.New(fmt.Sprintf("post: wrong topic size %d", len(args.Topic)))
	}

	if args.Type == "sym" {
		if err = ValidateKeyID(args.Key); err != nil {
			return err
		}
		params.KeySym, err = api.whisper.GetSymKey(args.Key)
		if err != nil {
			return err
		}
		if !validateSymmetricKey(params.KeySym) {
			return errors.New("post: key for symmetric encryption is invalid")
		}
		if len(params.Topic) == 0 {
			return errors.New("post: topic is missing for symmetric encryption")
		}
	} else if args.Type == "asym" {
		params.Dst = crypto.ToECDSAPub(common.FromHex(args.Key))
		if !ValidatePublicKey(params.Dst) {
			return errors.New("post: public key for asymmetric encryption is invalid")
		}
	} else {
		return errors.New("post: wrong type (sym/asym)")
	}

	// encrypt and send
	message := NewSentMessage(&params)
	if message == nil {
		return errors.New("post: failed create new message, probably due to failed rand function (OS level)")
	}
	envelope, err := message.Wrap(&params)
	if err != nil {
		return err
	}
	if envelope.size() > api.whisper.maxMsgLength {
		return errors.New("post: message is too big")
	}

	if len(args.TargetPeer) != 0 {
		n, err := discover.ParseNode(args.TargetPeer)
		if err != nil {
			return errors.New("post: failed to parse enode of target peer: " + err.Error())
		}
		return api.whisper.SendP2PMessage(n.ID[:], envelope)
	} else if args.PowTarget < api.whisper.minPoW {
		return errors.New("post: target PoW is less than minimum PoW, the message can not be sent")
	}

	return api.whisper.Send(envelope)
}

type PostArgs struct {
	Type       string        `json:"type"`       // "sym"/"asym" (symmetric or asymmetric)
	TTL        uint32        `json:"ttl"`        // time-to-live in seconds
	SignWith   string        `json:"signWith"`   // id of the signing key
	Key        string        `json:"key"`        // id of encryption key
	Topic      hexutil.Bytes `json:"topic"`      // topic (4 bytes)
	Padding    hexutil.Bytes `json:"padding"`    // optional padding bytes
	Payload    hexutil.Bytes `json:"payload"`    // payload to be encrypted
	PowTime    uint32        `json:"powTime"`    // maximal time in seconds to be spent on PoW
	PowTarget  float64       `json:"powTarget"`  // minimal PoW required for this message
	TargetPeer string        `json:"targetPeer"` // peer id (for p2p message only)
}

type WhisperFilterArgs struct {
	Symmetric  bool     // encryption type
	Key        string   // id of the key to be used for decryption
	SignedWith string   // public key of the sender to be verified
	MinPoW     float64  // minimal PoW requirement
	Topics     [][]byte // list of topics (up to 4 bytes each) to match
	AllowP2P   bool     // indicates wheather direct p2p messages are allowed for this filter
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

	switch obj.Type {
	case "sym":
		args.Symmetric = true
	case "asym":
		args.Symmetric = false
	default:
		return errors.New("wrong type (sym/asym)")
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
	Dst       string  `json:"recipientPublicKey"`
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
