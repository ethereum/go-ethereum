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

package eradb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEraDatabase(t *testing.T) {
	// Create the database
	db, err := New("testdata")
	require.NoError(t, err)
	defer db.Close()

	block, err := db.GetBlockByNumber(15000)
	require.NoError(t, err)
	require.NotNil(t, block, "block not found")
	require.Equal(t, uint64(15000), block.NumberU64())

	// Get Header
	header, err := db.GetHeaderByNumber(15000)
	require.NoError(t, err)
	require.NotNil(t, header, "header not found")
	require.Equal(t, uint64(15000), header.Number.Uint64())

	// Get Receipts
	receipts, err := db.GetReceiptsByNumber(15000)
	require.NoError(t, err)
	require.NotNil(t, receipts, "receipts not found")
	require.Equal(t, 0, len(receipts), "receipts length mismatch")
}
