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

package ethtest

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// TestWrapperFixtures checks the committed wrapper fixtures against the wrapper encoding
// and the metadata the suite derives from them.
func TestWrapperFixtures(t *testing.T) {
	f, err := loadFixtures("testdata")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]struct {
		fixtures []*wrapperFixture
		mode     uint64
	}{
		"direct":        {f.Direct, wrapperModeDirect},
		"inline":        {f.Inline, wrapperModeDirect},
		"unknownScheme": {f.UnknownScheme, wrapperModeDirect},
		"recursive":     {f.Recursive, wrapperModeRecursive},
	}
	for name, kind := range kinds {
		if len(kind.fixtures) == 0 {
			t.Errorf("%s: no fixtures", name)
		}
		for i, w := range kind.fixtures {
			b, err := w.body()
			if err != nil {
				t.Fatalf("%s %d: %v", name, i, err)
			}
			if b.Mode != kind.mode {
				t.Errorf("%s %d: mode %d, want %d", name, i, b.Mode, kind.mode)
			}
			if len(w.Wrapper) > maxWrapperBytes {
				t.Errorf("%s %d: %d bytes exceed MAX_WRAPPER_BYTES", name, i, len(w.Wrapper))
			}
			if len(b.Transactions) != len(w.Transactions) {
				t.Fatalf("%s %d: %d transactions, metadata for %d", name, i, len(b.Transactions), len(w.Transactions))
			}
			for j, tx := range w.Transactions {
				if uint64(len(b.Transactions[j])) != tx.Size {
					t.Errorf("%s %d: transaction %d has %d bytes, metadata says %d", name, i, j, len(b.Transactions[j]), tx.Size)
				}
				if j > 0 && w.Transactions[j-1].Hash.Cmp(tx.Hash) >= 0 {
					t.Errorf("%s %d: transactions are not in ascending hash order", name, i)
				}
			}
			deps := w.deps()
			for j := range deps {
				if len(deps[j]) != 96 || j > 0 && bytes.Compare(deps[j-1], deps[j]) >= 0 {
					t.Errorf("%s %d: dependencies are not canonical", name, i)
				}
			}
			ann := w.announcement()
			if ann.Hash != crypto.Keccak256Hash(w.Wrapper) || ann.Length != uint64(len(w.Wrapper)) {
				t.Errorf("%s %d: announcement does not describe the wrapper", name, i)
			}
			inline := kind.mode == wrapperModeDirect && len(w.Transactions) <= maxInlineTransactions
			if inline {
				if err := checkMeta(ann.Meta, ann.Length); err != nil {
					t.Errorf("%s %d: %v", name, i, err)
				}
			} else if !bytes.Equal(ann.Meta, emptyList) {
				t.Errorf("%s %d: announcement carries metadata it must leave empty", name, i)
			}
		}
	}
	// A corrupted witness changes the wrapper but keeps its encoding canonical.
	bad := f.Direct[0].withCorruptWitness()
	if bad.hash() == f.Direct[0].hash() || len(bad.Wrapper) != len(f.Direct[0].Wrapper) {
		t.Error("witness corruption changed the wrapper length or not its hash")
	}
	if _, err := bad.body(); err != nil {
		t.Error(err)
	}
}
