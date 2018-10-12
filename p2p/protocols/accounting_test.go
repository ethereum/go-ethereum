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

type dummyBalance struct {
	amount int64
	peer   *Peer
}

type dummyPrices struct{}

type perBytesMsg struct {
	Content string
}

type perUnitMsg struct{}
type zeroMsg struct{}

func (d *dummyPrices) Price(msg interface{}) *Price {
	switch msg.(type) {
	case *perBytesMsg:
		return &Price{
			PerByte: true,
			Value:   int64(100),
		}
	case *perUnitMsg:
		return &Price{
			PerByte: false,
			Value:   int64(-99),
		}
	}
	return nil
}

func (d *dummyBalance) Add(amount int64, peer *Peer) error {
	d.amount = amount
	d.peer = peer
	return nil
}

func TestNoHook(t *testing.T) {
	spec := createTestSpec()
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)
	ctx := context.TODO()
	msg := &perBytesMsg{Content: "testBalance"}
	peer.Send(ctx, msg)
	peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
}

func TestSendBalance(t *testing.T) {
	balance := &dummyBalance{}
	prices := &dummyPrices{}

	spec := createTestSpec()
	spec.Hook = NewAccounting(balance, prices)

	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)
	ctx := context.TODO()
	msg := &perBytesMsg{Content: "testBalance"}
	size, _ := rlp.EncodeToBytes(msg)
	peer.Send(ctx, msg)
	if balance.amount != int64((len(size) * 100)) {
		t.Fatalf("Expected price to be %d but is %d", (len(size) * 100), balance.amount)
	}

	msg2 := &perUnitMsg{}
	peer.Send(ctx, msg2)
	if balance.amount != int64(-99) {
		t.Fatalf("Expected price to be %d but is %d", -99, balance.amount)
	}

	balance.amount = 77
	msg3 := &zeroMsg{}
	peer.Send(ctx, msg3)
	if balance.amount != int64(77) {
		t.Fatalf("Expected price to be %d but is %d", 77, balance.amount)
	}
}

func TestReceiveBalance(t *testing.T) {
	balance := &dummyBalance{}
	prices := &dummyPrices{}

	spec := createTestSpec()
	spec.Hook = NewAccounting(balance, prices)

	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	rw := &dummyRW{}
	peer := NewPeer(p, rw, spec)
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
		t.Fatalf("Expected price to be %d but is %d", (len(size) * (-100)), balance.amount)
	}

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
		t.Fatalf("Expected price to be %d but is %d", 99, balance.amount)
	}

	msg3 := &zeroMsg{}
	rw.msg = msg3
	rw.code, _ = spec.GetCode(msg3)
	//need to reset cause no accounting won't overwrite
	balance.amount = -888
	err = peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Expected no error, but got error: %v", err)
	}

	if balance.amount != int64(-888) {
		t.Fatalf("Expected price to be %d but is %d", -888, balance.amount)
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
