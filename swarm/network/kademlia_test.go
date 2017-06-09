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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pot"
)

func init() {
	h := log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	// h := log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	// h := log.CallerFileHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

func testKadPeerAddr(s string) *bzzAddr {
	a := pot.NewAddressFromString(s)
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

type testDepthNotification struct {
	rec string
	po  uint8
}

type dropError struct {
	error
	addr string
}

func (self *testDropPeer) Drop(err error) {
	err2 := &dropError{err, binStr(self)}
	self.dropc <- err2
}

func (self *testDiscPeer) NotifyDepth(po uint8) error {
	key := binStr(self)
	self.lock.Lock()
	defer self.lock.Unlock()
	self.notifications[key] = po
	return nil
}

func (self *testDiscPeer) NotifyPeer(p OverlayAddr, po uint8) error {
	key := binStr(self)
	key += binStr(p)
	self.lock.Lock()
	defer self.lock.Unlock()
	log.Trace(fmt.Sprintf("key %v=>%v", key, po))
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
	base := pot.NewAddressFromString(b)
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

func (k *testKademlia) On(ons ...string) *testKademlia {
	for _, s := range ons {
		k.Kademlia.On(k.newTestKadPeer(s).(OverlayConn))
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
	ch := make(chan OverlayAddr)
	go func() {
		defer close(ch)
		for _, s := range regs {
			ch <- testKadPeerAddr(s)
		}
	}()
	err := k.Kademlia.Register(ch)
	log.Trace(fmt.Sprintf("register %v addresses: %v", len(regs), err))

	return k
}

func testSuggestPeer(t *testing.T, k *testKademlia, expAddr string, expPo int, expWant bool) error {
	addr, o, want := k.SuggestPeer()
	if binStr(addr) != expAddr {
		return fmt.Errorf("incorrect peer address suggested. expected %v, got %v", expAddr, binStr(addr))
	}
	if o != expPo {
		return fmt.Errorf("incorrect prox order suggested. expected %v, got %v", expPo, o)
	}
	if want != expWant {
		return fmt.Errorf("expected SuggestPeer to want peers: %v", expWant)
	}
	// t.Logf("%v", k)
	return nil
}

func binStr(a OverlayPeer) string {
	if a == nil {
		return "<nil>"
	}
	return pot.ToBin(a.Address())[:8]
}

func TestSuggestPeerFindPeers(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000").On("00100000")
	err := testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 2 row gap, saturated proxbin, no callables -> want PO 0
	k.On("00010000")
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 1 row gap (1 less), saturated proxbin, no callables -> want PO 1
	k.On("10000000")
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// no gap (1 less), saturated proxbin, no callables -> do not want more
	k.On("01000000", "00100001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// oversaturated proxbin, > do not want more
	k.On("00100001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// reintroduce gap, disconnected peer callable
	log.Info(k.String())
	k.Off("01000000")
	log.Info(k.String())
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// second time disconnected peer not callable
	// with reasonably set Interval
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// on and off again, peer callable again
	k.On("01000000")
	k.Off("01000000")
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000000")
	// new closer peer appears, it is immediately wanted
	k.Register("00010001")
	err = testSuggestPeer(t, k, "00010001", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// PO1 disconnects
	k.On("00010001")
	log.Info(k.String())
	k.Off("01000000")
	log.Info(k.String())
	// second time, gap filling
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000000")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 2
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.Register("01000001")
	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("10000001")
	log.Trace("Kad:\n%v", k.String())
	err = testSuggestPeer(t, k, "01000001", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("10000001")
	k.On("01000001")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.MinBinSize = 3
	k.Register("10000010")
	err = testSuggestPeer(t, k, "10000010", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("10000010")
	err = testSuggestPeer(t, k, "<nil>", 1, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000010")
	err = testSuggestPeer(t, k, "<nil>", 2, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("00100010")
	err = testSuggestPeer(t, k, "<nil>", 3, true)
	if err != nil {
		log.Trace("Kad:\n%v", k.String())
		t.Fatal(err.Error())
	}

	k.On("00010010")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		log.Trace("Kad:\n%v", k.String())
		t.Fatal(err.Error())
	}

}

func TestSuggestPeerRetries(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000")
	cycle := 50 * time.Millisecond
	k.RetryInterval = int(cycle)
	k.MaxRetries = 3
	k.RetryExponent = 3
	k.Register("01000000")
	k.On("00000001", "00000010")
	err := testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	// cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, true)
	if err != nil {
		t.Fatal(err.Error())
	}

	cycle *= time.Duration(k.RetryExponent)
	time.Sleep(cycle)
	err = testSuggestPeer(t, k, "01000000", 0, false)
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
	k := newTestKademlia("00000000")
	k.On("10000000", "11000000", "10100000", "10010000", "10000010")
	k.On("01000000", "01100000", "01000100", "01000010", "01000001")
	k.On("00100000", "00110000", "00100010", "00100001")
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
		"10100000",
		"11000000",
		"01000100",
		"01100000",
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
	k := newTestKademlia("00000000").On("01000000", "00100000").Register("10000000", "10000001")
	h := k.String()
	expH := "\n=========================================================================\nMon Feb 27 12:10:28 UTC 2017 KΛÐΞMLIΛ hive: queen's address: 000000\npopulation: 2 (4), MinProxBinSize: 2, MinBinSize: 1, MaxBinSize: 4\n============ PROX LIMIT: 0 ==========================================\n000  0                                           |  2 840000 800000\n001  1 400000                                    |  1 400000\n002  1 200000                                    |  1 200000\n003  0                                           |  0\n004  0                                           |  0\n005  0                                           |  0\n006  0                                           |  0\n007  0                                           |  0\n========================================================================="
	if expH[100:] != h[100:] {
		t.Fatalf("incorrect hive output. expected %v, got %v", expH, h)
	}
}

func (self *testKademlia) checkNotifications(npeers []*testPeerNotification, nprox []*testDepthNotification) error {
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
	k := newTestKademlia("00000000")
	k.Discovery = true
	k.MinProxBinSize = 3
	k.On("01000000", "00100000")
	time.Sleep(1000 * time.Millisecond)
	err := k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"01000000", "00100000", 1},
		},
		[]*testDepthNotification{
			&testDepthNotification{"00100000", 0},
			&testDepthNotification{"01000000", 0},
		},
	)
	if err != nil {
		t.Fatal(err.Error())
	}
	k = k.On("10000000")
	time.Sleep(100 * time.Millisecond)

	k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"01000000", "10000000", 0},
			&testPeerNotification{"00100000", "10000000", 0},
		},
		[]*testDepthNotification{
			&testDepthNotification{"10000000", 0},
		},
	)

	k = k.On("01000001")
	time.Sleep(100 * time.Millisecond)

	k.checkNotifications(
		[]*testPeerNotification{
			&testPeerNotification{"01000000", "01000001", 5},
			&testPeerNotification{"00100000", "01000001", 1},
			&testPeerNotification{"10000000", "01000001", 0},
		},
		[]*testDepthNotification{
			&testDepthNotification{"10000000", 0},
			&testDepthNotification{"01000000", 0},
			&testDepthNotification{"01000001", 0},
			&testDepthNotification{"00100000", 0},
		},
	)
}
