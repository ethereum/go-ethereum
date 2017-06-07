package pss

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/pot"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	pssServiceName = "pss"
	bzzServiceName = "bzz"
)

var services = newServices()

func init() {
	adapters.RegisterServices(services)
	hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
	hf := log.LvlFilterHandler(log.LvlTrace, hs)
	h := log.CallerFileHandler(hf)
	log.Root().SetHandler(h)
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

func TestSimpleLinear(t *testing.T) {
	var err error
	nodeconfig := adapters.RandomNodeConfig()
	addr := network.NewAddrFromNodeID(nodeconfig.ID)
	_ = p2ptest.NewTestPeerPool()
	ps := NewTestPss(addr.Over())

	ping := &Ping{
		C: make(chan struct{}),
	}

	err = RegisterPssProtocol(ps, &PingTopic, PingProtocol, NewPingProtocol(ping.PingHandler))

	if err != nil {
		t.Fatalf("Failed to register virtual protocol in pss: %v", err)
	}
	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		id := p.ID()
		pp := protocols.NewPeer(p, rw, pssSpec)
		bp := &testOverlayConn{
			Peer: pp,
			addr: network.ToOverlayAddr(id[:]),
		}
		h := pot.NewHashAddressFromBytes(bp.addr)
		ps.fwdPool[h.Address] = pp
		ps.Overlay.On(bp)
		defer ps.Overlay.Off(bp)
		log.Debug(fmt.Sprintf("%v", ps.Overlay))
		return bp.Run(ps.handlePssMsg)
	}

	pt := p2ptest.NewProtocolTester(t, nodeconfig.ID, 2, run)

	msg := NewPingMsg(network.ToOverlayAddr(pt.IDs[0].Bytes()), PingProtocol, PingTopic, []byte{1, 2, 3})

	exchange := p2ptest.Exchange{
		Expects: []p2ptest.Expect{
			p2ptest.Expect{
				Code: 0,
				Msg:  msg,
				Peer: pt.IDs[0],
			},
		},
		Triggers: []p2ptest.Trigger{
			p2ptest.Trigger{
				Code: 0,
				Msg:  msg,
				Peer: pt.IDs[1],
			},
		},
	}

	err = pt.TestExchanges(exchange)
	if err != nil {
		t.Fatalf("exchange failed %v", err)
	}
}

func TestFullRandom50n(t *testing.T) {
	adapter := adapters.NewSimAdapter(services)
	testFullRandom(t, adapter, 50, 50, 50)
}

func TestFullRandom25n(t *testing.T) {
	adapter := adapters.NewSimAdapter(services)
	testFullRandom(t, adapter, 25, 25, 25)
}

func TestFullRandom10n(t *testing.T) {
	adapter := adapters.NewSimAdapter(services)
	testFullRandom(t, adapter, 10, 10, 10)
}

func TestFullRandom5n(t *testing.T) {
	adapter := adapters.NewSimAdapter(services)
	testFullRandom(t, adapter, 5, 5, 5)
}

