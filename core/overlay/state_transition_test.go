// Copyright 2025 The go-ethereum Authors
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

package overlay

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

func TestLoadStoreTransitionState(t *testing.T) {
	has := &TransitionState{
		CurrentAccountHash: testrand.Hash(),
		CurrentSlotHash:    testrand.Hash(),
		BaseRoot:           testrand.Hash(),
	}
	db := rawdb.NewMemoryDatabase()
	if err := StoreTransitionState(db, common.Hash{0x1}, has); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTransitionState(db, common.Hash{0x1})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, has) {
		spew.Dump(got)
		spew.Dump(has)
		t.Fatalf("Unexpected transition state")
	}
}
