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
	"sync"
)

// RecordSorter is an external merge-sort: records accumulate in an in-memory
// run, spilled to a temporary file past the budget, and reading merges the
// runs. It serves EIP-8347's mainnet-scale sorts, tree leaves in key order
// (plain bytewise, the keys being prefix-free) first among them. Duplicate
// keys are corruption and error out at sort or merge.
type RecordSorter struct {
	tmpDir    string
	budget    int // bytes of buffered records held at once, across both runs
	validate  func(key, value []byte) error
	buffered  int
	arena     []byte        // pending run's keys and values, packed back to back
	index     []arenaRecord // pending run, offsets into arena
	runs      []*os.File
	merging   *RecordStream // merged stream, its goroutine stopped at Close
	sealed    bool
	discarded bool

	// A full run is sorted and written on a goroutine of its own while Add
	// fills the spare buffers; spilling delivers the file and the buffers.
	spilling   chan spilled
	spareArena []byte
	spareIndex []arenaRecord
}

// spilled is the outcome of a background spill, with the buffers it held.
type spilled struct {
	run   *os.File
	arena []byte
	index []arenaRecord
	err   error
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

// arenaRecordBytes is what a buffered record costs beyond its key and value:
// its arenaRecord in the index. The arena holds keys and values exactly, so
// the budget is the memory the two runs hold.
const arenaRecordBytes = 24

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
	if s.budget > 0 && s.arena == nil {
		// One allocation per run buffer, the size the run spills at: a growing
		// arena would copy itself several times over before reaching it.
		s.arena = make([]byte, 0, max(s.budget/2, 1))
	}
	off := len(s.arena)
	s.arena = append(append(s.arena, key...), value...)
	// The hook sees the arena's copy: handing it the caller's slices would
	// move every caller's key and value to the heap.
	if s.validate != nil {
		if err := s.validate(s.arena[off:off+len(key)], s.arena[off+len(key):]); err != nil {
			s.arena = s.arena[:off]
			return err
		}
	}
	s.index = append(s.index, arenaRecord{off: off, klen: len(key), vlen: len(value)})
	s.buffered += len(key) + len(value) + arenaRecordBytes
	if s.budget > 0 && s.buffered >= max(s.budget/2, 1) {
		return s.spillAsync()
	}
	return nil
}

// spillAsync hands the pending run to a goroutine that sorts and writes it,
// and carries on in the spare buffers. One spill runs at a time, so the two
// runs held stay within the budget.
func (s *RecordSorter) spillAsync() error {
	if err := s.wait(); err != nil {
		return err
	}
	tmpDir, arena, index := s.tmpDir, s.arena, s.index
	s.arena, s.index, s.buffered = s.spareArena[:0], s.spareIndex[:0], 0
	if s.index == nil {
		// The first spill has no spare: start the second run at the first's size.
		s.index = make([]arenaRecord, 0, cap(index))
	}
	s.spilling = make(chan spilled, 1)
	go func(done chan<- spilled) {
		run, err := writeRun(tmpDir, arena, index)
		done <- spilled{run, arena, index, err}
	}(s.spilling)
	return nil
}

// wait collects the background spill, if one runs.
func (s *RecordSorter) wait() error {
	if s.spilling == nil {
		return nil
	}
	r := <-s.spilling
	s.spilling = nil
	s.spareArena, s.spareIndex = r.arena, r.index
	if r.err != nil {
		return r.err
	}
	s.runs = append(s.runs, r.run)
	return nil
}

// spill writes the pending run out before returning.
func (s *RecordSorter) spill() error {
	if err := s.wait(); err != nil {
		return err
	}
	if len(s.index) == 0 {
		return nil
	}
	run, err := writeRun(s.tmpDir, s.arena, s.index)
	if err != nil {
		return err
	}
	s.runs = append(s.runs, run)
	s.arena, s.index, s.buffered = s.arena[:0], s.index[:0], 0
	return nil
}

// runWriters holds the spill writers: a run's 1 MiB buffer would otherwise be
// garbage once per spill.
var runWriters = sync.Pool{New: func() any { return bufio.NewWriterSize(nil, 1<<20) }}

