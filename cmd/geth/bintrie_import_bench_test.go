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
	"log/slog"
	"math"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// importMarks are the log lines closing each import phase, in order.
var importMarks = []struct{ msg, name string }{
	{"Verified preimage file", "preimages"},
	{"Verified snapshot consistency", "snapshot"},
	{"Verified code against its hashes", "replay"},
	{"Verified consensus anchoring", "check2"},
	{"Import complete", "readback"},
}

// BenchmarkImport times the import of a mainnet-shaped snapshot and preimage
// file, generated once into $BINTRIE_BENCH_DIR; without it the benchmark is
// skipped. $BINTRIE_BENCH_ACCOUNTS sizes the state, 5% of mainnet's accounts
// by default, and the sort budget scales with it like mainnet's 4096 MB to its
// 424 M accounts, so the sorters spill to as many runs as on mainnet.
func BenchmarkImport(b *testing.B) {
	dir, accounts := benchSize(b, "accounts")
	root := benchArtifacts(b, dir, accounts)
	budget := int(sortBudget * int64(accounts) / 424_160_000)

	defer debug.SetGCPercent(debug.SetGCPercent(50)) // the command's --gogc
	defer log.SetDefault(log.Root())
	for _, workers := range []int{4, 8} {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			clock := &phaseClock{marks: make(map[string]time.Time)}
			log.SetDefault(log.NewLogger(clock))
			dest, tmp := filepath.Join(dir, "dest"), filepath.Join(dir, "tmp")
			defer os.RemoveAll(dest)
			spent := make([]time.Duration, len(importMarks))
			for range b.N {
				b.StopTimer()
				os.RemoveAll(dest)
				os.MkdirAll(tmp, 0o755)
				db := openBenchDB(b, dest)
				start := time.Now()
				b.StartTimer()
				_, err := importState(context.Background(), db, importOptions{
					snapshot:          filepath.Join(dir, "snapshot.bin"),
					preimages:         filepath.Join(dir, "preimages.bin"),
					anchor:            &types.Header{Number: new(big.Int), Root: root},
					conversionOptions: conversionOptions{tmpDir: tmp, sortBudget: budget, workers: workers},
				})
				b.StopTimer()
				db.Close()
				if err != nil {
					b.Fatal(err)
				}
				for i, m := range importMarks {
					at := clock.at(m.msg)
					spent[i] += at.Sub(start)
					start = at
				}
			}
			for i, m := range importMarks {
				b.ReportMetric(spent[i].Seconds()/float64(b.N), m.name+"-s")
			}
		})
	}
}

