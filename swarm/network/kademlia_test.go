// Copyright 2017 The go-ethereum Authors
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
package network

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/pot"
)

func testKadPeerAddr(s string) *bzzAddr {
	a := pot.NewHashAddress(s).Bytes()
	return &bzzAddr{OAddr: a, UAddr: a}
}

type testDropPeer struct {
	Peer
	dropc chan error
}

type testDiscPeer struct {
	*testDropPeer
	lock          *sync.Mutex
	notifications map[string]uint8
}

type testPeerNotification struct {
	rec  string
	addr string
	po   uint8
}

type testProxNotification struct {
	rec string
	po  uint8
}

type dropError struct {
	error
	addr string
}

func (self *testDropPeer) Drop(err error) {
	err2 := &dropError{err, overlayStr(self)}
	self.dropc <- err2
}

func (self *testDiscPeer) NotifyProx(po uint8) error {
	key := overlayStr(self)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.notifications[key] = po
	return nil
}

func (self *testDiscPeer) NotifyPeer(p OverlayPeer, po uint8) error {
	key := overlayStr(self)
	key += overlayStr(p)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.notifications[key] = po
	return nil
}

type testKademlia struct {
	*Kademlia
	Discovery     bool
	dropc         chan error
	lock          *sync.Mutex
	notifications map[string]uint8
}

func newTestKademlia(b string) *testKademlia {
	params := NewKadParams()
	params.MinBinSize = 1
	params.MinProxBinSize = 2
	base := pot.NewHashAddress(b).Bytes()
	return &testKademlia{
		NewKademlia(base, params),
		false,
		make(chan error),
		&sync.Mutex{},
		make(map[string]uint8),
	}
}

func (k *testKademlia) newTestKadPeer(s string) Peer {
	dp := &testDropPeer{&bzzPeer{bzzAddr: testKadPeerAddr(s)}, k.dropc}
	if k.Discovery {
		return Peer(&testDiscPeer{dp, k.lock, k.notifications})
	}
	return Peer(dp)
}

func overlayStr(a OverlayPeer) string {
	// log.Error(fmt.Sprintf("PeerAddr: %v (%T)", a, a))
	// if a == (*KadPeer)(nil) || a == (*testDiscPeer)(nil) || a == (*bzzPeer)(nil) || a == nil {
	// 	return "<nil>"
	// }
	// var p Peer
	// s, ok := a.(*KadPeer)
	// if ok {
	// 	p = s.Peer
	// } else {
	// 	p = a.(*testDiscPeer).Peer
	// }
	// log.Error(fmt.Sprintf("PeerAddr: %v (%T)", p, p))
	// if p == (Peer)(nil) || p == (*testDiscPeer)(nil) || p == (*bzzPeer)(nil) {
	// 	return "<nil>"
	// }
	// return pot.NewHashAddressFromBytes(p.OverlayAddr()).Bin()[:6]
	// if a == nil {
	// 	return "<nil>"
	// }
	// k, ok := a.(*KadPeer)
	// if ok && k.Peer != nil {
	// 	return pot.ToBin(a.(*KadPeer).Peer.Over())[:6]
	// }
	// return pot.ToBin(a.Over())[:6]
	return pot.ToBin(a.Address())
}

func (k *testKademlia) On(ons ...string) *testKademlia {
	for _, s := range ons {
		p := k.newTestKadPeer(s)
		k.Kademlia.On(p)
	}
	return k
}

func (k *testKademlia) Off(offs ...string) *testKademlia {
	for _, s := range offs {
		k.Kademlia.Off(k.newTestKadPeer(s).(OverlayConn))
	}

	return k
}

func (k *testKademlia) Register(regs ...string) *testKademlia {
	var ps []Addr
	for _, s := range regs {
		ps = append(ps, Addr(testKadPeerAddr(s)))
	}
	k.Kademlia.Register(ps...)
	return k
}

func testSuggestPeer(t *testing.T, k *testKademlia, expAddr string, expPo int, expWant bool) error {
	addr, o, want := k.SuggestPeer()
	if overlayStr(addr) != expAddr {
		return fmt.Errorf("incorrect peer address suggested. expected %v, got %v", expAddr, overlayStr(addr))
	}
	if o != expPo {
		return fmt.Errorf("incorrect prox order suggested. expected %v, got %v", expPo, o)
	}
	if want != expWant {
		return fmt.Errorf("expected SuggestPeer to want peers: %v", expWant)
	}
	return nil
}

