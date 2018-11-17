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

package pss

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/swarm/log"
)

type protoCtrl struct {
	C        chan bool
	protocol *Protocol
	run      func(*p2p.Peer, p2p.MsgReadWriter) error
}

// simple ping pong protocol test for the pss devp2p emulation
func TestProtocol(t *testing.T) {
	t.Run("32", testProtocol)
	t.Run("8", testProtocol)
	t.Run("0", testProtocol)
}

func testProtocol(t *testing.T) {

	// address hint size
	var addrsize int64
	paramstring := strings.Split(t.Name(), "/")
	addrsize, _ = strconv.ParseInt(paramstring[1], 10, 0)
	log.Info("protocol test", "addrsize", addrsize)

	topic := PingTopic.String()

	clients, err := setupNetwork(2, false)
	if err != nil {
		t.Fatal(err)
	}
	var loaddrhex string
	err = clients[0].Call(&loaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 1 baseaddr fail: %v", err)
	}
	loaddrhex = loaddrhex[:2+(addrsize*2)]
	var roaddrhex string
	err = clients[1].Call(&roaddrhex, "pss_baseAddr")
	if err != nil {
		t.Fatalf("rpc get node 2 baseaddr fail: %v", err)
	}
	roaddrhex = roaddrhex[:2+(addrsize*2)]
	lnodeinfo := &p2p.NodeInfo{}
	err = clients[0].Call(&lnodeinfo, "admin_nodeInfo")
	if err != nil {
		t.Fatalf("rpc nodeinfo node 11 fail: %v", err)
	}

	var lpubkey string
	err = clients[0].Call(&lpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 1 pubkey fail: %v", err)
	}
	var rpubkey string
	err = clients[1].Call(&rpubkey, "pss_getPublicKey")
	if err != nil {
		t.Fatalf("rpc get node 2 pubkey fail: %v", err)
	}

	time.Sleep(time.Millisecond * 1000) // replace with hive healthy code

	lmsgC := make(chan APIMsg)
	lctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lsub, err := clients[0].Subscribe(lctx, "pss", lmsgC, "receive", topic)
	if err != nil {
		t.Fatal(err)
	}
	defer lsub.Unsubscribe()
	rmsgC := make(chan APIMsg)
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
	if err != nil {
		t.Fatal(err)
	}
	defer rsub.Unsubscribe()

	// set reciprocal public keys
	err = clients[0].Call(nil, "pss_setPeerPublicKey", rpubkey, topic, roaddrhex)
	if err != nil {
		t.Fatal(err)
	}
	err = clients[1].Call(nil, "pss_setPeerPublicKey", lpubkey, topic, loaddrhex)
	if err != nil {
		t.Fatal(err)
	}

	// add right peer's public key as protocol peer on left
	p := p2p.NewPeer(enode.ID{}, fmt.Sprintf("%x", common.FromHex(loaddrhex)), []p2p.Cap{})
	_, err = pssprotocols[lnodeinfo.ID].protocol.AddPeer(p, PingTopic, true, rpubkey)
	if err != nil {
		t.Fatal(err)
	}

	// sends ping asym, checks delivery
	pssprotocols[lnodeinfo.ID].C <- false
	select {
	case <-lmsgC:
		log.Debug("lnode ok")
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	select {
	case <-rmsgC:
		log.Debug("rnode ok")
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}

	// sends ping asym, checks delivery
	pssprotocols[lnodeinfo.ID].C <- false
	select {
	case <-lmsgC:
		log.Debug("lnode ok")
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	select {
	case <-rmsgC:
		log.Debug("rnode ok")
	case cerr := <-lctx.Done():
		t.Fatalf("test message timed out: %v", cerr)
	}
	rw := pssprotocols[lnodeinfo.ID].protocol.pubKeyRWPool[rpubkey]
	pssprotocols[lnodeinfo.ID].protocol.RemovePeer(true, rpubkey)
	if err := rw.WriteMsg(p2p.Msg{
		Size:    3,
		Payload: bytes.NewReader([]byte("foo")),
	}); err == nil {
		t.Fatalf("expected error on write")
	}
}
