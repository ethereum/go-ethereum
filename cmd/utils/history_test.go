// Copyright 2023 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"bytes"
	"crypto/sha256"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

var (
	count uint64 = 128
	step  uint64 = 16
)

func TestHistoryImportAndExport(t *testing.T) {
	var (
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		genesis = &core.Genesis{
			Config: params.TestChainConfig,
			Alloc:  types.GenesisAlloc{address: {Balance: big.NewInt(1000000000000000000)}},
		}
		signer = types.LatestSigner(genesis.Config)
	)

	// Generate chain.
	db, blocks, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), int(count), func(i int, g *core.BlockGen) {
		if i == 0 {
			return
		}
		tx, err := types.SignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:    genesis.Config.ChainID,
			Nonce:      uint64(i - 1),
			GasTipCap:  common.Big0,
			GasFeeCap:  g.PrevBlock(0).BaseFee(),
			Gas:        50000,
			To:         &common.Address{0xaa},
			Value:      big.NewInt(int64(i)),
			Data:       nil,
			AccessList: nil,
		})
		if err != nil {
			t.Fatalf("error creating tx: %v", err)
		}
		g.AddTx(tx)
	})

	// Initialize BlockChain.
	chain, err := core.NewBlockChain(db, nil, genesis, nil, ethash.NewFaker(), vm.Config{}, nil)
	if err != nil {
		t.Fatalf("unable to initialize chain: %v", err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("error inserting chain: %v", err)
	}

	// Make temp directory for era files.
	dir, err := os.MkdirTemp("", "history-export-test")
	if err != nil {
		t.Fatalf("error creating temp test directory: %v", err)
	}
	defer os.RemoveAll(dir)

	// Export history to temp directory.
	if err := ExportHistory(chain, dir, 0, count, step); err != nil {
		t.Fatalf("error exporting history: %v", err)
	}

	// Read checksums.
	b, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		t.Fatalf("failed to read checksums: %v", err)
	}
	checksums := strings.Split(string(b), "\n")

	// Verify each Era.
	entries, _ := era.ReadDir(dir, "mainnet")
	for i, filename := range entries {
		func() {
			f, err := os.Open(filepath.Join(dir, filename))
			if err != nil {
				t.Fatalf("error opening era file: %v", err)
			}
			var (
				h   = sha256.New()
				buf = bytes.NewBuffer(nil)
			)
			if _, err := io.Copy(h, f); err != nil {
				t.Fatalf("unable to recalculate checksum: %v", err)
			}
			if got, want := common.BytesToHash(h.Sum(buf.Bytes()[:])).Hex(), checksums[i]; got != want {
				t.Fatalf("checksum %d does not match: got %s, want %s", i, got, want)
			}
			e, err := era.From(f)
			if err != nil {
				t.Fatalf("error opening era: %v", err)
			}
			defer e.Close()
			it, err := era.NewIterator(e)
			if err != nil {
				t.Fatalf("error making era reader: %v", err)
			}
			for j := 0; it.Next(); j++ {
				n := i*int(step) + j
				if it.Error() != nil {
					t.Fatalf("error reading block entry %d: %v", n, it.Error())
				}
				block, receipts, err := it.BlockAndReceipts()
				if err != nil {
					t.Fatalf("error reading block entry %d: %v", n, err)
				}
				want := chain.GetBlockByNumber(uint64(n))
				if want, got := uint64(n), block.NumberU64(); want != got {
					t.Fatalf("blocks out of order: want %d, got %d", want, got)
				}
				if want.Hash() != block.Hash() {
					t.Fatalf("block hash mismatch %d: want %s, got %s", n, want.Hash().Hex(), block.Hash().Hex())
				}
				if got := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil)); got != want.TxHash() {
					t.Fatalf("tx hash %d mismatch: want %s, got %s", n, want.TxHash(), got)
				}
				if got := types.CalcUncleHash(block.Uncles()); got != want.UncleHash() {
					t.Fatalf("uncle hash %d mismatch: want %s, got %s", n, want.UncleHash(), got)
				}
				if got := types.DeriveSha(receipts, trie.NewStackTrie(nil)); got != want.ReceiptHash() {
					t.Fatalf("receipt root %d mismatch: want %s, got %s", n, want.ReceiptHash(), got)
				}
			}
		}()
	}

	// Now import Era.
	db2, err := rawdb.NewDatabaseWithFreezer(rawdb.NewMemoryDatabase(), "", "", false)
	if err != nil {
		panic(err)
	}
	t.Cleanup(func() {
		db2.Close()
	})

	genesis.MustCommit(db2, triedb.NewDatabase(db, triedb.HashDefaults))
	imported, err := core.NewBlockChain(db2, nil, genesis, nil, ethash.NewFaker(), vm.Config{}, nil)
	if err != nil {
		t.Fatalf("unable to initialize chain: %v", err)
	}
	if err := ImportHistory(imported, db2, dir, "mainnet"); err != nil {
		t.Fatalf("failed to import chain: %v", err)
	}
	if have, want := imported.CurrentHeader(), chain.CurrentHeader(); have.Hash() != want.Hash() {
		t.Fatalf("imported chain does not match expected, have (%d, %s) want (%d, %s)", have.Number, have.Hash(), want.Number, want.Hash())
	}
}
