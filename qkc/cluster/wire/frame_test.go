// Copyright 2026-2027, QuarkChain.

package wire

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"
)

var (
	// Test vectors serialized by pyquarkchain serializers using synthetic test values.
	// NOT real production network data.
	//
	// Payload content (Ping/Pong commands):
	//   - id = "id" (ASCII, 2 bytes)
	//   - full_shard_id_list = [1, 2]
	//   - root_tip = None (Ping only)
	//   - opcode = 0x81 (PING), 0x82 (PONG) from ClusterOp (CLUSTER_OP_BASE=128)
	pythonClusterPing = "0000001300000001000000000000303981000000000000000100000002696400000002000000010000000200"
	pythonNoMetaPing  = "0000001381000000000000000100000002696400000002000000010000000200"
)

// ---- wire compatibility (pyquarkchain vectors) ----

func TestWireCompat_Meta(t *testing.T) {
	wire := mustHex(t, pythonClusterPing)
	f, err := ReadFrame(bytes.NewReader(wire), 0)
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x81 || f.RPCID != 1 {
		t.Fatalf("unexpected header: opcode=0x%02x rpcid=%d", f.Opcode, f.RPCID)
	}
	if f.Meta.Branch != 1 || f.Meta.ClusterPeerID != 12345 {
		t.Fatalf("unexpected meta: %+v", f.Meta)
	}
	var out bytes.Buffer
	if err := WriteFrame(&out, f); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	if !bytes.Equal(out.Bytes(), wire) {
		t.Fatalf("frame mismatch:\n  go  %x\n  py  %x", out.Bytes(), wire)
	}
}

func TestWireCompat_NoMeta(t *testing.T) {
	wire := mustHex(t, pythonNoMetaPing)
	f, err := ReadFrameNoMeta(bytes.NewReader(wire), 0)
	if err != nil {
		t.Fatalf("ReadFrameNoMeta: %v", err)
	}
	if f.Opcode != 0x81 || f.RPCID != 1 {
		t.Fatalf("unexpected header: opcode=0x%02x rpcid=%d", f.Opcode, f.RPCID)
	}
	var out bytes.Buffer
	if err := WriteFrameNoMeta(&out, f); err != nil {
		t.Fatalf("WriteFrameNoMeta: %v", err)
	}
	if !bytes.Equal(out.Bytes(), wire) {
		t.Fatalf("frame mismatch:\n  go  %x\n  py  %x", out.Bytes(), wire)
	}
}

// ---- round-trip: write → read ----

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		meta    ClusterMetadata
		opcode  byte
		rpcID   uint64
		payload []byte
	}{
		{"nil_payload", ClusterMetadata{}, 0x01, 0, nil},
		{"with_meta", ClusterMetadata{Branch: 5, ClusterPeerID: 999}, 0x10, 7, []byte("hello")},
		{"max_rpc_id", ClusterMetadata{}, 0xC4, 0xFFFFFFFFFFFFFFFF, []byte("x")},
		{"empty_payload", ClusterMetadata{}, 0x81, 1, []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wire := writeFrameForTest(tc.meta, tc.opcode, tc.rpcID, tc.payload)
			got, err := ReadFrame(bytes.NewReader(wire), 0)
			if err != nil {
				t.Fatalf("ReadFrame: %v", err)
			}
			if got.Opcode != tc.opcode || got.RPCID != tc.rpcID || got.Meta != tc.meta {
				t.Errorf("mismatch: got %+v, want Meta=%+v Opcode=%x RPCID=%d", got, tc.meta, tc.opcode, tc.rpcID)
			}
			if !bytes.Equal(got.Payload, tc.payload) {
				t.Errorf("payload mismatch")
			}
		})
	}
}

// ---- wire format layout ----

func TestWireFormatLayout(t *testing.T) {
	f := &Frame{
		Meta:    ClusterMetadata{Branch: 1, ClusterPeerID: 0x1122334455667788},
		Opcode:  0x42,
		RPCID:   0xDEADBEEFCAFEBABE,
		Payload: []byte{0xAA, 0xBB, 0xCC},
	}
	wire := writeFrameForTest(f.Meta, f.Opcode, f.RPCID, f.Payload)

	if got := binary.BigEndian.Uint32(wire[0:4]); got != 3 {
		t.Errorf("payload_len: got %d, want 3", got)
	}
	if got := binary.BigEndian.Uint32(wire[4:8]); got != 1 {
		t.Errorf("branch: got %d, want 1", got)
	}
	if got := binary.BigEndian.Uint64(wire[8:16]); got != 0x1122334455667788 {
		t.Errorf("cluster_peer_id: got 0x%x", got)
	}
	if wire[16] != 0x42 {
		t.Errorf("opcode: got 0x%02x, want 0x42", wire[16])
	}
	if got := binary.BigEndian.Uint64(wire[17:25]); got != 0xDEADBEEFCAFEBABE {
		t.Errorf("rpc_id: got 0x%x", got)
	}
	if !bytes.Equal(wire[25:28], []byte{0xAA, 0xBB, 0xCC}) {
		t.Errorf("payload: got %x", wire[25:28])
	}
}

// ---- multi-frame stream ----

