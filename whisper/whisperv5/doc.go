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

/*
Package whisper implements the Whisper protocol (version 5).

Whisper combines aspects of both DHTs and datagram messaging systems (e.g. UDP).
As such it may be likened and compared to both, not dissimilar to the
matter/energy duality (apologies to physicists for the blatant abuse of a
fundamental and beautiful natural principle).

Whisper is a pure identity-based messaging system. Whisper provides a low-level
(non-application-specific) but easily-accessible API without being based upon
or prejudiced by the low-level hardware attributes and characteristics,
particularly the notion of singular endpoints.
*/
package whisperv5

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/message"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	EnvelopeVersion    = uint64(0)
	ProtocolVersion    = uint64(5)
	ProtocolVersionStr = "5.0"
	ProtocolName       = "shh"

	statusCode           = 0 // used by whisper protocol
	messagesCode         = 1 // normal whisper message
	p2pCode              = 2 // peer-to-peer message (to be consumed by the peer, but not forwarded any further)
	p2pRequestCode       = 3 // peer-to-peer message, used by Dapp protocol
	NumberOfMessageCodes = 64

	paddingMask   = byte(3)
	signatureFlag = byte(4)

	TopicLength     = 4
	signatureLength = 65
	aesKeyLength    = 32
	AESNonceLength  = 12
	keyIdSize       = 32

	MaxMessageSize        = uint32(10 * 1024 * 1024) // maximum accepted size of a message.
	DefaultMaxMessageSize = uint32(1024 * 1024)
	DefaultMinimumPoW     = 0.001

	padSizeLimit      = 256 // just an arbitrary number, could be changed without breaking the protocol (must not exceed 2^24)
	messageQueueLimit = 1024

	expirationCycle   = time.Second
	transmissionCycle = 300 * time.Millisecond

	DefaultTTL     = 50 // seconds
	SynchAllowance = 10 // seconds
)

type unknownVersionError uint64

func (e unknownVersionError) Error() string {
	return fmt.Sprintf("invalid envelope version %d", uint64(e))
}

// MailServer represents a mail server, capable of
// archiving the old messages for subsequent delivery
// to the peers. Any implementation must ensure that both
// functions are thread-safe. Also, they must return ASAP.
// DeliverMail should use directMessagesCode for delivery,
// in order to bypass the expiry checks.
type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(whisperPeer *Peer, request *Envelope)
}

// NotificationServer represents a notification server,
// capable of screening incoming envelopes for special
// topics, and once located, subscribe client nodes as
// recipients to notifications (push notifications atm)
type NotificationServer interface {
	// Start initializes notification sending loop
	Start(server *p2p.Server) error

	// Stop stops notification sending loop, releasing related resources
	Stop() error
}

// MessageState holds the current delivery status of a whisper p2p message.
type MessageState struct {
	IsP2P     bool              `json:"is_p2p"`
	Reason    error             `json:"reason"`
	Envelope  Envelope          `json:"envelope"`
	Timestamp time.Time         `json:"timestamp"`
	Source    NewMessage        `json:"source"`
	Status    message.Status    `json:"status"`
	Direction message.Direction `json:"direction"`
	Received  ReceivedMessage   `json:"received"`
}

// DeliveryServer represents a small message status
// notification system where a message delivery status
// update event is delivered to it's underline system
// for both rpc messages and p2p messages.
type DeliveryServer interface {
	SendState(MessageState)
}
