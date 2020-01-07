// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"math/big"
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	"github.com/ethereum/go-ethereum/les/protocol"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const protocolVersion = protocol.Lpv2

var (
	hash    = common.HexToHash("deadbeef")
	genesis = common.HexToHash("cafebabe")
	headNum = uint64(1234)
	td      = big.NewInt(123)
)

func newNodeID(t *testing.T) *enode.Node {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal("generate key err:", err)
	}
	return enode.NewV4(&key.PublicKey, net.IP{}, 35000, 35000)
}

// ulc connects to trusted peer and send announceType=announceTypeSigned
func TestPeerHandshakeSetAnnounceTypeToAnnounceTypeSignedForTrustedPeer(t *testing.T) {
	id := newNodeID(t).ID()

	// peer to connect(on ulc side)
	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		trusted: true,
		rw: &rwStub{
			WriteHook: func(recvList protocol.KeyValueList) {
				recv, _ := recvList.ToMap()
				var reqType uint64
				err := recv.Get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}
				if reqType != announceTypeSigned {
					t.Fatal("Expected announceTypeSigned")
				}
			},
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("serveHeaders", nil)
				l = l.Add("serveChainSince", uint64(0))
				l = l.Add("serveStateSince", uint64(0))
				l = l.Add("txRelay", nil)
				l = l.Add("flowControl/BL", uint64(0))
				l = l.Add("flowControl/MRR", uint64(0))
				l = l.Add("flowControl/MRC", testCostList(0))
				return l
			},
		},
		network: protocol.NetworkId,
	}
	err := p.Handshake(td, hash, headNum, genesis, nil)
	if err != nil {
		t.Fatalf("Handshake error: %s", err)
	}
	if p.announceType != announceTypeSigned {
		t.Fatal("Incorrect announceType")
	}
}

func TestPeerHandshakeAnnounceTypeSignedForTrustedPeersPeerNotInTrusted(t *testing.T) {
	id := newNodeID(t).ID()
	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		rw: &rwStub{
			WriteHook: func(recvList protocol.KeyValueList) {
				// checking that ulc sends to peer allowedRequests=noRequests and announceType != announceTypeSigned
				recv, _ := recvList.ToMap()
				var reqType uint64
				err := recv.Get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}
				if reqType == announceTypeSigned {
					t.Fatal("Expected not announceTypeSigned")
				}
			},
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("serveHeaders", nil)
				l = l.Add("serveChainSince", uint64(0))
				l = l.Add("serveStateSince", uint64(0))
				l = l.Add("txRelay", nil)
				l = l.Add("flowControl/BL", uint64(0))
				l = l.Add("flowControl/MRR", uint64(0))
				l = l.Add("flowControl/MRC", testCostList(0))
				return l
			},
		},
		network: protocol.NetworkId,
	}
	err := p.Handshake(td, hash, headNum, genesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.announceType == announceTypeSigned {
		t.Fatal("Incorrect announceType")
	}
}

func TestPeerHandshakeDefaultAllRequests(t *testing.T) {
	id := newNodeID(t).ID()

	s := generateLesServer()

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		rw: &rwStub{
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("announceType", uint64(announceTypeSigned))
				l = l.Add("allowedRequests", uint64(0))
				return l
			},
		},
		network: protocol.NetworkId,
	}

	err := p.Handshake(td, hash, headNum, genesis, s)
	if err != nil {
		t.Fatal(err)
	}

	if p.onlyAnnounce {
		t.Fatal("Incorrect announceType")
	}
}

func TestPeerHandshakeServerSendOnlyAnnounceRequestsHeaders(t *testing.T) {
	id := newNodeID(t).ID()

	s := generateLesServer()
	s.config.UltraLightOnlyAnnounce = true

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		rw: &rwStub{
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("announceType", uint64(announceTypeSigned))
				return l
			},
			WriteHook: func(l protocol.KeyValueList) {
				for _, v := range l {
					if v.Key == "serveHeaders" ||
						v.Key == "serveChainSince" ||
						v.Key == "serveStateSince" ||
						v.Key == "txRelay" {
						t.Fatalf("%v exists", v.Key)
					}
				}
			},
		},
		network: protocol.NetworkId,
	}

	err := p.Handshake(td, hash, headNum, genesis, s)
	if err != nil {
		t.Fatal(err)
	}
}
func TestPeerHandshakeClientReceiveOnlyAnnounceRequestsHeaders(t *testing.T) {
	id := newNodeID(t).ID()

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		rw: &rwStub{
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("flowControl/BL", uint64(0))
				l = l.Add("flowControl/MRR", uint64(0))
				l = l.Add("flowControl/MRC", protocol.RequestCostList{})

				l = l.Add("announceType", uint64(announceTypeSigned))

				return l
			},
		},
		network: protocol.NetworkId,
		trusted: true,
	}

	err := p.Handshake(td, hash, headNum, genesis, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !p.onlyAnnounce {
		t.Fatal("onlyAnnounce must be true")
	}
}

func TestPeerHandshakeClientReturnErrorOnUselessPeer(t *testing.T) {
	id := newNodeID(t).ID()

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocolVersion,
		rw: &rwStub{
			ReadHook: func(l protocol.KeyValueList) protocol.KeyValueList {
				l = l.Add("flowControl/BL", uint64(0))
				l = l.Add("flowControl/MRR", uint64(0))
				l = l.Add("flowControl/MRC", protocol.RequestCostList{})
				l = l.Add("announceType", uint64(announceTypeSigned))
				return l
			},
		},
		network: protocol.NetworkId,
	}

	err := p.Handshake(td, hash, headNum, genesis, nil)
	if err == nil {
		t.FailNow()
	}
}

func generateLesServer() *LesServer {
	s := &LesServer{
		lesCommons: lesCommons{
			config: &eth.Config{UltraLightOnlyAnnounce: true},
		},
		defParams: flowcontrol.ServerParams{
			BufLimit:    uint64(300000000),
			MinRecharge: uint64(50000),
		},
		fcManager: flowcontrol.NewClientManager(nil, &mclock.System{}),
	}
	s.costTracker, _ = newCostTracker(rawdb.NewMemoryDatabase(), s.config)
	return s
}

type rwStub struct {
	ReadHook  func(l protocol.KeyValueList) protocol.KeyValueList
	WriteHook func(l protocol.KeyValueList)
}

func (s *rwStub) ReadMsg() (p2p.Msg, error) {
	payload := protocol.KeyValueList{}
	payload = payload.Add("protocolVersion", uint64(protocolVersion))
	payload = payload.Add("networkId", uint64(protocol.NetworkId))
	payload = payload.Add("headTd", td)
	payload = payload.Add("headHash", hash)
	payload = payload.Add("headNum", headNum)
	payload = payload.Add("genesisHash", genesis)

	if s.ReadHook != nil {
		payload = s.ReadHook(payload)
	}
	size, p, err := rlp.EncodeToReader(payload)
	if err != nil {
		return p2p.Msg{}, err
	}
	return p2p.Msg{
		Size:    uint32(size),
		Payload: p,
	}, nil
}

func (s *rwStub) WriteMsg(m p2p.Msg) error {
	recvList := protocol.KeyValueList{}
	if err := m.Decode(&recvList); err != nil {
		return err
	}
	if s.WriteHook != nil {
		s.WriteHook(recvList)
	}
	return nil
}
