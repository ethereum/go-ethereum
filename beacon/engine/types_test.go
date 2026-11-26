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

package engine

import (
	"math/big"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

func TestBlobs(t *testing.T) {
	var (
		emptyBlob          = new(kzg4844.Blob)
		emptyBlobCommit, _ = kzg4844.BlobToCommitment(emptyBlob)
		emptyBlobProof, _  = kzg4844.ComputeBlobProof(emptyBlob, emptyBlobCommit)
		emptyCellProof, _  = kzg4844.ComputeCellProofs(emptyBlob)
	)
	header := types.Header{}
	block := types.NewBlock(&header, &types.Body{}, nil, nil)

	sidecarWithoutCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, []kzg4844.Proof{emptyBlobProof})
	env := BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithoutCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 1 {
		t.Fatalf("Expect 1 proof in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}

	sidecarWithCellProofs := types.NewBlobTxSidecar(types.BlobSidecarVersion0, []kzg4844.Blob{*emptyBlob}, []kzg4844.Commitment{emptyBlobCommit}, emptyCellProof)
	env = BlockToExecutableData(block, common.Big0, []*types.BlobTxSidecar{sidecarWithCellProofs}, nil)
	if len(env.BlobsBundle.Proofs) != 128 {
		t.Fatalf("Expect 128 proofs in blobs bundle, got %v", len(env.BlobsBundle.Proofs))
	}
}

// makeEncTx builds a dynamic-fee transaction with dataLen bytes of calldata and
// returns its canonical binary encoding. The transaction is unsigned, which is
// irrelevant to encode/decode round-tripping.
func makeEncTx(t *testing.T, nonce uint64, dataLen int) []byte {
	t.Helper()
	to := common.Address{0x1}
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(0),
		Data:      make([]byte, dataLen),
	})
	enc, err := tx.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return enc
}

