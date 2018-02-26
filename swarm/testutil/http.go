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
	"context"
	"io/ioutil"
	"math/big"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/swarm/api"
	httpapi "github.com/ethereum/go-ethereum/swarm/api/http"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

type fakeBackend struct {
	blocknumber int64
}

func (f *fakeBackend) HeaderByNumber(context context.Context, _ string, bigblock *big.Int) (*types.Header, error) {
	f.blocknumber++
	biggie := big.NewInt(f.blocknumber)
	return &types.Header{
		Number: biggie,
	}, nil
}

func NewTestSwarmServer(t *testing.T) *TestSwarmServer {
	dir, err := ioutil.TempDir("", "swarm-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	storeparams := &storage.StoreParams{
		ChunkDbPath:   dir,
		DbCapacity:    5000000,
		CacheCapacity: 5000,
	}
	localStore, err := storage.NewLocalStore(storage.MakeHashFunc(storage.SHA3Hash), storeparams, make([]byte, 32), nil)
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

	rh, err := storage.NewTestResourceHandler(resourceDir, &fakeBackend{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	a := api.NewApi(dpa, nil, rh)
	srv := httptest.NewServer(httpapi.NewServer(a))
	return &TestSwarmServer{
		Server: srv,
		Dpa:    dpa,
		dir:    dir,
		Hasher: storage.MakeHashFunc(storage.SHA3Hash)(),
		cleanup: func() {
			srv.Close()
			rh.Close()
			dpa.Stop()
			os.RemoveAll(dir)
			os.RemoveAll(resourceDir)
		},
	}
}

type TestSwarmServer struct {
	*httptest.Server
	Hasher  storage.SwarmHash
	Dpa     *storage.DPA
	dir     string
	cleanup func()
}

func (t *TestSwarmServer) Close() {
	t.cleanup()
}
