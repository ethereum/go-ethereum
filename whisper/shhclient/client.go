// Copyright 2017 The go-ethereum Authors
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

package shhclient

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

// Client defines typed wrappers for the Whisper v6 RPC API.
type Client struct {
	c *rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	c, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{c}
}

// Version returns the Whisper sub-protocol version.
func (sc *Client) Version(ctx context.Context) (string, error) {
	var result string
	err := sc.c.CallContext(ctx, &result, "shh_version")
	return result, err
}

// Info returns diagnostic information about the whisper node.
func (sc *Client) Info(ctx context.Context) (whisper.Info, error) {
	var info whisper.Info
	err := sc.c.CallContext(ctx, &info, "shh_info")
	return info, err
}

// SetMaxMessageSize sets the maximal message size allowed by this node. Incoming
// and outgoing messages with a larger size will be rejected. Whisper message size
// can never exceed the limit imposed by the underlying P2P protocol (10 Mb).
func (sc *Client) SetMaxMessageSize(ctx context.Context, size uint32) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_setMaxMessageSize", size)
}

// SetMinimumPoW (experimental) sets the minimal PoW required by this node.
// This experimental function was introduced for the future dynamic adjustment of
// PoW requirement. If the node is overwhelmed with messages, it should raise the
// PoW requirement and notify the peers. The new value should be set relative to
// the old value (e.g. double). The old value could be obtained via shh_info call.
func (sc *Client) SetMinimumPoW(ctx context.Context, pow float64) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_setMinPoW", pow)
}

// MarkTrustedPeer marks specific peer trusted, which will allow it to send historic (expired) messages.
// Note This function is not adding new nodes, the node needs to exists as a peer.
func (sc *Client) MarkTrustedPeer(ctx context.Context, enode string) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_markTrustedPeer", enode)
}

// NewKeyPair generates a new public and private key pair for message decryption and encryption.
// It returns an identifier that can be used to refer to the key.
func (sc *Client) NewKeyPair(ctx context.Context) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_newKeyPair")
}

// AddPrivateKey stored the key pair, and returns its ID.
func (sc *Client) AddPrivateKey(ctx context.Context, key []byte) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_addPrivateKey", hexutil.Bytes(key))
}

// DeleteKeyPair delete the specifies key.
func (sc *Client) DeleteKeyPair(ctx context.Context, id string) (string, error) {
	var ignored bool
	return id, sc.c.CallContext(ctx, &ignored, "shh_deleteKeyPair", id)
}

// HasKeyPair returns an indication if the node has a private key or
// key pair matching the given ID.
func (sc *Client) HasKeyPair(ctx context.Context, id string) (bool, error) {
	var has bool
	return has, sc.c.CallContext(ctx, &has, "shh_hasKeyPair", id)
}

// PublicKey return the public key for a key ID.
func (sc *Client) PublicKey(ctx context.Context, id string) ([]byte, error) {
	var key hexutil.Bytes
	return []byte(key), sc.c.CallContext(ctx, &key, "shh_getPublicKey", id)
}

// PrivateKey return the private key for a key ID.
func (sc *Client) PrivateKey(ctx context.Context, id string) ([]byte, error) {
	var key hexutil.Bytes
	return []byte(key), sc.c.CallContext(ctx, &key, "shh_getPrivateKey", id)
}

// NewSymmetricKey generates a random symmetric key and returns its identifier.
// Can be used encrypting and decrypting messages where the key is known to both parties.
func (sc *Client) NewSymmetricKey(ctx context.Context) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_newSymKey")
}

// AddSymmetricKey stores the key, and returns its identifier.
func (sc *Client) AddSymmetricKey(ctx context.Context, key []byte) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_addSymKey", hexutil.Bytes(key))
}

// GenerateSymmetricKeyFromPassword generates the key from password, stores it, and returns its identifier.
func (sc *Client) GenerateSymmetricKeyFromPassword(ctx context.Context, passwd string) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_generateSymKeyFromPassword", passwd)
}

// HasSymmetricKey returns an indication if the key associated with the given id is stored in the node.
func (sc *Client) HasSymmetricKey(ctx context.Context, id string) (bool, error) {
	var found bool
	return found, sc.c.CallContext(ctx, &found, "shh_hasSymKey", id)
}

// GetSymmetricKey returns the symmetric key associated with the given identifier.
func (sc *Client) GetSymmetricKey(ctx context.Context, id string) ([]byte, error) {
	var key hexutil.Bytes
	return []byte(key), sc.c.CallContext(ctx, &key, "shh_getSymKey", id)
}

// DeleteSymmetricKey deletes the symmetric key associated with the given identifier.
func (sc *Client) DeleteSymmetricKey(ctx context.Context, id string) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_deleteSymKey", id)
}

// Post a message onto the network.
func (sc *Client) Post(ctx context.Context, message whisper.NewMessage) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_post", message)
}

// SubscribeMessages subscribes to messages that match the given criteria. This method
// is only supported on bi-directional connections such as websockets and IPC.
// NewMessageFilter uses polling and is supported over HTTP.
func (sc *Client) SubscribeMessages(ctx context.Context, criteria whisper.Criteria, ch chan<- *whisper.Message) (ethereum.Subscription, error) {
	return sc.c.ShhSubscribe(ctx, ch, "messages", criteria)
}

// NewMessageFilter creates a filter within the node. This filter can be used to poll
// for new messages (see FilterMessages) that satisfy the given criteria. A filter can
// timeout when it was polled for in whisper.filterTimeout.
func (sc *Client) NewMessageFilter(ctx context.Context, criteria whisper.Criteria) (string, error) {
	var id string
	return id, sc.c.CallContext(ctx, &id, "shh_newMessageFilter", criteria)
}

// DeleteMessageFilter removes the filter associated with the given id.
func (sc *Client) DeleteMessageFilter(ctx context.Context, id string) error {
	var ignored bool
	return sc.c.CallContext(ctx, &ignored, "shh_deleteMessageFilter", id)
}

// FilterMessages retrieves all messages that are received between the last call to
// this function and match the criteria that where given when the filter was created.
func (sc *Client) FilterMessages(ctx context.Context, id string) ([]*whisper.Message, error) {
	var messages []*whisper.Message
	return messages, sc.c.CallContext(ctx, &messages, "shh_getFilterMessages", id)
}
