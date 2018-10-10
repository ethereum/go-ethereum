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
	mrand "math/rand"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/state"
	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

type testPriceOracle struct{}

func (tpo *testPriceOracle) Accountable(msg interface{}) bool {
	_, ok := testSpec.GetCode(msg)
	return ok
}

func (tpo *testPriceOracle) Price(size uint32, msg interface{}) (protocols.EntryDirection, uint64) {
	switch msg := msg.(type) {
	case *testCheapMsg:
		return msg.Price(size)
	case *testSizeBasedMsg:
		return msg.Price(size)
	case *testChargeRecvMsg:
		return msg.Price(size)
	}
	return protocols.ChargeNone, 0
}

var testSpec = &protocols.Spec{
	Name:       "swapTestSpec",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
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
type testCheapMsg struct{}
type testSizeBasedMsg struct {
	Data []byte
}
type testChargeRecvMsg struct{}

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
func TestRepeatedBookings(t *testing.T) {
	//create a test swap account
	swap, testDir := createTestSwap(t)
	defer os.RemoveAll(testDir)

	testPeer := newDummyPeer()
	amount := mrand.Intn(100)
	cnt := 1 + mrand.Intn(10)
	for i := 0; i < cnt; i++ {
		swap.Credit(testPeer.Peer, uint64(amount))
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
		swap.Debit(testPeer2.Peer, uint64(amount))
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
	swap.Credit(testPeer2.Peer, uint64(amount1))
	swap.Credit(testPeer2.Peer, uint64(amount2))
	swap.Debit(testPeer2.Peer, uint64(amount3))

	expectedBalance = expectedBalance + int64(amount1+amount2-amount3)
	realBalance = swap.balances[testPeer2.ID()]

	if expectedBalance != realBalance {
		t.Fatal(fmt.Sprintf("After mixed debits and credits, expected balance to be: %d, but is: %d", expectedBalance, realBalance))
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

type dummyPeer struct {
	*protocols.Peer
	testFunc func(error)
}

func (dp *dummyPeer) Drop(err error) {
	dp.testFunc(err)
}

//creates a dummy protocols.Peer with dummy MsgReadWriter
func newDummyPeer() *dummyPeer {
	id := adapters.RandomNodeConfig().ID
	protoPeer := protocols.NewPeer(p2p.NewPeer(id, "testPeer", nil), &dummyRW{}, testSpec)
	dummy := &dummyPeer{
		Peer: protoPeer,
	}
	return dummy
}
