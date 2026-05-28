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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestSetHeadCleansTxLookupEntries pins the contract added for #33744:
// rewinding the chain past a block must delete every tx-lookup entry
// the block contained. Before the fix delFn left those entries behind,
// so a subsequent eth_getTransactionByHash resolved the hash to a
// block position that no longer existed.
func TestSetHeadCleansTxLookupEntries(t *testing.T) {
	var (
		key, _    = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address   = crypto.PubkeyToAddress(key.PublicKey)
		funds     = big.NewInt(1_000_000_000_000_000)
		chainHead = uint64(10)
		gspec     = &Genesis{
			Config:  params.TestChainConfig,
			Alloc:   types.GenesisAlloc{address: {Balance: funds}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
	)

	// Build a chain where every block carries one signed transaction so we
	// have a non-empty body for each height.
	_, blocks, _ := GenerateChainWithGenesis(gspec, ethash.NewFaker(), int(chainHead), func(i int, gen *BlockGen) {
		tx, err := types.SignTx(
			types.NewTransaction(gen.TxNonce(address), common.Address{0x00}, big.NewInt(1000), params.TxGas, gen.BaseFee(), nil),
			types.LatestSigner(gspec.Config), key,
		)
		if err != nil {
			t.Fatalf("sign tx for block %d: %v", i, err)
		}
		gen.AddTx(tx)
	})

	db := rawdb.NewMemoryDatabase()
	chain, err := NewBlockChain(db, gspec, ethash.NewFaker(), nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer chain.Stop()

	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("insert chain (failed at block %d): %v", n, err)
	}

	// Sanity check: every block's tx-lookup entry is present pre-rewind.
	for _, b := range blocks {
		for _, tx := range b.Transactions() {
			if rawdb.ReadTxLookupEntry(db, tx.Hash()) == nil {
				t.Fatalf("missing tx-lookup entry pre-rewind for block %d tx %s",
					b.NumberU64(), tx.Hash().Hex())
			}
		}
	}

	// Rewind past blocks 6..10. delFn must drop the tx-lookup entries for
	// every transaction those blocks contained.
	const rewindTo = uint64(5)
	if err := chain.SetHead(rewindTo); err != nil {
		t.Fatalf("SetHead(%d): %v", rewindTo, err)
	}

	// Lookups for kept blocks (1..rewindTo) must still resolve.
	for _, b := range blocks[:rewindTo] {
		for _, tx := range b.Transactions() {
			if rawdb.ReadTxLookupEntry(db, tx.Hash()) == nil {
				t.Fatalf("kept-block tx-lookup entry missing at height %d tx %s",
					b.NumberU64(), tx.Hash().Hex())
			}
		}
	}
	// Lookups for rewound blocks (rewindTo+1..) must be gone.
	for _, b := range blocks[rewindTo:] {
		for _, tx := range b.Transactions() {
			if lookup := rawdb.ReadTxLookupEntry(db, tx.Hash()); lookup != nil {
				t.Fatalf("stale tx-lookup entry survived rewind at height %d tx %s (lookup→block %d)",
					b.NumberU64(), tx.Hash().Hex(), *lookup)
			}
		}
	}
}
