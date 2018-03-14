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
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/p2p"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	set "gopkg.in/fatih/set.v0"
)

// LibP2PStream is a wrapper used to implement the MsgReadWriter
// interface for libp2p's streams.
type LibP2PStream struct {
	stream inet.Stream
}

const (
	codeLength        = 8
	payloadSizeLength = 4
)

// ReadMsg implements the MsgReadWriter interface to read messages
// from lilbp2p streams.
func (stream *LibP2PStream) ReadMsg() (p2p.Msg, error) {
	codeBytes := make([]byte, codeLength)
	nbytes, err := stream.stream.Read(codeBytes)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != len(codeBytes) {
		return p2p.Msg{}, fmt.Errorf("Invalid message header length: expected %d, got %d", codeLength, nbytes)
	}
	code := binary.LittleEndian.Uint64(codeBytes)

	sizeBytes := make([]byte, payloadSizeLength)
	nbytes, err = stream.stream.Read(sizeBytes)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != len(sizeBytes) {
		return p2p.Msg{}, fmt.Errorf("Invalid message size length: expected %d, got %d", len(sizeBytes), nbytes)
	}
	size := binary.LittleEndian.Uint32(sizeBytes)
	if size > math.MaxInt32 {
		return p2p.Msg{}, fmt.Errorf("Invalid message size length: got %d which is above the max of %d", size, math.MaxInt32)
	}

	payload := make([]byte, size)
	nbytes, err = stream.stream.Read(payload)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != int(size) {
		return p2p.Msg{}, fmt.Errorf("Invalid message payload length: expected %d, got %d", size, nbytes)
	}

	return p2p.Msg{Code: code, Size: size, Payload: bytes.NewReader(payload)}, nil
}

// WriteMsg implements the MsgReadWriter interface to write messages
// to lilbp2p streams.
func (stream *LibP2PStream) WriteMsg(msg p2p.Msg) error {
	// Refuse to write messages with an unsigned size greater than
	// a signed 32-bit integer size. This is because len() returns
	// an int, forcing a conversion at some locations in the code,
	// and on some blatforms that might cause an issue.
	if msg.Size > math.MaxInt32 {
		return fmt.Errorf("Payload size must be a maximum of %d bytes", math.MaxInt32)
	}

	data := make([]byte, msg.Size+codeLength+payloadSizeLength)

	binary.LittleEndian.PutUint64(data[0:codeLength], msg.Code)
	binary.LittleEndian.PutUint32(data[codeLength:codeLength+payloadSizeLength], msg.Size)

	nbytes, err := msg.Payload.Read(data[codeLength+payloadSizeLength:])
	if nbytes > math.MaxInt32 || uint32(nbytes) != msg.Size {
		return fmt.Errorf("Invalid size read in libp2p stream: read %d bytes, was expecting %d bytes", nbytes, msg.Size)
	} else if err != nil {
		return err
	}

	nbytes, err = stream.stream.Write(data)

	if err != nil {
		return err
	}

	if nbytes != len(data) {
		return fmt.Errorf("Invalid size written in libp2p stream: wrote %d bytes, was expecting %d bytes", nbytes, msg.Size)
	}

	return nil
}

// LibP2PPeer implements Peer for libp2p
type LibP2PPeer struct {
	*PeerBase

	id peer.ID
}

func newLibP2PPeer(w *Whisper, pid peer.ID, rw p2p.MsgReadWriter) Peer {
	return &LibP2PPeer{
		&PeerBase{
			host:           w,
			ws:             rw,
			trusted:        false,
			powRequirement: 0.0,
			known:          set.New(),
			quit:           make(chan struct{}),
			bloomFilter:    makeFullNodeBloom(),
			fullNode:       true,
		},
		pid,
	}
}

// ID returns the id of the peer
func (p *LibP2PPeer) ID() string {
	return p.id.String()
}

func (p *LibP2PPeer) handshake() error {
	err := p.handshakeBase()
	if err != nil {
		return fmt.Errorf("peer [%x] %s", p.ID(), err.Error())
	}
	return nil
}

// LibP2PWhisperServer implements WhisperServer for libp2p.
type LibP2PWhisperServer struct {
	Host host.Host

	Peers []*LibP2PPeer
}

// Start starts the server
func (server *LibP2PWhisperServer) Start() error {
	server.Host.SetStreamHandler(WhisperProtocolString, func (stream inet.Stream) {
		defer stream.Close()
	
		pid := stream.Conn().RemotePeer()
		var peer Peer
		for _, p := range server.Peers {
			if p.id == pid {
				peer = p
				break
			}
		}

		whisper := server.Peers[0].host
		lps := &LibP2PStream{stream}

		// Unknown peer
		if peer == nil {
			peer = newLibP2PPeer(whisper, pid, lps)
			// TODO check critical section
			server.Peers = append(server.Peers, peer.(*LibP2PPeer))
		}

		whisper.runMessageLoop(peer, lps)
	})
	return nil
}

// Stop stops the server
func (server *LibP2PWhisperServer) Stop() {
	server.Host.Close()
}

// PeerCount returns the peer count for the node
func (server *LibP2PWhisperServer) PeerCount() int {
	return 0
}

// Enode returns the enode address of the node
func (server *LibP2PWhisperServer) Enode() string {
	addr := server.Host.Addrs()[0]
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", server.Host.ID().Pretty()))
	fullAddr := addr.Encapsulate(hostAddr)
	return fullAddr.String()
}

func (server *LibP2PWhisperServer) AddPeer(addr ma.Multiaddr) *LibP2PPeer {
	fmt.Println("Adding peer: ", addr)
	pid, err := addr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		// XXX
		panic(err)
	}
	return &LibP2PPeer{id: peer.ID(pid)}
}

// NewLibP2PWhisperServer creates a new WhisperServer with
// a libp2p backend.
func NewLibP2PWhisperServer() (WhisperServer, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 384)
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", WhisperPort)),
		libp2p.Identity(priv),
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("Error setting up the libp2p network: %s", err)
	}

	server := &LibP2PWhisperServer{h, []*LibP2PPeer{}}
	return server, nil
}
