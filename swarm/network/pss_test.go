package network

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	protocolName    = "foo"
	protocolVersion = 42
)

func init() {
	h := log.CallerFileHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

// example protocol implementation peer
// message handlers are methods of this
// channels allow receipt reporting from p2p.Protocol message handler
type pssTestPeer struct {
	*protocols.Peer
	hasProtocol bool
	successC    chan bool
	resultC     chan int
}

// example node simulation peer
// modeled from swarm/network/simulations/discovery/discovery_test.go - commit 08b1e42f
// contains reporting channel for expect results so we can collect all async incoming msgs before deciding results
type pssTestNode struct {
	*Hive
	*Pss

	id      *adapters.NodeId
	network *simulations.Network
	trigger chan *adapters.NodeId
	run     adapters.RunProtocol
	ct      *protocols.CodeMap
	expectC chan []int
	ws      *http.Handler
	apifunc func() []rpc.API
}

func (n *pssTestNode) Add(peer Peer) error {
	err := n.Hive.Add(peer)
	time.Sleep(time.Millisecond * 250)
	n.triggerCheck()
	return err
}

func (n *pssTestNode) Remove(peer Peer) {
	n.Hive.Remove(peer)
}

func (n *pssTestNode) hiveKeepAlive() <-chan time.Time {
	return time.Tick(time.Second * 10)
}

func (n *pssTestNode) triggerCheck() {
	go func() { n.trigger <- n.id }()
}

func (n *pssTestNode) OverlayAddr() []byte {
	return n.Pss.Overlay.GetAddr().OverlayAddr()
}

func (n *pssTestNode) UnderlayAddr() []byte {
	return n.id.Bytes()
}

// the content of the msgs we're sending in the tests
type pssTestPayload struct {
	Data string
}

func (m *pssTestPayload) String() string {
	return m.Data
}

type pssTestService struct {
	node    *pssTestNode // get addrs from this
	msgFunc func(interface{}) error
}

func newPssTestService(t *testing.T, handlefunc func(interface{}) error, testnode *pssTestNode) *pssTestService {
	hp := NewHiveParams()
	//hp.CallInterval = 250
	testnode.Hive = NewHive(hp, testnode.Pss.Overlay)
	return &pssTestService{
		//nid := adapters.NewNodeId(addr.UnderlayAddr())
		msgFunc: handlefunc,
		node:    testnode,
	}
}

func (self *pssTestService) Start(server p2p.Server) error {
	return self.node.Hive.Start(server, self.node.hiveKeepAlive)
}

func (self *pssTestService) Stop() error {
	self.node.Hive.Stop()
	return nil
}

func (self *pssTestService) Protocols() []p2p.Protocol {
	ct := BzzCodeMap()
	for _, m := range DiscoveryMsgs {
		ct.Register(m)
	}
	ct.Register(&PssMsg{})

	srv := func(p Peer) error {
		p.Register(&PssMsg{}, self.msgFunc)
		self.node.Add(p)
		p.DisconnectHook(func(err error) {
			self.node.Remove(p)
		})
		return nil
	}

	proto := Bzz(self.node.OverlayAddr(), self.node.UnderlayAddr(), ct, srv, nil, nil)

	return []p2p.Protocol{*proto}
}

func (self *pssTestService) APIs() []rpc.API {
	return []rpc.API{
		rpc.API{
			Namespace: "eth",
			Version:   "0.1/pss",
			Service:   NewPssApi(self.node.Pss),
			Public:    true,
		},
	}
	return nil
}

func TestPssCache(t *testing.T) {
	var err error
	to, _ := hex.DecodeString("08090a0b0c0d0e0f1011121314150001020304050607161718191a1b1c1d1e1f")
	oaddr, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	uaddr, _ := hex.DecodeString("101112131415161718191a1b1c1d1e1f000102030405060708090a0b0c0d0e0f")
	ps := makePss(oaddr)
	pp := NewPssParams()
	topic, _ := MakeTopic(protocolName, protocolVersion)
	data := []byte("foo")
	fwdaddr := RandomAddr()
	msg := &PssMsg{
		Payload: pssEnvelope{
			TTL:         0,
			SenderOAddr: oaddr,
			SenderUAddr: uaddr,
			Topic:       topic,
			Payload:     data,
		},
	}
	msg.SetRecipient(to)

	msgtwo := &PssMsg{
		Payload: pssEnvelope{
			TTL:         0,
			SenderOAddr: uaddr,
			SenderUAddr: oaddr,
			Topic:       topic,
			Payload:     data,
		},
	}
	msgtwo.SetRecipient(to)

	digest := ps.hashMsg(msg)
	digesttwo := ps.hashMsg(msgtwo)

	if digest != 3595343914 {
		t.Fatalf("digest - got: %d, expected: %d", digest, 3595343914)
	}

	if digest == digesttwo {
		t.Fatalf("different msgs return same crc: %d", digesttwo)
	}

	// check the sender cache
	err = ps.addFwdCacheSender(fwdaddr.OverlayAddr(), digest)
	if err != nil {
		t.Fatalf("write to pss sender cache failed: %v", err)
	}

	if !ps.checkFwdCache(fwdaddr.OverlayAddr(), digest) {
		t.Fatalf("message %v should have SENDER record in cache but checkCache returned false", msg)
	}

	if ps.checkFwdCache(fwdaddr.OverlayAddr(), digesttwo) {
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

	err = ps.AddToCache(fwdaddr.OverlayAddr(), msgtwo)
	if err != nil {
		t.Fatalf("public accessor cache write failed: %v", err)
	}

	if !ps.checkFwdCache(fwdaddr.OverlayAddr(), digesttwo) {
		t.Fatalf("message %v should have SENDER record in cache but checkCache returned false", msgtwo)
	}
}

func TestPssRegisterHandler(t *testing.T) {
	var topic PssTopic
	var err error
	addr := RandomAddr()
	ps := makePss(addr.UnderlayAddr())

	topic, _ = MakeTopic(protocolName, protocolVersion)
	err = ps.Register(topic, func(msg []byte, p *p2p.Peer, sender []byte) error { return nil })
	if err != nil {
		t.Fatalf("couldnt register protocol 'foo' v 42: %v", err)
	}

	topic, _ = MakeTopic(protocolName, protocolVersion)
	err = ps.Register(topic, func(msg []byte, p *p2p.Peer, sender []byte) error { return nil })
	if err == nil {
		t.Fatalf("register protocol 'abc..789' v 65536 should have failed")
	}
}

func TestPssFullRandom10_10_5(t *testing.T) {
	testPssFullRandom(t, 10, 10, 5)
}

func TestPssFullRandom50_50_5(t *testing.T) {
	testPssFullRandom(t, 50, 50, 5)
}

func TestPssFullRandom50_50_25(t *testing.T) {
	testPssFullRandom(t, 50, 50, 25)
}

func TestPssFullRandom10_100_50(t *testing.T) {
	testPssFullRandom(t, 10, 100, 50)
}

func TestPssFullRandom50_100_50(t *testing.T) {
	testPssFullRandom(t, 50, 100, 50)
}

func TestPssFullRandom100_100_5(t *testing.T) {
	testPssFullRandom(t, 100, 100, 5)
}

func TestPssFullRandom100_100_25(t *testing.T) {
	testPssFullRandom(t, 100, 100, 25)
}

func TestPssFullRandom100_100_50(t *testing.T) {
	testPssFullRandom(t, 100, 100, 50)
}

func testPssFullRandom(t *testing.T, numsends int, numnodes int, numfullnodes int) {
	var action func(ctx context.Context) error
	var i int
	var check func(ctx context.Context, id *adapters.NodeId) (bool, error)
	var ctx context.Context
	var result *simulations.StepResult
	var timeout time.Duration
	var cancel context.CancelFunc

	fullnodes := []*adapters.NodeId{}
	sends := []int{}                                       // sender/receiver ids array indices pairs
	expectnodes := make(map[*adapters.NodeId]int)          // how many messages we're expecting on each respective node
	expectnodesids := []*adapters.NodeId{}                 // the nodes to expect on (needed by checker)
	expectnodesresults := make(map[*adapters.NodeId][]int) // which messages expect actually got

	vct := protocols.NewCodeMap(protocolName, protocolVersion, 65535, &pssTestPayload{})
	topic, _ := MakeTopic(protocolName, protocolVersion)

	trigger := make(chan *adapters.NodeId)
	testpeers := make(map[*adapters.NodeId]*pssTestPeer)
	net, nodes := newPssSimulationTester(t, numnodes, numfullnodes, trigger, vct, protocolName, protocolVersion, testpeers)

	ids := []*adapters.NodeId{}

	// connect the peers
	action = func(ctx context.Context) error {
		for id, _ := range nodes {
			ids = append(ids, id)
			if _, ok := testpeers[id]; ok {
				log.Trace(fmt.Sprintf("adding fullnode %x to testpeers %p", common.ByteLabel(id.Bytes()), testpeers))
				fullnodes = append(fullnodes, id)
			}
		}
		for i, id := range ids {
			var peerId *adapters.NodeId
			if i != 0 {
				peerId = ids[i-1]
				if err := net.Connect(id, peerId); err != nil {
					return err
				}
			}
		}
		return nil
	}
	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		node, ok := nodes[id]
		if !ok {
			return false, fmt.Errorf("unknown node: %s (%v)", id, node)
		} else {
			log.Trace(fmt.Sprintf("sim check ok node %v", id))
		}

		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)

	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}
	cancel()

	// ensure that we didn't get lost in concurrency issues
	if len(fullnodes) != numfullnodes {
		t.Fatalf("corrupt fullnodes array, expected %d, have %d", numfullnodes, len(fullnodes))
	}

	// ensure that the channel is clean
	trigger = make(chan *adapters.NodeId)

	// randomly decide which nodes to send to and from
	rand.Seed(time.Now().Unix())
	for i = 0; i < numsends; i++ {
		s := rand.Int() % numfullnodes
		r := s
		for r == s {
			r = rand.Int() % numfullnodes
		}
		log.Trace(fmt.Sprintf("rnd pss: idx %d->%d (%x -> %x)", s, r, common.ByteLabel(fullnodes[s].Bytes()), common.ByteLabel(fullnodes[r].Bytes())))
		expectnodes[fullnodes[r]]++
		sends = append(sends, s, r)
	}

	// distinct array of nodes to expect on
	for k, _ := range expectnodes {
		expectnodesids = append(expectnodesids, k)
	}

	// wait a bit for the kademlias to fill up
	z, _ := time.ParseDuration(fmt.Sprintf("%dms", (numnodes * 25)))
	if z.Seconds() < 1.0 {
		z = time.Second
	}
	time.Sleep(z)

	// send and monitor receive of pss
	action = func(ctx context.Context) error {
		code, _ := vct.GetCode(&pssTestPayload{})

		for i := 0; i < len(sends); i += 2 {
			msgbytes, _ := makeMsg(code, &pssTestPayload{
				Data: fmt.Sprintf("%v", i+1),
			})
			go func(i int, expectnodesresults map[*adapters.NodeId][]int) {
				expectnode := fullnodes[sends[i+1]] // the receiving node
				sendnode := fullnodes[sends[i]]     // the sending node
				oaddr := nodes[expectnode].OverlayAddr()
				err := nodes[sendnode].Pss.Send(oaddr, topic, msgbytes)
				if err != nil {
					t.Fatalf("could not send pss: %v", err)
				}

				select {
				// if the pss is delivered
				case <-testpeers[expectnode].successC:
					log.Trace(fmt.Sprintf("got successC from node %x", common.ByteLabel(expectnode.Bytes())))
					expectnodesresults[expectnode] = append(expectnodesresults[expectnode], <-testpeers[expectnode].resultC)
				// if not we time out, -1 means fail tick
				case <-time.NewTimer(time.Second).C:
					log.Trace(fmt.Sprintf("result timed out on node %x", common.ByteLabel(expectnode.Bytes())))
					expectnodesresults[expectnode] = append(expectnodesresults[expectnode], -1)
				}

				// we can safely send to the check handler if we got feedback for all msgs we sent to a particular node
				if len(expectnodesresults[expectnode]) == expectnodes[expectnode] {
					trigger <- expectnode
					nodes[expectnode].expectC <- expectnodesresults[expectnode]
				}
			}(i, expectnodesresults)
		}
		return nil
	}

	// results
	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		receives := <-nodes[id].expectC
		log.Trace(fmt.Sprintf("expect received %d msgs on from node %x: %v", len(receives), common.ByteLabel(id.Bytes()), receives))
		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: expectnodesids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}

	t.Log("Simulation Passed:")

	for i := 0; i < len(sends); i += 2 {
		t.Logf("Pss #%d: oaddr %x -> %x (uaddr %x -> %x)", i/2+1,
			common.ByteLabel(nodes[fullnodes[sends[i]]].Pss.GetAddr().OverlayAddr()),
			common.ByteLabel(nodes[fullnodes[sends[i+1]]].Pss.GetAddr().OverlayAddr()),
			common.ByteLabel(fullnodes[sends[i]].Bytes()),
			common.ByteLabel(fullnodes[sends[i+1]].Bytes()))
	}
	totalfails := 0
	for id, results := range expectnodesresults {
		fails := 0
		for _, r := range results {
			if r == -1 {
				fails++
			}
		}
		t.Logf("Node oaddr %x (uaddr %x) was sent %d msgs, of which %d failed", common.ByteLabel(nodes[id].Pss.GetAddr().OverlayAddr()), common.ByteLabel(id.Bytes()), len(results), fails)
		totalfails += fails
	}
	t.Logf("Total sent: %d, total fail: %d (%.2f%%)", len(sends)/2, totalfails, (float32(totalfails)/float32(len(sends)/2+1))*100)

	for _, node := range nodes {
		logstring := fmt.Sprintf("Node oaddr %x kademlia: ", common.ByteLabel(node.Pss.Overlay.GetAddr().OverlayAddr()))
		node.Pss.Overlay.EachLivePeer(nil, 256, func(p Peer, po int, isprox bool) bool {
			logstring += fmt.Sprintf("%x ", common.ByteLabel(p.OverlayAddr()))
			return true
		})
		t.Log(logstring)
	}
}

