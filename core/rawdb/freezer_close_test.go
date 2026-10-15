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

package rawdb

import (
	"errors"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/metrics"
)

// TestFreezerTableCloseAfterSyncFailure checks that Close releases all file
// handles even if the final sync fails.
func TestFreezerTableCloseAfterSyncFailure(t *testing.T) {
	t.Parallel()
	f, err := newTable(t.TempDir(), "closefail", metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge(), 50, freezerTableConfig{noSnappy: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	// Force a file rotation so more than one data file is open.
	batch := f.newBatch()
	for i := uint64(0); i < 4; i++ {
		if err := batch.AppendRaw(i, getChunk(20, int(i))); err != nil {
			t.Fatal(err)
		}
	}
	if err := batch.commit(); err != nil {
		t.Fatal(err)
	}
	var (
		index = f.index
		meta  = f.metadata.file
		files []*os.File
	)
	for _, file := range f.files {
		files = append(files, file)
	}
	// Make the final sync fail by swapping in an already-closed head handle.
	broken, err := os.CreateTemp(t.TempDir(), "broken")
	if err != nil {
		t.Fatal(err)
	}
	broken.Close()
	f.head = broken

	if err := f.Close(); err == nil {
		t.Fatal("expected close error")
	}
	for _, file := range append([]*os.File{index, meta}, files...) {
		if _, err := file.Stat(); !errors.Is(err, os.ErrClosed) {
			t.Errorf("file %s still open after Close", file.Name())
		}
	}
}
