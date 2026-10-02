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

package blobpool

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/billy"
)

// TestLimboLegacyMigration checks that a limbo entry in the legacy limboBlob
// layout is flagged for conversion by newLimbo instead of being dropped, so the
// conversion queue can migrate it to the blobTxForPool layout.
func TestLimboLegacyMigration(t *testing.T) {
	key, _ := crypto.GenerateKey()
	tx := makeMultiBlobTx(0, 10, 100, 100, 2, 0, key)

	dir := t.TempDir()

	// Write a single entry using the legacy on-disk layout.
	store, err := billy.Open(billy.Options{Path: dir}, newSlotterEIP7594(params.BlobTxMaxBlobs), nil)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	legacy := struct {
		TxHash common.Hash
		Block  uint64
		Tx     *types.Transaction
	}{tx.Hash(), 42, tx}
	data, err := rlp.EncodeToBytes(&legacy)
	if err != nil {
		t.Fatalf("failed to encode legacy entry: %v", err)
	}
	id, err := store.Put(data)
	if err != nil {
		t.Fatalf("failed to store legacy entry: %v", err)
	}
	store.Close()

	// Open the limbo, which should flag the legacy entry for conversion.
	l, convert, err := newLimbo(new(params.ChainConfig), dir)
	if err != nil {
		t.Fatalf("failed to open limbo: %v", err)
	}
	defer l.Close()

	if len(convert) != 1 || convert[0] != id {
		t.Fatalf("legacy entry not flagged for conversion: got %v, want [%d]", convert, id)
	}
}