// benchSize returns the benchmark's state size, from $BINTRIE_BENCH_ACCOUNTS
// or 5% of mainnet's accounts, and its fixture directory under
// $BINTRIE_BENCH_DIR, skipping the benchmark without one.
func benchSize(b *testing.B, kind string) (string, int) {
	dir := os.Getenv("BINTRIE_BENCH_DIR")
	if dir == "" {
		b.Skip("BINTRIE_BENCH_DIR not set")
	}
	accounts := 21_208_000
	if s := os.Getenv("BINTRIE_BENCH_ACCOUNTS"); s != "" {
		var err error
		if accounts, err = strconv.Atoi(s); err != nil {
			b.Fatal(err)
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d", kind, accounts)), accounts
}

// selfConvertMarks are the log lines closing each self-conversion phase but
// the last, the read-back, which ends with the conversion.
var selfConvertMarks = []struct{ msg, name string }{
	{"Converting state", "scan"},
	{"Built the binary tree", "build"},
	{"Verified code against its hashes", "code"},
	{"Verified consensus anchoring", "check2"},
}

// BenchmarkSelfConvert times an online self-conversion of a mainnet-shaped
// state the way convertAt runs it when no block follows the preimage file's
// anchor: the merkle database and its preimage file are generated once, and
// every run converts a copy of the database. The sort budget scales like
// BenchmarkImport's, from the node's, but the collector paces itself against
// the live heap: a ballast holds the rest of the node's budget, so it collects
// as often as on mainnet. The ballast is never written, so it costs no RSS.
func BenchmarkSelfConvert(b *testing.B) {
	dir, accounts := benchSize(b, "selfconvert")
	root := benchSource(b, dir, accounts)
	budget := int(onlineSortBudget * int64(accounts) / 424_160_000)
	ballast := make([]byte, onlineSortBudget-budget)
	defer runtime.KeepAlive(ballast)

	defer debug.SetGCPercent(debug.SetGCPercent(50)) // the node's --gogc
	defer log.SetDefault(log.Root())
	clock := &phaseClock{marks: make(map[string]time.Time)}
	log.SetDefault(log.NewLogger(clock))
	dest, tmp := filepath.Join(dir, "dest"), filepath.Join(dir, "tmp")
	defer os.RemoveAll(dest)
	spent := make([]time.Duration, len(selfConvertMarks)+1)
	for range b.N {
		b.StopTimer()
		os.RemoveAll(dest)
		os.RemoveAll(tmp)
		os.MkdirAll(tmp, 0o755)
		if err := os.CopyFS(dest, os.DirFS(filepath.Join(dir, "source"))); err != nil {
			b.Fatal(err)
		}
		db := openBenchDB(b, dest)
		start := time.Now()
		b.StartTimer()
		_, err := benchSelfConvert(db, filepath.Join(dir, "preimages.bin"), root, tmp, budget, utils.PBTImportWorkersFlag.Value, nil)
		b.StopTimer()
		end := time.Now()
		db.Close()
		if err != nil {
			b.Fatal(err)
		}
		for i, m := range selfConvertMarks {
			at := clock.at(m.msg)
			spent[i] += at.Sub(start)
			start = at
		}
		spent[len(selfConvertMarks)] += end.Sub(start)
	}
	for i, m := range selfConvertMarks {
		b.ReportMetric(spent[i].Seconds()/float64(b.N), m.name+"-s")
	}
	b.ReportMetric(spent[len(selfConvertMarks)].Seconds()/float64(b.N), "readback-s")
}

// benchSelfConvert converts db's persisted state with the keys of the
// preimage file at path, as convertAt does with no access list to read,
// pausing while a block is processed (y), and returns the root it built.
func benchSelfConvert(db ethdb.Database, path string, root common.Hash, tmp string, budget, workers int, y *blockYield) (common.Hash, error) {
	f, ranges, pieces, err := splitPreimages(path, workers)
	if err != nil {
		return common.Hash{}, err
	}
	defer f.Close()
	persisted, sources, err := capturePersisted(db, ranges)
	if err != nil {
		return common.Hash{}, err
	}
	if persisted != root {
		return common.Hash{}, fmt.Errorf("persisted state %x, want %x", persisted, root)
	}
	release := func() {
		for _, s := range sources {
			s.release()
		}
	}
	defer release()
	lists, err := accessListKeys(context.Background(), nil, 0, 0, ranges, tmp)
	if err != nil {
		return common.Hash{}, err
	}
	defer func() {
		for _, k := range lists {
			k.close()
		}
	}()
	plan := scanPlan{ranges: ranges, release: release}
	for i := range ranges {
		keys, err := newDriftKeys(pieces[i], lists[i])
		if err != nil {
			return common.Hash{}, err
		}
		plan.sources = append(plan.sources, sources[i])
		plan.keys = append(plan.keys, keys)
	}
	compactions, _ := db.(ethdb.CompactionController) // as convertAt passes the node's
	return convertPlan(context.Background(), db, plan, y, compactions, conversionOptions{
		sortBudget: budget,
		tmpDir:     tmp,
		workers:    workers,
		anchor:     &types.Header{Number: new(big.Int), Root: root},
	})
}

// phaseClock is a log handler noting when each message was last logged.
type phaseClock struct {
	mu    sync.Mutex
	marks map[string]time.Time
}

func (c *phaseClock) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}
func (c *phaseClock) WithAttrs([]slog.Attr) slog.Handler { return c }
func (c *phaseClock) WithGroup(string) slog.Handler      { return c }

func (c *phaseClock) Handle(_ context.Context, r slog.Record) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.marks[r.Message] = r.Time
	return nil
}

