// Copyright 2024 The go-ethereum Authors
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
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// benchHash deterministically derives a hash from the given group/index pair.
func benchHash(group, idx uint64) common.Hash {
	var h common.Hash
	binary.BigEndian.PutUint64(h[0:8], group)
	binary.BigEndian.PutUint64(h[24:32], idx)
	return h
}

// makeBenchDiffLayers builds a stack of diff layers, each mutating perLayer
// accounts and perLayer single-slot storages. The first `hot` keys are shared
// across every layer (modelling frequently-touched accounts), while the rest
// are unique to a layer (the long tail touched by exactly one layer).
func makeBenchDiffLayers(layers, perLayer, hot int) []*diffLayer {
	diffs := make([]*diffLayer, layers)
	for i := 0; i < layers; i++ {
		accounts := make(map[common.Hash][]byte, perLayer)
		storages := make(map[common.Hash]map[common.Hash][]byte, perLayer)
		for j := 0; j < perLayer; j++ {
			var key common.Hash
			if j < hot {
				key = benchHash(0, uint64(j)) // shared across all layers
			} else {
				key = benchHash(uint64(i+1), uint64(j)) // unique per layer
			}
			accounts[key] = []byte{0x01}
			storages[key] = map[common.Hash][]byte{{0x01}: {0x02}}
		}
		diffs[i] = &diffLayer{
			root:   benchHash(0xffff, uint64(i+1)),
			states: NewStateSetWithOrigin(accounts, storages, nil, nil, false),
		}
	}
	return diffs
}

func newBenchLookup() *lookup {
	return &lookup{
		accounts:   make(map[common.Hash]layerList),
		storages:   make(map[[64]byte]layerList),
		descendant: func(common.Hash, common.Hash) bool { return false },
	}
}

// BenchmarkLookupAddLayer measures the cost of building the lookup index for a
// full stack of in-memory diff layers (one addLayer call per layer).
func BenchmarkLookupAddLayer(b *testing.B) {
	const (
		layers   = 128
		perLayer = 2000
		hot      = 200
	)
	diffs := makeBenchDiffLayers(layers, perLayer, hot)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := newBenchLookup()
		for _, d := range diffs {
			l.addLayer(d)
		}
	}
}

// BenchmarkLookupRemoveLayer measures the cost of unlinking a full stack of
// diff layers from the lookup index (one removeLayer call per layer, oldest
// first, as happens during flattening).
func BenchmarkLookupRemoveLayer(b *testing.B) {
	const (
		layers   = 128
		perLayer = 2000
		hot      = 200
	)
	diffs := makeBenchDiffLayers(layers, perLayer, hot)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		l := newBenchLookup()
		for _, d := range diffs {
			l.addLayer(d)
		}
		b.StartTimer()

		for _, d := range diffs {
			if err := l.removeLayer(d); err != nil {
				b.Fatal(err)
			}
		}
	}
}
