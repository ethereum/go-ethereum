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
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	wouldHaveAccounted = errors.New("ignore this error")
)

type dummy struct {
	content string
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
		Size:       5,
		Payload:    bytes.NewReader(getDummyMsg()),
		ReceivedAt: time.Now(),
	}, nil
}

func getDummyMsg() []byte {
	msg := &dummy{content: "test"}
	r, _ := rlp.EncodeToBytes(msg)

	var b bytes.Buffer
	wmsg := WrappedMsg{
		Context: b.Bytes(),
		Size:    uint32(len(r)),
		Payload: r,
	}

	rr, _ := rlp.EncodeToBytes(wmsg)
	return rr
}

func createTestSpec() *Spec {
	spec := &Spec{
		Name:       "test",
		Version:    42,
		MaxMsgSize: 10 * 1024,
		Messages: []interface{}{
			dummy{},
		},
	}
	return spec
}

type dummyBalanceMgr struct{}
type dummyPriceOracle struct{}

func (d *dummyPriceOracle) Price(uint32, interface{}) (EntryDirection, uint64) {
	return ChargeSender, 99
}

func (d *dummyBalanceMgr) Credit(peer *Peer, amount uint64) error {
	return wouldHaveAccounted
}

func (d *dummyBalanceMgr) Debit(peer *Peer, amount uint64) error {
	return wouldHaveAccounted
}

// Test that passing a nil hook doesn't affect sending
func TestProtocolNilHook(t *testing.T) {
	spec := createTestSpec()
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)

	peer.Send(context.Background(), dummy{})
	peer.handleIncoming(func(ctx context.Context, msg interface{}) error {
		return nil
	})
}

func TestProtocolHook(t *testing.T) {
	spec := createTestSpec()
	spec.Hook = NewAccountingHook(&dummyBalanceMgr{}, &dummyPriceOracle{})
	id := adapters.RandomNodeConfig().ID
	p := p2p.NewPeer(id, "testPeer", nil)
	peer := NewPeer(p, &dummyRW{}, spec)

	err := peer.Send(context.Background(), dummy{})
	if err == nil || err != wouldHaveAccounted {
		t.Fatal("Expected fake accounting to happen, but didn't")
	}

	err = peer.handleIncoming(nil)
	if err == nil || err != wouldHaveAccounted {
		t.Fatal("Expected fake accounting to happen, but didn't")
	}
}
