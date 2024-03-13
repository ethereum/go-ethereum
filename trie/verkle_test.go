// Copyright 2023 go-ethereum Authors
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

package trie

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

var (
	accounts = map[common.Address]*types.StateAccount{
		{1}: {
			Nonce:    100,
			Balance:  uint256.NewInt(100),
			CodeHash: common.Hash{0x1}.Bytes(),
		},
		{2}: {
			Nonce:    200,
			Balance:  uint256.NewInt(200),
			CodeHash: common.Hash{0x2}.Bytes(),
		},
	}
	storages = map[common.Address]map[common.Hash][]byte{
		{1}: {
			common.Hash{10}: []byte{10},
			common.Hash{11}: []byte{11},
			common.MaxHash:  []byte{0xff},
		},
		{2}: {
			common.Hash{20}: []byte{20},
			common.Hash{21}: []byte{21},
			common.MaxHash:  []byte{0xff},
		},
	}
)

func TestVerkleTreeReadWrite(t *testing.T) {
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.PathScheme)
	tr, _ := NewVerkleTrie(types.EmptyVerkleHash, db, utils.NewPointCache(100))

	for addr, acct := range accounts {
		if err := tr.UpdateAccount(addr, acct); err != nil {
			t.Fatalf("Failed to update account, %v", err)
		}
		for key, val := range storages[addr] {
			if err := tr.UpdateStorage(addr, key.Bytes(), val); err != nil {
				t.Fatalf("Failed to update account, %v", err)
			}
		}
	}

	for addr, acct := range accounts {
		stored, err := tr.GetAccount(addr)
		if err != nil {
			t.Fatalf("Failed to get account, %v", err)
		}
		if !reflect.DeepEqual(stored, acct) {
			t.Fatal("account is not matched")
		}
		for key, val := range storages[addr] {
			stored, err := tr.GetStorage(addr, key.Bytes())
			if err != nil {
				t.Fatalf("Failed to get storage, %v", err)
			}
			if !bytes.Equal(stored, val) {
				t.Fatal("storage is not matched")
			}
		}
	}
}
