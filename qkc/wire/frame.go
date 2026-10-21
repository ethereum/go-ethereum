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

// Package wire defines frame codecs for internal
// cluster and external peer communication.
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// Frame is the common message body used by all connection types.
type Frame struct {
	Opcode  byte
	RPCID   uint64 // Request ID; responses echo it. Zero means the frame is not an RPC.
	Payload []byte
}

// ChainFrame is an internal TCP frame sent over a direct chain↔chain
// connection. The remote chain is identified by the connection.
//
// Wire format: [4B payload_len][1B opcode][8B rpc_id][payload]. payload_len
// counts payload bytes only.
type ChainFrame struct {
	Frame
}

// MasterFrame is an internal TCP frame sent over a Master↔chain connection.
// The chain is identified by the connection. ClusterPeerID identifies the
// virtual connection for an external peer mapped by the Master to this chain.
// Zero means ordinary Master↔chain traffic; non-zero routes the peer command
// through that virtual connection.
//
// Wire format: [4B payload_len][8B cluster_peer_id][1B opcode][8B rpc_id][payload].
// payload_len counts payload bytes only.
type MasterFrame struct {
	ClusterPeerID uint64
	Frame
}

// PeerFrame is an external Master↔Master message. ChainID selects the target
// chain; zero is reserved for Master-level messages. The devp2p message
// boundary frames this value, so the codec does not add a length prefix.
//
// Wire format: [4B chain_id][1B opcode][8B rpc_id][payload].
type PeerFrame struct {
	ChainID uint32
	Frame
}

const (
	opcodeSize      = 1
	rpcIDSize       = 8
	frameHeader     = 4 // payload_len prefix
	frameBodyHeader = opcodeSize + rpcIDSize
)

// ReadChainFrame reads one chain↔chain internal TCP frame.
// maxPayloadLen == 0 disables payload-size checking.
func ReadChainFrame(r io.Reader, maxPayloadLen uint32) (*ChainFrame, error) {
	body, _, err := readInternalFrame(r, maxPayloadLen, false)
	if err != nil {
		return nil, fmt.Errorf("reading chain frame: %w", err)
	}
	return &ChainFrame{Frame: *body}, nil
}

// WriteChainFrame writes one chain↔chain internal TCP frame.
func WriteChainFrame(w io.Writer, frame *ChainFrame) error {
	if frame == nil {
		return errors.New("nil chain frame")
	}
	if err := writeInternalFrame(w, frame.Frame, nil); err != nil {
		return fmt.Errorf("writing chain frame: %w", err)
	}
	return nil
}

// ReadMasterFrame reads one Master↔chain internal TCP frame.
// maxPayloadLen == 0 disables payload-size checking.
func ReadMasterFrame(r io.Reader, maxPayloadLen uint32) (*MasterFrame, error) {
	body, peerID, err := readInternalFrame(r, maxPayloadLen, true)
	if err != nil {
		return nil, fmt.Errorf("reading master frame: %w", err)
	}
	return &MasterFrame{
		ClusterPeerID: peerID,
		Frame:         *body,
	}, nil
}

// WriteMasterFrame writes one Master↔chain internal TCP frame.
func WriteMasterFrame(w io.Writer, frame *MasterFrame) error {
	if frame == nil {
		return errors.New("nil master frame")
	}
	if err := writeInternalFrame(w, frame.Frame, &frame.ClusterPeerID); err != nil {
		return fmt.Errorf("writing master frame: %w", err)
	}
	return nil
}

