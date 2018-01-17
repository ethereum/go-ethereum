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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var services = adapters.Services{
	"delivery": newDeliveryService,
	"syncer":   newSyncerService,
}

var (
	adapter  = flag.String("adapter", "sim", "type of simulation: sim|socket|exec|docker")
	loglevel = flag.Int("loglevel", 2, "verbosity of logs")
)

func init() {
	flag.Parse()
	// register the Delivery service which will run as a devp2p
	// protocol when using the exec adapter
	adapters.RegisterServices(services)

	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

var (
	delivery    *Delivery
	localStores []storage.ChunkStore
	addrs       []Addr
	fileHash    storage.Key
	nodeCount   int
)

func setLocalStores(addrs ...Addr) (func(), error) {
	var datadirs []string
	localStores = make([]storage.ChunkStore, len(addrs))
	var err error
	for i, addr := range addrs {
		// TODO: remove temp datadir after test
		var datadir string
		datadir, err = ioutil.TempDir("", "streamer")
		if err != nil {
			break
		}
		var localStore *storage.LocalStore
		localStore, err = storage.NewTestLocalStoreForAddr(datadir, addr.Over())
		if err != nil {
			break
		}
		datadirs = append(datadirs, datadir)
		localStores[i] = localStore
	}
	teardown := func() {
		for _, datadir := range datadirs {
			os.RemoveAll(datadir)
		}
	}
	return teardown, err
}

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

func testSimulation(t *testing.T, simf func(adapters.NodeAdapter) (*simulations.StepResult, error)) {
	var err error
	var result *simulations.StepResult
	startedAt := time.Now()

	switch *adapter {
	case "sim":
		t.Logf("simadapter")
		result, err = simf(adapters.NewSimAdapter(services))
	case "socket":
		result, err = simf(adapters.NewSocketAdapter(services))
	case "exec":
		baseDir, err0 := ioutil.TempDir("", "swarm-test")
		if err0 != nil {
			t.Fatal(err0)
		}
		defer os.RemoveAll(baseDir)
		result, err = simf(adapters.NewExecAdapter(baseDir))
	case "docker":
		adapter, err0 := adapters.NewDockerAdapter()
		if err0 != nil {
			t.Fatal(err0)
		}
		result, err = simf(adapter)
	default:
		t.Fatal("adapter needs to be one of sim, socket, exec, docker")
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Simulation with %d nodes passed in %s", len(result.Passes), result.FinishedAt.Sub(result.StartedAt))
	var min, max time.Duration
	var sum int
	for _, pass := range result.Passes {
		duration := pass.Sub(result.StartedAt)
		if sum == 0 || duration < min {
			min = duration
		}
		if duration > max {
			max = duration
		}
		sum += int(duration.Nanoseconds())
	}
	t.Logf("Min: %s, Max: %s, Average: %s", min, max, time.Duration(sum/len(result.Passes))*time.Nanosecond)
	finishedAt := time.Now()
	t.Logf("Setup: %s, shutdown: %s", result.StartedAt.Sub(startedAt), finishedAt.Sub(result.FinishedAt))
}

func runSimulation(nodes, conns int, serviceName string, toAddr func(discover.NodeID) *BzzAddr, action func(*simulations.Network) func(context.Context) error, trigger func(*simulations.Network) chan discover.NodeID, check func(*simulations.Network, *storage.DPA) func(context.Context, discover.NodeID) (bool, error), adapter adapters.NodeAdapter) (*simulations.StepResult, error) {
	// create network
	net := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: serviceName,
	})
	defer net.Shutdown()
	ids := make([]discover.NodeID, nodes)
	nodeCount = 0
	addrs = make([]Addr, nodes)
	// start nodes
	for i := 0; i < nodes; i++ {
		node, err := net.NewNode()
		if err != nil {
			return nil, fmt.Errorf("error creating node: %s", err)
		}
		ids[i] = node.ID()
		addrs[i] = toAddr(ids[i])
	}
	// set nodes number of localstores globally available
	teardown, err := setLocalStores(addrs...)
	defer teardown()
	if err != nil {
		return nil, err
	}

	for i := 0; i < nodes; i++ {
		if err := net.Start(ids[i]); err != nil {
			return nil, fmt.Errorf("error starting node %s: %s", ids[i].TerminalString(), err)
		}
	}

	// run a simulation which connects the 10 nodes in a chain
	wg := sync.WaitGroup{}
	for i := range ids {
		// collect the overlay addresses, to
		for j := 0; j < conns; j++ {
			var k int
			if j == 0 {
				k = i - 1
			} else {
				k = rand.Intn(len(ids))
			}
			if i > 0 {
				wg.Add(1)
				go func(i, k int) {
					defer wg.Done()
					net.Connect(ids[i], ids[k])
				}(i, k)
			}
		}
	}
	wg.Wait()

	log.Debug(fmt.Sprintf("nodes: %v", len(addrs)))

	// create an only locally retrieving dpa for the pivot node to test
	// if retriee requests have arrived
	dpa := storage.NewDPA(localStores[0], storage.NewChunkerParams())
	dpa.Start()
	defer dpa.Stop()
	timeout := 300 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action(net),
		Trigger: trigger(net),
		Expect: &simulations.Expectation{
			Nodes: ids[0:1],
			Check: check(net, dpa),
		},
	})
	return result, nil
}

func newStreamerTester(t *testing.T) (*p2ptest.ProtocolTester, *Streamer, *storage.LocalStore, func(), error) {
	// setup
	addr := RandomAddr() // tested peers peer address
	to := NewKademlia(addr.OAddr, NewKadParams())

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

	dbAccess := NewDbAccess(localStore)
	delivery := NewDelivery(to, dbAccess)
	streamer := NewStreamer(delivery)
	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		bzzPeer := &bzzPeer{
			Peer:      protocols.NewPeer(p, rw, StreamerSpec),
			localAddr: addr,
			BzzAddr:   NewAddrFromNodeID(p.ID()),
		}
		to.On(bzzPeer)
		return streamer.Run(bzzPeer)
	}
	protocolTester := p2ptest.NewProtocolTester(t, NewNodeIDFromAddr(addr), 1, run)

	err = waitForPeers(streamer, 1*time.Second)
	if err != nil {
		return nil, nil, nil, nil, errors.New("timeout: peer is not created")
	}

	return protocolTester, streamer, localStore, teardown, nil
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

type testStreamerService struct {
	index    int
	addr     *BzzAddr
	streamer *Streamer
	run      func(p *p2p.Peer, rw p2p.MsgReadWriter) error
}

func (tds *testStreamerService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    StreamerSpec.Name,
			Version: StreamerSpec.Version,
			Length:  StreamerSpec.Length(),
			Run:     tds.run,
			// NodeInfo: ,
			// PeerInfo: ,
		},
	}
}

func (b *testStreamerService) APIs() []rpc.API {
	return []rpc.API{}
}

func (b *testStreamerService) Start(server *p2p.Server) error {
	return nil
}

func (b *testStreamerService) Stop() error {
	return nil
}
