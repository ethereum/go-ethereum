// Copyright 2019 The go-ethereum Authors
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
package chaindb

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type testWriter interface {
	WriteHeadHeaderHash(common.Hash)
	WriteHeadBlockHash(common.Hash)
	WriteHeadFastBlockHash(common.Hash)
	WriteCanonicalHash(uint64, common.Hash)
	DeleteCanonicalHash(uint64)
	writeHeaderNumber(common.Hash, uint64)
	WriteHeader(*types.Header)
	DeleteHeader(common.Hash, uint64)
	WriteBody(common.Hash, uint64, *types.Body)
	DeleteBody(common.Hash, uint64)
	WriteBlock(*types.Block)
	DeleteBlock(common.Hash, uint64)
	WriteReceipts(common.Hash, uint64, types.Receipts)
	DeleteReceipts(common.Hash, uint64)
	WriteTD(common.Hash, uint64, *big.Int)
	DeleteTD(common.Hash, uint64)
	WriteTxLookupEntries(*types.Block)
	DeleteTxLookupEntry(common.Hash)
	WritePreimages(preimages map[common.Hash][]byte)
}

var tests = []struct {
	name          string
	createWrapper func(*ChainDB) *testWriterWrapper
}{
	{
		"ChainDB",
		chainDBWrapper,
	},
	{
		"Batch",
		batchWrapper,
	},
}

func chainDBWrapper(chainDB *ChainDB) *testWriterWrapper {
	return &testWriterWrapper{
		chainDB,
		func() error {
			return nil
		},
	}
}

func batchWrapper(chainDB *ChainDB) *testWriterWrapper {
	batch := chainDB.NewBatch()
	return &testWriterWrapper{
		batch,
		func() error {
			return batch.Write()
		},
	}
}

type testWriterWrapper struct {
	writer testWriter
	done   func() error
}

func TestChainDB_ReadWriteHeadHeaderHash(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			want := common.HexToHash("0x1")

			if got := chainDB.ReadHeadHeaderHash(); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadHeadHeaderHash() = %v, want %v", got, common.Hash{})
			}

			wrapper.writer.WriteHeadHeaderHash(want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadHeadHeaderHash(); got != want {
				t.Fatalf("chainDB.ReadHeadHeaderHash() = %v, want %v", got, want)
			}
		})
	}
}

func TestChainDB_ReadWriteHeadBlockHash(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			want := common.HexToHash("0x1")

			if got := chainDB.ReadHeadBlockHash(); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadHeadBlockHash() = %v, want %v", got, common.Hash{})
			}

			wrapper.writer.WriteHeadBlockHash(want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadHeadBlockHash(); got != want {
				t.Fatalf("chainDB.ReadHeadBlockHash() = %v, want %v", got, want)
			}
		})
	}
}

func TestChainDB_ReadWriteHeadFastBlockHash(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			want := common.HexToHash("0x1")

			if got := chainDB.ReadHeadFastBlockHash(); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadHeadFastBlockHash() = %v, want %v", got, common.Hash{})
			}

			wrapper.writer.WriteHeadFastBlockHash(want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadHeadFastBlockHash(); got != want {
				t.Fatalf("chainDB.ReadFastHeadBlockHash() = %v, want %v", got, want)
			}
		})
	}
}

func TestChainDB_ReadWriteDeleteCanonicalHash(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			number := uint64(22)
			want := common.HexToHash("0x1")

			if got := chainDB.ReadCanonicalHash(number); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadCanonicalHash(%d) = %v, want %v", number, got, common.Hash{})
			}

			wrapper.writer.WriteCanonicalHash(number, want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadCanonicalHash(number); got != want {
				t.Fatalf("chainDB.ReadCanonicalHash(%d) = %+v, want %+v", number, got, want)
			}

			wrapper.writer.DeleteCanonicalHash(number)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadCanonicalHash(number); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadCanonicalHash(%d) = %v, want %v", number, got, common.Hash{})
			}
		})
	}
}

