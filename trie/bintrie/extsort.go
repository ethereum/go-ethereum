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

package bintrie

import (
	"bufio"
	"bytes"
	"container/heap"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
)

// RecordSorter is an external merge-sort: records accumulate in an in-memory
// run, spilled to a temporary file past the budget, and reading merges the
// runs. It serves EIP-8347's mainnet-scale sorts, tree leaves in key order
// (plain bytewise, the keys being prefix-free) first among them. Duplicate
// keys are corruption and error out at sort or merge.
type RecordSorter struct {
	tmpDir    string
	budget    int // bytes of buffered records that trigger a spill
	validate  func(key, value []byte) error
	buffered  int
	arena     []byte        // pending run's keys and values, packed back to back
	index     []arenaRecord // pending run, offsets into arena
	runs      []*os.File
	sealed    bool
	discarded bool
}

// arenaRecord locates one record inside a RecordSorter's arena: its key at
// off, its value right after. Storing offsets instead of slices keeps Add to
// one append per field and lets a spill reset the arena in place rather than
// freeing per-record backing arrays.
type arenaRecord struct {
	off, klen, vlen int
}

// key and value slice the arena for one record, capacity clipped so that an
// append on either cannot reach the next record.
func (r arenaRecord) key(arena []byte) []byte {
	return arena[r.off : r.off+r.klen : r.off+r.klen]
}

func (r arenaRecord) value(arena []byte) []byte {
	end := r.off + r.klen + r.vlen
	return arena[r.off+r.klen : end : end]
}

// spillRecordOverhead approximates per-record bookkeeping bytes, so the
// budget tracks real memory.
const spillRecordOverhead = 64

// NewRecordSorter creates a sorter spilling to tmpDir past budget bytes
// (non-positive: never spills). validate runs per Add; nil accepts all.
func NewRecordSorter(tmpDir string, budget int, validate func(key, value []byte) error) *RecordSorter {
	return &RecordSorter{tmpDir: tmpDir, budget: budget, validate: validate}
}

// NewLeafSorter creates a sorter for tree leaves: zone-conformant keys,
// 32-byte non-zero values.
func NewLeafSorter(tmpDir string, budget int) *RecordSorter {
	return NewRecordSorter(tmpDir, budget, validateLeafRecord)
}

// validateLeafRecord enforces the tree-leaf shape on a sorter record.
func validateLeafRecord(key, value []byte) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if len(value) != 32 {
		return fmt.Errorf("bintrie: leaf values must be 32 bytes, got %d", len(value))
	}
	if isZeroValue(value) {
		return fmt.Errorf("bintrie: zero value for key %x reached the sorter", key)
	}
	return nil
}

// Add buffers one record. Keys must be 1 to 255 bytes for the run encoding,
// plus whatever the validation hook demands. Add copies both key and value
// into its own arena; the caller's slices are free to reuse the moment Add
// returns.
func (s *RecordSorter) Add(key, value []byte) error {
	if s.sealed {
		return errors.New("bintrie: sorter already sorted")
	}
	if len(key) == 0 || len(key) > math.MaxUint8 {
		return fmt.Errorf("bintrie: sorter keys must be 1 to 255 bytes, got %d", len(key))
	}
	if uint64(len(value)) > math.MaxUint32 {
		return fmt.Errorf("bintrie: sorter value of %d bytes exceeds the run encoding", len(value))
	}
	if s.validate != nil {
		if err := s.validate(key, value); err != nil {
			return err
		}
	}
	off := len(s.arena)
	s.arena = append(append(s.arena, key...), value...)
	s.index = append(s.index, arenaRecord{off: off, klen: len(key), vlen: len(value)})
	s.buffered += len(key) + len(value) + spillRecordOverhead
	if s.budget > 0 && s.buffered >= s.budget {
		return s.spill()
	}
	return nil
}

