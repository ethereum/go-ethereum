// Copyright 2018 The go-ethereum Authors
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
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Gets a chunk of data, filled with 'b'
func getChunk(size int, b int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(b)
	}
	return data
}

func print(t *testing.T, f *freezerTable, item uint64) {
	a, err := f.Retrieve(item)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("db[%d] =  %x\n", item, a)
}

// TestFreezerBasics test initializing a freezertable from scratch, writing to the table,
// and reading it back.
func TestFreezerBasics(t *testing.T) {
	t.Parallel()
	// set cutoff at 50 bytes
	f, err := newCustomTable(os.TempDir(),
		fmt.Sprintf("unittest-%d", rand.Uint64()),
		metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter(), 50, true)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Write 15 bytes 255 times, results in 85 files
	for x := 0; x < 255; x++ {
		data := getChunk(15, x)
		f.Append(uint64(x), data)
	}

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
			t.Fatal(err)
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
		rm, wm, sc = metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
		f          *freezerTable
		err        error
	)
	f, err = newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
	if err != nil {
		t.Fatal(err)
	}
	// Write 15 bytes 255 times, results in 85 files
	for x := 0; x < 255; x++ {
		data := getChunk(15, x)
		f.Append(uint64(x), data)
		f.Close()
		f, err = newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
		f, err = newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestFreezerRepairDanglingHead tests that we can recover if index entries are removed
func TestFreezerRepairDanglingHead(t *testing.T) {
	t.Parallel()
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("dangling_headtest-%d", rand.Uint64())

	{ // Fill table
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		for x := 0; x < 255; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
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
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("dangling_headtest-%d", rand.Uint64())

	{ // Fill a table and close it
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		for x := 0; x < 0xff; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err == nil {
			if err != nil {
				t.Fatal(err)
			}
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
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		// The first item should be there
		if _, err = f.Retrieve(0); err != nil {
			t.Fatal(err)
		}
		// The second item should be missing
		if _, err = f.Retrieve(1); err == nil {
			t.Errorf("Expected error for missing index entry")
		}
		// We should now be able to store items again, from item = 1
		for x := 1; x < 0xff; x++ {
			data := getChunk(15, ^x)
			f.Append(uint64(x), data)
		}
		f.Close()
	}
	// And if we open it, we should now be able to read all of them (new values)
	{
		f, _ := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("snappytest-%d", rand.Uint64())
	// Open with snappy
	{
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 255 times
		for x := 0; x < 0xff; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
		f.Close()
	}
	// Open without snappy
	{
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, false)
		if _, err = f.Retrieve(0); err == nil {
			f.Close()
			t.Fatalf("expected empty table")
		}
	}

	// Open with snappy
	{
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("dangling_indextest-%d", rand.Uint64())

	{ // Fill a table and close it
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 9 times : 150 bytes
		for x := 0; x < 9; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
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
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		if f.items != 7 {
			f.Close()
			t.Fatalf("expected %d items, got %d", 7, f.items)
		}
		if err := assertFileSize(fileToCrop, 15); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFreezerTruncate(t *testing.T) {

	t.Parallel()
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("truncation-%d", rand.Uint64())

	{ // Fill table
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 30 times
		for x := 0; x < 30; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// Reopen, truncate
	{
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("truncationfirst-%d", rand.Uint64())
	{ // Fill table
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 80 bytes, splitting out into two files
		f.Append(0, getChunk(40, 0xFF))
		f.Append(1, getChunk(40, 0xEE))
		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
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
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		if f.items != 1 {
			f.Close()
			t.Fatalf("expected %d items, got %d", 0, f.items)
		}
		// Write 40 bytes
		f.Append(1, getChunk(40, 0xDD))
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
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("read_truncate-%d", rand.Uint64())
	{ // Fill table
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 15 bytes 30 times
		for x := 0; x < 30; x++ {
			data := getChunk(15, x)
			f.Append(uint64(x), data)
		}
		// The last item should be there
		if _, err = f.Retrieve(f.items - 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	// Reopen and read all files
	{
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 50, true)
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
		for x := 0; x < 30; x++ {
			data := getChunk(15, ^x)
			if err := f.Append(uint64(x), data); err != nil {
				t.Fatalf("error %v", err)
			}
		}
		f.Close()
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	rm, wm, sc := metrics.NewMeter(), metrics.NewMeter(), metrics.NewCounter()
	fname := fmt.Sprintf("offset-%d", rand.Uint64())
	{ // Fill table
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 40, true)
		if err != nil {
			t.Fatal(err)
		}
		// Write 6 x 20 bytes, splitting out into three files
		f.Append(0, getChunk(20, 0xFF))
		f.Append(1, getChunk(20, 0xEE))

		f.Append(2, getChunk(20, 0xdd))
		f.Append(3, getChunk(20, 0xcc))

		f.Append(4, getChunk(20, 0xbb))
		f.Append(5, getChunk(20, 0xaa))
		f.printIndex()
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
			offset:  tailId,
			filenum: itemOffset,
		}
		buf := zeroIndex.marshallBinary()
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
		f, err := newCustomTable(os.TempDir(), fname, rm, wm, sc, 40, true)
		if err != nil {
			t.Fatal(err)
		}
		f.printIndex()
		// It should allow writing item 6
		f.Append(6, getChunk(20, 0x99))

		// It should be fine to fetch 4,5,6
		if got, err := f.Retrieve(4); err != nil {
			t.Fatal(err)
		} else if exp := getChunk(20, 0xbb); !bytes.Equal(got, exp) {
			t.Fatalf("expected %x got %x", exp, got)
		}
		if got, err := f.Retrieve(5); err != nil {
			t.Fatal(err)
		} else if exp := getChunk(20, 0xaa); !bytes.Equal(got, exp) {
			t.Fatalf("expected %x got %x", exp, got)
		}
		if got, err := f.Retrieve(6); err != nil {
			t.Fatal(err)
		} else if exp := getChunk(20, 0x99); !bytes.Equal(got, exp) {
			t.Fatalf("expected %x got %x", exp, got)
		}

		// It should error at 0, 1,2,3
		for i := 0; i < 4; i++ {
			if _, err := f.Retrieve(uint64(i)); err == nil {
				t.Fatal("expected err")
			}
		}
	}
}

// TODO (?)
// - test that if we remove several head-files, aswell as data last data-file,
//   the index is truncated accordingly
// Right now, the freezer would fail on these conditions:
// 1. have data files d0, d1, d2, d3
// 2. remove d2,d3
//
// However, all 'normal' failure modes arising due to failing to sync() or save a file should be
// handled already, and the case described above can only (?) happen if an external process/user
// deletes files from the filesystem.
