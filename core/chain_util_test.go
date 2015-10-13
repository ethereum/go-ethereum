// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = common.String2Big(ext.ParentTimestamp).Uint64()
	d.ParentDifficulty = common.String2Big(ext.ParentDifficulty)
	d.CurrentTimestamp = common.String2Big(ext.CurrentTimestamp).Uint64()
	d.CurrentBlocknumber = common.String2Big(ext.CurrentBlocknumber)
	d.CurrentDifficulty = common.String2Big(ext.CurrentDifficulty)

	return nil
}

func TestDifficulty(t *testing.T) {
	file, err := os.Open("../tests/files/BasicTests/difficulty.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	tests := make(map[string]diffTest)
	err = json.NewDecoder(file).Decode(&tests)
	if err != nil {
		t.Fatal(err)
	}

	for name, test := range tests {
		number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
		diff := CalcDifficulty(test.CurrentTimestamp, test.ParentTimestamp, number, test.ParentDifficulty)
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}

// Tests block header storage and retrieval operations.
func TestHeaderStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test header to move around the database and make sure it's really new
	header := &types.Header{Extra: []byte("test header")}
	if entry := GetHeader(db, header.Hash()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	// Write and verify the header in the database
	if err := WriteHeader(db, header); err != nil {
		t.Fatalf("Failed to write header into database: %v", err)
	}
	if entry := GetHeader(db, header.Hash()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != header.Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, header)
	}
	if entry := GetHeaderRLP(db, header.Hash()); entry == nil {
		t.Fatalf("Stored header RLP not found")
	} else {
		hasher := sha3.NewKeccak256()
		hasher.Write(entry)

		if hash := common.BytesToHash(hasher.Sum(nil)); hash != header.Hash() {
			t.Fatalf("Retrieved RLP header mismatch: have %v, want %v", entry, header)
		}
	}
	// Delete the header and verify the execution
	DeleteHeader(db, header.Hash())
	if entry := GetHeader(db, header.Hash()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
}

// Tests block body storage and retrieval operations.
func TestBodyStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test body to move around the database and make sure it's really new
	body := &types.Body{Uncles: []*types.Header{{Extra: []byte("test header")}}}

	hasher := sha3.NewKeccak256()
	rlp.Encode(hasher, body)
	hash := common.BytesToHash(hasher.Sum(nil))

	if entry := GetBody(db, hash); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	// Write and verify the body in the database
	if err := WriteBody(db, hash, body); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBody(db, hash); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions)) != types.DeriveSha(types.Transactions(body.Transactions)) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(body.Uncles) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, body)
	}
	if entry := GetBodyRLP(db, hash); entry == nil {
		t.Fatalf("Stored body RLP not found")
	} else {
		hasher := sha3.NewKeccak256()
		hasher.Write(entry)

		if calc := common.BytesToHash(hasher.Sum(nil)); calc != hash {
			t.Fatalf("Retrieved RLP body mismatch: have %v, want %v", entry, body)
		}
	}
	// Delete the body and verify the execution
	DeleteBody(db, hash)
	if entry := GetBody(db, hash); entry != nil {
		t.Fatalf("Deleted body returned: %v", entry)
	}
}

// Tests block storage and retrieval operations.
func TestBlockStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test block to move around the database and make sure it's really new
	block := types.NewBlockWithHeader(&types.Header{
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
	})
	if entry := GetBlock(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	if entry := GetHeader(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	if entry := GetBody(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	// Write and verify the block in the database
	if err := WriteBlock(db, block); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
	if entry := GetHeader(db, block.Hash()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != block.Header().Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, block.Header())
	}
	if entry := GetBody(db, block.Hash()); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions)) != types.DeriveSha(block.Transactions()) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(block.Uncles()) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, &types.Body{block.Transactions(), block.Uncles()})
	}
	// Delete the block and verify the execution
	DeleteBlock(db, block.Hash())
	if entry := GetBlock(db, block.Hash()); entry != nil {
		t.Fatalf("Deleted block returned: %v", entry)
	}
	if entry := GetHeader(db, block.Hash()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
	if entry := GetBody(db, block.Hash()); entry != nil {
		t.Fatalf("Deleted body returned: %v", entry)
	}
}