func testFullRandom(t *testing.T, adapter adapters.NodeAdapter, nodecount int, fullnodecount int, msgcount int) {
	var i int
	var msgfromids []discover.NodeID
	var msgreceived []discover.NodeID
	var cancelmain func()
	var triggerptr *chan discover.NodeID

	msgtoids := make([]discover.NodeID, msgcount)

	wg := sync.WaitGroup{}
	wg.Add(msgcount)

	psslog := make(map[discover.NodeID]log.Logger)
	psslogmain := log.New("psslog", "*")

	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID: "0",
	})
	defer net.Shutdown()

	timeout := 15 * time.Second
	ctx, cancelmain := context.WithTimeout(context.Background(), timeout)
	defer cancelmain()

	trigger := make(chan discover.NodeID)
	triggerptr = &trigger

	ids := make([]discover.NodeID, nodecount)
	fullids := ids[0:fullnodecount]
	fullpeers := make(map[discover.NodeID][]byte)

	for i = 0; i < nodecount; i++ {
		nodeconfig := adapters.RandomNodeConfig()
		nodeconfig.Services = []string{"bzz", "pss"}
		node, err := net.NewNodeWithConfig(nodeconfig)
		if err != nil {
			t.Fatalf("error starting node: %s", err)
		}

		if err := net.Start(node.ID()); err != nil {
			t.Fatalf("error starting node %s: %s", node.ID().TerminalString(), err)
		}

		if err := triggerChecks(ctx, &wg, triggerptr, net, node.ID()); err != nil {
			t.Fatal("error triggering checks for node %s: %s", node.ID().TerminalString(), err)
		}
		ids[i] = node.ID()
		if i < fullnodecount {
			fullpeers[ids[i]] = network.ToOverlayAddr(node.ID().Bytes())
			psslog[ids[i]] = log.New("psslog", fmt.Sprintf("%x", fullpeers[ids[i]]))
		}
		log.Debug("psslog starting node", "id", nodeconfig.ID)
	}

	for i, id := range fullids {
		msgfromids = append(msgfromids, id)
		msgtoids[i] = fullids[(i+(len(fullids)/2)+1)%len(fullids)]
	}

	// run a simulation which connects the 10 nodes in a ring and waits
	// for full peer discovery
	action := func(ctx context.Context) error {
		for i, id := range ids {
			peerID := ids[(i+1)%len(ids)]
			if net.GetConn(id, peerID) != nil {
				continue
			}
			if err := net.Connect(id, peerID); err != nil {
				return err
			}
			psslog[id].Debug("conn ok", "one", id, "other", peerID)
		}
		return nil
	}
	check := func(ctx context.Context, id discover.NodeID) (bool, error) {
		var tgt []byte
		var fwd struct {
			Addr  []byte
			Count int
		}
		select {
		case <-ctx.Done():
			wg.Done()
			psslog[id].Error("conn failed!", "id", id)
			return false, ctx.Err()
		default:
		}
		for i, fid := range msgfromids {
			if id == fid {
				tgt = network.ToOverlayAddr(msgtoids[(i+(len(msgtoids)/2)+1)%len(msgtoids)].Bytes())
				break
			}
		}
		p := net.GetNode(id)
		if p == nil {
			return false, fmt.Errorf("Unknown node: %v", id)
		}
		c, err := p.Client()
		if err != nil {
			return false, err
		}
		for fwd.Count < 2 {
			c.CallContext(context.Background(), &fwd, "pss_getForwarder", tgt)
			time.Sleep(time.Microsecond * 250)
		}
		psslog[id].Debug("fwd check ok", "topaddr", fmt.Sprintf("%x", common.ByteLabel(fwd.Addr)), "kadcount", fwd.Count)
		return true, nil
	}

	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
		cancelmain()
	}

	trigger = make(chan discover.NodeID)
	triggerptr = &trigger

	action = func(ctx context.Context) error {
		var rpcerr error
		for ii, id := range msgfromids {
			node := net.GetNode(id)
			if node == nil {
				return fmt.Errorf("unknown node: %s", id)
			}
			client, err := node.Client()
			if err != nil {
				return fmt.Errorf("error getting node client: %s", err)
			}
			msg := PingMsg{Created: time.Now()}
			code, _ := PingProtocol.GetCode(&PingMsg{})
			pmsg, _ := NewProtocolMsg(code, msg)
			client.CallContext(ctx, &rpcerr, "pss_send", PingTopic, APIMsg{
				Addr: fullpeers[msgtoids[ii]],
				Msg:  pmsg,
			})
			if rpcerr != nil {
				return fmt.Errorf("error rpc send id %x: %v", id, rpcerr)
			}
		}
		return nil
	}
	check = func(ctx context.Context, id discover.NodeID) (bool, error) {

		select {
		case <-ctx.Done():
			wg.Done()
			return false, ctx.Err()
		default:
		}
		msgreceived = append(msgreceived, id)
		psslog[id].Info("trigger received", "id", id, "len", len(msgreceived))
		wg.Done()
		return true, nil
	}

	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: msgtoids,
			Check: check,
		},
	})
	if result.Error != nil {
		psslogmain.Error("msg failed!", "err", result.Error)
		cancelmain()
		t.Fatalf("simulation failed: %s", result.Error)
	}

	if len(msgreceived) != len(msgtoids) {
		t.Fatalf("Simulation Failed, got %d of %d msgs", len(msgreceived), len(msgtoids))
	}

	wg.Wait()
	psslogmain.Info("done!")
	t.Logf("Simulation Passed, got %d of %d msgs", len(msgreceived), len(msgtoids))
	//t.Logf("Duration: %s", result.FinishedAt.Sub(result.StartedAt))
}

