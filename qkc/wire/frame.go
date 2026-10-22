// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// Package wire defines frame codecs for internal cluster and external peer
// communication.
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// ClusterFrame is the common frame used by BaseConn for internal cluster
// communication. ClusterPeerID routes traffic through a virtual peer connection
// on Master↔chain links; it is zero for ordinary cluster traffic and is omitted
// entirely on direct chain↔chain links.
//
// The codec selected for the connection determines whether ClusterPeerID is
// present on the wire.
type ClusterFrame struct {
	ClusterPeerID uint64
	Opcode        byte
	RPCID         uint64 // Request ID; responses echo it. Zero means non-RPC.
	Payload       []byte
}

// P2PFrame is an external cluster↔cluster message carried as one devp2p
// protocol message. ChainID selects the destination chain; the devp2p message
// boundary supplies the frame length.
//
// Wire payload: [4B chain_id][1B opcode][8B rpc_id][payload].
type P2PFrame struct {
	ChainID uint32
	Opcode  byte
	RPCID   uint64
	Payload []byte
}

const (
	opcodeSize      = 1
	rpcIDSize       = 8
	frameHeader     = 4 // internal TCP payload length prefix
	frameBodyHeader = opcodeSize + rpcIDSize
)

// ReadClusterFrame reads one direct chain↔chain internal TCP frame, which does
// not carry a ClusterPeerID. maxPayloadLen == 0 disables payload-size checking.
func ReadClusterFrame(r io.Reader, maxPayloadLen uint32) (*ClusterFrame, error) {
	return readClusterFrame(r, maxPayloadLen, false)
}

// WriteClusterFrame writes one direct chain↔chain internal TCP frame, which
// does not carry a ClusterPeerID.
func WriteClusterFrame(w io.Writer, frame *ClusterFrame) error {
	return writeClusterFrame(w, frame, false)
}

// ReadClusterFrameWithPeerID reads one Master↔chain internal TCP frame. The
// 8-byte ClusterPeerID routes a virtual peer command through the connection.
// maxPayloadLen == 0 disables payload-size checking.
func ReadClusterFrameWithPeerID(r io.Reader, maxPayloadLen uint32) (*ClusterFrame, error) {
	return readClusterFrame(r, maxPayloadLen, true)
}

// WriteClusterFrameWithPeerID writes one Master↔chain internal TCP frame.
func WriteClusterFrameWithPeerID(w io.Writer, frame *ClusterFrame) error {
	return writeClusterFrame(w, frame, true)
}

func readClusterFrame(r io.Reader, maxPayloadLen uint32, hasPeerID bool) (*ClusterFrame, error) {
	var payloadLen uint32
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("reading cluster frame length: %w", err)
	}
	if maxPayloadLen != 0 && payloadLen > maxPayloadLen {
		return nil, fmt.Errorf("cluster frame payload too large: %d > %d", payloadLen, maxPayloadLen)
	}

	frame := new(ClusterFrame)
	if hasPeerID {
		if err := binary.Read(r, binary.BigEndian, &frame.ClusterPeerID); err != nil {
			return nil, fmt.Errorf("reading cluster peer ID: %w", normalizeTruncatedEOF(err))
		}
	}

	body := make([]byte, frameBodyHeader+int(payloadLen))
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("reading cluster frame body (payload_len=%d): %w", payloadLen, normalizeTruncatedEOF(err))
	}
	message, err := unmarshalFrameBody(body)
	if err != nil {
		return nil, err
	}
	frame.Opcode = message.Opcode
	frame.RPCID = message.RPCID
	frame.Payload = message.Payload
	return frame, nil
}

func writeClusterFrame(w io.Writer, frame *ClusterFrame, withPeerID bool) error {
	if frame == nil {
		return errors.New("nil cluster frame")
	}
	if !withPeerID && frame.ClusterPeerID != 0 {
		return errors.New("cluster peer ID is not supported by this frame codec")
	}
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return errors.New("payload too large")
	}

	body := marshalFrameBody(frame.Opcode, frame.RPCID, frame.Payload)
	extra := 0
	if withPeerID {
		extra = 8
	}
	buf := make([]byte, frameHeader+extra+len(body))
	binary.BigEndian.PutUint32(buf[:frameHeader], uint32(len(frame.Payload)))
	if withPeerID {
		binary.BigEndian.PutUint64(buf[frameHeader:frameHeader+8], frame.ClusterPeerID)
	}
	copy(buf[frameHeader+extra:], body)
	_, err := w.Write(buf)
	return err
}

// ReadP2PFrame reads one complete devp2p message payload. The devp2p message
// boundary supplies the frame length, so this codec reads to EOF without an
// additional length prefix. maxPayloadLen == 0 uses the maximum uint32 payload
// size.
func ReadP2PFrame(r io.Reader, maxPayloadLen uint32) (*P2PFrame, error) {
	if maxPayloadLen == 0 {
		maxPayloadLen = math.MaxUint32
	}
	maxRead := int64(maxPayloadLen) + 4 + frameBodyHeader
	// Read one byte beyond the limit to detect an oversized message.
	data, err := io.ReadAll(io.LimitReader(r, maxRead+1))
	if err != nil {
		return nil, fmt.Errorf("reading P2P frame: %w", err)
	}
	if int64(len(data)) > maxRead {
		return nil, fmt.Errorf("P2P frame payload too large: maximum %d", maxPayloadLen)
	}
	if len(data) < 4+frameBodyHeader {
		return nil, fmt.Errorf("P2P frame too short: %d < %d", len(data), 4+frameBodyHeader)
	}

	message, err := unmarshalFrameBody(data[4:])
	if err != nil {
		return nil, err
	}
	return &P2PFrame{
		ChainID: binary.BigEndian.Uint32(data[:4]),
		Opcode:  message.Opcode,
		RPCID:   message.RPCID,
		Payload: message.Payload,
	}, nil
}

// WriteP2PFrame writes one external cluster↔cluster message payload. The caller
// sends these bytes as one devp2p message; this codec adds no length prefix.
func WriteP2PFrame(w io.Writer, frame *P2PFrame) error {
	if frame == nil {
		return errors.New("nil P2P frame")
	}
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return errors.New("payload too large")
	}

	body := marshalFrameBody(frame.Opcode, frame.RPCID, frame.Payload)
	buf := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(buf[:4], frame.ChainID)
	copy(buf[4:], body)
	_, err := w.Write(buf)
	return err
}

type frameMessage struct {
	Opcode  byte
	RPCID   uint64
	Payload []byte
}

func marshalFrameBody(opcode byte, rpcID uint64, payload []byte) []byte {
	body := make([]byte, frameBodyHeader+len(payload))
	body[0] = opcode
	binary.BigEndian.PutUint64(body[1:frameBodyHeader], rpcID)
	copy(body[frameBodyHeader:], payload)
	return body
}

func unmarshalFrameBody(body []byte) (frameMessage, error) {
	if len(body) < frameBodyHeader {
		return frameMessage{}, fmt.Errorf("frame body too short: %d < %d", len(body), frameBodyHeader)
	}
	return frameMessage{
		Opcode:  body[0],
		RPCID:   binary.BigEndian.Uint64(body[1:frameBodyHeader]),
		Payload: body[frameBodyHeader:],
	}, nil
}

// normalizeTruncatedEOF preserves EOF before a frame starts as a graceful
// disconnect. EOF after frame bytes have been read means truncation and becomes
// io.ErrUnexpectedEOF.
func normalizeTruncatedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
