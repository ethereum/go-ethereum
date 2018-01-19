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

package testutil

import (
	"crypto/ecdsa"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
	httpapi "github.com/ethereum/go-ethereum/swarm/api/http"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func NewTestSwarmServer(t *testing.T) *TestSwarmServer {
	dir, err := ioutil.TempDir("", "swarm-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	storeparams := &storage.StoreParams{
		ChunkDbPath:   dir,
		DbCapacity:    5000000,
		CacheCapacity: 5000,
		Radius:        0,
	}
	localStore, err := storage.NewLocalStore(storage.MakeHashFunc(storage.SHA3Hash), storeparams)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	chunker := storage.NewTreeChunker(storage.NewChunkerParams())
	dpa := &storage.DPA{
		Chunker:    chunker,
		ChunkStore: localStore,
	}
	dpa.Start()

	// mutable resources test setup
	resourceDir, err := ioutil.TempDir("", "swarm-resource-test")
	if err != nil {
		t.Fatal(err)
	}
	ipcPath := filepath.Join(resourceDir, "test.ipc")
	ipcl, err := rpc.CreateIPCListener(ipcPath)
	if err != nil {
		t.Fatal(err)
	}
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("eth", &FakeRPC{})
	go func() {
		rpcServer.ServeListener(ipcl)
	}()
	rpcClean := func() {
		rpcServer.Stop()
	}

	// connect to fake rpc
	rpcClient, err := rpc.Dial(ipcPath)
	if err != nil {
		t.Fatal(err)
	}
	ethClient := ethclient.NewClient(rpcClient)

	rh, err := storage.NewResourceHandler(resourceDir, &testCloudStore{}, ethClient, nil)
	if err != nil {
		t.Fatal(err)
	}

	a := api.NewApi(dpa, nil, rh)
	srv := httptest.NewServer(httpapi.NewServer(a))
	return &TestSwarmServer{
		Server: srv,
		Dpa:    dpa,
		dir:    dir,
		hasher: storage.MakeHashFunc(storage.SHA3Hash)(),
		cleanup: func() {
			rh.Close()
			rpcClean()
			os.RemoveAll(dir)
			os.RemoveAll(resourceDir)
		},
	}
}

type TestSwarmServer struct {
	*httptest.Server
	hasher     storage.SwarmHash
	privatekey *ecdsa.PrivateKey
	Dpa        *storage.DPA
	dir        string
	cleanup    func()
}

func (t *TestSwarmServer) Close() {
	t.Server.Close()
	t.Dpa.Stop()
	os.RemoveAll(t.dir)
}

type testCloudStore struct {
}

func (c *testCloudStore) Store(*storage.Chunk) {
}

func (c *testCloudStore) Deliver(*storage.Chunk) {
}

func (c *testCloudStore) Retrieve(*storage.Chunk) {
}

// for faking the rpc service, since we don't need the whole node stack
type FakeRPC struct {
	blocknumber uint64
}

func (r *FakeRPC) BlockNumber() (string, error) {
	return strconv.FormatUint(r.blocknumber, 10), nil
}
