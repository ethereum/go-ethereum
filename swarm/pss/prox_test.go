package pss

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
)

// needed to make the enode id of the receiving node available to the handler for triggers
type handlerContextFunc func(*adapters.NodeConfig) *handler

// struct to notify reception of messages to simulation driver
// TODO To make code cleaner:
// - consider a separate pss unwrap to message event in sim framework (this will make eventual message propagation analysis with pss easier/possible in the future)
// - consider also test api calls to inspect handling results of messages
type handlerNotification struct {
	id     enode.ID
	serial uint64
}

var (
	pof   = pot.DefaultPof(256) // generate messages and index them
	topic = BytesToTopic([]byte{0x00, 0x00, 0x06, 0x82})
	mu    sync.Mutex // keeps handlerDone in sync
	sim   *simulation.Simulation

	mx               sync.Mutex // prevents data race for the test variables
	handlerDone      bool       // set to true on termination of the simulation run
	requiredMessages int
	allowedMessages  int
	messageCount     int

	kademlias    map[enode.ID]*network.Kademlia
	nodeAddrs    map[enode.ID][]byte      // make predictable overlay addresses from the generated random enode ids
	recipients   map[int][]enode.ID       // for logging output only
	allowed      map[int][]enode.ID       // allowed recipients
	expectedMsgs map[enode.ID][]uint64    // message serials we expect respective nodes to receive
	allowedMsgs  map[enode.ID][]uint64    // message serials we expect respective nodes to receive
	senders      map[int]enode.ID         // originating nodes of the messages (intention is to choose as far as possible from the receiving neighborhood)
	handlerC     chan handlerNotification // passes message from pss message handler to simulation driver
	doneC        chan struct{}            // terminates the handler channel listener
	errC         chan error               // error to pass to main sim thread
	msgC         chan handlerNotification // message receipt notification to main sim thread
	msgs         [][]byte                 // recipient addresses of messages
)

func resetTestVariables() {
	handlerDone = false
	requiredMessages = 0
	allowedMessages = 0
	msgs = nil
	sim = nil
	resetMsgCount()
	kademlias = make(map[enode.ID]*network.Kademlia)
	nodeAddrs = make(map[enode.ID][]byte)
	recipients = make(map[int][]enode.ID)
	allowed = make(map[int][]enode.ID)
	expectedMsgs = make(map[enode.ID][]uint64)
	allowedMsgs = make(map[enode.ID][]uint64)
	senders = make(map[int]enode.ID)
	handlerC = make(chan handlerNotification)
	doneC = make(chan struct{})
	errC = make(chan error)
	msgC = make(chan handlerNotification)
}

func getMsgCount() int {
	mx.Lock()
	defer mx.Unlock()
	return messageCount
}

func resetMsgCount() {
	mx.Lock()
	messageCount = 0
	mx.Unlock()
}

func incrementMsgCount() int {
	mx.Lock()
	defer mx.Unlock()
	messageCount++
	return messageCount
}

func isDone() bool {
	mu.Lock()
	defer mu.Unlock()
	return handlerDone
}

func setDone() {
	mu.Lock()
	defer mu.Unlock()
	handlerDone = true
}

