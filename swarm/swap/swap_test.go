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

	//create a dummy peer
	testPeer := newDummyPeer()
	sp := NewSwapPeer(testPeer, swap)
	sp.handlerFunc = dummyMsgHandler

	//send the message
	ctx := context.Background()
	err := sp.Send(ctx, &testExceedsPayAtMsg{})
	if err != nil {
		t.Fatal("Unecpected error on sending message", "err", err)
	}

	//check that a cheque is present
	cheques := sp.swapAccount.chequeManager.openDebitCheques[sp.ID()]
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
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	testPeer := newDummyPeer()
	sp := NewSwapPeer(testPeer, swap)
	sp.handlerFunc = dummyMsgHandler

	ctx := context.Background()
	err := sp.Send(ctx, &testExceedsDropAtMsg{})
	if err != ErrInsufficientFunds {
		t.Fatal("Expected test to fail with insufficient funds, but it didn't")
	}
}

//send a message with cost,
//then check that the balance has the expected amount
func TestSendCheapMessage(t *testing.T) {
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	testPeer := newDummyPeer()
	sp := NewSwapPeer(testPeer, swap)
	//set an arbitrary test balance value
	testBalance := big.NewInt(1234567890)
	sp.balance = testBalance

	//send the message
	msg := &testCheapMsg{}
	ctx := context.Background()
	err := sp.Send(ctx, msg)
	if err != nil {
		t.Fatal("Unexpected error sending message")
	}

	//check the new balance
	if sp.balance.Cmp(testBalance.Sub(testBalance, msg.Price())) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			testBalance.Sub(testBalance, msg.Price()).String(), sp.balance.String()))
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

	//create the dummy p2p protocol Peer
	testPeer := newDummyPeer()
	//create the "source" swap peer
	sp := NewSwapPeer(testPeer, swap)
	//create a reference an arbitrary balance
	testBalance := big.NewInt(1234567890)
	//assign the same value to the peer
	sp.balance = big.NewInt(1234567890)

	//send a message, should trigger saving to stateStore
	msg := &testCheapMsg{}
	ctx := context.Background()
	err := sp.Send(ctx, msg)
	if err != nil {
		log.Error(err.Error())
		t.Fatal("Unexpected error sending message")
	}

	//create a new peer with same underlying protocols.Peer
	//this will try to load the balance from the stateStore,
	//as it is the same discover.NodeID
	sp2 := NewSwapPeer(testPeer, swap)

	//compare the balances
	expectedBalance := &big.Int{}
	expectedBalance.Sub(testBalance, msg.Price())
	if sp2.balance.Cmp(expectedBalance) != 0 {
		t.Fatal(fmt.Sprintf("Unexpected balance value after sending cheap message test. Expected balance: %s, balance is: %s",
			expectedBalance.String(), sp2.balance.String()))
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
	swap, err3 := New(stateStore)
	if err3 != nil {
		t.Fatal(err3)
	}
	return swap, dir
}

//dummy message handler (needed or we will have a panic in the accounting)
func dummyMsgHandler(ctx context.Context, msg interface{}) error {
	return nil
}

//tests some basic things over RPC
func TestSwapRPC(t *testing.T) {
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	// create the two nodes
	stack_one, err := newServiceNode(p2pPort, 0, 0)
	if err != nil {
		t.Fatal("Create servicenode #1 fail", "err", err)
	}

	instance := NewSwapProtocol(swap)
	// wrapper function for servicenode to start the service
	swapsvc := func(ctx *node.ServiceContext) (node.Service, error) {
		return &API{
			Protocol: instance,
		}, nil
	}

	// register adds the service to the services the servicenode starts when started
	err = stack_one.Register(swapsvc)
	if err != nil {
		t.Fatal("Register service in servicenode #1 fail", "err", err)
	}
	// start the nodes
	err = stack_one.Start()
	if err != nil {
		t.Fatal("servicenode #1 start failed", "err", err)
	}

	// connect to the servicenode RPCs
	rpcclient_one, err := rpc.Dial(filepath.Join(stack_one.DataDir(), ipcpath))
	if err != nil {
		t.Fatal("connect to servicenode #1 IPC fail", "err", err)
	}
	defer os.RemoveAll(stack_one.DataDir())

	var balance *big.Int
	err = rpcclient_one.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode #1 RPC failed", "err", err)
	}
	log.Debug("servicenode #1 balance", "balance-1", balance)

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

	swap.peers[id1] = NewSwapPeer(dummyPeer1, swap)
	swap.peers[id2] = NewSwapPeer(dummyPeer2, swap)
	swap.peers[id1].balance = fakeBalance1
	swap.peers[id2].balance = fakeBalance2

	err = rpcclient_one.Call(&balance, "swap_balanceWithPeer", id1)
	if err != nil {
		t.Fatal("servicenode #1 RPC failed", "err", err)
	}
	log.Debug("balance1", "balance-1", balance)
	if balance.Cmp(fakeBalance1) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to fake balance %s, but it is not", balance.String(), fakeBalance1.String()))
	}

	err = rpcclient_one.Call(&balance, "swap_balanceWithPeer", id2)
	if err != nil {
		t.Fatal("servicenode #1 RPC failed", "err", err)
	}
	log.Debug("balance2", "balance-2", balance)
	if balance.Cmp(fakeBalance2) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to fake balance %s, but it is not", balance.String(), fakeBalance2.String()))
	}

	err = rpcclient_one.Call(&balance, "swap_balance")
	if err != nil {
		t.Fatal("servicenode #1 RPC failed", "err", err)
	}
	log.Debug("balance", "balance", balance)

	fakeSum := big.NewInt(fake1 + fake2)
	if balance.Cmp(fakeSum) != 0 {
		t.Fatal(fmt.Sprintf("Expected balance %s to be equal to sum %s, but it is not", balance.String(), fakeSum.String()))
	}
}

//creates a dummy protocols.Peer with dummy MsgReadWriter
func newDummyPeer() *protocols.Peer {
	id := adapters.RandomNodeConfig().ID
	return protocols.NewPeer(p2p.NewPeer(id, "testPeer", nil), &dummyRW{}, testSpec)
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
