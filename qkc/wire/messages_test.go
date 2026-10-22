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
	"reflect"
	"testing"
)

// ---- opcode → message registry ----

func TestNewClusterMessageRegistry(t *testing.T) {
	cases := []struct {
		op   ClusterOp
		want interface{}
	}{
		{ClusterOpPing, &PingRequest{}},
		{ClusterOpPong, &PongResponse{}},
		{ClusterOpConnectToChainsRequest, &ConnectToChainsRequest{}},
		{ClusterOpConnectToChainsResponse, &ConnectToChainsResponse{}},
	}
	for _, c := range cases {
		got, err := NewClusterMessage(c.op)
		if err != nil {
			t.Fatalf("NewClusterMessage(0x%02x): %v", byte(c.op), err)
		}
		if reflect.TypeOf(got) != reflect.TypeOf(c.want) {
			t.Fatalf("NewClusterMessage(0x%02x) = %T, want %T", byte(c.op), got, c.want)
		}
	}

	if _, err := NewClusterMessage(ClusterOp(0x99)); err == nil {
		t.Fatal("NewClusterMessage: unknown opcode must be rejected")
	}
}

func TestNewCommandMessageRegistry(t *testing.T) {
	msg, err := NewCommandMessage(CommandOpHello)
	if err != nil {
		t.Fatalf("NewCommandMessage(Hello): %v", err)
	}
	if _, ok := msg.(*HelloCommand); !ok {
		t.Fatalf("NewCommandMessage(Hello) = %T, want *HelloCommand", msg)
	}
}

// ---- SSZ round trips ----

// roundTripSSZ marshals msg, unmarshals into a fresh value of the same type
// via the registry-independent path, and re-marshals: canonical encoding
// must reproduce the input bytes exactly. This avoids false mismatches from
// nil vs empty slices on dynamic fields.
func roundTripSSZ(t *testing.T, msg SSZMessage) {
	t.Helper()
	enc, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	if size := msg.SizeSSZ(); size != len(enc) {
		t.Fatalf("SizeSSZ() = %d, but MarshalSSZ produced %d bytes", size, len(enc))
	}

	fresh := reflect.New(reflect.TypeOf(msg).Elem()).Interface().(SSZMessage)
	if err := fresh.UnmarshalSSZ(enc); err != nil {
		t.Fatalf("UnmarshalSSZ: %v", err)
	}
	dec, err := fresh.MarshalSSZ()
	if err != nil {
		t.Fatalf("re-MarshalSSZ: %v", err)
	}
	if !bytes.Equal(enc, dec) {
		t.Fatalf("round trip mismatch:\n  got  %x\n  want %x", dec, enc)
	}
}

func TestPingPongSSZRoundTrip(t *testing.T) {
	ping := &PingRequest{
		ID:         []byte{0xde, 0xad, 0xbe, 0xef},
		ChainID:    0x10002,
		HasRootTip: true,
		RootTip:    []byte{0xaa, 0xbb},
	}
	roundTripSSZ(t, ping)

	// Explicit deep-equality on the non-empty case.
	var got PingRequest
	if err := got.UnmarshalSSZ(mustMarshalSSZ(t, ping)); err != nil {
		t.Fatalf("UnmarshalSSZ: %v", err)
	}
	if !reflect.DeepEqual(&got, ping) {
		t.Fatalf("deep mismatch: got %+v, want %+v", &got, ping)
	}

	pong := &PongResponse{ID: []byte("id"), ChainID: 2}
	roundTripSSZ(t, pong)

	// Boundary: all dynamic fields empty.
	roundTripSSZ(t, &PingRequest{})
	roundTripSSZ(t, &PongResponse{})
}

func TestConnectToChainsSSZRoundTrip(t *testing.T) {
	req := &ConnectToChainsRequest{
		ChainInfoList: []ChainInfo{
			{ID: []byte("s1"), Host: []byte("10.0.0.1"), Port: 38391, ChainID: 1},
			{ID: []byte("s2"), Host: []byte("10.0.0.2"), Port: 38392, ChainID: 2},
		},
	}
	roundTripSSZ(t, req)

	// Results mirror the request order; empty bytes mean success.
	resp := &ConnectToChainsResponse{
		ResultList: [][]byte{
			nil,                    // success
			[]byte("dial timeout"), // failure
		},
	}
	roundTripSSZ(t, resp)

	// Boundary: empty lists.
	roundTripSSZ(t, &ConnectToChainsRequest{})
	roundTripSSZ(t, &ConnectToChainsResponse{})
}

