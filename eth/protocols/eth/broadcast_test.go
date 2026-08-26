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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

func TestCanServeTransaction(t *testing.T) {
	sparse := types.NewCustodyBitmap([]uint64{0, 1, 2, 3, 4, 5, 6, 7})
	recoverableIndices := make([]uint64, kzg4844.DataPerBlob)
	for i := range recoverableIndices {
		recoverableIndices[i] = uint64(i)
	}
	recoverable := types.NewCustodyBitmap(recoverableIndices)

	tests := []struct {
		name    string
		version uint
		custody *types.CustodyBitmap
		want    bool
	}{
		{"legacy transaction to eth/71", ETH71, nil, true},
		{"sparse blob transaction to eth/71", ETH71, &sparse, false},
		{"recoverable blob transaction to eth/71", ETH71, &recoverable, true},
		{"sparse blob transaction to eth/72", ETH72, &sparse, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := canServeTransaction(test.version, test.custody); got != test.want {
				t.Fatalf("canServeTransaction(%d, %v) = %v, want %v", test.version, test.custody, got, test.want)
			}
		})
	}
}
