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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pot"
)

func init() {
	h := log.LvlFilterHandler(log.LvlWarn, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	log.Root().SetHandler(h)
}

func testKadPeerAddr(s string) *BzzAddr {
	a := pot.NewAddressFromString(s)
	return &BzzAddr{OAddr: a, UAddr: a}
}

type testDropPeer struct {
	Peer
	dropc chan error
}

type dropError struct {
	error
	addr string
}

func (d *testDropPeer) Drop(err error) {
	err2 := &dropError{err, binStr(d)}
	d.dropc <- err2
}

type testKademlia struct {
	*Kademlia
	Discovery bool
	dropc     chan error
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
	}
}

func (k *testKademlia) newTestKadPeer(s string) Peer {
	return &testDropPeer{&BzzPeer{BzzAddr: testKadPeerAddr(s)}, k.dropc}
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
	var as []OverlayAddr
	for _, s := range regs {
		as = append(as, testKadPeerAddr(s))
	}
	err := k.Kademlia.Register(as)
	if err != nil {
		panic(err.Error())
	}
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
	return nil
}

func binStr(a OverlayPeer) string {
	if a == nil {
		return "<nil>"
	}
	return pot.ToBin(a.Address())[:8]
}

func TestSuggestPeerBug(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000").On(
		"10000000", "11000000",
		"01000000",

		"00010000", "00011000",
	).Off(
		"01000000",
	)
	err := testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestSuggestPeerFindPeers(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000").On("00100000")
	err := testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 2 row gap, saturated proxbin, no callables -> want PO 0
	k.On("00010000")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	// 1 row gap (1 less), saturated proxbin, no callables -> want PO 1
	k.On("10000000")
	err = testSuggestPeer(t, k, "<nil>", 1, false)
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
	// log.Info(k.String())
	k.Off("01000000")
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
	err = testSuggestPeer(t, k, "<nil>", 0, false)
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
	err = testSuggestPeer(t, k, "<nil>", 1, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("01000010")
	err = testSuggestPeer(t, k, "<nil>", 2, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("00100010")
	err = testSuggestPeer(t, k, "<nil>", 3, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	k.On("00010010")
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestSuggestPeerRetries(t *testing.T) {
	// 2 row gap, unsaturated proxbin, no callables -> want PO 0
	k := newTestKademlia("00000000")
	cycle := time.Second
	k.RetryInterval = uint(cycle)
	k.MaxRetries = 50
	k.RetryExponent = 2
	sleep := func(n int) {
		t := k.RetryInterval
		for i := 1; i < n; i++ {
			t *= k.RetryExponent
		}
		time.Sleep(time.Duration(t))
	}

	k.Register("01000000")
	k.On("00000001", "00000010")
	err := testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(1)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(1)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(2)
	err = testSuggestPeer(t, k, "01000000", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

	sleep(2)
	err = testSuggestPeer(t, k, "<nil>", 0, false)
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestPruning(t *testing.T) {
	k := newTestKademlia("00000000")
	k.On("10000000", "11000000", "10100000", "10010000", "10001000", "10000100")
	k.On("01000000", "01100000", "01000100", "01000010", "01000001")
	k.On("00100000", "00110000", "00100010", "00100001")
	k.MaxBinSize = 4
	k.MinBinSize = 3
	prune := make(chan time.Time)
	defer close(prune)
	k.Prune((<-chan time.Time)(prune))
	prune <- time.Now()
	quitc := make(chan bool)
	timeout := time.NewTimer(1000 * time.Millisecond)
	n := 0
	dropped := make(map[string]error)
	expDropped := []string{
		"10010000",
		"10100000",
		"11000000",
		"01000100",
		"01100000",
	}
	go func() {
		for e := range k.dropc {
			err := e.(*dropError)
			dropped[err.addr] = err.error
			n++
			if n == len(expDropped) {
				break
			}
		}
		close(quitc)
	}()
	select {
	case <-quitc:
	case <-timeout.C:
		t.Fatalf("timeout waiting for dropped peers. expected %v, got %v", len(expDropped), len(dropped))
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
	k.MaxProxDisplay = 8
	h := k.String()
	expH := "\n=========================================================================\nMon Feb 27 12:10:28 UTC 2017 KΛÐΞMLIΛ hive: queen's address: 000000\npopulation: 2 (4), MinProxBinSize: 2, MinBinSize: 1, MaxBinSize: 4\n000  0                              |  2 8100 (0) 8000 (0)\n============ DEPTH: 1 ==========================================\n001  1 4000                         |  1 4000 (0)\n002  1 2000                         |  1 2000 (0)\n003  0                              |  0\n004  0                              |  0\n005  0                              |  0\n006  0                              |  0\n007  0                              |  0\n========================================================================="
	if expH[104:] != h[104:] {
		t.Fatalf("incorrect hive output. expected %v, got %v", expH, h)
	}
}
