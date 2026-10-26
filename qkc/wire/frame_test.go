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

package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"
	"testing"
)

const testMaxPayloadLen = 1024

func TestClusterFrameRoundTrip(t *testing.T) {
	frame := &ClusterFrame{Opcode: 0x42, RPCID: 0x0102030405060708, Payload: []byte("chain")}
	var buf bytes.Buffer
	if err := WriteClusterFrame(&buf, frame); err != nil {
		t.Fatal(err)
	}
	encoded := buf.Bytes()
	if got := binary.BigEndian.Uint32(encoded[:4]); got != uint32(len(frame.Payload)) {
		t.Fatalf("payload length = %d, want %d", got, len(frame.Payload))
	}
	got, err := ReadClusterFrame(&buf, testMaxPayloadLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestClusterFrameWithPeerIDRoundTrip(t *testing.T) {
	frame := &ClusterFrame{
		ClusterPeerID: 0x1122334455667788,
		Opcode:        0x81,
		RPCID:         9,
		Payload:       []byte("master-chain"),
	}
	var buf bytes.Buffer
	if err := WriteClusterFrameWithPeerID(&buf, frame); err != nil {
		t.Fatal(err)
	}
	encoded := buf.Bytes()
	if got := binary.BigEndian.Uint64(encoded[4:12]); got != frame.ClusterPeerID {
		t.Fatalf("cluster peer ID = 0x%x, want 0x%x", got, frame.ClusterPeerID)
	}
	got, err := ReadClusterFrameWithPeerID(&buf, testMaxPayloadLen)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClusterPeerID != frame.ClusterPeerID || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestP2PFrameRoundTrip(t *testing.T) {
	frame := &P2PFrame{ChainID: 0x01020304, Opcode: 0x02, RPCID: 7, Payload: []byte("p2p")}
	encoded, err := EncodeP2PFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) != chainIDSize+frameBodyHeader+len(frame.Payload) {
		t.Fatalf("encoded length = %d", len(encoded))
	}
	got, err := DecodeP2PFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChainID != frame.ChainID || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestP2PFramePayloadAliasesInput(t *testing.T) {
	var encoded bytes.Buffer
	if err := WriteP2PFrame(&encoded, &P2PFrame{Payload: []byte("abc")}); err != nil {
		t.Fatal(err)
	}
	data := encoded.Bytes()
	frame, err := DecodeP2PFrame(data)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-1] = 'z'
	if !bytes.Equal(frame.Payload, []byte("abz")) {
		t.Fatalf("payload = %q, want input buffer changes to be visible", frame.Payload)
	}
}

func TestRootChainID(t *testing.T) {
	if RootChainID != math.MaxUint32 {
		t.Fatalf("root chain ID = %d, want %d", RootChainID, uint32(math.MaxUint32))
	}
	var encoded bytes.Buffer
	if err := WriteP2PFrame(&encoded, &P2PFrame{ChainID: RootChainID}); err != nil {
		t.Fatal(err)
	}
	frame, err := DecodeP2PFrame(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if frame.ChainID != RootChainID {
		t.Fatalf("decoded chain ID = %d, want %d", frame.ChainID, RootChainID)
	}
}

func TestFrameWireLayouts(t *testing.T) {
	const peerID = 0x1122334455667788
	frame := &ClusterFrame{Opcode: 0x10, RPCID: 0x0102030405060708, Payload: []byte{0xaa}}

	var chain bytes.Buffer
	if err := WriteClusterFrame(&chain, frame); err != nil {
		t.Fatal(err)
	}
	wantChain := []byte{0, 0, 0, 1, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(chain.Bytes(), wantChain) {
		t.Fatalf("ClusterFrame bytes = %x, want %x", chain.Bytes(), wantChain)
	}

	frame.ClusterPeerID = peerID
	var master bytes.Buffer
	if err := WriteClusterFrameWithPeerID(&master, frame); err != nil {
		t.Fatal(err)
	}
	wantMaster := []byte{0, 0, 0, 1, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(master.Bytes(), wantMaster) {
		t.Fatalf("ClusterFrame with peer ID bytes = %x, want %x", master.Bytes(), wantMaster)
	}

	var p2p bytes.Buffer
	if err := WriteP2PFrame(&p2p, &P2PFrame{ChainID: 0xaabbccdd, Opcode: frame.Opcode, RPCID: frame.RPCID, Payload: frame.Payload}); err != nil {
		t.Fatal(err)
	}
	wantP2P := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(p2p.Bytes(), wantP2P) {
		t.Fatalf("P2PFrame bytes = %x, want %x", p2p.Bytes(), wantP2P)
	}
}

func TestMultiFrameClusterStream(t *testing.T) {
	frames := []*ClusterFrame{
		{Opcode: 1, RPCID: 1, Payload: []byte("one")},
		{Opcode: 2, RPCID: 2, Payload: []byte("two")},
	}
	var stream bytes.Buffer
	for _, frame := range frames {
		if err := WriteClusterFrame(&stream, frame); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range frames {
		got, err := ReadClusterFrame(&stream, testMaxPayloadLen)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if got.Opcode != want.Opcode || got.RPCID != want.RPCID || !bytes.Equal(got.Payload, want.Payload) {
			t.Fatalf("frame %d = %+v, want %+v", i, got, want)
		}
	}
}

func TestFrameReadErrors(t *testing.T) {
	if _, err := ReadClusterFrame(bytes.NewReader(nil), testMaxPayloadLen); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream error = %v, want EOF", err)
	}
	if _, err := ReadClusterFrameWithPeerID(bytes.NewReader([]byte{0, 0}), testMaxPayloadLen); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated length error = %v, want unexpected EOF", err)
	}
	var buf bytes.Buffer
	if err := WriteClusterFrame(&buf, &ClusterFrame{Payload: []byte("payload")}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClusterFrame(bytes.NewReader(buf.Bytes()), 3); err == nil {
		t.Fatal("oversized payload accepted")
	}
	if _, err := DecodeP2PFrame([]byte{1, 2, 3}); err == nil {
		t.Fatal("short P2P frame accepted")
	}
	if err := WriteClusterFrame(&buf, &ClusterFrame{ClusterPeerID: 1}); err == nil {
		t.Fatal("direct chain codec accepted a cluster peer ID")
	}
	if _, err := EncodeP2PFrame(nil); err == nil {
		t.Fatal("EncodeP2PFrame accepted a nil frame")
	}
}

func TestFrameEOFContract(t *testing.T) {
	if _, err := ReadClusterFrame(bytes.NewReader(nil), testMaxPayloadLen); !errors.Is(err, io.EOF) {
		t.Fatalf("clean EOF = %v, want EOF", err)
	}
	if _, err := ReadClusterFrame(bytes.NewReader([]byte{0, 0}), testMaxPayloadLen); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial length = %v, want unexpected EOF", err)
	}
	if _, err := ReadClusterFrameWithPeerID(bytes.NewReader([]byte{0, 0, 0, 0}), testMaxPayloadLen); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("missing cluster peer ID = %v, want unexpected EOF", err)
	}
	if _, err := ReadClusterFrame(bytes.NewReader([]byte{0, 0, 0, 0}), testMaxPayloadLen); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("missing frame body = %v, want unexpected EOF", err)
	}
	var truncated bytes.Buffer
	if err := WriteClusterFrame(&truncated, &ClusterFrame{Payload: []byte("payload")}); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClusterFrame(bytes.NewReader(truncated.Bytes()[:truncated.Len()-1]), testMaxPayloadLen); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated payload = %v, want unexpected EOF", err)
	}
}

func TestClusterFrameFinalReadError(t *testing.T) {
	readErr := errors.New("read failed")
	codecs := []struct {
		name  string
		read  func(io.Reader, uint32) (*ClusterFrame, error)
		write func(io.Writer, *ClusterFrame) error
	}{
		{"direct", ReadClusterFrame, WriteClusterFrame},
		{"peer", ReadClusterFrameWithPeerID, WriteClusterFrameWithPeerID},
	}
	for _, codec := range codecs {
		t.Run(codec.name, func(t *testing.T) {
			frame := &ClusterFrame{Opcode: 0x42, RPCID: 7, Payload: []byte("payload")}
			var encoded bytes.Buffer
			if err := codec.write(&encoded, frame); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name    string
				cut     int
				readErr error
				wantErr error
			}{
				{"complete with error", encoded.Len(), readErr, nil},
				{"complete with EOF", encoded.Len(), io.EOF, nil},
				{"truncated with error", encoded.Len() - 1, readErr, readErr},
				{"truncated with EOF", encoded.Len() - 1, io.EOF, io.ErrUnexpectedEOF},
			} {
				t.Run(tc.name, func(t *testing.T) {
					r := &finalErrorReader{data: encoded.Bytes()[:tc.cut], err: tc.readErr}
					got, err := codec.read(r, uint32(len(frame.Payload)))
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("read error = %v, want %v", err, tc.wantErr)
					}
					if tc.wantErr == nil && (got == nil || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload)) {
						t.Fatalf("complete frame = %+v, want %+v", got, frame)
					}
				})
			}
		})
	}
}

