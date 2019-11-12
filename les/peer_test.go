package les

import (
	"math/big"
	"testing"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/mclock"
	"github.com/maticnetwork/bor/les/flowcontrol"
	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/rlp"
)

const (
	test_networkid   = 10
	protocol_version = lpv2
)

var (
	hash    = common.HexToHash("some string")
	genesis = common.HexToHash("genesis hash")
	headNum = uint64(1234)
	td      = big.NewInt(123)
)

//ulc connects to trusted peer and send announceType=announceTypeSigned
func TestPeerHandshakeSetAnnounceTypeToAnnounceTypeSignedForTrustedPeer(t *testing.T) {
	id := newNodeID(t).ID()

	//peer to connect(on ulc side)
	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocol_version,
		trusted: true,
		rw: &rwStub{
			WriteHook: func(recvList keyValueList) {
				//checking that ulc sends to peer allowedRequests=onlyAnnounceRequests and announceType = announceTypeSigned
				recv, _ := recvList.decode()
				var reqType uint64

				err := recv.get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}

				if reqType != announceTypeSigned {
					t.Fatal("Expected announceTypeSigned")
				}
			},
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("serveHeaders", nil)
				l = l.add("serveChainSince", uint64(0))
				l = l.add("serveStateSince", uint64(0))
				l = l.add("txRelay", nil)
				l = l.add("flowControl/BL", uint64(0))
				l = l.add("flowControl/MRR", uint64(0))
				l = l.add("flowControl/MRC", testCostList(0))

				return l
			},
		},
		network: test_networkid,
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
		version: protocol_version,
		rw: &rwStub{
			WriteHook: func(recvList keyValueList) {
				//checking that ulc sends to peer allowedRequests=noRequests and announceType != announceTypeSigned
				recv, _ := recvList.decode()
				var reqType uint64

				err := recv.get("announceType", &reqType)
				if err != nil {
					t.Fatal(err)
				}

				if reqType == announceTypeSigned {
					t.Fatal("Expected not announceTypeSigned")
				}
			},
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("serveHeaders", nil)
				l = l.add("serveChainSince", uint64(0))
				l = l.add("serveStateSince", uint64(0))
				l = l.add("txRelay", nil)
				l = l.add("flowControl/BL", uint64(0))
				l = l.add("flowControl/MRR", uint64(0))
				l = l.add("flowControl/MRC", testCostList(0))

				return l
			},
		},
		network: test_networkid,
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
		version: protocol_version,
		rw: &rwStub{
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("announceType", uint64(announceTypeSigned))
				l = l.add("allowedRequests", uint64(0))

				return l
			},
		},
		network: test_networkid,
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
	s.onlyAnnounce = true

	p := peer{
		Peer:    p2p.NewPeer(id, "test peer", []p2p.Cap{}),
		version: protocol_version,
		rw: &rwStub{
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("announceType", uint64(announceTypeSigned))

				return l
			},
			WriteHook: func(l keyValueList) {
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
		network: test_networkid,
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
		version: protocol_version,
		rw: &rwStub{
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("flowControl/BL", uint64(0))
				l = l.add("flowControl/MRR", uint64(0))
				l = l.add("flowControl/MRC", RequestCostList{})

				l = l.add("announceType", uint64(announceTypeSigned))

				return l
			},
		},
		network: test_networkid,
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
		version: protocol_version,
		rw: &rwStub{
			ReadHook: func(l keyValueList) keyValueList {
				l = l.add("flowControl/BL", uint64(0))
				l = l.add("flowControl/MRR", uint64(0))
				l = l.add("flowControl/MRC", RequestCostList{})

				l = l.add("announceType", uint64(announceTypeSigned))

				return l
			},
		},
		network: test_networkid,
	}

	err := p.Handshake(td, hash, headNum, genesis, nil)
	if err == nil {
		t.FailNow()
	}
}

func generateLesServer() *LesServer {
	s := &LesServer{
		defParams: flowcontrol.ServerParams{
			BufLimit:    uint64(300000000),
			MinRecharge: uint64(50000),
		},
		fcManager: flowcontrol.NewClientManager(nil, &mclock.System{}),
	}
	return s
}

type rwStub struct {
	ReadHook  func(l keyValueList) keyValueList
	WriteHook func(l keyValueList)
}

func (s *rwStub) ReadMsg() (p2p.Msg, error) {
	payload := keyValueList{}
	payload = payload.add("protocolVersion", uint64(protocol_version))
	payload = payload.add("networkId", uint64(test_networkid))
	payload = payload.add("headTd", td)
	payload = payload.add("headHash", hash)
	payload = payload.add("headNum", headNum)
	payload = payload.add("genesisHash", genesis)

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
	recvList := keyValueList{}
	if err := m.Decode(&recvList); err != nil {
		return err
	}

	if s.WriteHook != nil {
		s.WriteHook(recvList)
	}

	return nil
}