// Tests that partial block contents don't get reassembled into full blocks.
func TestPartialBlockStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	block := types.NewBlockWithHeader(&types.Header{
		Extra:       []byte("test block"),
		UncleHash:   types.EmptyUncleHash,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
	})
	// Store a header and check that it's not recognized as a block
	if err := WriteHeader(db, block.Header()); err != nil {
		t.Fatalf("Failed to write header into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteHeader(db, block.Hash())

	// Store a body and check that it's not recognized as a block
	if err := WriteBody(db, block.Hash(), &types.Body{block.Transactions(), block.Uncles()}); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteBody(db, block.Hash())

	// Store a header and a body separately and check reassembly
	if err := WriteHeader(db, block.Header()); err != nil {
		t.Fatalf("Failed to write header into database: %v", err)
	}
	if err := WriteBody(db, block.Hash(), &types.Body{block.Transactions(), block.Uncles()}); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
}

// Tests block total difficulty storage and retrieval operations.
func TestTdStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test TD to move around the database and make sure it's really new
	hash, td := common.Hash{}, big.NewInt(314)
	if entry := GetTd(db, hash); entry != nil {
		t.Fatalf("Non existent TD returned: %v", entry)
	}
	// Write and verify the TD in the database
	if err := WriteTd(db, hash, td); err != nil {
		t.Fatalf("Failed to write TD into database: %v", err)
	}
	if entry := GetTd(db, hash); entry == nil {
		t.Fatalf("Stored TD not found")
	} else if entry.Cmp(td) != 0 {
		t.Fatalf("Retrieved TD mismatch: have %v, want %v", entry, td)
	}
	// Delete the TD and verify the execution
	DeleteTd(db, hash)
	if entry := GetTd(db, hash); entry != nil {
		t.Fatalf("Deleted TD returned: %v", entry)
	}
}

// Tests that canonical numbers can be mapped to hashes and retrieved.
func TestCanonicalMappingStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test canonical number and assinged hash to move around
	hash, number := common.Hash{0: 0xff}, uint64(314)
	if entry := GetCanonicalHash(db, number); entry != (common.Hash{}) {
		t.Fatalf("Non existent canonical mapping returned: %v", entry)
	}
	// Write and verify the TD in the database
	if err := WriteCanonicalHash(db, hash, number); err != nil {
		t.Fatalf("Failed to write canonical mapping into database: %v", err)
	}
	if entry := GetCanonicalHash(db, number); entry == (common.Hash{}) {
		t.Fatalf("Stored canonical mapping not found")
	} else if entry != hash {
		t.Fatalf("Retrieved canonical mapping mismatch: have %v, want %v", entry, hash)
	}
	// Delete the TD and verify the execution
	DeleteCanonicalHash(db, number)
	if entry := GetCanonicalHash(db, number); entry != (common.Hash{}) {
		t.Fatalf("Deleted canonical mapping returned: %v", entry)
	}
}

