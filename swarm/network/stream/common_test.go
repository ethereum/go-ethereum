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
	"encoding/binary"
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
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/db"
	colorable "github.com/mattn/go-colorable"
)

var (
	deliveries   map[discover.NodeID]*Delivery
	stores       map[discover.NodeID]storage.ChunkStore
	toAddr       func(discover.NodeID) *network.BzzAddr
	peerCount    func(discover.NodeID) int
	adapter      = flag.String("adapter", "sim", "type of simulation: sim|exec|docker")
	loglevel     = flag.Int("loglevel", 2, "verbosity of logs")
	nodes        = flag.Int("nodes", 0, "number of nodes")
	chunks       = flag.Int("chunks", 0, "number of chunks")
	useMockStore = flag.Bool("mockstore", false, "disabled mock store (default: enabled)")
)

var (
	defaultSkipCheck  bool
	waitPeerErrC      chan error
	chunkSize         = 4096
	registries        map[discover.NodeID]*TestRegistry
	createStoreFunc   func(id discover.NodeID, addr *network.BzzAddr) (storage.ChunkStore, error)
	getRetrieveFunc   = defaultRetrieveFunc
	subscriptionCount = 0
	globalStore       mock.GlobalStorer
	globalStoreDir    string
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

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

func createGlobalStore() {
	var err error
	globalStoreDir, err = ioutil.TempDir("", "global.store")
	if err != nil {
		log.Error("Error initiating global store temp directory!", "err", err)
		return
	}
	globalStore, err = db.NewGlobalStore(globalStoreDir)
	if err != nil {
		log.Error("Error initiating global store!", "err", err)
	}
}

// NewStreamerService
func NewStreamerService(ctx *adapters.ServiceContext) (node.Service, error) {
	var err error
	id := ctx.Config.ID
	addr := toAddr(id)
	kad := network.NewKademlia(addr.Over(), network.NewKadParams())
	stores[id], err = createStoreFunc(id, addr)
	if err != nil {
		return nil, err
	}
	store := stores[id].(*storage.LocalStore)
	db := storage.NewDBAPI(store)
	delivery := NewDelivery(kad, db)
	deliveries[id] = delivery
	r := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
		SkipCheck:  defaultSkipCheck,
		DoRetrieve: false,
	})
	RegisterSwarmSyncerServer(r, db)
	RegisterSwarmSyncerClient(r, db)
	go func() {
		waitPeerErrC <- waitForPeers(r, 1*time.Second, peerCount(id))
	}()
	fileStore := storage.NewFileStore(storage.NewNetStore(store, getRetrieveFunc(id)), storage.NewFileStoreParams())
	testRegistry := &TestRegistry{Registry: r, fileStore: fileStore}
	registries[id] = testRegistry
	return testRegistry, nil
}

func defaultRetrieveFunc(id discover.NodeID) func(ctx context.Context, chunk *storage.Chunk) error {
	return nil
}

func datadirsCleanup() {
	for _, id := range ids {
		os.RemoveAll(datadirs[id])
	}
	if globalStoreDir != "" {
		os.RemoveAll(globalStoreDir)
	}
}

