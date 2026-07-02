// Copyright 2026-2027, QuarkChain.

package wire

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
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
	pythonClusterPong = "00000012000000010000000000003039820000000000000001000000026964000000020000000100000002"
	pythonNoMetaPing  = "0000001381000000000000000100000002696400000002000000010000000200"
	pythonNoMetaPong  = "00000012820000000000000001000000026964000000020000000100000002"
)

func TestPythonVectors_ClusterPing(t *testing.T) {
	wire := mustPythonVectorBytes(t, pythonClusterPing)
	f, err := ReadFrame(bytes.NewReader(wire))
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

func TestPythonVectors_ClusterPong(t *testing.T) {
	wire := mustPythonVectorBytes(t, pythonClusterPong)
	f, err := ReadFrame(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if f.Opcode != 0x82 || f.RPCID != 1 {
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

func TestPythonVectors_NoMetaPing(t *testing.T) {
	wire := mustPythonVectorBytes(t, pythonNoMetaPing)
	f, err := ReadFrameNoMeta(bytes.NewReader(wire))
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

func TestPythonVectors_NoMetaPong(t *testing.T) {
	wire := mustPythonVectorBytes(t, pythonNoMetaPong)
	f, err := ReadFrameNoMeta(bytes.NewReader(wire))
	if err != nil {
		t.Fatalf("ReadFrameNoMeta: %v", err)
	}
	if f.Opcode != 0x82 || f.RPCID != 1 {
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

func mustPythonVectorBytes(t *testing.T, hexStr string) []byte {
	t.Helper()
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("decode python hex: %v", err)
	}
	return b
}

func TestRoundTrip_Meta(t *testing.T) {
	cases := []struct {
		name string
		f    *Frame
	}{
		{"empty", &Frame{Opcode: 1, RPCID: 0, Payload: nil}},
		{"with_meta", &Frame{Meta: ClusterMetadata{Branch: 5, ClusterPeerID: 999}, Opcode: 0x10, RPCID: 7, Payload: []byte("hello")}},
		{"large_rpc_id", &Frame{Opcode: 0xC4, RPCID: 0xFFFFFFFFFFFFFFFF, Payload: []byte("x")}},
		{"zero_meta", &Frame{Meta: ClusterMetadata{}, Opcode: 0x81, RPCID: 1, Payload: []byte{}}},
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

func TestReadFrame_EOF(t *testing.T) {
	if _, err := ReadFrame(bytes.NewReader(nil)); err == nil {
		t.Error("expected error on empty stream")
	}
}

func TestReadFrame_Truncated(t *testing.T) {
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 100)
	if _, err := ReadFrame(bytes.NewReader(hdr)); err == nil {
		t.Error("expected error on truncated frame")
	}
}

func TestReadFrame_PayloadLimit(t *testing.T) {
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 100)

	if _, err := ReadFrameWithMaxPayload(bytes.NewReader(hdr), 64); err == nil {
		t.Fatal("expected error when payload_len exceeds limit")
	}

	if _, err := ReadFrameWithMaxPayload(bytes.NewReader(hdr), 0); err == nil {
		t.Fatal("expected error when maxPayloadLen is zero")
	}

	if _, err := ReadFrame(bytes.NewReader(hdr)); err == nil {
		t.Error("expected error on truncated frame without payload limit")
	}
}

func TestReadFrameNoMeta_PayloadLimit(t *testing.T) {
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, 32)

	if _, err := ReadFrameNoMetaWithMaxPayload(bytes.NewReader(hdr), 16); err == nil {
		t.Fatal("expected error when payload_len exceeds limit")
	}

	if _, err := ReadFrameNoMetaWithMaxPayload(bytes.NewReader(hdr), 0); err == nil {
		t.Fatal("expected error when maxPayloadLen is zero")
	}
}

func TestClusterMetadata_RoundTrip(t *testing.T) {
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
}

func TestUnmarshalClusterMetadata_InvalidLength(t *testing.T) {
	for _, n := range []int{0, 4, 8, 11, 13, 16} {
		b := make([]byte, n)
		if _, err := UnmarshalClusterMetadata(b); err == nil {
			t.Errorf("expected error for %d-byte input", n)
		}
	}
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
