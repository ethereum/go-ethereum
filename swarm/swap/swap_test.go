// Copyreeeeight 2018 The go-ethereum Authors
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

package swap

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/state"
	colorable "github.com/mattn/go-colorable"
)

var (
	p2pPort       = 30100
	ipcpath       = ".swarm.ipc"
	datadirPrefix = ".data_"
	stackW        = &sync.WaitGroup{}
	loglevel      = flag.Int("loglevel", 2, "verbosity of logs")
)

var testSpec = &protocols.Spec{
	Name:       "swapTestSpec",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		testExceedsPayAtMsg{},
		testExceedsDropAtMsg{},
		testCheapMsg{},
		testSizeBasedMsg{},
		testChargeRecvMsg{},
	},
}

//dummy implementation of a MsgReadWriter
//this allows for quick and easy unit tests without
//having to build up the complete protocol
type dummyRW struct{}

func (d *dummyRW) WriteMsg(msg p2p.Msg) error {
	return nil
}

func (d *dummyRW) ReadMsg() (p2p.Msg, error) {
	return p2p.Msg{
		Code:       0,
		Size:       0,
		Payload:    nil,
		ReceivedAt: time.Now(),
	}, nil
}

//define a couple of messages for tests
type testExceedsPayAtMsg struct{}
type testExceedsDropAtMsg struct{}
type testCheapMsg struct{}
type testSizeBasedMsg struct{}
type testChargeRecvMsg struct{}

//this message is just one unit more expensive than the payment threshold
func (tmsg *testExceedsPayAtMsg) Price() *big.Int {
	diff := &big.Int{}
	diff = diff.Abs(payAt)
	return diff.Add(diff, big.NewInt(1))
}

//this message is just one unit more expensive than the disconnect threshold
func (tmsg *testExceedsDropAtMsg) Price() *big.Int {
	diff := &big.Int{}
	diff = diff.Abs(dropAt)
	return diff.Add(diff, big.NewInt(1))
}

//a message with an arbitrary cost
func (tmsg *testCheapMsg) Price() *big.Int {
	return big.NewInt(100)
}

//a message which needs to be charged based on price
func (tmsg *testSizeBasedMsg) Price() *big.Int {
	return big.NewInt(222)
}

//a message which needs to be charged based on price
func (tmsg *testChargeRecvMsg) Price() *big.Int {
	return big.NewInt(999)
}

func init() {
	flag.Parse()

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

//check that the disconnect threshold is below the payment threshold
func TestLimits(t *testing.T) {
	if dropAt.Cmp(payAt) > -1 {
		t.Fatal(fmt.Sprintf("dropAt limit is not lower than payAt limit, dropAt: %s, payAt: %s", dropAt.String(), payAt.String()))
	}
}

//unit test for exceeds pay limit
//when the payment threshold is reached, a cheque will be issued
//this test checks that a cheque is present if a message is sent
//which exceeds the payment threshold
//(note: the details of cheque handling will need to be fleshed out
//in future iterations, current implementation is very primitive)
func TestExceedsPayAt(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	peerID := adapters.RandomNodeConfig().ID

	code, ok := testSpec.GetCode(&testExceedsPayAtMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peerID,
		MsgCode:  &code,
		MsgSize:  new(uint32),
		Error:    "",
	}

	swap.handleMsgEvent(event)

	//check that a cheque is present
	cheques := swap.chequeManager.openDebitCheques[peerID]
	if cheques == nil {
		t.Fatal("Expected cheques for this peer to be present, but are nil")
	}
	if len(cheques) == 0 {
		t.Fatal("Expected a cheque to have arrived, but len is zero")
	}
	if cheques[0].serial != 1 {
		t.Fatal(fmt.Sprintf("Expected the serial to be one (first message but is: %d", cheques[0].serial))
	}
	absPayAt := big.NewInt(0)
	if cheques[0].amount.Cmp(absPayAt.Abs(payAt)) != 0 {
		t.Fatal(fmt.Sprintf("Expected the serial to be one (first message but is: %d", cheques[0].serial))
	}
}

//unit test for exceeds drop limit
//tests that a message is being sent which crosses the drop limit
//in that case, we should receive a InsufficientFunds error
func TestExceedsDropAt(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	//peerID := adapters.RandomNodeConfig().ID
	peer := newDummyPeer()
	swap.protocol.setPeer(peer.Peer)

	code, ok := testSpec.GetCode(&testExceedsDropAtMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peer.ID(),
		MsgCode:  &code,
		MsgSize:  new(uint32),
		Error:    "",
	}

	swap.handleMsgEvent(event)
	if event.Error != ErrInsufficientFunds.Error() {
		t.Fatal("Excpected this test to fail with insufficient funds, but it did not")
	}
}

//send a message with cost,
//then check that the balance has the expected amount
func TestSendCheapMessage(t *testing.T) {
	fmt.Println("send")
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	//peerID := adapters.RandomNodeConfig().ID
	peer := newDummyPeer()
	swap.protocol.setPeer(peer.Peer)
	//set an arbitrary test balance value
	testBalance := big.NewInt(1234567890)
	swap.peers[peer.ID()] = testBalance

	code, ok := testSpec.GetCode(&testCheapMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peer.ID(),
		MsgCode:  &code,
		MsgSize:  new(uint32),
		Error:    "",
	}

	swap.handleMsgEvent(event)

	peerBalance := swap.peers[peer.ID()]
	msg := &testCheapMsg{}

	//check the new balance
	if peerBalance.Cmp(testBalance.Sub(testBalance, msg.Price())) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			testBalance.Sub(testBalance, msg.Price()).String(), peerBalance.String()))
	}
}

