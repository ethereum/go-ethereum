package pss

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
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
	sim                *simulation.Simulation
	kademlias          map[enode.ID]*network.Kademlia
	nodeAddresses      map[enode.ID][]byte // make predictable overlay addresses from the generated random enode ids
	senders            map[int]enode.ID    // originating nodes of the messages (intention is to choose as far as possible from the receiving neighborhood)
	recipientAddresses [][]byte

	requiredMsgCount int
	requiredMsgs     map[enode.ID][]uint64 // message serials we expect respective nodes to receive
	allowedMsgs      map[enode.ID][]uint64 // message serials we expect respective nodes to receive

	notifications []handlerNotification // notification queue
	totalMsgCount int
	handlerDone   bool // set to true on termination of the simulation run
	mu            sync.Mutex
}

var (
	pof   = pot.DefaultPof(256) // generate messages and index them
	topic = BytesToTopic([]byte{0xf3, 0x9e, 0x06, 0x82})
)

func (td *testData) pushNotification(val handlerNotification) {
	td.mu.Lock()
	td.notifications = append(td.notifications, val)
	td.mu.Unlock()
}

func (td *testData) popNotification() (first handlerNotification, exist bool) {
	td.mu.Lock()
	if len(td.notifications) > 0 {
		exist = true
		first = td.notifications[0]
		td.notifications = td.notifications[1:]
	}
	td.mu.Unlock()
	return first, exist
}

func (td *testData) getMsgCount() int {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.totalMsgCount
}

func (td *testData) incrementMsgCount() int {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.totalMsgCount++
	return td.totalMsgCount
}

func (td *testData) isDone() bool {
	td.mu.Lock()
	defer td.mu.Unlock()
	return td.handlerDone
}

func (td *testData) setDone() {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.handlerDone = true
}

func newTestData() *testData {
	return &testData{
		kademlias:     make(map[enode.ID]*network.Kademlia),
		nodeAddresses: make(map[enode.ID][]byte),
		requiredMsgs:  make(map[enode.ID][]uint64),
		allowedMsgs:   make(map[enode.ID][]uint64),
		senders:       make(map[int]enode.ID),
	}
}

func (td *testData) getKademlia(nodeId *enode.ID) (*network.Kademlia, error) {
	kadif, ok := td.sim.NodeItem(*nodeId, simulation.BucketKeyKademlia)
	if !ok {
		return nil, fmt.Errorf("no kademlia entry for %v", nodeId)
	}
	kad, ok := kadif.(*network.Kademlia)
	if !ok {
		return nil, fmt.Errorf("invalid kademlia entry for %v", nodeId)
	}
	return kad, nil
}

func (td *testData) init(msgCount int) error {
	log.Debug("TestProxNetwork start")

	for _, nodeId := range td.sim.NodeIDs() {
		kad, err := td.getKademlia(&nodeId)
		if err != nil {
			return err
		}
		td.nodeAddresses[nodeId] = kad.BaseAddr()
	}

	for i := 0; i < int(msgCount); i++ {
		msgAddr := pot.RandomAddress() // we choose message addresses randomly
		td.recipientAddresses = append(td.recipientAddresses, msgAddr.Bytes())
		smallestPo := 256
		var targets []enode.ID
		var closestPO int

		// loop through all nodes and find the required and allowed recipients of each message
		// (for more information, please see the comment to the main test function)
		for _, nod := range td.sim.Net.GetNodes() {
			po, _ := pof(td.recipientAddresses[i], td.nodeAddresses[nod.ID()], 0)
			depth := td.kademlias[nod.ID()].NeighbourhoodDepth()

			// only nodes with closest IDs (wrt the msg address) will be required recipients
			if po > closestPO {
				closestPO = po
				targets = nil
				targets = append(targets, nod.ID())
			} else if po == closestPO {
				targets = append(targets, nod.ID())
			}

			if po >= depth {
				td.allowedMsgs[nod.ID()] = append(td.allowedMsgs[nod.ID()], uint64(i))
			}

			// a node with the smallest PO (wrt msg) will be the sender,
			// in order to increase the distance the msg must travel
			if po < smallestPo {
				smallestPo = po
				td.senders[i] = nod.ID()
			}
		}

		td.requiredMsgCount += len(targets)
		for _, id := range targets {
			td.requiredMsgs[id] = append(td.requiredMsgs[id], uint64(i))
		}

		log.Debug("nn for msg", "targets", len(targets), "msgidx", i, "msg", common.Bytes2Hex(msgAddr[:8]), "sender", td.senders[i], "senderpo", smallestPo)
	}
	log.Debug("recipientAddresses to receive", "count", td.requiredMsgCount)
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
	t.Run("16_nodes,_16_messages,_16_seconds", func(t *testing.T) {
		testProxNetwork(t, 16, 16, 16*time.Second)
	})
}

func TestProxNetworkLong(t *testing.T) {
	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("8_nodes,_100_messages,_30_seconds", func(t *testing.T) {
		testProxNetwork(t, 8, 100, 30*time.Second)
	})
	t.Run("16_nodes,_100_messages,_30_seconds", func(t *testing.T) {
		testProxNetwork(t, 16, 100, 30*time.Second)
	})
	t.Run("32_nodes,_100_messages,_60_seconds", func(t *testing.T) {
		testProxNetwork(t, 32, 100, 1*time.Minute)
	})
	t.Run("64_nodes,_100_messages,_60_seconds", func(t *testing.T) {
		testProxNetwork(t, 64, 100, 1*time.Minute)
	})
	t.Run("128_nodes,_100_messages,_120_seconds", func(t *testing.T) {
		testProxNetwork(t, 128, 100, 2*time.Minute)
	})
}

