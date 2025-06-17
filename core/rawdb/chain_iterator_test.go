// Copyright 2020 The go-ethereum Authors
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

package rawdb

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

func initDatabaseWithTransactions(db ethdb.Database) ([]*types.Block, []*types.Transaction) {
	var blocks []*types.Block
	var txs []*types.Transaction
	to := common.BytesToAddress([]byte{0x11})

	// Write empty genesis block
	block := types.NewBlock(&types.Header{Number: big.NewInt(int64(0))}, nil, nil, newTestHasher())
	WriteBlock(db, block)
	WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	blocks = append(blocks, block)

	// Create transactions.
	for i := uint64(1); i <= 10; i++ {
		var tx *types.Transaction
		if i%2 == 0 {
			tx = types.NewTx(&types.LegacyTx{
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		} else {
			tx = types.NewTx(&types.AccessListTx{
				ChainID:  big.NewInt(1337),
				Nonce:    i,
				GasPrice: big.NewInt(11111),
				Gas:      1111,
				To:       &to,
				Value:    big.NewInt(111),
				Data:     []byte{0x11, 0x11, 0x11},
			})
		}
		txs = append(txs, tx)
		block := types.NewBlock(&types.Header{Number: big.NewInt(int64(i))}, &types.Body{Transactions: types.Transactions{tx}}, nil, newTestHasher())
		WriteBlock(db, block)
		WriteCanonicalHash(db, block.Hash(), block.NumberU64())
		blocks = append(blocks, block)
	}

	return blocks, txs
}

func TestPruneTransactionIndex(t *testing.T) {
	chainDB := NewMemoryDatabase()
	blocks, _ := initDatabaseWithTransactions(chainDB)
	lastBlock := blocks[len(blocks)-1].NumberU64()
	pruneBlock := lastBlock - 3

	for i := uint64(0); i <= lastBlock; i++ {
		WriteTxLookupEntriesByBlock(chainDB, blocks[i])
	}
	WriteTxIndexTail(chainDB, 0)

	// Check all transactions are in index.
	for _, block := range blocks {
		for _, tx := range block.Transactions() {
			num := ReadTxLookupEntry(chainDB, tx.Hash())
			if num == nil || *num != block.NumberU64() {
				t.Fatalf("wrong TxLookup entry: %x -> %v", tx.Hash(), num)
			}
		}
	}

	PruneTransactionIndex(chainDB, pruneBlock)

	// Check transactions from old blocks not included.
	for _, block := range blocks {
		for _, tx := range block.Transactions() {
			num := ReadTxLookupEntry(chainDB, tx.Hash())
			if block.NumberU64() < pruneBlock && num != nil {
				t.Fatalf("TxLookup entry not removed: %x -> %v", tx.Hash(), num)
			}
			if block.NumberU64() >= pruneBlock && (num == nil || *num != block.NumberU64()) {
				t.Fatalf("wrong TxLookup entry after pruning: %x -> %v", tx.Hash(), num)
			}
		}
	}
}