func TestMultiFrameStream(t *testing.T) {
	frames := []*Frame{
		{Meta: ClusterMetadata{Branch: 0, ClusterPeerID: 0}, Opcode: 0x81, RPCID: 0, Payload: []byte("ping")},
		{Meta: ClusterMetadata{Branch: 2, ClusterPeerID: 999}, Opcode: 0x05, RPCID: 100, Payload: []byte("block_data")},
		{Meta: ClusterMetadata{Branch: 1, ClusterPeerID: 0}, Opcode: 0x03, RPCID: 200, Payload: nil},
	}
	var stream bytes.Buffer
	for _, f := range frames {
		stream.Write(writeFrameForTest(f.Meta, f.Opcode, f.RPCID, f.Payload))
	}

	reader := bytes.NewReader(stream.Bytes())
	for i, want := range frames {
		got, err := ReadFrame(reader, 0)
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

// ---- error paths ----

func TestReadErrors(t *testing.T) {
	cases := []struct {
		name string
		r    io.Reader
		read func(io.Reader, uint32) (*Frame, error)
	}{
		{"meta_empty_stream", bytes.NewReader(nil), ReadFrame},
		{"meta_truncated", bytes.NewReader(truncatedHeader()), ReadFrame},
		{"nometa_empty_stream", bytes.NewReader(nil), ReadFrameNoMeta},
		{"nometa_truncated", bytes.NewReader(truncatedHeader()), ReadFrameNoMeta},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.read(tc.r, 0); err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func truncatedHeader() []byte {
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 100)
	return hdr
}

// ---- payload size limit ----

func TestReadFrame_PayloadLimit(t *testing.T) {
	payload := bytes.Repeat([]byte{0xAB}, 100)
	frame := &Frame{Meta: ClusterMetadata{Branch: 1, ClusterPeerID: 2}, Opcode: 0x81, RPCID: 1, Payload: payload}

	// limit == 0: unbounded — full frame with 100-byte payload is accepted.
	{
		wire := writeFrameForTest(frame.Meta, frame.Opcode, frame.RPCID, frame.Payload)
		got, err := ReadFrame(bytes.NewReader(wire), 0)
		if err != nil {
			t.Fatalf("limit=0 should accept full frame, got: %v", err)
		}
		if !bytes.Equal(got.Payload, payload) {
			t.Fatalf("limit=0 payload mismatch")
		}
	}

	// limit == payload length: boundary, accepted.
	{
		wire := writeFrameForTest(frame.Meta, frame.Opcode, frame.RPCID, frame.Payload)
		if _, err := ReadFrame(bytes.NewReader(wire), uint32(len(payload))); err != nil {
			t.Fatalf("limit==payload_len should accept, got: %v", err)
		}
	}

	// limit < payload length: rejected before metadata is read.
	{
		wire := writeFrameForTest(frame.Meta, frame.Opcode, frame.RPCID, frame.Payload)
		_, err := ReadFrame(bytes.NewReader(wire), uint32(len(payload)-1))
		if err == nil {
			t.Fatal("limit < payload_len should be rejected")
		}
	}
}

func TestReadFrameNoMeta_PayloadLimit(t *testing.T) {
	payload := bytes.Repeat([]byte{0xCD}, 32)
	frame := &Frame{Opcode: 0x82, RPCID: 7, Payload: payload}

	// limit == 0: unbounded.
	{
		wire := writeFrameNoMetaForTest(frame.Opcode, frame.RPCID, frame.Payload)
		got, err := ReadFrameNoMeta(bytes.NewReader(wire), 0)
		if err != nil {
			t.Fatalf("limit=0 should accept full frame, got: %v", err)
		}
		if !bytes.Equal(got.Payload, payload) {
			t.Fatalf("limit=0 payload mismatch")
		}
	}

	// limit == payload length: boundary, accepted.
	{
		wire := writeFrameNoMetaForTest(frame.Opcode, frame.RPCID, frame.Payload)
		if _, err := ReadFrameNoMeta(bytes.NewReader(wire), uint32(len(payload))); err != nil {
			t.Fatalf("limit==payload_len should accept, got: %v", err)
		}
	}

	// limit < payload length: rejected.
	{
		wire := writeFrameNoMetaForTest(frame.Opcode, frame.RPCID, frame.Payload)
		_, err := ReadFrameNoMeta(bytes.NewReader(wire), uint32(len(payload)-1))
		if err == nil {
			t.Fatal("limit < payload_len should be rejected")
		}
	}
}

// ---- ClusterMetadata serialization ----

func TestClusterMetadata(t *testing.T) {
	// Round-trip: edge cases.
	cases := []ClusterMetadata{
		{Branch: 0, ClusterPeerID: 0},
		{Branch: 1, ClusterPeerID: 12345},
		{Branch: 0xFFFFFFFF, ClusterPeerID: 0xFFFFFFFFFFFFFFFF},
	}
	for _, m := range cases {
		wire := MarshalClusterMetadata(m)
		got, err := UnmarshalClusterMetadata(wire)
		if err != nil {
			t.Fatalf("UnmarshalClusterMetadata(%+v): %v", m, err)
		}
		if got != m {
			t.Errorf("round-trip mismatch: got %+v, want %+v", got, m)
		}
	}

	// Invalid lengths.
	for _, n := range []int{0, 4, 8, 11, 13, 16} {
		b := make([]byte, n)
		if _, err := UnmarshalClusterMetadata(b); err == nil {
			t.Errorf("expected error for %d-byte input", n)
		}
	}
}

// ---- helpers ----

func mustHex(t *testing.T, hexStr string) []byte {
	t.Helper()
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	return b
}

func writeFrameForTest(meta ClusterMetadata, opcode byte, rpcID uint64, payload []byte) []byte {
	var buf bytes.Buffer
	_ = WriteFrame(&buf, &Frame{Meta: meta, Opcode: opcode, RPCID: rpcID, Payload: payload})
	return buf.Bytes()
}

func writeFrameNoMetaForTest(opcode byte, rpcID uint64, payload []byte) []byte {
	var buf bytes.Buffer
	_ = WriteFrameNoMeta(&buf, &Frame{Opcode: opcode, RPCID: rpcID, Payload: payload})
	return buf.Bytes()
}
