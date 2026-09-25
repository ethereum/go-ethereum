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

package pathdb

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// buildAccountLayers constructs a stack of `total` diff layers on top of an
// empty disk layer. Each layer holds `perLayer` random slim-encoded accounts.
// A single "hot" account is injected into the layer at `depth` (0 == bottom-most
// diff layer, total-1 == top-most) and its hash is returned so a benchmark can
// repeatedly resolve it. This mirrors the production layout where a recently
// touched account lives in one of the in-memory diff layers and must be located
// by walking down from the top layer.
func buildAccountLayers(total, perLayer, depth int) (layer, common.Hash) {
	hotHash := common.BytesToHash(testrand.Bytes(32))
	hotBlob := types.SlimAccountRLP(types.StateAccount{
		Nonce:    1,
		Balance:  uint256.NewInt(100),
		Root:     types.EmptyRootHash,
		CodeHash: types.EmptyCodeHash[:],
	})

	fill := func(parent layer, index int) *diffLayer {
		accounts := make(map[common.Hash][]byte, perLayer)
		for i := 0; i < perLayer; i++ {
			accounts[common.BytesToHash(testrand.Bytes(32))] = types.SlimAccountRLP(types.StateAccount{
				Nonce:    uint64(i),
				Balance:  uint256.NewInt(uint64(i)),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash[:],
			})
		}
		if index == depth {
			accounts[hotHash] = hotBlob
		}
		states := NewStateSetWithOrigin(accounts, nil, nil, nil, false)
		return newDiffLayer(parent, common.Hash{}, 0, 0, NewNodeSetWithOrigin(nil, nil), states)
	}

	var l layer = emptyLayer()
	for i := 0; i < total; i++ {
		l = fill(l, i)
	}
	return l, hotHash
}

// benchmarkAccountRead resolves the slim account blob by walking the layer stack
// and then RLP-decodes it into a *types.SlimAccount, exactly as reader.Account
// does. The decode is the cost the diff-layer refactor (storing decoded accounts
// instead of slim RLP) would move out of the read path.
func benchmarkAccountRead(b *testing.B, total, depth int) {
	l, hotHash := buildAccountLayers(total, 1000, depth)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blob, err := l.account(hotHash, 0)
		if err != nil {
			b.Fatal(err)
		}
		account := new(types.SlimAccount)
		if err := rlp.DecodeBytes(blob, account); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAccountReadTop reads an account living in the top-most diff layer:
// the lookup is cheap (depth 0) so the cost is dominated by the per-read RLP
// decode.
func BenchmarkAccountReadTop(b *testing.B) { benchmarkAccountRead(b, 128, 127) }

// BenchmarkAccountReadBottom reads an account living in the bottom-most diff
// layer of a 128-deep stack: this pays both the full traversal and the decode.
func BenchmarkAccountReadBottom(b *testing.B) { benchmarkAccountRead(b, 128, 0) }

// benchmarkAccountReadOnly isolates the layer traversal from the decode, to
// quantify how much of the read cost is the RLP decode (which the refactor
// removes) versus the lookup itself (which it does not).
func benchmarkAccountReadOnly(b *testing.B, total, depth int) {
	l, hotHash := buildAccountLayers(total, 1000, depth)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := l.account(hotHash, 0); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAccountLookupTop measures only the traversal+lookup for a top-layer
// account, with no decode. Compare against BenchmarkAccountReadTop to attribute
// cost to the decode.
func BenchmarkAccountLookupTop(b *testing.B) { benchmarkAccountReadOnly(b, 128, 127) }

// BenchmarkAccountLookupBottom measures only the traversal+lookup for a
// bottom-layer account. Compare against BenchmarkAccountReadBottom.
func BenchmarkAccountLookupBottom(b *testing.B) { benchmarkAccountReadOnly(b, 128, 0) }

// benchmarkAccountObject reads the decoded account directly via accountObject,
// which is the post-refactor hot path: the diff layers retain the account in
// decoded form, so no per-read RLP decode is performed. Compare against
// benchmarkAccountRead to measure the decode that was eliminated.
func benchmarkAccountObject(b *testing.B, total, depth int) {
	l, hotHash := buildAccountLayers(total, 1000, depth)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := l.accountObject(hotHash, 0); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAccountObjectTop reads a top-layer account in decoded form (no
// decode). Compare against BenchmarkAccountReadTop.
func BenchmarkAccountObjectTop(b *testing.B) { benchmarkAccountObject(b, 128, 127) }

// BenchmarkAccountObjectBottom reads a bottom-layer account in decoded form.
// Compare against BenchmarkAccountReadBottom.
func BenchmarkAccountObjectBottom(b *testing.B) { benchmarkAccountObject(b, 128, 0) }
