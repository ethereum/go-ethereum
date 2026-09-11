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

package txpool

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

type filterTypeSubpool struct {
	SubPool
	supported map[byte]bool
}

func (p *filterTypeSubpool) FilterType(kind byte) bool {
	return p.supported[kind]
}

func TestTxPoolFilterType(t *testing.T) {
	tests := []struct {
		name     string
		subpools []SubPool
		kind     byte
		want     bool
	}{
		{
			name: "no subpool supports type",
			subpools: []SubPool{
				&filterTypeSubpool{supported: map[byte]bool{types.LegacyTxType: true}},
				&filterTypeSubpool{supported: map[byte]bool{types.BlobTxType: true}},
			},
			kind: types.DynamicFeeTxType,
			want: false,
		},
		{
			name: "first subpool supports type",
			subpools: []SubPool{
				&filterTypeSubpool{supported: map[byte]bool{types.DynamicFeeTxType: true}},
				&filterTypeSubpool{supported: map[byte]bool{types.BlobTxType: true}},
			},
			kind: types.DynamicFeeTxType,
			want: true,
		},
		{
			name: "second subpool supports type",
			subpools: []SubPool{
				&filterTypeSubpool{supported: map[byte]bool{types.DynamicFeeTxType: true}},
				&filterTypeSubpool{supported: map[byte]bool{types.BlobTxType: true}},
			},
			kind: types.BlobTxType,
			want: true,
		},
		{
			name: "unknown type",
			subpools: []SubPool{
				&filterTypeSubpool{supported: map[byte]bool{types.DynamicFeeTxType: true}},
				&filterTypeSubpool{supported: map[byte]bool{types.BlobTxType: true}},
			},
			kind: 0xff,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := &TxPool{subpools: tt.subpools}
			if got := pool.FilterType(tt.kind); got != tt.want {
				t.Fatalf("FilterType(%#x) = %t, want %t", tt.kind, got, tt.want)
			}
		})
	}
}
