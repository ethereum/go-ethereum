package client

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	pssServiceName = "pss"
	bzzServiceName = "bzz"
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

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
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
		nodes[i], err = net.NewNodeWithConfig(&adapters.NodeConfig{
			Services: []string{"bzz", "pss"},
		})
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
	stateStore := newTestStore()
	kademlias := make(map[discover.NodeID]*network.Kademlia)
	kademlia := func(id discover.NodeID) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		addr := network.NewAddrFromNodeID(id)
		params := network.NewKadParams()
		params.MinProxBinSize = 2
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(addr.Over(), params)
		return kademlias[id]
	}
	return adapters.Services{
		"pss": func(ctx *adapters.ServiceContext) (node.Service, error) {
			cachedir, err := ioutil.TempDir("", "pss-cache")
			if err != nil {
				return nil, fmt.Errorf("create pss cache tmpdir failed", "error", err)
			}
			dpa, err := storage.NewLocalDPA(cachedir, make([]byte, 32))
			if err != nil {
				return nil, fmt.Errorf("local dpa creation failed", "error", err)
			}
			ctxlocal, _ := context.WithTimeout(context.Background(), time.Second)
			keys, err := wapi.NewKeyPair(ctxlocal)
			privkey, err := w.GetPrivateKey(keys)
			psparams := pss.NewPssParams(privkey)
			pskad := kademlia(ctx.Config.ID)
			ps := pss.NewPss(pskad, dpa, psparams)
			pshparams := pss.NewHandshakeParams()
			pshparams.SymKeySendLimit = sendLimit
			err = pss.SetHandshakeController(ps, pshparams)
			if err != nil {
				return nil, fmt.Errorf("handshake controller fail: %v", err)
			}
			return ps, nil
		},
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := network.NewAddrFromNodeID(ctx.Config.ID)
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore), nil
		},
	}
}

// copied from swarm/network/protocol_test_go
type testStore struct {
	sync.Mutex

	values map[string][]byte
}

func newTestStore() *testStore {
	return &testStore{values: make(map[string][]byte)}
}

func (t *testStore) Load(key string) ([]byte, error) {
	return nil, nil
}

func (t *testStore) Save(key string, v []byte) error {
	return nil
}