func TestChainDB_ReadWriteHeaderNumber(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			hash := common.HexToHash("0x1")
			want := uint64(1)

			if got := chainDB.ReadHeaderNumber(hash); got != nil {
				t.Fatalf("chainDB.ReadHeaderNumber(%q) = %d, want <nil>", hash.String(), *got)
			}

			wrapper.writer.writeHeaderNumber(hash, want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadHeaderNumber(hash); got == nil {
				t.Fatalf("chainDB.ReadHeaderNumber(%q) = <nil>, want %d", hash.String(), want)
			} else if *got != want {
				t.Fatalf("chainDB.ReadHeaderNumber(%q) = %d, want %d", hash.String(), *got, want)
			}
		})
	}

}

func TestChainDB_HasReadWriteDeleteHeader(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			want := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(2), Time: big.NewInt(3), Extra: []byte("test header")}

			if present := chainDB.HasHeader(want.Hash(), want.Number.Uint64()); present {
				t.Fatalf("chainDB.HasHeader(%q, %d) = true, want false", want.Hash().String(), want.Number.Uint64())
			}

			if got := chainDB.ReadHeader(want.Hash(), want.Number.Uint64()); got != nil {
				t.Fatalf("chainDB.ReadHeader(%q, %d) = %+v, want <nil>", want.Hash().String(), want.Number.Uint64(), got)
			}

			wrapper.writer.WriteHeader(want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasHeader(want.Hash(), want.Number.Uint64()); !present {
				t.Fatalf("chainDB.HasHeader(%q, %d) = false, want true", want.Hash().String(), want.Number.Uint64())
			}

			got := chainDB.ReadHeader(want.Hash(), want.Number.Uint64())
			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(big.Int{})); diff != "" {
				t.Fatalf("chainDB.ReadHeader(%q, %d) headers differ (-want +got):\n%s", want.Hash().String(), want.Number.Uint64(), diff)
			}

			wrapper.writer.DeleteHeader(want.Hash(), want.Number.Uint64())
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasHeader(want.Hash(), want.Number.Uint64()); present {
				t.Fatalf("chainDB.HasHeader(%q, %d) = true, want false", want.Hash().String(), want.Number.Uint64())
			}

			if got := chainDB.ReadHeader(want.Hash(), want.Number.Uint64()); got != nil {
				t.Fatalf("chainDB.ReadHeader(%q, %d) = %+v, want <nil>", want.Hash().String(), want.Number.Uint64(), got)
			}
		})
	}
}

