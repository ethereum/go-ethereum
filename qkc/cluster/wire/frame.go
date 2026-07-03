// Copyright 2026-2027, QuarkChain.

// Package wire implements a binary frame codec compatible with pyquarkchain.
//
// Wire format (per frame):
//
//	[4B payload_len] [metaSize B metadata] [1B opcode] [8B rpc_id] [payload bytes]
//
// payload_len is the length of payload bytes only.
// metadata size depends on metaSize parameter:
//   - 12 bytes: ClusterMetadata (branch uint32 + cluster_peer_id uint64) for master↔slave
//   - 0 bytes:  no metadata for slave↔slave traffic
package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Frame is a complete protocol frame.
// Wire layout after metadata: [1B opcode][8B rpc_id][N bytes payload]
type Frame struct {
	Meta    ClusterMetadata
	Opcode  byte
	RPCID   uint64
	Payload []byte
}

const (
	metaSize    = 12 // ClusterMetadata.get_byte_size() = 12 (branch 4B + cluster_peer_id 8B)
	opcodeSize  = 1
	rpcIDSize   = 8
	frameHeader = 4 // payload_len prefix
)

// ReadFrame reads a frame with 12-byte ClusterMetadata.
// maxPayloadLen == 0 disables payload-size checking.
func ReadFrame(r io.Reader, maxPayloadLen uint32) (*Frame, error) {
	return readFrame(r, metaSize, maxPayloadLen)
}

// ReadFrameNoMeta reads a frame with 0-byte metadata.
// maxPayloadLen == 0 disables payload-size checking.
func ReadFrameNoMeta(r io.Reader, maxPayloadLen uint32) (*Frame, error) {
	return readFrame(r, 0, maxPayloadLen)
}

// readFrame is the underlying frame reader.
func readFrame(r io.Reader, metaSize int, maxPayloadLen uint32) (*Frame, error) {
	// 1. Read 4-byte big-endian payload length
	var payloadLen uint32
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("reading frame length: %w", err)
	}

	if maxPayloadLen != 0 && payloadLen > maxPayloadLen {
		return nil, fmt.Errorf("frame payload too large: %d > %d", payloadLen, maxPayloadLen)
	}

	// 2. Read metadata (size depends on metadata_class)
	var meta ClusterMetadata
	if metaSize > 0 {
		if metaSize != 12 {
			return nil, fmt.Errorf("unsupported metaSize %d (only 12 supported)", metaSize)
		}
		metaBuf := make([]byte, metaSize)
		if _, err := io.ReadFull(r, metaBuf); err != nil {
			return nil, fmt.Errorf("reading metadata: %w", err)
		}
		meta = ClusterMetadata{
			Branch:        binary.BigEndian.Uint32(metaBuf[0:4]),
			ClusterPeerID: binary.BigEndian.Uint64(metaBuf[4:12]),
		}
	}

	// 3. Read raw_data_without_size: [1B opcode][8B rpc_id][N bytes payload]
	bodySize := opcodeSize + rpcIDSize + int(payloadLen)
	body := make([]byte, bodySize)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("reading frame body (payload_len=%d): %w", payloadLen, err)
	}

	return &Frame{
		Meta:    meta,
		Opcode:  body[0],
		RPCID:   binary.BigEndian.Uint64(body[1:9]),
		Payload: body[9:],
	}, nil
}

// WriteFrame serializes f with 12-byte ClusterMetadata and writes it to w.
func WriteFrame(w io.Writer, f *Frame) error {
	return writeFrameWithMetaSize(w, f, metaSize)
}

// WriteFrameNoMeta writes a frame with 0-byte metadata.
func WriteFrameNoMeta(w io.Writer, f *Frame) error {
	return writeFrameWithMetaSize(w, f, 0)
}

// writeFrameWithMetaSize serializes f with the given metadata size and writes it to w.
func writeFrameWithMetaSize(w io.Writer, f *Frame, metaSize int) error {
	payloadLen := uint32(len(f.Payload))
	if int(payloadLen) != len(f.Payload) {
		return errors.New("payload too large")
	}

	total := frameHeader + metaSize + opcodeSize + rpcIDSize + int(payloadLen)
	buf := make([]byte, total)

	binary.BigEndian.PutUint32(buf[0:frameHeader], payloadLen)

	// Metadata
	if metaSize > 0 {
		binary.BigEndian.PutUint32(buf[frameHeader:frameHeader+4], f.Meta.Branch)
		binary.BigEndian.PutUint64(buf[frameHeader+4:frameHeader+metaSize], f.Meta.ClusterPeerID)
	}

	// Opcode
	buf[frameHeader+metaSize] = f.Opcode

	binary.BigEndian.PutUint64(buf[frameHeader+metaSize+opcodeSize:frameHeader+metaSize+opcodeSize+rpcIDSize], f.RPCID)

	// Payload
	copy(buf[frameHeader+metaSize+opcodeSize+rpcIDSize:], f.Payload)

	_, err := w.Write(buf)
	return err
}