//try restoring a balance from state store
//this is simulated by creating a node,
//assigning it an arbitrary balance,
//send a message (triggers to save to store),
//then create a different SwapPeer instance with same peerID,
//which will try to load a balance from the stateStore
func TestRestoreBalanceFromStateStore(t *testing.T) {
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	//peerID := adapters.RandomNodeConfig().ID
	peer := newDummyPeer()
	swap.protocol.setPeer(peer.Peer)
	//create a reference an arbitrary balance
	testBalance := big.NewInt(1234567890)
	//assign the same value to the peer
	swap.peers[peer.ID()] = big.NewInt(1234567890)

	code, ok := testSpec.GetCode(&testCheapMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peer.ID(),
		MsgCode:  &code,
		MsgSize:  new(uint32),
		Error:    "",
	}

	swap.handleMsgEvent(event)

	peerBalance := swap.peers[peer.ID()]
	expectedBalance := &big.Int{}
	swap.stateStore.Get(peer.ID().String(), expectedBalance)

	//compare the balances
	expectedBalance.Sub(testBalance, (&testCheapMsg{}).Price())
	if peerBalance.Cmp(expectedBalance) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			expectedBalance.String(), peerBalance.String()))
	}
}

//send a message with cost,
//then check that the balance has the expected amount
//the message is charged size based
func TestSendSizeBasedMsg(t *testing.T) {
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	peer := newDummyPeer()
	swap.protocol.setPeer(peer.Peer)
	//set an arbitrary test balance value
	testBalance := big.NewInt(1234567890)
	swap.peers[peer.ID()] = testBalance

	code, ok := testSpec.GetCode(&testSizeBasedMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	size := new(uint32)
	*size = 500

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peer.ID(),
		MsgCode:  &code,
		MsgSize:  size,
		Error:    "",
	}

	swap.handleMsgEvent(event)

	peerBalance := swap.peers[peer.ID()]
	msg := &testSizeBasedMsg{}
	expectedBalance := &big.Int{}
	expectedBalance.Mul(msg.Price(), big.NewInt(int64(*size)))

	//check the new balance
	if peerBalance.Cmp(testBalance.Sub(testBalance, expectedBalance)) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			expectedBalance.String(), peerBalance.String()))
	}
}

//charge as being the receiver of a receiver-pays message
//as this test for simplicity only sends messages,
//it means that the balance is changed in positive,
//instead of negative as with all other tests
func TestChargeAsReceiver(t *testing.T) {
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	swap.priceOracle = setupPriceMatrix()

	node := createAndStartSvcNode(swap, t)
	defer node.Stop()
	defer os.RemoveAll(node.DataDir())

	//peerID := adapters.RandomNodeConfig().ID
	peer := newDummyPeer()
	swap.protocol.setPeer(peer.Peer)
	//set an arbitrary test balance value
	testBalance := big.NewInt(1234567890)
	swap.peers[peer.ID()] = testBalance

	code, ok := testSpec.GetCode(&testChargeRecvMsg{})
	if !ok {
		t.Fatal("test message not found in spec")
	}

	event := &p2p.PeerEvent{
		Type:     p2p.PeerEventTypeMsgSend,
		Protocol: testSpec.Name,
		Peer:     peer.ID(),
		MsgCode:  &code,
		MsgSize:  new(uint32),
		Error:    "",
	}

	swap.handleMsgEvent(event)

	peerBalance := swap.peers[peer.ID()]
	msg := &testChargeRecvMsg{}

	//check the new balance
	if peerBalance.Cmp(testBalance.Add(testBalance, msg.Price())) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			testBalance.Add(testBalance, msg.Price()).String(), peerBalance.String()))
	}
}

//create a test swap account
//creates a stateStore for persistence and a Swap account
func createTestSwap(t *testing.T) (*Swap, string) {
	dir, err := ioutil.TempDir("", "swap_test_store")
	if err != nil {
		t.Fatal(err)
	}
	stateStore, err2 := state.NewDBStore(dir)
	if err2 != nil {
		t.Fatal(err2)
	}
	swap := New(stateStore)
	return swap, dir
}

