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
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/pot"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	mockdb "github.com/ethereum/go-ethereum/swarm/storage/mock/db"
	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel     = flag.Int("loglevel", 2, "verbosity of logs")
	nodes        = flag.Int("nodes", 0, "number of nodes")
	chunks       = flag.Int("chunks", 0, "number of chunks")
	useMockStore = flag.Bool("mockstore", false, "disabled mock store (default: enabled)")
	longrunning  = flag.Bool("longrunning", false, "do run long-running tests")
	waitKademlia = flag.Bool("waitkademlia", true, "wait for healthy kademlia before checking files availability")

	bucketKeyDB        = simulation.BucketKey("db")
	bucketKeyStore     = simulation.BucketKey("store")
	bucketKeyFileStore = simulation.BucketKey("filestore")
	bucketKeyNetStore  = simulation.BucketKey("netstore")
	bucketKeyDelivery  = simulation.BucketKey("delivery")
	bucketKeyRegistry  = simulation.BucketKey("registry")

	chunkSize = 4096
	pof       = pot.DefaultPof(256)
)

func init() {
	flag.Parse()

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

func createGlobalStore() (string, *mockdb.GlobalStore, error) {
	var globalStore *mockdb.GlobalStore
	globalStoreDir, err := ioutil.TempDir("", "global.store")
	if err != nil {
		log.Error("Error initiating global store temp directory!", "err", err)
		return "", nil, err
	}
	globalStore, err = mockdb.NewGlobalStore(globalStoreDir)
	if err != nil {
		log.Error("Error initiating global store!", "err", err)
		return "", nil, err
	}
	return globalStoreDir, globalStore, nil
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
		SkipCheck: false,
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

//generate a random file (string)
func generateRandomFile() (string, error) {
	//generate a random file size between minFileSize and maxFileSize
	fileSize := rand.Intn(maxFileSize-minFileSize) + minFileSize
	log.Debug(fmt.Sprintf("Generated file with filesize %d kB", fileSize))
	b := make([]byte, fileSize*1024)
	_, err := crand.Read(b)
	if err != nil {
		log.Error("Error generating random file.", "err", err)
		return "", err
	}
	return string(b), nil
}

//create a local store for the given node
func createTestLocalStorageForId(id discover.NodeID, addr *network.BzzAddr) (storage.ChunkStore, string, error) {
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