func TestPssFullLinearEcho(t *testing.T) {

	var action func(ctx context.Context) error
	var check func(ctx context.Context, id *adapters.NodeId) (bool, error)
	var ctx context.Context
	var result *simulations.StepResult
	var timeout time.Duration
	var cancel context.CancelFunc

	var firstpssnode *adapters.NodeId
	var secondpssnode *adapters.NodeId

	vct := protocols.NewCodeMap(protocolName, protocolVersion, 65535, &pssTestPayload{})
	topic, _ := MakeTopic(protocolName, protocolVersion)

	fullnodes := []*adapters.NodeId{}
	trigger := make(chan *adapters.NodeId)
	testpeers := make(map[*adapters.NodeId]*pssTestPeer)
	net, nodes := newPssSimulationTester(t, 3, 2, trigger, vct, protocolName, protocolVersion, testpeers)
	ids := []*adapters.NodeId{} // ohh risky! but the action for a specific id should come before the expect anyway

	action = func(ctx context.Context) error {
		var thinnodeid *adapters.NodeId
		for id, _ := range nodes {
			ids = append(ids, id)
			if _, ok := testpeers[id]; ok {
				log.Trace(fmt.Sprintf("adding fullnode %x to testpeers %p", common.ByteLabel(id.Bytes()), testpeers))
				fullnodes = append(fullnodes, id)
			} else {
				thinnodeid = id
			}
		}
		if err := net.Connect(fullnodes[0], thinnodeid); err != nil {
			return err
		}
		if err := net.Connect(thinnodeid, fullnodes[1]); err != nil {
			return err
		}

		/*for i, id := range ids {
			var peerId *adapters.NodeId
			if i != 0 {
				peerId = ids[i-1]
				if err := net.Connect(id, peerId); err != nil {
					return err
				}
			}
		}*/
		return nil
	}
	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		node, ok := nodes[id]
		if !ok {
			return false, fmt.Errorf("unknown node: %s (%v)", id, node)
		} else {
			log.Trace(fmt.Sprintf("sim check ok node %v", id))
		}

		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)

	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}
	cancel()

	nonode := &adapters.NodeId{}
	firstpssnode = nonode
	secondpssnode = nonode

	// first find a node that we're connected to
	for firstpssnode == nonode {
		log.Debug(fmt.Sprintf("Waiting for pss relaypeer for %x close to %x ...", common.ByteLabel(nodes[fullnodes[0]].OverlayAddr()), common.ByteLabel(nodes[ids[1]].OverlayAddr())))
		nodes[fullnodes[0]].Pss.Overlay.EachLivePeer(nodes[fullnodes[1]].OverlayAddr(), 256, func(p Peer, po int, isprox bool) bool {
			for _, id := range ids {
				if id.NodeID == p.ID() {
					firstpssnode = id
					log.Debug(fmt.Sprintf("PSS relay found; relaynode %v kademlia %v", common.ByteLabel(id.Bytes()), common.ByteLabel(firstpssnode.Bytes())))
				}
			}
			if firstpssnode == nonode {
				return true
			}
			return false
		})
		if firstpssnode == nonode {
			time.Sleep(time.Millisecond * 100)
		}
	}

	// then find the node it's connected to
	for secondpssnode == nonode {
		log.Debug(fmt.Sprintf("PSS kademlia: Waiting for recipientpeer for %x close to %x ...", common.ByteLabel(nodes[firstpssnode].OverlayAddr()), common.ByteLabel(nodes[fullnodes[1]].OverlayAddr())))
		nodes[firstpssnode].Pss.Overlay.EachLivePeer(nodes[fullnodes[1]].OverlayAddr(), 256, func(p Peer, po int, isprox bool) bool {
			for _, id := range ids {
				if id.NodeID == p.ID() && id.NodeID != fullnodes[0].NodeID {
					secondpssnode = id
					log.Debug(fmt.Sprintf("PSS recipient found; relaynode %v kademlia %v", common.ByteLabel(id.Bytes()), common.ByteLabel(secondpssnode.Bytes())))
				}
			}
			if secondpssnode == nonode {
				return true
			}
			return false
		})
		if secondpssnode == nonode {
			time.Sleep(time.Millisecond * 100)
		}
	}

	action = func(ctx context.Context) error {
		code, _ := vct.GetCode(&pssTestPayload{})
		msgbytes, _ := makeMsg(code, &pssTestPayload{
			Data: "ping",
		})

		go func() {
			oaddr := nodes[secondpssnode].OverlayAddr()
			err := nodes[ids[0]].Pss.Send(oaddr, topic, msgbytes)
			if err != nil {
				t.Fatalf("could not send pss: %v", err)
			}
			trigger <- ids[0]
		}()

		return nil
	}
	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		// also need to know if the protocolpeer is set up
		time.Sleep(time.Millisecond * 100)
		return <-testpeers[ids[0]].successC, nil
		//return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: []*adapters.NodeId{ids[0]},
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}

	t.Log("Simulation Passed:")
}

