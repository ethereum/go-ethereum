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
	"crypto/ecdsa"
	"crypto/sha256"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// createV1BlobTx creates a blob transaction with version 1 sidecar for testing.
func createV1BlobTx(nonce uint64, key *ecdsa.PrivateKey) *types.Transaction {
	blob := &kzg4844.Blob{byte(nonce)}
	commitment, _ := kzg4844.BlobToCommitment(blob)
	cellProofs, _ := kzg4844.ComputeCellProofs(blob)

	blobtx := &types.BlobTx{
		ChainID:    uint256.MustFromBig(params.MainnetChainConfig.ChainID),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(1),
		GasFeeCap:  uint256.NewInt(1000),
		Gas:        21000,
		BlobFeeCap: uint256.NewInt(100),
		BlobHashes: []common.Hash{kzg4844.CalcBlobHashV1(sha256.New(), &commitment)},
		Value:      uint256.NewInt(100),
		Sidecar:    types.NewBlobTxSidecar(types.BlobSidecarVersion1, []kzg4844.Blob{*blob}, []kzg4844.Commitment{commitment}, cellProofs),
	}
	return types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), blobtx)
}

func TestConversionQueueBasic(t *testing.T) {
	queue := newConversionQueue()
	defer queue.close()

	key, _ := crypto.GenerateKey()
	tx := makeTx(0, 1, 1, 1, key)
	if err := queue.convert(tx); err != nil {
		t.Fatalf("Expected successful conversion, got error: %v", err)
	}
	if tx.BlobTxSidecar().Version != types.BlobSidecarVersion1 {
		t.Errorf("Expected sidecar version to be %d, got %d", types.BlobSidecarVersion1, tx.BlobTxSidecar().Version)
	}
}

func TestConversionQueueV1BlobTx(t *testing.T) {
	queue := newConversionQueue()
	defer queue.close()

	key, _ := crypto.GenerateKey()
	tx := createV1BlobTx(0, key)
	version := tx.BlobTxSidecar().Version

	err := queue.convert(tx)
	if err != nil {
		t.Fatalf("Expected successful conversion, got error: %v", err)
	}
	if tx.BlobTxSidecar().Version != version {
		t.Errorf("Expected sidecar version to remain %d, got %d", version, tx.BlobTxSidecar().Version)
	}
}

func TestConversionQueueClosed(t *testing.T) {
	queue := newConversionQueue()

	// Close the queue first
	queue.close()
	key, _ := crypto.GenerateKey()
	tx := makeTx(0, 1, 1, 1, key)

	err := queue.convert(tx)
	if err == nil {
		t.Fatal("Expected error when converting on closed queue, got nil")
	}
}

func TestConversionQueueDoubleClose(t *testing.T) {
	queue := newConversionQueue()
	queue.close()
	queue.close() // Should not panic
}

func TestConversionQueueAutoRestartBatch(t *testing.T) {
	queue := newConversionQueue()
	defer queue.close()

	key, _ := crypto.GenerateKey()

	// Create a heavy transaction to ensure the first batch runs long enough
	// for subsequent tasks to be queued while it is active.
	heavy := makeMultiBlobTx(0, 1, 1, 1, int(params.BlobTxMaxBlobs), 0, key, types.BlobSidecarVersion0)

	var wg sync.WaitGroup
	wg.Add(1)
	heavyDone := make(chan error, 1)
	go func() {
		defer wg.Done()
		heavyDone <- queue.convert(heavy)
	}()

	// Give the conversion worker a head start so that the following tasks are
	// enqueued while the first batch is running.
	time.Sleep(200 * time.Millisecond)

	tx1 := makeTx(1, 1, 1, 1, key)
	tx2 := makeTx(2, 1, 1, 1, key)

	wg.Add(2)
	done1 := make(chan error, 1)
	done2 := make(chan error, 1)
	go func() { defer wg.Done(); done1 <- queue.convert(tx1) }()
	go func() { defer wg.Done(); done2 <- queue.convert(tx2) }()

	select {
	case err := <-done1:
		if err != nil {
			t.Fatalf("tx1 conversion error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for tx1 conversion")
	}

	select {
	case err := <-done2:
		if err != nil {
			t.Fatalf("tx2 conversion error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for tx2 conversion")
	}

	select {
	case err := <-heavyDone:
		if err != nil {
			t.Fatalf("heavy conversion error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for heavy conversion")
	}

	wg.Wait()

	if tx1.BlobTxSidecar().Version != types.BlobSidecarVersion1 {
		t.Fatalf("tx1 sidecar version mismatch: have %d, want %d", tx1.BlobTxSidecar().Version, types.BlobSidecarVersion1)
	}
	if tx2.BlobTxSidecar().Version != types.BlobSidecarVersion1 {
		t.Fatalf("tx2 sidecar version mismatch: have %d, want %d", tx2.BlobTxSidecar().Version, types.BlobSidecarVersion1)
	}
}
