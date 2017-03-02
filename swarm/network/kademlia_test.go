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
	"testing"
	"time"

	// "github.com/ethereum/go-ethereum/logger"
	// "github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pot"
)

func testKadPeerAddr(s string) *peerAddr {
	a := pot.NewHashAddress(s).Bytes()
	return &peerAddr{OAddr: a, UAddr: a}
}

func testKadPeer(s string) *bzzPeer {
	return &bzzPeer{peerAddr: testKadPeerAddr(s)}
}

func testStr(a PeerAddr) string {
	s, _ := a.(*KadPeer)
	if s == nil {
		return "<nil>"
	}
	// return s.String()
	// return fmt.Sprintf("%06x", s.OverlayAddr())
	return pot.NewHashAddressFromBytes(a.(*KadPeer).PeerAddr.OverlayAddr()).Bin()[:6]
	// return a.(*KadPeer).String()[:6] //.HashAddress.String()[:6]
	// return a.(*KadPeer).String() //[:6] //.HashAddress.String()[:6]
	// return "wtf"
}

type testKademlia struct {
	*Kademlia
}

func newTestKademlia(b string) *testKademlia {
	params := NewKadParams()
	params.MinBinSize = 1
	params.MinProxBinSize = 2
	base := pot.NewHashAddress(b).Bytes()
	return &testKademlia{NewKademlia(base, params)}
}

func (k *testKademlia) On(ons ...string) *testKademlia {
	for _, s := range ons {
		k.Kademlia.On(testKadPeer(s))
	}
	return k
}

func (k *testKademlia) Off(offs ...string) *testKademlia {
	for _, s := range offs {
		k.Kademlia.Off(testKadPeer(s))
	}
	return k
}

func (k *testKademlia) Register(regs ...string) *testKademlia {
	var ps []PeerAddr
	for _, s := range regs {
		ps = append(ps, PeerAddr(testKadPeerAddr(s)))
	}
	k.Kademlia.Register(ps...)
	return k
}

func testSuggestPeer(t *testing.T, k *testKademlia, expAddr string, expPo int, expWant bool) error {
	addr, o, want := k.SuggestPeer()
	if testStr(addr) != expAddr {
		return fmt.Errorf("incorrect peer address suggested. expected %v, got %v", expAddr, testStr(addr))
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

func TestKademliaHiveString(t *testing.T) {
	k := newTestKademlia("000000").On("010000", "001000").Register("100000", "100001")
	h := k.String()
	expH := "\n=========================================================================\nMon Feb 27 12:10:28 UTC 2017 KΛÐΞMLIΛ hive: queen's address: 000000\npopulation: 2 (4), ProxBinSize: 2, MinBinSize: 1, MaxBinSize: 4\n============ PROX LIMIT: 0 ==========================================\n000  0                                           |  2 840000 800000\n001  1 400000                                    |  1 400000\n002  1 200000                                    |  1 200000\n003  0                                           |  0\n004  0                                           |  0\n005  0                                           |  0\n006  0                                           |  0\n007  0                                           |  0\n========================================================================="
	if expH[100:] != h[100:] {
		t.Fatalf("incorrect hive output. expected %v, got %v", expH, h)
	}
}
