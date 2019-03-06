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
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	mockmem "github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel     = flag.Int("loglevel", 2, "verbosity of logs")
	nodes        = flag.Int("nodes", 0, "number of nodes")
	chunks       = flag.Int("chunks", 0, "number of chunks")
	useMockStore = flag.Bool("mockstore", false, "disabled mock store (default: enabled)")
	longrunning  = flag.Bool("longrunning", false, "do run long-running tests")

	bucketKeyDB        = simulation.BucketKey("db")
	bucketKeyStore     = simulation.BucketKey("store")
	bucketKeyFileStore = simulation.BucketKey("filestore")
	bucketKeyNetStore  = simulation.BucketKey("netstore")
	bucketKeyDelivery  = simulation.BucketKey("delivery")
	bucketKeyRegistry  = simulation.BucketKey("registry")

	chunkSize = 4096
	pof       = network.Pof
)

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

// newNetStoreAndDelivery is a default constructor for BzzAddr, NetStore and Delivery, used in Simulations
func newNetStoreAndDelivery(ctx *adapters.ServiceContext, bucket *sync.Map) (*network.BzzAddr, *storage.NetStore, *Delivery, func(), error) {
	addr := network.NewAddr(ctx.Config.Node())

	netStore, delivery, cleanup, err := netStoreAndDeliveryWithAddr(ctx, bucket, addr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

	return addr, netStore, delivery, cleanup, nil
}

// newNetStoreAndDeliveryWithBzzAddr is a constructor for NetStore and Delivery, used in Simulations, accepting any BzzAddr
func newNetStoreAndDeliveryWithBzzAddr(ctx *adapters.ServiceContext, bucket *sync.Map, addr *network.BzzAddr) (*storage.NetStore, *Delivery, func(), error) {
	netStore, delivery, cleanup, err := netStoreAndDeliveryWithAddr(ctx, bucket, addr)
	if err != nil {
		return nil, nil, nil, err
	}

	netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New

	return netStore, delivery, cleanup, nil
}

// newNetStoreAndDeliveryWithRequestFunc is a constructor for NetStore and Delivery, used in Simulations, accepting any NetStore.RequestFunc
func newNetStoreAndDeliveryWithRequestFunc(ctx *adapters.ServiceContext, bucket *sync.Map, rf network.RequestFunc) (*network.BzzAddr, *storage.NetStore, *Delivery, func(), error) {
	addr := network.NewAddr(ctx.Config.Node())

	netStore, delivery, cleanup, err := netStoreAndDeliveryWithAddr(ctx, bucket, addr)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	netStore.NewNetFetcherFunc = network.NewFetcherFactory(rf, true).New

	return addr, netStore, delivery, cleanup, nil
}

func netStoreAndDeliveryWithAddr(ctx *adapters.ServiceContext, bucket *sync.Map, addr *network.BzzAddr) (*storage.NetStore, *Delivery, func(), error) {
	n := ctx.Config.Node()

	store, datadir, err := createTestLocalStorageForID(n.ID(), addr)
	if *useMockStore {
		store, datadir, err = createMockStore(mockmem.NewGlobalStore(), n.ID(), addr)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	localStore := store.(*storage.LocalStore)
	netStore, err := storage.NewNetStore(localStore, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	fileStore := storage.NewFileStore(netStore, storage.NewFileStoreParams())

	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	delivery := NewDelivery(kad, netStore)

	bucket.Store(bucketKeyStore, store)
	bucket.Store(bucketKeyDB, netStore)
	bucket.Store(bucketKeyDelivery, delivery)
	bucket.Store(bucketKeyFileStore, fileStore)
	// for the kademlia object, we use the global key from the simulation package,
	// as the simulation will try to access it in the WaitTillHealthy with that key
	bucket.Store(simulation.BucketKeyKademlia, kad)

	cleanup := func() {
		netStore.Close()
		os.RemoveAll(datadir)
	}

	return netStore, delivery, cleanup, nil
}

func newStreamerTester(registryOptions *RegistryOptions) (*p2ptest.ProtocolTester, *Registry, *storage.LocalStore, func(), error) {
	// setup
	addr := network.RandomAddr() // tested peers peer address
	to := network.NewKademlia(addr.OAddr, network.NewKadParams())

	// temp datadir
	datadir, err := ioutil.TempDir("", "streamer")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	removeDataDir := func() {
		os.RemoveAll(datadir)
	}

	params := storage.NewDefaultLocalStoreParams()
	params.Init(datadir)
	params.BaseKey = addr.Over()

	localStore, err := storage.NewTestLocalStoreForAddr(params)
	if err != nil {
		removeDataDir()
		return nil, nil, nil, nil, err
	}

	netStore, err := storage.NewNetStore(localStore, nil)
	if err != nil {
		removeDataDir()
		return nil, nil, nil, nil, err
	}

	delivery := NewDelivery(to, netStore)
	netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, true).New
	streamer := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), registryOptions, nil)
	teardown := func() {
		streamer.Close()
		removeDataDir()
	}
	protocolTester := p2ptest.NewProtocolTester(addr.ID(), 1, streamer.runProtocol)

	err = waitForPeers(streamer, 10*time.Second, 1)
	if err != nil {
		teardown()
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

// not used in this context, only to fulfill ChunkStore interface
func (rrs *roundRobinStore) Has(ctx context.Context, addr storage.Address) bool {
	panic("RoundRobinStor doesn't support HasChunk")
}

func (rrs *roundRobinStore) Get(ctx context.Context, addr storage.Address) (storage.Chunk, error) {
	return nil, errors.New("get not well defined on round robin store")
}

func (rrs *roundRobinStore) Put(ctx context.Context, chunk storage.Chunk) error {
	i := atomic.AddUint32(&rrs.index, 1)
	idx := int(i) % len(rrs.stores)
	return rrs.stores[idx].Put(ctx, chunk)
}

func (rrs *roundRobinStore) Close() {
	for _, store := range rrs.stores {
		store.Close()
	}
}

func readAll(fileStore *storage.FileStore, hash []byte) (int64, error) {
	r, _ := fileStore.Retrieve(context.TODO(), hash)
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

func uploadFilesToNodes(sim *simulation.Simulation) ([]storage.Address, []string, error) {
	nodes := sim.UpNodeIDs()
	nodeCnt := len(nodes)
	log.Debug(fmt.Sprintf("Uploading %d files to nodes", nodeCnt))
	//array holding generated files
	rfiles := make([]string, nodeCnt)
	//array holding the root hashes of the files
	rootAddrs := make([]storage.Address, nodeCnt)

	var err error
	//for every node, generate a file and upload
	for i, id := range nodes {
		item, ok := sim.NodeItem(id, bucketKeyFileStore)
		if !ok {
			return nil, nil, fmt.Errorf("Error accessing localstore")
		}
		fileStore := item.(*storage.FileStore)
		//generate a file
		rfiles[i], err = generateRandomFile()
		if err != nil {
			return nil, nil, err
		}
		//store it (upload it) on the FileStore
		ctx := context.TODO()
		rk, wait, err := fileStore.Store(ctx, strings.NewReader(rfiles[i]), int64(len(rfiles[i])), false)
		log.Debug("Uploaded random string file to node")
		if err != nil {
			return nil, nil, err
		}
		err = wait(ctx)
		if err != nil {
			return nil, nil, err
		}
		rootAddrs[i] = rk
	}
	return rootAddrs, rfiles, nil
}

//generate a random file (string)
func generateRandomFile() (string, error) {
	//generate a random file size between minFileSize and maxFileSize
	fileSize := rand.Intn(maxFileSize-minFileSize) + minFileSize
	log.Debug(fmt.Sprintf("Generated file with filesize %d kB", fileSize))
	b := testutil.RandomBytes(1, fileSize*1024)
	return string(b), nil
}

//create a local store for the given node
func createTestLocalStorageForID(id enode.ID, addr *network.BzzAddr) (storage.ChunkStore, string, error) {
	var datadir string
	var err error
	datadir, err = ioutil.TempDir("", fmt.Sprintf("syncer-test-%s", id.TerminalString()))
	if err != nil {
		return nil, "", err
	}
	var store storage.ChunkStore
	params := storage.NewDefaultLocalStoreParams()
	params.ChunkDbPath = datadir
	params.BaseKey = addr.Over()
	store, err = storage.NewTestLocalStoreForAddr(params)
	if err != nil {
		os.RemoveAll(datadir)
		return nil, "", err
	}
	return store, datadir, nil
}

// watchDisconnections receives simulation peer events in a new goroutine and sets atomic value
// disconnected to true in case of a disconnect event.
func watchDisconnections(ctx context.Context, sim *simulation.Simulation) (disconnected *boolean) {
	log.Debug("Watching for disconnections")
	disconnections := sim.PeerEvents(
		ctx,
		sim.NodeIDs(),
		simulation.NewPeerEventsFilter().Drop(),
	)
	disconnected = new(boolean)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d := <-disconnections:
				if d.Error != nil {
					log.Error("peer drop event error", "node", d.NodeID, "peer", d.PeerID, "err", d.Error)
				} else {
					log.Error("peer drop", "node", d.NodeID, "peer", d.PeerID)
				}
				disconnected.set(true)
			}
		}
	}()
	return disconnected
}

// boolean is used to concurrently set
// and read a boolean value.
type boolean struct {
	v  bool
	mu sync.RWMutex
}

// set sets the value.
func (b *boolean) set(v bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.v = v
}

// bool reads the value.
func (b *boolean) bool() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.v
}
