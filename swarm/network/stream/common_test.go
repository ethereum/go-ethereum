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
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var (
	adapter  = flag.String("adapter", "sim", "type of simulation: sim|socket|exec|docker")
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

var (
	defaultSkipCheck bool
	waitPeerErrC     chan error
	chunkSize        = 4096
)

var services = adapters.Services{
	"streamer": NewStreamerService,
}

func init() {
	flag.Parse()
	// register the Delivery service which will run as a devp2p
	// protocol when using the exec adapter
	adapters.RegisterServices(services)

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

}

// NewStreamerService
func NewStreamerService(ctx *adapters.ServiceContext) (node.Service, error) {
	id := ctx.Config.ID
	addr := toAddr(id)
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	store := stores[id].(*storage.LocalStore)
	db := storage.NewDBAPI(store)
	delivery := NewDelivery(kad, db)
	deliveries[id] = delivery
	netStore := storage.NewNetStore(store, nil)
	r := NewRegistry(addr, delivery, netStore, defaultSkipCheck)
	RegisterSwarmSyncerServer(r, db)
	RegisterSwarmSyncerClient(r, db)
	go func() {
		waitPeerErrC <- waitForPeers(r, 1*time.Second, peerCount(id))
	}()
	return r, nil
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
	streamer := NewRegistry(addr, delivery, localStore, defaultSkipCheck)
	protocolTester := p2ptest.NewProtocolTester(t, network.NewNodeIDFromAddr(addr), 1, streamer.runProtocol)

	err = waitForPeers(streamer, 1*time.Second, 1)
	if err != nil {
		return nil, nil, nil, nil, errors.New("timeout: peer is not created")
	}

	return protocolTester, streamer, localStore, teardown, nil
}

func waitForPeers(streamer *Registry, timeout time.Duration, expectedPeers int) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	timeoutTimer := time.NewTimer(timeout)
	for {
		select {
		case <-ticker.C:
			if streamer.peersCount() >= expectedPeers {
				return nil
			}
		case <-timeoutTimer.C:
			return errors.New("timeout")
		}
	}
}

type roundRobinStore struct {
	index  uint32
	stores []storage.ChunkStore
}

func newRoundRobinStore(stores ...storage.ChunkStore) *roundRobinStore {
	return &roundRobinStore{
		stores: stores,
	}
}

func (rrs *roundRobinStore) Get(key storage.Key) (*storage.Chunk, error) {
	return nil, errors.New("get not well defined on round robin store")
}

func (rrs *roundRobinStore) Put(chunk *storage.Chunk) {
	i := atomic.AddUint32(&rrs.index, 1)
	idx := int(i) % len(rrs.stores)
	rrs.stores[idx].Put(chunk)
}

func (rrs *roundRobinStore) Close() {
	for _, store := range rrs.stores {
		store.Close()
	}
}