func TestPssFullWS(t *testing.T) {

	// settings for ws servers
	var srvsendep = "localhost:18546"
	var srvrecvep = "localhost:18547"
	var clientrecvok, clientsendok bool
	var clientrecv, clientsend *rpc.Client

	var action func(ctx context.Context) error
	var check func(ctx context.Context, id *adapters.NodeId) (bool, error)
	var ctx context.Context
	var result *simulations.StepResult
	var timeout time.Duration
	var cancel context.CancelFunc

	var firstpssnode, secondpssnode *adapters.NodeId
	fullnodes := []*adapters.NodeId{}
	vct := protocols.NewCodeMap(protocolName, protocolVersion, 65535, &pssTestPayload{})
	topic, _ := MakeTopic(pingTopicName, pingTopicVersion)

	trigger := make(chan *adapters.NodeId)
	testpeers := make(map[*adapters.NodeId]*pssTestPeer)
	simnet, nodes := newPssSimulationTester(t, 3, 2, trigger, vct, protocolName, protocolVersion, testpeers)
	ids := []*adapters.NodeId{} // ohh risky! but the action for a specific id should come before the expect anyway

	action = func(ctx context.Context) error {
		var thinnodeid *adapters.NodeId
		for id, node := range nodes {
			ids = append(ids, id)
			if _, ok := testpeers[id]; ok {
				log.Trace(fmt.Sprintf("adding fullnode %x to testpeers %p", common.ByteLabel(id.Bytes()), testpeers))
				fullnodes = append(fullnodes, id)
				node.Pss.Register(topic, node.Pss.GetPingHandler())
				srv := rpc.NewServer()
				for _, rpcapi := range node.apifunc() {
					srv.RegisterName(rpcapi.Namespace, rpcapi.Service)
				}
				ws := srv.WebsocketHandler([]string{"*"})
				node.ws = &ws
			} else {
				thinnodeid = id
			}
		}
		if err := simnet.Connect(fullnodes[0], thinnodeid); err != nil {
			return err
		}
		if err := simnet.Connect(thinnodeid, fullnodes[1]); err != nil {
			return err
		}

		return nil
	}

	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		node, ok := nodes[id]
		if !ok {
			return false, fmt.Errorf("unknown node: %s (%v)", id, node)
		} else {
			log.Trace(fmt.Sprintf("sim check ok node %v", id))
		}

		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)

	result = simulations.NewSimulation(simnet).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}
	cancel()

	nonode := &adapters.NodeId{}
	firstpssnode = nonode
	secondpssnode = nonode

	// first find a node that we're connected to
	for firstpssnode == nonode {
		log.Debug(fmt.Sprintf("Waiting for pss relaypeer for %x close to %x ...", common.ByteLabel(nodes[fullnodes[0]].OverlayAddr()), common.ByteLabel(nodes[fullnodes[1]].OverlayAddr())))
		nodes[fullnodes[0]].Pss.Overlay.EachLivePeer(nodes[fullnodes[1]].OverlayAddr(), 256, func(p Peer, po int, isprox bool) bool {
			for _, id := range ids {
				if id.NodeID == p.ID() {
					firstpssnode = id
					log.Debug(fmt.Sprintf("PSS relay found; relaynode %x", common.ByteLabel(nodes[firstpssnode].OverlayAddr())))
				}
			}
			if firstpssnode == nonode {
				return true
			}
			return false
		})
		if firstpssnode == nonode {
			time.Sleep(time.Millisecond * 100)
		}
	}

	// then find the node it's connected to
	for secondpssnode == nonode {
		log.Debug(fmt.Sprintf("PSS kademlia: Waiting for recipientpeer for %x close to %x ...", common.ByteLabel(nodes[firstpssnode].OverlayAddr()), common.ByteLabel(nodes[fullnodes[1]].OverlayAddr())))
		nodes[firstpssnode].Pss.Overlay.EachLivePeer(nodes[fullnodes[1]].OverlayAddr(), 256, func(p Peer, po int, isprox bool) bool {
			for _, id := range ids {
				if id.NodeID == p.ID() && id.NodeID != fullnodes[0].NodeID {
					secondpssnode = id
					log.Debug(fmt.Sprintf("PSS recipient found; relaynode %x", common.ByteLabel(nodes[secondpssnode].OverlayAddr())))
				}
			}
			if secondpssnode == nonode {
				return true
			}
			return false
		})
		if secondpssnode == nonode {
			time.Sleep(time.Millisecond * 100)
		}
	}

	srvrecvl, err := net.Listen("tcp", srvrecvep)
	if err != nil {
		t.Fatalf("Tcp (recv) on %s failed: %v", srvrecvep, err)
	}
	go func() {
		err := http.Serve(srvrecvl, *nodes[fullnodes[1]].ws)
		if err != nil {
			t.Fatalf("http serve (recv) on %s failed: %v", srvrecvep, err)
		}
	}()

	srvsendl, err := net.Listen("tcp", srvsendep)
	if err != nil {
		t.Fatalf("Tcp (send) on %s failed: %v", srvsendep, err)
	}
	go func() {
		err := http.Serve(srvsendl, *nodes[fullnodes[0]].ws)
		if err != nil {
			t.Fatalf("http serve (send) on %s failed: %v", srvrecvep, err)
		}
	}()

	for !clientrecvok {
		log.Trace("attempting clientrecv connect")
		clientrecv, err = rpc.DialWebsocket(context.Background(), "ws://"+srvrecvep, "ws://localhost")
		if err == nil {
			clientrecvok = true
		} else {
			log.Debug("clientrecv failed, retrying", "error", err)
			time.Sleep(time.Millisecond * 250)
		}
	}

	for !clientsendok {
		log.Trace("attempting clientsend connect")
		clientsend, err = rpc.DialWebsocket(context.Background(), "ws://"+srvsendep, "ws://localhost")
		if err == nil {
			clientsendok = true
		} else {
			log.Debug("clientsend failed, retrying", "error", err)
			time.Sleep(time.Millisecond * 250)
		}
	}

	trigger = make(chan *adapters.NodeId)
	ch := make(chan string)

	action = func(ctx context.Context) error {
		go func() {
			clientrecv.EthSubscribe(ctx, ch, "newMsg", topic)
			clientsend.Call(nil, "eth_sendRaw", nodes[secondpssnode].Pss.Overlay.GetAddr().OverlayAddr(), topic, []byte("ping"))
			trigger <- secondpssnode
		}()
		return nil
	}
	check = func(ctx context.Context, id *adapters.NodeId) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		select {
		case msg := <-ch:
			log.Trace(fmt.Sprintf("notify!: %v", msg))
		case <-time.NewTimer(time.Second).C:
			log.Trace(fmt.Sprintf("no notifies :'("))
		}
		// also need to know if the protocolpeer is set up

		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result = simulations.NewSimulation(simnet).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: []*adapters.NodeId{secondpssnode},
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatalf("simulation failed: %s", result.Error)
	}

	t.Log("Simulation Passed:")
}

