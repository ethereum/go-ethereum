package pss

import (
	"bytes"
	// "context"
	// "crypto/ecdsa"
	"encoding/hex"
	// "encoding/json"
	"flag"
	"fmt"
	// "io/ioutil"
	"math/rand"
	"os"
	// "sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	// "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	// "github.com/ethereum/go-ethereum/p2p/protocols"
	// "github.com/ethereum/go-ethereum/p2p/simulations"
	// "github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	// p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	// "github.com/ethereum/go-ethereum/swarm/storage"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
)

const (
	pssServiceName = "pss"
	bzzServiceName = "bzz"
)

var (
	snapshotfile string
	debugflag    = flag.Bool("v", false, "verbose")

	// custom logging
	psslogmain log.Logger
)

var services = newServices()

func init() {

	flag.Parse()
	rand.Seed(time.Now().Unix())

	adapters.RegisterServices(services)

	loglevel := log.LvlDebug
	if *debugflag {
		loglevel = log.LvlTrace
	}

	psslogmain = log.New("psslog", "*")
	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	hf := log.LvlFilterHandler(loglevel, hs)
	h := log.CallerFileHandler(hf)
	log.Root().SetHandler(h)

}

func TestKeys(t *testing.T) {
	key, err := crypto.GenerateKey()
	addr := network.RandomAddress.Over()
	topic := whisper.NewTopic("foo", 42)
}

func TestCache(t *testing.T) {
	var err error
	to, _ := hex.DecodeString("08090a0b0c0d0e0f1011121314150001020304050607161718191a1b1c1d1e1f")
	oaddr, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	proofbytes, _ := hex.DecodeString("822fff7527f7ae630c1224921e50a7ca1b27324f00f3966623bd503780c7ab33")
	ps := NewTestPss(oaddr)
	pp := NewPssParams(false)
	data := []byte("foo")
	datatwo := []byte("bar")
	fwdaddr := network.RandomAddr()
	msg := &PssMsg{
		Payload: &Envelope{
			TTL:     0,
			From:    oaddr,
			Topic:   PingTopic,
			Payload: data,
		},
		To: to,
	}

	msgtwo := &PssMsg{
		Payload: &Envelope{
			TTL:     0,
			From:    oaddr,
			Topic:   PingTopic,
			Payload: datatwo,
		},
		To: to,
	}

	digest, err := ps.storeMsg(msg)
	if err != nil {
		t.Fatalf("could not store cache msgone: %v", err)
	}
	digesttwo, err := ps.storeMsg(msgtwo)
	if err != nil {
		t.Fatalf("could not store cache msgtwo: %v", err)
	}

	if !bytes.Equal(digest[:], proofbytes) {
		t.Fatalf("digest - got: %x, expected: %x", digest, proofbytes)
	}

	if digest == digesttwo {
		t.Fatalf("different msgs return same crc: %d", digesttwo)
	}

	// check the sender cache
	err = ps.addFwdCacheSender(fwdaddr.Over(), digest)
	if err != nil {
		t.Fatalf("write to pss sender cache failed: %v", err)
	}

	if !ps.checkFwdCache(fwdaddr.Over(), digest) {
		t.Fatalf("message %v should have SENDER record in cache but checkCache returned false", msg)
	}

	if ps.checkFwdCache(fwdaddr.Over(), digesttwo) {
		t.Fatalf("message %v should NOT have SENDER record in cache but checkCache returned true", msgtwo)
	}

	// check the expire cache
	err = ps.addFwdCacheExpire(digest)
	if err != nil {
		t.Fatalf("write to pss expire cache failed: %v", err)
	}

	if !ps.checkFwdCache(nil, digest) {
		t.Fatalf("message %v should have EXPIRE record in cache but checkCache returned false", msg)
	}

	if ps.checkFwdCache(nil, digesttwo) {
		t.Fatalf("message %v should NOT have EXPIRE record in cache but checkCache returned true", msgtwo)
	}

	time.Sleep(pp.Cachettl)
	if ps.checkFwdCache(nil, digest) {
		t.Fatalf("message %v should have expired from cache but checkCache returned true", msg)
	}

	err = ps.AddToCache(fwdaddr.Over(), msgtwo)
	if err != nil {
		t.Fatalf("public accessor cache write failed: %v", err)
	}

	if !ps.checkFwdCache(fwdaddr.Over(), digesttwo) {
		t.Fatalf("message %v should have SENDER record in cache but checkCache returned false", msgtwo)
	}
}