func TestChainDB_HasReadWriteDeleteBody(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			hash := common.HexToHash("0x1")
			number := uint64(22)
			tx := types.NewTransaction(1, common.HexToAddress("0x2"), big.NewInt(333), 4444, big.NewInt(55555), []byte{66, 66, 66})
			want := &types.Body{Transactions: []*types.Transaction{tx}, Uncles: []*types.Header{{Number: big.NewInt(1), Difficulty: big.NewInt(2), Time: big.NewInt(3), Extra: []byte("test ommer header")}}}

			if present := chainDB.HasBody(hash, number); present {
				t.Fatalf("chainDB.HasBody(%q, %d) = true, want false", hash.String(), number)
			}

			if got := chainDB.ReadBody(hash, number); got != nil {
				t.Fatalf("chainDB.ReadBody(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}

			wrapper.writer.WriteBody(hash, number, want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasBody(hash, number); !present {
				t.Fatalf("chainDB.HasBody(%q, %d) = false, want true", hash.String(), number)
			}

			got := chainDB.ReadBody(hash, number)
			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(big.Int{}, types.Transaction{})); diff != "" {
				t.Fatalf("chainDB.ReadBody(%q, %d) bodies differ (-want +got):\n%s", hash.String(), number, diff)
			}

			wrapper.writer.DeleteBody(hash, number)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasBody(hash, number); present {
				t.Fatalf("chainDB.HasBody(%q, %d) = true, want false", hash, number)
			}

			if got := chainDB.ReadBody(hash, number); got != nil {
				t.Fatalf("chainDB.ReadBody(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}
		})
	}
}

func TestChainDB_ReadWriteDeleteBlock(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			header := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(2), Time: big.NewInt(3), Extra: []byte("test header")}
			tx := types.NewTransaction(1, common.HexToAddress("0x2"), big.NewInt(333), 4444, big.NewInt(55555), []byte{66, 66, 66})
			ommerHeader := &types.Header{Number: big.NewInt(4), Difficulty: big.NewInt(5), Time: big.NewInt(6), Extra: []byte("test ommer header")}
			want := types.NewBlock(header, []*types.Transaction{tx}, []*types.Header{ommerHeader}, nil)

			if got := chainDB.ReadBlock(want.Hash(), want.NumberU64()); got != nil {
				t.Fatalf("chainDB.ReadBlock(%q, %d) = %+v, want <nil>", want.Hash().String(), want.NumberU64(), got)
			}

			wrapper.writer.WriteBlock(want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			got := chainDB.ReadBlock(want.Hash(), want.NumberU64())
			if diff := cmp.Diff(want, got, cmpopts.IgnoreUnexported(big.Int{}, types.Block{}, types.Transaction{})); diff != "" {
				t.Fatalf("chainDB.ReadBlock(%q, %d) bodies differ (-want +got):\n%s", want.Hash().String(), want.NumberU64(), diff)
			}

			wrapper.writer.DeleteBlock(want.Hash(), want.NumberU64())
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadBlock(want.Hash(), want.NumberU64()); got != nil {
				t.Fatalf("chainDB.ReadBlock(%q, %d) = %+v, want <nil>", want.Hash().String(), want.NumberU64(), got)
			}
		})
	}
}

func TestChainDB_HasReadWriteDeleteReceipts(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			hash := common.HexToHash("0x1")
			number := uint64(22)
			want := types.Receipts{types.NewReceipt(nil, false, 333)}

			if present := chainDB.HasReceipts(hash, number); present {
				t.Fatalf("chainDB.HasReceipts(%q, %d) = true, want false", hash.String(), number)
			}

			if got := chainDB.ReadReceipts(hash, number); got != nil {
				t.Fatalf("chainDB.ReadReceipts(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}

			wrapper.writer.WriteReceipts(hash, number, want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasReceipts(hash, number); !present {
				t.Fatalf("chainDB.HasReceipts(%q, %d) = false, want true", hash.String(), number)
			}

			got := chainDB.ReadReceipts(hash, number)
			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("chainDB.ReadReceipts(%q, %d) bodies differ (-want +got):\n%s", hash.String(), number, diff)
			}

			wrapper.writer.DeleteReceipts(hash, number)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if present := chainDB.HasReceipts(hash, number); present {
				t.Fatalf("chainDB.HasReceipts(%q, %d) = true, want false", hash, number)
			}

			if got := chainDB.ReadReceipts(hash, number); got != nil {
				t.Fatalf("chainDB.ReadReceipts(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}
		})
	}
}

func cmpBigInt(a, b *big.Int) bool {
	fmt.Println("a", a)
	fmt.Println("b", b)
	return a.Cmp(b) == 0
}

func TestChainDB_ReadWriteDeleteTxLookupEntries(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			header := &types.Header{}
			tx := types.NewTransaction(1, common.HexToAddress("0x2"), big.NewInt(333), 4444, big.NewInt(55555), []byte{66, 66, 66})
			txs := []*types.Transaction{tx}
			block := types.NewBlock(header, txs, []*types.Header{}, nil)

			if got := chainDB.ReadTxLookupEntry(tx.Hash()); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadTxLookupEntry(%q) = %s, want %s", tx.Hash(), got.String(), (common.Hash{}).String())
			}

			if got, blockHash, blockNumber, txIndex := chainDB.ReadTransaction(tx.Hash()); got != nil {
				t.Fatalf("chainDB.ReadTransaction(%q) = %+v, %s, %d, %d, want <nil>, %s, 0, 0", tx.Hash(), got, blockHash.String(), blockNumber, txIndex, tx.Hash().String())
			}

			// WriteBlock ensures the hash-to-number mapping and block bodies for transaction retrieval are present.
			wrapper.writer.WriteBlock(block)
			wrapper.writer.WriteTxLookupEntries(block)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadTxLookupEntry(tx.Hash()); got != block.Hash() {
				t.Fatalf("chainDB.ReadTxLookupEntry(%q) = %s, want %s", tx.Hash(), got.String(), block.Hash().String())
			}

			got, blockHash, blockNumber, txIndex := chainDB.ReadTransaction(tx.Hash())
			if diff := cmp.Diff(tx, got, cmpopts.IgnoreUnexported(types.Transaction{})); diff != "" {
				t.Fatalf("chainDB.ReadTransaction(%q) differ (-want +got):\n%s", tx.Hash().String(), diff)
			}
			if blockHash != block.Hash() || blockNumber != block.NumberU64() || txIndex != 0 {
				t.Fatalf("chainDB.ReadTransaction(%q) = ..., %s, %d, %d, want ..., %s, %d, %d", tx.Hash().String(), blockHash.String(), blockNumber, txIndex, block.Hash().String(), block.NumberU64(), 0)
			}

			wrapper.writer.DeleteTxLookupEntry(tx.Hash())
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadTxLookupEntry(tx.Hash()); got != (common.Hash{}) {
				t.Fatalf("chainDB.ReadTxLookupEntry(%q) = %s, want %s", tx.Hash(), got.String(), (common.Hash{}).String())
			}

			if got, blockHash, blockNumber, txIndex := chainDB.ReadTransaction(tx.Hash()); got != nil {
				t.Fatalf("chainDB.ReadTransaction(%q) = %+v, %s, %d, %d, want <nil>, %s, 0, 0", tx.Hash(), got, blockHash.String(), blockNumber, txIndex, tx.Hash().String())
			}
		})
	}
}