func getCmdParams(t *testing.T) (int, int) {
	args := strings.Split(t.Name(), "/")
	msgCount, err := strconv.ParseInt(args[2], 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	nodeCount, err := strconv.ParseInt(args[1], 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return int(msgCount), int(nodeCount)
}

func readSnapshot(t *testing.T, nodeCount int) simulations.Snapshot {
	f, err := os.Open(fmt.Sprintf("testdata/snapshot_%d.json", nodeCount))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	jsonbyte, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	var snap simulations.Snapshot
	err = json.Unmarshal(jsonbyte, &snap)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func assignTestVariables(sim *simulation.Simulation, msgCount int) {
	log.Debug("TestProxNetwork start")
	for _, nodeId := range sim.NodeIDs() {
		nodeAddrs[nodeId] = nodeIDToAddr(nodeId)
	}

	for i := 0; i < int(msgCount); i++ {
		msgAddr := pot.RandomAddress() // we choose message addresses randomly
		msgs = append(msgs, msgAddr.Bytes())
		smallestPo := 256
		var targets []enode.ID
		var closestPO int

		// loop through all nodes and add the message to recipient indices
		for _, nod := range sim.Net.GetNodes() {
			po, _ := pof(msgs[i], nodeAddrs[nod.ID()], 0)
			depth := kademlias[nod.ID()].NeighbourhoodDepth()

			// only nodes with closest IDs (wrt msg) will receive the msg
			if po > closestPO {
				closestPO = po
				targets = nil
				targets = append(targets, nod.ID())
			} else if po == closestPO {
				targets = append(targets, nod.ID())
			}

			if po >= depth {
				allowedMessages++
				allowed[i] = append(allowed[i], nod.ID())
				allowedMsgs[nod.ID()] = append(allowedMsgs[nod.ID()], uint64(i))
			}

			// a node with the smallest PO (wrt msg) will be the sender
			if po < smallestPo {
				smallestPo = po
				senders[i] = nod.ID()
			}
		}

		requiredMessages += len(targets)
		for _, id := range targets {
			recipients[i] = append(recipients[i], id)
			expectedMsgs[id] = append(expectedMsgs[id], uint64(i))
		}

		log.Debug("nn for msg", "targets", len(recipients[i]), "msgidx", i, "msg", common.Bytes2Hex(msgAddr[:8]), "sender", senders[i], "senderpo", smallestPo)
	}
	log.Debug("msgs to receive", "count", requiredMessages)
}

func TestProxNetwork(t *testing.T) {
	t.Run("16/16", testProxNetwork)
}

// params in run name: nodes/msgs
func TestProxNetworkLong(t *testing.T) {
	if !*longrunning {
		t.Skip("run with --longrunning flag to run extensive network tests")
	}
	t.Run("8/100", testProxNetwork)
	t.Run("16/100", testProxNetwork)
	t.Run("32/100", testProxNetwork)
	t.Run("64/100", testProxNetwork)
}

// This tests generates a sequenced number of messages with random addresses.
// It then calculates which nodes in the network have the address of each message
// within their nearest neighborhood depth, and stores them as recipients.
// Upon sending the messages, it verifies that the respective message is passed to the message handlers of these recipients.
// It will fail if a recipient handles a message it should not, or if after propagation not all expected messages are handled (timeout)
func testProxNetwork(t *testing.T) {
	resetTestVariables()
	msgCount, nodeCount := getCmdParams(t)
	handlerContextFuncs := make(map[Topic]handlerContextFunc)
	handlerContextFuncs[topic] = nodeMsgHandler
	services := newProxServices(true, handlerContextFuncs, kademlias)
	sim = simulation.New(services)
	defer sim.Close()
	err := sim.UploadSnapshot(fmt.Sprintf("testdata/snapshot_%d.json", nodeCount))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3) // todo: increase
	defer cancel()
	snap := readSnapshot(t, nodeCount)
	err = sim.WaitTillSnapshotRecreated(ctx, snap)
	if err != nil {
		t.Fatalf("failed to recreate snapshot: %s", err)
	}
	assignTestVariables(sim, msgCount)
	result := sim.Run(ctx, runFunc)
	if result.Error != nil {
		// context deadline exceeded
		// however, it might just mean that not all possible messages are received
		// now we must check if all required messages are received
		cnt := getMsgCount()
		log.Debug("TestProxNetwork finnished", "rcv", cnt)
		if cnt < requiredMessages {
			t.Fatal(result.Error)
		}
	}
	t.Logf("completed %d", result.Duration)
}

func sendAllMsgs(sim *simulation.Simulation, msgs [][]byte, senders map[int]enode.ID) {
	for i, msg := range msgs {
		log.Debug("sending msg", "idx", i, "from", senders[i])
		nodeClient, err := sim.Net.GetNode(senders[i]).Client()
		if err != nil {
			log.Crit(err.Error())
		}
		var uvarByte [8]byte
		binary.PutUvarint(uvarByte[:], uint64(i))
		nodeClient.Call(nil, "pss_sendRaw", hexutil.Encode(msg), hexutil.Encode(topic[:]), hexutil.Encode(uvarByte[:]))
	}
	log.Debug("all messages sent")
}

func runFunc(ctx context.Context, sim *simulation.Simulation) error {
	go handlerChannelListener(ctx)
	go sendAllMsgs(sim, msgs, senders)
	received := 0

	// collect incoming messages and terminate with corresponding status when message handler listener ends
	for {
		select {
		case err := <-errC:
			return err
		case hn := <-msgC:
			received++
			log.Debug("msg received", "msgs_received", received, "total_expected", requiredMessages, "id", hn.id, "serial", hn.serial)
			if received == allowedMessages {
				doneC <- struct{}{}
				close(doneC)
				return nil
			}
		}
	}
	return nil
}

func handlerChannelListener(ctx context.Context) {
	for {
		select {
		case <-doneC: // graceful exit
			setDone()
			errC <- nil
			return

		case <-ctx.Done(): // timeout or cancel
			setDone()
			errC <- ctx.Err()
			return

		// incoming message from pss message handler
		case handlerNotification := <-handlerC:
			// check if recipient has already received all its messages and notify to fail the test if so
			aMsgs := allowedMsgs[handlerNotification.id]
			if len(aMsgs) == 0 {
				setDone()
				errC <- fmt.Errorf("too many messages received by recipient %x", handlerNotification.id)
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
				setDone()
				errC <- fmt.Errorf("message %d received by wrong recipient %v", handlerNotification.serial, handlerNotification.id)
				return
			}

			// message is ok, so remove that message serial from the recipient expectation array and notify the main sim thread
			aMsgs[idx] = aMsgs[len(aMsgs)-1]
			aMsgs = aMsgs[:len(aMsgs)-1]
			msgC <- handlerNotification
		}
	}
}

func nodeMsgHandler(config *adapters.NodeConfig) *handler {
	return &handler{
		f: func(msg []byte, p *p2p.Peer, asymmetric bool, keyid string) error {
			cnt := incrementMsgCount()
			log.Debug("nodeMsgHandler rcv", "cnt", cnt)

			// using simple serial in message body, makes it easy to keep track of who's getting what
			serial, c := binary.Uvarint(msg)
			if c <= 0 {
				log.Crit(fmt.Sprintf("corrupt message received by %x (uvarint parse returned %d)", config.ID, c))
			}

			if isDone() {
				return errors.New("handlers aborted") // terminate if simulation is over
			}

			// pass message context to the listener in the simulation
			handlerC <- handlerNotification{
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
func newProxServices(allowRaw bool, handlerContextFuncs map[Topic]handlerContextFunc, kademlias map[enode.ID]*network.Kademlia) map[string]simulation.ServiceFunc {
	stateStore := state.NewInmemoryStore()
	kademlia := func(id enode.ID) *network.Kademlia {
		if k, ok := kademlias[id]; ok {
			return k
		}
		params := network.NewKadParams()
		params.MaxBinSize = 3
		params.MinBinSize = 1
		params.MaxRetries = 1000
		params.RetryExponent = 2
		params.RetryInterval = 1000000
		kademlias[id] = network.NewKademlia(id[:], params)
		return kademlias[id]
	}
	return map[string]simulation.ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			// normally translation of enode id to swarm address is concealed by the network package
			// however, we need to keep track of it in the test driver aswell.
			// if the translation in the network package changes, that can cause thiese tests to unpredictably fail
			// therefore we keep a local copy of the translation here
			addr := network.NewAddr(ctx.Config.Node())
			addr.OAddr = nodeIDToAddr(ctx.Config.Node().ID())

			hp := network.NewHiveParams()
			hp.Discovery = false
			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kademlia(ctx.Config.ID), stateStore, nil, nil), nil, nil
		},
		"pss": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			// execadapter does not exec init()
			initTest()

			// create keys in whisper and set up the pss object
			ctxlocal, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			keys, err := wapi.NewKeyPair(ctxlocal)
			privkey, err := w.GetPrivateKey(keys)
			pssp := NewPssParams().WithPrivateKey(privkey)
			pssp.AllowRaw = allowRaw
			pskad := kademlia(ctx.Config.ID)
			ps, err := NewPss(pskad, pssp)
			if err != nil {
				return nil, nil, err
			}
			b.Store(simulation.BucketKeyKademlia, pskad)

			// register the handlers we've been passed
			var deregisters []func()
			for tpc, hndlrFunc := range handlerContextFuncs {
				deregisters = append(deregisters, ps.Register(&tpc, hndlrFunc(ctx.Config)))
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

// makes sure we create the addresses the same way in driver and service setup
func nodeIDToAddr(id enode.ID) []byte {
	return id.Bytes()
}
