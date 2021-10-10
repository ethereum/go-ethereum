// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// TestFreezerBasics test initializing a freezertable from scratch, writing to the table,
// and reading it back.
func TestFreezerBasics(t *testing.T) {
	t.Parallel()
	// set cutoff at 50 bytes
	f, err := newTable(os.TempDir(),
		fmt.Sprintf("unittest-%d", rand.Uint64()),
		metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge(), 50, true)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Write 15 bytes 255 times, results in 85 files
	writeChunks(t, f, 255, 15)

	//print(t, f, 0)
	//print(t, f, 1)
	//print(t, f, 2)
	//
	//db[0] =  000000000000000000000000000000
	//db[1] =  010101010101010101010101010101
	//db[2] =  020202020202020202020202020202

	for y := 0; y < 255; y++ {
		exp := getChunk(15, y)
		got, err := f.Retrieve(uint64(y))
		if err != nil {
			t.Fatalf("reading item %d: %v", y, err)
		}
		if !bytes.Equal(got, exp) {
			t.Fatalf("test %d, got \n%x != \n%x", y, got, exp)
		}
	}
	// Check that we cannot read too far
	_, err = f.Retrieve(uint64(255))
	if err != errOutOfBounds {
		t.Fatal(err)
	}
}

// TestFreezerBasicsClosing tests same as TestFreezerBasics, but also closes and reopens the freezer between
// every operation
func TestFreezerBasicsClosing(t *testing.T) {
	t.Parallel()
	// set cutoff at 50 bytes
	var (
		fname      = fmt.Sprintf("basics-close-%d", rand.Uint64())
		rm, wm, sg = metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
		f          *freezerTable
		err        error
	)
	f, err = newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
	if err != nil {
		t.Fatal(err)
	}

	// Write 15 bytes 255 times, results in 85 files.
	// In-between writes, the table is closed and re-opened.
	for x := 0; x < 255; x++ {
		data := getChunk(15, x)
		batch := f.newBatch()
		require.NoError(t, batch.AppendRaw(uint64(x), data))
		require.NoError(t, batch.commit())
		f.Close()

		f, err = newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer f.Close()

	for y := 0; y < 255; y++ {
		exp := getChunk(15, y)
		got, err := f.Retrieve(uint64(y))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, exp) {
			t.Fatalf("test %d, got \n%x != \n%x", y, got, exp)
		}
		f.Close()
		f, err = newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestFreezerRepairDanglingHead tests that we can recover if index entries are removed
func TestFreezerRepairDanglingHead(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("dangling_headtest-%d", rand.Uint64())

	// Fill table
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		writeChunks(t, f, 255, 15)

		// The last item should be there
		if _, err = f.Retrieve(0xfe); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	// open the index
	idxFile, err := os.OpenFile(filepath.Join(os.TempDir(), fmt.Sprintf("%s.ridx", fname)), os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open index file: %v", err)
	}
	// Remove 4 bytes
	stat, err := idxFile.Stat()
	if err != nil {
		t.Fatalf("Failed to stat index file: %v", err)
	}
	idxFile.Truncate(stat.Size() - 4)
	idxFile.Close()

	// Now open it again
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// The last item should be missing
		if _, err = f.Retrieve(0xff); err == nil {
			t.Errorf("Expected error for missing index entry")
		}
		// The one before should still be there
		if _, err = f.Retrieve(0xfd); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}
}

// TestFreezerRepairDanglingHeadLarge tests that we can recover if very many index entries are removed
func TestFreezerRepairDanglingHeadLarge(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("dangling_headtest-%d", rand.Uint64())

	// Fill a table and close it
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		writeChunks(t, f, 255, 15)

		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	// open the index
	idxFile, err := os.OpenFile(filepath.Join(os.TempDir(), fmt.Sprintf("%s.ridx", fname)), os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("Failed to open index file: %v", err)
	}
	// Remove everything but the first item, and leave data unaligned
	// 0-indexEntry, 1-indexEntry, corrupt-indexEntry
	idxFile.Truncate(indexEntrySize + indexEntrySize + indexEntrySize/2)
	idxFile.Close()

	// Now open it again
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// The first item should be there
		if _, err = f.Retrieve(0); err != nil {
			t.Fatal(err)
		}
		// The second item should be missing
		if _, err = f.Retrieve(1); err == nil {
			t.Errorf("Expected error for missing index entry")
		}
		// We should now be able to store items again, from item = 1
		batch := f.newBatch()
		for x := 1; x < 0xff; x++ {
			require.NoError(t, batch.AppendRaw(uint64(x), getChunk(15, ^x)))
		}
		require.NoError(t, batch.commit())
		f.Close()
	}

	// And if we open it, we should now be able to read all of them (new values)
	{
		f, _ := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		for y := 1; y < 255; y++ {
			exp := getChunk(15, ^y)
			got, err := f.Retrieve(uint64(y))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, exp) {
				t.Fatalf("test %d, got \n%x != \n%x", y, got, exp)
			}
		}
	}
}