func (c *phaseClock) at(msg string) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.marks[msg]
}

func (c *phaseClock) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.marks)
}

// stage sets each block's stage of marks, the first not closed by the time
// the block began, by when the lines closing them were last logged.
func (c *phaseClock) stage(times []blockTime, marks []struct{ msg, name string }) {
	for i := range times {
		times[i].stage = len(marks) - 1
		for j, m := range marks {
			if at := c.at(m.msg); !at.IsZero() && times[i].at.Before(at) {
				times[i].stage = j
				break
			}
		}
	}
}

// openBenchDB opens a database as `geth bintrie import` does with default flags.
func openBenchDB(b *testing.B, dir string) ethdb.Database {
	kv, err := pebble.New(filepath.Join(dir, "chaindata"), 512, utils.MakeDatabaseHandles(0), "", false)
	if err != nil {
		b.Fatal(err)
	}
	db, err := rawdb.Open(kv, rawdb.OpenOptions{Ancient: filepath.Join(dir, "ancient")})
	if err != nil {
		b.Fatal(err)
	}
	return db
}

// benchArtifacts returns the anchor root of the pair in dir, generating the
// pair first if absent: a synthetic state of n accounts is converted with both
// artifacts, then dropped.
func benchArtifacts(b *testing.B, dir string, n int) common.Hash {
	rootFile := filepath.Join(dir, "root")
	if blob, err := os.ReadFile(rootFile); err == nil {
		return common.BytesToHash(blob)
	}
	b.Logf("generating %d accounts into %s", n, dir)
	src, tmp := filepath.Join(dir, "source"), filepath.Join(dir, "tmp")
	os.RemoveAll(src)
	os.MkdirAll(tmp, 0o755)
	db := openBenchDB(b, src)
	root := commitMainnetShape(b, db, n)
	ro := triedb.NewDatabase(db, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	_, err := convertState(context.Background(), db, ro, root, conversionOptions{
		tmpDir:       tmp,
		sortBudget:   sortBudget,
		workers:      runtime.NumCPU(),
		snapshotPath: filepath.Join(dir, "snapshot.bin"),
		preimagePath: filepath.Join(dir, "preimages.bin"),
	})
	ro.Close()
	db.Close()
	if err != nil {
		b.Fatal(err)
	}
	os.RemoveAll(src)
	if err := os.WriteFile(rootFile, root[:], 0o644); err != nil {
		b.Fatal(err)
	}
	return root
}

// benchSource returns the state root of the merkle database in dir/source,
// generating it first if absent, with the preimage file at that root beside
// it as dir/preimages.bin.
func benchSource(b *testing.B, dir string, n int) common.Hash {
	rootFile := filepath.Join(dir, "root")
	if blob, err := os.ReadFile(rootFile); err == nil {
		return common.BytesToHash(blob)
	}
	b.Logf("generating %d accounts into %s", n, dir)
	src, tmp := filepath.Join(dir, "source"), filepath.Join(dir, "tmp")
	os.RemoveAll(src)
	os.MkdirAll(tmp, 0o755)
	db := openBenchDB(b, src)
	root := commitMainnetShape(b, db, n)
	ro := triedb.NewDatabase(db, &triedb.Config{Preimages: true, PathDB: pathdb.ReadOnly})
	_, err := exportArtifacts(context.Background(), db, ro, root, conversionOptions{
		tmpDir:       tmp,
		sortBudget:   sortBudget,
		workers:      runtime.NumCPU(),
		snapshotPath: filepath.Join(tmp, "snapshot.bin"),
		preimagePath: filepath.Join(dir, "preimages.bin"),
	})
	ro.Close()
	db.Close()
	os.RemoveAll(tmp)
	if err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(rootFile, root[:], 0o644); err != nil {
		b.Fatal(err)
	}
	return root
}

// commitMainnetShape commits n accounts shaped after mainnet's 424 M accounts
// and 1.67 B slots. 15% hold code, about 23 accounts per distinct code of
// 0.5-24 KB, and 0.5% of the rest are EIP-7702 delegations. Four contracts hold
// 1% of the slots each, one contract in a hundred up to 20,000, the others up
// to 16; a tenth of a contract's keys are small indices, the rest hashed keys.
func commitMainnetShape(b *testing.B, db ethdb.Database, n int) common.Hash {
	cfg := *pathdb.Defaults
	cfg.NoAsyncFlush = true
	tdb := triedb.NewDatabase(db, &triedb.Config{Preimages: true, PathDB: &cfg})
	sdb := state.NewDatabase(tdb, state.NewCodeDB(db))
	var (
		rng    = rand.New(rand.NewPCG(8347, 8297))
		codes  = max(n*15/100/23, 1)
		whales = n * 394 / 10_000 // 1% of ~3.94 slots per account
		root   = types.EmptyRootHash
		chunk  = 50_000
	)
	upTo := func(hi float64) int { return int(math.Exp(rng.Float64()*math.Log(hi+1))) - 1 }
	for lo := 0; lo < n; lo += chunk {
		st, err := state.New(root, sdb)
		if err != nil {
			b.Fatal(err)
		}
		for i := lo; i < min(lo+chunk, n); i++ {
			var addr common.Address
			binary.BigEndian.PutUint64(addr[12:], uint64(i)+1)
			whale := i%(n/4+1) == n/8
			switch kind := rng.IntN(1000); {
			case whale || kind < 150:
				st.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
				st.SetCode(addr, benchCode(rng.IntN(codes)), tracing.CodeChangeUnspecified)
				slots := upTo(16)
				if whale {
					slots = whales
				} else if rng.IntN(100) == 0 {
					slots = upTo(20_000)
				}
				for j := range slots {
					var key, value common.Hash
					if j < min(slots/10, 64) {
						key[31] = byte(j)
					} else {
						binary.BigEndian.PutUint64(key[:8], rng.Uint64())
						binary.BigEndian.PutUint64(key[24:], rng.Uint64())
					}
					binary.BigEndian.PutUint64(value[24:], 1+rng.Uint64())
					st.SetState(addr, key, value)
				}
			case kind < 155:
				var target common.Address
				binary.BigEndian.PutUint64(target[12:], rng.Uint64())
				st.SetNonce(addr, 1+rng.Uint64N(100), tracing.NonceChangeUnspecified)
				st.SetCode(addr, types.AddressToDelegation(target), tracing.CodeChangeUnspecified)
			default:
				st.SetNonce(addr, rng.Uint64N(1000), tracing.NonceChangeUnspecified)
				st.SetBalance(addr, uint256.NewInt(1+rng.Uint64N(1e18)), tracing.BalanceChangeUnspecified)
			}
		}
		if root, err = st.Commit(uint64(lo/chunk)+1, true, true); err != nil {
			b.Fatal(err)
		}
		// Flush the layers now and then, so memory stays flat at any size.
		if (lo/chunk)%16 == 15 {
			if err := tdb.Commit(root, false); err != nil {
				b.Fatal(err)
			}
		}
	}
	if err := tdb.Commit(root, false); err != nil {
		b.Fatal(err)
	}
	if err := tdb.Close(); err != nil {
		b.Fatal(err)
	}
	return root
}

// benchCode returns the k-th distinct code, of 0.5-24 KB.
func benchCode(k int) []byte {
	rng := rand.New(rand.NewPCG(uint64(k), 7702))
	code := make([]byte, int(math.Exp(rng.Float64()*math.Log(48)))*512)
	for i := range code {
		code[i] = byte(rng.Uint32())
	}
	return code
}
