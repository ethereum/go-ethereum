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
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

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

// The conversion benchmark, on the database every real node has: pebble with
// an ancient store. Not a CI check; benchmarks do not run under plain
// `go test`. Invocations:
//
//	go test ./cmd/geth -run - -bench BenchmarkConvertState -benchtime 1x
//	BINTRIE_BENCH_ACCOUNTS=70000 BINTRIE_BENCH_VERBOSE=1 \
//	  go test ./cmd/geth -run - -bench BenchmarkConvertState -benchtime 1x -timeout 1h -v
//
// BINTRIE_BENCH_ACCOUNTS sizes the fixture (default 5000; 70k lands the
// source datadir around 108 MB); BINTRIE_BENCH_VERBOSE turns on the
// converter's phase logs for time attribution.

// benchConvertAccounts resolves the fixture size.
func benchConvertAccounts() int {
	if s := os.Getenv("BINTRIE_BENCH_ACCOUNTS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			panic(fmt.Sprintf("bad BINTRIE_BENCH_ACCOUNTS %q", s))
		}
		return n
	}
	return 5000
}

// benchConvertAddr derives a deterministic address.
func benchConvertAddr(i int) common.Address {
	var a common.Address
	binary.BigEndian.PutUint64(a[12:], uint64(i)+1)
	return a
}

// benchConvertCode returns the bytecode of account i: nothing for nine in
// ten, a shared blob for one in forty, one multi-stem blob, and a unique
// ~1 KB blob otherwise, so the code zone sees sharing, spanning and dedup.
func benchConvertCode(i int) []byte {
	switch {
	case i%10 != 0:
		return nil
	case i == 10:
		code := make([]byte, 10*1024)
		for j := range code {
			code[j] = 0x5b
		}
		return code
	case i%40 == 0:
		code := make([]byte, 1024)
		for j := range code {
			code[j] = 0x60
		}
		return code
	default:
		code := make([]byte, 1024)
		binary.BigEndian.PutUint64(code, uint64(i))
		for j := 8; j < len(code); j++ {
			code[j] = 0x5b
		}
		return code
	}
}

// buildConvertSource populates a merkle state with preimages on a real
// pebble database: `accounts` accounts, each with two storage slots (one in
// the header range, one past it), a tenth of them contracts. Committed in
// batches - a single giant commit spawns one goroutine per account.
func buildConvertSource(b *testing.B, dir string, accounts int) (ethdb.Database, common.Hash) {
	b.Helper()
	pdb, err := pebble.New(filepath.Join(dir, "kv"), 0, 0, "", false)
	if err != nil {
		b.Fatal(err)
	}
	disk, err := rawdb.Open(pdb, rawdb.OpenOptions{Ancient: filepath.Join(dir, "ancient")})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { disk.Close() })

	pcfg := *pathdb.Defaults
	pcfg.NoAsyncFlush = true // determinism: flush inline
	tdb := triedb.NewDatabase(disk, &triedb.Config{
		Preimages: true,
		PathDB:    &pcfg,
	})
	db := state.NewDatabase(tdb, state.NewCodeDB(disk))

	var (
		root     = types.EmptyRootHash
		perBatch = 50_000
	)
	for start := 0; start < accounts; start += perBatch {
		statedb, err := state.New(root, db)
		if err != nil {
			b.Fatal(err)
		}
		for i := start; i < min(start+perBatch, accounts); i++ {
			addr := benchConvertAddr(i)
			statedb.SetBalance(addr, uint256.NewInt(uint64(i)+1), tracing.BalanceChangeUnspecified)
			statedb.SetNonce(addr, uint64(i%300)+1, tracing.NonceChangeUnspecified)
			if code := benchConvertCode(i); code != nil {
				statedb.SetCode(addr, code, tracing.CodeChangeUnspecified)
			}
			statedb.SetState(addr, common.BigToHash(big.NewInt(int64(i%64))), common.BigToHash(big.NewInt(int64(i)+1)))
			statedb.SetState(addr, common.BigToHash(big.NewInt(int64(64+i%4096))), common.BigToHash(big.NewInt(int64(i)+2)))
		}
		root, err = statedb.Commit(uint64(start/perBatch), true, true)
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := tdb.Commit(root, false); err != nil {
		b.Fatal(err)
	}
	tdb.Close()
	// Drive the data into SSTables so the size below means something.
	if err := pdb.Compact(nil, nil); err != nil {
		b.Fatal(err)
	}
	return disk, root
}

// dirSizeMB sums a directory tree's file sizes, tolerating files the
// database's background compaction removes mid-walk.
func dirSizeMB(b *testing.B, dir string) float64 {
	b.Helper()
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		b.Fatal(err)
	}
	return float64(total) / (1 << 20)
}

func BenchmarkConvertState(b *testing.B) {
	if os.Getenv("BINTRIE_BENCH_VERBOSE") != "" {
		log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	}
	var (
		accounts = benchConvertAccounts()
		dir      = b.TempDir()
	)
	chaindb, root := buildConvertSource(b, dir, accounts)
	srcMB := dirSizeMB(b, dir)
	b.Logf("fixture: %d accounts (%d slots), source datadir %.1f MB", accounts, 2*accounts, srcMB)

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := wipeBinaryTrieState(chaindb, ""); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if _, err := convertState(chaindb, src, root, conversionOptions{tmpDir: dir}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	seconds := b.Elapsed().Seconds() / float64(b.N)
	b.ReportMetric(float64(accounts)/seconds, "accounts/s")
	b.ReportMetric(srcMB/seconds, "srcMB/s")
	b.Logf("converted datadir now %.1f MB", dirSizeMB(b, dir))
}

// BenchmarkImportState measures the consumer against the producer on the same
// fixture: a conversion exports the artifacts once, then each iteration
// verifies and ingests them into the wiped namespace. The question it answers
// is whether consuming keeps up with producing, which is what bounds how far
// an anchor can lag.
func BenchmarkImportState(b *testing.B) {
	if os.Getenv("BINTRIE_BENCH_VERBOSE") != "" {
		log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	}
	var (
		accounts = benchConvertAccounts()
		dir      = b.TempDir()
		snapPath = filepath.Join(dir, "snapshot.bin")
		prePath  = filepath.Join(dir, "preimages.bin")
	)
	chaindb, root := buildConvertSource(b, dir, accounts)
	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	if _, err := convertState(chaindb, src, root, conversionOptions{
		tmpDir:       dir,
		snapshotPath: snapPath,
		preimagePath: prePath,
	}); err != nil {
		b.Fatalf("fixture conversion failed: %v", err)
	}
	src.Close()
	snapMB, preMB := fileSizeMB(b, snapPath), fileSizeMB(b, prePath)
	b.Logf("artifacts: snapshot %.1f MB, preimages %.1f MB, %d accounts", snapMB, preMB, accounts)

	header := &types.Header{Number: new(big.Int), Root: root}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := wipeBinaryTrieState(chaindb, ""); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if _, err := importState(chaindb, importOptions{
			snapshot:          snapPath,
			preimages:         prePath,
			anchor:            header,
			conversionOptions: conversionOptions{tmpDir: dir},
		}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	seconds := b.Elapsed().Seconds() / float64(b.N)
	b.ReportMetric(float64(accounts)/seconds, "accounts/s")
	b.ReportMetric((snapMB+preMB)/seconds, "artifactMB/s")
}

// fileSizeMB returns a file's size in MB.
func fileSizeMB(b *testing.B, path string) float64 {
	b.Helper()
	info, err := os.Stat(path)
	if err != nil {
		b.Fatal(err)
	}
	return float64(info.Size()) / (1 << 20)
}
