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

package blobpool

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// TestLimboUpdateRoundTrip checks that update() relocates a tracked blob
// transaction to a new block while keeping it pullable.
//
// This is the regression test for #34944. The previous implementation deleted
// the existing limbo entry before inserting the relocated one, so a failure of
// the second step would have permanently dropped the blob sidecar. The current
// implementation writes the new entry first and only drops the old one once
// the new one is durable; the round-trip below exercises that happy path.
func TestLimboUpdateRoundTrip(t *testing.T) {
	limbo, err := newLimbo(params.MainnetChainConfig, t.TempDir())
	if err != nil {
		t.Fatalf("failed to open limbo: %v", err)
	}
	defer limbo.Close()

	key, _ := crypto.GenerateKey()
	tx := makeTx(0, 1, 1, 1, key)
	hash := tx.Hash()

	if err := limbo.push(newBlobTxForPool(tx), 100); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if _, ok := limbo.index[hash]; !ok {
		t.Fatalf("tx not indexed after push")
	}
	oldID := limbo.index[hash]
	if _, ok := limbo.groups[100][oldID]; !ok {
		t.Fatalf("tx not in groups[100] after push")
	}

	limbo.update(hash, 101)

	if _, ok := limbo.groups[100]; ok {
		t.Fatalf("old block group not cleaned up after update")
	}
	newID, ok := limbo.index[hash]
	if !ok {
		t.Fatalf("tx no longer indexed after update")
	}
	if _, ok := limbo.groups[101][newID]; !ok {
		t.Fatalf("tx not in groups[101] after update")
	}

	pulled, err := limbo.pull(hash)
	if err != nil {
		t.Fatalf("pull after update failed: %v", err)
	}
	if pulled.Tx.Hash() != hash {
		t.Fatalf("pulled tx hash mismatch: got %x want %x", pulled.Tx.Hash(), hash)
	}
}
