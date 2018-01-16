// +build go1.8

package pss

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

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

	clients, err := setupNetwork(2)
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
	defer lsub.Unsubscribe()
	rmsgC := make(chan APIMsg)
	rctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	rsub, err := clients[1].Subscribe(rctx, "pss", rmsgC, "receive", topic)
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
	nid, _ := discover.HexID("0x00") // this hack is needed to satisfy the p2p method
	p := p2p.NewPeer(nid, fmt.Sprintf("%x", common.FromHex(loaddrhex)), []p2p.Cap{})
	_, err = pssprotocols[lnodeinfo.ID].protocol.AddPeer(p, pssprotocols[lnodeinfo.ID].run, PingTopic, true, rpubkey)
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

}
