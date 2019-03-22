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

package eth

import (
	"bytes"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
)

var dumper = spew.ConfigState{Indent: "    "}

func accountRangeExpect(t *testing.T, trie *state.Trie, statedb *state.StateDB, start *common.Address, requestedNum int, expectedNum int) AccountRangeResult {
	result, err := accountRange(*trie, start, requestedNum)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Addresses) != expectedNum {
		t.Fatalf("expected %d results.  Got %d", expectedNum, len(result.Addresses))
	}

	for i := range result.Addresses {
		if !statedb.Exist(result.Addresses[i]) {
			t.Fatalf("account not found in state %s", result.Addresses[i].String())
		}
	}

	return result
}

func TestAccountRangeAt(t *testing.T) {
	var (
		statedb  = state.NewDatabase(ethdb.NewMemDatabase())
		state, _ = state.New(common.Hash{}, statedb)
		addrs    = [512]common.Address{}
	)

	for i := 0; i < 512; i++ {
		addr := fmt.Sprintf("%x", i)
		addrs[i] = common.HexToAddress(addr)
	}

	for i := range addrs {
		state.SetBalance(addrs[i], big.NewInt(1))
	}

	state.Commit(true)
	root := state.IntermediateRoot(true)

	trie, err := statedb.OpenTrie(root)
	if err != nil {
		t.Fatal(err)
	}

	// test getting number of results less than max
	accountRangeExpect(t, &trie, state, &common.Address{0x0}, 128, 128)

	// test getting number of results greater than max
	accountRangeExpect(t, &trie, state, &common.Address{0x0}, 512, 256)

	// test pagination
	firstResult := accountRangeExpect(t, &trie, state, &common.Address{0x0}, 128, 128)
	secondResult := accountRangeExpect(t, &trie, state, &firstResult.Next, 128, 128)

	for i := range firstResult.Addresses {
		for j := range secondResult.Addresses {
			if bytes.Equal(firstResult.Addresses[i].Bytes(), secondResult.Addresses[j].Bytes()) {
				t.Fatalf("pagination test failed:  results should not overlap")
			}
		}
	}
}

func TestStorageRangeAt(t *testing.T) {
	// Create a state where account 0x010000... has a few storage entries.
	var (
		state, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()))
		addr     = common.Address{0x01}
		keys     = []common.Hash{ // hashes of Keys of storage
			common.HexToHash("340dd630ad21bf010b4e676dbfa9ba9a02175262d1fa356232cfde6cb5b47ef2"),
			common.HexToHash("426fcb404ab2d5d8e61a3d918108006bbb0a9be65e92235bb10eefbdb6dcd053"),
			common.HexToHash("48078cfed56339ea54962e72c37c7f588fc4f8e5bc173827ba75cb10a63a96a5"),
			common.HexToHash("5723d2c3a83af9b735e3b7f21531e5623d183a9095a56604ead41f3582fdfb75"),
		}
		storage = storageMap{
			keys[0]: {Key: &common.Hash{0x02}, Value: common.Hash{0x01}},
			keys[1]: {Key: &common.Hash{0x04}, Value: common.Hash{0x02}},
			keys[2]: {Key: &common.Hash{0x01}, Value: common.Hash{0x03}},
			keys[3]: {Key: &common.Hash{0x03}, Value: common.Hash{0x04}},
		}
	)
	for _, entry := range storage {
		state.SetState(addr, *entry.Key, entry.Value)
	}

	// Check a few combinations of limit and start/end.
	tests := []struct {
		start []byte
		limit int
		want  StorageRangeResult
	}{
		{
			start: []byte{}, limit: 0,
			want: StorageRangeResult{storageMap{}, &keys[0]},
		},
		{
			start: []byte{}, limit: 100,
			want: StorageRangeResult{storage, nil},
		},
		{
			start: []byte{}, limit: 2,
			want: StorageRangeResult{storageMap{keys[0]: storage[keys[0]], keys[1]: storage[keys[1]]}, &keys[2]},
		},
		{
			start: []byte{0x00}, limit: 4,
			want: StorageRangeResult{storage, nil},
		},
		{
			start: []byte{0x40}, limit: 2,
			want: StorageRangeResult{storageMap{keys[1]: storage[keys[1]], keys[2]: storage[keys[2]]}, &keys[3]},
		},
	}
	for _, test := range tests {
		result, err := storageRangeAt(state.StorageTrie(addr), test.start, test.limit)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(result, test.want) {
			t.Fatalf("wrong result for range 0x%x.., limit %d:\ngot %s\nwant %s",
				test.start, test.limit, dumper.Sdump(result), dumper.Sdump(&test.want))
		}
	}
}
