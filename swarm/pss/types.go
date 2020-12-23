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

package pss

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

const (
	defaultWhisperTTL = 6000
)

const (
	pssControlSym = 1
	pssControlRaw = 1 << 1
)

var (
	topicHashMutex = sync.Mutex{}
	topicHashFunc  = storage.MakeHashFunc("SHA256")()
	rawTopic       = Topic{}
)

// Topic is the PSS encapsulation of the Whisper topic type
type Topic whisper.TopicType

func (t *Topic) String() string {
	return hexutil.Encode(t[:])
}

// MarshalJSON implements the json.Marshaler interface
func (t Topic) MarshalJSON() (b []byte, err error) {
	return json.Marshal(t.String())
}

// MarshalJSON implements the json.Marshaler interface
func (t *Topic) UnmarshalJSON(input []byte) error {
	topicbytes, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	copy(t[:], topicbytes)
	return nil
}

// PssAddress is an alias for []byte. It represents a variable length address
type PssAddress []byte

// MarshalJSON implements the json.Marshaler interface
func (a PssAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Encode(a[:]))
}

// UnmarshalJSON implements the json.Marshaler interface
func (a *PssAddress) UnmarshalJSON(input []byte) error {
	b, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	for _, bb := range b {
		*a = append(*a, bb)
	}
	return nil
}

// holds the digest of a message used for caching
type pssDigest [digestLength]byte

// conceals bitwise operations on the control flags byte
type msgParams struct {
	raw bool
	sym bool
}

func newMsgParamsFromBytes(paramBytes []byte) *msgParams {
	if len(paramBytes) != 1 {
		return nil
	}
	return &msgParams{
		raw: paramBytes[0]&pssControlRaw > 0,
		sym: paramBytes[0]&pssControlSym > 0,
	}
}

func (m *msgParams) Bytes() (paramBytes []byte) {
	var b byte
	if m.raw {
		b |= pssControlRaw
	}
	if m.sym {
		b |= pssControlSym
	}
	paramBytes = append(paramBytes, b)
	return paramBytes
}

// PssMsg encapsulates messages transported over pss.
type PssMsg struct {
	To      []byte
	Control []byte
	Expire  uint32
	Payload *whisper.Envelope
}

func newPssMsg(param *msgParams) *PssMsg {
	return &PssMsg{
		Control: param.Bytes(),
	}
}

// message is flagged as raw / external encryption
func (msg *PssMsg) isRaw() bool {
	return msg.Control[0]&pssControlRaw > 0
}

// message is flagged as symmetrically encrypted
func (msg *PssMsg) isSym() bool {
	return msg.Control[0]&pssControlSym > 0
}

// serializes the message for use in cache
func (msg *PssMsg) serialize() []byte {
	rlpdata, _ := rlp.EncodeToBytes(struct {
		To      []byte
		Payload *whisper.Envelope
	}{
		To:      msg.To,
		Payload: msg.Payload,
	})
	return rlpdata
}

// String representation of PssMsg
func (msg *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %x", common.ToHex(msg.To))
}

// Signature for a message handler function for a PssMsg
// Implementations of this type are passed to Pss.Register together with a topic,
type HandlerFunc func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error

type handlerCaps struct {
	raw  bool
	prox bool
}

// Handler defines code to be executed upon reception of content.
type handler struct {
	f    HandlerFunc
	caps *handlerCaps
}

// NewHandler returns a new message handler
func NewHandler(f HandlerFunc) *handler {
	return &handler{
		f:    f,
		caps: &handlerCaps{},
	}
}

// WithRaw is a chainable method that allows raw messages to be handled.
func (h *handler) WithRaw() *handler {
	h.caps.raw = true
	return h
}

// WithProxBin is a chainable method that allows sending messages with full addresses to neighbourhoods using the kademlia depth as reference
func (h *handler) WithProxBin() *handler {
	h.caps.prox = true
	return h
}

// the stateStore handles saving and loading PSS peers and their corresponding keys
// it is currently unimplemented
type stateStore struct {
	values map[string][]byte
}

func (store *stateStore) Load(key string) ([]byte, error) {
	return nil, nil
}

func (store *stateStore) Save(key string, v []byte) error {
	return nil
}

// BytesToTopic hashes an arbitrary length byte slice and truncates it to the length of a topic, using only the first bytes of the digest
func BytesToTopic(b []byte) Topic {
	topicHashMutex.Lock()
	defer topicHashMutex.Unlock()
	topicHashFunc.Reset()
	topicHashFunc.Write(b)
	return Topic(whisper.BytesToTopic(topicHashFunc.Sum(nil)))
}
