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
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	mrand "math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/state"
	colorable "github.com/mattn/go-colorable"
)

var (
	p2pPort       = 30100
	ipcpath       = ".swarm.ipc"
	datadirPrefix = ".data_"
	loglevel      = flag.Int("loglevel", 2, "verbosity of logs")
)

type testPriceOracle struct{}

func (tpo *testPriceOracle) Accountable(msg interface{}) bool {
	_, ok := testSpec.GetCode(msg)
	return ok
}

func (tpo *testPriceOracle) Price(size uint32, msg interface{}) (protocols.EntryDirection, uint64) {
	switch msg := msg.(type) {
	case *testExceedsPayAtMsg:
		return msg.Price(size)
	case *testExceedsDropAtMsg:
		return msg.Price(size)
	case *testCheapMsg:
		return msg.Price(size)
	case *testSizeBasedMsg:
		return msg.Price(size)
	case *testChargeRecvMsg:
		return msg.Price(size)
	}
	return false, 0
}

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
type testSizeBasedMsg struct {
	Data []byte
}
type testChargeRecvMsg struct{}

//this message is just one unit more expensive than the payment threshold
func (tmsg *testExceedsPayAtMsg) Price(size uint32) (protocols.EntryDirection, uint64) {
	return protocols.ChargeSender, uint64(math.Abs(float64(payAt)) + float64(1))
}

//this message is just one unit more expensive than the disconnect threshold
func (tmsg *testExceedsDropAtMsg) Price(size uint32) (protocols.EntryDirection, uint64) {
	return protocols.ChargeSender, uint64(math.Abs(float64(dropAt)) + float64(1))
}

//a message with an arbitrary cost
func (tmsg *testCheapMsg) Price(size uint32) (protocols.EntryDirection, uint64) {
	return protocols.ChargeSender, uint64(10)
}

//a message which needs to be charged based on price
func (tmsg *testSizeBasedMsg) Price(size uint32) (protocols.EntryDirection, uint64) {
	return protocols.ChargeSender, uint64(size) * uint64(100)
}

//a message which needs to be charged based on price
func (tmsg *testChargeRecvMsg) Price(size uint32) (protocols.EntryDirection, uint64) {
	return protocols.ChargeReceiver, uint64(999)
}

func init() {
	flag.Parse()

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

//check that the disconnect threshold is below the payment threshold
func TestLimits(t *testing.T) {
	if dropAt >= payAt {
		t.Fatal(fmt.Sprintf("dropAt limit is not lower than payAt limit, dropAt: %d, payAt: %d", dropAt, payAt))
	}
}

//check that the disconnect threshold is below the payment threshold
func TestRepeatedBookings(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	testPeer := newDummyPeer()
	//size is irrelevant for this test
	size := uint32(0)
	amount := mrand.Intn(100)
	cnt := 1 + mrand.Intn(10)
	for i := 0; i < cnt; i++ {
		swap.Credit(testPeer.Peer.Peer, uint64(amount), size)
	}
	expectedBalance := int64(cnt * amount)
	realBalance := swap.balances[testPeer.ID()]
	if expectedBalance != realBalance {
		t.Fatal(fmt.Sprintf("After %d credits of %d, expected balance to be: %d, but is: %d", cnt, amount, expectedBalance, realBalance))
	}

	testPeer2 := newDummyPeer()
	amount = mrand.Intn(100)
	cnt = 1 + mrand.Intn(10)
	for i := 0; i < cnt; i++ {
		swap.Debit(testPeer2.Peer.Peer, uint64(amount), size)
	}
	expectedBalance = int64(0 - (cnt * amount))
	realBalance = swap.balances[testPeer2.ID()]
	if expectedBalance != realBalance {
		t.Fatal(fmt.Sprintf("After %d debits of %d, expected balance to be: %d, but is: %d", cnt, amount, expectedBalance, realBalance))
	}

	//mixed debits and credits
	amount1 := mrand.Intn(100)
	amount2 := mrand.Intn(100)
	amount3 := mrand.Intn(100)
	swap.Credit(testPeer2.Peer.Peer, uint64(amount1), size)
	swap.Credit(testPeer2.Peer.Peer, uint64(amount2), size)
	swap.Debit(testPeer2.Peer.Peer, uint64(amount3), size)

	expectedBalance = expectedBalance + int64(amount1+amount2-amount3)
	realBalance = swap.balances[testPeer2.ID()]

	if expectedBalance != realBalance {
		t.Fatal(fmt.Sprintf("After mixed debits and credits, expected balance to be: %d, but is: %d", expectedBalance, realBalance))
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

	msg := &testExceedsPayAtMsg{}
	ctx := context.Background()
	testPeer := newDummyPeer()
	testPeer.Send(ctx, msg)

	_, price := msg.Price(0)
	if swap.balances[testPeer.ID()] != int64(0-price) {
		t.Fatalf("Expected balance to be %d, but is %d", price, swap.balances[testPeer.ID()])
	}

	//check that a cheque has been created
	cheques := swap.chequeManager.openDebitCheques[testPeer.ID()]
	if cheques == nil {
		t.Fatal("Expected cheques for this peer to be present, but are nil")
	}
	if len(cheques) == 0 {
		t.Fatal("Expected a cheque to have arrived, but len is zero")
	}
	if cheques[0].serial != 1 {
		t.Fatal(fmt.Sprintf("Expected the serial to be one (first message but is: %d", cheques[0].serial))
	}
	if float64(cheques[0].amount) != float64(math.Abs(float64(payAt))) {
		t.Fatal(fmt.Sprintf("Expected cheques amount to be equal to payAt limit, but it is: %d", cheques[0].amount))
	}
}

//unit test for exceeds drop limit
//tests that a message is being sent which crosses the drop limit
//in that case, we should receive a InsufficientFunds error
func TestExceedsDropAt(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	msg := &testExceedsDropAtMsg{}
	ctx := context.Background()
	testPeer := newDummyPeer()
	err := testPeer.Send(ctx, msg)

	_, price := msg.Price(0)
	if swap.balances[testPeer.ID()] != int64(0-price) {
		t.Fatalf("Expected balance to be %d, but is %d", price, swap.balances[testPeer.ID()])
	}

	if err != ErrInsufficientFunds {
		t.Fatal("Expected this test to fail with insufficient funds, but it did not")
	}
}

//send a message with cost,
//then check that the balance has the expected amount
func TestSendCheapMessage(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	msg := &testCheapMsg{}
	ctx := context.Background()
	testPeer := newDummyPeer()
	testPeer.Send(ctx, msg)

	//check the new balance
	_, price := msg.Price(0)
	if swap.balances[testPeer.ID()] != int64(0-price) {
		t.Fatalf("Expected balance to be %d, but is %d", price, swap.balances[testPeer.ID()])
	}
}

//try restoring a balance from state store
//this is simulated by creating a node,
//assigning it an arbitrary balance,
//send a message (triggers to save to store),
//then create a different SwapPeer instance with same peerID,
//which will try to load a balance from the stateStore
func TestRestoreBalanceFromStateStore(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	msg := &testCheapMsg{}
	ctx := context.Background()
	testPeer := newDummyPeer()
	testPeer.Send(ctx, msg)

	//check the new balance
	_, price := msg.Price(0)
	expectedBalance := int64(0 - price)
	if swap.balances[testPeer.ID()] != expectedBalance {
		t.Fatalf("Expected balance to be %d, but is %d", price, swap.balances[testPeer.ID()])
	}

	var tmpBalance int64
	swap.stateStore.Get(testPeer.ID().String(), &tmpBalance)
	//compare the balances
	if expectedBalance != tmpBalance {
		t.Fatal(fmt.Sprintf("Unexpected balance value in stateStore after sending cheap message test - stateStore should have saved it. Expected balance: %d, balance is: %d",
			expectedBalance, tmpBalance))
	}

	swap.stateStore.Close()
	swap.stateStore = nil

	stateStore, err := state.NewDBStore(testDir)
	if err != nil {
		t.Fatal(err)
	}

	var newBalance int64
	stateStore.Get(testPeer.ID().String(), &newBalance)

	//compare the balances
	if expectedBalance != newBalance {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %d, balance is: %d",
			expectedBalance, newBalance))
	}
}

