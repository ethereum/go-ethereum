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

package fetcher

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

// benchmarkNotify measures the allocation cost of TxFetcher.Notify when each
// peer's announcement batch contains `unknown` fresh hashes followed by
// `known` duplicates (simulated by validateMeta returning ErrAlreadyKnown).
//
// The steady-state case on a warm node is unknown == 0: every hash has already
// been seen from some other peer. The pre-allocation of
// make([]common.Hash, 0, len(hashes)) + make([]txMetadata, 0, len(hashes))
// used to force ~2 * 32 * len(hashes) bytes of waste per call in that case.
func benchmarkNotify(b *testing.B, unknown, known int) {
	b.Helper()

	total := unknown + known
	hashes := make([]common.Hash, total)
	for i := range hashes {
		// Bit-pattern hashes so the first `unknown` look fresh and the rest
		// trigger the "already-known" fast path.
		hashes[i][0] = byte(i & 0xff)
		hashes[i][1] = byte(i >> 8)
		if i >= unknown {
			// Distinguish "known" hashes by a unique byte so we can keep a
			// tiny set that validateMeta treats as already in the pool.
			hashes[i][31] = 1
		}
	}
	types := make([]byte, total)
	for i := range types {
		types[i] = 0x03 // BlobTx type, valid per validateMeta
	}
	sizes := make([]uint32, total)
	for i := range sizes {
		sizes[i] = 128
	}

	// validateMeta discriminates by the marker byte we embedded in each hash:
	// trailing-byte == 1 means "already in the local pool".
	validate := func(h common.Hash, _ byte) error {
		if h[31] == 1 {
			return txpool.ErrAlreadyKnown
		}
		return nil
	}

	fetcher := NewTxFetcher(
		nil,
		validate,
		func(txs []*gethtypes.Transaction) []error { return make([]error, len(txs)) },
		func(string, []common.Hash) error { return nil },
		nil,
	)
	// Don't start the fetcher loop; Notify's fast path only hits if the
	// internal select fires, but when there are zero unknowns we return early
	// before touching the channel. For unknown > 0 we drop the announcement
	// by draining the notify channel in a goroutine.
	if unknown > 0 {
		go func() {
			for range fetcher.notify {
			}
		}()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a distinct peer id each call so Notify can't short-circuit
		// on duplicate peer state.
		if err := fetcher.Notify("peer", types, sizes, hashes); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNotify_AllKnown is the hot steady-state case: every announced hash
// is already in the local pool. Pre-fix this paid 2 * 32 * len(hashes) bytes
// per call for slices that never received an append.
func BenchmarkNotify_AllKnown(b *testing.B) {
	benchmarkNotify(b, 0, 256)
}

// BenchmarkNotify_HalfNew is a mixed case with 50% fresh hashes.
func BenchmarkNotify_HalfNew(b *testing.B) {
	benchmarkNotify(b, 128, 128)
}

// BenchmarkNotify_AllNew is the worst case for the lazy allocation: every
// hash is fresh so the slice must be allocated anyway. This guards against
// regressing the common path.
func BenchmarkNotify_AllNew(b *testing.B) {
	benchmarkNotify(b, 256, 0)
}