func TestSuggestPeerFindPeers(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("000000").On("001000")
	// k.MinProxBinSize = 2
	// k.MinBinSize = 2
	err := testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 2 row gap, saturated proxbin, no callables -> want PO 0
	k.On("000100")
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 1 row gap (1 less), saturated proxbin, no callables -> want PO 1
	k.On("100000")
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// no gap (1 less), saturated proxbin, no callables -> do not want more
	k.On("010000", "001001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// oversaturated proxbin, > do not want more
	k.On("001001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// reintroduce gap, disconnected peer callable
	k.Off("010000")
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// second time disconnected peer not callable
	// with reasonably set Interval
	// err = testSuggestPeer(t, k, "010000", 2, true)
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// on and off again, peer callable again
	k.On("010000")
	k.Off("010000")
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("010000")
	k.Off("010000")
	// PO1 disconnects
	// new closer peer appears, it is immediately wanted
	// k.Off("010000")
	k.Register("000101")
	err = testSuggestPeer(t, k, "000101", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// second time, gap filling
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("010000")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 2
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("100001")
	k.On("010001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 3
	k.On("100010")
	k.On("010010")
	err = testSuggestPeer(t, k, "<nil>", 2, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("001010")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestSuggestPeerRetries(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("000000")
	cycle := 50 * time.Millisecond
	k.RetryInterval = int(cycle)
	k.MaxRetries = 3
	k.RetryExponent = 3
	k.Register("010000")
	k.On("000001", "000010")
	err := testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "010000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestPruning(t *testing.T) {
	k := newTestKademlia("000000")
	k.On("100000", "110000", "101000", "100100", "100010")
	k.On("010000", "011000", "010100", "010010", "010001")
	k.On("001000", "001100", "001010", "001001")
	k.MaxBinSize = 4
	k.MinBinSize = 3
	prune := make(chan time.Time)
	defer close(prune)
	k.Prune((<-chan time.Time)(prune))
	prune <- time.Now()
	errc := make(chan error)
	timeout := time.NewTimer(1000 * time.Millisecond)
	n := 0
	dropped := make(map[string]error)
	go func() {
		for e := range k.dropc {
			err := e.(*dropError)
			dropped[err.addr] = err.error
			n++
			if n == 4 {
				break
			}
		}
		close(errc)
	}()
	select {
	case <-errc:
	case <-timeout.C:
		t.Fatalf("timeout waiting for 4 peers to be dropped")
	}
	// TODO: this is now based on just taking the first 2 peers
	// in order of connecting
	expDropped := []string{
		"101000",
		"110000",
		"010100",
		"011000",
	}
	for _, addr := range expDropped {
		err := dropped[addr]
		if err == nil {
			t.Fatalf("expected peer %v to be dropped", addr)
		}
		if err.Error() != "bucket full" {
			t.Fatalf("incorrect error. expected %v, got %v", "bucket full", err)
		}
	}
}

func TestKademliaHiveString(t *testing.T) {
	k := newTestKademlia("000000").On("010000", "001000").Register("100000", "100001")
	h := k.String()
	expH := "\n=========================================================================\nMon Feb 27 12:10:28 UTC 2017 KΛÐΞMLIΛ hive: queen's address: 000000\npopulation: 2 (4), MinProxBinSize: 2, MinBinSize: 1, MaxBinSize: 4\n============ PROX LIMIT: 0 ==========================================\n000  0                                           |  2 840000 800000\n001  1 400000                                    |  1 400000\n002  1 200000                                    |  1 200000\n003  0                                           |  0\n004  0                                           |  0\n005  0                                           |  0\n006  0                                           |  0\n007  0                                           |  0\n========================================================================="
	if expH[100:] != h[100:] {
		t.Fatalf("incorrect hive output. expected %v, got %v", expH, h)
	}
}

func (self *testKademlia) checkNotifications(npeers []*testPeerNotification, nprox []*testProxNotification) error {
	for _, pn := range npeers {
		key := pn.rec + pn.addr
		po, found := self.notifications[key]
		if !found || pn.po != po {
			return fmt.Errorf("%v, expected to have notified %v about peer %v (%v)", key, pn.rec, pn.addr, pn.po)
		}
		delete(self.notifications, key)
	}
	for _, pn := range nprox {
		key := pn.rec
		po, found := self.notifications[key]
		if !found || pn.po != po {
			return fmt.Errorf("expected to have notified %v about new prox limit %v", pn.rec, pn.po)
		}
		delete(self.notifications, key)
	}
	if len(self.notifications) > 0 {
		return fmt.Errorf("%v unexpected notifications", len(self.notifications))
	}
	return nil
}

func TestNotifications(t *testing.T) {
	k := newTestKademlia("000000")
	k.Discovery = true
	k.MinProxBinSize = 3
	k.On("010000", "001000")
	time.Sleep(100 * time.Millisecond)
	err := k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"010000", "001000", 1},
		},
		[]*testProxNotification{
			&testProxNotification{"001000", 0},
			&testProxNotification{"010000", 0},
		},
	)
	if err != nil {
		t.Fatal(err.Error())
	}
	k = k.On("100000")
	time.Sleep(100 * time.Millisecond)

	k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"010000", "100000", 0},
			&testPeerNotification{"001000", "100000", 0},
		},
		[]*testProxNotification{
			&testProxNotification{"100000", 0},
		},
	)

	k = k.On("010001")
	time.Sleep(100 * time.Millisecond)

	k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"010000", "010001", 5},
			&testPeerNotification{"001000", "010001", 1},
			&testPeerNotification{"100000", "010001", 0},
		},
		[]*testProxNotification{
			&testProxNotification{"100000", 0},
			&testProxNotification{"010000", 0},
			&testProxNotification{"010001", 0},
			&testProxNotification{"001000", 0},
		},
	)
}
