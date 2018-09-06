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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"
)

type TestServer interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// simulated timeProvider
type fakeTimeProvider struct {
	currentTime uint64
}

func (f *fakeTimeProvider) Tick() {
	f.currentTime++
}

func (f *fakeTimeProvider) Now() mru.Timestamp {
	return mru.Timestamp{Time: f.currentTime}
}

func NewTestSwarmServer(t *testing.T, serverFunc func(*api.API) TestServer, resolver api.Resolver) *TestSwarmServer {
	dir, err := ioutil.TempDir("", "swarm-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	storeparams := storage.NewDefaultLocalStoreParams()
	storeparams.DbCapacity = 5000000
	storeparams.CacheCapacity = 5000
	storeparams.Init(dir)
	localStore, err := storage.NewLocalStore(storeparams, nil)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	fileStore := storage.NewFileStore(localStore, storage.NewFileStoreParams())

	// mutable resources test setup
	resourceDir, err := ioutil.TempDir("", "swarm-resource-test")
	if err != nil {
		t.Fatal(err)
	}

	fakeTimeProvider := &fakeTimeProvider{
		currentTime: 42,
	}
	mru.TimestampProvider = fakeTimeProvider
	rhparams := &mru.HandlerParams{}
	rh, err := mru.NewTestHandler(resourceDir, rhparams)
	if err != nil {
		t.Fatal(err)
	}

	a := api.NewAPI(fileStore, resolver, rh.Handler, nil)
	srv := httptest.NewServer(serverFunc(a))
	return &TestSwarmServer{
		Server:            srv,
		FileStore:         fileStore,
		dir:               dir,
		Hasher:            storage.MakeHashFunc(storage.DefaultHash)(),
		timestampProvider: fakeTimeProvider,
		cleanup: func() {
			srv.Close()
			rh.Close()
			os.RemoveAll(dir)
			os.RemoveAll(resourceDir)
		},
	}
}

type TestSwarmServer struct {
	*httptest.Server
	Hasher            storage.SwarmHash
	FileStore         *storage.FileStore
	dir               string
	cleanup           func()
	timestampProvider *fakeTimeProvider
}

func (t *TestSwarmServer) Close() {
	t.cleanup()
}

func (t *TestSwarmServer) GetCurrentTime() mru.Timestamp {
	return t.timestampProvider.Now()
}