// test framework below

// numnodes: how many nodes to create
// pssnodeidx: on which node indices to start the pss
// net: the simulated network
// trigger: hook needed for simulation event reporting
// vct: codemap for virtual protocol
// name: name for virtual protocol (and pss topic)
// version: name for virtual protocol (and pss topic)
// testpeers: pss-specific peers, with hook needed for simulation event reporting

// the simulation tester constructor is currently a hack to fit previous code with later stack using node.Services to start SimNodes

func newPssSimulationTester(t *testing.T, numnodes int, numfullnodes int, trigger chan *adapters.NodeId, vct *protocols.CodeMap, name string, version int, testpeers map[*adapters.NodeId]*pssTestPeer) (*simulations.Network, map[*adapters.NodeId]*pssTestNode) {
	topic, _ := MakeTopic(name, version)
	nodes := make(map[*adapters.NodeId]*pssTestNode, numnodes)
	psss := make(map[*adapters.NodeId]*Pss)
	var simnet *simulations.Network
	serviceFunc := func(id *adapters.NodeId) node.Service {
		node := &pssTestNode{
			Pss:     psss[id],
			Hive:    nil,
			id:      id,
			network: simnet,
			trigger: trigger,
			ct:      vct,
			apifunc: func() []rpc.API { return nil },
			expectC: make(chan []int),
		}

		// set up handlers for the encapsulating PssMsg

		var handlefunc func(interface{}) error

		addr := NewPeerAddrFromNodeId(id)

		if testpeers[id] != nil {
			handlefunc = makePssHandleProtocol(psss[id])
			log.Trace(fmt.Sprintf("Making full protocol id %x addr %x (testpeers %p)", common.ByteLabel(id.Bytes()), common.ByteLabel(addr.OverlayAddr()), testpeers))
		} else {
			handlefunc = makePssHandleForward(psss[id])
		}

		// protocols are now registered by invoking node services
		// since adapters.SimNode implements p2p.Server, needed for the services to start, we use this as a convenience wrapper

		testservice := newPssTestService(t, handlefunc, node)

		// the network sim wants a adapters.NodeAdapter, so we pass back to it a SimNode
		// this is the SimNode member of the testNode initialized above, but assigned through the service start
		// that is so say: node == testservice.node, but we access it as a member of testservice below for clarity (to the extent that this can be clear)

		nodes[id] = testservice.node
		testservice.node.apifunc = testservice.APIs
		return testservice
	}
	adapter := adapters.NewSimAdapter(map[string]adapters.ServiceFunc{"pss": serviceFunc})
	simnet = simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		Id:      "0",
		Backend: true,
	})
	configs := make([]*adapters.NodeConfig, numnodes)
	for i := 0; i < numnodes; i++ {
		configs[i] = adapters.RandomNodeConfig()
		configs[i].Service = "pss"
	}
	for i, conf := range configs {
		addr := NewPeerAddrFromNodeId(conf.Id)
		psss[conf.Id] = makePss(addr.OverlayAddr())
		if i < numfullnodes {
			tp := &pssTestPeer{
				Peer: &protocols.Peer{
					Peer: &p2p.Peer{},
				},
				successC: make(chan bool),
				resultC:  make(chan int),
			}
			testpeers[conf.Id] = tp
			targetprotocol := makeCustomProtocol(name, version, vct, testpeers[conf.Id])
			pssprotocol := NewPssProtocol(psss[conf.Id], &topic, vct, targetprotocol)
			psss[conf.Id].Register(topic, pssprotocol.GetHandler())
		}

		if err := simnet.NewNodeWithConfig(conf); err != nil {
			t.Fatalf("error creating node %s: %s", conf.Id.Label(), err)
		}
		if err := simnet.Start(conf.Id); err != nil {
			t.Fatalf("error starting node %s: %s", conf.Id.Label(), err)
		}
	}

	return simnet, nodes
}

