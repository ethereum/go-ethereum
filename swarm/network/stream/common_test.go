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

package stream

import (
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	adapter  = flag.String("adapter", "sim", "type of simulation: sim|socket|exec|docker")
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

var services = adapters.Services{
	"delivery": newDeliveryService,
	"syncer":   newSyncerService,
}

func init() {
	flag.Parse()
	// register the Delivery service which will run as a devp2p
	// protocol when using the exec adapter
	adapters.RegisterServices(services)

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

var (
	delivery *Delivery
	fileHash storage.Key
)

func mustReadAll(dpa *storage.DPA, hash storage.Key) (int, error) {
	r := dpa.Retrieve(fileHash)
	buf := make([]byte, 1024)
	var n, total int
	var err error
	for (total == 0 || n > 0) && err == nil {
		n, err = r.ReadAt(buf, int64(total))
		total += n
	}
	if err != nil && err != io.EOF {
		return total, err
	}
	return total, nil
}

func newStreamerTester(t *testing.T) (*p2ptest.ProtocolTester, *Registry, *storage.LocalStore, func(), error) {
	// setup
	addr := network.RandomAddr() // tested peers peer address
	to := network.NewKademlia(addr.OAddr, network.NewKadParams())

	// temp datadir
	datadir, err := ioutil.TempDir("", "streamer")
	if err != nil {
		return nil, nil, nil, func() {}, err
	}
	teardown := func() {
		os.RemoveAll(datadir)
	}

	localStore, err := storage.NewTestLocalStoreForAddr(datadir, addr.Over())
	if err != nil {
		return nil, nil, nil, teardown, err
	}

	db := storage.NewDBAPI(localStore)
	delivery := NewDelivery(to, db)
	streamer := NewRegistry(delivery)
	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		bzzPeer := &network.BzzPeer{
			Peer:      protocols.NewPeer(p, rw, Spec),
			localAddr: addr,
			BzzAddr:   network.NewAddrFromNodeID(p.ID()),
		}
		to.On(bzzPeer)
		return streamer.Run(bzzPeer)
	}
	protocolTester := p2ptest.NewProtocolTester(t, network.NewNodeIDFromAddr(addr), 1, run)

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		return nil, nil, nil, nil, errors.New("timeout: peer is not created")
	}

	return protocolTester, streamer, localStore, teardown, nil
}

func waitForPeers(streamer *Registry, timeout time.Duration) error {
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