func testProxNetwork(t *testing.T, nodeCount int, msgCount int, timeout time.Duration) {
	td := newTestData()
	handlerContextFuncs := make(map[Topic]handlerContextFunc)
	handlerContextFuncs[topic] = nodeMsgHandler
	services := newProxServices(td, true, handlerContextFuncs, td.kademlias)
	td.sim = simulation.New(services)
	defer td.sim.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	filename := fmt.Sprintf("testdata/snapshot_%d.json", nodeCount)
	err := td.sim.UploadSnapshot(ctx, filename)
	if err != nil {
		t.Fatal(err)
	}
	err = td.init(msgCount) // initialize the test data
	if err != nil {
		t.Fatal(err)
	}
	wrapper := func(c context.Context, _ *simulation.Simulation) error {
		return testRoutine(td, c)
	}
	result := td.sim.Run(ctx, wrapper) // call the main test function
	if result.Error != nil {
		timedOut := result.Error == context.DeadlineExceeded
		if !timedOut || td.getMsgCount() < td.requiredMsgCount {
			t.Fatal(result.Error)
		}
	}
}

func (td *testData) sendAllMsgs() error {
	nodes := make(map[int]*rpc.Client)
	for i := range td.recipientAddresses {
		nodeClient, err := td.sim.Net.GetNode(td.senders[i]).Client()
		if err != nil {
			return err
		}
		nodes[i] = nodeClient
	}

	for i, msg := range td.recipientAddresses {
		log.Debug("sending msg", "idx", i, "from", td.senders[i])
		nodeClient := nodes[i]
		var uvarByte [8]byte
		binary.PutUvarint(uvarByte[:], uint64(i))
		nodeClient.Call(nil, "pss_sendRaw", hexutil.Encode(msg), hexutil.Encode(topic[:]), hexutil.Encode(uvarByte[:]))
	}
	return nil
}

func isMoreTimeLeft(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// testRoutine is the main test function, called by Simulation.Run()
func testRoutine(td *testData, ctx context.Context) error {

	hasMoreRound := func(err error, hadMessage bool) bool {
		return err == nil && (hadMessage || isMoreTimeLeft(ctx))
	}

	if err := td.sendAllMsgs(); err != nil {
		return err
	}

	var err error
	received := 0
	hadMessage := false

	for oneMoreRound := true; oneMoreRound; oneMoreRound = hasMoreRound(err, hadMessage) {
		message, hadMessage := td.popNotification()

		if !isMoreTimeLeft(ctx) {
			// Stop handlers from sending more messages.
			// Note: only best effort, race is possible.
			td.setDone()
		}

		if hadMessage {
			if td.isAllowedMessage(message) {
				received++
				log.Debug("msg received", "msgs_received", received, "total_expected", td.requiredMsgCount, "id", message.id, "serial", message.serial)
			} else {
				err = fmt.Errorf("message %d received by wrong recipient %v", message.serial, message.id)
			}
		} else {
			time.Sleep(32 * time.Millisecond)
		}
	}

	if err != nil {
		return err
	}

	if td.getMsgCount() < td.requiredMsgCount {
		return ctx.Err()
	}
	return nil
}

func (td *testData) isAllowedMessage(n handlerNotification) bool {
	// check if message serial is in expected messages for this recipient
	for _, s := range td.allowedMsgs[n.id] {
		if n.serial == s {
			return true
		}
	}
	return false
}

func (td *testData) removeAllowedMessage(id enode.ID, index int) {
	last := len(td.allowedMsgs[id]) - 1
	td.allowedMsgs[id][index] = td.allowedMsgs[id][last]
	td.allowedMsgs[id] = td.allowedMsgs[id][:last]
}

func nodeMsgHandler(td *testData, config *adapters.NodeConfig) *handler {
	return &handler{
		f: func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
			if td.isDone() {
				return nil // terminate if simulation is over
			}

			td.incrementMsgCount()

			// using simple serial in message body, makes it easy to keep track of who's getting what
			serial, c := binary.Uvarint(msg)
			if c <= 0 {
				log.Crit(fmt.Sprintf("corrupt message received by %x (uvarint parse returned %d)", config.ID, c))
			}

			td.pushNotification(handlerNotification{id: config.ID, serial: serial})
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
func newProxServices(td *testData, allowRaw bool, handlerContextFuncs map[Topic]handlerContextFunc, kademlias map[enode.ID]*network.Kademlia) map[string]simulation.ServiceFunc {
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
			bzzKey := network.PrivateKeyToBzzKey(bzzPrivateKey)
			pskad := kademlia(ctx.Config.ID, bzzKey)
			b.Store(simulation.BucketKeyKademlia, pskad)
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
			b.Store(simulation.BucketKeyKademlia, pskad)
			ps, err := NewPss(pskad, pssp)
			if err != nil {
				return nil, nil, err
			}

			// register the handlers we've been passed
			var deregisters []func()
			for tpc, hndlrFunc := range handlerContextFuncs {
				deregisters = append(deregisters, ps.Register(&tpc, hndlrFunc(td, ctx.Config)))
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