func setupPriceMatrix() PriceOracle {

	priceOracle := &DefaultPriceOracle{}
	priceOracle.priceMatrix = make(map[string]map[uint64]*PriceTag)
	priceOracle.priceMatrix[testSpec.Name] = make(map[uint64]*PriceTag)
	protoMatrix := priceOracle.priceMatrix[testSpec.Name]
	protoMatrix[0] = &PriceTag{
		Direction: ChargeSender,
		Price:     (&testExceedsPayAtMsg{}).Price(),
	}

	protoMatrix[1] = &PriceTag{
		Direction: ChargeSender,
		Price:     (&testExceedsDropAtMsg{}).Price(),
	}

	protoMatrix[2] = &PriceTag{
		Direction: ChargeSender,
		Price:     (&testCheapMsg{}).Price(),
	}

	protoMatrix[3] = &PriceTag{
		Direction: ChargeSender,
		Price:     (&testSizeBasedMsg{}).Price(),
		SizeBased: true,
	}

	protoMatrix[4] = &PriceTag{
		Direction: ChargeReceiver,
		Price:     (&testChargeRecvMsg{}).Price(),
	}

	return priceOracle
}

//dummy message handler (needed or we will have a panic in the accounting)
func dummyMsgHandler(ctx context.Context, msg interface{}) error {
	return nil
}

func createAndStartSvcNode(swap *Swap, t *testing.T) *node.Node {
	stack, err := newServiceNode(p2pPort, 0, 0)
	if err != nil {
		t.Fatal("Create servicenode #1 fail", "err", err)
	}

	swapsvc := func(ctx *node.ServiceContext) (node.Service, error) {
		return swap, nil
	}

	err = stack.Register(swapsvc)
	if err != nil {
		t.Fatal("Register service in servicenode #1 fail", "err", err)
	}

	// start the nodes
	err = stack.Start()
	if err != nil {
		t.Fatal("servicenode #1 start failed", "err", err)
	}

	return stack
}

//tests some basic things over RPC
func TestSwapRPC(t *testing.T) {

	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	stack := createAndStartSvcNode(swap, t)
	defer stack.Stop()
	defer os.RemoveAll(stack.DataDir())

	// connect to the servicenode RPCs
	rpcclient, err := rpc.Dial(filepath.Join(stack.DataDir(), ipcpath))
	if err != nil {
		t.Fatal("connect to servicenode IPC fail", "err", err)
	}
	defer os.RemoveAll(stack.DataDir())

	var balance *big.Int
	err = rpcclient.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("servicenode balance", "balance", balance)

	if balance.Cmp(big.NewInt(0)) != 0 {
		t.Fatal("Expected balance to be 0 but it is not")
	}

	dummyPeer1 := newDummyPeer()
	dummyPeer2 := newDummyPeer()
	id1 := dummyPeer1.ID()
	id2 := dummyPeer2.ID()

	fake1 := int64(234)
	fake2 := int64(-100)
	fakeBalance1 := big.NewInt(fake1)
	fakeBalance2 := big.NewInt(fake2)

	swap.peers[id1] = fakeBalance1
	swap.peers[id2] = fakeBalance2

	err = rpcclient.Call(&balance, "swap_balanceWithPeer", id1)
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance1", "balance-1", balance)
	if balance.Cmp(fakeBalance1) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to fake balance %s, but it is not", balance.String(), fakeBalance1.String()))
	}

	err = rpcclient.Call(&balance, "swap_balanceWithPeer", id2)
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance2", "balance-2", balance)
	if balance.Cmp(fakeBalance2) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to fake balance %s, but it is not", balance.String(), fakeBalance2.String()))
	}

	err = rpcclient.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance", "balance", balance)

	fakeSum := big.NewInt(fake1 + fake2)
	if balance.Cmp(fakeSum) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to sum %s, but it is not", balance.String(), fakeSum.String()))
	}
}

type dummyPeer struct {
	*Peer
	testFunc func(error)
}

func (dp *dummyPeer) Drop(err error) {
	fmt.Println("DD")
	dp.testFunc(err)
}

//creates a dummy protocols.Peer with dummy MsgReadWriter
func newDummyPeer() *dummyPeer {
	id := adapters.RandomNodeConfig().ID
	protoPeer := protocols.NewPeer(p2p.NewPeer(id, "testPeer", nil), &dummyRW{}, testSpec)
	dummy := &dummyPeer{
		Peer: NewPeer(protoPeer),
	}
	return dummy
}

//creates a p2p.Service node stub
func newServiceNode(port int, httpport int, wsport int, modules ...string) (*node.Node, error) {
	cfg := &node.DefaultConfig
	cfg.P2P.ListenAddr = fmt.Sprintf(":%d", port)
	cfg.P2P.EnableMsgEvents = true
	cfg.P2P.NoDiscovery = true
	cfg.IPCPath = ipcpath
	cfg.DataDir = fmt.Sprintf("%s%d", datadirPrefix, port)
	if httpport > 0 {
		cfg.HTTPHost = node.DefaultHTTPHost
		cfg.HTTPPort = httpport
	}
	if wsport > 0 {
		cfg.WSHost = node.DefaultWSHost
		cfg.WSPort = wsport
		cfg.WSOrigins = []string{"*"}
		for i := 0; i < len(modules); i++ {
			cfg.WSModules = append(cfg.WSModules, modules[i])
		}
	}
	stack, err := node.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("ServiceNode create fail: %v", err)
	}
	return stack, nil
}
