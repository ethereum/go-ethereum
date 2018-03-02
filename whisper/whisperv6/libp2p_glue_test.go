// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisperv6

import (
	"bytes"
	"context"
	"encoding/binary"
	"io/ioutil"
	"math"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/p2p"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	testutil "github.com/libp2p/go-testutil"
)

const (
	testProtocolID = "/whispertesting/6.1"
)

// Create a network with n mock hosts. Each host in the array is linked to
// all hosts preceding it, and has dialed them.
func createTestNetwork(ctx context.Context, t *testing.T, n int) []host.Host {
	net := make([]host.Host, n)
	mn := mocknet.New(ctx)

	for i := 0; i < n; i++ {
		a := testutil.RandLocalTCPAddress()
		sk, _, err := testutil.RandTestKeyPair(512)
		if err != nil {
			t.Fatal(err)
		}
		h, err := mn.AddPeer(sk, a)
		if err != nil {
			t.Fatal(err)
		}
		n := h.Network()

		net[i] = h

		// Link to all previous hosts
		for j := 0; j < i; j++ {
			_, err = mn.LinkPeers(net[j].ID(), h.ID())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := n.DialPeer(ctx, net[j].ID()); err != nil {
				t.Error(err)
			}
		}
	}

	return net
}

func TestSimpleCode(t *testing.T) {
	ctx := context.Background()
	hosts := createTestNetwork(ctx, t, 2)

	code := rand.Uint64()
	size := rand.Uint32() % 512
	payload := make([]byte, size)
	n, err := rand.Read(payload)
	if err != nil || uint32(n) != size {
		t.Fatalf("Read %d random bytes instead of the expected %d, err: %v", n, size, err)
	}

	hosts[0].SetStreamHandler(testProtocolID, func(s inet.Stream) {
		defer s.Close()

		raw := make([]byte, size+codeLength+payloadSizeLength)
		n, err = s.Read(raw)

		if len(raw) != n || err != nil {
			t.Fatalf("Error reading output of encoding (%d/%d bytes) %s", n, len(raw), err)
		}

		c := binary.LittleEndian.Uint64(raw[:8])
		if c != code {
			t.Fatalf("Invalid code retreived %d, expected %d", c, code)
		}
		decodedSize := binary.LittleEndian.Uint32(raw[8:12])
		if decodedSize != size {
			t.Fatalf("Invalid size retreived %d, expected %d", decodedSize, size)
		}

		if !bytes.Equal(payload, raw[codeLength+payloadSizeLength:]) {
			t.Fatalf("Encoded payload differ from source")
		}
	})
	stream, err := hosts[1].NewStream(ctx, hosts[0].ID(), testProtocolID)
	if err != nil {
		t.Fatal(err)
	}

	msg := p2p.Msg{
		Code:    code,
		Size:    size,
		Payload: bytes.NewReader(payload),
	}

	lps := LibP2PStream{
		stream: stream,
	}

	err = lps.WriteMsg(msg)

	if err != nil {
		t.Fatalf("Error encoding a message to the stream: %s", err)
	}

	stream.Close()
}

func TestSimpleDecode(t *testing.T) {
	coded := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}

	ctx := context.Background()
	hosts := createTestNetwork(ctx, t, 2)

	hosts[0].SetStreamHandler(testProtocolID, func(s inet.Stream) {
		defer s.Close()

		lps := LibP2PStream{
			stream: s,
		}

		msg, err := lps.ReadMsg()
		if err != nil {
			t.Fatalf("Error decoding message: %s", err)
		}

		if msg.Code != 0xdeadbeef {
			t.Fatalf("Error decoding message code %d instead of %d", msg.Code, 0xdeadbeef)
		}
		if msg.Size != 5 {
			t.Fatalf("Error decoding message size %d instead of %d", msg.Size, 5)
		}
	})

	stream, err := hosts[1].NewStream(ctx, hosts[0].ID(), testProtocolID)
	if err != nil {
		t.Fatal(err)
	}

	n, err := stream.Write(coded)
	if err != nil || n != len(coded) {
		t.Fatalf("Error writing %d bytes to stream: %s, %d bytes written", len(coded), err, n)
	}

	stream.Close()
}

