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
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era/execdb"
	"github.com/ethereum/go-ethereum/internal/era/onedb"
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

// TestEreDatabase checks that the store can serve bodies and receipts from a
// directory of ere files, and that the receipts returned are byte-identical to
// the ones derived from the equivalent era1 files.
func TestEreDatabase(t *testing.T) {
	dir := t.TempDir()
	convertEra1ToEre(t, "testdata/sepolia-00000-643a00f7.era1", dir, "sepolia", 0)
	convertEra1ToEre(t, "testdata/sepolia-00021-b8814b14.era1", dir, "sepolia", 21)

	db, err := New(dir)
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

	// Cross-check against the era1 store: both backends must return the same
	// storage encoding.
	eraDB, err := New("testdata")
	require.NoError(t, err)
	defer eraDB.Close()
	for _, num := range []uint64{0, 1024, 172032, 175881, 180223} {
		want, err := eraDB.GetRawReceipts(num)
		require.NoError(t, err)
		got, err := db.GetRawReceipts(num)
		require.NoError(t, err)
		assert.Equal(t, want, got, "receipts mismatch at block %d", num)

		wantBody, err := eraDB.GetRawBody(num)
		require.NoError(t, err)
		gotBody, err := db.GetRawBody(num)
		require.NoError(t, err)
		assert.Equal(t, wantBody, gotBody, "body mismatch at block %d", num)
	}
}

func TestEraStoreRejectsNoReceiptsProfile(t *testing.T) {
	dir := t.TempDir()
	stubName := "mainnet-00000-deadbeef-noreceipts.ere"
	stubPath := filepath.Join(dir, stubName)

	// Write a non-empty stub so the glob finds the file. Contents don't matter
	// because the noreceipts check fires before execdb.Open is called.
	err := os.WriteFile(stubPath, []byte("stub"), 0644)
	require.NoError(t, err)

	db, err := New(dir)
	require.NoError(t, err)
	defer db.Close()

	// Any block in epoch 0 should trigger the same rejection.
	const block = uint64(0)

	_, err = db.GetRawBody(block)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "era store does not support noreceipts profile")

	_, err = db.GetRawReceipts(block)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "era store does not support noreceipts profile")
}

// convertEra1ToEre reads an era1 file and writes its contents as an ere file
// into dir, using the canonical ere file name.
func convertEra1ToEre(t *testing.T, era1Path, dir, network string, epoch int) {
	t.Helper()

	e, err := onedb.Open(era1Path)
	require.NoError(t, err)
	defer e.Close()

	f, err := os.CreateTemp(dir, "ere-convert-*")
	require.NoError(t, err)
	defer f.Close()

	builder := execdb.NewBuilder(f)
	td, err := e.InitialTD()
	require.NoError(t, err)

	it, err := e.Iterator()
	require.NoError(t, err)
	for it.Next() {
		block, receipts, err := it.BlockAndReceipts()
		require.NoError(t, err)
		td.Add(td, block.Difficulty())
		require.NoError(t, builder.Add(block, receipts, new(big.Int).Set(td)))
	}
	require.NoError(t, it.Error())

	lastHash, err := builder.Finalize()
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Rename(f.Name(), filepath.Join(dir, execdb.Filename(network, epoch, lastHash))))
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