// triggerChecks triggers a simulation step check whenever a peer is added or
// removed from the given node
// connections and connectionstarget are temporary kademlia check workarounds
func triggerChecks(ctx context.Context, wg *sync.WaitGroup, trigger *chan discover.NodeID, net *simulations.Network, id discover.NodeID) error {

	quitC := make(chan struct{})
	got := false

	node := net.GetNode(id)
	if node == nil {
		return fmt.Errorf("unknown node: %s", id)
	}
	client, err := node.Client()
	if err != nil {
		return err
	}

	peerevents := make(chan *p2p.PeerEvent)
	peersub, err := client.Subscribe(context.Background(), "admin", peerevents, "peerEvents")
	if err != nil {
		return fmt.Errorf("error getting peer events for node %v: %s", id, err)
	}

	msgevents := make(chan APIMsg)
	msgsub, err := client.Subscribe(context.Background(), "pss", msgevents, "receive", PingTopic)
	if err != nil {
		return fmt.Errorf("error getting msg events for node %v: %s", id, err)
	}

	go func() {
		defer msgsub.Unsubscribe()
		defer peersub.Unsubscribe()
		for {
			select {
			case event := <-peerevents:
				if event.Type == "add" && !got {
					got = true
					*trigger <- id
				}
			case <-msgevents:
				*trigger <- id
			case err := <-peersub.Err():
				if err != nil {
					log.Error(fmt.Sprintf("error getting peer events for node %v", id), "err", err)
				}
				return

			case err := <-msgsub.Err():
				if err != nil {
					log.Error(fmt.Sprintf("error getting msg for node %v", id), "err", err)
				}
				return
			case <-quitC:
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		quitC <- struct{}{}
	}()

	return nil
}

func newServices() adapters.Services {
	stateStore := adapters.NewSimStateStore()
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
		//"pss": func(id discover.NodeID, snapshot []byte) node.Service {
		"pss": func(ctx *adapters.ServiceContext) (node.Service, error) {
			cachedir, err := ioutil.TempDir("", "pss-cache")
			if err != nil {
				return nil, fmt.Errorf("create pss cache tmpdir failed", "error", err)
			}
			dpa, err := storage.NewLocalDPA(cachedir)
			if err != nil {
				return nil, fmt.Errorf("local dpa creation failed", "error", err)
			}

			pssp := NewPssParams(true)
			ps := NewPss(kademlia(ctx.Config.ID), dpa, pssp)

			ping := &Ping{
				C: make(chan struct{}),
			}
			err = RegisterPssProtocol(ps, &PingTopic, PingProtocol, NewPingProtocol(ping.PingHandler))
			if err != nil {
				log.Error("Couldnt register pss protocol", "err", err)
				os.Exit(1)
			}

			return ps, nil
		},
		//"bzz": func(id discover.NodeID, snapshot []byte) node.Service {
		"bzz": func(ctx *adapters.ServiceContext) (node.Service, error) {
			addr := network.NewAddrFromNodeID(ctx.Config.ID)
			hp := network.NewHiveParams()
			hp.Discovery = true
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore), nil
		},
	}
}

type connmap struct {
	conns   map[discover.NodeID][]discover.NodeID
	healthy map[discover.NodeID]bool
	lock    sync.Mutex
}

type testOverlayConn struct {
	*protocols.Peer
	addr []byte
}

func (self *testOverlayConn) Address() []byte {
	return self.addr
}

func (self *testOverlayConn) Off() network.OverlayAddr {
	return self
}

func (self *testOverlayConn) Drop(err error) {
}

func (self *testOverlayConn) Update(o network.OverlayAddr) network.OverlayAddr {
	return self
}
