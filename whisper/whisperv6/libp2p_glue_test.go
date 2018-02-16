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
	"encoding/binary"
	"context"
	"bytes"
	"testing"
	"math/rand"

	"github.com/ethereum/go-ethereum/p2p"
	host "github.com/libp2p/go-libp2p-host"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	testutil "github.com/libp2p/go-testutil"
	inet "github.com/libp2p/go-libp2p-net"
)

const (
	testProtocolID = "/whispertesting/6.1"
)

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

	hosts[0].SetStreamHandler(testProtocolID, func (s inet.Stream) {
		defer s.Close()

		raw := make([]byte, size + codeLength + payloadSizeLength)
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
