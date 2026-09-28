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

package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

func TestBlobPoolPriceBumpAppliesToLegacyPool(t *testing.T) {
	stack, err := node.New(&node.Config{DataDir: t.TempDir(), P2P: p2p.Config{ListenAddr: "127.0.0.1:0", NoDiscovery: true}})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}
	defer stack.Close()

	config := ethconfig.Defaults
	config.Genesis = &core.Genesis{Config: params.MergedTestChainConfig, Difficulty: common.Big0}
	config.BlobPool.PriceBump = 50
	if _, err := New(stack, &config); err != nil {
		t.Fatalf("failed to create eth service: %v", err)
	}
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	if config.TxPool.BlobPriceBump != 50 {
		t.Fatalf("legacypool blob price bump: have %d, want 50", config.TxPool.BlobPriceBump)
	}
	if config.TxPool.Limbo == "" {
		t.Fatal("legacypool limbo directory not set")
	}
}
