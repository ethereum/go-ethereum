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
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/billy"
	"github.com/holiman/uint256"
)

// TestProviderExtensionServing checks, end to end, that a transaction acquired
// as a full fetch (only the data cells delivered) ends up fully servable and
// fully advertised: the buffer completes it, the pool stores it, and
//
//   - GetCustody reports all-ones -- the mask the tx announcement carries, so
//     peers may request any column, and
//   - GetBlobCells serves extension columns (>= DataPerBlob) byte-correctly --
//     the exact call behind both the eth GetCells handler (p2p serving) and the
//     engine_getBlobsV4 cache miss path (CL serving).
//
// This test intentionally uses only APIs that predate the provider-extension
// change, so it can run against older code to demonstrate the gap: there, the
// stored/advertised custody is just the data cells and extension columns are
// returned as nil.
func TestProviderExtensionServing(t *testing.T) {
	storage := t.TempDir()
	os.MkdirAll(filepath.Join(storage, pendingTransactionStore), 0700)
	store, _ := billy.Open(billy.Options{Path: filepath.Join(storage, pendingTransactionStore)}, newSlotterEIP7594(params.BlobTxMaxBlobs), nil)
	store.Close()

	var (
		key, _    = crypto.GenerateKey()
		addr      = crypto.PubkeyToAddress(key.PublicKey)
		blobCount = 2
		// Post-eth/72 shape: blob payload elided, commitments and cell proofs kept.
		tx   = removeBlobs(makeMultiBlobTx(0, 10, 2*params.InitialBaseFee, 100, blobCount, 0, key))
		hash = tx.Hash()
	)
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.AddBalance(addr, uint256.NewInt(1_000_000_000_000_000_000), tracing.BalanceChangeUnspecified)
	statedb.Commit(params.Rules{IsEIP158: true}, 0)

	chain := &testBlockChain{
		config:  params.MainnetChainConfig,
		basefee: uint256.NewInt(params.InitialBaseFee),
		blobfee: uint256.NewInt(params.BlobTxMinBlobGasprice),
		statedb: statedb,
	}
	pool := New(Config{Datadir: storage}, chain, nil)
	if err := pool.Init(1, chain.CurrentBlock(), newReserver()); err != nil {
		t.Fatalf("failed to create blob pool: %v", err)
	}
	defer pool.Close()

	// Wire the buffer to the real pool, as the eth handler does, and run the
	// full-fetch ingest flow: tx body first, then a data-cells-only delivery.
	buf := NewBlobBuffer(BlobBufferFunctions{
		ValidateTx: pool.ValidateTxBasics,
		AddToPool:  pool.AddPooledTx,
		DropPeer:   func(peer string) {},
	})
	if err := buf.AddTx([]*types.Transaction{tx}, "peerA")[0]; err != nil {
		t.Fatalf("AddTx: %v", err)
	}
	dataIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range dataIndices {
		dataIndices[i] = uint64(i)
	}
	buf.AddCells(hash, map[string]*PeerDelivery{"peerB": makePeerDelivery(t, 0, blobCount, dataIndices)}, types.NewCustodyBitmap(dataIndices))

	hashes, errs := buf.Flush()
	if len(hashes) != 1 || errs[0] != nil {
		t.Fatalf("expected 1 pooled tx, got %d (err %v)", len(hashes), errs)
	}

	// Advertising: the announcement mask is built from GetCustody, so all-ones
	// here means peers are invited to request any column.
	if custody := pool.GetCustody(hash); custody == nil || *custody != types.CustodyBitmapAll {
		have := -1
		if custody != nil {
			have = custody.OneCount()
		}
		t.Errorf("advertised custody not all-ones: have %d cells, want %d", have, kzg4844.CellsPerBlob)
	}

	// Serving: request extension columns the node never downloaded.
	extIndices := make([]uint64, 32)
	for i := range extIndices {
		extIndices[i] = uint64(kzg4844.DataPerBlob + i)
	}
	vhashes := pool.GetBlobHashes(hash)
	if len(vhashes) != blobCount {
		t.Fatalf("expected %d versioned hashes, got %d", blobCount, len(vhashes))
	}
	cells, proofs, err := pool.GetBlobCells(vhashes, types.NewCustodyBitmap(extIndices))
	if err != nil {
		t.Fatalf("GetBlobCells: %v", err)
	}
	for b := 0; b < blobCount; b++ {
		truth, err := kzg4844.ComputeCells([]kzg4844.Blob{*testBlobs[b]})
		if err != nil {
			t.Fatal(err)
		}
		for k, idx := range extIndices {
			if cells[b] == nil || cells[b][k] == nil {
				t.Errorf("blob %d: extension cell %d not served", b, idx)
				continue
			}
			if *cells[b][k] != truth[idx] {
				t.Errorf("blob %d: extension cell %d does not match ground truth", b, idx)
			}
			if proofs[b] == nil || proofs[b][k] == nil {
				t.Errorf("blob %d: extension proof %d not served", b, idx)
			}
		}
	}
}
