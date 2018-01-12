// Copyright 2016 The go-ethereum Authors
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
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func init() {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlWarn, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
}

// TODO: extract newStreamer
func newStreamerTester(t *testing.T) (*p2ptest.ProtocolTester, *Streamer, func(), error) {
	// setup
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())

	// temp datadir
	datadir, err := ioutil.TempDir("", "streamer")
	if err != nil {
		return nil, nil, func() {}, err
	}
	teardown := func() {
		os.RemoveAll(datadir)
	}

	localStore, err := storage.NewTestLocalStore(datadir)
	if err != nil {
		return nil, nil, teardown, err
	}

	dbAccess := NewDbAccess(localStore)
	streamer := NewStreamer(to, dbAccess)

	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		bzzPeer := &bzzPeer{
			Peer:      protocols.NewPeer(p, rw, StreamerSpec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		}
		return streamer.Run(bzzPeer)
	}

	protocolTester := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), 1, run)
	return protocolTester, streamer, teardown, nil
}

// TODO
// func newStreamer() (*Streamer, error) {
//
// }

func TestStreamerSubscribe(t *testing.T) {
	tester, streamer, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	err = streamer.Subscribe(tester.IDs[0], "foo", nil, 0, 0, Top, true)
	if err == nil || err.Error() != "stream foo not registered" {
		t.Fatalf("Expected error %v, got %v", "stream foo not registered", err)
	}
}

type testIncomingStreamer struct {
	t []byte
}

func (self *testIncomingStreamer) NeedData([]byte) func() {
	return nil
}

func (self *testIncomingStreamer) BatchDone(string, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	return nil
}

func TestStreamerRegisterIncoming(t *testing.T) {
	// TODO: we only need streamer
	tester, streamer, teardown, err := newStreamerTester(t)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

	streamer.RegisterIncomingStreamer("foo", func(p *StreamerPeer, t []byte) (IncomingStreamer, error) {
		return &testIncomingStreamer{
			t: t,
		}, nil
	})

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		t.Fatal("timeout: peer is not created")
	}

	err = streamer.Subscribe(tester.IDs[0], "foo", nil, 0, 0, Top, true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func waitForPeers(streamer *Streamer, timeout time.Duration) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	timeoutTimer := time.NewTimer(timeout)
	for {
		select {
		case <-ticker.C:
			if len(streamer.peers) > 0 {
				return nil
			}
		case <-timeoutTimer.C:
			return errors.New("timeout")
		}
	}
}