// TestSnappyDetection tests that we fail to open a snappy database and vice versa
func TestSnappyDetection(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("snappytest-%d", rand.Uint64())

	// Open with snappy
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		writeChunks(t, f, 255, 15)
		f.Close()
	}

	// Open without snappy
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.Retrieve(0); err == nil {
			f.Close()
			t.Fatalf("expected empty table")
		}
	}

	// Open with snappy
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// There should be 255 items
		if _, err = f.Retrieve(0xfe); err != nil {
			f.Close()
			t.Fatalf("expected no error, got %v", err)
		}
	}
}

func assertFileSize(f string, size int64) error {
	stat, err := os.Stat(f)
	if err != nil {
		return err
	}
	if stat.Size() != size {
		return fmt.Errorf("error, expected size %d, got %d", size, stat.Size())
	}
	return nil
}

// TestFreezerRepairDanglingIndex checks that if the index has more entries than there are data,
// the index is repaired
func TestFreezerRepairDanglingIndex(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("dangling_indextest-%d", rand.Uint64())

	// Fill a table and close it
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 9 times : 150 bytes
		writeChunks(t, f, 9, 15)

		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			f.Close()
			t.Fatal(err)
		}
		f.Close()
		// File sizes should be 45, 45, 45 : items[3, 3, 3)
	}

	// Crop third file
	fileToCrop := filepath.Join(os.TempDir(), fmt.Sprintf("%s.0002.rdat", fname))
	// Truncate third file: 45 ,45, 20
	{
		if err := assertFileSize(fileToCrop, 45); err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(fileToCrop, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		file.Truncate(20)
		file.Close()
	}

	// Open db it again
	// It should restore the file(s) to
	// 45, 45, 15
	// with 3+3+1 items
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if f.items != 7 {
			t.Fatalf("expected %d items, got %d", 7, f.items)
		}
		if err := assertFileSize(fileToCrop, 15); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFreezerTruncate(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("truncation-%d", rand.Uint64())

	// Fill table
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 30 times
		writeChunks(t, f, 30, 15)

		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	// Reopen, truncate
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		f.truncate(10) // 150 bytes
		if f.items != 10 {
			t.Fatalf("expected %d items, got %d", 10, f.items)
		}
		// 45, 45, 45, 15 -- bytes should be 15
		if f.headBytes != 15 {
			t.Fatalf("expected %d bytes, got %d", 15, f.headBytes)
		}
	}
}

// TestFreezerRepairFirstFile tests a head file with the very first item only half-written.
// That will rewind the index, and _should_ truncate the head file
func TestFreezerRepairFirstFile(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("truncationfirst-%d", rand.Uint64())

	// Fill table
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 80 bytes, splitting out into two files
		batch := f.newBatch()
		require.NoError(t, batch.AppendRaw(0, getChunk(40, 0xFF)))
		require.NoError(t, batch.AppendRaw(1, getChunk(40, 0xEE)))
		require.NoError(t, batch.commit())

		// The last item should be there
		if _, err = f.Retrieve(1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	// Truncate the file in half
	fileToCrop := filepath.Join(os.TempDir(), fmt.Sprintf("%s.0001.rdat", fname))
	{
		if err := assertFileSize(fileToCrop, 40); err != nil {
			t.Fatal(err)
		}
		file, err := os.OpenFile(fileToCrop, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		file.Truncate(20)
		file.Close()
	}

	// Reopen
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		if f.items != 1 {
			f.Close()
			t.Fatalf("expected %d items, got %d", 0, f.items)
		}

		// Write 40 bytes
		batch := f.newBatch()
		require.NoError(t, batch.AppendRaw(1, getChunk(40, 0xDD)))
		require.NoError(t, batch.commit())

		f.Close()

		// Should have been truncated down to zero and then 40 written
		if err := assertFileSize(fileToCrop, 40); err != nil {
			t.Fatal(err)
		}
	}
}

// TestFreezerReadAndTruncate tests:
// - we have a table open
// - do some reads, so files are open in readonly
// - truncate so those files are 'removed'
// - check that we did not keep the rdonly file descriptors
func TestFreezerReadAndTruncate(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("read_truncate-%d", rand.Uint64())

	// Fill table
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 30 times
		writeChunks(t, f, 30, 15)

		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}

	// Reopen and read all files
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		if f.items != 30 {
			f.Close()
			t.Fatalf("expected %d items, got %d", 0, f.items)
		}
		for y := byte(0); y < 30; y++ {
			f.Retrieve(uint64(y))
		}

		// Now, truncate back to zero
		f.truncate(0)

		// Write the data again
		batch := f.newBatch()
		for x := 0; x < 30; x++ {
			require.NoError(t, batch.AppendRaw(uint64(x), getChunk(15, ^x)))
		}
		require.NoError(t, batch.commit())
		f.Close()
	}
}

