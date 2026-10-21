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
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	frame := &ChainFrame{Frame: Frame{Opcode: 0x42, RPCID: 0x0102030405060708, Payload: []byte("chain")}}
	var buf bytes.Buffer
	if err := WriteChainFrame(&buf, frame); err != nil {
		t.Fatal(err)
	}
	encoded := buf.Bytes()
	if got := binary.BigEndian.Uint32(encoded[:4]); got != uint32(len(frame.Payload)) {
		t.Fatalf("payload length = %d, want %d", got, len(frame.Payload))
	}
	got, err := ReadChainFrame(&buf, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestMasterFrameRoundTrip(t *testing.T) {
	frame := &MasterFrame{ClusterPeerID: 0x1122334455667788, Frame: Frame{Opcode: 0x81, RPCID: 9, Payload: []byte("master-chain")}}
	var buf bytes.Buffer
	if err := WriteMasterFrame(&buf, frame); err != nil {
		t.Fatal(err)
	}
	encoded := buf.Bytes()
	if got := binary.BigEndian.Uint64(encoded[4:12]); got != frame.ClusterPeerID {
		t.Fatalf("cluster peer ID = 0x%x, want 0x%x", got, frame.ClusterPeerID)
	}
	got, err := ReadMasterFrame(&buf, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClusterPeerID != frame.ClusterPeerID || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestPeerFrameRoundTrip(t *testing.T) {
	frame := &PeerFrame{ChainID: 0x01020304, Frame: Frame{Opcode: 0x02, RPCID: 7, Payload: []byte("p2p")}}
	var encoded bytes.Buffer
	if err := WritePeerFrame(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	if encoded.Len() != 4+frameBodyHeader+len(frame.Payload) {
		t.Fatalf("encoded length = %d", encoded.Len())
	}
	got, err := ReadPeerFrame(&encoded, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChainID != frame.ChainID || got.Opcode != frame.Opcode || got.RPCID != frame.RPCID || !bytes.Equal(got.Payload, frame.Payload) {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, frame)
	}
}

func TestFrameWireLayouts(t *testing.T) {
	body := Frame{Opcode: 0x10, RPCID: 0x0102030405060708, Payload: []byte{0xaa}}

	var chain bytes.Buffer
	if err := WriteChainFrame(&chain, &ChainFrame{Frame: body}); err != nil {
		t.Fatal(err)
	}
	wantChain := []byte{0, 0, 0, 1, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(chain.Bytes(), wantChain) {
		t.Fatalf("ChainFrame bytes = %x, want %x", chain.Bytes(), wantChain)
	}

	var master bytes.Buffer
	if err := WriteMasterFrame(&master, &MasterFrame{ClusterPeerID: 0x1122334455667788, Frame: body}); err != nil {
		t.Fatal(err)
	}
	wantMaster := []byte{0, 0, 0, 1, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(master.Bytes(), wantMaster) {
		t.Fatalf("MasterFrame bytes = %x, want %x", master.Bytes(), wantMaster)
	}

	var peer bytes.Buffer
	if err := WritePeerFrame(&peer, &PeerFrame{ChainID: 0xaabbccdd, Frame: body}); err != nil {
		t.Fatal(err)
	}
	wantPeer := []byte{0xaa, 0xbb, 0xcc, 0xdd, 0x10, 1, 2, 3, 4, 5, 6, 7, 8, 0xaa}
	if !bytes.Equal(peer.Bytes(), wantPeer) {
		t.Fatalf("PeerFrame bytes = %x, want %x", peer.Bytes(), wantPeer)
	}
}

func TestMultiFrameChainStream(t *testing.T) {
	frames := []*ChainFrame{
		{Frame: Frame{Opcode: 1, RPCID: 1, Payload: []byte("one")}},
		{Frame: Frame{Opcode: 2, RPCID: 2, Payload: []byte("two")}},
	}
	var stream bytes.Buffer
	for _, frame := range frames {
		if err := WriteChainFrame(&stream, frame); err != nil {
			t.Fatal(err)
		}
	}
	for i, want := range frames {
		got, err := ReadChainFrame(&stream, 0)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if got.Opcode != want.Opcode || got.RPCID != want.RPCID || !bytes.Equal(got.Payload, want.Payload) {
			t.Fatalf("frame %d = %+v, want %+v", i, got, want)
		}
	}
}

func TestFrameReadErrors(t *testing.T) {
	if _, err := ReadChainFrame(bytes.NewReader(nil), 0); !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream error = %v, want EOF", err)
	}
	if _, err := ReadMasterFrame(bytes.NewReader([]byte{0, 0}), 0); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated length error = %v, want unexpected EOF", err)
	}
	var buf bytes.Buffer
	_ = WriteChainFrame(&buf, &ChainFrame{Frame: Frame{Payload: []byte("payload")}})
	if _, err := ReadChainFrame(bytes.NewReader(buf.Bytes()), 3); err == nil {
		t.Fatal("oversized payload accepted")
	}
	if _, err := ReadPeerFrame(bytes.NewReader([]byte{1, 2, 3}), 0); err == nil {
		t.Fatal("short P2P frame accepted")
	}
}

func TestFrameEOFContract(t *testing.T) {
	if _, err := ReadChainFrame(bytes.NewReader(nil), 0); !errors.Is(err, io.EOF) {
		t.Fatalf("clean EOF = %v, want EOF", err)
	}
	if _, err := ReadChainFrame(bytes.NewReader([]byte{0, 0}), 0); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial length = %v, want unexpected EOF", err)
	}
	if _, err := ReadMasterFrame(bytes.NewReader([]byte{0, 0, 0, 0}), 0); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("missing cluster peer ID = %v, want unexpected EOF", err)
	}
	if _, err := ReadChainFrame(bytes.NewReader([]byte{0, 0, 0, 0}), 0); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("missing frame body = %v, want unexpected EOF", err)
	}
}

func TestMasterAndPeerFramePayloadLimits(t *testing.T) {
	master := &MasterFrame{Frame: Frame{Payload: []byte("payload")}}
	var masterBytes bytes.Buffer
	if err := WriteMasterFrame(&masterBytes, master); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadMasterFrame(bytes.NewReader(masterBytes.Bytes()), 3); err == nil {
		t.Fatal("MasterFrame accepted payload above limit")
	}

	peer := &PeerFrame{Frame: Frame{Payload: []byte("payload")}}
	var peerBytes bytes.Buffer
	if err := WritePeerFrame(&peerBytes, peer); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPeerFrame(bytes.NewReader(peerBytes.Bytes()), 3); err == nil {
		t.Fatal("PeerFrame accepted payload above limit")
	}
}
