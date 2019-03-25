package pss

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
)

// needed to make the enode id of the receiving node available to the handler for triggers
type handlerContextFunc func(*testData, *adapters.NodeConfig) *handler

// struct to notify reception of messages to simulation driver
// TODO To make code cleaner:
// - consider a separate pss unwrap to message event in sim framework (this will make eventual message propagation analysis with pss easier/possible in the future)
// - consider also test api calls to inspect handling results of messages
type handlerNotification struct {
	id     enode.ID
	serial uint64
}

type testData struct {
	mu               sync.Mutex
	sim              *simulation.Simulation
	handlerDone      bool // set to true on termination of the simulation run
	requiredMessages int
	allowedMessages  int
	messageCount     int
	kademlias        map[enode.ID]*network.Kademlia
	nodeAddrs        map[enode.ID][]byte      // make predictable overlay addresses from the generated random enode ids
	recipients       map[int][]enode.ID       // for logging output only
	allowed          map[int][]enode.ID       // allowed recipients
	expectedMsgs     map[enode.ID][]uint64    // message serials we expect respective nodes to receive
	allowedMsgs      map[enode.ID][]uint64    // message serials we expect respective nodes to receive
	senders          map[int]enode.ID         // originating nodes of the messages (intention is to choose as far as possible from the receiving neighborhood)
	handlerC         chan handlerNotification // passes message from pss message handler to simulation driver
	doneC            chan struct{}            // terminates the handler channel listener
	errC             chan error               // error to pass to main sim thread
	msgC             chan handlerNotification // message receipt notification to main sim thread
	msgs             [][]byte                 // recipient addresses of messages
}

var (
	pof   = pot.DefaultPof(256) // generate messages and index them
	topic = BytesToTopic([]byte{0xf3, 0x9e, 0x06, 0x82})
)

func (d *testData) getMsgCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.messageCount
}

func (d *testData) incrementMsgCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messageCount++
	return d.messageCount
}

func (d *testData) isDone() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.handlerDone
}

func (d *testData) setDone() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlerDone = true
}