func TestFreezerOffset(t *testing.T) {
	t.Parallel()
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("offset-%d", rand.Uint64())

	// Fill table
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 40, true)
		if err != nil {
			t.Fatal(err)
		}

		// Write 6 x 20 bytes, splitting out into three files
		batch := f.newBatch()
		require.NoError(t, batch.AppendRaw(0, getChunk(20, 0xFF)))
		require.NoError(t, batch.AppendRaw(1, getChunk(20, 0xEE)))

		require.NoError(t, batch.AppendRaw(2, getChunk(20, 0xdd)))
		require.NoError(t, batch.AppendRaw(3, getChunk(20, 0xcc)))

		require.NoError(t, batch.AppendRaw(4, getChunk(20, 0xbb)))
		require.NoError(t, batch.AppendRaw(5, getChunk(20, 0xaa)))
		require.NoError(t, batch.commit())

		t.Log(f.dumpIndexString(0, 100))
		f.Close()
	}

	// Now crop it.
	{
		// delete files 0 and 1
		for i := 0; i < 2; i++ {
			p := filepath.Join(os.TempDir(), fmt.Sprintf("%v.%04d.rdat", fname, i))
			if err := os.Remove(p); err != nil {
				t.Fatal(err)
			}
		}
		// Read the index file
		p := filepath.Join(os.TempDir(), fmt.Sprintf("%v.ridx", fname))
		indexFile, err := os.OpenFile(p, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		indexBuf := make([]byte, 7*indexEntrySize)
		indexFile.Read(indexBuf)

		// Update the index file, so that we store
		// [ file = 2, offset = 4 ] at index zero

		tailId := uint32(2)     // First file is 2
		itemOffset := uint32(4) // We have removed four items
		zeroIndex := indexEntry{
			filenum: tailId,
			offset:  itemOffset,
		}
		buf := zeroIndex.append(nil)
		// Overwrite index zero
		copy(indexBuf, buf)
		// Remove the four next indices by overwriting
		copy(indexBuf[indexEntrySize:], indexBuf[indexEntrySize*5:])
		indexFile.WriteAt(indexBuf, 0)
		// Need to truncate the moved index items
		indexFile.Truncate(indexEntrySize * (1 + 2))
		indexFile.Close()
	}

	// Now open again
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 40, true)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		t.Log(f.dumpIndexString(0, 100))

		// It should allow writing item 6.
		batch := f.newBatch()
		require.NoError(t, batch.AppendRaw(6, getChunk(20, 0x99)))
		require.NoError(t, batch.commit())

		checkRetrieveError(t, f, map[uint64]error{
			0: errOutOfBounds,
			1: errOutOfBounds,
			2: errOutOfBounds,
			3: errOutOfBounds,
		})
		checkRetrieve(t, f, map[uint64][]byte{
			4: getChunk(20, 0xbb),
			5: getChunk(20, 0xaa),
			6: getChunk(20, 0x99),
		})
	}

	// Edit the index again, with a much larger initial offset of 1M.
	{
		// Read the index file
		p := filepath.Join(os.TempDir(), fmt.Sprintf("%v.ridx", fname))
		indexFile, err := os.OpenFile(p, os.O_RDWR, 0644)
		if err != nil {
			t.Fatal(err)
		}
		indexBuf := make([]byte, 3*indexEntrySize)
		indexFile.Read(indexBuf)

		// Update the index file, so that we store
		// [ file = 2, offset = 1M ] at index zero

		tailId := uint32(2)           // First file is 2
		itemOffset := uint32(1000000) // We have removed 1M items
		zeroIndex := indexEntry{
			offset:  itemOffset,
			filenum: tailId,
		}
		buf := zeroIndex.append(nil)
		// Overwrite index zero
		copy(indexBuf, buf)
		indexFile.WriteAt(indexBuf, 0)
		indexFile.Close()
	}

	// Check that existing items have been moved to index 1M.
	{
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 40, true)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		t.Log(f.dumpIndexString(0, 100))

		checkRetrieveError(t, f, map[uint64]error{
			0:      errOutOfBounds,
			1:      errOutOfBounds,
			2:      errOutOfBounds,
			3:      errOutOfBounds,
			999999: errOutOfBounds,
		})
		checkRetrieve(t, f, map[uint64][]byte{
			1000000: getChunk(20, 0xbb),
			1000001: getChunk(20, 0xaa),
		})
	}
}

