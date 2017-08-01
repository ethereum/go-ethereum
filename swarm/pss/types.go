package pss

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	TopicLength            = 32
	DefaultTTL             = 6000
	defaultDigestCacheTTL  = time.Second
	defaultWhisperWorkTime = 3
	//defaultWhisperPoW      = 0.00000000001
	defaultWhisperPoW           = 0.001
	defaultSymKeyLength         = 32
	defaultPartialAddressLength = 8
)

// Pss configuration parameters
type PssParams struct {
	Cachettl   time.Duration
	privatekey *ecdsa.PrivateKey
}

// Sane defaults for Pss
func NewPssParams(privatekey *ecdsa.PrivateKey) *PssParams {
	return &PssParams{
		Cachettl:   defaultDigestCacheTTL,
		privatekey: privatekey,
	}
}

// Encapsulates messages transported over pss.
type PssMsg struct {
	To      []byte
	Payload *whisper.Envelope
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

// String representation of Topic
//func (self *Topic) String() string {
//	return fmt.Sprintf("%x", self)
//}

// Constructs a new PssTopic from a given name and version.
//
// Analogous to the name and version members of p2p.Protocol.
func NewTopic(s string, v int) (topic whisper.TopicType) {
	h := sha3.NewKeccak256()
	h.Write([]byte(s))
	buf := make([]byte, TopicLength/8)
	binary.PutUvarint(buf, uint64(v))
	h.Write(buf)
	//copy(topic[:], h.Sum(buf)[:])
	topic = whisper.BytesToTopic(h.Sum(buf)[:4])
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