func getCmdParams(t *testing.T) (int, int, time.Duration) {
	args := strings.Split(t.Name(), "/")
	msgCount, err := strconv.ParseInt(args[2], 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	nodeCount, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	timeoutStr := fmt.Sprintf("%ss", args[3])
	timeoutDur, err := time.ParseDuration(timeoutStr)
	if err != nil {
		t.Fatal(err)
	}
	return int(msgCount), int(nodeCount), timeoutDur
}

func newTestData() *testData {
	return &testData{
		kademlias:    make(map[enode.ID]*network.Kademlia),
		nodeAddrs:    make(map[enode.ID][]byte),
		recipients:   make(map[int][]enode.ID),
		allowed:      make(map[int][]enode.ID),
		expectedMsgs: make(map[enode.ID][]uint64),
		allowedMsgs:  make(map[enode.ID][]uint64),
		senders:      make(map[int]enode.ID),
		handlerC:     make(chan handlerNotification),
		doneC:        make(chan struct{}),
		errC:         make(chan error),
		msgC:         make(chan handlerNotification),
	}
}

func (d *testData) getKademlia(nodeId *enode.ID) (*network.Kademlia, error) {
	kadif, ok := d.sim.NodeItem(*nodeId, simulation.BucketKeyKademlia)
	if !ok {
		return nil, fmt.Errorf("no kademlia entry for %v", nodeId)
	}
	kad, ok := kadif.(*network.Kademlia)
	if !ok {
		return nil, fmt.Errorf("invalid kademlia entry for %v", nodeId)
	}
	return kad, nil
}

func (d *testData) init(msgCount int) error {
	log.Debug("TestProxNetwork start")

	for _, nodeId := range d.sim.NodeIDs() {
		kad, err := d.getKademlia(&nodeId)
		if err != nil {
			return err
		}
		d.nodeAddrs[nodeId] = kad.BaseAddr()
	}

	for i := 0; i < int(msgCount); i++ {
		msgAddr := pot.RandomAddress() // we choose message addresses randomly
		d.msgs = append(d.msgs, msgAddr.Bytes())
		smallestPo := 256
		var targets []enode.ID
		var closestPO int

		// loop through all nodes and find the required and allowed recipients of each message
		// (for more information, please see the comment to the main test function)
		for _, nod := range d.sim.Net.GetNodes() {
			po, _ := pof(d.msgs[i], d.nodeAddrs[nod.ID()], 0)
			depth := d.kademlias[nod.ID()].NeighbourhoodDepth()

			// only nodes with closest IDs (wrt the msg address) will be required recipients
			if po > closestPO {
				closestPO = po
				targets = nil
				targets = append(targets, nod.ID())
			} else if po == closestPO {
				targets = append(targets, nod.ID())
			}

			if po >= depth {
				d.allowedMessages++
				d.allowed[i] = append(d.allowed[i], nod.ID())
				d.allowedMsgs[nod.ID()] = append(d.allowedMsgs[nod.ID()], uint64(i))
			}

			// a node with the smallest PO (wrt msg) will be the sender,
			// in order to increase the distance the msg must travel
			if po < smallestPo {
				smallestPo = po
				d.senders[i] = nod.ID()
			}
		}

		d.requiredMessages += len(targets)
		for _, id := range targets {
			d.recipients[i] = append(d.recipients[i], id)
			d.expectedMsgs[id] = append(d.expectedMsgs[id], uint64(i))
		}

		log.Debug("nn for msg", "targets", len(d.recipients[i]), "msgidx", i, "msg", common.Bytes2Hex(msgAddr[:8]), "sender", d.senders[i], "senderpo", smallestPo)
	}
	log.Debug("msgs to receive", "count", d.requiredMessages)
	return nil
}

// Here we test specific functionality of the pss, setting the prox property of
// the handler. The tests generate a number of messages with random addresses.
// Then, for each message it calculates which nodes have the msg address
// within its nearest neighborhood depth, and stores those nodes as possible
// recipients. Those nodes that are the closest to the message address (nodes
// belonging to the deepest PO wrt the msg address) are stored as required
// recipients. The difference between allowed and required recipients results
// from the fact that the nearest neighbours are not necessarily reciprocal.
// Upon sending the messages, the test verifies that the respective message is
// passed to the message handlers of these required recipients. The test fails
// if a message is handled by recipient which is not listed among the allowed
// recipients of this particular message. It also fails after timeout, if not
// all the required recipients have received their respective messages.
//
// For example, if proximity order of certain msg address is 4, and node X
// has PO=5 wrt the message address, and nodes Y and Z have PO=6, then:
// nodes Y and Z will be considered required recipients of the msg,
// whereas nodes X, Y and Z will be allowed recipients.
func TestProxNetwork(t *testing.T) {
	t.Run("16/16/15", testProxNetwork)
}

// params in run name: nodes/msgs
func TestProxNetworkLong(t *testing.T) {
	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("8/100/30", testProxNetwork)
	t.Run("16/100/30", testProxNetwork)
	t.Run("32/100/60", testProxNetwork)
	t.Run("64/100/60", testProxNetwork)
	t.Run("128/100/120", testProxNetwork)
}

func testProxNetwork(t *testing.T) {
	tstdata := newTestData()
	msgCount, nodeCount, timeout := getCmdParams(t)
	handlerContextFuncs := make(map[Topic]handlerContextFunc)
	handlerContextFuncs[topic] = nodeMsgHandler
	services := newProxServices(tstdata, true, handlerContextFuncs, tstdata.kademlias)
	tstdata.sim = simulation.New(services)
	defer tstdata.sim.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	filename := fmt.Sprintf("testdata/snapshot_%d.json", nodeCount)
	err := tstdata.sim.UploadSnapshot(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	err = tstdata.init(msgCount) // initialize the test data
	if err != nil {
		t.Fatal(err)
	}
	wrapper := func(c context.Context, _ *simulation.Simulation) error {
		return testRoutine(tstdata, c)
	}
	result := tstdata.sim.Run(ctx, wrapper) // call the main test function
	if result.Error != nil {
		// context deadline exceeded
		// however, it might just mean that not all possible messages are received
		// now we must check if all required messages are received
		cnt := tstdata.getMsgCount()
		log.Debug("TestProxNetwork finished", "rcv", cnt)
		if cnt < tstdata.requiredMessages {
			t.Fatal(result.Error)
		}
	}
	t.Logf("completed %d", result.Duration)
}

func (tstdata *testData) sendAllMsgs() {
	for i, msg := range tstdata.msgs {
		log.Debug("sending msg", "idx", i, "from", tstdata.senders[i])
		nodeClient, err := tstdata.sim.Net.GetNode(tstdata.senders[i]).Client()
		if err != nil {
			tstdata.errC <- err
		}
		var uvarByte [8]byte
		binary.PutUvarint(uvarByte[:], uint64(i))
		nodeClient.Call(nil, "pss_sendRaw", hexutil.Encode(msg), hexutil.Encode(topic[:]), hexutil.Encode(uvarByte[:]))
	}
	log.Debug("all messages sent")
}

// testRoutine is the main test function, called by Simulation.Run()
func testRoutine(tstdata *testData, ctx context.Context) error {
	go handlerChannelListener(tstdata, ctx)
	go tstdata.sendAllMsgs()
	received := 0

	// collect incoming messages and terminate with corresponding status when message handler listener ends
	for {
		select {
		case err := <-tstdata.errC:
			return err
		case hn := <-tstdata.msgC:
			received++
			log.Debug("msg received", "msgs_received", received, "total_expected", tstdata.requiredMessages, "id", hn.id, "serial", hn.serial)
			if received == tstdata.allowedMessages {
				close(tstdata.doneC)
				return nil
			}
		}
	}
	return nil
}

func handlerChannelListener(tstdata *testData, ctx context.Context) {
	for {
		select {
		case <-tstdata.doneC: // graceful exit
			tstdata.setDone()
			tstdata.errC <- nil
			return

		case <-ctx.Done(): // timeout or cancel
			tstdata.setDone()
			tstdata.errC <- ctx.Err()
			return

		// incoming message from pss message handler
		case handlerNotification := <-tstdata.handlerC:
			// check if recipient has already received all its messages and notify to fail the test if so
			aMsgs := tstdata.allowedMsgs[handlerNotification.id]
			if len(aMsgs) == 0 {
				tstdata.setDone()
				tstdata.errC <- fmt.Errorf("too many messages received by recipient %x", handlerNotification.id)
				return
			}

			// check if message serial is in expected messages for this recipient and notify to fail the test if not
			idx := -1
			for i, msg := range aMsgs {
				if handlerNotification.serial == msg {
					idx = i
					break
				}
			}
			if idx == -1 {
				tstdata.setDone()
				tstdata.errC <- fmt.Errorf("message %d received by wrong recipient %v", handlerNotification.serial, handlerNotification.id)
				return
			}

			// message is ok, so remove that message serial from the recipient expectation array and notify the main sim thread
			aMsgs[idx] = aMsgs[len(aMsgs)-1]
			aMsgs = aMsgs[:len(aMsgs)-1]
			tstdata.msgC <- handlerNotification
		}
	}
}

func nodeMsgHandler(tstdata *testData, config *adapters.NodeConfig) *handler {
	return &handler{
		f: func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
			cnt := tstdata.incrementMsgCount()
			log.Debug("nodeMsgHandler rcv", "cnt", cnt)

			// using simple serial in message body, makes it easy to keep track of who's getting what
			serial, c := binary.Uvarint(msg)
			if c <= 0 {
				log.Crit(fmt.Sprintf("corrupt message received by %x (uvarint parse returned %d)", config.ID, c))
			}

			if tstdata.isDone() {
				return errors.New("handlers aborted") // terminate if simulation is over
			}

			// pass message context to the listener in the simulation
			tstdata.handlerC <- handlerNotification{
				id:     config.ID,
				serial: serial,
			}
			return nil
		},
		caps: &handlerCaps{
			raw:  true, // we use raw messages for simplicity
			prox: true,
		},
	}
}

// an adaptation of the same services setup as in pss_test.go
// replaces pss_test.go when those tests are rewritten to the new swarm/network/simulation package
func newProxServices(tstdata *testData, allowRaw bool, handlerContextFuncs map[Topic]handlerContextFunc, kademlias map[enode.ID]*network.Kademlia) map[string]simulation.ServiceFunc {
	stateStore := state.NewInmemoryStore()
	kademlia := func(id enode.ID, bzzkey []byte) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		params := network.NewKadParams()
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(bzzkey, params)
		return kademlias[id]
	}
	return map[string]simulation.ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			var err error
			var bzzPrivateKey *ecdsa.PrivateKey
			// normally translation of enode id to swarm address is concealed by the network package
			// however, we need to keep track of it in the test driver as well.
			// if the translation in the network package changes, that can cause these tests to unpredictably fail
			// therefore we keep a local copy of the translation here
			addr := network.NewAddr(ctx.Config.Node())
			bzzPrivateKey, err = simulation.BzzPrivateKeyFromConfig(ctx.Config)
			if err != nil {
				return nil, nil, err
			}
			addr.OAddr = network.PrivateKeyToBzzKey(bzzPrivateKey)
			b.Store(simulation.BucketKeyBzzPrivateKey, bzzPrivateKey)
			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID, addr.OAddr), stateStore, nil, nil), nil, nil
		},
		"pss": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			// execadapter does not exec init()
			initTest()

			// create keys in whisper and set up the pss object
			ctxlocal, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			keys, err := wapi.NewKeyPair(ctxlocal)
			privkey, err := w.GetPrivateKey(keys)
			pssp := NewPssParams().WithPrivateKey(privkey)
			pssp.AllowRaw = allowRaw
			bzzPrivateKey, err := simulation.BzzPrivateKeyFromConfig(ctx.Config)
			if err != nil {
				return nil, nil, err
			}
			bzzKey := network.PrivateKeyToBzzKey(bzzPrivateKey)
			pskad := kademlia(ctx.Config.ID, bzzKey)
			ps, err := NewPss(pskad, pssp)
			if err != nil {
				return nil, nil, err
			}

			// register the handlers we've been passed
			var deregisters []func()
			for tpc, hndlrFunc := range handlerContextFuncs {
				deregisters = append(deregisters, ps.Register(&tpc, hndlrFunc(tstdata, ctx.Config)))
			}

			// if handshake mode is set, add the controller
			// TODO: This should be hooked to the handshake test file
			if useHandshake {
				SetHandshakeController(ps, NewHandshakeParams())
			}

			// we expose some api calls for cheating
			ps.addAPI(rpc.API{
				Namespace: "psstest",
				Version:   "0.3",
				Service:   NewAPITest(ps),
				Public:    false,
			})

			b.Store(simulation.BucketKeyKademlia, pskad)

			// return Pss and cleanups
			return ps, func() {
				// run the handler deregister functions in reverse order
				for i := len(deregisters); i > 0; i-- {
					deregisters[i-1]()
				}
			}, nil
		},
	}
}
