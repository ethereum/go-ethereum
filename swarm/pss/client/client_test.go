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

package client

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/state"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

type protoCtrl struct {
	C        chan bool
	protocol *pss.Protocol
	run      func(*p2p.Peer, p2p.MsgReadWriter) error
}

var (
	debugdebugflag = flag.Bool("vv", false, "veryverbose")
	debugflag      = flag.Bool("v", false, "verbose")
	w              *whisper.Whisper
	wapi           *whisper.PublicWhisperAPI
	// custom logging
	psslogmain   log.Logger
	pssprotocols map[string]*protoCtrl
	sendLimit    = uint16(256)
)

var services = newServices()

func init() {
	flag.Parse()
	rand.Seed(time.Now().Unix())

	adapters.RegisterServices(services)

	loglevel := log.LvlInfo
	if *debugflag {
		loglevel = log.LvlDebug
	} else if *debugdebugflag {
		loglevel = log.LvlTrace
	}

	psslogmain = log.New("psslog", "*")
	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	hf := log.LvlFilterHandler(loglevel, hs)
	h := log.CallerFileHandler(hf)
	log.Root().SetHandler(h)

	w = whisper.New(&whisper.DefaultConfig)
	wapi = whisper.NewPublicWhisperAPI(w)

	pssprotocols = make(map[string]*protoCtrl)
}

// ping pong exchange across one expired symkey
func TestClientHandshake(t *testing.T) {
	sendLimit = 3

	clients, err := setupNetwork(2)
	if err != nil {
		t.Fatal(err)
	}

	lpsc, err := NewClientWithRPC(clients[0])
	if err != nil {
		t.Fatal(err)
	}
	rpsc, err := NewClientWithRPC(clients[1])
	if err != nil {
		t.Fatal(err)
	}
	lpssping := &pss.Ping{
		OutC: make(chan bool),
		InC:  make(chan bool),
		Pong: false,
	}
	rpssping := &pss.Ping{
		OutC: make(chan bool),
		InC:  make(chan bool),
		Pong: false,
	}
	lproto := pss.NewPingProtocol(lpssping)
	rproto := pss.NewPingProtocol(rpssping)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err = lpsc.RunProtocol(ctx, lproto)
	if err != nil {
		t.Fatal(err)
	}
	err = rpsc.RunProtocol(ctx, rproto)
	if err != nil {
		t.Fatal(err)
	}
	topic := pss.PingTopic.String()

	var loaddr string
	err = clients[0].Call(&loaddr, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	var roaddr string
	err = clients[1].Call(&roaddr, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}

	var lpubkey string
	err = clients[0].Call(&lpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 1 pubkey fail: %v", err)
	}
	var rpubkey string
	err = clients[1].Call(&rpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 2 pubkey fail: %v", err)
	}

	err = clients[0].Call(nil, "pss_setPeerPublicKey", rpubkey, topic, roaddr)
	if err != nil {
		t.Fatal(err)
	}
	err = clients[1].Call(nil, "pss_setPeerPublicKey", lpubkey, topic, loaddr)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	roaddrbytes, err := hexutil.Decode(roaddr)
	if err != nil {
		t.Fatal(err)
	}
	err = lpsc.AddPssPeer(rpubkey, roaddrbytes, pss.PingProtocol)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	for i := uint16(0); i <= sendLimit; i++ {
		lpssping.OutC <- false
		got := <-rpssping.InC
		log.Warn("ok", "idx", i, "got", got)
		time.Sleep(time.Second)
	}

	rw := lpsc.peerPool[pss.PingTopic][rpubkey]
	lpsc.RemovePssPeer(rpubkey, pss.PingProtocol)
	if err := rw.WriteMsg(p2p.Msg{
		Size:    3,
		Payload: bytes.NewReader([]byte("foo")),
	}); err == nil {
		t.Fatalf("expected error on write")
	}
}

func setupNetwork(numnodes int) (clients []*rpc.Client, err error) {
	nodes := make([]*simulations.Node, numnodes)
	clients = make([]*rpc.Client, numnodes)
	if numnodes < 2 {
		return nil, fmt.Errorf("Minimum two nodes in network")
	}
	adapter := adapters.NewSimAdapter(services)
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "bzz",
	})
	for i := 0; i < numnodes; i++ {
		nodeconf := adapters.RandomNodeConfig()
		nodeconf.Services = []string{"bzz", "pss"}
		nodes[i], err = net.NewNodeWithConfig(nodeconf)
		if err != nil {
			return nil, fmt.Errorf("error creating node 1: %v", err)
		}
		err = net.Start(nodes[i].ID())
		if err != nil {
			return nil, fmt.Errorf("error starting node 1: %v", err)
		}
		if i > 0 {
			err = net.Connect(nodes[i].ID(), nodes[i-1].ID())
			if err != nil {
				return nil, fmt.Errorf("error connecting nodes: %v", err)
			}
		}
		clients[i], err = nodes[i].Client()
		if err != nil {
			return nil, fmt.Errorf("create node 1 rpc client fail: %v", err)
		}
	}
	if numnodes > 2 {
		err = net.Connect(nodes[0].ID(), nodes[len(nodes)-1].ID())
		if err != nil {
			return nil, fmt.Errorf("error connecting first and last nodes")
		}
	}
	return clients, nil
}

func newServices() adapters.Services {
	stateStore := state.NewInmemoryStore()
	kademlias := make(map[enode.ID]*network.Kademlia)
	kademlia := func(id enode.ID) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		params := network.NewKadParams()
		params.MinProxBinSize = 2
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(id[:], params)
		return kademlias[id]
	}
	return adapters.Services{
		"pss": func(ctx *adapters.ServiceContext) (node.Service, error) {
			ctxlocal, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			keys, err := wapi.NewKeyPair(ctxlocal)
			if err != nil {
				return nil, err
			}
			privkey, err := w.GetPrivateKey(keys)
			if err != nil {
				return nil, err
			}
			psparams := pss.NewPssParams().WithPrivateKey(privkey)
			pskad := kademlia(ctx.Config.ID)
			ps, err := pss.NewPss(pskad, psparams)
			if err != nil {
				return nil, err
			}
			pshparams := pss.NewHandshakeParams()
			pshparams.SymKeySendLimit = sendLimit
			err = pss.SetHandshakeController(ps, pshparams)
			if err != nil {
				return nil, fmt.Errorf("handshake controller fail: %v", err)
			}
			return ps, nil
		},
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := network.NewAddr(ctx.Config.Node())
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore, nil, nil), nil
		},
	}
}

// copied from swarm/network/protocol_test_go
type testStore struct {
	sync.Mutex

	values map[string][]byte
}

func (t *testStore) Load(key string) ([]byte, error) {
	return nil, nil
}

func (t *testStore) Save(key string, v []byte) error {
	return nil
}
