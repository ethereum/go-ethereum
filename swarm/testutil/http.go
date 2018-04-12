// Copyright 2017 The go-etherfact Authors
// This file is part of the go-etherfact library.
//
// The go-etherfact library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherfact library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherfact library. If not, see <http://www.gnu.org/licenses/>.

package testutil

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/EtherFact-Project/go-etherfact/swarm/api"
	httpapi "github.com/EtherFact-Project/go-etherfact/swarm/api/http"
	"github.com/EtherFact-Project/go-etherfact/swarm/storage"
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
	localStore, err := storage.NewLocalStore(storage.MakeHashFunc("SHA3"), storeparams)
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
	a := api.NewApi(dpa, nil)
	srv := httptest.NewServer(httpapi.NewServer(a))
	return &TestSwarmServer{
		Server: srv,
		Dpa:    dpa,
		dir:    dir,
	}
}

type TestSwarmServer struct {
	*httptest.Server

	Dpa *storage.DPA
	dir string
}

func (t *TestSwarmServer) Close() {
	t.Server.Close()
	t.Dpa.Stop()
	os.RemoveAll(t.dir)
}
