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
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// The synthetic block's shape, calibrated to about 50 ms on 8 cores without an
// import, mainnet's newPayload median on the benchmark box: on an EPYC 9454P,
// execution takes 24 ms, reads 19, hashing 6 and the commit 2.
const (
	blockExecOps   = 25_000 // serial: one hashed, allocated state object each
	blockReads     = 1536   // parallel: random flat-state reads, prefetcher-like
	blockReaders   = 8
	blockWrites    = 1024    // the commit's flat-state updates
	blockHashBytes = 2 << 20 // hashed by each of the commit's 4 goroutines
)

// BenchmarkImportBesideBlocks measures what an online import costs the node's
// block processing. A stand-in for the node - a database holding flat state
// outside the binary tree namespace, a live heap like the node's caches and a
// synthetic block processed every $BLOCKS_INTERVAL - runs alone, then beside an
// import of BenchmarkImport's pair into the same database, with the in-node
// sort budget scaled to the pair. It reports block latency for both, overall
// and per import stage, and the import's duration.
//
// $BLOCKS_STATE_GB (4) sizes the flat state, $BLOCKS_HEAP_GB (2) the live heap,
// $BLOCKS_ALONE (2m) the first stretch and $IMPORT_WORKERS (4) the import. Run
// it bounded like a node, for example under systemd-run -p MemoryMax=10G
// taskset -c 0-7, so that the import's I/O competes for the page cache, and
// keep other work off those cores' SMT siblings. A page is charged to the
// cgroup that first read it, so drop the state and the pair from the page
// cache before each run and read the state back inside the cgroup: pages an
// earlier run left cached are out of this run's reach, and runs differ.
func BenchmarkImportBesideBlocks(b *testing.B) {
	dir := os.Getenv("BINTRIE_BENCH_DIR")
	if dir == "" {
		b.Skip("BINTRIE_BENCH_DIR not set")
	}
	var (
		accounts = envInt(b, "BINTRIE_BENCH_ACCOUNTS", 21_208_000)
		stateGB  = envInt(b, "BLOCKS_STATE_GB", 4)
		heapGB   = envInt(b, "BLOCKS_HEAP_GB", 2)
		workers  = envInt(b, "IMPORT_WORKERS", 4)
		interval = envDuration(b, "BLOCKS_INTERVAL", time.Second)
		alone    = envDuration(b, "BLOCKS_ALONE", 2*time.Minute)
	)
	pairDir := filepath.Join(dir, fmt.Sprintf("accounts-%d", accounts))
	root := benchArtifacts(b, pairDir, accounts)
	budget := int(onlineSortBudget * int64(accounts) / 424_160_000)
	stateDir, keys := blocksState(b, dir, stateGB)

	defer debug.SetGCPercent(debug.SetGCPercent(50)) // the node's --gogc
	heap := liveHeap(heapGB)
	defer heap.release()

	stages := &importStage{}
	defer log.SetDefault(log.Root())
	log.SetDefault(log.NewLogger(stages))

	dest, tmp := filepath.Join(dir, "beside"), filepath.Join(dir, "beside-tmp")
	defer os.RemoveAll(dest)
	for range b.N {
		b.StopTimer()
		os.RemoveAll(dest)
		os.RemoveAll(tmp)
		os.MkdirAll(tmp, 0o755)
		if err := cloneDir(stateDir, dest); err != nil {
			b.Fatal(err)
		}
		db := openBenchDB(b, dest)
		node := &standIn{db: db, keys: keys, yield: newBlockYield()}

		stages.reset()
		baseline := node.run(interval, closeAfter(alone), stages)

		stages.reset()
		done := make(chan struct{})
		var (
			importErr error
			spent     time.Duration
		)
		b.StartTimer()
		go func() {
			defer close(done)
			start := time.Now()
			_, importErr = importState(context.Background(), db, importOptions{
				snapshot:          filepath.Join(pairDir, "snapshot.bin"),
				preimages:         filepath.Join(pairDir, "preimages.bin"),
				anchor:            &types.Header{Number: new(big.Int), Root: root},
				yield:             node.yield,
				conversionOptions: conversionOptions{tmpDir: tmp, sortBudget: budget, workers: workers},
			})
			spent = time.Since(start)
		}()
		beside := node.run(interval, done, stages)
		b.StopTimer()
		db.Close()
		if importErr != nil {
			b.Fatal(importErr)
		}
		reportBlocks(b, baseline, beside, spent)
	}
}

