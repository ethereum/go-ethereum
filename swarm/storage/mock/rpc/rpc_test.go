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

package rpc

import (
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/test"
)

// TestDBStore is running test for a GlobalStore
// using test.MockStore function.
func TestRPCStore(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	test.MockStore(t, store, 30)
}

// TestRPCStoreListings is running test for a GlobalStore
// using test.MockStoreListings function.
func TestRPCStoreListings(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	test.MockStoreListings(t, store, 1000)
}

// newTestStore creates a temporary GlobalStore
// that will be closed when returned cleanup function
// is called.
func newTestStore(t *testing.T) (s *GlobalStore, cleanup func()) {
	t.Helper()

	serverStore := mem.NewGlobalStore()

	server := rpc.NewServer()
	if err := server.RegisterName("mockStore", serverStore); err != nil {
		t.Fatal(err)
	}

	store := NewGlobalStore(rpc.DialInProc(server))
	return store, func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	}
}
