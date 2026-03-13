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

package execdb

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestEraE(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		start       uint64
		preMerge    int
		postMerge   int
		accumulator bool // whether accumulator should exist
	}{
		{
			name:        "pre-merge",
			start:       0,
			preMerge:    128,
			postMerge:   0,
			accumulator: true,
		},
		{
			name:        "post-merge",
			start:       0,
			preMerge:    0,
			postMerge:   64,
			accumulator: false,
		},
		{
			name:        "transition",
			start:       0,
			preMerge:    32,
			postMerge:   32,
			accumulator: true,
		},
		{
			name:        "non-zero-start",
			start:       8192,
			preMerge:    64,
			postMerge:   0,
			accumulator: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := os.CreateTemp(t.TempDir(), "erae-test")
			if err != nil {
				t.Fatalf("error creating temp file: %v", err)
			}
			defer f.Close()

			// Build test data.
			type blockData struct {
				header, body, receipts []byte
				hash                   common.Hash
				td                     *big.Int
				difficulty             *big.Int
			}
			var (
				builder     = NewBuilder(f)
				blocks      []blockData
				totalBlocks = tt.preMerge + tt.postMerge
				finalTD     = big.NewInt(int64(tt.preMerge))
			)

			// Add pre-merge blocks.
			for i := 0; i < tt.preMerge; i++ {
				num := tt.start + uint64(i)
				blk := blockData{
					header:     mustEncode(&types.Header{Number: big.NewInt(int64(num)), Difficulty: big.NewInt(1)}),
					body:       mustEncode(&types.Body{Transactions: []*types.Transaction{types.NewTransaction(0, common.Address{byte(i)}, nil, 0, nil, nil)}}),
					receipts:   mustEncode([]types.SlimReceipt{{CumulativeGasUsed: uint64(i)}}),
					hash:       common.Hash{byte(i)},
					td:         big.NewInt(int64(i + 1)),
					difficulty: big.NewInt(1),
				}
				blocks = append(blocks, blk)
				if err := builder.AddRLP(blk.header, blk.body, blk.receipts, num, blk.hash, blk.td, blk.difficulty); err != nil {
					t.Fatalf("error adding pre-merge block %d: %v", i, err)
				}
			}

			// Add post-merge blocks.
			for i := 0; i < tt.postMerge; i++ {
				idx := tt.preMerge + i
				num := tt.start + uint64(idx)
				blk := blockData{
					header:     mustEncode(&types.Header{Number: big.NewInt(int64(num)), Difficulty: big.NewInt(0)}),
					body:       mustEncode(&types.Body{}),
					receipts:   mustEncode([]types.SlimReceipt{}),
					hash:       common.Hash{byte(idx)},
					difficulty: big.NewInt(0),
				}
				blocks = append(blocks, blk)
				if err := builder.AddRLP(blk.header, blk.body, blk.receipts, num, blk.hash, nil, big.NewInt(0)); err != nil {
					t.Fatalf("error adding post-merge block %d: %v", idx, err)
				}
			}

			// Finalize and check return values.
			epochID, err := builder.Finalize()
			if err != nil {
				t.Fatalf("error finalizing: %v", err)
			}

			// Verify epoch ID is always the last block hash.
			expectedLastHash := blocks[len(blocks)-1].hash
			if epochID != expectedLastHash {
				t.Fatalf("wrong epoch ID: want %s, got %s", expectedLastHash.Hex(), epochID.Hex())
			}

			// Verify accumulator presence.
			if tt.accumulator {
				if builder.Accumulator() == nil {
					t.Fatal("expected non-nil accumulator")
				}
			} else {
				if builder.Accumulator() != nil {
					t.Fatalf("expected nil accumulator, got %s", builder.Accumulator().Hex())
				}
			}

			// Open and verify the era file.
			e, err := Open(f.Name())
			if err != nil {
				t.Fatalf("failed to open era: %v", err)
			}
			defer e.Close()

			// Verify metadata.
			if e.Start() != tt.start {
				t.Fatalf("wrong start block: want %d, got %d", tt.start, e.Start())
			}
			if e.Count() != uint64(totalBlocks) {
				t.Fatalf("wrong block count: want %d, got %d", totalBlocks, e.Count())
			}

			// Verify accumulator in file.
			if tt.accumulator {
				accRoot, err := e.Accumulator()
				if err != nil {
					t.Fatalf("error getting accumulator: %v", err)
				}
				if accRoot != *builder.Accumulator() {
					t.Fatalf("accumulator mismatch: builder has %s, file contains %s",
						builder.Accumulator().Hex(), accRoot.Hex())
				}
			} else {
				if _, err := e.Accumulator(); err == nil {
					t.Fatal("expected error when reading accumulator from post-merge epoch")
				}
			}

			// Verify blocks via raw iterator.
			it, err := NewRawIterator(e)
			if err != nil {
				t.Fatalf("failed to make iterator: %v", err)
			}
			for i := 0; i < totalBlocks; i++ {
				if !it.Next() {
					t.Fatalf("expected more entries at %d", i)
				}
				if it.Error() != nil {
					t.Fatalf("unexpected error: %v", it.Error())
				}

				// Check header.
				rawHeader, err := io.ReadAll(it.Header)
				if err != nil {
					t.Fatalf("error reading header: %v", err)
				}
				if !bytes.Equal(rawHeader, blocks[i].header) {
					t.Fatalf("mismatched header at %d", i)
				}

				// Check body.
				rawBody, err := io.ReadAll(it.Body)
				if err != nil {
					t.Fatalf("error reading body: %v", err)
				}
				if !bytes.Equal(rawBody, blocks[i].body) {
					t.Fatalf("mismatched body at %d", i)
				}

				// Check receipts.
				rawReceipts, err := io.ReadAll(it.Receipts)
				if err != nil {
					t.Fatalf("error reading receipts: %v", err)
				}
				if !bytes.Equal(rawReceipts, blocks[i].receipts) {
					t.Fatalf("mismatched receipts at %d", i)
				}

				// Check TD (only for epochs that have TD stored).
				if tt.preMerge > 0 && it.TotalDifficulty != nil {
					rawTd, err := io.ReadAll(it.TotalDifficulty)
					if err != nil {
						t.Fatalf("error reading TD: %v", err)
					}
					slices.Reverse(rawTd)
					td := new(big.Int).SetBytes(rawTd)
					var expectedTD *big.Int
					if i < tt.preMerge {
						expectedTD = blocks[i].td
					} else {
						// Post-merge blocks in transition epoch use final TD.
						expectedTD = finalTD
					}
					if td.Cmp(expectedTD) != 0 {
						t.Fatalf("mismatched TD at %d: want %s, got %s", i, expectedTD, td)
					}
				}
			}

			// Verify random access.
			for _, blockNum := range []uint64{tt.start, tt.start + uint64(totalBlocks) - 1} {
				blk, err := e.GetBlockByNumber(blockNum)
				if err != nil {
					t.Fatalf("error getting block %d: %v", blockNum, err)
				}
				if blk.Number().Uint64() != blockNum {
					t.Fatalf("wrong block number: want %d, got %d", blockNum, blk.Number().Uint64())
				}
			}

			// Verify out-of-range access fails.
			if _, err := e.GetBlockByNumber(tt.start + uint64(totalBlocks)); err == nil {
				t.Fatal("expected error for out-of-range block")
			}
			if tt.start > 0 {
				if _, err := e.GetBlockByNumber(tt.start - 1); err == nil {
					t.Fatal("expected error for block before start")
				}
			}

			// Verify high-level iterator.
			hlIt, err := e.Iterator()
			if err != nil {
				t.Fatalf("failed to create iterator: %v", err)
			}
			count := 0
			for hlIt.Next() {
				blk, err := hlIt.Block()
				if err != nil {
					t.Fatalf("error getting block: %v", err)
				}
				if blk.Number().Uint64() != tt.start+uint64(count) {
					t.Fatalf("wrong block number: want %d, got %d", tt.start+uint64(count), blk.Number().Uint64())
				}
				count++
			}
			if hlIt.Error() != nil {
				t.Fatalf("iterator error: %v", hlIt.Error())
			}
			if count != totalBlocks {
				t.Fatalf("wrong iteration count: want %d, got %d", totalBlocks, count)
			}
		})
	}
}