func TestSimpleCodeDecode(t *testing.T) {
	ctx := context.Background()
	hosts := createTestNetwork(ctx, t, 2)

	code := rand.Uint64()
	size := rand.Uint32() % 512
	payload := make([]byte, size)
	n, err := rand.Read(payload)
	if err != nil || uint32(n) != size {
		t.Fatalf("Read %d random bytes instead of the expected %d, err: %v", n, size, err)
	}

	hosts[0].SetStreamHandler(testProtocolID, func(s inet.Stream) {
		defer s.Close()

		lps := LibP2PStream{
			stream: s,
		}

		msg, err := lps.ReadMsg()
		if err != nil {
			t.Fatalf("Error decoding message: %s", err)
		}

		if msg.Code != code {
			t.Fatalf("Error decoding message code %d instead of %d", msg.Code, code)
		}
		if int(msg.Size) != len(payload) {
			t.Fatalf("Error decoding message size %d instead of %d", msg.Size, len(payload))
		}

		readPayload := make([]byte, len(payload))
		sizeRead, err := msg.Payload.Read(readPayload)
		if err != nil || sizeRead != len(payload) {
			t.Fatalf("Error reading payload from source: %s (%d bytes read for %d expected)", err, sizeRead, len(payload))
		} else if !bytes.Equal(payload, readPayload) {
			t.Fatal("Encoded payload differ from source")
		}
	})
	stream, err := hosts[1].NewStream(ctx, hosts[0].ID(), testProtocolID)
	if err != nil {
		t.Fatal(err)
	}

	msg := p2p.Msg{
		Code:    code,
		Size:    size,
		Payload: bytes.NewReader(payload),
	}

	lps := LibP2PStream{
		stream: stream,
	}

	err = lps.WriteMsg(msg)

	if err != nil {
		t.Fatalf("Error encoding a message to the stream: %s", err)
	}

	stream.Close()
}

func TestMaxWriteSize(t *testing.T) {
	ctx := context.Background()
	hosts := createTestNetwork(ctx, t, 2)

	code := rand.Uint64()
	// This isn't the size that will be reported, but if I actually
	// require 2GB or RAM the CI servers will fail.
	size := 10
	payload := make([]byte, size)
	n, err := rand.Read(payload)
	if err != nil || n != size {
		t.Fatalf("Read %d random bytes instead of the expected %d, err: %v", n, size, err)
	}

	hosts[0].SetStreamHandler(testProtocolID, func(s inet.Stream) {
		defer s.Close()

		dummy := []byte{0x0}
		_, _ = s.Read(dummy)
	})
	stream, err := hosts[1].NewStream(ctx, hosts[0].ID(), testProtocolID)
	if err != nil {
		t.Fatal(err)
	}

	msg := p2p.Msg{
		Code:    code,
		Size:    math.MaxInt32 + 1,
		Payload: bytes.NewReader(payload),
	}

	lps := LibP2PStream{
		stream: stream,
	}

	err = lps.WriteMsg(msg)

	if err.Error() != "Payload size must be a maximum of 2147483647 bytes" {
		t.Fatal("Should have returned an error with invalid payload size")
	}

	stream.Close()
}

func TestMaxReadSize(t *testing.T) {
	coded := []byte{0xef, 0xbe, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0xF0, 0x01, 0x02, 0x03, 0x04, 0x05}

	ctx := context.Background()
	hosts := createTestNetwork(ctx, t, 2)

	hosts[0].SetStreamHandler(testProtocolID, func(s inet.Stream) {
		defer s.Close()

		lps := LibP2PStream{
			stream: s,
		}

		_, err := lps.ReadMsg()
		if err.Error() != "Invalid message size length: got 4026531845 which is above the max of 2147483647" {
			t.Fatal("Did not detect an invalid payload size")
		}

		// WORKAROUND Need to read the whole content of the stream for the
		// stream to be properly closed by the underlying implementation.
		_, _ = ioutil.ReadAll(s)
	})

	stream, err := hosts[1].NewStream(ctx, hosts[0].ID(), testProtocolID)
	if err != nil {
		t.Fatal(err)
	}

	n, err := stream.Write(coded)
	if err != nil || n != len(coded) {
		t.Fatalf("Error writing %d bytes to stream: %s, %d bytes written", len(coded), err, n)
	}

	stream.Close()
}
