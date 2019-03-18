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

package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
)

type TestServer interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func NewTestSwarmServer(t *testing.T, serverFunc func(*api.API) TestServer, resolver api.Resolver) *TestSwarmServer {
	swarmDir, err := ioutil.TempDir("", "swarm-storage-test")
	if err != nil {
		t.Fatal(err)
	}

	storeParams := storage.NewDefaultLocalStoreParams()
	storeParams.DbCapacity = 5000000
	storeParams.CacheCapacity = 5000
	storeParams.Init(swarmDir)
	localStore, err := storage.NewLocalStore(storeParams, nil)
	if err != nil {
		os.RemoveAll(swarmDir)
		t.Fatal(err)
	}
	fileStore := storage.NewFileStore(localStore, storage.NewFileStoreParams())
	// Swarm feeds test setup
	feedsDir, err := ioutil.TempDir("", "swarm-feeds-test")
	if err != nil {
		t.Fatal(err)
	}

	feeds, err := feed.NewTestHandler(feedsDir, &feed.HandlerParams{})
	if err != nil {
		t.Fatal(err)
	}

	swarmApi := api.NewAPI(fileStore, resolver, feeds.Handler, nil)
	apiServer := httptest.NewServer(serverFunc(swarmApi))

	tss := &TestSwarmServer{
		Server:    apiServer,
		FileStore: fileStore,
		dir:       swarmDir,
		Hasher:    storage.MakeHashFunc(storage.DefaultHash)(),
		cleanup: func() {
			apiServer.Close()
			fileStore.Close()
			feeds.Close()
			os.RemoveAll(swarmDir)
			os.RemoveAll(feedsDir)
		},
		CurrentTime: 42,
	}
	feed.TimestampProvider = tss
	return tss
}

type TestSwarmServer struct {
	*httptest.Server
	Hasher      storage.SwarmHash
	FileStore   *storage.FileStore
	dir         string
	cleanup     func()
	CurrentTime uint64
}

func (t *TestSwarmServer) Close() {
	t.cleanup()
}

func (t *TestSwarmServer) Now() feed.Timestamp {
	return feed.Timestamp{Time: t.CurrentTime}
}
