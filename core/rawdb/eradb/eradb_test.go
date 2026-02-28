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
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEraDatabase(t *testing.T) {
	db, err := New("testdata")
	require.NoError(t, err)
	defer db.Close()

	r, err := db.GetRawBody(175881)
	require.NoError(t, err)
	var body *types.Body
	err = rlp.DecodeBytes(r, &body)
	require.NoError(t, err)
	require.NotNil(t, body, "block body not found")
	assert.Equal(t, 3, len(body.Transactions))

	r, err = db.GetRawReceipts(175881)
	require.NoError(t, err)
	var receipts []*types.ReceiptForStorage
	err = rlp.DecodeBytes(r, &receipts)
	require.NoError(t, err)
	require.NotNil(t, receipts, "receipts not found")
	assert.Equal(t, 3, len(receipts), "receipts length mismatch")
}

func TestEraDatabaseConcurrentOpen(t *testing.T) {
	db, err := New("testdata")
	require.NoError(t, err)
	defer db.Close()

	const N = 25
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			r, err := db.GetRawBody(1024)
			if err != nil {
				t.Error("err:", err)
			}
			if len(r) == 0 {
				t.Error("empty body")
			}
		}()
	}
	wg.Wait()
}

func TestEraDatabaseConcurrentOpenClose(t *testing.T) {
	db, err := New("testdata")
	require.NoError(t, err)
	defer db.Close()

	const N = 10
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			r, err := db.GetRawBody(1024)
			if err == errClosed {
				return
			}
			if err != nil {
				t.Error("err:", err)
			}
			if len(r) == 0 {
				t.Error("empty body")
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Close()
	}()
	wg.Wait()
}