// TestInitialTD tests the InitialTD calculation separately since it requires
// specific TD/difficulty values.
func TestInitialTD(t *testing.T) {
	t.Parallel()

	f, err := os.CreateTemp(t.TempDir(), "erae-initial-td-test")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}
	defer f.Close()

	builder := NewBuilder(f)

	// First block: difficulty=5, TD=10, so initial TD = 10-5 = 5.
	header := mustEncode(&types.Header{Number: big.NewInt(0), Difficulty: big.NewInt(5)})
	body := mustEncode(&types.Body{})
	receipts := mustEncode([]types.SlimReceipt{})

	if err := builder.AddRLP(header, body, receipts, 0, common.Hash{0}, big.NewInt(10), big.NewInt(5)); err != nil {
		t.Fatalf("error adding block: %v", err)
	}

	// Second block: difficulty=3, TD=13.
	header2 := mustEncode(&types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(3)})
	if err := builder.AddRLP(header2, body, receipts, 1, common.Hash{1}, big.NewInt(13), big.NewInt(3)); err != nil {
		t.Fatalf("error adding block: %v", err)
	}

	if _, err := builder.Finalize(); err != nil {
		t.Fatalf("error finalizing: %v", err)
	}

	e, err := Open(f.Name())
	if err != nil {
		t.Fatalf("failed to open era: %v", err)
	}
	defer e.Close()

	initialTD, err := e.InitialTD()
	if err != nil {
		t.Fatalf("error getting initial TD: %v", err)
	}

	// Initial TD should be TD[0] - Difficulty[0] = 10 - 5 = 5.
	if initialTD.Cmp(big.NewInt(5)) != 0 {
		t.Fatalf("wrong initial TD: want 5, got %s", initialTD)
	}
}

func mustEncode(obj any) []byte {
	b, err := rlp.EncodeToBytes(obj)
	if err != nil {
		panic(fmt.Sprintf("failed to encode obj: %v", err))
	}
	return b
}