func TestRegisterHandler(t *testing.T) {
	var err error
	addr := network.RandomAddr()
	ps := NewTestPss(addr.OAddr)
	from := network.RandomAddr()
	payload := []byte("payload")
	topic := NewTopic(pssSpec.Name, int(pssSpec.Version))
	wrongtopic := NewTopic("foo", 42)
	checkMsg := func(msg []byte, p *p2p.Peer, sender []byte) error {
		if !bytes.Equal(from.OAddr, sender) {
			return fmt.Errorf("sender mismatch. expected %x, got %x", from.OAddr, sender)
		}
		if !bytes.Equal(msg, payload) {
			return fmt.Errorf("sender mismatch. expected %x, got %x", msg, payload)
		}
		return nil
	}
	deregister := ps.Register(&topic, checkMsg)
	pssmsg := &PssMsg{Payload: NewEnvelope(from.OAddr, topic, payload)}
	err = ps.Process(pssmsg)
	if err != nil {
		t.Fatal(err)
	}
	var i int
	err = ps.Process(&PssMsg{Payload: NewEnvelope(from.OAddr, wrongtopic, payload)})
	expErr := ""
	if err == nil || err.Error() == expErr {
		t.Fatalf("unhandled topic expected '%v', got '%v'", expErr, err)
	}
	deregister2 := ps.Register(&topic, func(msg []byte, p *p2p.Peer, sender []byte) error { i++; return nil })
	err = ps.Process(pssmsg)
	if err != nil {
		t.Fatal(err)
	}
	if i != 1 {
		t.Fatalf("second registerer handler did not run")
	}
	deregister()
	deregister2()
	err = ps.Process(&PssMsg{Payload: NewEnvelope(from.OAddr, topic, payload)})
	expErr = ""
	if err == nil || err.Error() == expErr {
		t.Fatalf("reregister handler expected %v, got %v", expErr, err)
	}
}

