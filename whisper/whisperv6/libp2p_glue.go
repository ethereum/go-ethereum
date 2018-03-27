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
	"github.com/ethereum/go-ethereum/log"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/p2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	set "gopkg.in/fatih/set.v0"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
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

	connectionStream *LibP2PStream
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
		nil,
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

	whisper *Whisper
}

func (server *LibP2PWhisperServer) connectToPeer(p *LibP2PPeer) error {
	log.Info("opening stream to peer: ", p.id.Pretty(), "from peer", server.Host.ID().Pretty())

	// Create a stream with the peer
	s, err := server.Host.NewStream(context.Background(), p.id, WhisperProtocolString)
	if err != nil {
		panic(err)
	}

	// Save the stream
	lps := LibP2PStream{
		stream: s,
	}
	p.connectionStream = &lps

	// Send a first message to notify the remote peer we want
	// to connect
	connectMsg := p2p.Msg{
		Code: lp2pPeerCode,
		Size: 0,
		Payload: bytes.NewReader([]byte{0}),
	}

	err = lps.WriteMsg(connectMsg)
	
	return err
}

// Start starts the server
func (server *LibP2PWhisperServer) Start() error {
	server.Host.SetStreamHandler(WhisperProtocolString, func (stream inet.Stream) {
		log.Info("opening stream from new peer")
	
		pid := stream.Conn().RemotePeer()
		var peer Peer
		for _, p := range server.Peers {
			if p.id == pid {
				peer = p
				break
			}
		}

		lps := &LibP2PStream{stream}

		// Unknown peer
		if peer == nil {
			peer = newLibP2PPeer(server.whisper, pid, lps)
			// TODO check critical section
			server.Peers = append(server.Peers, peer.(*LibP2PPeer))
		}

		go server.whisper.runMessageLoop(peer, lps)
	})

	fmt.Println("Currently having the following peers:", server.Peers)

	// Open a stream to every peer currently known
	var err error
	for _, p := range server.Peers {
		if e := server.connectToPeer(p); e != nil {
			err = e
		}
		}

	return err
}

// Stop stops the server
func (server *LibP2PWhisperServer) Stop() {
	for _, p := range server.Peers {
		// TODO send disconnect message
		p.connectionStream.stream.Close()
	}

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

// AddPeer is a helper function to add peers to the server
func (server *LibP2PWhisperServer) AddPeer(addr ma.Multiaddr) *LibP2PPeer {
	log.Info("Adding peer: ", addr)
	pid, err := addr.ValueForProtocol(ma.P_IPFS)
	if err != nil {
		// XXX
		panic(err)
	}
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		panic(err)
	}
	ipfsaddrpart, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", pid))
	ipaddr := addr.Decapsulate(ipfsaddrpart)
	server.Host.Peerstore().AddAddr(peerid, ipaddr, pstore.PermanentAddrTTL)
	newPeer := &LibP2PPeer{id: peerid}
	server.Peers = append(server.Peers, newPeer)

	return newPeer
}

// NewLibP2PWhisperServer creates a new WhisperServer with
// a libp2p backend.
func NewLibP2PWhisperServer(port uint, whisper *Whisper) (WhisperServer, error) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, 384)
	if err != nil {
		return nil, fmt.Errorf("Error creating libp2p server: %v", err)
	}
	nodeID, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("Error creating libp2p server identity: %v pubkey=%v", err, pub)
	}
	serverAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil, fmt.Errorf("Error creating libp2p server address: %v port=%d", err, port)
	}

	ps := pstore.NewPeerstore()
	ps.AddPrivKey(nodeID, priv)
	ps.AddPubKey(nodeID, pub)

	network, err := swarm.NewNetwork(context.Background(), []ma.Multiaddr{serverAddr}, nodeID, ps, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating libp2p network: %v port=%d", err, port)
	}
	h := basichost.New(network)

	server := &LibP2PWhisperServer{h, []*LibP2PPeer{}, whisper}
	return server, nil
}