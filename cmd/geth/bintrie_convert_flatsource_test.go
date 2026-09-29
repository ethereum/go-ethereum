// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// convertSourceScheme converts artifactAlloc() from a freshly committed
// genesis held under the given scheme and returns the binary root and the
// snapshot and preimage artifact bytes.
func convertSourceScheme(t *testing.T, pathScheme bool) (common.Hash, []byte, []byte) {
	t.Helper()
	opts := artifactOptions(t)
	_, _, binRoot, err := convertGenesis(t, artifactAlloc(), pathScheme, opts)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	snapPath, prePath := opts.snapshotPath, opts.preimagePath
	snap, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	pre, err := os.ReadFile(prePath)
	if err != nil {
		t.Fatal(err)
	}
	return binRoot, snap, pre
}

// TestConvertFlatSourceParity converts one state (storage in and past the
// header range, code, a delegation, a shared code hash) through pathdb's
// flat state and through a hash-scheme source's trie node walk. The root and
// both artifacts must be byte-identical: the source changes, not the leaves.
func TestConvertFlatSourceParity(t *testing.T) {
	flatRoot, flatSnap, flatPre := convertSourceScheme(t, true)
	walkRoot, walkSnap, walkPre := convertSourceScheme(t, false)
	if flatRoot != walkRoot {
		t.Fatalf("flat root %x != node-walk root %x", flatRoot, walkRoot)
	}
	if !bytes.Equal(flatSnap, walkSnap) {
		t.Fatal("flat and node-walk snapshot artifacts differ")
	}
	if !bytes.Equal(flatPre, walkPre) {
		t.Fatal("flat and node-walk preimage artifacts differ")
	}
}
