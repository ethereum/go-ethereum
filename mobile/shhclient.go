// Copyright 2018 The go-ethereum Authors
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

// Contains a wrapper for the Whisper client.

package geth

import (
	"github.com/ethereum/go-ethereum/whisper/shhclient"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// WhisperClient provides access to the Ethereum APIs.
type WhisperClient struct {
	client *shhclient.Client
}

// NewWhisperClient connects a client to the given URL.
func NewWhisperClient(rawurl string) (client *WhisperClient, _ error) {
	rawClient, err := shhclient.Dial(rawurl)
	return &WhisperClient{rawClient}, err
}

// GetVersion returns the Whisper sub-protocol version.
func (wc *WhisperClient) GetVersion(ctx *Context) (version string, _ error) {
	return wc.client.Version(ctx.context)
}

// Info returns diagnostic information about the whisper node.
func (wc *WhisperClient) GetInfo(ctx *Context) (info *Info, _ error) {
	rawInfo, err := wc.client.Info(ctx.context)
	return &Info{&rawInfo}, err
}

// SetMaxMessageSize sets the maximal message size allowed by this node. Incoming
// and outgoing messages with a larger size will be rejected. Whisper message size
// can never exceed the limit imposed by the underlying P2P protocol (10 Mb).
func (wc *WhisperClient) SetMaxMessageSize(ctx *Context, size int32) error {
	return wc.client.SetMaxMessageSize(ctx.context, uint32(size))
}

// SetMinimumPoW (experimental) sets the minimal PoW required by this node.
// This experimental function was introduced for the future dynamic adjustment of
// PoW requirement. If the node is overwhelmed with messages, it should raise the
// PoW requirement and notify the peers. The new value should be set relative to
// the old value (e.g. double). The old value could be obtained via shh_info call.
func (wc *WhisperClient) SetMinimumPoW(ctx *Context, pow float64) error {
	return wc.client.SetMinimumPoW(ctx.context, pow)
}

// Marks specific peer trusted, which will allow it to send historic (expired) messages.
// Note This function is not adding new nodes, the node needs to exists as a peer.
func (wc *WhisperClient) MarkTrustedPeer(ctx *Context, enode string) error {
	return wc.client.MarkTrustedPeer(ctx.context, enode)
}

// NewKeyPair generates a new public and private key pair for message decryption and encryption.
// It returns an identifier that can be used to refer to the key.
func (wc *WhisperClient) NewKeyPair(ctx *Context) (string, error) {
	return wc.client.NewKeyPair(ctx.context)
}

// AddPrivateKey stored the key pair, and returns its ID.
func (wc *WhisperClient) AddPrivateKey(ctx *Context, key []byte) (string, error) {
	return wc.client.AddPrivateKey(ctx.context, key)
}

// DeleteKeyPair delete the specifies key.
func (wc *WhisperClient) DeleteKeyPair(ctx *Context, id string) (string, error) {
	return wc.client.DeleteKeyPair(ctx.context, id)
}

// HasKeyPair returns an indication if the node has a private key or
// key pair matching the given ID.
func (wc *WhisperClient) HasKeyPair(ctx *Context, id string) (bool, error) {
	return wc.client.HasKeyPair(ctx.context, id)
}

// GetPublicKey return the public key for a key ID.
func (wc *WhisperClient) GetPublicKey(ctx *Context, id string) ([]byte, error) {
	return wc.client.PublicKey(ctx.context, id)
}

// GetPrivateKey return the private key for a key ID.
func (wc *WhisperClient) GetPrivateKey(ctx *Context, id string) ([]byte, error) {
	return wc.client.PrivateKey(ctx.context, id)
}

// NewSymmetricKey generates a random symmetric key and returns its identifier.
// Can be used encrypting and decrypting messages where the key is known to both parties.
func (wc *WhisperClient) NewSymmetricKey(ctx *Context) (string, error) {
	return wc.client.NewSymmetricKey(ctx.context)
}

// AddSymmetricKey stores the key, and returns its identifier.
func (wc *WhisperClient) AddSymmetricKey(ctx *Context, key []byte) (string, error) {
	return wc.client.AddSymmetricKey(ctx.context, key)
}

// GenerateSymmetricKeyFromPassword generates the key from password, stores it, and returns its identifier.
func (wc *WhisperClient) GenerateSymmetricKeyFromPassword(ctx *Context, passwd string) (string, error) {
	return wc.client.GenerateSymmetricKeyFromPassword(ctx.context, passwd)
}

// HasSymmetricKey returns an indication if the key associated with the given id is stored in the node.
func (wc *WhisperClient) HasSymmetricKey(ctx *Context, id string) (bool, error) {
	return wc.client.HasSymmetricKey(ctx.context, id)
}

// GetSymmetricKey returns the symmetric key associated with the given identifier.
func (wc *WhisperClient) GetSymmetricKey(ctx *Context, id string) ([]byte, error) {
	return wc.client.GetSymmetricKey(ctx.context, id)
}

// DeleteSymmetricKey deletes the symmetric key associated with the given identifier.
func (wc *WhisperClient) DeleteSymmetricKey(ctx *Context, id string) error {
	return wc.client.DeleteSymmetricKey(ctx.context, id)
}

// Post a message onto the network.
func (wc *WhisperClient) Post(ctx *Context, message *NewMessage) (string, error) {
	return wc.client.Post(ctx.context, *message.newMessage)
}

// NewHeadHandler is a client-side subscription callback to invoke on events and
// subscription failure.
type NewMessageHandler interface {
	OnNewMessage(message *Message)
	OnError(failure string)
}

// SubscribeMessages subscribes to messages that match the given criteria. This method
// is only supported on bi-directional connections such as websockets and IPC.
// NewMessageFilter uses polling and is supported over HTTP.
func (wc *WhisperClient) SubscribeMessages(ctx *Context, criteria *Criteria, handler NewMessageHandler, buffer int) (*Subscription, error) {
	// Subscribe to the event internally
	ch := make(chan *whisper.Message, buffer)
	rawSub, err := wc.client.SubscribeMessages(ctx.context, *criteria.criteria, ch)
	if err != nil {
		return nil, err
	}
	// Start up a dispatcher to feed into the callback
	go func() {
		for {
			select {
			case message := <-ch:
				handler.OnNewMessage(&Message{message})

			case err := <-rawSub.Err():
				if err != nil {
					handler.OnError(err.Error())
				}
				return
			}
		}
	}()
	return &Subscription{rawSub}, nil
}

// NewMessageFilter creates a filter within the node. This filter can be used to poll
// for new messages (see FilterMessages) that satisfy the given criteria. A filter can
// timeout when it was polled for in whisper.filterTimeout.
func (wc *WhisperClient) NewMessageFilter(ctx *Context, criteria *Criteria) (string, error) {
	return wc.client.NewMessageFilter(ctx.context, *criteria.criteria)
}

// DeleteMessageFilter removes the filter associated with the given id.
func (wc *WhisperClient) DeleteMessageFilter(ctx *Context, id string) error {
	return wc.client.DeleteMessageFilter(ctx.context, id)
}

// GetFilterMessages retrieves all messages that are received between the last call to
// this function and match the criteria that where given when the filter was created.
func (wc *WhisperClient) GetFilterMessages(ctx *Context, id string) (*Messages, error) {
	rawFilterMessages, err := wc.client.FilterMessages(ctx.context, id)
	if err != nil {
		return nil, err
	}
	res := make([]*whisper.Message, len(rawFilterMessages))
	copy(res, rawFilterMessages)
	return &Messages{res}, nil
}