// TestDecodeWorkers checks that the worker pool is sized by both transaction
// count and total encoded bytes, so a block of a few large transactions is
// decoded concurrently instead of serially.
func TestDecodeWorkers(t *testing.T) {
	const maxProcs = 8
	blob := func(n, size int) [][]byte {
		enc := make([][]byte, n)
		for i := range enc {
			enc[i] = make([]byte, size)
		}
		return enc
	}
	tests := []struct {
		name string
		enc  [][]byte
		max  int
		want int
	}{
		{"empty", nil, maxProcs, 0},
		{"few small -> serial", blob(4, 100), maxProcs, 0},
		{"many small -> count parallel", blob(96, 100), maxProcs, 3},
		// 4 x 300 KiB: count/32 == 0, but 1.2 MiB / 256 KiB == 4 -> parallel.
		{"few large -> byte parallel", blob(4, 300*1024), maxProcs, 4},
		{"byte parallel capped by maxProcs", blob(4, 300*1024), 2, 2},
		// 2 x 1 MiB: bytes term is 8, but capped at the 2 transactions.
		{"few very large capped by tx count", blob(2, 1024*1024), maxProcs, 2},
		// single large tx cannot be parallelized -> 1 worker (serial).
		{"single large -> serial", blob(1, 1024*1024), maxProcs, 1},
	}
	for _, tt := range tests {
		if got := decodeWorkers(tt.enc, tt.max); got != tt.want {
			t.Errorf("%s: decodeWorkers = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// TestDecodeTransactionsRoundtrip checks that DecodeTransactions returns the
// original transactions for both the serial and the parallel paths, and that a
// malformed entry yields an error naming the lowest failing index.
func TestDecodeTransactionsRoundtrip(t *testing.T) {
	sizes := map[string]struct {
		n, dataLen int
	}{
		"serial small":        {3, 64},
		"count parallel":      {96, 64},
		"byte parallel (few)": {6, 300 * 1024}, // 1.8 MiB across 6 txs -> parallel by bytes
	}
	for name, s := range sizes {
		enc := make([][]byte, s.n)
		want := make([]common.Hash, s.n)
		for i := range enc {
			enc[i] = makeEncTx(t, uint64(i), s.dataLen)
			var tx types.Transaction
			if err := tx.UnmarshalBinary(enc[i]); err != nil {
				t.Fatalf("%s: setup decode: %v", name, err)
			}
			want[i] = tx.Hash()
		}
		got, err := DecodeTransactions(enc)
		if err != nil {
			t.Fatalf("%s: DecodeTransactions: %v", name, err)
		}
		if len(got) != s.n {
			t.Fatalf("%s: got %d txs, want %d", name, len(got), s.n)
		}
		for i, tx := range got {
			if tx.Hash() != want[i] {
				t.Errorf("%s: tx %d hash mismatch", name, i)
			}
		}
	}

	// A malformed entry must surface an error naming its (lowest) index.
	enc := [][]byte{
		makeEncTx(t, 0, 32),
		makeEncTx(t, 1, 32),
		{0x02, 0xff}, // typed (dynamic-fee) prefix with a garbage payload
		makeEncTx(t, 3, 32),
	}
	_, err := DecodeTransactions(enc)
	if err == nil {
		t.Fatal("expected error for malformed transaction, got nil")
	}
	if !strings.Contains(err.Error(), "transaction 2") {
		t.Errorf("error %q does not name failing index 2", err)
	}
}

// TestDecodeTransactionsSkewed decodes a block whose transaction sizes are highly
// uneven (a few large transactions among many small ones). The shared-cursor
// distribution must still return every transaction correctly regardless of where
// the large ones sit.
func TestDecodeTransactionsSkewed(t *testing.T) {
	const n = 300
	enc := make([][]byte, n)
	want := make([]common.Hash, n)
	for i := range enc {
		dataLen := 64
		if i%40 == 0 {
			dataLen = 400 * 1024 // a few large txs; enough total bytes to force parallel decode
		}
		enc[i] = makeEncTx(t, uint64(i), dataLen)
		var tx types.Transaction
		if err := tx.UnmarshalBinary(enc[i]); err != nil {
			t.Fatalf("setup decode %d: %v", i, err)
		}
		want[i] = tx.Hash()
	}
	if w := decodeWorkers(enc, runtime.GOMAXPROCS(0)); w < 2 {
		t.Fatalf("test needs the parallel path, got %d workers", w)
	}
	got, err := DecodeTransactions(enc)
	if err != nil {
		t.Fatalf("DecodeTransactions: %v", err)
	}
	if len(got) != n {
		t.Fatalf("got %d txs, want %d", len(got), n)
	}
	for i, tx := range got {
		if tx == nil {
			t.Fatalf("tx %d is nil", i)
		}
		if tx.Hash() != want[i] {
			t.Errorf("tx %d hash mismatch", i)
		}
	}
}

// TestDecodeTransactionsLowestError checks that on the parallel path a block with
// several malformed transactions reports the error for the lowest-indexed one,
// matching serial decode order, even though workers decode out of order.
func TestDecodeTransactionsLowestError(t *testing.T) {
	const n = 256
	enc := make([][]byte, n)
	for i := range enc {
		enc[i] = makeEncTx(t, uint64(i), 64)
	}
	// Corrupt several transactions; the lowest index must be the one reported.
	for _, i := range []int{37, 100, 211} {
		enc[i] = []byte{0x02, 0xff} // typed (dynamic-fee) prefix with a garbage payload
	}
	if w := decodeWorkers(enc, runtime.GOMAXPROCS(0)); w < 2 {
		t.Fatalf("test needs the parallel path, got %d workers", w)
	}
	// Run repeatedly: the parallel path is nondeterministic, so a single pass can
	// miss a scheduling that would report the wrong index.
	for range 50 {
		_, err := DecodeTransactions(enc)
		if err == nil {
			t.Fatal("expected error for malformed transactions, got nil")
		}
		if !strings.Contains(err.Error(), "transaction 37") {
			t.Fatalf("error %q does not name lowest failing index 37", err)
		}
	}
}