func makePss(addr []byte) *Pss {
	kp := NewKadParams()
	kp.MinProxBinSize = 3

	pp := NewPssParams()

	overlay := NewKademlia(addr, kp)
	ps := NewPss(overlay, pp)
	//overlay.Prune(time.Tick(time.Millisecond * 250))
	return ps
}

func makeCustomProtocol(name string, version int, ct *protocols.CodeMap, testpeer *pssTestPeer) *p2p.Protocol {
	run := func(p *protocols.Peer) error {
		log.Trace(fmt.Sprintf("running pss vprotocol on peer %v", p))
		if testpeer == nil {
			testpeer = &pssTestPeer{}
		}
		testpeer.Peer = p
		p.Register(&pssTestPayload{}, testpeer.SimpleHandlePssPayload)
		err := p.Run()
		return err
	}

	return protocols.NewProtocol(name, uint(version), run, ct, nil, nil)
}

func makeFakeMsg(ps *Pss, ct *protocols.CodeMap, topic PssTopic, senderaddr PeerAddr, content string) PssMsg {
	data := pssTestPayload{}
	code, found := ct.GetCode(&data)
	if !found {
		return PssMsg{}
	}

	data.Data = content

	rlpbundle, err := makeMsg(code, data)
	if err != nil {
		return PssMsg{}
	}

	pssenv := pssEnvelope{
		SenderOAddr: senderaddr.OverlayAddr(),
		SenderUAddr: senderaddr.UnderlayAddr(),
		Topic:       topic,
		TTL:         DefaultTTL,
		Payload:     rlpbundle,
	}
	pssmsg := PssMsg{
		Payload: pssenv,
	}
	pssmsg.SetRecipient(ps.Overlay.GetAddr().OverlayAddr())

	return pssmsg
}

