package slave

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

// =============================================================================
// Python compatibility reference vectors
//
// Each vector below is the exact wire bytes a Python peer would send/receive
// for a given frame.  Source: qkc/quarkchain/protocol.py
//   - Connection.write_raw_data  (lines 302-308)
//   - Connection.read_metadata_and_raw_data  (lines 285-300)
//
// Wire layout: [4B payload_len] [metaSize B metadata] [1B opcode] [8B rpc_id] [payload]
//
// metadata_class:
//   ClusterMetadata (12B) for master↔slave traffic
//   Metadata (0B)      for slave↔slave traffic
// =============================================================================

// pingMasterWire: meta=(branch=0, peer=0), opcode=0x81 (PING), rpc_id=1, payload=empty
//
//	Equivalent Python: write_raw_command(op=ClusterOp.PING, cmd_data=b"", rpc_id=1, metadata=ClusterMetadata(0, 0))
//	payload_len = 0
//	00000000 | 00000000 0000000000000000 | 81 | 0000000000000001
var pingMasterWire = "00000000" + "00000000" + "0000000000000000" + "81" + "0000000000000001"

// pongMasterWire: meta=(branch=0, peer=0), opcode=0x82 (PONG), rpc_id=1, payload=empty
//
//	Equivalent Python: write_raw_command(op=ClusterOp.PONG, cmd_data=b"", rpc_id=1, metadata=ClusterMetadata(0, 0))
//	00000000 | 00000000 0000000000000000 | 82 | 0000000000000001
var pongMasterWire = "00000000" + "00000000" + "0000000000000000" + "82" + "0000000000000001"

// peerNewBlockWire: meta=(branch=1, peer=12345), opcode=0x01 (NEW_MINOR_BLOCK_HEADER_LIST), rpc_id=0, payload=12B
//
//	cluster_peer_id=12345=0x3039, rpc_id=0 (NON-RPC fire-and-forget)
//	payload = 02 00 00 00 (list len=2) + a1b2c3d4 + e5f60718
//	0000000c | 00000001 0000000000003039 | 01 | 0000000000000000 | 02000000a1b2c3d4e5f60718
var peerNewBlockWire = "0000000c" +
	"00000001" + "0000000000003039" +
	"01" + "0000000000000000" +
	"02000000a1b2c3d4e5f60718"

// xshardWire: 0-byte metadata, opcode=0x93 (ADD_XSHARD_TX_LIST_REQUEST), rpc_id=42, payload=56B
//
//	Used for slave↔slave direct TCP (Python SlaveConnection, metadata_class=Metadata)
//	payload_len = 56 = 0x38
//	payload = 01 00 00 00 (list len=1) + ff*32 + 00*20
//	00000038 | (no meta) | 93 | 000000000000002a | 01000000 + ff×32 + 00×20
var xshardWire = "00000038" +
	"93" + "000000000000002a" +
	"01000000" +
	strings.Repeat("ff", 32) +
	strings.Repeat("00", 20)

// largePayloadWire: meta=(branch=3, peer=0), opcode=0x10, rpc_id=7, payload=10000×0xAB
//
//	payload_len = 10000 = 0x2710
//	00002710 | 00000003 0000000000000000 | 10 | 0000000000000007 | ab×10000
var largePayloadWire = "00002710" +
	"00000003" + "0000000000000000" +
	"10" + "0000000000000007" +
	strings.Repeat("ab", 10000)

// =============================================================================
// ReadFrame tests — parse Python-generated wire bytes
// =============================================================================

