// Copyright 2023 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/crypto"
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
		if err := tr.UpdateAccount(addr, acct, 0); err != nil {
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

func TestVerkleRollBack(t *testing.T) {
	db := newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.PathScheme)
	tr, _ := NewVerkleTrie(types.EmptyVerkleHash, db, utils.NewPointCache(100))

	for addr, acct := range accounts {
		// create more than 128 chunks of code
		code := make([]byte, 129*32)
		for i := 0; i < len(code); i += 2 {
			code[i] = 0x60
			code[i+1] = byte(i % 256)
		}
		if err := tr.UpdateAccount(addr, acct, len(code)); err != nil {
			t.Fatalf("Failed to update account, %v", err)
		}
		for key, val := range storages[addr] {
			if err := tr.UpdateStorage(addr, key.Bytes(), val); err != nil {
				t.Fatalf("Failed to update account, %v", err)
			}
		}
		hash := crypto.Keccak256Hash(code)
		if err := tr.UpdateContractCode(addr, hash, code); err != nil {
			t.Fatalf("Failed to update contract, %v", err)
		}
	}

	// Check that things were created
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

	// ensure there is some code in the 2nd group of the 1st account
	keyOf2ndGroup := utils.CodeChunkKeyWithEvaluatedAddress(tr.cache.Get(common.Address{1}.Bytes()), uint256.NewInt(128))
	chunk, err := tr.root.Get(keyOf2ndGroup, nil)
	if err != nil {
		t.Fatalf("Failed to get account, %v", err)
	}
	if len(chunk) == 0 {
		t.Fatal("account was not created ")
	}

	// Rollback first account and check that it is gone
	addr1 := common.Address{1}
	err = tr.RollBackAccount(addr1)
	if err != nil {
		t.Fatalf("error rolling back address 1: %v", err)
	}

	// ensure the account is gone
	stored, err := tr.GetAccount(addr1)
	if err != nil {
		t.Fatalf("Failed to get account, %v", err)
	}
	if stored != nil {
		t.Fatal("account was not deleted")
	}

	// ensure that the last code chunk is also gone from the tree
	chunk, err = tr.root.Get(keyOf2ndGroup, nil)
	if err != nil {
		t.Fatalf("Failed to get account, %v", err)
	}
	if len(chunk) != 0 {
		t.Fatal("account was not deleted")
	}
}