//send a message with cost,
//then check that the balance has the expected amount
//the message is charged size based
func TestSendSizeBasedMsg(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	msg := &testSizeBasedMsg{}
	msg.Data = make([]byte, 16)
	rand.Read(msg.Data)
	ctx := context.Background()
	testPeer := newDummyPeer()
	testPeer.Send(ctx, msg)

	//check the new balance
	r, err := rlp.EncodeToBytes(msg)
	if err != nil {
		t.Fatal(err)
	}

	size := uint32(len(r))
	_, price := msg.Price(size)
	expectedBalance := int64(0 - price)
	expectedPrice := uint64(size * 100)
	if price != expectedPrice {
		t.Fatalf("Expected price to be %d, but is %d", expectedPrice, price)
	}
	if swap.balances[testPeer.ID()] != expectedBalance {
		t.Fatalf("Expected balance to be %d, but is %d", expectedBalance, swap.balances[testPeer.ID()])
	}
}

//charge as being the receiver of a receiver-pays message
//as this test for simplicity only sends messages,
//it means that the balance is changed in positive,
//instead of negative as with all other tests
func TestChargeAsReceiver(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	msg := &testChargeRecvMsg{}
	ctx := context.Background()
	testPeer := newDummyPeer()
	testPeer.Send(ctx, msg)

	//check the new balance
	_, price := msg.Price(0)
	if swap.balances[testPeer.ID()] != int64(price) {
		t.Fatalf("Expected balance to be %d, but is %d", price, swap.balances[testPeer.ID()])
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
	testSpec.Hook = protocols.NewAccountingHook(swap, &testPriceOracle{})
	return swap, dir
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

	var balance int64
	err = rpcclient.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("servicenode balance", "balance", balance)

	if balance != 0 {
		t.Fatal("Expected balance to be 0 but it is not")
	}

	dummyPeer1 := newDummyPeer()
	dummyPeer2 := newDummyPeer()
	id1 := dummyPeer1.ID()
	id2 := dummyPeer2.ID()

	fake1 := int64(234)
	fake2 := int64(-100)

	swap.balances[id1] = fake1
	swap.balances[id2] = fake2

	err = rpcclient.Call(&balance, "swap_balanceWithPeer", id1)
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance1", "balance-1", balance)
	if balance != fake1 {
		t.Fatal(fmt.Sprintf("Expected balance %d to be equal to fake balance %d, but it is not", balance, fake1))
	}

	err = rpcclient.Call(&balance, "swap_balanceWithPeer", id2)
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance2", "balance-2", balance)
	if balance != fake2 {
		t.Fatal(fmt.Sprintf("Expected balance %d to be equal to fake balance %d, but it is not", balance, fake2))
	}

	err = rpcclient.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode RPC failed", "err", err)
	}
	log.Debug("balance", "balance", balance)

	fakeSum := fake1 + fake2
	if balance != fakeSum {
		t.Fatal(fmt.Sprintf("Expected balance %d to be equal to sum %d, but it is not", balance, fakeSum))
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
