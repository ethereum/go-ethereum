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

package storage

import (
	"encoding/hex"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/network"
)

var (
	errUnknown = errors.New("unknown error")
)

type mockRetrieve struct {
	requests map[string]int
}

func NewMockRetrieve() *mockRetrieve {
	return &mockRetrieve{requests: make(map[string]int)}
}

func newDummyChunk(addr Address) *Chunk {
	chunk := NewChunk(addr, make(chan bool))
	chunk.SData = []byte{3, 4, 5}
	chunk.Size = 3

	return chunk
}

func (m *mockRetrieve) retrieve(chunk *Chunk) error {
	hkey := hex.EncodeToString(chunk.Addr)
	m.requests[hkey] += 1

	// on second call return error
	if m.requests[hkey] == 2 {
		return errUnknown
	}

	// on third call return data
	if m.requests[hkey] == 3 {
		*chunk = *newDummyChunk(chunk.Addr)
		go func() {
			time.Sleep(100 * time.Millisecond)
			close(chunk.ReqC)
		}()

		return nil
	}

	return nil
}

func TestNetstoreFailedRequest(t *testing.T) {
	searchTimeout = 300 * time.Millisecond

	// setup
	addr := network.RandomAddr() // tested peers peer address

	// temp datadir
	datadir, err := ioutil.TempDir("", "netstore")
	if err != nil {
		t.Fatal(err)
	}
	params := NewDefaultLocalStoreParams()
	params.Init(datadir)
	params.BaseKey = addr.Over()
	localStore, err := NewTestLocalStoreForAddr(params)
	if err != nil {
		t.Fatal(err)
	}

	r := NewMockRetrieve()
	netStore := NewNetStore(localStore, r.retrieve)

	key := Address{}

	// first call is done by the retry on ErrChunkNotFound, no need to do it here
	// _, err = netStore.Get(key)
	// if err == nil || err != ErrChunkNotFound {
	// 	t.Fatalf("expected to get ErrChunkNotFound, but got: %s", err)
	// }

	// second call
	_, err = netStore.Get(key)
	if got := r.requests[hex.EncodeToString(key)]; got != 2 {
		t.Fatalf("expected to have called retrieve two times, but got: %v", got)
	}
	if err != errUnknown {
		t.Fatalf("expected to get an unknown error, but got: %s", err)
	}

	// third call
	chunk, err := netStore.Get(key)
	if got := r.requests[hex.EncodeToString(key)]; got != 3 {
		t.Fatalf("expected to have called retrieve three times, but got: %v", got)
	}
	if err != nil || chunk == nil {
		t.Fatalf("expected to get a chunk but got: %v, %s", chunk, err)
	}
	if len(chunk.SData) != 3 {
		t.Fatalf("expected to get a chunk with size 3, but got: %v", chunk.SData)
	}
}
