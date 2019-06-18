// Copyright 2016 The go-ethereum Authors
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
	"context"

	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/network/timeouts"
)

// LNetStore is a wrapper of NetStore, which implements the chunk.Store interface. It is used only by the FileStore,
// the component used by the Swarm API to store and retrieve content and to split and join chunks.
type LNetStore struct {
	*NetStore
}

// NewLNetStore is a constructor for LNetStore
func NewLNetStore(store *NetStore) *LNetStore {
	return &LNetStore{
		NetStore: store,
	}
}

// Get converts a chunk reference to a chunk Request (with empty Origin), handled by the NetStore, and
// returns the requested chunk, or error.
func (n *LNetStore) Get(ctx context.Context, mode chunk.ModeGet, ref Address) (ch Chunk, err error) {
	ctx, cancel := context.WithTimeout(ctx, timeouts.FetcherGlobalTimeout)
	defer cancel()

	return n.NetStore.Get(ctx, mode, NewRequest(ref))
}