// writeRun sorts a run and writes it to a temporary file, rewound for reading.
func writeRun(tmpDir string, arena []byte, index []arenaRecord) (*os.File, error) {
	if err := sortRun(arena, index); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(tmpDir, "bintrie-sort-*")
	if err != nil {
		return nil, err
	}
	// Until the run is returned this function owns the file; errors remove
	// it rather than orphan a spill on a full disk.
	discard := func(err error) (*os.File, error) {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	w := runWriters.Get().(*bufio.Writer)
	w.Reset(f)
	defer func() {
		w.Reset(nil)
		runWriters.Put(w)
	}()
	for _, rec := range index {
		if err := writeRunRecord(w, rec.key(arena), rec.value(arena)); err != nil {
			return discard(err)
		}
	}
	if err := w.Flush(); err != nil {
		return discard(err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return discard(err)
	}
	return f, nil
}

// sortRun orders a run and rejects duplicates, which sorting makes adjacent.
func sortRun(arena []byte, index []arenaRecord) error {
	slices.SortFunc(index, func(a, b arenaRecord) int {
		return bytes.Compare(a.key(arena), b.key(arena))
	})
	for i := 1; i < len(index); i++ {
		if bytes.Equal(index[i-1].key(arena), index[i].key(arena)) {
			return fmt.Errorf("bintrie: duplicate key %x in sort input", index[i].key(arena))
		}
	}
	return nil
}

// Absorb moves other's records into s: other's pending run spills and its
// runs join s's, so s.Sort merges both record sets into one stream, with
// duplicates across them rejected like any other. It lets parallel producers
// sort into sorters of their own and share one merge. other holds nothing
// afterwards and only Close remains valid on it.
func (s *RecordSorter) Absorb(other *RecordSorter) error {
	if s.sealed || other.sealed {
		return errors.New("bintrie: sorter already sorted")
	}
	if err := other.spill(); err != nil {
		return err
	}
	s.runs = append(s.runs, other.runs...)
	other.runs, other.arena, other.index, other.spareArena, other.spareIndex = nil, nil, nil, nil, nil
	other.sealed = true
	return nil
}

// Sort seals the sorter and returns the merged, ascending record stream.
// Close on the sorter releases the temporary files.
func (s *RecordSorter) Sort() (*RecordStream, error) {
	if s.sealed {
		return nil, errors.New("bintrie: sorter already sorted")
	}
	s.sealed = true
	if err := s.wait(); err != nil {
		return nil, err
	}

	// Everything fit in memory: serve the pending run.
	if len(s.runs) == 0 {
		if err := sortRun(s.arena, s.index); err != nil {
			return nil, err
		}
		return &RecordStream{arena: s.arena, index: s.index}, nil
	}
	// Spill the tail so the merge reads uniform sources. Nothing is added
	// after Sort, so the run buffers go now rather than at Close.
	if err := s.spill(); err != nil {
		return nil, err
	}
	s.arena, s.index, s.spareArena, s.spareIndex = nil, nil, nil, nil
	var (
		h    runHeap
		size = mergeBufSize(s.budget, len(s.runs))
	)
	for _, f := range s.runs {
		src := &runReader{r: bufio.NewReaderSize(f, size)}
		if err := src.advance(); err != nil {
			if err == io.EOF {
				continue
			}
			return nil, err
		}
		h = append(h, src)
	}
	heap.Init(&h)
	s.merging = newMergeStream(h)
	return s.merging, nil
}

// mergeBufSize sizes each run's read buffer for a merge of runs runs: 512 KiB,
// less once the buffers together would pass a quarter of the budget, never
// under 64 KiB. A mainnet sort merges thousands of runs.
func mergeBufSize(budget, runs int) int {
	if budget <= 0 || runs == 0 {
		return 1 << 19
	}
	return min(1<<19, max(64<<10, budget/(4*runs)))
}

// Close removes the temporary files; call after any returned stream drains.
func (s *RecordSorter) Close() {
	if s.discarded {
		return
	}
	s.discarded = true
	if s.spilling != nil {
		if r := <-s.spilling; r.run != nil {
			s.runs = append(s.runs, r.run)
		}
		s.spilling = nil
	}
	if s.merging != nil {
		s.merging.stop()
	}
	for _, f := range s.runs {
		name := f.Name()
		f.Close()
		os.Remove(name)
	}
	s.runs = nil
	s.arena, s.index, s.spareArena, s.spareIndex = nil, nil, nil, nil
}

// RecordStream yields the sorted records: the in-memory path serves the
// pending arena, the merged path batches a goroutine merges ahead of Next.
type RecordStream struct {
	arena []byte
	index []arenaRecord
	next  int

	batch   *mergeBatch // being read, recycled when the next one starts
	batches chan *mergeBatch
	free    chan *mergeBatch
	quit    chan struct{}
	done    chan struct{}
}

// mergeBatch carries merged records, then the merge's end or error.
type mergeBatch struct {
	arena []byte
	index []arenaRecord
	err   error
}

// mergeBatchSize and mergeBatches size the read-ahead: records per batch and
// batches in flight.
var (
	mergeBatchSize = 4096
	mergeBatches   = 4
)

// newMergeStream starts merging the run heap ahead of the reader.
func newMergeStream(h runHeap) *RecordStream {
	ls := &RecordStream{
		batches: make(chan *mergeBatch, mergeBatches),
		free:    make(chan *mergeBatch, mergeBatches),
		quit:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	for range mergeBatches {
		ls.free <- new(mergeBatch)
	}
	go ls.merge(h)
	return ls
}

// merge pops the run heap into batches until the runs end, fail or stop.
func (ls *RecordStream) merge(h runHeap) {
	defer close(ls.done)
	defer close(ls.batches)
	var lastKey []byte
	for {
		var b *mergeBatch
		select {
		case b = <-ls.free:
		case <-ls.quit:
			return
		}
		b.arena, b.index, b.err = b.arena[:0], b.index[:0], nil
		for len(b.index) < mergeBatchSize && b.err == nil {
			if len(h) == 0 {
				b.err = io.EOF
				break
			}
			src := h[0]
			if lastKey != nil && bytes.Compare(src.key, lastKey) <= 0 {
				// Equal keys across runs; within-run duplicates died at spill time.
				b.err = fmt.Errorf("bintrie: duplicate key %x across sort runs", src.key)
				break
			}
			lastKey = append(lastKey[:0], src.key...)
			off := len(b.arena)
			b.arena = append(append(b.arena, src.key...), src.value...)
			b.index = append(b.index, arenaRecord{off: off, klen: len(src.key), vlen: len(src.value)})
			if err := src.advance(); err == io.EOF {
				heap.Pop(&h)
			} else if err != nil {
				b.err = err
			} else {
				heap.Fix(&h, 0)
			}
		}
		select {
		case ls.batches <- b:
		case <-ls.quit:
			return
		}
		if b.err != nil {
			return
		}
	}
}

// stop ends the merge, so the runs can be released.
func (ls *RecordStream) stop() {
	close(ls.quit)
	<-ls.done
}

// Next returns the next record in ascending key order, io.EOF at the end.
// The returned slices are valid until the following call.
func (ls *RecordStream) Next() (key, value []byte, err error) {
	for ls.next >= len(ls.index) {
		if ls.batches == nil {
			return nil, nil, io.EOF
		}
		if b := ls.batch; b != nil {
			if b.err != nil {
				return nil, nil, b.err
			}
			ls.free <- b
		}
		b, ok := <-ls.batches
		if !ok {
			return nil, nil, errors.New("bintrie: sort stream closed")
		}
		ls.batch, ls.arena, ls.index, ls.next = b, b.arena, b.index, 0
	}
	rec := ls.index[ls.next]
	ls.next++
	return rec.key(ls.arena), rec.value(ls.arena), nil
}

// runReader streams one spilled run.
type runReader struct {
	r          *bufio.Reader
	buf        []byte
	key, value []byte
}

// advance reads the next record into the reader's buffer, overwriting the
// last, io.EOF at the end. The run encoding puts both lengths ahead of both
// payloads, so the buffer is sized before any payload byte is read.
func (rr *runReader) advance() error {
	klen, err := rr.r.ReadByte()
	if err != nil {
		return err // io.EOF included
	}
	var vlenBuf [4]byte
	if _, err := io.ReadFull(rr.r, vlenBuf[:]); err != nil {
		return fmt.Errorf("bintrie: truncated sort run value length: %w", err)
	}
	n := int(klen) + int(binary.BigEndian.Uint32(vlenBuf[:]))
	if cap(rr.buf) < n {
		rr.buf = make([]byte, n)
	}
	buf := rr.buf[:n]
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
