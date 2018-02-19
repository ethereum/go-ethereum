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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream/intervals"
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
	"streamer":          NewStreamerService,
	"intervalsStreamer": newIntervalsStreamerService,
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
	r := NewRegistry(addr, delivery, netStore, intervals.NewMemStore(), defaultSkipCheck)
	RegisterSwarmSyncerServer(r, db)
	RegisterSwarmSyncerClient(r, db)
	go func() {
		waitPeerErrC <- waitForPeers(r, 1*time.Second, peerCount(id))
	}()
	return &TestRegistry{Registry: r}, nil
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
	removeDataDir := func() {
		os.RemoveAll(datadir)
	}

	localStore, err := storage.NewTestLocalStoreForAddr(datadir, addr.Over())
	if err != nil {
		return nil, nil, nil, removeDataDir, err
	}

	db := storage.NewDBAPI(localStore)
	delivery := NewDelivery(to, db)
	streamer := NewRegistry(addr, delivery, localStore, intervals.NewMemStore(), defaultSkipCheck)
	teardown := func() {
		streamer.Close()
		removeDataDir()
	}
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

type TestRegistry struct {
	*Registry
}

func (r *TestRegistry) APIs() []rpc.API {
	a := r.Registry.APIs()
	a = append(a, rpc.API{
		Namespace: "stream",
		Version:   "0.1",
		Service:   r,
		Public:    true,
	})
	return a
}

func readAll(dpa *storage.DPA, hash []byte) (int64, error) {
	r := dpa.Retrieve(hash)
	buf := make([]byte, 1024)
	var n int
	var total int64
	var err error
	for (total == 0 || n > 0) && err == nil {
		n, err = r.ReadAt(buf, total)
		total += int64(n)
	}
	if err != nil && err != io.EOF {
		return total, err
	}
	return total, nil
}

func (r *TestRegistry) ReadAll(hash common.Hash) (int64, error) {
	return readAll(r.api.dpa, hash[:])
}

type TestExternalRegistry struct {
	*Registry
	hashesChan chan []byte
}

func (r *TestExternalRegistry) APIs() []rpc.API {
	a := r.Registry.APIs()
	a = append(a, rpc.API{
		Namespace: "stream",
		Version:   "0.1",
		Service:   r,
		Public:    true,
	})
	return a
}

func (r *TestExternalRegistry) GetHashes(ctx context.Context, peerId discover.NodeID, s Stream) (*rpc.Subscription, error) {

	peer := r.getPeer(peerId)

	client, err := peer.getClient(s)
	if err != nil {
		return nil, err
	}

	c := client.Client.(*testExternalClient)

	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, fmt.Errorf("Subscribe not supported")
	}

	sub := notifier.CreateSubscription()

	go func() {
		for {
			select {
			case h := <-c.hashes:
				if err := notifier.Notify(sub.ID, h); err != nil {
					log.Warn(fmt.Sprintf("rpc sub notifier notify stream %s: %v", s, err))
				}
			case err := <-sub.Err():
				log.Warn(fmt.Sprintf("caught subscription error in stream %s: %v", s, err))
			case <-notifier.Closed():
				log.Warn(fmt.Sprintf("rpc sub notifier closed"))
			}
		}
	}()

	return sub, nil
}

// TODO: merge functionalities of testExternalClient and testExternalServer
// with testClient and testServer.

type testExternalClient struct {
	t []byte
	// wait0     chan bool
	// batchDone chan bool
	hashes chan []byte
}

func newTestExternalClient(t []byte, hashesChan chan []byte) *testExternalClient {
	return &testExternalClient{
		t: t,
		// wait0:     make(chan bool),
		// batchDone: make(chan bool),
		hashes: hashesChan,
	}
}

func (self *testExternalClient) NeedData(hash []byte) func() {
	self.hashes <- hash
	return func() {}
}

func (self *testExternalClient) BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	// close(self.batchDone)
	return nil
}

func (self *testExternalClient) Close() {}

type testExternalServer struct {
	t []byte
}

func newTestExternalServer(t []byte) *testExternalServer {
	return &testExternalServer{
		t: t,
	}
}

func (self *testExternalServer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	return make([]byte, HashSize), from + 1, to + 1, nil, nil
}

func (self *testExternalServer) GetData([]byte) ([]byte, error) {
	return nil, nil
}

func (self *testExternalServer) Close() {
}