// ReadPeerFrame reads one complete devp2p message payload. The devp2p message
// boundary supplies the frame length, so this codec reads to EOF without a
// TCP length prefix. maxPayloadLen == 0 uses the maximum uint32 payload size.
func ReadPeerFrame(r io.Reader, maxPayloadLen uint32) (*PeerFrame, error) {
	if maxPayloadLen == 0 {
		maxPayloadLen = math.MaxUint32
	}
	maxRead := int64(maxPayloadLen) + 4 + frameBodyHeader
	// Read one byte beyond the limit to detect an oversized message.
	data, err := io.ReadAll(io.LimitReader(r, maxRead+1))
	if err != nil {
		return nil, fmt.Errorf("reading peer frame: %w", err)
	}
	if int64(len(data)) > maxRead {
		return nil, fmt.Errorf("peer frame payload too large: maximum %d", maxPayloadLen)
	}
	if len(data) < 4+frameBodyHeader {
		return nil, fmt.Errorf("peer frame too short: %d < %d", len(data), 4+frameBodyHeader)
	}

	body, err := unmarshalFrameBody(data[4:])
	if err != nil {
		return nil, err
	}
	return &PeerFrame{ChainID: binary.BigEndian.Uint32(data[:4]), Frame: body}, nil
}

// WritePeerFrame writes one external Master↔Master message payload. The caller
// sends these bytes as one devp2p message; this codec adds no length prefix.
func WritePeerFrame(w io.Writer, frame *PeerFrame) error {
	if frame == nil {
		return errors.New("nil peer frame")
	}
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return errors.New("payload too large")
	}

	body := marshalFrameBody(frame.Frame)
	buf := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(buf[:4], frame.ChainID)
	copy(buf[4:], body)
	_, err := w.Write(buf)
	return err
}

func readInternalFrame(r io.Reader, maxPayloadLen uint32, hasVirtualPeerID bool) (*Frame, uint64, error) {
	// 1. Read 4-byte big-endian payload length
	var payloadLen uint32
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, 0, fmt.Errorf("reading length: %w", err)
	}

	if maxPayloadLen != 0 && payloadLen > maxPayloadLen {
		return nil, 0, fmt.Errorf("frame payload too large: %d > %d", payloadLen, maxPayloadLen)
	}

	// 2. Read cluster peer ID when this connection routes peer traffic.
	var peerID uint64
	if hasVirtualPeerID {
		if err := binary.Read(r, binary.BigEndian, &peerID); err != nil {
			return nil, 0, fmt.Errorf("reading cluster peer ID: %w", normalizeTruncatedEOF(err))
		}
	}

	// 3. Read body
	buf := make([]byte, frameBodyHeader+int(payloadLen))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, 0, fmt.Errorf("reading frame body (payload_len=%d): %w", payloadLen, normalizeTruncatedEOF(err))
	}
	body, err := unmarshalFrameBody(buf)
	return &body, peerID, err
}

func writeInternalFrame(w io.Writer, frame Frame, virtualPeerID *uint64) error {
	if uint64(len(frame.Payload)) > math.MaxUint32 {
		return errors.New("payload too large")
	}

	body := marshalFrameBody(frame)

	extra := 0
	if virtualPeerID != nil {
		extra = 8
	}
	buf := make([]byte, frameHeader+extra+len(body))
	binary.BigEndian.PutUint32(buf[:frameHeader], uint32(len(frame.Payload)))
	if virtualPeerID != nil {
		binary.BigEndian.PutUint64(buf[frameHeader:frameHeader+8], *virtualPeerID)
	}
	copy(buf[frameHeader+extra:], body)
	_, err := w.Write(buf)
	return err
}

func marshalFrameBody(frame Frame) []byte {
	body := make([]byte, frameBodyHeader+len(frame.Payload))
	body[0] = frame.Opcode
	binary.BigEndian.PutUint64(body[1:frameBodyHeader], frame.RPCID)
	copy(body[frameBodyHeader:], frame.Payload)
	return body
}

func unmarshalFrameBody(body []byte) (Frame, error) {
	if len(body) < frameBodyHeader {
		return Frame{}, fmt.Errorf("frame body too short: %d < %d", len(body), frameBodyHeader)
	}
	return Frame{Opcode: body[0], RPCID: binary.BigEndian.Uint64(body[1:frameBodyHeader]), Payload: body[frameBodyHeader:]}, nil
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
