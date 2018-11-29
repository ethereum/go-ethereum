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
	"bytes"
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestAccessors tests most basic Put and Get functionalities
// for different accessors. This test validates that the chunk
// is retrievable from the database, not if all indexes are set
// correctly.
func TestAccessors(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	for _, m := range []Mode{
		ModeSyncing,
		ModeUpload,
		ModeRequest,
		ModeSynced,
		ModeAccess,
	} {
		t.Run(ModeName(m), func(t *testing.T) {
			a := db.Accessor(m)

			want := generateRandomChunk()

			err := a.Put(context.Background(), want)
			if err != nil {
				t.Fatal(err)
			}

			got, err := a.Get(context.Background(), want.Address())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got.Data(), want.Data()) {
				t.Errorf("got chunk data %x, want %x", got.Data(), want.Data())
			}
		})
	}

	// Removal mode is a special case as it removes the chunk
	// from the database.
	t.Run(ModeName(ModeRemoval), func(t *testing.T) {
		a := db.Accessor(ModeUpload)

		want := generateRandomChunk()

		// first put a random chunk to the database
		err := a.Put(context.Background(), want)
		if err != nil {
			t.Fatal(err)
		}

		got, err := a.Get(context.Background(), want.Address())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Data(), want.Data()) {
			t.Errorf("got chunk data %x, want %x", got.Data(), want.Data())
		}

		a = db.Accessor(ModeRemoval)

		// removal accessor actually removes the chunk on Put
		err = a.Put(context.Background(), want)
		if err != nil {
			t.Fatal(err)
		}

		// chunk should not be found
		wantErr := storage.ErrChunkNotFound
		_, err = a.Get(context.Background(), want.Address())
		if err != wantErr {
			t.Errorf("got error %v, expected %v", err, wantErr)
		}
	})
}
