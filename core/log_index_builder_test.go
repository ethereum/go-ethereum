// Copyright 2026 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestBuildLogIndexForBlock pins the level-0 wrapper: one table per block,
// carrying the builder root of the block's entries including the parent's
// block entry (one-block delay).
func TestBuildLogIndexForBlock(t *testing.T) {
	parentHash := common.HexToHash("0x978ce0036b6d1c62d716045505587d15cc85a1def92f9f450937b6467295e517")
	receipts := types.Receipts{
		{TxHash: common.HexToHash("0xca2d12d1b8132de09d0d668cc87349dc70134bee3010e03ddb2d83f7160bd6e3"), Logs: []*types.Log{
			{Address: common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"), Topics: []common.Hash{
				common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
			}},
		}},
	}
	wantRoot := types.NewIndexBuilder()
	wantRoot.AddBlockEntries(parentHash, 42, receipts)
	tables := BuildLogIndexForBlock(42, parentHash, receipts)
	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}
	if tables[0].FirstBlock != 42 || tables[0].TableSize != 1 {
		t.Errorf("table = {first %d, size %d}, want {42, 1}", tables[0].FirstBlock, tables[0].TableSize)
	}
	if tables[0].Root != wantRoot.Build() {
		t.Errorf("root %x, want %x", tables[0].Root, wantRoot.Build())
	}
}