// blockTime is one block's processing time and the import stage it ran in.
type blockTime struct {
	took  time.Duration
	stage int
}

// reportBlocks reports the block latencies alone and beside the import,
// overall and per stage, as milliseconds.
func reportBlocks(b *testing.B, alone, beside []blockTime, spent time.Duration) {
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
	sorted := func(ts []blockTime, stage int) []time.Duration {
		var out []time.Duration
		for _, t := range ts {
			if stage < 0 || t.stage == stage {
				out = append(out, t.took)
			}
		}
		slices.Sort(out)
		return out
	}
	pct := func(ds []time.Duration, p float64) time.Duration {
		if len(ds) == 0 {
			return 0
		}
		return ds[min(len(ds)-1, int(p*float64(len(ds))))]
	}
	a, s := sorted(alone, -1), sorted(beside, -1)
	for _, p := range []struct {
		name string
		q    float64
	}{{"p50", 0.50}, {"p95", 0.95}, {"p99", 0.99}} {
		b.ReportMetric(ms(pct(a, p.q)), "alone-"+p.name+"-ms")
		b.ReportMetric(ms(pct(s, p.q)), "beside-"+p.name+"-ms")
	}
	b.ReportMetric(ms(s[len(s)-1]), "beside-max-ms")
	b.ReportMetric(float64(len(s)), "blocks")
	b.ReportMetric(spent.Seconds(), "import-s")

	var lines []string
	for i := range len(importMarks) {
		st := sorted(beside, i)
		lines = append(lines, fmt.Sprintf("%-9s n=%-4d p50=%6.1f p95=%6.1f p99=%6.1f max=%6.1f ms",
			importMarks[i].name, len(st), ms(pct(st, 0.5)), ms(pct(st, 0.95)), ms(pct(st, 0.99)), ms(pct(st, 1))))
	}
	b.Logf("blocks beside the import, by stage:\n%s", strings.Join(lines, "\n"))
}

// standIn stands in for the node's block processing over a database it shares
// with the import. It announces each block to yield, as the chain announces
// its inserts.
type standIn struct {
	db    ethdb.Database
	keys  int // flat-state records, keyed by blocksStateKey
	seq   uint64
	yield *blockYield
}

// run processes a block every interval until stop fires, the next starting
// as soon as the last ends if it overran, and returns their processing times.
func (s *standIn) run(interval time.Duration, stop <-chan struct{}, stages *importStage) []blockTime {
	var (
		times []blockTime
		tick  = time.NewTicker(interval)
	)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return times
		case <-tick.C:
		}
		stage := stages.current()
		start := time.Now()
		s.yield.set(true)
		s.block()
		s.yield.set(false)
		times = append(times, blockTime{time.Since(start), stage})
	}
}

// block processes one synthetic block: a serial execution allocating and
// hashing state objects, parallel random reads of the flat state, and a commit
// that hashes on four goroutines and writes one batch.
func (s *standIn) block() {
	s.seq++
	rng := rand.New(rand.NewPCG(s.seq, 0))

	objects := make(map[common.Hash][]byte, blockExecOps)
	var seed [8]byte
	for i := range blockExecOps {
		binary.BigEndian.PutUint64(seed[:], s.seq<<32|uint64(i))
		objects[crypto.Keccak256Hash(seed[:])] = make([]byte, 96)
	}

	var wg sync.WaitGroup
	wg.Add(blockReaders)
	for r := range blockReaders {
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(s.seq, uint64(r)+1))
			for range blockReads / blockReaders {
				s.db.Get(blocksStateKey(rng.IntN(s.keys)))
			}
		}()
	}
	wg.Wait()

	wg.Add(4)
	for range 4 {
		go func() {
			defer wg.Done()
			h := crypto.NewKeccakState()
			buf := make([]byte, 4096)
			for n := 0; n < blockHashBytes; n += len(buf) {
				h.Write(buf)
			}
			h.Read(buf[:32])
		}()
	}
	wg.Wait()
	batch := s.db.NewBatch()
	value := make([]byte, 100)
	for range blockWrites {
		binary.BigEndian.PutUint64(value, rng.Uint64())
		batch.Put(blocksStateKey(rng.IntN(s.keys)), value)
	}
	if err := batch.Write(); err != nil {
		panic(err)
	}
}