func TestChainDB_ReadWriteDeleteTD(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			hash := common.HexToHash("0x1")
			number := uint64(22)
			want := big.NewInt(3333)

			if got := chainDB.ReadTD(hash, number); got != nil {
				t.Fatalf("chainDB.ReadTD(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}

			wrapper.writer.WriteTD(hash, number, want)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadTD(hash, number); got == nil || got.Cmp(want) != 0 {
				t.Fatalf("chainDB.ReadTD(%q, %d) = %+v, want %+v", hash.String(), number, got, want)
			}

			wrapper.writer.DeleteTD(hash, number)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			if got := chainDB.ReadTD(hash, number); got != nil {
				t.Fatalf("chainDB.ReadTD(%q, %d) = %+v, want <nil>", hash.String(), number, got)
			}
		})
	}
}

func TestChainDB_ReadWritePreimages(t *testing.T) {
	for _, tc := range tests {
		tc := tc // Capture test case.
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := ethdb.NewMemDatabase()
			chainDB := Wrap(db)
			wrapper := tc.createWrapper(chainDB)

			preimages := map[common.Hash][]byte{
				common.HexToHash("0x1"): common.Hex2Bytes("0xa"),
				common.HexToHash("0x2"): common.Hex2Bytes("0xb"),
				common.HexToHash("0x3"): common.Hex2Bytes("0xc"),
			}

			for preimage := range preimages {
				if got := chainDB.ReadPreimage(preimage); got != nil {
					t.Fatalf("chainDB.ReadPreimage(%q) = %s, want <nil>", preimage, hex.EncodeToString(got))
				}
			}

			wrapper.writer.WritePreimages(preimages)
			if err := wrapper.done(); err != nil {
				t.Fatalf("wrapper.done() = %v, want <nil>", err)
			}

			for preimage, want := range preimages {
				if got := chainDB.ReadPreimage(preimage); !bytes.Equal(got, want) {
					t.Fatalf("chainDB.ReadPreimage(%q) = %s, want %s", preimage, hex.EncodeToString(got), hex.EncodeToString(want))
				}
			}
		})
	}
}