func TestPythonRead_Ping(t *testing.T) {
	wire, _ := hex.DecodeString(pingMasterWire)
	f, err := ReadFrame(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x81 {
		t.Errorf("Opcode: got 0x%02x, want 0x81", f.Opcode)
	}
	if f.RPCID != 1 {
		t.Errorf("RPCID: got %d, want 1", f.RPCID)
	}
	if f.Meta != (Metadata{Branch: 0, ClusterPeerID: 0}) {
		t.Errorf("Meta: got %+v", f.Meta)
	}
	if len(f.Payload) != 0 {
		t.Errorf("Payload len: got %d, want 0", len(f.Payload))
	}
}

func TestPythonRead_Pong(t *testing.T) {
	wire, _ := hex.DecodeString(pongMasterWire)
	f, err := ReadFrame(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x82 {
		t.Errorf("Opcode: got 0x%02x, want 0x82", f.Opcode)
	}
	if f.RPCID != 1 {
		t.Errorf("RPCID: got %d, want 1", f.RPCID)
	}
}

func TestPythonRead_PeerNewBlock(t *testing.T) {
	wire, _ := hex.DecodeString(peerNewBlockWire)
	f, err := ReadFrame(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x01 {
		t.Errorf("Opcode: got 0x%02x, want 0x01", f.Opcode)
	}
	if f.RPCID != 0 {
		t.Errorf("RPCID: got %d, want 0 (non-RPC)", f.RPCID)
	}
	if f.Meta.Branch != 1 {
		t.Errorf("Branch: got %d, want 1", f.Meta.Branch)
	}
	if f.Meta.ClusterPeerID != 12345 {
		t.Errorf("ClusterPeerID: got %d, want 12345", f.Meta.ClusterPeerID)
	}
}

func TestPythonRead_Xshard(t *testing.T) {
	wire, _ := hex.DecodeString(xshardWire)
	f, err := ReadFrameNoMeta(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrameNoMeta: %v", err)
	}
	if f.Opcode != 0x93 {
		t.Errorf("Opcode: got 0x%02x, want 0x93", f.Opcode)
	}
	if f.RPCID != 42 {
		t.Errorf("RPCID: got %d, want 42", f.RPCID)
	}
	if len(f.Payload) != 56 {
		t.Errorf("Payload len: got %d, want 56", len(f.Payload))
	}
}

func TestPythonRead_LargePayload(t *testing.T) {
	wire, _ := hex.DecodeString(largePayloadWire)
	f, err := ReadFrame(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x10 {
		t.Errorf("Opcode: got 0x%02x, want 0x10", f.Opcode)
	}
	if f.RPCID != 7 {
		t.Errorf("RPCID: got %d, want 7", f.RPCID)
	}
	if f.Meta.Branch != 3 {
		t.Errorf("Branch: got %d, want 3", f.Meta.Branch)
	}
	if len(f.Payload) != 10000 {
		t.Errorf("Payload len: got %d, want 10000", len(f.Payload))
	}
	for i, b := range f.Payload {
		if b != 0xAB {
			t.Errorf("Payload[%d]: got 0x%02x, want 0xAB", i, b)
			break
		}
	}
}

// =============================================================================
// WriteFrame tests — Go's wire output must match Python's byte-for-byte.
// This is the strongest compatibility test: any wire-format drift is caught.
// =============================================================================

func TestPythonWrite_Ping(t *testing.T) {
	want, _ := hex.DecodeString(pingMasterWire)
	got := writeFrameForTest(Metadata{Branch: 0, ClusterPeerID: 0}, 0x81, 1, nil)
	if !bytes.Equal(got, want) {
		t.Errorf("WriteFrame mismatch:\n  got  %x\n  want %x", got, want)
	}
}

func TestPythonWrite_PeerNewBlock(t *testing.T) {
	want, _ := hex.DecodeString(peerNewBlockWire)
	payload := []byte{0x02, 0x00, 0x00, 0x00, 0xa1, 0xb2, 0xc3, 0xd4, 0xe5, 0xf6, 0x07, 0x18}
	got := writeFrameForTest(Metadata{Branch: 1, ClusterPeerID: 12345}, 0x01, 0, payload)
	if !bytes.Equal(got, want) {
		t.Errorf("WriteFrame mismatch:\n  got  %x\n  want %x", got, want)
	}
}

func TestPythonWrite_Xshard(t *testing.T) {
	want, _ := hex.DecodeString(xshardWire)
	payload := append([]byte{0x01, 0x00, 0x00, 0x00}, bytes.Repeat([]byte{0xff}, 32)...)
	payload = append(payload, bytes.Repeat([]byte{0x00}, 20)...)
	got := writeFrameNoMetaForTest(0x93, 42, payload)
	if !bytes.Equal(got, want) {
		t.Errorf("WriteFrameNoMeta mismatch:\n  got  %x\n  want %x", got, want)
	}
}

func TestPythonWrite_LargePayload(t *testing.T) {
	want, _ := hex.DecodeString(largePayloadWire)
	payload := bytes.Repeat([]byte{0xAB}, 10000)
	got := writeFrameForTest(Metadata{Branch: 3, ClusterPeerID: 0}, 0x10, 7, payload)
	if !bytes.Equal(got, want) {
		t.Errorf("WriteFrame mismatch (large):\n  got  %d bytes\n  want %d bytes", len(got), len(want))
	}
}

// =============================================================================
// Write+Read round-trip — defensive tests independent of the Python reference
// =============================================================================

func TestRoundTrip_Meta(t *testing.T) {
	cases := []struct {
		name string
		f    *Frame
	}{
		{"empty", &Frame{Opcode: 1, RPCID: 0, Payload: nil}},
		{"with_meta", &Frame{Meta: Metadata{Branch: 5, ClusterPeerID: 999}, Opcode: 0x10, RPCID: 7, Payload: []byte("hello")}},
		{"large_rpc_id", &Frame{Opcode: 0xC4, RPCID: 0xFFFFFFFFFFFFFFFF, Payload: []byte("x")}},
		{"zero_meta", &Frame{Meta: Metadata{}, Opcode: 0x81, RPCID: 1, Payload: []byte{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire := writeFrameForTest(tc.f.Meta, tc.f.Opcode, tc.f.RPCID, tc.f.Payload)
			got, err := ReadFrame(bytes.NewReader(wire))
			if err != nil {
				t.Fatalf("ReadFrame: %v", err)
			}
			if got.Opcode != tc.f.Opcode || got.RPCID != tc.f.RPCID || got.Meta != tc.f.Meta {
				t.Errorf("mismatch: got %+v, want %+v", got, tc.f)
			}
			if !bytes.Equal(got.Payload, tc.f.Payload) {
				t.Errorf("payload mismatch")
			}
		})
	}
}

func TestRoundTrip_NoMeta(t *testing.T) {
	original := &Frame{Opcode: 0x93, RPCID: 99, Payload: []byte("xshard-data")}
	wire := writeFrameNoMetaForTest(original.Opcode, original.RPCID, original.Payload)
	got, err := ReadFrameNoMeta(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrameNoMeta: %v", err)
	}
	if got.Opcode != original.Opcode || got.RPCID != original.RPCID {
		t.Errorf("mismatch: got %+v", got)
	}
	if !bytes.Equal(got.Payload, original.Payload) {
		t.Errorf("payload mismatch")
	}
}

// =============================================================================
// Wire-format layout — verify byte-level structure with hand-computed expected
// values, independent of any Python reference.
// =============================================================================

func TestWireFormatLayout(t *testing.T) {
	f := &Frame{
		Meta:    Metadata{Branch: 1, ClusterPeerID: 0x1122334455667788},
		Opcode:  0x42,
		RPCID:   0xDEADBEEFCAFEBABE,
		Payload: []byte{0xAA, 0xBB, 0xCC},
	}
	wire := writeFrameForTest(f.Meta, f.Opcode, f.RPCID, f.Payload)

	// payload_len = 3
	if got := binary.BigEndian.Uint32(wire[0:4]); got != 3 {
		t.Errorf("payload_len: got %d, want 3", got)
	}
	// branch = 1
	if got := binary.BigEndian.Uint32(wire[4:8]); got != 1 {
		t.Errorf("branch: got %d, want 1", got)
	}
	// cluster_peer_id = 0x1122334455667788
	if got := binary.BigEndian.Uint64(wire[8:16]); got != 0x1122334455667788 {
		t.Errorf("cluster_peer_id: got 0x%x", got)
	}
	// opcode = 0x42
	if wire[16] != 0x42 {
		t.Errorf("opcode: got 0x%02x, want 0x42", wire[16])
	}
	// rpc_id = 0xDEADBEEFCAFEBABE
	if got := binary.BigEndian.Uint64(wire[17:25]); got != 0xDEADBEEFCAFEBABE {
		t.Errorf("rpc_id: got 0x%x", got)
	}
	// payload
	if !bytes.Equal(wire[25:28], []byte{0xAA, 0xBB, 0xCC}) {
		t.Errorf("payload: got %x", wire[25:28])
	}
}

func TestMultiFrameStream(t *testing.T) {
	// 3 consecutive frames on a single stream, matching Python's back-to-back
	// write_raw_data() calls on the same TCP connection.
	frames := []*Frame{
		{Meta: Metadata{Branch: 0, ClusterPeerID: 0}, Opcode: 0x81, RPCID: 0, Payload: []byte("ping")},
		{Meta: Metadata{Branch: 2, ClusterPeerID: 999}, Opcode: 0x05, RPCID: 100, Payload: []byte("block_data")},
		{Meta: Metadata{Branch: 1, ClusterPeerID: 0}, Opcode: 0x03, RPCID: 200, Payload: nil},
	}
	var stream bytes.Buffer
	for _, f := range frames {
		stream.Write(writeFrameForTest(f.Meta, f.Opcode, f.RPCID, f.Payload))
	}

	reader := bytes.NewReader(stream.Bytes())
	for i, want := range frames {
		got, err := ReadFrame(reader)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if got.Opcode != want.Opcode || got.RPCID != want.RPCID || got.Meta != want.Meta {
			t.Errorf("frame %d mismatch: got %+v, want %+v", i, got, want)
		}
		if !bytes.Equal(got.Payload, want.Payload) {
			t.Errorf("frame %d payload mismatch", i)
		}
	}
}

// =============================================================================
// Error handling
// =============================================================================

func TestReadFrame_EOF(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader(nil)); err == nil {
		t.Error("expected error on empty stream")
	}
}

func TestReadFrame_Truncated(t *testing.T) {
	// payload_len says 100 bytes, but we only give 4 bytes
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 100)
	if _, err := ReadFrame(bytes.NewReader(hdr)); err == nil {
		t.Error("expected error on truncated frame")
	}
}

// =============================================================================
// Helpers
// =============================================================================

// writeFrameForTest is a thin wrapper that returns the wire bytes directly.
func writeFrameForTest(meta Metadata, opcode byte, rpcID uint64, payload []byte) []byte {
	var buf bytes.Buffer
	_ = WriteFrame(&buf, &Frame{Meta: meta, Opcode: opcode, RPCID: rpcID, Payload: payload})
	return buf.Bytes()
}

func writeFrameNoMetaForTest(opcode byte, rpcID uint64, payload []byte) []byte {
	var buf bytes.Buffer
	_ = WriteFrameNoMeta(&buf, &Frame{Opcode: opcode, RPCID: rpcID, Payload: payload})
	return buf.Bytes()
}