// Tests that head headers and head blocks can be assigned, individually.
func TestHeadStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	blockHead := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block header")})
	blockFull := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block full")})
	blockFast := types.NewBlockWithHeader(&types.Header{Extra: []byte("test block fast")})

	// Check that no head entries are in a pristine database
	if entry := GetHeadHeaderHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non head header entry returned: %v", entry)
	}
	if entry := GetHeadBlockHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non head block entry returned: %v", entry)
	}
	if entry := GetHeadFastBlockHash(db); entry != (common.Hash{}) {
		t.Fatalf("Non fast head block entry returned: %v", entry)
	}
	// Assign separate entries for the head header and block
	if err := WriteHeadHeaderHash(db, blockHead.Hash()); err != nil {
		t.Fatalf("Failed to write head header hash: %v", err)
	}
	if err := WriteHeadBlockHash(db, blockFull.Hash()); err != nil {
		t.Fatalf("Failed to write head block hash: %v", err)
	}
	if err := WriteHeadFastBlockHash(db, blockFast.Hash()); err != nil {
		t.Fatalf("Failed to write fast head block hash: %v", err)
	}
	// Check that both heads are present, and different (i.e. two heads maintained)
	if entry := GetHeadHeaderHash(db); entry != blockHead.Hash() {
		t.Fatalf("Head header hash mismatch: have %v, want %v", entry, blockHead.Hash())
	}
	if entry := GetHeadBlockHash(db); entry != blockFull.Hash() {
		t.Fatalf("Head block hash mismatch: have %v, want %v", entry, blockFull.Hash())
	}
	if entry := GetHeadFastBlockHash(db); entry != blockFast.Hash() {
		t.Fatalf("Fast head block hash mismatch: have %v, want %v", entry, blockFast.Hash())
	}
}

func TestMipmapBloom(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	receipt1 := new(types.Receipt)
	receipt1.Logs = vm.Logs{
		&vm.Log{Address: common.BytesToAddress([]byte("test"))},
		&vm.Log{Address: common.BytesToAddress([]byte("address"))},
	}
	receipt2 := new(types.Receipt)
	receipt2.Logs = vm.Logs{
		&vm.Log{Address: common.BytesToAddress([]byte("test"))},
		&vm.Log{Address: common.BytesToAddress([]byte("address1"))},
	}

	WriteMipmapBloom(db, 1, types.Receipts{receipt1})
	WriteMipmapBloom(db, 2, types.Receipts{receipt2})

	for _, level := range MIPMapLevels {
		bloom := GetMipmapBloom(db, 2, level)
		if !bloom.Test(new(big.Int).SetBytes([]byte("address1"))) {
			t.Error("expected test to be included on level:", level)
		}
	}

	// reset
	db, _ = ethdb.NewMemDatabase()
	receipt := new(types.Receipt)
	receipt.Logs = vm.Logs{
		&vm.Log{Address: common.BytesToAddress([]byte("test"))},
	}
	WriteMipmapBloom(db, 999, types.Receipts{receipt1})

	receipt = new(types.Receipt)
	receipt.Logs = vm.Logs{
		&vm.Log{Address: common.BytesToAddress([]byte("test 1"))},
	}
	WriteMipmapBloom(db, 1000, types.Receipts{receipt})

	bloom := GetMipmapBloom(db, 1000, 1000)
	if bloom.TestBytes([]byte("test")) {
		t.Error("test should not have been included")
	}
}

func TestMipmapChain(t *testing.T) {
	dir, err := ioutil.TempDir("", "mipmap")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		db, _   = ethdb.NewLDBDatabase(dir, 16)
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))

		hash1 = common.BytesToHash([]byte("topic1"))
	)
	defer db.Close()

	genesis := WriteGenesisBlockForTesting(db, GenesisAccount{addr, big.NewInt(1000000)})
	chain, receipts := GenerateChain(genesis, db, 1010, func(i int, gen *BlockGen) {
		var receipts types.Receipts
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{
				&vm.Log{
					Address: addr,
					Topics:  []common.Hash{hash1},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		case 1000:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{&vm.Log{Address: addr2}}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}

		}

		// store the receipts
		err := PutReceipts(db, receipts)
		if err != nil {
			t.Fatal(err)
		}
		WriteMipmapBloom(db, uint64(i+1), receipts)
	})
	for i, block := range chain {
		WriteBlock(db, block)
		if err := WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := WriteHeadBlockHash(db, block.Hash()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := PutBlockReceipts(db, block.Hash(), receipts[i]); err != nil {
			t.Fatal("error writing block receipts:", err)
		}
	}

	bloom := GetMipmapBloom(db, 0, 1000)
	if bloom.TestBytes(addr2[:]) {
		t.Error("address was included in bloom and should not have")
	}
}
