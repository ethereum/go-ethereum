// Package slave: binary frame codec compatible with pyquarkchain protocol.py.
//
// Wire format (per-frame):
//
//	[4B payload_len] [metaSize B metadata] [1B opcode] [8B rpc_id] [payload_len bytes]
//
// This matches Python's Connection.read_metadata_and_raw_data /
// Connection.write_raw_data exactly:
//
//	Python: protocol.py lines 285-308
//
// Metadata sizes (matching Python Metadata subclasses):
//
//	ClusterMetadata.get_byte_size() = 12   (branch 4B + cluster_peer_id 8B)
//	    Used for master ↔ slave traffic.
//
//	Metadata.get_byte_size()        =  0   (base class)
//	    Used for slave ↔ slave traffic (SlaveConnection).
//
//	P2PMetadata.get_byte_size()     =  4   (branch 4B)
//	    Used for inter-cluster P2P.  Handled by Python master only;
//	    Go slave never sends or receives P2PMetadata frames.
//
// payload_len definition (matching Python line 305):
//
//	cmd_length_bytes = (len(raw_data) - 8 - 1).to_bytes(4, "big")
//
// That is, payload_len = len(raw_data) - 9, where raw_data is
// [1B opcode][8B rpc_id][N bytes payload].
package slave

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Metadata is the 12-byte frame header carrying routing information.
// Matches Python's ClusterMetadata:
//
//	class ClusterMetadata(Metadata):
//	    FIELDS = [("branch", Branch), ("cluster_peer_id", uint64)]
//	    @staticmethod
//	    def get_byte_size(): return 12
type Metadata struct {
	Branch        uint32 // shard identifier (Python: Branch = uint32)
	ClusterPeerID uint64 // 0 = cluster RPC (master commands), ≠0 = specific external peer
}

// Frame is a complete protocol frame.
// raw_data layout on wire: [1B opcode][8B rpc_id][N bytes payload]
type Frame struct {
	Meta    Metadata
	Opcode  byte
	RPCID   uint64
	Payload []byte
}

const (
	metaSize      = 12 // ClusterMetadata.get_byte_size() = 12 (branch 4B + cluster_peer_id 8B)
	opcodeSize    = 1
	rpcIDSize     = 8
	frameHeader   = 4                                               // payload_len prefix
	totalOverhead = frameHeader + metaSize + opcodeSize + rpcIDSize // 4+12+1+8 = 25
)

// ReadFrame reads a frame with 12-byte ClusterMetadata.
//
// Matches Python's Connection.read_metadata_and_raw_data (protocol.py lines 285-300)
// with metadata_class = ClusterMetadata (get_byte_size() == 12).
//
// Used for master ↔ slave traffic.
// For slave ↔ slave traffic, use ReadFrameNoMeta (0-byte metadata).
func ReadFrame(r io.Reader) (*Frame, error) {
	return readFrameWithMetaSize(r, metaSize)
}

// ReadFrameNoMeta reads a frame with 0-byte Metadata.
//
// Matches Python's SlaveConnection which uses metadata_class = Metadata
// (get_byte_size() == 0).
//
// Used for slave ↔ slave direct TCP traffic.
func ReadFrameNoMeta(r io.Reader) (*Frame, error) {
	return readFrameWithMetaSize(r, 0)
}