func checkRetrieve(t *testing.T, f *freezerTable, items map[uint64][]byte) {
	t.Helper()

	for item, wantBytes := range items {
		value, err := f.Retrieve(item)
		if err != nil {
			t.Fatalf("can't get expected item %d: %v", item, err)
		}
		if !bytes.Equal(value, wantBytes) {
			t.Fatalf("item %d has wrong value %x (want %x)", item, value, wantBytes)
		}
	}
}

func checkRetrieveError(t *testing.T, f *freezerTable, items map[uint64]error) {
	t.Helper()

	for item, wantError := range items {
		value, err := f.Retrieve(item)
		if err == nil {
			t.Fatalf("unexpected value %x for item %d, want error %v", item, value, wantError)
		}
		if err != wantError {
			t.Fatalf("wrong error for item %d: %v", item, err)
		}
	}
}

// Gets a chunk of data, filled with 'b'
func getChunk(size int, b int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(b)
	}
	return data
}

// TODO (?)
// - test that if we remove several head-files, aswell as data last data-file,
//   the index is truncated accordingly
// Right now, the freezer would fail on these conditions:
// 1. have data files d0, d1, d2, d3
// 2. remove d2,d3
//
// However, all 'normal' failure modes arising due to failing to sync() or save a file
// should be handled already, and the case described above can only (?) happen if an
// external process/user deletes files from the filesystem.

func writeChunks(t *testing.T, ft *freezerTable, n int, length int) {
	t.Helper()

	batch := ft.newBatch()
	for i := 0; i < n; i++ {
		if err := batch.AppendRaw(uint64(i), getChunk(length, i)); err != nil {
			t.Fatalf("AppendRaw(%d, ...) returned error: %v", i, err)
		}
	}
	if err := batch.commit(); err != nil {
		t.Fatalf("Commit returned error: %v", err)
	}
}

// TestSequentialRead does some basic tests on the RetrieveItems.
func TestSequentialRead(t *testing.T) {
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("batchread-%d", rand.Uint64())
	{ // Fill table
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 30 times
		writeChunks(t, f, 30, 15)
		f.DumpIndex(0, 30)
		f.Close()
	}
	{ // Open it, iterate, verify iteration
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		items, err := f.RetrieveItems(0, 10000, 100000)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := len(items), 30; have != want {
			t.Fatalf("want %d items, have %d ", want, have)
		}
		for i, have := range items {
			want := getChunk(15, i)
			if !bytes.Equal(want, have) {
				t.Fatalf("data corruption: have\n%x\n, want \n%x\n", have, want)
			}
		}
		f.Close()
	}
	{ // Open it, iterate, verify byte limit. The byte limit is less than item
		// size, so each lookup should only return one item
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 40, true)
		if err != nil {
			t.Fatal(err)
		}
		items, err := f.RetrieveItems(0, 10000, 10)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := len(items), 1; have != want {
			t.Fatalf("want %d items, have %d ", want, have)
		}
		for i, have := range items {
			want := getChunk(15, i)
			if !bytes.Equal(want, have) {
				t.Fatalf("data corruption: have\n%x\n, want \n%x\n", have, want)
			}
		}
		f.Close()
	}
}

// TestSequentialReadByteLimit does some more advanced tests on batch reads.
// These tests check that when the byte limit hits, we correctly abort in time,
// but also properly do all the deferred reads for the previous data, regardless
// of whether the data crosses a file boundary or not.
func TestSequentialReadByteLimit(t *testing.T) {
	rm, wm, sg := metrics.NewMeter(), metrics.NewMeter(), metrics.NewGauge()
	fname := fmt.Sprintf("batchread-2-%d", rand.Uint64())
	{ // Fill table
		f, err := newTable(os.TempDir(), fname, rm, wm, sg, 100, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 10 bytes 30 times,
		// Splitting it at every 100 bytes (10 items)
		writeChunks(t, f, 30, 10)
		f.Close()
	}
	for i, tc := range []struct {
		items uint64
		limit uint64
		want  int
	}{
		{9, 89, 8},
		{10, 99, 9},
		{11, 109, 10},
		{100, 89, 8},
		{100, 99, 9},
		{100, 109, 10},
	} {
		{
			f, err := newTable(os.TempDir(), fname, rm, wm, sg, 100, true)
			if err != nil {
				t.Fatal(err)
			}
			items, err := f.RetrieveItems(0, tc.items, tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if have, want := len(items), tc.want; have != want {
				t.Fatalf("test %d: want %d items, have %d ", i, want, have)
			}
			for ii, have := range items {
				want := getChunk(10, ii)
				if !bytes.Equal(want, have) {
					t.Fatalf("test %d: data corruption item %d: have\n%x\n, want \n%x\n", i, ii, have, want)
				}
			}
			f.Close()
		}
	}
}