type finalErrorReader struct {
	data []byte
	err  error
}

func (r *finalErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

func TestFramePayloadLimits(t *testing.T) {
	if _, err := ReadClusterFrame(bytes.NewReader(nil), 0); err == nil || !strings.Contains(err.Error(), "maxPayloadLen") {
		t.Fatalf("zero direct-frame limit error = %v, want invalid limit", err)
	}
	if _, err := ReadClusterFrameWithPeerID(bytes.NewReader(nil), 0); err == nil || !strings.Contains(err.Error(), "maxPayloadLen") {
		t.Fatalf("zero peer-frame limit error = %v, want invalid limit", err)
	}

	cluster := &ClusterFrame{Payload: []byte("payload")}
	var clusterBytes bytes.Buffer
	if err := WriteClusterFrameWithPeerID(&clusterBytes, cluster); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClusterFrameWithPeerID(bytes.NewReader(clusterBytes.Bytes()), 3); err == nil {
		t.Fatal("ClusterFrame accepted payload above limit")
	}
	oversizedHeader := make([]byte, frameHeader)
	binary.BigEndian.PutUint32(oversizedHeader, MaxFramePayloadLen+1)
	if _, err := ReadClusterFrame(bytes.NewReader(oversizedHeader), MaxFramePayloadLen); err == nil || !strings.Contains(err.Error(), "payload too large") {
		t.Fatalf("oversized ClusterFrame error = %v, want payload too large", err)
	}
	emptyFrame := make([]byte, frameHeader+frameBodyHeader)
	r := bytes.NewReader(emptyFrame)
	if _, err := ReadClusterFrame(r, MaxFramePayloadLen+1); err == nil || !strings.Contains(err.Error(), "maxPayloadLen must be between") {
		t.Fatalf("unsupported maxPayloadLen error = %v, want invalid configuration", err)
	}
	if r.Len() != len(emptyFrame) {
		t.Fatal("read frame data before rejecting unsupported maxPayloadLen")
	}
	if _, err := ReadClusterFrame(r, MaxFramePayloadLen); err != nil {
		t.Fatalf("maximum supported maxPayloadLen error = %v", err)
	}
}

func TestClusterFrameMaxPayload(t *testing.T) {
	const payloadLimit = 16 << 20
	if MaxFramePayloadLen != payloadLimit {
		t.Fatalf("maximum payload = %d, want %d", MaxFramePayloadLen, payloadLimit)
	}
	payload := bytes.Repeat([]byte{0xa5}, payloadLimit+1)
	for _, codec := range []struct {
		name   string
		peerID uint64
		read   func(io.Reader, uint32) (*ClusterFrame, error)
		write  func(io.Writer, *ClusterFrame) error
	}{
		{"direct", 0, ReadClusterFrame, WriteClusterFrame},
		{"peer", 0x1122334455667788, ReadClusterFrameWithPeerID, WriteClusterFrameWithPeerID},
	} {
		t.Run(codec.name, func(t *testing.T) {
			frame := &ClusterFrame{ClusterPeerID: codec.peerID, Opcode: 0x42, RPCID: 7, Payload: payload[:payloadLimit]}
			var encoded bytes.Buffer
			if err := codec.write(&encoded, frame); err != nil {
				t.Fatal(err)
			}
			got, err := codec.read(&encoded, payloadLimit)
			if err != nil {
				t.Fatal(err)
			}
			if got.ClusterPeerID != frame.ClusterPeerID || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
				t.Fatal("maximum-size frame round trip mismatch")
			}
			if encoded.Len() != 0 {
				t.Fatalf("unread frame bytes = %d", encoded.Len())
			}
			frame.Payload = payload
			if err := codec.write(&encoded, frame); err == nil {
				t.Fatal("writer accepted payload above 16 MiB")
			}
			if encoded.Len() != 0 {
				t.Fatal("writer emitted frame data before rejecting oversized payload")
			}
		})
	}
}

func TestClusterFrameForgedLength(t *testing.T) {
	for _, codec := range []struct {
		name string
		read func(io.Reader, uint32) (*ClusterFrame, error)
	}{
		{"direct", ReadClusterFrame},
		{"peer", ReadClusterFrameWithPeerID},
	} {
		t.Run(codec.name, func(t *testing.T) {
			for _, length := range []uint32{(16 << 20) + 1, math.MaxUint32} {
				// Leave bytes after the header to detect reads of peer ID or body.
				var data [frameHeader + clusterPeerIDSize + frameBodyHeader]byte
				binary.BigEndian.PutUint32(data[:frameHeader], length)
				r := bytes.NewReader(data[:])
				if _, err := codec.read(r, MaxFramePayloadLen); err == nil || !strings.Contains(err.Error(), "payload too large") {
					t.Fatalf("forged length %d error = %v, want payload too large", length, err)
				}
				if r.Len() != len(data)-frameHeader {
					t.Fatalf("read beyond length header for forged length %d", length)
				}
			}
		})
	}
}