// closeAfter returns a channel closed once d has passed.
func closeAfter(d time.Duration) <-chan struct{} {
	ch := make(chan struct{})
	time.AfterFunc(d, func() { close(ch) })
	return ch
}

// blocksStateKey returns the flat-state key of the i-th stand-in record, in
// the merkle namespace beside which the import writes the binary tree's.
func blocksStateKey(i int) []byte {
	var n [8]byte
	binary.BigEndian.PutUint64(n[:], uint64(i))
	return append(slices.Clone(rawdb.SnapshotAccountPrefix), crypto.Keccak256(n[:])...)
}

// blocksState returns a compacted database holding gb GB of stand-in flat
// state, built once under dir, and how many records it holds.
func blocksState(b *testing.B, dir string, gb int) (string, int) {
	path := filepath.Join(dir, fmt.Sprintf("blocks-state-%dgb", gb))
	keys := gb * 1e9 / 140 // a 33-byte key and a 100-byte value each, plus overhead
	if _, err := os.Stat(filepath.Join(path, "done")); err == nil {
		return path, keys
	}
	b.Logf("generating %d GB of flat state into %s", gb, path)
	os.RemoveAll(path)
	db := openBenchDB(b, path)
	batch := db.NewBatch()
	value := make([]byte, 100)
	rng := rand.NewChaCha8([32]byte{1})
	for i := range keys {
		rng.Read(value)
		batch.Put(blocksStateKey(i), value)
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				b.Fatal(err)
			}
			batch.Reset()
		}
	}
	if err := batch.Write(); err != nil {
		b.Fatal(err)
	}
	if err := db.Compact(nil, nil); err != nil {
		b.Fatal(err)
	}
	db.Close()
	if err := os.WriteFile(filepath.Join(path, "done"), nil, 0o644); err != nil {
		b.Fatal(err)
	}
	return path, keys
}

// cloneDir copies the database at src to dst, linking the immutable tables
// rather than copying them.
func cloneDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case strings.HasSuffix(path, ".sst"):
			return os.Link(path, target)
		default:
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, in); err != nil {
				out.Close()
				return err
			}
			return out.Close()
		}
	})
}

// ballast is a live, pointer-heavy heap standing in for the node's caches, so
// that each GC cycle marks about as much as it does in a node.
type ballast struct{ nodes []*ballastNode }

type ballastNode struct {
	next *ballastNode
	_    [112]byte // a 128-byte object, mostly not pointers
}

func liveHeap(gb int) *ballast {
	h := &ballast{nodes: make([]*ballastNode, gb*1e9/128)}
	for i := range h.nodes {
		h.nodes[i] = &ballastNode{}
		if i > 0 {
			h.nodes[i].next = h.nodes[i-1]
		}
	}
	return h
}

func (h *ballast) release() { h.nodes = nil }

// importStage is a log handler tracking which import stage runs, by the lines
// closing each (importMarks).
type importStage struct{ stage atomic.Int32 }

func (s *importStage) reset()       { s.stage.Store(0) }
func (s *importStage) current() int { return int(s.stage.Load()) }

func (s *importStage) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}
func (s *importStage) WithAttrs([]slog.Attr) slog.Handler { return s }
func (s *importStage) WithGroup(string) slog.Handler      { return s }

func (s *importStage) Handle(_ context.Context, r slog.Record) error {
	for i, m := range importMarks {
		if r.Message == m.msg {
			s.stage.Store(int32(min(i+1, len(importMarks)-1)))
		}
	}
	return nil
}

func envInt(b *testing.B, name string, def int) int {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		b.Fatalf("%s: %v", name, err)
	}
	return n
}

func envDuration(b *testing.B, name string, def time.Duration) time.Duration {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		b.Fatalf("%s: %v", name, err)
	}
	return d
}
