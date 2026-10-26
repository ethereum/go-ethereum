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
	chainIDSize       = 4
	clusterPeerIDSize = 8
	opcodeSize        = 1
	rpcIDSize         = 8
	frameHeader       = 4 // internal TCP payload length prefix
	frameBodyHeader   = opcodeSize + rpcIDSize
)

// MaxFramePayloadLen caps internal frame payloads at 16 MiB to bound upfront
// allocations while accommodating payloads relayed from devp2p messages.
// External messages must separately obey the devp2p limit, including P2P frame
// headers. Larger internal data, such as block lists for sync, must be split
// by size at the protocol level.
const MaxFramePayloadLen = 16 << 20

// ClusterFrame carries one internal TCP message between cluster processes.
//
// Wire layout:
//
//	[4B payload_len][8B cluster_peer_id, Master↔chain only][1B opcode][8B rpc_id][payload]
//
// payload_len counts only payload. ClusterPeerID routes traffic through a
// virtual peer connection on Master↔chain links; it is zero for ordinary
// cluster traffic and is omitted entirely on direct chain↔chain links.
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

	body := make([]byte, frameBodyHeader+int(payloadLen))
	if _, err := io.ReadFull(r, body); err != nil {
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

	extra := 0
	if withPeerID {
		extra = clusterPeerIDSize
	}
	buf := make([]byte, frameHeader+extra+frameBodyHeader+len(frame.Payload))
	binary.BigEndian.PutUint32(buf[:frameHeader], uint32(len(frame.Payload)))
	offset := frameHeader
	if withPeerID {
		binary.BigEndian.PutUint64(buf[offset:offset+clusterPeerIDSize], frame.ClusterPeerID)
		offset += clusterPeerIDSize
	}
	buf[offset] = frame.Opcode
	binary.BigEndian.PutUint64(buf[offset+opcodeSize:offset+frameBodyHeader], frame.RPCID)
	copy(buf[offset+frameBodyHeader:], frame.Payload)
	_, err := w.Write(buf)
	return err
}

// DecodeP2PFrame decodes one complete devp2p message payload. The caller must
// enforce its configured payload size limit and pass exactly one message's bytes.
// The returned Payload aliases data. Copy the payload if it must survive
// changes to or reuse of the input buffer.
func DecodeP2PFrame(data []byte) (*P2PFrame, error) {
	if len(data) < chainIDSize+frameBodyHeader {
		return nil, fmt.Errorf("P2P frame too short: %d < %d", len(data), chainIDSize+frameBodyHeader)
	}

	body := data[chainIDSize:]
	return &P2PFrame{
		ChainID: binary.BigEndian.Uint32(data[:chainIDSize]),
		Opcode:  body[0],
		RPCID:   binary.BigEndian.Uint64(body[1:frameBodyHeader]),
		Payload: body[frameBodyHeader:],
	}, nil
}

// EncodeP2PFrame encodes one external cluster↔cluster message payload. The
// caller enforces the devp2p message size limit; this codec adds no length
// prefix.
func EncodeP2PFrame(frame *P2PFrame) ([]byte, error) {
	if frame == nil {
		return nil, errors.New("nil P2P frame")
	}
	buf := make([]byte, chainIDSize+frameBodyHeader+len(frame.Payload))
	binary.BigEndian.PutUint32(buf[:chainIDSize], frame.ChainID)
	buf[chainIDSize] = frame.Opcode
	binary.BigEndian.PutUint64(buf[chainIDSize+opcodeSize:chainIDSize+frameBodyHeader], frame.RPCID)
	copy(buf[chainIDSize+frameBodyHeader:], frame.Payload)
	return buf, nil
}

// WriteP2PFrame writes one P2P frame to w.
func WriteP2PFrame(w io.Writer, frame *P2PFrame) error {
	buf, err := EncodeP2PFrame(frame)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
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
