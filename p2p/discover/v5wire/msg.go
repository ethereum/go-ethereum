// Copyright 2019 The go-ethereum Authors
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

package v5wire

import (
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

// Packet is implemented by all message types.
type Packet interface {
	Name() string    // Name returns a string corresponding to the message type.
	Kind() byte      // Kind returns the message type.
	SetReqID([]byte) // Sets the request ID.
}

// Message types.
const (
	PingMsg byte = iota + 1
	PongMsg
	FindnodeMsg
	NodesMsg
	TalkRequestMsg
	TalkResponseMsg
	RequestTicketMsg
	TicketMsg
	RegtopicMsg
	RegconfirmationMsg
	TopicQueryMsg

	UnknownPacket   = byte(255) // any non-decryptable packet
	WhoareyouPacket = byte(254) // the WHOAREYOU packet
)

// Protocol messages.
type (
	// Unknown represents any packet that can't be decrypted.
	Unknown struct {
		Nonce Nonce
	}

	// WHOAREYOU contains the handshake challenge.
	Whoareyou struct {
		Header    Header
		Nonce     Nonce    // Nonce of request packet
		IDNonce   [16]byte // Identity proof data
		RecordSeq uint64   // ENR sequence number of recipient

		// Node is the locally known node record of recipient.
		// This must be set by the caller of Encode.
		Node *enode.Node

		sent mclock.AbsTime // for handshake GC.
	}

	// PING is sent during liveness checks.
	Ping struct {
		ReqID  []byte
		ENRSeq uint64
	}

	// PONG is the reply to PING.
	Pong struct {
		ReqID  []byte
		ENRSeq uint64
		ToIP   net.IP // These fields should mirror the UDP envelope address of the ping
		ToPort uint16 // packet, which provides a way to discover the the external address (after NAT).
	}

	// FINDNODE is a query for nodes in the given bucket.
	Findnode struct {
		ReqID     []byte
		Distances []uint
	}

	// NODES is the reply to FINDNODE and TOPICQUERY.
	Nodes struct {
		ReqID []byte
		Total uint8
		Nodes []*enr.Record
	}

	// TALKREQ is an application-level request.
	TalkRequest struct {
		ReqID    []byte
		Protocol string
		Message  []byte
	}

	// TALKRESP is the reply to TALKREQ.
	TalkResponse struct {
		ReqID   []byte
		Message []byte
	}

	// REQUESTTICKET requests a ticket for a topic queue.
	RequestTicket struct {
		ReqID []byte
		Topic []byte
	}

	// TICKET is the response to REQUESTTICKET.
	Ticket struct {
		ReqID  []byte
		Ticket []byte
	}

	// REGTOPIC registers the sender in a topic queue using a ticket.
	Regtopic struct {
		ReqID  []byte
		Ticket []byte
		ENR    *enr.Record
	}

	// REGCONFIRMATION is the reply to REGTOPIC.
	Regconfirmation struct {
		ReqID      []byte
		Registered bool
	}

	// TOPICQUERY asks for nodes with the given topic.
	TopicQuery struct {
		ReqID []byte
		Topic []byte
	}
)

// DecodeMessage decodes the message body of a packet.
func DecodeMessage(ptype byte, body []byte) (Packet, error) {
	var dec Packet
	switch ptype {
	case PingMsg:
		dec = new(Ping)
	case PongMsg:
		dec = new(Pong)
	case FindnodeMsg:
		dec = new(Findnode)
	case NodesMsg:
		dec = new(Nodes)
	case TalkRequestMsg:
		dec = new(TalkRequest)
	case TalkResponseMsg:
		dec = new(TalkResponse)
	case RequestTicketMsg:
		dec = new(RequestTicket)
	case TicketMsg:
		dec = new(Ticket)
	case RegtopicMsg:
		dec = new(Regtopic)
	case RegconfirmationMsg:
		dec = new(Regconfirmation)
	case TopicQueryMsg:
		dec = new(TopicQuery)
	default:
		return nil, fmt.Errorf("unknown packet type %d", ptype)
	}
	if err := rlp.DecodeBytes(body, dec); err != nil {
		return nil, err
	}
	return dec, nil
}

func (*Whoareyou) Name() string    { return "WHOAREYOU/v5" }
func (*Whoareyou) Kind() byte      { return WhoareyouPacket }
func (*Whoareyou) SetReqID([]byte) {}

func (*Unknown) Name() string    { return "UNKNOWN/v5" }
func (*Unknown) Kind() byte      { return UnknownPacket }
func (*Unknown) SetReqID([]byte) {}

func (*Ping) Name() string         { return "PING/v5" }
func (*Ping) Kind() byte           { return PingMsg }
func (p *Ping) SetReqID(id []byte) { p.ReqID = id }

func (*Pong) Name() string         { return "PONG/v5" }
func (*Pong) Kind() byte           { return PongMsg }
func (p *Pong) SetReqID(id []byte) { p.ReqID = id }

func (*Findnode) Name() string         { return "FINDNODE/v5" }
func (*Findnode) Kind() byte           { return FindnodeMsg }
func (p *Findnode) SetReqID(id []byte) { p.ReqID = id }

func (*Nodes) Name() string         { return "NODES/v5" }
func (*Nodes) Kind() byte           { return NodesMsg }
func (p *Nodes) SetReqID(id []byte) { p.ReqID = id }

func (*TalkRequest) Name() string         { return "TALKREQ/v5" }
func (*TalkRequest) Kind() byte           { return TalkRequestMsg }
func (p *TalkRequest) SetReqID(id []byte) { p.ReqID = id }

func (*TalkResponse) Name() string         { return "TALKRESP/v5" }
func (*TalkResponse) Kind() byte           { return TalkResponseMsg }
func (p *TalkResponse) SetReqID(id []byte) { p.ReqID = id }

func (*RequestTicket) Name() string         { return "REQTICKET/v5" }
func (*RequestTicket) Kind() byte           { return RequestTicketMsg }
func (p *RequestTicket) SetReqID(id []byte) { p.ReqID = id }

func (*Regtopic) Name() string         { return "REGTOPIC/v5" }
func (*Regtopic) Kind() byte           { return RegtopicMsg }
func (p *Regtopic) SetReqID(id []byte) { p.ReqID = id }

func (*Ticket) Name() string         { return "TICKET/v5" }
func (*Ticket) Kind() byte           { return TicketMsg }
func (p *Ticket) SetReqID(id []byte) { p.ReqID = id }

func (*Regconfirmation) Name() string         { return "REGCONFIRMATION/v5" }
func (*Regconfirmation) Kind() byte           { return RegconfirmationMsg }
func (p *Regconfirmation) SetReqID(id []byte) { p.ReqID = id }

func (*TopicQuery) Name() string         { return "TOPICQUERY/v5" }
func (*TopicQuery) Kind() byte           { return TopicQueryMsg }
func (p *TopicQuery) SetReqID(id []byte) { p.ReqID = id }
