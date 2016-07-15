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
	"bytes"
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

func TestDifficultyFrontier(t *testing.T) {
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
		diff := calcDifficultyFrontier(test.CurrentTimestamp, test.ParentTimestamp, number, test.ParentDifficulty)
		if diff.Cmp(test.CurrentDifficulty) != 0 {
			t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
		}
	}
}

// Tests block header storage and retrieval operations.
func TestHeaderStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	// Create a test header to move around the database and make sure it's really new
	header := &types.Header{Number: big.NewInt(42), Extra: []byte("test header")}
	if entry := GetHeader(db, header.Hash(), header.Number.Uint64()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	// Write and verify the header in the database
	if err := WriteHeader(db, header); err != nil {
		t.Fatalf("Failed to write header into database: %v", err)
	}
	if entry := GetHeader(db, header.Hash(), header.Number.Uint64()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != header.Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, header)
	}
	if entry := GetHeaderRLP(db, header.Hash(), header.Number.Uint64()); entry == nil {
		t.Fatalf("Stored header RLP not found")
	} else {
		hasher := sha3.NewKeccak256()
		hasher.Write(entry)

		if hash := common.BytesToHash(hasher.Sum(nil)); hash != header.Hash() {
			t.Fatalf("Retrieved RLP header mismatch: have %v, want %v", entry, header)
		}
	}
	// Delete the header and verify the execution
	DeleteHeader(db, header.Hash(), header.Number.Uint64())
	if entry := GetHeader(db, header.Hash(), header.Number.Uint64()); entry != nil {
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

	if entry := GetBody(db, hash, 0); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	// Write and verify the body in the database
	if err := WriteBody(db, hash, 0, body); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBody(db, hash, 0); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions)) != types.DeriveSha(types.Transactions(body.Transactions)) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(body.Uncles) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, body)
	}
	if entry := GetBodyRLP(db, hash, 0); entry == nil {
		t.Fatalf("Stored body RLP not found")
	} else {
		hasher := sha3.NewKeccak256()
		hasher.Write(entry)

		if calc := common.BytesToHash(hasher.Sum(nil)); calc != hash {
			t.Fatalf("Retrieved RLP body mismatch: have %v, want %v", entry, body)
		}
	}
	// Delete the body and verify the execution
	DeleteBody(db, hash, 0)
	if entry := GetBody(db, hash, 0); entry != nil {
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
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	if entry := GetHeader(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent header returned: %v", entry)
	}
	if entry := GetBody(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent body returned: %v", entry)
	}
	// Write and verify the block in the database
	if err := WriteBlock(db, block); err != nil {
		t.Fatalf("Failed to write block into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored block not found")
	} else if entry.Hash() != block.Hash() {
		t.Fatalf("Retrieved block mismatch: have %v, want %v", entry, block)
	}
	if entry := GetHeader(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored header not found")
	} else if entry.Hash() != block.Header().Hash() {
		t.Fatalf("Retrieved header mismatch: have %v, want %v", entry, block.Header())
	}
	if entry := GetBody(db, block.Hash(), block.NumberU64()); entry == nil {
		t.Fatalf("Stored body not found")
	} else if types.DeriveSha(types.Transactions(entry.Transactions)) != types.DeriveSha(block.Transactions()) || types.CalcUncleHash(entry.Uncles) != types.CalcUncleHash(block.Uncles()) {
		t.Fatalf("Retrieved body mismatch: have %v, want %v", entry, block.Body())
	}
	// Delete the block and verify the execution
	DeleteBlock(db, block.Hash(), block.NumberU64())
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Deleted block returned: %v", entry)
	}
	if entry := GetHeader(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Deleted header returned: %v", entry)
	}
	if entry := GetBody(db, block.Hash(), block.NumberU64()); entry != nil {
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
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteHeader(db, block.Hash(), block.NumberU64())

	// Store a body and check that it's not recognized as a block
	if err := WriteBody(db, block.Hash(), block.NumberU64(), block.Body()); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry != nil {
		t.Fatalf("Non existent block returned: %v", entry)
	}
	DeleteBody(db, block.Hash(), block.NumberU64())

	// Store a header and a body separately and check reassembly
	if err := WriteHeader(db, block.Header()); err != nil {
		t.Fatalf("Failed to write header into database: %v", err)
	}
	if err := WriteBody(db, block.Hash(), block.NumberU64(), block.Body()); err != nil {
		t.Fatalf("Failed to write body into database: %v", err)
	}
	if entry := GetBlock(db, block.Hash(), block.NumberU64()); entry == nil {
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
	if entry := GetTd(db, hash, 0); entry != nil {
		t.Fatalf("Non existent TD returned: %v", entry)
	}
	// Write and verify the TD in the database
	if err := WriteTd(db, hash, 0, td); err != nil {
		t.Fatalf("Failed to write TD into database: %v", err)
	}
	if entry := GetTd(db, hash, 0); entry == nil {
		t.Fatalf("Stored TD not found")
	} else if entry.Cmp(td) != 0 {
		t.Fatalf("Retrieved TD mismatch: have %v, want %v", entry, td)
	}
	// Delete the TD and verify the execution
	DeleteTd(db, hash, 0)
	if entry := GetTd(db, hash, 0); entry != nil {
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

// Tests that transactions and associated metadata can be stored and retrieved.
func TestTransactionStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	tx1 := types.NewTransaction(1, common.BytesToAddress([]byte{0x11}), big.NewInt(111), big.NewInt(1111), big.NewInt(11111), []byte{0x11, 0x11, 0x11})
	tx2 := types.NewTransaction(2, common.BytesToAddress([]byte{0x22}), big.NewInt(222), big.NewInt(2222), big.NewInt(22222), []byte{0x22, 0x22, 0x22})
	tx3 := types.NewTransaction(3, common.BytesToAddress([]byte{0x33}), big.NewInt(333), big.NewInt(3333), big.NewInt(33333), []byte{0x33, 0x33, 0x33})
	txs := []*types.Transaction{tx1, tx2, tx3}

	block := types.NewBlock(&types.Header{Number: big.NewInt(314)}, txs, nil, nil)

	// Check that no transactions entries are in a pristine database
	for i, tx := range txs {
		if txn, _, _, _ := GetTransaction(db, tx.Hash()); txn != nil {
			t.Fatalf("tx #%d [%x]: non existent transaction returned: %v", i, tx.Hash(), txn)
		}
	}
	// Insert all the transactions into the database, and verify contents
	if err := WriteTransactions(db, block); err != nil {
		t.Fatalf("failed to write transactions: %v", err)
	}
	for i, tx := range txs {
		if txn, hash, number, index := GetTransaction(db, tx.Hash()); txn == nil {
			t.Fatalf("tx #%d [%x]: transaction not found", i, tx.Hash())
		} else {
			if hash != block.Hash() || number != block.NumberU64() || index != uint64(i) {
				t.Fatalf("tx #%d [%x]: positional metadata mismatch: have %x/%d/%d, want %x/%v/%v", i, tx.Hash(), hash, number, index, block.Hash(), block.NumberU64(), i)
			}
			if tx.String() != txn.String() {
				t.Fatalf("tx #%d [%x]: transaction mismatch: have %v, want %v", i, tx.Hash(), txn, tx)
			}
		}
	}
	// Delete the transactions and check purge
	for i, tx := range txs {
		DeleteTransaction(db, tx.Hash())
		if txn, _, _, _ := GetTransaction(db, tx.Hash()); txn != nil {
			t.Fatalf("tx #%d [%x]: deleted transaction returned: %v", i, tx.Hash(), txn)
		}
	}
}

// Tests that receipts can be stored and retrieved.
func TestReceiptStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	receipt1 := &types.Receipt{
		PostState:         []byte{0x01},
		CumulativeGasUsed: big.NewInt(1),
		Logs: vm.Logs{
			&vm.Log{Address: common.BytesToAddress([]byte{0x11})},
			&vm.Log{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         big.NewInt(111111),
	}
	receipt2 := &types.Receipt{
		PostState:         []byte{0x02},
		CumulativeGasUsed: big.NewInt(2),
		Logs: vm.Logs{
			&vm.Log{Address: common.BytesToAddress([]byte{0x22})},
			&vm.Log{Address: common.BytesToAddress([]byte{0x02, 0x22})},
		},
		TxHash:          common.BytesToHash([]byte{0x22, 0x22}),
		ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         big.NewInt(222222),
	}
	receipts := []*types.Receipt{receipt1, receipt2}

	// Check that no receipt entries are in a pristine database
	for i, receipt := range receipts {
		if r := GetReceipt(db, receipt.TxHash); r != nil {
			t.Fatalf("receipt #%d [%x]: non existent receipt returned: %v", i, receipt.TxHash, r)
		}
	}
	// Insert all the receipts into the database, and verify contents
	if err := WriteReceipts(db, receipts); err != nil {
		t.Fatalf("failed to write receipts: %v", err)
	}
	for i, receipt := range receipts {
		if r := GetReceipt(db, receipt.TxHash); r == nil {
			t.Fatalf("receipt #%d [%x]: receipt not found", i, receipt.TxHash)
		} else {
			rlpHave, _ := rlp.EncodeToBytes(r)
			rlpWant, _ := rlp.EncodeToBytes(receipt)

			if bytes.Compare(rlpHave, rlpWant) != 0 {
				t.Fatalf("receipt #%d [%x]: receipt mismatch: have %v, want %v", i, receipt.TxHash, r, receipt)
			}
		}
	}
	// Delete the receipts and check purge
	for i, receipt := range receipts {
		DeleteReceipt(db, receipt.TxHash)
		if r := GetReceipt(db, receipt.TxHash); r != nil {
			t.Fatalf("receipt #%d [%x]: deleted receipt returned: %v", i, receipt.TxHash, r)
		}
	}
}

// Tests that receipts associated with a single block can be stored and retrieved.
func TestBlockReceiptStorage(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()

	receipt1 := &types.Receipt{
		PostState:         []byte{0x01},
		CumulativeGasUsed: big.NewInt(1),
		Logs: vm.Logs{
			&vm.Log{Address: common.BytesToAddress([]byte{0x11})},
			&vm.Log{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          common.BytesToHash([]byte{0x11, 0x11}),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         big.NewInt(111111),
	}
	receipt2 := &types.Receipt{
		PostState:         []byte{0x02},
		CumulativeGasUsed: big.NewInt(2),
		Logs: vm.Logs{
			&vm.Log{Address: common.BytesToAddress([]byte{0x22})},
			&vm.Log{Address: common.BytesToAddress([]byte{0x02, 0x22})},
		},
		TxHash:          common.BytesToHash([]byte{0x22, 0x22}),
		ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         big.NewInt(222222),
	}
	receipts := []*types.Receipt{receipt1, receipt2}

	// Check that no receipt entries are in a pristine database
	hash := common.BytesToHash([]byte{0x03, 0x14})
	if rs := GetBlockReceipts(db, hash, 0); len(rs) != 0 {
		t.Fatalf("non existent receipts returned: %v", rs)
	}
	// Insert the receipt slice into the database and check presence
	if err := WriteBlockReceipts(db, hash, 0, receipts); err != nil {
		t.Fatalf("failed to write block receipts: %v", err)
	}
	if rs := GetBlockReceipts(db, hash, 0); len(rs) == 0 {
		t.Fatalf("no receipts returned")
	} else {
		for i := 0; i < len(receipts); i++ {
			rlpHave, _ := rlp.EncodeToBytes(rs[i])
			rlpWant, _ := rlp.EncodeToBytes(receipts[i])

			if bytes.Compare(rlpHave, rlpWant) != 0 {
				t.Fatalf("receipt #%d: receipt mismatch: have %v, want %v", i, rs[i], receipts[i])
			}
		}
	}
	// Delete the receipt slice and check purge
	DeleteBlockReceipts(db, hash, 0)
	if rs := GetBlockReceipts(db, hash, 0); len(rs) != 0 {
		t.Fatalf("deleted receipts returned: %v", rs)
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
		db, _   = ethdb.NewLDBDatabase(dir, 0, 0)
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))

		hash1 = common.BytesToHash([]byte("topic1"))
	)
	defer db.Close()

	genesis := WriteGenesisBlockForTesting(db, GenesisAccount{addr, big.NewInt(1000000)})
	chain, receipts := GenerateChain(nil, genesis, db, 1010, func(i int, gen *BlockGen) {
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
		err := WriteReceipts(db, receipts)
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
		if err := WriteBlockReceipts(db, block.Hash(), block.NumberU64(), receipts[i]); err != nil {
			t.Fatal("error writing block receipts:", err)
		}
	}

	bloom := GetMipmapBloom(db, 0, 1000)
	if bloom.TestBytes(addr2[:]) {
		t.Error("address was included in bloom and should not have")
	}
}