// readFrameWithMetaSize is the underlying frame reader.
//
// Wire layout (matching Python read_metadata_and_raw_data):
//
//	size_bytes = await read_fully(4)               → 4B payload_len (big-endian)
//	metadata_bytes = await read_fully(metaSize)    → metaSize B metadata
//	raw_data_without_size = await read_fully(1+8+size) → opcode + rpc_id + payload
func readFrameWithMetaSize(r io.Reader, metaSize int) (*Frame, error) {
	// 1. Read 4-byte big-endian payload length
	//    Python: size_bytes = await self.__read_fully(4, allow_eof=True)
	//            size = int.from_bytes(size_bytes, byteorder="big")
	var payloadLen uint32
	if err := binary.Read(r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("reading frame length: %w", err)
	}

	// 2. Read metadata (size depends on metadata_class)
	//    Python: metadata_bytes = await self.__read_fully(self.metadata_class.get_byte_size())
	//            metadata = self.metadata_class.deserialize(metadata_bytes)
	var meta Metadata
	if metaSize > 0 {
		if metaSize != 12 {
			return nil, fmt.Errorf("unsupported metaSize %d (only 0 or 12 supported)", metaSize)
		}
		metaBuf := make([]byte, metaSize)
		if _, err := io.ReadFull(r, metaBuf); err != nil {
			return nil, fmt.Errorf("reading metadata: %w", err)
		}
		meta = Metadata{
			Branch:        binary.BigEndian.Uint32(metaBuf[0:4]),
			ClusterPeerID: binary.BigEndian.Uint64(metaBuf[4:12]),
		}
	}

	// 3. Read raw_data_without_size: [1B opcode][8B rpc_id][N bytes payload]
	//    Python: raw_data_without_size = await self.__read_fully(1 + 8 + size)
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
//
// Matches Python's Connection.write_raw_data (protocol.py lines 302-308)
// with metadata_class = ClusterMetadata.
//
// Used for master ↔ slave traffic.
func WriteFrame(w io.Writer, f *Frame) error {
	return writeFrameWithMetaSize(w, f, metaSize)
}

// WriteFrameNoMeta writes a frame with 0-byte Metadata.
//
// Matches Python's SlaveConnection which uses metadata_class = Metadata
// (get_byte_size() == 0).
//
// Used for slave ↔ slave traffic.
func WriteFrameNoMeta(w io.Writer, f *Frame) error {
	return writeFrameWithMetaSize(w, f, 0)
}

// writeFrameWithMetaSize serializes f with the given metadata size and writes
// it to w.
//
// Wire layout (matching Python write_raw_data, protocol.py lines 302-308):
//
//	cmd_length_bytes = (len(raw_data) - 8 - 1).to_bytes(4, "big")
//	self.writer.write(cmd_length_bytes)       → 4B payload_len
//	self.writer.write(metadata.serialize())   → metaSize B
//	self.writer.write(raw_data)               → [1B opcode][8B rpc_id][payload]
func writeFrameWithMetaSize(w io.Writer, f *Frame, metaSize int) error {
	payloadLen := uint32(len(f.Payload))
	if int(payloadLen) != len(f.Payload) {
		return errors.New("payload too large")
	}

	// Build the buffer in one write (Python writes in 3 chunks, but the wire
	// bytes are identical).
	total := frameHeader + metaSize + opcodeSize + rpcIDSize + int(payloadLen)
	buf := make([]byte, total)

	// Frame length (payload only): matches Python's cmd_length_bytes
	binary.BigEndian.PutUint32(buf[0:frameHeader], payloadLen)

	// Metadata
	if metaSize > 0 {
		binary.BigEndian.PutUint32(buf[frameHeader:frameHeader+4], f.Meta.Branch)
		binary.BigEndian.PutUint64(buf[frameHeader+4:frameHeader+metaSize], f.Meta.ClusterPeerID)
	}

	// Opcode
	buf[frameHeader+metaSize] = f.Opcode

	// RPC ID (big-endian, matches Python: rpc_id.to_bytes(8, "big"))
	binary.BigEndian.PutUint64(buf[frameHeader+metaSize+opcodeSize:frameHeader+metaSize+opcodeSize+rpcIDSize], f.RPCID)

	// Payload
	copy(buf[frameHeader+metaSize+opcodeSize+rpcIDSize:], f.Payload)

	_, err := w.Write(buf)
	return err
}

// MarshalMetadata serializes Metadata into its 12-byte wire representation.
func MarshalMetadata(m Metadata) []byte {
	buf := make([]byte, metaSize)
	binary.BigEndian.PutUint32(buf[0:4], m.Branch)
	binary.BigEndian.PutUint64(buf[4:12], m.ClusterPeerID)
	return buf
}

// UnmarshalMetadata deserializes a 12-byte wire representation into Metadata.
func UnmarshalMetadata(b []byte) (Metadata, error) {
	if len(b) != metaSize {
		return Metadata{}, fmt.Errorf("metadata must be %d bytes, got %d", metaSize, len(b))
	}
	return Metadata{
		Branch:        binary.BigEndian.Uint32(b[0:4]),
		ClusterPeerID: binary.BigEndian.Uint64(b[4:12]),
	}, nil
}

// ── Convenience wrappers (used in tests) ─────────────────────────────────────

// ReadFrameFromReader wraps r in a bufio.Reader.
func ReadFrameFromReader(r io.Reader) (*Frame, error) {
	return ReadFrame(bufio.NewReader(r))
}

// WriteFrameToWriter wraps w with a bufio.Writer and flushes.
func WriteFrameToWriter(w io.Writer, frame *Frame) error {
	bw := bufio.NewWriter(w)
	if err := WriteFrame(bw, frame); err != nil {
		return err
	}
	return bw.Flush()
}

// ReadFrameNoMetaFromReader wraps r for ReadFrameNoMeta.
func ReadFrameNoMetaFromReader(r io.Reader) (*Frame, error) {
	return ReadFrameNoMeta(bufio.NewReader(r))
}

// WriteFrameNoMetaToWriter wraps w for WriteFrameNoMeta.
func WriteFrameNoMetaToWriter(w io.Writer, frame *Frame) error {
	bw := bufio.NewWriter(w)
	if err := WriteFrameNoMeta(bw, frame); err != nil {
		return err
	}
	return bw.Flush()
}
