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

package protocols

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
)

//dummy Balance implementation
type dummyBalance struct {
	amount int64
	peer   *Peer
}

//dummy Prices implementation
type dummyPrices struct{}

//a dummy message which needs size based accounting
type perBytesMsg struct {
	Content string
}

//a dummy message which is paid for per unit
type perUnitMsg struct{}

//a dummy message which doesn't need any payment
type zeroMsg struct{}

//return the price for the defined messages
func (d *dummyPrices) Price(msg interface{}) *Price {
	switch msg.(type) {
	//size based message cost, sender pays
	case *perBytesMsg:
		return &Price{
			PerByte: true,
			Value:   int64(100),
		}
		//unitary cost, receiver pays
	case *perUnitMsg:
		return &Price{
			PerByte: false,
			Value:   int64(-99),
		}
	}
	return nil
}

//dummy accounting implementation, only stores values for later check
func (d *dummyBalance) Add(amount int64, peer *Peer) error {
	d.amount = amount
	d.peer = peer
	return nil
}

//We need to test that if the hook is not defined, then message infrastructure
//(send,receive) still works
func TestNoHook(t *testing.T) {
	//create a test spec
	spec := createTestSpec()
	//a random node
	id := adapters.RandomNodeConfig().ID
	//a peer
	p := p2p.NewPeer(id, "testPeer", nil)
	rw := &dummyRW{}
	peer := NewPeer(p, rw, spec)
	ctx := context.TODO()
	msg := &perBytesMsg{Content: "testBalance"}
	//send a message
	peer.Send(ctx, msg)
	//simulate receiving a message
	rw.msg = msg
	peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
	//all should just work and not result in any error
}

//lowest level unit test
func TestSend(t *testing.T) {
	//create instances
	balance := &dummyBalance{}
	prices := &dummyPrices{}
	//create the spec
	spec := createTestSpec()
	//create the accounting hook for the spec
	spec.Hook = NewAccounting(balance, prices)
	//create a peer
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)
	msg := &perBytesMsg{Content: "testBalance"}

	size, _ := rlp.EncodeToBytes(msg)
	spec.Hook.Send(peer, uint32(len(size)), msg)
	if balance.amount != int64(len(size)*100) {
		t.Fatalf("Expected balance to be %d but is %d", (len(size) * 100), balance.amount)
	}

	//send a unitary, receiver pays message
	msg2 := &perUnitMsg{}
	spec.Hook.Send(peer, 0, msg2)
	//check that the balance is actually the expected
	if balance.amount != int64(-99) {
		t.Fatalf("Expected balance to be %d but is %d", -99, balance.amount)
	}

	//set arbitrary balance
	balance.amount = 77
	//send a non accounted message, balance should not change
	msg3 := &zeroMsg{}
	spec.Hook.Send(peer, 0, msg3)
	if balance.amount != int64(77) {
		t.Fatalf("Expected balance to be %d but is %d", 77, balance.amount)
	}
}

//lowest level unit test
func TestReceive(t *testing.T) {
	//create instances
	balance := &dummyBalance{}
	prices := &dummyPrices{}
	//create the spec
	spec := createTestSpec()
	//create the accounting hook for the spec
	spec.Hook = NewAccounting(balance, prices)
	//create a peer
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)
	msg := &perBytesMsg{Content: "testBalance"}

	size, _ := rlp.EncodeToBytes(msg)
	spec.Hook.Receive(peer, uint32(len(size)), msg)
	if balance.amount != int64(len(size)*-100) {
		t.Fatalf("Expected balance to be %d but is %d", (len(size) * 100), balance.amount)
	}

	//send a unitary, receiver pays message
	msg2 := &perUnitMsg{}
	spec.Hook.Receive(peer, 0, msg2)
	//check that the balance is actually the expected
	if balance.amount != int64(99) {
		t.Fatalf("Expected balance to be %d but is %d", -99, balance.amount)
	}

	//set arbitrary balance
	balance.amount = 77
	//send a non accounted message, balance should not change
	msg3 := &zeroMsg{}
	spec.Hook.Receive(peer, 0, msg3)
	if balance.amount != int64(77) {
		t.Fatalf("Expected balance to be %d but is %d", 77, balance.amount)
	}
}

