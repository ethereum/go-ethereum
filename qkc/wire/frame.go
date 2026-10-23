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

// RootChainID is reserved for root-chain traffic handled by the Master. It is
// never a valid chain ID.
const RootChainID uint32 = math.MaxUint32

const (
	opcodeSize      = 1
	rpcIDSize       = 8
	frameHeader     = 4 // internal TCP payload length prefix
	frameBodyHeader = opcodeSize + rpcIDSize
)

// MaxFramePayloadLen caps internal payloads at 2 GB on every architecture.
// The frame headers still fit in a 32-bit int at this limit.
const MaxFramePayloadLen = 2_000_000_000

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
// protocol message. The devp2p message boundary supplies the frame length.
//
// Wire payload: [4B chain_id][1B opcode][8B rpc_id][payload].
type P2PFrame struct {
	// ChainID selects the destination chain. RootChainID routes root-chain
	// traffic to the Master.
	ChainID uint32
	Opcode  byte
	RPCID   uint64
	Payload []byte
}

// ReadClusterFrame reads one direct chain↔chain internal TCP frame, which does
// not carry a ClusterPeerID. maxPayloadLen must be in [1, MaxFramePayloadLen].
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
// maxPayloadLen must be in [1, MaxFramePayloadLen].
func ReadClusterFrameWithPeerID(r io.Reader, maxPayloadLen uint32) (*ClusterFrame, error) {
	return readClusterFrame(r, maxPayloadLen, true)
}

// WriteClusterFrameWithPeerID writes one Master↔chain internal TCP frame.
func WriteClusterFrameWithPeerID(w io.Writer, frame *ClusterFrame) error {
	return writeClusterFrame(w, frame, true)
}

func readClusterFrame(r io.Reader, maxPayloadLen uint32, hasPeerID bool) (*ClusterFrame, error) {
	if maxPayloadLen == 0 || maxPayloadLen > MaxFramePayloadLen {
		return nil, fmt.Errorf("cluster frame maxPayloadLen must be between 1 and %d: %d", MaxFramePayloadLen, maxPayloadLen)
	}
	var payloadLen uint32
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("reading cluster frame length: %w", err)
	}
	if payloadLen > maxPayloadLen {
		return nil, fmt.Errorf("cluster frame payload too large: %d > %d", payloadLen, maxPayloadLen)
	}

	frame := new(ClusterFrame)
	if hasPeerID {
		if err := binary.Read(r, binary.BigEndian, &frame.ClusterPeerID); err != nil {
			return nil, fmt.Errorf("reading cluster peer ID: %w", normalizeTruncatedEOF(err))
		}
	}

	bodyLen := frameBodyHeader + int(payloadLen)
	body, err := io.ReadAll(io.LimitReader(r, int64(bodyLen)))
	if len(body) != bodyLen {
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return nil, fmt.Errorf("reading cluster frame body (payload_len=%d): %w", payloadLen, normalizeTruncatedEOF(err))
	}
	frame.Opcode = body[0]
	frame.RPCID = binary.BigEndian.Uint64(body[1:frameBodyHeader])
	frame.Payload = body[frameBodyHeader:]
	return frame, nil
}

func writeClusterFrame(w io.Writer, frame *ClusterFrame, withPeerID bool) error {
	if frame == nil {
		return errors.New("nil cluster frame")
	}
	if !withPeerID && frame.ClusterPeerID != 0 {
		return errors.New("cluster peer ID is not supported by this frame codec")
	}
	if len(frame.Payload) > MaxFramePayloadLen {
		return errors.New("payload too large")
	}

	body := make([]byte, frameBodyHeader+len(frame.Payload))
	body[0] = frame.Opcode
	binary.BigEndian.PutUint64(body[1:frameBodyHeader], frame.RPCID)
	copy(body[frameBodyHeader:], frame.Payload)
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

// DecodeP2PFrame decodes one complete devp2p message payload. The caller must
// enforce its configured payload size limit and pass exactly one message's bytes.
// The returned Payload aliases data. Copy the payload if it must survive
// changes to or reuse of the input buffer.
func DecodeP2PFrame(data []byte) (*P2PFrame, error) {
	if len(data) < 4+frameBodyHeader {
		return nil, fmt.Errorf("P2P frame too short: %d < %d", len(data), 4+frameBodyHeader)
	}

	body := data[4:]
	return &P2PFrame{
		ChainID: binary.BigEndian.Uint32(data[:4]),
		Opcode:  body[0],
		RPCID:   binary.BigEndian.Uint64(body[1:frameBodyHeader]),
		Payload: body[frameBodyHeader:],
	}, nil
}

// WriteP2PFrame writes one external cluster↔cluster message payload. The caller
// enforces the devp2p message size limit and sends these bytes as one message;
// this codec adds no length prefix.
func WriteP2PFrame(w io.Writer, frame *P2PFrame) error {
	if frame == nil {
		return errors.New("nil P2P frame")
	}
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return errors.New("payload too large")
	}

	body := make([]byte, frameBodyHeader+len(frame.Payload))
	body[0] = frame.Opcode
	binary.BigEndian.PutUint64(body[1:frameBodyHeader], frame.RPCID)
	copy(body[frameBodyHeader:], frame.Payload)
	buf := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(buf[:4], frame.ChainID)
	copy(buf[4:], body)
	_, err := w.Write(buf)
	return err
}

// normalizeTruncatedEOF converts io.EOF to io.ErrUnexpectedEOF. Use it only for
// reads after the frame length has been consumed, where EOF means the frame was
// truncated. The length read must not use it, so that EOF before a frame starts
// stays io.EOF and signals a graceful disconnect.
func normalizeTruncatedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