func TestHelloCommandSSZRoundTrip(t *testing.T) {
	msg := &HelloCommand{
		Version:              1,
		NetworkID:            2,
		PeerID:               [HashLength]byte{1, 2, 3},
		PeerIP:               [UInt128Length]byte{127, 0, 0, 1},
		PeerPort:             38291,
		ChainMaskList:        []uint32{1, 0x10001},
		RootBlockHeader:      []byte{0xaa, 0xbb},
		GenesisRootBlockHash: [HashLength]byte{4, 5, 6},
	}
	roundTripSSZ(t, msg)
}

func TestSSZRejectsInvalidOffset(t *testing.T) {
	// PongResponse has an 8-byte fixed section. Offsets cannot point into it.
	bad := []byte{0x07, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00}
	if err := new(PongResponse).UnmarshalSSZ(bad); err == nil {
		t.Fatal("UnmarshalSSZ accepted an offset inside the fixed section")
	}
}

func TestGeneratedCodecsRejectMalformedPayloads(t *testing.T) {
	cases := []struct {
		name     string
		offsetAt int
		new      func() SSZMessage
	}{
		{"PingRequest", 0, func() SSZMessage { return new(PingRequest) }},
		{"PongResponse", 0, func() SSZMessage { return new(PongResponse) }},
		{"ChainInfo", 0, func() SSZMessage { return new(ChainInfo) }},
		{"ConnectToChainsRequest", 0, func() SSZMessage { return new(ConnectToChainsRequest) }},
		{"ConnectToChainsResponse", 0, func() SSZMessage { return new(ConnectToChainsResponse) }},
		{"HelloCommand", 58, func() SSZMessage { return new(HelloCommand) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.new().MarshalSSZ()
			if err != nil {
				t.Fatalf("MarshalSSZ empty value: %v", err)
			}
			if len(encoded) < tc.offsetAt+4 {
				t.Fatalf("empty encoding too short for offset mutation: %x", encoded)
			}

			badOffset := append([]byte(nil), encoded...)
			for i := 0; i < 4; i++ {
				badOffset[tc.offsetAt+i] = 0
			}
			if err := tc.new().UnmarshalSSZ(badOffset); err == nil {
				t.Fatal("UnmarshalSSZ accepted a zero first offset")
			}

			if err := tc.new().UnmarshalSSZ(encoded[:len(encoded)-1]); err == nil {
				t.Fatal("UnmarshalSSZ accepted a truncated fixed section")
			}
		})
	}
}

func TestResultListGoldenEncoding(t *testing.T) {
	msg := &ConnectToChainsResponse{ResultList: [][]byte{nil, []byte("x")}}
	got, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	// The container points to ResultList at offset 4. ResultList's two
	// variable-size elements are located by offsets 8 and 8; the first element
	// is empty and the second contains "x".
	want := []byte{
		0x04, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x08, 0x00, 0x00, 0x00,
		0x78,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("result list encoding mismatch:\n  got  %x\n  want %x", got, want)
	}
}

// ---- golden encoding ----

// TestSSZGoldenEncoding pins the encoding of a PongResponse with a dynamic ID
// and a fixed-width ChainID.
//
//	PongResponse{ID: {0x01,0x02}, ChainID: 1}
//	  offset(ID) = 8 -> 08 00 00 00
//	  ChainID    = 1 -> 01 00 00 00
//	  ID            -> 01 02
func TestSSZGoldenEncoding(t *testing.T) {
	pong := &PongResponse{ID: []byte{0x01, 0x02}, ChainID: 1}
	got, err := pong.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	want := []byte{0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x02}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch:\n  got  %x\n  want %x", got, want)
	}

	// Empty ID: its offset points to the end of the fixed section.
	empty, err := (&PongResponse{}).MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	wantEmpty := []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(empty, wantEmpty) {
		t.Fatalf("empty golden mismatch:\n  got  %x\n  want %x", empty, wantEmpty)
	}
}

// ---- helpers ----

func mustMarshalSSZ(t *testing.T, msg SSZMessage) []byte {
	t.Helper()
	enc, err := msg.MarshalSSZ()
	if err != nil {
		t.Fatalf("MarshalSSZ: %v", err)
	}
	return enc
}