//Test sending with a peer, higher level
func TestSendWithPeer(t *testing.T) {
	//create instances
	balance := &dummyBalance{}
	prices := &dummyPrices{}
	//create the spec
	spec := createTestSpec()
	//create the accounting hook for the spec
	spec.Hook = NewAccounting(balance, prices)
	//create a peer
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)
	ctx := context.TODO()
	msg := &perBytesMsg{Content: "testBalance"}
	size, _ := rlp.EncodeToBytes(msg)
	//send a size based message
	peer.Send(ctx, msg)
	//check that the balance is actually the expected
	if balance.amount != int64((len(size) * 100)) {
		t.Fatalf("Expected balance to be %d but is %d", (len(size) * 100), balance.amount)
	}

	//send a unitary, receiver pays message
	msg2 := &perUnitMsg{}
	peer.Send(ctx, msg2)
	//check that the balance is actually the expected
	if balance.amount != int64(-99) {
		t.Fatalf("Expected balance to be %d but is %d", -99, balance.amount)
	}

	//set arbitrary balance
	balance.amount = 77
	//send a non accounted message, balance should not change
	msg3 := &zeroMsg{}
	peer.Send(ctx, msg3)
	if balance.amount != int64(77) {
		t.Fatalf("Expected balance to be %d but is %d", 77, balance.amount)
	}
}

//Test receiving with a peer, higher level
func TestReceiveWithPeer(t *testing.T) {
	//create instances
	balance := &dummyBalance{}
	prices := &dummyPrices{}
	//create the spec
	spec := createTestSpec()
	//create the accounting hook for the spec
	spec.Hook = NewAccounting(balance, prices)
	//create a peer
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	rw := &dummyRW{}
	peer := NewPeer(p, rw, spec)

	//simulate receiving a size based, sender-pays message
	msg := &perBytesMsg{Content: "testBalance"}
	size, _ := rlp.EncodeToBytes(msg)
	rw.msg = msg
	rw.code, _ = spec.GetCode(msg)
	err := peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, but got error: %v", err)
	}
	if balance.amount != int64((len(size) * (-100))) {
		t.Fatalf("Expected balance to be %d but is %d", (len(size) * (-100)), balance.amount)
	}

	//simulate receiving a size based, sender-pays message
	msg2 := &perUnitMsg{}
	rw.msg = msg2
	rw.code, _ = spec.GetCode(msg2)
	err = peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, but got error: %v", err)
	}
	if balance.amount != int64(99) {
		t.Fatalf("Expected balance to be %d but is %d", 99, balance.amount)
	}

	//set arbitrary balance an
	msg3 := &zeroMsg{}
	rw.msg = msg3
	rw.code, _ = spec.GetCode(msg3)
	//need to reset cause no accounting won't overwrite
	balance.amount = -888
	//simulate receiving a non accounted message, balance should not change
	err = peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, but got error: %v", err)
	}
	if balance.amount != int64(-888) {
		t.Fatalf("Expected balance to be %d but is %d", -888, balance.amount)
	}
}

//dummy implementation of a MsgReadWriter
//this allows for quick and easy unit tests without
//having to build up the complete protocol
type dummyRW struct {
	msg  interface{}
	size uint32
	code uint64
}

func (d *dummyRW) WriteMsg(msg p2p.Msg) error {
	return nil
}

func (d *dummyRW) ReadMsg() (p2p.Msg, error) {
	enc := bytes.NewReader(d.getDummyMsg())
	return p2p.Msg{
		Code:       d.code,
		Size:       d.size,
		Payload:    enc,
		ReceivedAt: time.Now(),
	}, nil
}

func (d *dummyRW) getDummyMsg() []byte {
	r, _ := rlp.EncodeToBytes(d.msg)
	var b bytes.Buffer
	wmsg := WrappedMsg{
		Context: b.Bytes(),
		Size:    uint32(len(r)),
		Payload: r,
	}
	rr, _ := rlp.EncodeToBytes(wmsg)
	d.size = uint32(len(rr))
	return rr
}

//create a test spec
func createTestSpec() *Spec {
	spec := &Spec{
		Name:       "test",
		Version:    42,
		MaxMsgSize: 10 * 1024,
		Messages: []interface{}{
			perBytesMsg{},
			perUnitMsg{},
			zeroMsg{},
		},
	}
	return spec
}
