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
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

// LeafSorter sorts a stream of key/value records too large to hold in memory:
// records accumulate in an in-memory run that is sorted and spilled to a
// temporary file when it outgrows the budget, and reading the sorted result
// merges every run. EIP-8347 asks for exactly this - "sort leaves by PBT key
// order (an external merge-sort for mainnet-scale state)" - and plain
// bytewise order is the PBT key order, since keys are prefix-free with the
// zone byte first.
//
// A duplicate key is an error, at Add time within the pending run and at
// merge time across runs. The one producer of legitimate duplicates, code
// shared between contracts, is deduplicated at derivation by code hash, so a
// duplicate surviving to the sorter means two different sources claimed the
// same leaf - corruption, not a coincidence to resolve silently.
type LeafSorter struct {
	tmpDir    string
	budget    int // bytes of buffered records that trigger a spill
	buffered  int
	pending   []leafRecord
	runs      []*os.File
	sealed    bool
	discarded bool
}

type leafRecord struct {
	key   []byte
	value []byte
}

// spillRecordOverhead approximates the bookkeeping bytes per buffered record
// beyond its key and value, so the budget tracks real memory rather than
// payload alone.
const spillRecordOverhead = 64

// NewLeafSorter creates a sorter spilling to files in tmpDir once the
// in-memory run exceeds budget bytes. A zero or negative budget sorts fully
// in memory and never spills.
func NewLeafSorter(tmpDir string, budget int) *LeafSorter {
	return &LeafSorter{tmpDir: tmpDir, budget: budget}
}

// Add buffers one record. Keys must be zone-conformant; values must be 32
// bytes and not all zero - the state layer resolves a zero value to absence,
// so a zero reaching the sorter is a bug in the caller, not a deletion to
// honour.
func (s *LeafSorter) Add(key, value []byte) error {
	if s.sealed {
		return errors.New("bintrie: sorter already sorted")
	}
	if err := validateKey(key); err != nil {
		return err
	}
	if len(value) != 32 {
		return fmt.Errorf("bintrie: sorter values must be 32 bytes, got %d", len(value))
	}
	if isZeroValue(value) {
		return fmt.Errorf("bintrie: zero value for key %x reached the sorter", key)
	}
	s.pending = append(s.pending, leafRecord{
		key:   bytes.Clone(key),
		value: bytes.Clone(value),
	})
	s.buffered += len(key) + len(value) + spillRecordOverhead
	if s.budget > 0 && s.buffered >= s.budget {
		return s.spill()
	}
	return nil
}

// spill sorts the pending run and writes it to a temporary file.
func (s *LeafSorter) spill() error {
	if len(s.pending) == 0 {
		return nil
	}
	if err := s.sortPending(); err != nil {
		return err
	}
	f, err := os.CreateTemp(s.tmpDir, "bintrie-sort-*")
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(f, 1<<20)
	for _, rec := range s.pending {
		if err := writeRunRecord(w, rec.key, rec.value); err != nil {
			f.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return err
	}
	s.runs = append(s.runs, f)
	s.pending = s.pending[:0]
	s.buffered = 0
	return nil
}

// sortPending orders the in-memory run and rejects duplicates, which sorting
// makes adjacent.
func (s *LeafSorter) sortPending() error {
	slices.SortFunc(s.pending, func(a, b leafRecord) int {
		return bytes.Compare(a.key, b.key)
	})
	for i := 1; i < len(s.pending); i++ {
		if bytes.Equal(s.pending[i-1].key, s.pending[i].key) {
			return fmt.Errorf("bintrie: duplicate key %x in sort input", s.pending[i].key)
		}
	}
	return nil
}

// Sort seals the sorter and returns the merged, ascending record stream. The
// caller owns the stream and must drain or close it; Close on the sorter
// releases the temporary files either way.
func (s *LeafSorter) Sort() (*LeafStream, error) {
	if s.sealed {
		return nil, errors.New("bintrie: sorter already sorted")
	}
	s.sealed = true

	// Everything fit in memory: serve the pending run directly.
	if len(s.runs) == 0 {
		if err := s.sortPending(); err != nil {
			return nil, err
		}
		return &LeafStream{pending: s.pending}, nil
	}
	// Spill the tail so the merge reads uniform sources.
	if err := s.spill(); err != nil {
		return nil, err
	}
	stream := &LeafStream{}
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

// Close removes the temporary files. Safe to call more than once, and safe
// while a returned stream is still live only after the stream is drained.
func (s *LeafSorter) Close() {
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
	s.pending = nil
}

// LeafStream yields the sorted records. Exactly one of pending and heap is
// populated: the in-memory fast path serves the slice, the merged path pops
// the run heap.
type LeafStream struct {
	pending []leafRecord
	next    int

	heap    runHeap
	lastKey []byte
}

// Next returns the next record in ascending key order, or io.EOF when the
// stream is exhausted. The returned slices are owned by the caller.
func (ls *LeafStream) Next() (key, value []byte, err error) {
	if ls.heap == nil {
		if ls.next >= len(ls.pending) {
			return nil, nil, io.EOF
		}
		rec := ls.pending[ls.next]
		ls.next++
		return rec.key, rec.value, nil
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

// advance reads the next record into the reader, io.EOF at the end.
func (rr *runReader) advance() error {
	klen, err := rr.r.ReadByte()
	if err != nil {
		return err // io.EOF included
	}
	key := make([]byte, klen)
	if _, err := io.ReadFull(rr.r, key); err != nil {
		return fmt.Errorf("bintrie: truncated sort run key: %w", err)
	}
	value := make([]byte, 32)
	if _, err := io.ReadFull(rr.r, value); err != nil {
		return fmt.Errorf("bintrie: truncated sort run value: %w", err)
	}
	rr.key, rr.value = key, value
	return nil
}

// writeRunRecord encodes one record as keyLen ‖ key ‖ value[32]. Key lengths
// are zone-fixed and at most 66, so one byte carries them.
func writeRunRecord(w *bufio.Writer, key, value []byte) error {
	if err := w.WriteByte(byte(len(key))); err != nil {
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