func makePssHandleForward(ps *Pss) func(msg interface{}) error {
	// for the simple check it passes on the message if it's not for us
	return func(msg interface{}) error {
		pssmsg := msg.(*PssMsg)
		if ps.IsSelfRecipient(pssmsg) {
			log.Trace("pss for us .. yay!")
		} else {
			log.Trace("passing on pss")
			return ps.Forward(pssmsg)
		}
		return nil
	}
}

func makePssHandleProtocol(ps *Pss) func(msg interface{}) error {
	return func(msg interface{}) error {
		pssmsg := msg.(*PssMsg)

		if ps.IsSelfRecipient(pssmsg) {
			log.Trace("pss for us ... let's process!")
			env := pssmsg.Payload
			umsg := env.Payload // this will be rlp encrypted
			f := ps.GetHandler(env.Topic)
			if f == nil {
				return fmt.Errorf("No registered handler for topic '%s'", env.Topic)
			}
			nid := adapters.NewNodeId(env.SenderUAddr)
			p := p2p.NewPeer(nid.NodeID, fmt.Sprintf("%x", common.ByteLabel(nid.Bytes())), []p2p.Cap{})
			return f(umsg, p, env.SenderOAddr)
		} else {
			log.Trace("pss was for someone else :'(")
			return ps.Forward(pssmsg)
		}
		return nil
	}
}

// echoes an incoming message
// it comes in through
// Any pointer receiver that has protocols.Peer
func (ptp *pssTestPeer) SimpleHandlePssPayload(msg interface{}) error {
	pmsg := msg.(*pssTestPayload)
	log.Trace(fmt.Sprintf("pssTestPayloadhandler got message %v", pmsg))
	if pmsg.Data == "ping" {
		pmsg.Data = "pong"
		log.Trace(fmt.Sprintf("pssTestPayloadhandler reply %v", pmsg))
		ptp.Send(pmsg)
	} else if pmsg.Data == "pong" {
		ptp.successC <- true
	} else {
		res, err := strconv.Atoi(pmsg.Data)
		if err != nil {
			log.Trace(fmt.Sprintf("pssTestPayloadhandlererr %v", err))
			ptp.successC <- false
		} else {
			log.Trace(fmt.Sprintf("pssTestPayloadhandler sending %d on chan", pmsg))
			ptp.successC <- true
			ptp.resultC <- res
		}
	}

	return nil
}
