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

package localstore

import (
	"context"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/ethereum/go-ethereum/swarm/shed"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// Accessor implements ChunkStore to manage data
// in DB with different modes of access and update.
type Accessor struct {
	db   *DB
	mode Mode
}

// Accessor returns a new Accessor with a specified Mode.
func (db *DB) Accessor(mode Mode) *Accessor {
	return &Accessor{
		mode: mode,
		db:   db,
	}
}

// Put uses the underlying DB for the specific mode of update to store the chunk.
func (u *Accessor) Put(ctx context.Context, ch storage.Chunk) error {
	return u.db.update(ctx, u.mode, chunkToItem(ch))
}

// Get uses the underlying DB for the specific mode of access to get the chunk.
func (u *Accessor) Get(_ context.Context, addr storage.Address) (chunk storage.Chunk, err error) {
	item := addressToItem(addr)
	out, err := u.db.access(u.mode, item)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, storage.ErrChunkNotFound
		}
		return nil, err
	}
	return storage.NewChunk(out.Address, out.Data), nil
}

// chunkToItem creates new IndexItem with data provided by the Chunk.
func chunkToItem(ch storage.Chunk) shed.IndexItem {
	return shed.IndexItem{
		Address: ch.Address(),
		Data:    ch.Data(),
	}
}

// addressToItem creates new IndexItem with a provided address.
func addressToItem(addr storage.Address) shed.IndexItem {
	return shed.IndexItem{
		Address: addr,
	}
}
