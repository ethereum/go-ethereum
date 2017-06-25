package pss

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	TopicLength           = 32
	DefaultTTL            = 6000
	defaultDigestCacheTTL = time.Second
)

// Pss configuration parameters
type PssParams struct {
	Cachettl time.Duration
	Debug    bool
}

// Sane defaults for Pss
func NewPssParams(debug bool) *PssParams {
	return &PssParams{
		Cachettl: defaultDigestCacheTTL,
		Debug:    debug,
	}
}

// Encapsulates messages transported over pss.
type PssMsg struct {
	To      []byte
	Payload *Envelope
}

// serializes the message for use in cache
func (msg *PssMsg) serialize() []byte {
	rlpdata, _ := rlp.EncodeToBytes(msg)
	return rlpdata
}

// String representation of PssMsg
func (self *PssMsg) String() string {
	return fmt.Sprintf("PssMsg: Recipient: %x", common.ByteLabel(self.To))
}

// Pre-Whisper placeholder, payload of PssMsg, sender address, Topic
type Envelope struct {
	Topic   Topic
	TTL     uint16
	Payload []byte
	From    []byte
}

// Creates A Pss envelope from sender address, topic and raw payload
func NewEnvelope(addr []byte, topic Topic, payload []byte) *Envelope {
	return &Envelope{
		From:    addr,
		Topic:   topic,
		TTL:     DefaultTTL,
		Payload: payload,
	}
}

// Convenience wrapper for devp2p protocol messages for transport over pss
type ProtocolMsg struct {
	Code       uint64
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

// Creates a ProtocolMsg
func NewProtocolMsg(code uint64, msg interface{}) ([]byte, error) {

	rlpdata, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return nil, err
	}

	// TODO verify that nested structs cannot be used in rlp
	smsg := &ProtocolMsg{
		Code:    code,
		Size:    uint32(len(rlpdata)),
		Payload: rlpdata,
	}

	return rlp.EncodeToBytes(smsg)
}

// Convenience wrapper for sending and receiving pss messages when using the pss API
type APIMsg struct {
	Msg  []byte
	Addr []byte
}

// for debugging, show nice hex version
func (self *APIMsg) String() string {
	return fmt.Sprintf("APIMsg: from: %s..., msg: %s...", common.ByteLabel(self.Msg), common.ByteLabel(self.Addr))
}

// Signature for a message handler function for a PssMsg
//
// Implementations of this type are passed to Pss.Register together with a topic,
type Handler func(msg []byte, p *p2p.Peer, from []byte) error

// Topic defines the context of a message being transported over pss
// It is used by pss to determine what action is to be taken on an incoming message
// Typically, one can map protocol handlers for the message payloads by mapping topic to them; see Pss.Register
type Topic [TopicLength]byte

// String representation of Topic
func (self *Topic) String() string {
	return fmt.Sprintf("%x", self)
}

// Constructs a new PssTopic from a given name and version.
//
// Analogous to the name and version members of p2p.Protocol.
func NewTopic(s string, v int) (topic Topic) {
	h := sha3.NewKeccak256()
	h.Write([]byte(s))
	buf := make([]byte, TopicLength/8)
	binary.PutUvarint(buf, uint64(v))
	h.Write(buf)
	copy(topic[:], h.Sum(buf)[:])
	return topic
}

// For devp2p protocol integration only
//
// Creates a serialized (non-buffered) version of a p2p.Msg, used in the specialized p2p.MsgReadwriter implementations used internally by pss
//
// Should not normally be called outside the pss package hierarchy
func ToP2pMsg(msg []byte) (p2p.Msg, error) {
	payload := &ProtocolMsg{}
	if err := rlp.DecodeBytes(msg, payload); err != nil {
		return p2p.Msg{}, fmt.Errorf("pss protocol handler unable to decode payload as p2p message: %v", err)
	}

	return p2p.Msg{
		Code:       payload.Code,
		Size:       uint32(len(payload.Payload)),
		ReceivedAt: time.Now(),
		Payload:    bytes.NewBuffer(payload.Payload),
	}, nil
}