//local stores need to be cleaned up after the sim is done
func localStoreCleanup() {
	log.Info("Cleaning up...")
	for _, id := range ids {
		registries[id].Close()
		stores[id].Close()
	}
	log.Info("Local store cleanup done")
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

	params := storage.NewDefaultLocalStoreParams()
	params.Init(datadir)
	params.BaseKey = addr.Over()

	localStore, err := storage.NewTestLocalStoreForAddr(params)
	if err != nil {
		return nil, nil, nil, removeDataDir, err
	}

	db := storage.NewDBAPI(localStore)
	delivery := NewDelivery(to, db)
	streamer := NewRegistry(addr, delivery, db, state.NewInmemoryStore(), &RegistryOptions{
		SkipCheck: defaultSkipCheck,
	})
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

func (rrs *roundRobinStore) Get(ctx context.Context, addr storage.Address) (*storage.Chunk, error) {
	return nil, errors.New("get not well defined on round robin store")
}

func (rrs *roundRobinStore) Put(ctx context.Context, chunk *storage.Chunk) {
	i := atomic.AddUint32(&rrs.index, 1)
	idx := int(i) % len(rrs.stores)
	rrs.stores[idx].Put(ctx, chunk)
}

func (rrs *roundRobinStore) Close() {
	for _, store := range rrs.stores {
		store.Close()
	}
}

type TestRegistry struct {
	*Registry
	fileStore *storage.FileStore
}

func (r *TestRegistry) APIs() []rpc.API {
	a := r.Registry.APIs()
	a = append(a, rpc.API{
		Namespace: "stream",
		Version:   "3.0",
		Service:   r,
		Public:    true,
	})
	return a
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

func (r *TestRegistry) ReadAll(hash common.Hash) (int64, error) {
	return readAll(r.fileStore, hash[:])
}

func (r *TestRegistry) Start(server *p2p.Server) error {
	return r.Registry.Start(server)
}

func (r *TestRegistry) Stop() error {
	return r.Registry.Stop()
}

type TestExternalRegistry struct {
	*Registry
}

func (r *TestExternalRegistry) APIs() []rpc.API {
	a := r.Registry.APIs()
	a = append(a, rpc.API{
		Namespace: "stream",
		Version:   "3.0",
		Service:   r,
		Public:    true,
	})
	return a
}

func (r *TestExternalRegistry) GetHashes(ctx context.Context, peerId discover.NodeID, s Stream) (*rpc.Subscription, error) {
	peer := r.getPeer(peerId)

	client, err := peer.getClient(ctx, s)
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
		// if we begin sending event immediately some events
		// will probably be dropped since the subscription ID might not be send to
		// the client.
		// ref: rpc/subscription_test.go#L65
		time.Sleep(1 * time.Second)
		for {
			select {
			case h := <-c.hashes:
				<-c.enableNotificationsC // wait for notification subscription to complete
				if err := notifier.Notify(sub.ID, h); err != nil {
					log.Warn(fmt.Sprintf("rpc sub notifier notify stream %s: %v", s, err))
				}
			case err := <-sub.Err():
				if err != nil {
					log.Warn(fmt.Sprintf("caught subscription error in stream %s: %v", s, err))
				}
			case <-notifier.Closed():
				log.Trace(fmt.Sprintf("rpc sub notifier closed"))
				return
			}
		}
	}()

	return sub, nil
}

func (r *TestExternalRegistry) EnableNotifications(peerId discover.NodeID, s Stream) error {
	peer := r.getPeer(peerId)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := peer.getClient(ctx, s)
	if err != nil {
		return err
	}

	close(client.Client.(*testExternalClient).enableNotificationsC)

	return nil
}

// TODO: merge functionalities of testExternalClient and testExternalServer
// with testClient and testServer.

type testExternalClient struct {
	hashes               chan []byte
	db                   *storage.DBAPI
	enableNotificationsC chan struct{}
}

func newTestExternalClient(db *storage.DBAPI) *testExternalClient {
	return &testExternalClient{
		hashes:               make(chan []byte),
		db:                   db,
		enableNotificationsC: make(chan struct{}),
	}
}

func (c *testExternalClient) NeedData(ctx context.Context, hash []byte) func() {
	chunk, _ := c.db.GetOrCreateRequest(ctx, hash)
	if chunk.ReqC == nil {
		return nil
	}
	c.hashes <- hash
	return func() {
		chunk.WaitToStore()
	}
}

func (c *testExternalClient) BatchDone(Stream, uint64, []byte, []byte) func() (*TakeoverProof, error) {
	return nil
}

func (c *testExternalClient) Close() {}

const testExternalServerBatchSize = 10

type testExternalServer struct {
	t         string
	keyFunc   func(key []byte, index uint64)
	sessionAt uint64
	maxKeys   uint64
	streamer  *TestExternalRegistry
}

func newTestExternalServer(t string, sessionAt, maxKeys uint64, keyFunc func(key []byte, index uint64)) *testExternalServer {
	if keyFunc == nil {
		keyFunc = binary.BigEndian.PutUint64
	}
	return &testExternalServer{
		t:         t,
		keyFunc:   keyFunc,
		sessionAt: sessionAt,
		maxKeys:   maxKeys,
	}
}

func (s *testExternalServer) SetNextBatch(from uint64, to uint64) ([]byte, uint64, uint64, *HandoverProof, error) {
	if from == 0 && to == 0 {
		from = s.sessionAt
		to = s.sessionAt + testExternalServerBatchSize
	}
	if to-from > testExternalServerBatchSize {
		to = from + testExternalServerBatchSize - 1
	}
	if from >= s.maxKeys && to > s.maxKeys {
		return nil, 0, 0, nil, io.EOF
	}
	if to > s.maxKeys {
		to = s.maxKeys
	}
	b := make([]byte, HashSize*(to-from+1))
	for i := from; i <= to; i++ {
		s.keyFunc(b[(i-from)*HashSize:(i-from+1)*HashSize], i)
	}
	return b, from, to, nil, nil
}

func (s *testExternalServer) GetData(context.Context, []byte) ([]byte, error) {
	return make([]byte, 4096), nil
}

func (s *testExternalServer) Close() {}

// Sets the global value defaultSkipCheck.
// It should be used in test function defer to reset the global value
// to the original value.
//
// defer setDefaultSkipCheck(defaultSkipCheck)
// defaultSkipCheck = skipCheck
//
// This works as defer function arguments evaluations are evaluated as ususal,
// but only the function body invocation is deferred.
func setDefaultSkipCheck(skipCheck bool) {
	defaultSkipCheck = skipCheck
}