// spill sorts the pending run and writes it to a temporary file.
func (s *RecordSorter) spill() error {
	if len(s.index) == 0 {
		return nil
	}
	if err := s.sortPending(); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.tmpDir, "bintrie-sort-*")
	if err != nil {
		return err
	}
	// Until the run is registered this function owns the file; errors remove
	// it rather than orphan a spill on a full disk.
	discard := func(err error) error {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	for _, rec := range s.index {
		if err := writeRunRecord(w, rec.key(s.arena), rec.value(s.arena)); err != nil {
			return discard(err)
		}
	}
	if err := w.Flush(); err != nil {
		return discard(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return discard(err)
	}
	s.runs = append(s.runs, f)
	s.arena = s.arena[:0]
	s.index = s.index[:0]
	s.buffered = 0
	return nil
}

// sortPending orders the in-memory run and rejects duplicates, which sorting
// makes adjacent.
func (s *RecordSorter) sortPending() error {
	arena := s.arena
	slices.SortFunc(s.index, func(a, b arenaRecord) int {
		return bytes.Compare(a.key(arena), b.key(arena))
	})
	for i := 1; i < len(s.index); i++ {
		if bytes.Equal(s.index[i-1].key(arena), s.index[i].key(arena)) {
			return fmt.Errorf("bintrie: duplicate key %x in sort input", s.index[i].key(arena))
		}
	}
	return nil
}

// Sort seals the sorter and returns the merged, ascending record stream.
// Close on the sorter releases the temporary files.
func (s *RecordSorter) Sort() (*RecordStream, error) {
	if s.sealed {
		return nil, errors.New("bintrie: sorter already sorted")
	}
	s.sealed = true

	// Everything fit in memory: serve the pending run.
	if len(s.runs) == 0 {
		if err := s.sortPending(); err != nil {
			return nil, err
		}
		return &RecordStream{arena: s.arena, index: s.index}, nil
	}
	// Spill the tail so the merge reads uniform sources. Nothing is added
	// after Sort, so the pending run's buffers go now rather than at Close.
	if err := s.spill(); err != nil {
		return nil, err
	}
	s.arena, s.index = nil, nil
	stream := &RecordStream{}
	for _, f := range s.runs {
		src := &runReader{r: bufio.NewReaderSize(f, 1<<20)}
		if err := src.advance(); err != nil {
			if err == io.EOF {
				continue
			}
			return nil, err
		}
		stream.heap = append(stream.heap, src)
	}
	heap.Init(&stream.heap)
	return stream, nil
}

// Close removes the temporary files; call after any returned stream drains.
func (s *RecordSorter) Close() {
	if s.discarded {
		return
	}
	s.discarded = true
	for _, f := range s.runs {
		name := f.Name()
		f.Close()
		os.Remove(name)
	}
	s.runs = nil
	s.arena = nil
	s.index = nil
}

// RecordStream yields the sorted records: the in-memory path serves the
// pending arena, the merged path pops the run heap.
type RecordStream struct {
	arena []byte
	index []arenaRecord
	next  int

	heap    runHeap
	lastKey []byte
}

// Next returns the next record in ascending key order, io.EOF at the end.
// The returned slices are the caller's.
func (ls *RecordStream) Next() (key, value []byte, err error) {
	if ls.heap == nil {
		if ls.next >= len(ls.index) {
			return nil, nil, io.EOF
		}
		rec := ls.index[ls.next]
		ls.next++
		return rec.key(ls.arena), rec.value(ls.arena), nil
	}
	if len(ls.heap) == 0 {
		return nil, nil, io.EOF
	}
	src := ls.heap[0]
	key, value = src.key, src.value
	if ls.lastKey != nil && bytes.Compare(key, ls.lastKey) <= 0 {
		// Equal keys across runs; within-run duplicates died at spill time.
		return nil, nil, fmt.Errorf("bintrie: duplicate key %x across sort runs", key)
	}
	ls.lastKey = bytes.Clone(key)
	if err := src.advance(); err != nil {
		if err != io.EOF {
			return nil, nil, err
		}
		heap.Pop(&ls.heap)
	} else {
		heap.Fix(&ls.heap, 0)
	}
	return key, value, nil
}

// runReader streams one spilled run.
type runReader struct {
	r     *bufio.Reader
	key   []byte
	value []byte
}

// advance reads the next record into the reader, io.EOF at the end. Key and
// value share one allocation: the run encoding puts both lengths ahead of
// both payloads, so the combined size is known before any payload byte is
// read.
func (rr *runReader) advance() error {
	klen, err := rr.r.ReadByte()
	if err != nil {
		return err // io.EOF included
	}
	var vlenBuf [4]byte
	if _, err := io.ReadFull(rr.r, vlenBuf[:]); err != nil {
		return fmt.Errorf("bintrie: truncated sort run value length: %w", err)
	}
	vlen := binary.BigEndian.Uint32(vlenBuf[:])
	buf := make([]byte, int(klen)+int(vlen))
	if _, err := io.ReadFull(rr.r, buf); err != nil {
		return fmt.Errorf("bintrie: truncated sort run record: %w", err)
	}
	rr.key, rr.value = buf[:klen], buf[klen:]
	return nil
}

// writeRunRecord encodes one record as keyLen ‖ valueLen ‖ key ‖ value: both
// lengths ahead of both payloads, so a reader sizes one combined buffer
// before it reads any payload byte.
func writeRunRecord(w *bufio.Writer, key, value []byte) error {
	if err := w.WriteByte(byte(len(key))); err != nil {
		return err
	}
	var vlen [4]byte
	binary.BigEndian.PutUint32(vlen[:], uint32(len(value)))
	if _, err := w.Write(vlen[:]); err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	_, err := w.Write(value)
	return err
}

// runHeap orders run readers by their current key.
type runHeap []*runReader

func (h runHeap) Len() int           { return len(h) }
func (h runHeap) Less(i, j int) bool { return bytes.Compare(h[i].key, h[j].key) < 0 }
func (h runHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *runHeap) Push(x any) { *h = append(*h, x.(*runReader)) }

func (h *runHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return item
}