//func TestSimpleLinear(t *testing.T) {
//	var err error
//	nodeconfig := adapters.RandomNodeConfig()
//	addr := network.NewAddrFromNodeID(nodeconfig.ID)
//	_ = p2ptest.NewTestPeerPool()
//	ps := NewTestPss(addr.Over())
//
//	ping := &Ping{
//		C: make(chan struct{}),
//	}
//
//	ps.Register(&PingTopic, RegisterPssProtocol(ps, &PingTopic, PingProtocol, NewPingProtocol(ping.PingHandler)).Handle)
//
//	if err != nil {
//		t.Fatalf("Failed to register virtual protocol in pss: %v", err)
//	}
//	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
//		id := p.ID()
//		pp := protocols.NewPeer(p, rw, pssSpec)
//		bp := &testOverlayConn{
//			Peer: pp,
//			addr: network.ToOverlayAddr(id[:]),
//		}
//		//a := pot.NewAddressFromBytes(bp.addr)
//		//ps.fwdPool[a] = pp
//		ps.fwdPool[id] = pp
//		ps.Overlay.On(bp)
//		defer ps.Overlay.Off(bp)
//		log.Debug(fmt.Sprintf("%v", ps.Overlay))
//		return bp.Run(ps.handlePssMsg)
//	}
//
//	pt := p2ptest.NewProtocolTester(t, nodeconfig.ID, 2, run)
//
//	msg := NewPingMsg(network.ToOverlayAddr(pt.IDs[0].Bytes()), PingProtocol, PingTopic, []byte{1, 2, 3})
//
//	exchange := p2ptest.Exchange{
//		Expects: []p2ptest.Expect{
//			p2ptest.Expect{
//				Code: 0,
//				Msg:  msg,
//				Peer: pt.IDs[0],
//			},
//		},
//		Triggers: []p2ptest.Trigger{
//			p2ptest.Trigger{
//				Code: 0,
//				Msg:  msg,
//				Peer: pt.IDs[1],
//			},
//		},
//	}
//
//	err = pt.TestExchanges(exchange)
//	if err != nil {
//		t.Fatalf("exchange failed %v", err)
//	}
//}
//
//func TestSnapshot_50_5(t *testing.T) {
//	testSnapshot(t, "testdata/snapshot_50.json", 5, true)
//}
//
//func TestSnapshot_5_50(t *testing.T) {
//	testSnapshot(t, "testdata/snapshot_5.json", 50, true)
//}
//
//func TestSnapshot_5_5(t *testing.T) {
//	testSnapshot(t, "testdata/snapshot_5.json", 5, true)
//}
//
//func testSnapshot(t *testing.T, snapshotfile string, msgcount int, sim bool) {
//
//
//	// choose the adapter to use
//	var adapter adapters.NodeAdapter
//	if sim {
//		adapter = adapters.NewSimAdapter(services)
//	} else {
//		baseDir, err := ioutil.TempDir("", "swarm-test")
//		if err != nil {
//			t.Fatal(err)
//		}
//		defer os.RemoveAll(baseDir)
//		adapter = adapters.NewExecAdapter(baseDir)
//	}
//
//	// process shapshot
//	jsonsnapshot, err := ioutil.ReadFile(snapshotfile)
//	if err != nil {
//		t.Fatalf("cant read snapshot: %s", snapshotfile)
//	}
//	snapshot := &simulations.Snapshot{}
//	err = json.Unmarshal(jsonsnapshot, snapshot)
//	if err != nil {
//		t.Fatalf("snapshot file unreadable: %v", err)
//	}
//	for _, node := range snapshot.Nodes {
//		node.Config.Services = []string{"bzz", "pss"}
//	}
//
//	// setup network with snapshot
//	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
//		ID: "0",
//	})
//	defer net.Shutdown()
//
//	err = net.Load(snapshot)
//	if err != nil {
//		t.Fatalf("invalid snapshot: %v", err)
//	}
//
//	timeout := 15 * time.Second
//	ctx, cancelmain := context.WithTimeout(context.Background(), timeout)
//	defer cancelmain()
//
//	// nodes expecting messages
//	recvids := make([]discover.NodeID, msgcount)
//
//	// the overlay address map to recvids
//	recvaddrs := make(map[discover.NodeID][]byte)
//
//	// messages actually received (registered through trigger and test check)
//	var msgreceived []discover.NodeID
//
//	// trigger for expect in test
//	trigger := make(chan discover.NodeID)
//
//	// one wait for every message
//	wg := sync.WaitGroup{}
//	wg.Add(msgcount)
//
//	action := func(ctx context.Context) error {
//		var rpcerr error
//		var rpcbyte []byte
//		for _, simnode := range net.Nodes {
//			if simnode == nil {
//				return fmt.Errorf("unknown node: %s", simnode.ID())
//			}
//			client, err := simnode.Client()
//			if err != nil {
//				return fmt.Errorf("error getting recp node client: %s", err)
//			}
//
//			err = client.Call(&rpcbyte, "pss_baseAddr")
//			if err != nil {
//				t.Fatalf("cant get overlayaddr: %v", err)
//			}
//
//			recvaddrs[simnode.ID()] = rpcbyte
//			err = client.Call(&rpcbyte, "pss_baseAddr")
//			if err != nil {
//				t.Fatalf("cant get overlayaddr: %v", err)
//			}
//
//			err = triggerChecks(ctx, &wg, &trigger, net, simnode.ID())
//			if err != nil {
//				t.Fatalf("trigger setup failed: %v", err)
//			}
//		}
//		for i := 0; i < msgcount; i++ {
//
//			idx := rand.Intn(len(net.Nodes))
//			sendernode := net.Nodes[idx]
//			toidx := rand.Intn(len(net.Nodes)-1)
//			if toidx >= idx {
//				toidx++
//			}
//			recvnode := net.Nodes[toidx]
//			recvids[i] = recvnode.ID()
//			msg := PingMsg{Created: time.Now()}
//			code, _ := PingProtocol.GetCode(&PingMsg{})
//			pmsg, _ := NewProtocolMsg(code, msg)
//
//			client, err := sendernode.Client()
//			if err != nil {
//				return fmt.Errorf("error getting sendernode client: %s", err)
//			}
//			client.CallContext(ctx, &rpcerr, "pss_send", PingTopic, APIMsg{
//				Addr: recvaddrs[recvnode.ID()],
//				Msg:  pmsg,
//			})
//			if rpcerr != nil {
//				return fmt.Errorf("error rpc send id %x: %v", sendernode.ID(), rpcerr)
//			}
//		}
//		return nil
//	}
//	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
//		select {
//			case <-ctx.Done():
//				wg.Done()
//				return false, ctx.Err()
//			default:
//		}
//		msgreceived = append(msgreceived, id)
//		psslogmain.Info("trigger received", "id", id, "len", len(msgreceived))
//		wg.Done()
//		return true, nil
//	}
//
//	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
//		Action:  action,
//		Trigger: trigger,
//		Expect: &simulations.Expectation{
//			Nodes: recvids,
//			Check: check,
//		},
//	})
//	if result.Error != nil {
//		psslogmain.Error("msg failed!", "err", result.Error)
//		cancelmain()
//		t.Fatalf("simulation failed: %s", result.Error)
//	}
//
//	wg.Wait()
//
//	if len(msgreceived) != msgcount {
//		t.Fatalf("Simulation Failed, got %d of %d msgs", len(msgreceived), msgcount)
//	}
//
//	psslogmain.Info("done!")
//	t.Logf("Simulation Passed, got %d of %d msgs", len(msgreceived), msgcount)
//	//t.Logf("Duration: %s", result.FinishedAt.Sub(result.StartedAt))
//}
//
//// triggerChecks triggers a simulation step check whenever a peer is added or
//// removed from the given node
//// connections and connectionstarget are temporary kademlia check workarounds
//func triggerChecks(ctx context.Context, wg *sync.WaitGroup, trigger *chan discover.NodeID, net *simulations.Network, id discover.NodeID) error {
//
//	quitC := make(chan struct{})
//
//	node := net.GetNode(id)
//	if node == nil {
//		return fmt.Errorf("unknown node: %s", id)
//	}
//	client, err := node.Client()
//	if err != nil {
//		return err
//	}
//
//	peerevents := make(chan *p2p.PeerEvent)
//	peersub, err := client.Subscribe(context.Background(), "admin", peerevents, "peerEvents")
//	if err != nil {
//		return fmt.Errorf("error getting peer events for node %v: %s", id, err)
//	}
//
//	msgevents := make(chan APIMsg)
//	msgsub, err := client.Subscribe(context.Background(), "pss", msgevents, "receive", PingTopic)
//	if err != nil {
//		return fmt.Errorf("error getting msg events for node %v: %s", id, err)
//	}
//
//	go func() {
//		defer msgsub.Unsubscribe()
//		defer peersub.Unsubscribe()
//		for {
//			select {
//			case <-msgevents:
//				psslogmain.Debug("incoming msg", "node", id)
//				*trigger <- id
//			case err := <-peersub.Err():
//				if err != nil {
//					log.Error(fmt.Sprintf("error getting peer events for node %v", id), "err", err)
//				}
//				return
//
//			case err := <-msgsub.Err():
//				if err != nil {
//					log.Error(fmt.Sprintf("error getting msg for node %v", id), "err", err)
//				}
//				return
//			case <-quitC:
//				return
//			}
//		}
//	}()
//
//	go func() {
//		wg.Wait()
//		quitC <- struct{}{}
//	}()
//
//	return nil
//}
//
//func newServices() adapters.Services {
//	stateStore := adapters.NewSimStateStore()
//	kademlias := make(map[discover.NodeID]*network.Kademlia)
//	kademlia := func(id discover.NodeID) *network.Kademlia {
//		if k, ok := kademlias[id]; ok {
//			return k
//		}
//		addr := network.NewAddrFromNodeID(id)
//		params := network.NewKadParams()
//		params.MinProxBinSize = 2
//		params.MaxBinSize = 3
//		params.MinBinSize = 1
//		params.MaxRetries = 1000
//		params.RetryExponent = 2
//		params.RetryInterval = 1000000
//		kademlias[id] = network.NewKademlia(addr.Over(), params)
//		return kademlias[id]
//	}
//	return adapters.Services{
//		//"pss": func(id discover.NodeID, snapshot []byte) node.Service {
//		"pss": func(ctx *adapters.ServiceContext) (node.Service, error) {
//			cachedir, err := ioutil.TempDir("", "pss-cache")
//			if err != nil {
//				return nil, fmt.Errorf("create pss cache tmpdir failed", "error", err)
//			}
//			dpa, err := storage.NewLocalDPA(cachedir)
//			if err != nil {
//				return nil, fmt.Errorf("local dpa creation failed", "error", err)
//			}
//
//			pssp := NewPssParams(true)
//			ps := NewPss(kademlia(ctx.Config.ID), dpa, pssp)
//
//			ping := &Ping{
//				C: make(chan struct{}),
//			}
//			ps.Register(&PingTopic, RegisterPssProtocol(ps, &PingTopic, PingProtocol, NewPingProtocol(ping.PingHandler)).Handle)
//			if err != nil {
//				log.Error("Couldnt register pss protocol", "err", err)
//				os.Exit(1)
//			}
//
//			return ps, nil
//		},
//		//"bzz": func(id discover.NodeID, snapshot []byte) node.Service {
//		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
//			addr := network.NewAddrFromNodeID(ctx.Config.ID)
//			hp := network.NewHiveParams()
//			hp.Discovery = false
//			config := &network.BzzConfig{
//				OverlayAddr:  addr.Over(),
//				UnderlayAddr: addr.Under(),
//				HiveParams:   hp,
//			}
//			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore), nil
//		},
//	}
//}
//
//type connmap struct {
//	conns   map[discover.NodeID][]discover.NodeID
//	healthy map[discover.NodeID]bool
//	lock    sync.Mutex
//}
//
//type testOverlayConn struct {
//	*protocols.Peer
//	addr []byte
//}
//
//func (self *testOverlayConn) Address() []byte {
//	return self.addr
//}
//
//func (self *testOverlayConn) Off() network.OverlayAddr {
//	return self
//}
//
//func (self *testOverlayConn) Drop(err error) {
//}
//
//func (self *testOverlayConn) Update(o network.OverlayAddr) network.OverlayAddr {
//	return self
//}
